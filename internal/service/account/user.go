package account

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"snowgo/internal/constants"
	"snowgo/internal/dal/model"
	"snowgo/internal/dal/query"
	"snowgo/internal/dal/repo"
	"snowgo/internal/dao/account"
	"snowgo/internal/service/log"
	"snowgo/pkg/xauth"
	"snowgo/pkg/xcache"
	"snowgo/pkg/xcryption"
	e "snowgo/pkg/xerror"
	"snowgo/pkg/xlogger"
	"sort"
	"time"
)

// UserRepo 定义User相关db操作接口
type UserRepo interface {
	CreateUser(ctx context.Context, user *model.User) (*model.User, error)
	TransactionCreateUser(ctx context.Context, tx *query.Query, user *model.User) (*model.User, error)
	TransactionUpdateUser(ctx context.Context, tx *query.Query, userId int32, username, tel, nickname string) error
	TransactionCreateUserRole(ctx context.Context, tx *query.Query, userRole *model.UserRole) error
	TransactionCreateUserRoleInBatches(ctx context.Context, tx *query.Query, userRoleList []*model.UserRole) error
	TransactionDeleteUserRole(ctx context.Context, tx *query.Query, userId int32) error
	TransactionDeleteById(ctx context.Context, tx *query.Query, userId int32) error
	GetRoleListByUserId(ctx context.Context, userId int32) ([]*model.Role, error)
	GetRoleIdsByUserId(ctx context.Context, userId int32) ([]int32, error)
	IsNameTelDuplicate(ctx context.Context, username, tel string, userId int32) (bool, error)
	IsExistByRoleId(ctx context.Context, roleId int32) (bool, error)
	CountRoleByIds(ctx context.Context, roleId []int32) (int64, error)
	GetUserById(ctx context.Context, userId int32) (*model.User, error)
	GetUserByUsername(ctx context.Context, username string) (*model.User, error)
	GetUserList(ctx context.Context, condition *account.UserListCondition) ([]*model.User, int64, error)
	DeleteById(ctx context.Context, userId int32) error
	ResetPwdById(ctx context.Context, userId int32, password string) error
}

type UserService struct {
	db          *repo.Repository
	userDao     UserRepo
	cache       xcache.Cache
	roleService *RoleService
	logService  *log.OperationLogService
}

func NewUserService(db *repo.Repository, userDao UserRepo, cache xcache.Cache, roleService *RoleService,
	logService *log.OperationLogService) *UserService {
	return &UserService{
		db:          db,
		cache:       cache,
		userDao:     userDao,
		roleService: roleService,
		logService:  logService,
	}
}

type UserParam struct {
	ID       int32   `json:"id"`
	Username string  `json:"username" binding:"required,max=64"`
	Password string  `json:"password"`
	Tel      string  `json:"tel" binding:"required"`
	Nickname string  `json:"nickname"`
	RoleIds  []int32 `json:"role_ids"`
}

type UserInfo struct {
	ID        int32
	Username  string
	Tel       string
	Nickname  string
	Status    string
	RoleList  []*UserRole
	CreatedAt time.Time
	UpdatedAt time.Time
}

type UserPermissionInfo struct {
	ID          int32             `json:"id"`
	Username    string            `json:"username"`
	Tel         string            `json:"tel"`
	Nickname    string            `json:"nickname"`
	Status      string            `json:"status"`
	RoleList    []*UserRole       `json:"role_list"`
	Menus       []*MenuInfo       `json:"menu_list"`
	Permissions []*UserPermission `json:"permission_list"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

type UserRole struct {
	ID   int32  `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
}

type UserList struct {
	List  []*UserInfo
	Total int64
}

type UserPermission struct {
	ID    int32  `json:"id"`
	Name  string `json:"name"`
	Perms string `json:"perms"`
}

type UserListCondition struct {
	Ids      []int32 `json:"ids" form:"ids"`
	Username string  `json:"username" form:"username"`
	Tel      string  `json:"tel" form:"tel"`
	Nickname string  `json:"nickname" form:"nickname"`
	Status   string  `json:"status" form:"status"`
	Offset   int32   `json:"offset" form:"offset"`
	Limit    int32   `json:"limit" form:"limit"`
}

// CreateUser 创建用户
func (u *UserService) CreateUser(ctx context.Context, userParam *UserParam) (int32, error) {
	// 获取登录ctx
	userContext, err := xauth.GetUserContext(ctx)
	if err != nil {
		return 0, err
	}

	// 检查用户名，或者电话是否存在
	isDuplicate, err := u.userDao.IsNameTelDuplicate(ctx, userParam.Username, userParam.Tel, 0)
	if err != nil {
		xlogger.Errorf("查询用户名或电话是否存在异常: %v", err)
		return 0, errors.WithMessage(err, "查询用户名或电话是否存在异常")
	}
	if isDuplicate {
		return 0, errors.New(e.UserNameTelExistError.GetErrMsg())
	}

	// 检查设置的角色id是否存在
	if len(userParam.RoleIds) > 0 {
		//isExist, err := u.userDao.IsExistByRoleId(ctx, userParam.RoleIds)
		roleLen, err := u.userDao.CountRoleByIds(ctx, userParam.RoleIds)
		if err != nil {
			xlogger.Errorf("查询角色数量存在异常: %v", err)
			return 0, errors.WithMessage(err, "查询角色数量存在异常")
		}
		if roleLen != int64(len(userParam.RoleIds)) {
			return 0, errors.New("设置的角色不存在")
		}
	}

	// 加密密码
	pwd, err := xcryption.HashPassword(userParam.Password)
	if err != nil {
		xlogger.Errorf("密码加密异常: %v", err)
		return 0, errors.WithMessage(err, "密码加密异常")
	}
	activeStatus := constants.UserStatusActive
	var userObj *model.User
	err = u.db.WriteQuery().Transaction(func(tx *query.Query) error {
		// 创建用户
		userObj, err = u.userDao.TransactionCreateUser(ctx, tx, &model.User{
			Username:  userParam.Username,
			Password:  pwd,
			Tel:       userParam.Tel,
			Nickname:  &userParam.Nickname,
			Status:    &activeStatus,
			IsDeleted: false,
		})
		if err != nil {
			xlogger.Errorf("用户创建失败: %+v err: %v", userParam, err)
			return errors.WithMessage(err, "用户创建失败")
		}

		// 创建用户-role关联, 设置roleId才去创建
		if len(userParam.RoleIds) > 0 {
			userRoles := make([]*model.UserRole, 0, len(userParam.RoleIds))
			for _, roleId := range userParam.RoleIds {
				userRoles = append(userRoles, &model.UserRole{
					UserID: userObj.ID,
					RoleID: roleId,
				})
			}
			err = u.userDao.TransactionCreateUserRoleInBatches(ctx, tx, userRoles)
			if err != nil {
				xlogger.Errorf("用户与角色关联关系创建失败: %+v err: %v", userParam, err)
				return errors.WithMessage(err, "用户与角色关联关系创建失败")
			}
		}

		// 创建操作日志
		err = u.logService.CreateOperationLog(ctx, tx, log.OperationLogInput{
			OperatorID:   int32(userContext.UserId),
			OperatorName: userContext.Username,
			OperatorType: constants.OperatorUser,
			Resource:     constants.ResourceUser,
			ResourceID:   userObj.ID,
			TraceID:      userContext.TraceId,
			Action:       constants.ActionCreate,
			BeforeData:   "",
			AfterData:    userParam,
			Description: fmt.Sprintf("用户(%d-%s)创建了用户(%d-%s)",
				userContext.UserId, userContext.Username, userObj.ID, userObj.Username),
			IP: userContext.IP,
		})
		if err != nil {
			xlogger.Errorf("操作日志创建失败: %+v err: %v", userParam, err)
			return errors.WithMessage(err, "操作日志创建失败")
		}

		return nil
	})

	if err != nil {
		return 0, err
	}
	xlogger.Infof("用户创建成功: %+v", userObj)
	return userObj.ID, nil
}

// UpdateUser 更新用户
func (u *UserService) UpdateUser(ctx context.Context, userParam *UserParam) (int32, error) {
	// 获取登录ctx
	userContext, err := xauth.GetUserContext(ctx)
	if err != nil {
		return 0, err
	}

	if userParam.ID <= 0 {
		return 0, errors.New(e.UserNotFound.GetErrMsg())
	}
	// 获取用户信息
	oldUser, err := u.userDao.GetUserById(ctx, userParam.ID)
	if err != nil {
		xlogger.Errorf("获取用户(%d)信息异常: %v", userParam.ID, err)
		return 0, errors.WithMessage(err, "用户信息查询失败")
	}

	// 检查用户名，或者电话是否存在
	isDuplicate, err := u.userDao.IsNameTelDuplicate(ctx, userParam.Username, userParam.Tel, userParam.ID)
	if err != nil {
		xlogger.Errorf("查询用户名或电话是否存在异常: %v", err)
		return 0, errors.WithMessage(err, "查询用户名或电话是否存在异常")
	}
	if isDuplicate {
		return 0, errors.New(e.UserNameTelExistError.GetErrMsg())
	}

	// 检查设置的角色id是否存在
	if len(userParam.RoleIds) > 0 {
		//isExist, err := u.userDao.IsExistByRoleId(ctx, userParam.RoleIds)
		roleLen, err := u.userDao.CountRoleByIds(ctx, userParam.RoleIds)
		if err != nil {
			xlogger.Errorf("查询角色数量存在异常: %v", err)
			return 0, errors.WithMessage(err, "查询角色数量存在异常")
		}
		if roleLen != int64(len(userParam.RoleIds)) {
			return 0, errors.New("设置的角色不存在")
		}
	}

	err = u.db.WriteQuery().Transaction(func(tx *query.Query) error {
		// 更新用户
		err = u.userDao.TransactionUpdateUser(ctx, tx, userParam.ID, userParam.Username, userParam.Tel, userParam.Nickname)
		if err != nil {
			xlogger.Errorf("用户更新失败: %+v err: %v", userParam, err)
			return errors.WithMessage(err, "用户更新失败")
		}

		// 删除用户关联角色
		err = u.userDao.TransactionDeleteUserRole(ctx, tx, userParam.ID)
		if err != nil {
			xlogger.Errorf("用户与角色关联关系删除失败: %v", err)
			return errors.WithMessage(err, "用户与角色关联关系删除失败")
		}

		// 创建用户-role关联, 设置roleId才去创建
		if len(userParam.RoleIds) > 0 {
			userRoles := make([]*model.UserRole, 0, len(userParam.RoleIds))
			for _, roleId := range userParam.RoleIds {
				userRoles = append(userRoles, &model.UserRole{
					UserID: userParam.ID,
					RoleID: roleId,
				})
			}
			err = u.userDao.TransactionCreateUserRoleInBatches(ctx, tx, userRoles)
			if err != nil {
				xlogger.Errorf("用户与角色关联关系创建失败: %+v err: %v", userParam, err)
				return errors.WithMessage(err, "用户与角色关联关系创建失败")
			}
		}

		// 创建操作日志
		err = u.logService.CreateOperationLog(ctx, tx, log.OperationLogInput{
			OperatorID:   int32(userContext.UserId),
			OperatorName: userContext.Username,
			OperatorType: constants.OperatorUser,
			Resource:     constants.ResourceUser,
			ResourceID:   userParam.ID,
			TraceID:      userContext.TraceId,
			Action:       constants.ActionUpdate,
			BeforeData:   oldUser,
			AfterData:    userParam,
			Description: fmt.Sprintf("用户(%d-%s)修改了用户(%d-%s)信息",
				userContext.UserId, userContext.Username, userParam.ID, userParam.Username),
			IP: userContext.IP,
		})
		if err != nil {
			xlogger.Errorf("操作日志创建失败: %+v err: %v", userParam, err)
			return errors.WithMessage(err, "操作日志创建失败")
		}

		return nil
	})

	if err != nil {
		return 0, err
	}
	xlogger.Infof("用户更新成功: old=%+v new=%+v", oldUser, userParam)

	// 清除用户对应角色缓存
	cacheKey := fmt.Sprintf("%s%d", constants.CacheUserRolePrefix, userParam.ID)
	if _, err := u.cache.Delete(ctx, cacheKey); err != nil {
		xlogger.Errorf("清除用户对应角色缓存失败: %v", err)
	}
	return userParam.ID, nil
}

// GetUserById 根据id获取用户信息
func (u *UserService) GetUserById(ctx context.Context, userId int32) (*UserInfo, error) {
	if userId <= 0 {
		return nil, errors.New(e.UserNotFound.GetErrMsg())
	}
	user, err := u.userDao.GetUserById(ctx, userId)
	if err != nil {
		xlogger.Errorf("获取用户(%d)信息异常: %v", userId, err)
		return nil, errors.WithMessage(err, "用户信息查询失败")
	}

	// 查询角色信息
	roleList, err := u.userDao.GetRoleListByUserId(ctx, userId)
	if err != nil {
		return nil, errors.WithMessage(err, "用户角色信息查询失败")
	}
	roles := make([]*UserRole, 0, len(roleList))
	for _, role := range roleList {
		roles = append(roles, &UserRole{
			ID:   role.ID,
			Code: role.Code,
			Name: *role.Name,
		})
	}
	return &UserInfo{
		ID:        user.ID,
		Username:  user.Username,
		Tel:       user.Tel,
		Nickname:  *user.Nickname,
		Status:    *user.Status,
		RoleList:  roles,
		CreatedAt: *user.CreatedAt,
		UpdatedAt: *user.UpdatedAt,
	}, nil
}

// GetUserList 获取用户列表信息
func (u *UserService) GetUserList(ctx context.Context, condition *UserListCondition) (*UserList, error) {
	userList, total, err := u.userDao.GetUserList(ctx, &account.UserListCondition{
		Ids:      condition.Ids,
		Username: condition.Username,
		Tel:      condition.Tel,
		Nickname: condition.Nickname,
		Status:   condition.Status,
		Offset:   condition.Offset,
		Limit:    condition.Limit,
	})
	if err != nil {
		xlogger.Errorf("获取用户信息列表异常: %v", err)
		return nil, errors.WithMessage(err, "用户信息列表查询失败")
	}
	userInfoList := make([]*UserInfo, 0, len(userList))
	for _, user := range userList {
		userInfoList = append(userInfoList, &UserInfo{
			ID:        user.ID,
			Username:  user.Username,
			Tel:       user.Tel,
			Nickname:  *user.Nickname,
			Status:    *user.Status,
			CreatedAt: *user.CreatedAt,
			UpdatedAt: *user.UpdatedAt,
		})
	}
	return &UserList{List: userInfoList, Total: total}, nil
}

// DeleteById 删除用户
func (u *UserService) DeleteById(ctx context.Context, userId int32) error {
	// 获取登录ctx
	userContext, err := xauth.GetUserContext(ctx)
	if err != nil {
		return err
	}

	if userId <= 0 {
		return errors.New(e.UserNotFound.GetErrMsg())
	}
	err = u.db.WriteQuery().Transaction(func(tx *query.Query) error {
		// 删除用户
		err := u.userDao.TransactionDeleteById(ctx, tx, userId)
		if err != nil {
			xlogger.Errorf("用户删除异常: %v", err)
			return errors.WithMessage(err, "用户删除异常")
		}

		// 删除用户-角色关联关系
		err = u.userDao.TransactionDeleteUserRole(ctx, tx, userId)
		if err != nil {
			xlogger.Errorf("用户与角色关联关系删除失败: %v", err)
			return errors.WithMessage(err, "用户与角色关联关系删除失败")
		}

		// 创建操作日志
		err = u.logService.CreateOperationLog(ctx, tx, log.OperationLogInput{
			OperatorID:   int32(userContext.UserId),
			OperatorName: userContext.Username,
			OperatorType: constants.OperatorUser,
			Resource:     constants.ResourceUser,
			ResourceID:   userId,
			TraceID:      userContext.TraceId,
			Action:       constants.ActionDelete,
			BeforeData:   "",
			AfterData:    "",
			Description: fmt.Sprintf("用户(%d-%s)删除了用户(%d)",
				userContext.UserId, userContext.Username, userId),
			IP: userContext.IP,
		})
		if err != nil {
			xlogger.Errorf("操作日志创建失败: %v", err)
			return errors.WithMessage(err, "操作日志创建失败")
		}

		return nil
	})
	xlogger.Infof("用户删除成功: %d", userId)

	// 清除用户对应角色缓存
	cacheKey := fmt.Sprintf("%s%d", constants.CacheUserRolePrefix, userId)
	if _, err := u.cache.Delete(ctx, cacheKey); err != nil {
		xlogger.Errorf("清除用户对应角色缓存失败: %v", err)
	}
	return err
}

// ResetPwdById 重置用户密码
func (u *UserService) ResetPwdById(ctx context.Context, userId int32, password string) error {
	if userId <= 0 {
		return errors.New(e.UserNotFound.GetErrMsg())
	}

	_, err := u.userDao.GetUserById(ctx, userId)
	if err != nil {
		xlogger.Errorf("获取用户(%d)信息异常: %v", userId, err)
		return errors.WithMessage(err, "用户信息查询失败")
	}

	// 密码加密
	pwd, err := xcryption.HashPassword(password)
	if err != nil {
		xlogger.Errorf("密码加密异常: %v", err)
		return errors.WithMessage(err, "密码加密异常")
	}
	err = u.userDao.ResetPwdById(ctx, userId, pwd)
	if err != nil {
		xlogger.Errorf("修改用户(%d)密码异常: %v", userId, err)
		return errors.WithMessage(err, "用户信息查询失败")
	}
	return nil
}

// Authenticate 用户登录校验
func (u *UserService) Authenticate(ctx context.Context, username, password string) (*UserInfo, error) {
	if len(username) <= 0 {
		return nil, errors.New(e.UserNotFound.GetErrMsg())
	}

	user, err := u.userDao.GetUserByUsername(ctx, username)
	if err != nil {
		xlogger.Errorf("获取用户(%s)信息异常: %v", username, err)
		return nil, errors.WithMessage(err, "用户信息查询失败")
	}

	// 密码加密
	isOk := xcryption.CheckPassword(user.Password, password)
	if !isOk {
		return nil, errors.New(e.AuthError.GetErrMsg())
	}
	return &UserInfo{
		ID:        user.ID,
		Username:  user.Username,
		Tel:       user.Tel,
		Nickname:  *user.Nickname,
		Status:    *user.Status,
		CreatedAt: *user.CreatedAt,
		UpdatedAt: *user.UpdatedAt,
	}, nil
}

// GetRoleIdsByUserId 根据userId拿该用户角色
func (u *UserService) GetRoleIdsByUserId(ctx context.Context, userId int32) ([]int32, error) {
	var roleIds []int32
	if userId <= 0 {
		return roleIds, errors.New(e.UserNotFound.GetErrMsg())
	}

	// 读缓存 user->roleId
	cacheKey := fmt.Sprintf("%s%d", constants.CacheUserRolePrefix, userId)
	if data, err := u.cache.Get(ctx, cacheKey); err == nil && data != "" {
		if strErr := json.Unmarshal([]byte(data), &roleIds); strErr == nil {
			return roleIds, nil
		}
	}

	//  缓存未命中或解析失败：查库拿 RoleId
	roleIds, err := u.userDao.GetRoleIdsByUserId(ctx, userId)
	if err != nil {
		xlogger.Errorf("查询用户角色id失败 uid=%d: %v", userId, err)
		return roleIds, errors.WithMessage(err, "查询用户角色id失败")
	}

	// 写缓存
	roleIdsBytes, err := json.Marshal(&roleIds)
	if err != nil {
		xlogger.Errorf("缓存用户角色id失败 uid=%d, roleIds: %v, %v", userId, roleIds, err)
		return roleIds, errors.WithMessage(err, "缓存用户角色id失败")
	}
	_ = u.cache.Set(ctx, cacheKey, string(roleIdsBytes), constants.CacheUserRoleExpirationDay*24*time.Hour)

	return roleIds, nil
}

// GetPermsListById 根据userId拿该用户所有接口权限标识
func (u *UserService) GetPermsListById(ctx context.Context, userId int32) ([]string, error) {
	if userId <= 0 {
		return nil, errors.New(e.UserNotFound.GetErrMsg())
	}

	// 根据userId拿到roleId
	roleIds, err := u.GetRoleIdsByUserId(ctx, userId)
	if err != nil {
		xlogger.Errorf("GetPermsListById 查询用户角色失败 uid=%d: %v", userId, err)
		return nil, err
	}

	// 如果没分配角色，直接返回空 perms
	if len(roleIds) <= 0 {
		return []string{}, nil
	}

	// 根据roleId拿到接口的perms列表
	permsList := make([]string, 0, 10)
	for _, roleId := range roleIds {
		perms, err := u.roleService.GetRolePermsListByRuleID(ctx, roleId)
		if err != nil {
			xlogger.Errorf("GetPermsListById 获取角色权限失败 rid=%v: %v", roleIds, err)
			return nil, err
		}
		permsList = append(permsList, perms...)
	}
	return permsList, nil
}

// GetUserPermissionById 根据id获取用户信息
func (u *UserService) GetUserPermissionById(ctx context.Context, userId int32) (*UserPermissionInfo, error) {
	if userId <= 0 {
		return nil, errors.New(e.UserNotFound.GetErrMsg())
	}
	// 获取用户信息
	user, err := u.GetUserById(ctx, userId)
	if err != nil {
		return nil, err
	}

	// 获取用户权限信息
	menuList := make([]*MenuData, 0, 20)             // 菜单信息()
	permissionList := make([]*UserPermission, 0, 10) // 按钮权限信息
	menuRoots := make([]*MenuInfo, 0, 10)
	menuMap := make(map[int32]struct{}, 20)
	permMap := make(map[int32]struct{}, 10)

	if len(user.RoleList) > 0 {
		for _, role := range user.RoleList {
			menus, err := u.roleService.GetRoleMenuListByRuleID(ctx, role.ID)
			if err != nil {
				return nil, err
			}
			for _, menu := range menus {
				// 按钮放到perm下面，用与渲染页面按钮
				if menu.MenuType == constants.MenuTypeBtn && menu.Perms != "" {
					if _, exists := permMap[menu.ID]; !exists {
						permissionList = append(permissionList, &UserPermission{
							ID:    menu.ID,
							Name:  menu.Name,
							Perms: menu.Perms,
						})
						permMap[menu.ID] = struct{}{}
					}
				}
				// dir跟menu放到菜单信息下面，用与渲染数据
				if menu.MenuType == constants.MenuTypeMenu || menu.MenuType == constants.MenuTypeDir {
					if _, exists := menuMap[menu.ID]; !exists {
						menuMap[menu.ID] = struct{}{}
						menuList = append(menuList, menu)
					}
				}
			}
		}

		// 构造 map[id]MenuInfo
		nodeMap := make(map[int32]*MenuInfo, len(menuList))
		for _, m := range menuList {
			nodeMap[m.ID] = &MenuInfo{
				ID:        m.ID,
				ParentID:  m.ParentID,
				MenuType:  m.MenuType,
				Name:      m.Name,
				Path:      m.Path,
				Icon:      m.Icon,
				Perms:     m.Perms,
				OrderNum:  m.OrderNum,
				CreatedAt: m.CreatedAt,
				UpdatedAt: m.UpdatedAt,
				Children:  []*MenuInfo{},
			}
		}

		for _, node := range nodeMap {
			if node.ParentID == 0 {
				menuRoots = append(menuRoots, node)
			} else if parent, ok := nodeMap[node.ParentID]; ok {
				parent.Children = append(parent.Children, node)
			} else {
				xlogger.Errorf("菜单[%d] 的父节点 [%d] 不存在，挂到根节点", node.ID, node.ParentID)
				menuRoots = append(menuRoots, node)
			}
		}

		// 递归排序
		var sortNodes func(nodes []*MenuInfo)
		sortNodes = func(nodes []*MenuInfo) {
			if len(nodes) == 0 {
				return
			}
			sort.SliceStable(nodes, func(i, j int) bool {
				return nodes[i].OrderNum < nodes[j].OrderNum
			})
			for _, n := range nodes {
				sortNodes(n.Children)
			}
		}
		sortNodes(menuRoots)
	}

	return &UserPermissionInfo{
		ID:          user.ID,
		Username:    user.Username,
		Tel:         user.Tel,
		Nickname:    user.Nickname,
		Status:      user.Status,
		RoleList:    user.RoleList,
		Menus:       menuRoots,
		Permissions: permissionList,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}, nil
}
