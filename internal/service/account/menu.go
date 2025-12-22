package account

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"snowgo/internal/constant"
	"snowgo/internal/dal/model"
	"snowgo/internal/dal/query"
	"snowgo/internal/dal/repo"
	"snowgo/internal/service/log"
	"snowgo/pkg/xauth"
	"snowgo/pkg/xcache"
	"snowgo/pkg/xlogger"
	"sort"
	"time"
)

type MenuRepo interface {
	CreateMenu(ctx context.Context, mn *model.Menu) (*model.Menu, error)
	TransactionCreateMenu(ctx context.Context, tx *query.Query, menu *model.Menu) (*model.Menu, error)
	UpdateMenu(ctx context.Context, mn *model.Menu) (*model.Menu, error)
	TransactionUpdateMenu(ctx context.Context, tx *query.Query, menu *model.Menu) (*model.Menu, error)
	DeleteById(ctx context.Context, id int32) error
	TransactionDeleteById(ctx context.Context, tx *query.Query, id int32) error
	GetById(ctx context.Context, id int32) (*model.Menu, error)
	GetByParentId(ctx context.Context, parentId int32) ([]*model.Menu, error)
	GetAllMenus(ctx context.Context) ([]*model.Menu, error)
	IsUsedMenuByIds(ctx context.Context, menuIds []int32) (bool, error)
	GetRoleIdsByIds(ctx context.Context, menuId int32) ([]int32, error)
}

type MenuService struct {
	db         *repo.Repository
	menuDao    MenuRepo
	cache      xcache.Cache
	logService *log.OperationLogService
}

func NewMenuService(db *repo.Repository, cache xcache.Cache, menuDao MenuRepo, logService *log.OperationLogService) *MenuService {
	return &MenuService{
		db:         db,
		cache:      cache,
		menuDao:    menuDao,
		logService: logService,
	}
}

type MenuParam struct {
	ID        int32  `json:"id"`
	ParentID  int32  `json:"parent_id"`
	MenuType  string `json:"menu_type" binding:"required,oneof=Dir Menu Btn"`
	Name      string `json:"name"`
	Path      string `json:"path"`
	Icon      string `json:"icon"`
	Perms     string `json:"perms"`
	SortOrder int32  `json:"sort_order" binding:"required,gte=0"`
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

// CreateMenu 创建菜单权限
func (s *MenuService) CreateMenu(ctx context.Context, p *MenuParam) (int32, error) {
	// 获取登录ctx
	userContext, err := xauth.GetUserContext(ctx)
	if err != nil {
		return 0, err
	}

	// 校验父节点
	if p.ParentID > 0 {
		if _, err := s.menuDao.GetById(ctx, p.ParentID); err != nil {
			return 0, errors.New("父级菜单不存在")
		}
	}

	menu := &model.Menu{
		ParentID:  p.ParentID,
		MenuType:  p.MenuType,
		Name:      p.Name,
		Path:      &p.Path,
		Icon:      &p.Icon,
		Perms:     &p.Perms,
		SortOrder: p.SortOrder,
	}
	var menuObj *model.Menu

	err = s.db.WriteQuery().Transaction(func(tx *query.Query) error {
		// 创建菜单
		menuObj, err = s.menuDao.TransactionCreateMenu(ctx, tx, menu)
		if err != nil {
			xlogger.ErrorfCtx(ctx, "创建菜单失败: %v", err)
			return errors.WithMessage(err, "创建菜单失败")
		}

		// 创建操作日志
		err = s.logService.CreateOperationLog(ctx, tx, log.OperationLogInput{
			OperatorID:   userContext.UserId,
			OperatorName: userContext.Username,
			OperatorType: constant.OperatorUser,
			Resource:     constant.ResourceMenu,
			ResourceID:   menuObj.ID,
			TraceID:      userContext.TraceId,
			Action:       constant.ActionCreate,
			BeforeData:   "",
			AfterData:    menuObj,
			Description: fmt.Sprintf("用户(%d-%s)创建了%s类型的菜单(%d-%s)",
				userContext.UserId, userContext.Username, menuObj.MenuType, menuObj.ID, menuObj.Name),
			IP: userContext.IP,
		})
		if err != nil {
			xlogger.ErrorfCtx(ctx, "操作日志创建失败: %v err: %v", menuObj, err)
			return errors.WithMessage(err, "操作日志创建失败")
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
		return errors.New("菜单ID无效")
	}

	// 获取原始角色信息（可选）
	oldMenu, err := s.menuDao.GetById(ctx, p.ID)
	if err != nil {
		return errors.WithMessage(err, "菜单不存在")
	}

	// 校验父节点
	if p.ParentID > 0 {
		if p.ParentID == p.ID {
			return errors.New("父级菜单不能是自己")
		}
		if _, err := s.menuDao.GetById(ctx, p.ParentID); err != nil {
			return errors.New("父级菜单不存在")
		}
	}
	// 更新
	mn := &model.Menu{
		ID:        p.ID,
		ParentID:  p.ParentID,
		MenuType:  p.MenuType,
		Name:      p.Name,
		Path:      &p.Path,
		Icon:      &p.Icon,
		Perms:     &p.Perms,
		SortOrder: p.SortOrder,
	}

	err = s.db.WriteQuery().Transaction(func(tx *query.Query) error {
		menuObj, err := s.menuDao.TransactionUpdateMenu(ctx, tx, mn)
		if err != nil {
			xlogger.ErrorfCtx(ctx, "更新菜单失败: %v", err)
			return errors.WithMessage(err, "更新菜单失败")
		}

		// 创建操作日志
		err = s.logService.CreateOperationLog(ctx, tx, log.OperationLogInput{
			OperatorID:   userContext.UserId,
			OperatorName: userContext.Username,
			OperatorType: constant.OperatorUser,
			Resource:     constant.ResourceMenu,
			ResourceID:   p.ID,
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
			return errors.WithMessage(err, "操作日志创建失败")
		}

		return nil
	})
	if err != nil {
		return err
	}

	xlogger.InfofCtx(ctx, "菜单更新成功: old=%+v new=%+v", oldMenu, mn)

	// 如果修改了接口权限，需要更新角色-接口权限数据缓存
	if *oldMenu.Perms != p.Perms {
		roleIds, err := s.menuDao.GetRoleIdsByIds(ctx, p.ID)
		if err != nil {
			xlogger.ErrorfCtx(ctx, "获取角色ids异常: %v", err)
		}
		for _, roleId := range roleIds {
			// 清除角色对应接口权限缓存
			cacheKey := fmt.Sprintf("%s%d", constant.CacheRoleMenuPrefix, roleId)
			if _, err := s.cache.Delete(ctx, cacheKey); err != nil {
				xlogger.ErrorfCtx(ctx, "清除角色对应接口权限缓存失败: %v", err)
			}
		}
	}

	// 清理菜单树缓存
	if _, err := s.cache.Delete(ctx, constant.CacheMenuTree); err != nil {
		xlogger.ErrorfCtx(ctx, "清理菜单树缓存失败: %v", err)
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
		return errors.New("菜单ID无效")
	}

	// 可以在此处校验是否存在子节点，若存在可拒绝删除(也可以改为递归删除)
	subMenus, _ := s.menuDao.GetByParentId(ctx, id)
	if len(subMenus) > 0 {
		return errors.New("存在子菜单，无法删除")
	}

	// 如果被角色使用，也不能删除
	isUsed, err := s.menuDao.IsUsedMenuByIds(ctx, []int32{id})
	if err != nil {
		return errors.WithMessage(err, "查询角色是否被使用失败")
	}
	if isUsed {
		return errors.New("该菜单权限已被使用，无法删除")
	}

	err = s.db.WriteQuery().Transaction(func(tx *query.Query) error {
		// 删除菜单
		err = s.menuDao.TransactionDeleteById(ctx, tx, id)
		if err != nil {
			xlogger.ErrorfCtx(ctx, "删除菜单失败: %v", err)
			return errors.WithMessage(err, "删除菜单失败")
		}

		// 创建操作日志
		err = s.logService.CreateOperationLog(ctx, tx, log.OperationLogInput{
			OperatorID:   userContext.UserId,
			OperatorName: userContext.Username,
			OperatorType: constant.OperatorUser,
			Resource:     constant.ResourceMenu,
			ResourceID:   id,
			TraceID:      userContext.TraceId,
			Action:       constant.ActionDelete,
			BeforeData:   "",
			AfterData:    "",
			Description: fmt.Sprintf("用户(%d-%s)删除了菜单(%d)",
				userContext.UserId, userContext.Username, id),
			IP: userContext.IP,
		})
		if err != nil {
			xlogger.ErrorfCtx(ctx, "操作日志创建失败: %v", err)
			return errors.WithMessage(err, "操作日志创建失败")
		}

		return nil
	})

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
	if data, err := s.cache.Get(ctx, constant.CacheMenuTree); err == nil && data != "" {
		var tree []*MenuInfo
		if err := json.Unmarshal([]byte(data), &tree); err == nil {
			xlogger.InfofCtx(ctx, "缓存获取菜单树")
			return tree, nil
		}
	}

	menus, err := s.menuDao.GetAllMenus(ctx)
	if err != nil {
		xlogger.ErrorfCtx(ctx, "获取全部菜单失败: %v", err)
		return nil, errors.WithMessage(err, "获取全部菜单失败")
	}

	// 构造 map[id]MenuInfo
	nodeMap := make(map[int32]*MenuInfo, len(menus))
	for _, m := range menus {
		nodeMap[m.ID] = &MenuInfo{
			ID:        m.ID,
			ParentID:  m.ParentID,
			MenuType:  m.MenuType,
			Name:      m.Name,
			Path:      *m.Path,
			Icon:      *m.Icon,
			Perms:     *m.Perms,
			SortOrder: m.SortOrder,
			CreatedAt: *m.CreatedAt,
			UpdatedAt: *m.UpdatedAt,
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
