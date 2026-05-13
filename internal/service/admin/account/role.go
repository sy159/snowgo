package account

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"slices"
	"snowgo/internal/constant"
	"snowgo/internal/dal/model"
	"snowgo/internal/dal/query"
	"snowgo/internal/dal/repo"
	"snowgo/internal/dao/admin/account"
	"snowgo/internal/service/admin/system"
	common "snowgo/pkg"
	"snowgo/pkg/xauth"
	"snowgo/pkg/xcache"
	"snowgo/pkg/xdatabase/mysql"
	e "snowgo/pkg/xerror"
	"snowgo/pkg/xlogger"
	"time"
)

type RoleRepo interface {
	IsCodeExists(ctx context.Context, code string, roleId int32) (bool, error)
	CreateRole(ctx context.Context, role *model.SysRole) (*model.SysRole, error)
	TransactionCreateRole(ctx context.Context, tx *query.Query, role *model.SysRole) (*model.SysRole, error)
	UpdateRole(ctx context.Context, role *model.SysRole) (*model.SysRole, error)
	TransactionUpdateRole(ctx context.Context, tx *query.Query, role *model.SysRole) (*model.SysRole, error)
	DeleteById(ctx context.Context, roleId int32) error
	GetRoleById(ctx context.Context, roleId int32) (*model.SysRole, error)
	GetRoleList(ctx context.Context, cond *account.RoleListCondition) ([]*model.SysRole, int64, error)
	TransactionCreateRoleMenu(ctx context.Context, tx *query.Query, roleMenuList []*model.SysRoleMenu) error
	TransactionDeleteRoleMenu(ctx context.Context, tx *query.Query, roleId int32) error
	TransactionDeleteById(ctx context.Context, tx *query.Query, roleId int32) error
	IsUsedUserByIds(ctx context.Context, userId int32) (bool, error)
	CountMenuByIds(ctx context.Context, ids []int32) (int64, error)
	GetMenuIdsByRoleId(ctx context.Context, roleId int32) ([]int32, error)
	GetMenuPermsByRoleId(ctx context.Context, roleId int32) ([]string, error)
	GetMenuPermsByRoleIds(ctx context.Context, roleIds []int32) ([]string, error)
	GetMenuListByRoleId(ctx context.Context, roleId int32) ([]*model.SysMenu, error)
	ListRoleMenuPerms(ctx context.Context) ([]*account.RoleMenuPerm, error)
	GetUserMenuIds(ctx context.Context, userId int32) ([]int32, error)
	IsSuperAdmin(ctx context.Context, userId int32) (bool, error)
}

// RolePermsGetter 定义角色权限与菜单查询的接口，用于解耦 UserService 对 RoleService 的直接依赖
type RolePermsGetter interface {
	GetRolePermsListByRuleID(ctx context.Context, roleId int32) ([]string, error)
	GetRoleMenuListByRuleID(ctx context.Context, roleId int32) ([]*MenuData, error)
	GetRolePermsListByRuleIds(ctx context.Context, roleIds []int32) ([]string, error)
}

type RoleService struct {
	db         *repo.Repository
	roleDao    RoleRepo
	cache      xcache.Cache
	logService system.OperationLogWriter
}

var _ RolePermsGetter = (*RoleService)(nil)

// NewRoleService 构造函数
func NewRoleService(db *repo.Repository, roleDao RoleRepo, cache xcache.Cache, logService system.OperationLogWriter) *RoleService {
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
	ErrRoleNotFound          = e.NewBizError(e.RoleNotFound)
	ErrRoleCodeUsed          = e.NewBizError(e.RoleCodeExist)
	ErrRoleUsed              = e.NewBizError(e.RoleUsed)
	ErrRoleIDInvalid         = e.NewBizError(e.RoleIDInvalid)
	ErrRoleMenuNotExist      = e.NewBizError(e.RoleMenuNotExist)
	ErrRoleMenuNotAuthorized = e.NewBizError(e.RoleMenuNotAuthorized)
)

// CreateRole 创建角色
func (s *RoleService) CreateRole(ctx context.Context, param *RoleParam) (int32, error) {
	// 获取登录ctx
	userContext, err := xauth.GetUserContext(ctx)
	if err != nil {
		return 0, err
	}

	// 校验 code 是否已存在（事务外快速失败，避免白开事务）
	exists, err := s.roleDao.IsCodeExists(ctx, param.Code, 0)
	if err != nil {
		return 0, fmt.Errorf("校验角色编码失败: %w", err)
	}
	if exists {
		return 0, ErrRoleCodeUsed
	}

	role := &model.SysRole{
		Name:        &param.Name,
		Code:        param.Code,
		Description: &param.Description,
	}

	var roleObj *model.SysRole
	// 事务创建角色，以及关联菜单权限
	err = s.db.WriteQuery().Transaction(func(tx *query.Query) error {
		// 校验菜单id是否都存在
		if len(param.MenuIds) > 0 {
			menuLen, err := s.roleDao.CountMenuByIds(ctx, param.MenuIds)
			if err != nil {
				xlogger.ErrorfCtx(ctx, "获取菜单数量异常: %v", err)
				return fmt.Errorf("校验设置的菜单失败: %w", err)
			}
			if menuLen != int64(len(param.MenuIds)) {
				return ErrRoleMenuNotExist
			}

			// 校验操作者是否有权限分配这些菜单
			operatorMenuIds, err := s.roleDao.GetUserMenuIds(ctx, userContext.UserId)
			if err != nil {
				xlogger.ErrorfCtx(ctx, "获取操作者菜单权限异常: %v", err)
				return fmt.Errorf("校验操作者菜单权限失败: %w", err)
			}
			for _, id := range param.MenuIds {
				if !slices.Contains(operatorMenuIds, id) {
					return ErrRoleMenuNotAuthorized
				}
			}
		}

		// 创建角色
		roleObj, err = s.roleDao.TransactionCreateRole(ctx, tx, role)
		if err != nil {
			// 唯一索引冲突兜底
			if mysql.IsDuplicateKeyErr(err) {
				return ErrRoleCodeUsed
			}
			xlogger.ErrorfCtx(ctx, "角色创建失败: %v", err)
			return fmt.Errorf("角色创建失败: %w", err)
		}

		// 创建角色与菜单关联关系
		if len(param.MenuIds) > 0 {
			roleMenuList := make([]*model.SysRoleMenu, 0, len(param.MenuIds))
			for _, menuId := range param.MenuIds {
				roleMenuList = append(roleMenuList, &model.SysRoleMenu{
					RoleID: roleObj.ID,
					MenuID: menuId,
				})
			}
			err = s.roleDao.TransactionCreateRoleMenu(ctx, tx, roleMenuList)
			if err != nil {
				xlogger.ErrorfCtx(ctx, "角色与菜单关联关系创建失败: %v", err)
				return fmt.Errorf("角色与菜单关联关系创建失败: %w", err)
			}
		}

		// 创建操作日志
		err = s.logService.CreateOperationLog(ctx, tx, &system.OperationLogInput{
			OperatorID:   userContext.UserId,
			OperatorName: userContext.Username,
			OperatorType: constant.OperatorUser,
			Resource:     constant.ResourceRole,
			ResourceID:   int64(roleObj.ID),
			TraceID:      userContext.TraceId,
			Action:       constant.ActionCreate,
			BeforeData:   nil,
			AfterData:    roleObj,
			Description: fmt.Sprintf("用户(%d-%s)创建了角色(%d-%s)",
				userContext.UserId, userContext.Username, roleObj.ID, role.Code),
			IP: userContext.IP,
		})
		if err != nil {
			xlogger.ErrorfCtx(ctx, "操作日志创建失败: %+v err: %v", roleObj, err)
			return fmt.Errorf("操作日志创建失败: %w", err)
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
		return ErrRoleIDInvalid
	}
	// 校验 code 是否重复（事务外快速失败，排除当前 role.ID）
	isDuplicate, err := s.roleDao.IsCodeExists(ctx, param.Code, param.ID)
	if err != nil {
		return fmt.Errorf("查询角色 code 是否存在异常: %w", err)
	}
	if isDuplicate {
		return ErrRoleCodeUsed
	}
	// 获取原始角色信息
	oldRole, err := s.roleDao.GetRoleById(ctx, param.ID)
	if err != nil {
		return ErrRoleNotFound
	}

	// 事务内更新角色，以及关联菜单权限
	var ruleObj *model.SysRole
	err = s.db.WriteQuery().Transaction(func(tx *query.Query) error {
		// 校验菜单id是否都存在
		if len(param.MenuIds) > 0 {
			menuLen, err := s.roleDao.CountMenuByIds(ctx, param.MenuIds)
			if err != nil {
				xlogger.ErrorfCtx(ctx, "获取菜单数量异常: %v", err)
				return fmt.Errorf("校验设置的菜单失败: %w", err)
			}
			if menuLen != int64(len(param.MenuIds)) {
				return ErrRoleMenuNotExist
			}

			// 校验操作者是否有权限分配这些菜单（超级管理员跳过）
			isSuperAdmin, err := s.roleDao.IsSuperAdmin(ctx, userContext.UserId)
			if err != nil {
				xlogger.ErrorfCtx(ctx, "判断操作者是否为超级管理员异常: %v", err)
				return fmt.Errorf("判断操作者身份失败: %w", err)
			}
			if !isSuperAdmin {
				operatorMenuIds, err := s.roleDao.GetUserMenuIds(ctx, userContext.UserId)
				if err != nil {
					xlogger.ErrorfCtx(ctx, "获取操作者菜单权限异常: %v", err)
					return fmt.Errorf("校验操作者菜单权限失败: %w", err)
				}
				for _, id := range param.MenuIds {
					if !slices.Contains(operatorMenuIds, id) {
						return ErrRoleMenuNotAuthorized
					}
				}
			}
		}

		// 更新字段
		ruleObj, err = s.roleDao.TransactionUpdateRole(ctx, tx, &model.SysRole{
			ID:          param.ID,
			Name:        &param.Name,
			Code:        param.Code,
			Description: &param.Description,
		})
		if err != nil {
			// 唯一索引冲突兜底
			if mysql.IsDuplicateKeyErr(err) {
				return ErrRoleCodeUsed
			}
			return fmt.Errorf("更新角色失败: %w", err)
		}

		// 删除角色关联权限
		err = s.roleDao.TransactionDeleteRoleMenu(ctx, tx, param.ID)
		if err != nil {
			xlogger.ErrorfCtx(ctx, "角色与菜单关联关系删除失败: %v", err)
			return fmt.Errorf("角色与菜单关联关系删除失败: %w", err)
		}

		// 创建角色与菜单关联关系
		if len(param.MenuIds) > 0 {
			roleMenuList := make([]*model.SysRoleMenu, 0, len(param.MenuIds))
			for _, menuId := range param.MenuIds {
				roleMenuList = append(roleMenuList, &model.SysRoleMenu{
					RoleID: param.ID,
					MenuID: menuId,
				})
			}
			err = s.roleDao.TransactionCreateRoleMenu(ctx, tx, roleMenuList)
			if err != nil {
				xlogger.ErrorfCtx(ctx, "角色与菜单关联关系创建失败: %v", err)
				return fmt.Errorf("角色与菜单关联关系创建失败: %w", err)
			}
		}

		// 创建操作日志
		err = s.logService.CreateOperationLog(ctx, tx, &system.OperationLogInput{
			OperatorID:   userContext.UserId,
			OperatorName: userContext.Username,
			OperatorType: constant.OperatorUser,
			Resource:     constant.ResourceRole,
			ResourceID:   int64(param.ID),
			TraceID:      userContext.TraceId,
			Action:       constant.ActionUpdate,
			BeforeData:   oldRole,
			AfterData:    ruleObj,
			Description: fmt.Sprintf("用户(%d-%s)修改了角色(%d-%s)信息",
				userContext.UserId, userContext.Username, param.ID, param.Code),
			IP: userContext.IP,
		})
		if err != nil {
			xlogger.ErrorfCtx(ctx, "操作日志创建失败: %+v err: %v", param, err)
			return fmt.Errorf("操作日志创建失败: %w", err)
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
		return ErrRoleIDInvalid
	}
	if id == constant.SuperAdminRoleId {
		return e.NewBizError(e.SuperAdminRoleCannotDelete)
	}
	// 查询被删除角色信息，用于操作日志记录
	oldRole, err := s.roleDao.GetRoleById(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrRoleNotFound
		}
		return fmt.Errorf("获取角色信息失败: %w", err)
	}

	err = s.db.WriteQuery().Transaction(func(tx *query.Query) error {
		// 检查角色是否被用户使用（事务内防止并发）
		isUsed, err := s.roleDao.IsUsedUserByIds(ctx, id)
		if err != nil {
			return fmt.Errorf("检查角色使用情况失败: %w", err)
		}
		if isUsed {
			return ErrRoleUsed
		}

		// 删除角色
		err = s.roleDao.TransactionDeleteById(ctx, tx, id)
		if err != nil {
			xlogger.ErrorfCtx(ctx, "角色删除失败: %v", err)
			return fmt.Errorf("角色删除失败: %w", err)
		}

		// 删除角色关联权限
		err = s.roleDao.TransactionDeleteRoleMenu(ctx, tx, id)
		if err != nil {
			xlogger.ErrorfCtx(ctx, "角色与菜单关联关系删除失败: %v", err)
			return fmt.Errorf("角色与菜单关联关系删除失败: %w", err)
		}

		// 创建操作日志
		err = s.logService.CreateOperationLog(ctx, tx, &system.OperationLogInput{
			OperatorID:   userContext.UserId,
			OperatorName: userContext.Username,
			OperatorType: constant.OperatorUser,
			Resource:     constant.ResourceRole,
			ResourceID:   int64(id),
			TraceID:      userContext.TraceId,
			Action:       constant.ActionDelete,
			BeforeData:   oldRole,
			AfterData:    nil,
			Description: fmt.Sprintf("用户(%d-%s)删除了角色(%d-%s)",
				userContext.UserId, userContext.Username, id, oldRole.Code),
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
		return nil, ErrRoleIDInvalid
	}
	// 获取role信息
	r, err := s.roleDao.GetRoleById(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRoleNotFound
		}
		xlogger.ErrorfCtx(ctx, "获取角色失败: %v", err)
		return nil, fmt.Errorf("获取角色失败: %w", err)
	}

	// 获取role对应的菜单ids
	menuIds, err := s.roleDao.GetMenuIdsByRoleId(ctx, id)
	if err != nil {
		xlogger.ErrorfCtx(ctx, "获取关联的菜单id列表失败: %v", err)
		return nil, fmt.Errorf("获取关联的菜单id列表失败: %w", err)
	}
	return &RoleInfo{
		ID:          r.ID,
		Name:        common.DerefOrZero(r.Name),
		Code:        r.Code,
		Description: common.DerefOrZero(r.Description),
		MenuIds:     menuIds,
		CreatedAt:   common.DerefOrZero(r.CreatedAt),
		UpdatedAt:   common.DerefOrZero(r.UpdatedAt),
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
		return nil, fmt.Errorf("角色列表查询失败: %w", err)
	}
	infos := make([]*RoleInfo, 0, len(list))
	for _, r := range list {
		infos = append(infos, &RoleInfo{
			ID:          r.ID,
			Name:        common.DerefOrZero(r.Name),
			Code:        r.Code,
			Description: common.DerefOrZero(r.Description),
			CreatedAt:   common.DerefOrZero(r.CreatedAt),
			UpdatedAt:   common.DerefOrZero(r.UpdatedAt),
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
		return nil, ErrRoleIDInvalid
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
	if data, ok, _ := s.cache.Get(ctx, cacheKey); ok {
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
			Path:      common.DerefOrZero(m.Path),
			Icon:      common.DerefOrZero(m.Icon),
			Perms:     common.DerefOrZero(m.Perms),
			SortOrder: m.SortOrder,
			CreatedAt: common.DerefOrZero(m.CreatedAt),
			UpdatedAt: common.DerefOrZero(m.UpdatedAt),
		})
	}

	// 缓存结果 15天
	if b, err := json.Marshal(menus); err == nil {
		_ = s.cache.Set(ctx, cacheKey, string(b), constant.CacheRoleMenuExpirationDay*24*time.Hour)
	}

	return menus, nil
}

// GetRolePermsListByRuleIds 批量获取多个角色的接口权限列表
func (s *RoleService) GetRolePermsListByRuleIds(ctx context.Context, roleIds []int32) ([]string, error) {
	if len(roleIds) == 0 {
		return nil, nil
	}
	return s.roleDao.GetMenuPermsByRoleIds(ctx, roleIds)
}
