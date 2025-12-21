package account

import (
	"context"
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
	"snowgo/internal/constant"
	"snowgo/internal/dal/query"
	"snowgo/internal/service/log"
	"snowgo/pkg/xauth"
	e "snowgo/pkg/xerror"
	"time"

	"github.com/pkg/errors"
	"snowgo/internal/dal/model"
	"snowgo/internal/dal/repo"
	"snowgo/internal/dao/account"
	"snowgo/pkg/xcache"
	"snowgo/pkg/xlogger"
)

type RoleRepo interface {
	IsCodeExists(ctx context.Context, code string, roleId int32) (bool, error)
	CreateRole(ctx context.Context, role *model.Role) (*model.Role, error)
	TransactionCreateRole(ctx context.Context, tx *query.Query, role *model.Role) (*model.Role, error)
	UpdateRole(ctx context.Context, role *model.Role) (*model.Role, error)
	DeleteById(ctx context.Context, roleId int32) error
	GetRoleById(ctx context.Context, roleId int32) (*model.Role, error)
	GetRoleList(ctx context.Context, cond *account.RoleListCondition) ([]*model.Role, int64, error)
	TransactionCreateRoleMenu(ctx context.Context, tx *query.Query, roleMenuList []*model.RoleMenu) error
	TransactionDeleteRoleMenu(ctx context.Context, tx *query.Query, roleId int32) error
	TransactionDeleteById(ctx context.Context, tx *query.Query, roleId int32) error
	IsUsedUserByIds(ctx context.Context, userId int32) (bool, error)
	CountMenuByIds(ctx context.Context, ids []int32) (int64, error)
	GetMenuIdsByRoleId(ctx context.Context, roleId int32) ([]int32, error)
	GetMenuPermsByRoleId(ctx context.Context, roleId int32) ([]string, error)
	GetMenuListByRoleId(ctx context.Context, roleId int32) ([]*model.Menu, error)
	ListRoleMenuPerms(ctx context.Context) ([]*account.RoleMenuPerm, error)
}

type RoleService struct {
	db         *repo.Repository
	roleDao    RoleRepo
	cache      xcache.Cache
	logService *log.OperationLogService
}

// NewRoleService 构造函数
func NewRoleService(db *repo.Repository, roleDao RoleRepo, cache xcache.Cache, logService *log.OperationLogService) *RoleService {
	return &RoleService{db: db, roleDao: roleDao, cache: cache, logService: logService}
}

// RoleParam 创建或更新角色输入
type RoleParam struct {
	ID          int32   `form:"id"`
	Name        string  `json:"name" binding:"required,max=128"`
	Code        string  `json:"code" binding:"required,max=64"`
	Description string  `json:"description"`
	MenuIds     []int32 `json:"menu_ids" binding:"required"`
}

// RoleInfo 返回给前端的角色信息
type RoleInfo struct {
	ID          int32     `json:"id"`
	Name        string    `json:"name"`
	Code        string    `json:"code"`
	Description string    `json:"description"`
	MenuIds     []int32   `json:"menu_ids"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// RoleList 返回角色列表
type RoleList struct {
	List  []*RoleInfo `json:"list"`
	Total int64       `json:"total"`
}

type RoleListCondition struct {
	Ids    []int32 `json:"ids" form:"ids"`
	Name   string  `json:"name" form:"name"`
	Code   string  `json:"code" form:"code"`
	Offset int32   `json:"offset" form:"offset"`
	Limit  int32   `json:"limit" form:"limit"`
}

var (
	ErrRoleNotFound = errors.New(e.RoleListError.GetErrMsg())
)

// CreateRole 创建角色
func (s *RoleService) CreateRole(ctx context.Context, param *RoleParam) (int32, error) {
	// 获取登录ctx
	userContext, err := xauth.GetUserContext(ctx)
	if err != nil {
		return 0, err
	}

	// 校验 code 是否存在
	exists, err := s.roleDao.IsCodeExists(ctx, param.Code, 0)
	if err != nil {
		xlogger.ErrorfCtx(ctx, "校验角色code异常: %v", err)
		return 0, errors.WithMessage(err, "校验角色编码失败")
	}
	if exists {
		return 0, errors.New("角色编码已存在")
	}

	// 校验 菜单id是否都存在
	if len(param.MenuIds) > 0 {
		menuLen, err := s.roleDao.CountMenuByIds(ctx, param.MenuIds)
		if err != nil {
			xlogger.ErrorfCtx(ctx, "获取菜单数量异常: %v", err)
			return 0, errors.WithMessage(err, "校验设置的菜单失败")
		}
		if menuLen != int64(len(param.MenuIds)) {
			return 0, errors.New("设置的菜单不存在")
		}
	}

	role := &model.Role{
		Name:        &param.Name,
		Code:        param.Code,
		Description: &param.Description,
	}

	var roleObj *model.Role
	// 事务创建角色，以及关联菜单权限
	err = s.db.WriteQuery().Transaction(func(tx *query.Query) error {
		// 创建角色
		roleObj, err = s.roleDao.TransactionCreateRole(ctx, tx, role)
		if err != nil {
			xlogger.ErrorfCtx(ctx, "角色创建失败: %v", err)
			return errors.WithMessage(err, "角色创建失败")
		}

		// 创建角色与菜单关联关系
		if len(param.MenuIds) > 0 {
			roleMenuList := make([]*model.RoleMenu, 0, len(param.MenuIds))
			for _, menuId := range param.MenuIds {
				roleMenuList = append(roleMenuList, &model.RoleMenu{
					RoleID: roleObj.ID,
					MenuID: menuId,
				})
			}
			err = s.roleDao.TransactionCreateRoleMenu(ctx, tx, roleMenuList)
			if err != nil {
				xlogger.ErrorfCtx(ctx, "角色与菜单关联关系创建失败: %v", err)
				return errors.WithMessage(err, "角色与菜单关联关系创建失败")
			}
		}

		// 创建操作日志
		err = s.logService.CreateOperationLog(ctx, tx, log.OperationLogInput{
			OperatorID:   userContext.UserId,
			OperatorName: userContext.Username,
			OperatorType: constant.OperatorUser,
			Resource:     constant.ResourceRole,
			ResourceID:   roleObj.ID,
			TraceID:      userContext.TraceId,
			Action:       constant.ActionCreate,
			BeforeData:   "",
			AfterData:    param,
			Description: fmt.Sprintf("用户(%d-%s)创建了角色(%d-%s)",
				userContext.UserId, userContext.Username, roleObj.ID, role.Code),
			IP: userContext.IP,
		})
		if err != nil {
			xlogger.ErrorfCtx(ctx, "操作日志创建失败: %+v err: %v", roleObj, err)
			return errors.WithMessage(err, "操作日志创建失败")
		}

		return nil
	})
	if err != nil {
		return 0, err
	}
	xlogger.InfofCtx(ctx, "角色创建成功: %+v", roleObj)
	return roleObj.ID, nil
}

// UpdateRole 更新角色信息
func (s *RoleService) UpdateRole(ctx context.Context, param *RoleParam) error {
	// 获取登录ctx
	userContext, err := xauth.GetUserContext(ctx)
	if err != nil {
		return err
	}

	if param.ID <= 0 {
		return errors.New("角色ID无效")
	}
	// 校验 code 是否重复（排除当前 role.ID）
	isDuplicate, err := s.roleDao.IsCodeExists(ctx, param.Code, param.ID)
	if err != nil {
		return errors.WithMessage(err, "查询角色 code 是否存在异常")
	}
	if isDuplicate {
		return errors.New("角色编码已存在")
	}
	// 获取原始角色信息（可选）
	oldRole, err := s.roleDao.GetRoleById(ctx, param.ID)
	if err != nil {
		return errors.WithMessage(err, "角色不存在")
	}

	// 校验 菜单id是否都存在
	if len(param.MenuIds) > 0 {
		menuLen, err := s.roleDao.CountMenuByIds(ctx, param.MenuIds)
		if err != nil {
			xlogger.ErrorfCtx(ctx, "获取菜单数量异常: %v", err)
			return errors.WithMessage(err, "校验设置的菜单失败")
		}
		if menuLen != int64(len(param.MenuIds)) {
			return errors.New("设置的菜单不存在")
		}
	}

	// 事务更新角色，以及关联菜单权限
	var ruleObj *model.Role
	err = s.db.WriteQuery().Transaction(func(tx *query.Query) error {
		// 更新字段
		ruleObj, err = s.roleDao.UpdateRole(ctx, &model.Role{
			ID:          param.ID,
			Name:        &param.Name,
			Code:        param.Code,
			Description: &param.Description,
		})
		if err != nil {
			return errors.WithMessage(err, "更新角色失败")
		}

		// 删除角色关联权限
		err = s.roleDao.TransactionDeleteRoleMenu(ctx, tx, param.ID)
		if err != nil {
			xlogger.ErrorfCtx(ctx, "角色与菜单关联关系删除失败: %v", err)
			return errors.WithMessage(err, "角色与菜单关联关系删除失败")
		}

		// 创建角色与菜单关联关系
		if len(param.MenuIds) > 0 {
			roleMenuList := make([]*model.RoleMenu, 0, len(param.MenuIds))
			for _, menuId := range param.MenuIds {
				roleMenuList = append(roleMenuList, &model.RoleMenu{
					RoleID: param.ID,
					MenuID: menuId,
				})
			}
			err = s.roleDao.TransactionCreateRoleMenu(ctx, tx, roleMenuList)
			if err != nil {
				xlogger.ErrorfCtx(ctx, "角色与菜单关联关系创建失败: %v", err)
				return errors.WithMessage(err, "角色与菜单关联关系创建失败")
			}
		}

		// 创建操作日志
		err = s.logService.CreateOperationLog(ctx, tx, log.OperationLogInput{
			OperatorID:   userContext.UserId,
			OperatorName: userContext.Username,
			OperatorType: constant.OperatorUser,
			Resource:     constant.ResourceRole,
			ResourceID:   param.ID,
			TraceID:      userContext.TraceId,
			Action:       constant.ActionUpdate,
			BeforeData:   oldRole,
			AfterData:    param,
			Description: fmt.Sprintf("用户(%d-%s)修改了角色(%d-%s)信息",
				userContext.UserId, userContext.Username, param.ID, param.Code),
			IP: userContext.IP,
		})
		if err != nil {
			xlogger.ErrorfCtx(ctx, "操作日志创建失败: %+v err: %v", param, err)
			return errors.WithMessage(err, "操作日志创建失败")
		}

		return nil
	})
	if err != nil {
		return err
	}

	xlogger.InfofCtx(ctx, "角色更新成功: old=%+v new=%+v", oldRole, ruleObj)

	// 清除角色对应接口权限缓存
	cacheKey := fmt.Sprintf("%s%d", constant.CacheRoleMenuPrefix, param.ID)
	if _, err := s.cache.Delete(ctx, cacheKey); err != nil {
		xlogger.ErrorfCtx(ctx, "清除角色对应接口权限缓存失败: %v", err)
	}

	return nil
}

// DeleteRole 删除角色
func (s *RoleService) DeleteRole(ctx context.Context, id int32) error {
	// 获取登录ctx
	userContext, err := xauth.GetUserContext(ctx)
	if err != nil {
		return err
	}

	if id <= 0 {
		return errors.New("角色ID无效")
	}

	// 如果用户使用了角色，不能删除
	isUsed, err := s.roleDao.IsUsedUserByIds(ctx, id)
	if err != nil {
		return errors.WithMessage(err, "")
	}
	if isUsed {
		return errors.New("该角色已被使用，无法删除")
	}

	err = s.db.WriteQuery().Transaction(func(tx *query.Query) error {
		// 删除角色
		err := s.roleDao.TransactionDeleteById(ctx, tx, id)
		if err != nil {
			xlogger.ErrorfCtx(ctx, "角色删除失败: %v", err)
			return errors.WithMessage(err, "角色删除失败")
		}

		// 删除角色关联权限
		err = s.roleDao.TransactionDeleteRoleMenu(ctx, tx, id)
		if err != nil {
			xlogger.ErrorfCtx(ctx, "角色与菜单关联关系删除失败: %v", err)
			return errors.WithMessage(err, "角色与菜单关联关系删除失败")
		}

		// 创建操作日志
		err = s.logService.CreateOperationLog(ctx, tx, log.OperationLogInput{
			OperatorID:   userContext.UserId,
			OperatorName: userContext.Username,
			OperatorType: constant.OperatorUser,
			Resource:     constant.ResourceRole,
			ResourceID:   id,
			TraceID:      userContext.TraceId,
			Action:       constant.ActionDelete,
			BeforeData:   "",
			AfterData:    "",
			Description: fmt.Sprintf("用户(%d-%s)删除了角色(%d)",
				userContext.UserId, userContext.Username, id),
			IP: userContext.IP,
		})
		if err != nil {
			xlogger.ErrorfCtx(ctx, "操作日志创建失败: %v", err)
			return errors.WithMessage(err, "操作日志创建失败")
		}

		return nil
	})
	if err != nil {
		return err
	}
	xlogger.InfofCtx(ctx, "角色删除成功: %d", id)

	// 清除角色对应接口权限缓存
	cacheKey := fmt.Sprintf("%s%d", constant.CacheRoleMenuPrefix, id)
	if _, err := s.cache.Delete(ctx, cacheKey); err != nil {
		xlogger.ErrorfCtx(ctx, "清除角色对应接口权限缓存失败: %v", err)
	}

	return nil
}

// GetRoleById 获取角色详情
func (s *RoleService) GetRoleById(ctx context.Context, id int32) (*RoleInfo, error) {
	if id <= 0 {
		return nil, errors.New("角色ID无效")
	}
	// 获取role信息
	r, err := s.roleDao.GetRoleById(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRoleNotFound
		}
		xlogger.ErrorfCtx(ctx, "获取角色失败: %v", err)
		return nil, errors.WithMessage(err, "获取角色失败")
	}

	// 获取role对应的菜单ids
	menuIds, err := s.roleDao.GetMenuIdsByRoleId(ctx, id)
	if err != nil {
		xlogger.ErrorfCtx(ctx, "获取关联的菜单id列表失败: %v", err)
		return nil, errors.WithMessage(err, "获取关联的菜单id列表失败")
	}
	return &RoleInfo{
		ID:          r.ID,
		Name:        *r.Name,
		Code:        r.Code,
		Description: *r.Description,
		MenuIds:     menuIds,
		CreatedAt:   *r.CreatedAt,
		UpdatedAt:   *r.UpdatedAt,
	}, nil
}

// ListRoles 获取角色列表
func (s *RoleService) ListRoles(ctx context.Context, cond *RoleListCondition) (*RoleList, error) {
	xlogger.InfofCtx(ctx, "获取角色列表: %+v", cond)
	list, total, err := s.roleDao.GetRoleList(ctx, &account.RoleListCondition{
		Ids:    cond.Ids,
		Name:   cond.Name,
		Code:   cond.Code,
		Offset: cond.Offset,
		Limit:  cond.Limit,
	})
	if err != nil {
		xlogger.ErrorfCtx(ctx, "角色列表查询失败: %v", err)
		return nil, errors.WithMessage(err, "角色列表查询失败")
	}
	infos := make([]*RoleInfo, 0, len(list))
	for _, r := range list {
		infos = append(infos, &RoleInfo{
			ID:          r.ID,
			Name:        *r.Name,
			Code:        r.Code,
			Description: *r.Description,
			CreatedAt:   *r.CreatedAt,
			UpdatedAt:   *r.UpdatedAt,
		})
	}
	return &RoleList{List: infos, Total: total}, nil
}

// GetRolePermsListByRuleID 获取角色对应接口权限列表
func (s *RoleService) GetRolePermsListByRuleID(ctx context.Context, roleId int32) ([]string, error) {
	//// 尝试从缓存读取
	//cacheKey := fmt.Sprintf("%s%d", constant.CacheRolePermsPrefix, roleId)
	//if data, err := s.cache.Get(ctx, cacheKey); err == nil && data != "" {
	//	var m []string
	//	if err := json.Unmarshal([]byte(data), &m); err == nil {
	//		return m, nil
	//	}
	//}
	//
	//// 获取roleId: perms数组
	//menuPermsList, err := s.roleDao.GetMenuPermsByRoleId(ctx, roleId)
	//if err != nil {
	//	xlogger.Errorf("list role menu perms is err: %v", err)
	//	return nil, err
	//}
	//
	//// 缓存结果 15天
	//if b, err := json.Marshal(menuPermsList); err == nil {
	//	_ = s.cache.Set(ctx, cacheKey, string(b), constant.CacheRolePermsExpirationDay*24*time.Hour)
	//}
	//
	//return menuPermsList, nil
	if roleId <= 0 {
		return nil, errors.New("角色ID无效")
	}
	menuList, err := s.GetRoleMenuListByRuleID(ctx, roleId)
	if err != nil {
		return nil, err
	}
	perms := make([]string, 0, len(menuList))
	for _, menu := range menuList {
		if menu.MenuType == constant.MenuTypeBtn && menu.Perms != "" {
			perms = append(perms, menu.Perms)
		}
	}
	return perms, nil
}

// GetRoleMenuListByRuleID 获取角色对应菜单列表
func (s *RoleService) GetRoleMenuListByRuleID(ctx context.Context, roleId int32) ([]*MenuData, error) {
	// 尝试从缓存读取
	cacheKey := fmt.Sprintf("%s%d", constant.CacheRoleMenuPrefix, roleId)
	if data, err := s.cache.Get(ctx, cacheKey); err == nil && data != "" {
		var m []*MenuData
		if err := json.Unmarshal([]byte(data), &m); err == nil {
			return m, nil
		}
	}

	// 获取roleId: perms数组
	menuList, err := s.roleDao.GetMenuListByRoleId(ctx, roleId)
	if err != nil {
		xlogger.ErrorfCtx(ctx, "list role menu is err: %v", err)
		return nil, err
	}

	menus := make([]*MenuData, 0, len(menuList))
	for _, m := range menuList {
		menus = append(menus, &MenuData{
			ID:        m.ID,
			ParentID:  m.ParentID,
			MenuType:  m.MenuType,
			Name:      m.Name,
			Path:      *m.Path,
			Icon:      *m.Icon,
			Perms:     *m.Perms,
			OrderNum:  m.OrderNum,
			CreatedAt: *m.CreatedAt,
			UpdatedAt: *m.UpdatedAt,
		})
	}

	// 缓存结果 15天
	if b, err := json.Marshal(menus); err == nil {
		_ = s.cache.Set(ctx, cacheKey, string(b), constant.CacheRoleMenuExpirationDay*24*time.Hour)
	}

	return menus, nil
}
