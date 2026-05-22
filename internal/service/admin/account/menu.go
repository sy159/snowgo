package account

import (
	"context"
	"encoding/json"
	"fmt"
	"snowgo/internal/constant"
	"snowgo/internal/dal/model"
	"snowgo/internal/dal/query"
	"snowgo/internal/dal/repo"
	"snowgo/internal/service/admin/contract"
	common "snowgo/pkg"
	"snowgo/pkg/xauth"
	"snowgo/pkg/xcache"
	e "snowgo/pkg/xerror"
	"snowgo/pkg/xlogger"
	"sort"
	"time"
)

type MenuRepo interface {
	CreateMenu(ctx context.Context, q *query.Query, mn *model.SysMenu) (*model.SysMenu, error)
	UpdateMenu(ctx context.Context, q *query.Query, mn *model.SysMenu) (*model.SysMenu, error)
	DeleteById(ctx context.Context, q *query.Query, id int32) error
	GetById(ctx context.Context, q *query.Query, id int32) (*model.SysMenu, error)
	GetByParentId(ctx context.Context, q *query.Query, parentId int32) ([]*model.SysMenu, error)
	GetAllMenus(ctx context.Context) ([]*model.SysMenu, error)
	IsUsedMenuByIds(ctx context.Context, q *query.Query, menuIds []int32) (bool, error)
	IsPermsExists(ctx context.Context, q *query.Query, perms string, excludeId int32) (bool, error)
	IsPathExists(ctx context.Context, q *query.Query, path string, excludeId int32) (bool, error)
	GetRoleIdsByIds(ctx context.Context, menuId int32) ([]int32, error)
}

type MenuService struct {
	db         *repo.Repository
	menuDao    MenuRepo
	cache      xcache.Cache
	logService contract.OperationLogWriter
}

func NewMenuService(db *repo.Repository, cache xcache.Cache, menuDao MenuRepo, logService contract.OperationLogWriter) *MenuService {
	return &MenuService{
		db:         db,
		cache:      cache,
		menuDao:    menuDao,
		logService: logService,
	}
}

type MenuParam struct {
	ID        int32   `json:"id"`
	ParentID  int32   `json:"parent_id"`
	MenuType  string  `json:"menu_type" binding:"required,oneof=Dir Menu Btn"`
	Name      string  `json:"name" binding:"required"`
	Path      *string `json:"path"`
	Icon      *string `json:"icon"`
	Perms     *string `json:"perms"`
	SortOrder int32   `json:"sort_order" binding:"gte=0"`
}

// MenuInfo 返回给前端的树节点结构
type MenuInfo struct {
	ID        int32       `json:"id"`
	ParentID  int32       `json:"parent_id"`
	MenuType  string      `json:"menu_type"`
	Name      string      `json:"name"`
	Path      string      `json:"path"`
	Icon      string      `json:"icon"`
	Perms     string      `json:"perms"`
	SortOrder int32       `json:"sort_order"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
	Children  []*MenuInfo `json:"children"`
}

// MenuData menu数据
type MenuData struct {
	ID        int32     `json:"id"`
	ParentID  int32     `json:"parent_id"`
	MenuType  string    `json:"menu_type"`
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	Icon      string    `json:"icon"`
	Perms     string    `json:"perms"`
	SortOrder int32     `json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

var (
	ErrMenuNotFound      = e.NewBizError(e.MenuNotFound)
	ErrMenuPermsExist    = e.NewBizError(e.MenuPermsExist)
	ErrMenuPathExist     = e.NewBizError(e.MenuPathExist)
	ErrMenuParentInvalid = e.NewBizError(e.MenuParentInvalid)
	ErrMenuParentSelf    = e.NewBizError(e.MenuParentSelf)
	ErrMenuHasChildren   = e.NewBizError(e.MenuHasChildren)
	ErrMenuUsedByRole    = e.NewBizError(e.MenuUsedByRole)
	ErrMenuIDInvalid     = e.NewBizError(e.MenuIDInvalid)
)

// CreateMenu 创建菜单权限
func (s *MenuService) CreateMenu(ctx context.Context, p *MenuParam) (int32, error) {
	// 获取登录ctx
	userContext, err := xauth.GetUserContext(ctx)
	if err != nil {
		return 0, err
	}

	// 校验父节点（创建时 p.ID 为 0，仅更新场景生效）
	if p.ID > 0 && p.ParentID == p.ID {
		return 0, ErrMenuParentSelf
	}

	menu := &model.SysMenu{
		ParentID:  p.ParentID,
		MenuType:  p.MenuType,
		Name:      p.Name,
		Path:      p.Path,
		Icon:      p.Icon,
		Perms:     p.Perms,
		SortOrder: p.SortOrder,
	}
	var menuObj *model.SysMenu

	err = s.db.WriteQuery().Transaction(func(tx *query.Query) error {
		// 校验父节点存在（事务内，防止并发删除父菜单）
		if p.ParentID > 0 {
			if _, err := s.menuDao.GetById(ctx, tx, p.ParentID); err != nil {
				return ErrMenuParentInvalid
			}
		}

		// 校验 perms 唯一性（事务内，menu 表无唯一索引，事务是唯一防线）
		permsVal := common.DerefOrZero(p.Perms)
		if permsVal != "" {
			exists, err := s.menuDao.IsPermsExists(ctx, tx, permsVal, 0)
			if err != nil {
				return fmt.Errorf("校验权限标识失败: %w", err)
			}
			if exists {
				return ErrMenuPermsExist
			}
		}
		// 校验 path 唯一性（事务内，menu 表无唯一索引，事务是唯一防线）
		pathVal := common.DerefOrZero(p.Path)
		if pathVal != "" {
			exists, err := s.menuDao.IsPathExists(ctx, tx, pathVal, 0)
			if err != nil {
				return fmt.Errorf("校验菜单路径失败: %w", err)
			}
			if exists {
				return ErrMenuPathExist
			}
		}

		// 创建菜单
		menuObj, err = s.menuDao.CreateMenu(ctx, tx, menu)
		if err != nil {
			xlogger.ErrorfCtx(ctx, "创建菜单失败: %v", err)
			return fmt.Errorf("创建菜单失败: %w", err)
		}

		// 创建操作日志
		err = s.logService.CreateOperationLog(ctx, tx, &contract.OperationLogInput{
			OperatorID:   userContext.UserId,
			OperatorName: userContext.Username,
			OperatorType: constant.OperatorUser,
			Resource:     constant.ResourceMenu,
			ResourceID:   int64(menuObj.ID),
			TraceID:      userContext.TraceId,
			Action:       constant.ActionCreate,
			BeforeData:   nil,
			AfterData:    menuObj,
			Description: fmt.Sprintf("用户(%d-%s)创建了%s类型的菜单(%d-%s)",
				userContext.UserId, userContext.Username, menuObj.MenuType, menuObj.ID, menuObj.Name),
			IP: userContext.IP,
		})
		if err != nil {
			xlogger.ErrorfCtx(ctx, "操作日志创建失败: %v err: %v", menuObj, err)
			return fmt.Errorf("操作日志创建失败: %w", err)
		}

		return nil
	})
	if err != nil {
		return 0, err
	}

	xlogger.InfofCtx(ctx, "菜单创建成功: %v", menuObj)

	// 清理菜单树缓存
	if _, err := s.cache.Delete(ctx, constant.CacheMenuTree); err != nil {
		xlogger.ErrorfCtx(ctx, "清理菜单树缓存失败: %v", err)
	}
	return menuObj.ID, nil
}

// UpdateMenu 更新菜单或按钮，校验同 CreateMenu
func (s *MenuService) UpdateMenu(ctx context.Context, p *MenuParam) error {
	// 获取登录ctx
	userContext, err := xauth.GetUserContext(ctx)
	if err != nil {
		return err
	}

	if p.ID <= 0 {
		return ErrMenuIDInvalid
	}

	// 获取原始角色信息（可选）
	oldMenu, err := s.menuDao.GetById(ctx, s.db.Query(), p.ID)
	if err != nil {
		return ErrMenuNotFound
	}

	// 更新
	mn := &model.SysMenu{
		ID:        p.ID,
		ParentID:  p.ParentID,
		MenuType:  p.MenuType,
		Name:      p.Name,
		Path:      p.Path,
		Icon:      p.Icon,
		Perms:     p.Perms,
		SortOrder: p.SortOrder,
	}

	err = s.db.WriteQuery().Transaction(func(tx *query.Query) error {
		// 校验父节点（事务内，防止并发删除父菜单）
		if p.ParentID > 0 {
			if p.ParentID == p.ID {
				return ErrMenuParentSelf
			}
			if _, err := s.menuDao.GetById(ctx, tx, p.ParentID); err != nil {
				return ErrMenuParentInvalid
			}
		}

		// 校验 perms 唯一性（事务内，menu 表无唯一索引，事务是唯一防线）
		permsVal := common.DerefOrZero(p.Perms)
		if permsVal != "" {
			exists, err := s.menuDao.IsPermsExists(ctx, tx, permsVal, p.ID)
			if err != nil {
				return fmt.Errorf("校验权限标识失败: %w", err)
			}
			if exists {
				return ErrMenuPermsExist
			}
		}
		// 校验 path 唯一性（事务内，menu 表无唯一索引，事务是唯一防线）
		pathVal := common.DerefOrZero(p.Path)
		if pathVal != "" {
			exists, err := s.menuDao.IsPathExists(ctx, tx, pathVal, p.ID)
			if err != nil {
				return fmt.Errorf("校验菜单路径失败: %w", err)
			}
			if exists {
				return ErrMenuPathExist
			}
		}

		menuObj, err := s.menuDao.UpdateMenu(ctx, tx, mn)
		if err != nil {
			xlogger.ErrorfCtx(ctx, "更新菜单失败: %v", err)
			return fmt.Errorf("更新菜单失败: %w", err)
		}

		// 创建操作日志
		err = s.logService.CreateOperationLog(ctx, tx, &contract.OperationLogInput{
			OperatorID:   userContext.UserId,
			OperatorName: userContext.Username,
			OperatorType: constant.OperatorUser,
			Resource:     constant.ResourceMenu,
			ResourceID:   int64(p.ID),
			TraceID:      userContext.TraceId,
			Action:       constant.ActionUpdate,
			BeforeData:   oldMenu,
			AfterData:    menuObj,
			Description: fmt.Sprintf("用户(%d-%s)修改了%s类型的菜单(%d-%s)信息",
				userContext.UserId, userContext.Username, p.MenuType, p.ID, p.Name),
			IP: userContext.IP,
		})
		if err != nil {
			xlogger.ErrorfCtx(ctx, "操作日志创建失败: %v err: %v", p, err)
			return fmt.Errorf("操作日志创建失败: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	xlogger.InfofCtx(ctx, "菜单更新成功: old=%+v new=%+v", oldMenu, mn)

	// 清理菜单树缓存
	if _, err := s.cache.Delete(ctx, constant.CacheMenuTree); err != nil {
		xlogger.ErrorfCtx(ctx, "清理菜单树缓存失败: %v", err)
	}

	// 清理绑定了该菜单的角色缓存（精准失效）
	roleIds, err := s.menuDao.GetRoleIdsByIds(ctx, p.ID)
	if err != nil {
		xlogger.ErrorfCtx(ctx, "获取角色ids异常: %v", err)
	}
	for _, roleId := range roleIds {
		cacheKey := fmt.Sprintf("%s%d", constant.CacheRoleMenuPrefix, roleId)
		if _, err := s.cache.Delete(ctx, cacheKey); err != nil {
			xlogger.ErrorfCtx(ctx, "清除角色对应接口权限缓存失败: %v", err)
		}
	}
	return nil
}

// DeleteMenuById 删除菜单或按钮
func (s *MenuService) DeleteMenuById(ctx context.Context, id int32) error {
	// 获取登录ctx
	userContext, err := xauth.GetUserContext(ctx)
	if err != nil {
		return err
	}

	if id <= 0 {
		return ErrMenuIDInvalid
	}
	// 查询被删除菜单信息，用于操作日志记录
	oldMenu, err := s.menuDao.GetById(ctx, s.db.Query(), id)
	if err != nil {
		return ErrMenuNotFound
	}

	err = s.db.WriteQuery().Transaction(func(tx *query.Query) error {
		// 检查是否有子菜单（事务内，防止并发创建子菜单）
		subMenus, err := s.menuDao.GetByParentId(ctx, tx, id)
		if err != nil {
			xlogger.ErrorfCtx(ctx, "查询子菜单失败 menu_id=%d err: %v", id, err)
			return fmt.Errorf("查询子菜单失败: %w", err)
		}
		if len(subMenus) > 0 {
			return ErrMenuHasChildren
		}

		// 检查是否被角色使用（事务内，防止并发绑定角色）
		isUsed, err := s.menuDao.IsUsedMenuByIds(ctx, tx, []int32{id})
		if err != nil {
			return fmt.Errorf("查询角色是否被使用失败: %w", err)
		}
		if isUsed {
			return ErrMenuUsedByRole
		}

		// 删除菜单
		err = s.menuDao.DeleteById(ctx, tx, id)
		if err != nil {
			xlogger.ErrorfCtx(ctx, "删除菜单失败: %v", err)
			return fmt.Errorf("删除菜单失败: %w", err)
		}

		// 创建操作日志
		err = s.logService.CreateOperationLog(ctx, tx, &contract.OperationLogInput{
			OperatorID:   userContext.UserId,
			OperatorName: userContext.Username,
			OperatorType: constant.OperatorUser,
			Resource:     constant.ResourceMenu,
			ResourceID:   int64(id),
			TraceID:      userContext.TraceId,
			Action:       constant.ActionDelete,
			BeforeData:   oldMenu,
			AfterData:    nil,
			Description: fmt.Sprintf("用户(%d-%s)删除了%s类型的菜单(%d-%s)",
				userContext.UserId, userContext.Username, oldMenu.MenuType, id, oldMenu.Name),
			IP: userContext.IP,
		})
		if err != nil {
			xlogger.ErrorfCtx(ctx, "操作日志创建失败: %v", err)
			return fmt.Errorf("操作日志创建失败: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	xlogger.InfofCtx(ctx, "菜单删除成功: %d", id)

	// 清理菜单树缓存
	if _, err := s.cache.Delete(ctx, constant.CacheMenuTree); err != nil {
		xlogger.ErrorfCtx(ctx, "清理菜单树缓存失败: %v", err)
	}
	return nil
}

// GetMenuTree 获取菜单树
func (s *MenuService) GetMenuTree(ctx context.Context) ([]*MenuInfo, error) {
	// 尝试缓存
	if data, ok, _ := s.cache.Get(ctx, constant.CacheMenuTree); ok {
		var tree []*MenuInfo
		if err := json.Unmarshal([]byte(data), &tree); err == nil {
			xlogger.InfofCtx(ctx, "缓存获取菜单树")
			return tree, nil
		}
	}

	menus, err := s.menuDao.GetAllMenus(ctx)
	if err != nil {
		xlogger.ErrorfCtx(ctx, "获取全部菜单失败: %v", err)
		return nil, fmt.Errorf("获取全部菜单失败: %w", err)
	}

	// 构造 map[id]MenuInfo
	nodeMap := make(map[int32]*MenuInfo, len(menus))
	for _, m := range menus {
		nodeMap[m.ID] = &MenuInfo{
			ID:        m.ID,
			ParentID:  m.ParentID,
			MenuType:  m.MenuType,
			Name:      m.Name,
			Path:      common.DerefOrZero(m.Path),
			Icon:      common.DerefOrZero(m.Icon),
			Perms:     common.DerefOrZero(m.Perms),
			SortOrder: m.SortOrder,
			CreatedAt: common.DerefOrZero(m.CreatedAt),
			UpdatedAt: common.DerefOrZero(m.UpdatedAt),
			Children:  []*MenuInfo{},
		}
	}

	// 构建树结构
	var roots []*MenuInfo
	for _, node := range nodeMap {
		if node.ParentID == 0 {
			roots = append(roots, node)
		} else if parent, ok := nodeMap[node.ParentID]; ok {
			parent.Children = append(parent.Children, node)
		} else {
			xlogger.ErrorfCtx(ctx, "菜单[%d] 的父节点 [%d] 不存在，挂到根节点", node.ID, node.ParentID)
			roots = append(roots, node)
		}
	}

	// 递归排序
	var sortNodes func(nodes []*MenuInfo)
	sortNodes = func(nodes []*MenuInfo) {
		if len(nodes) == 0 {
			return
		}
		sort.SliceStable(nodes, func(i, j int) bool {
			return nodes[i].SortOrder < nodes[j].SortOrder
		})
		for _, n := range nodes {
			sortNodes(n.Children)
		}
	}
	sortNodes(roots)

	// 缓存结果 15天
	if bs, err := json.Marshal(roots); err == nil {
		if err := s.cache.Set(ctx, constant.CacheMenuTree, string(bs), constant.CacheMenuTreeExpirationDay*24*time.Hour); err != nil {
			xlogger.ErrorfCtx(ctx, "缓存菜单树失败: %v", err)
		}
	}
	return roots, nil
}
