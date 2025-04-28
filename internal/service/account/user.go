package account

import (
	"context"
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
	"strconv"
	"time"
)

// UserRepo 定义User相关db操作接口
type UserRepo interface {
	CreateUser(ctx context.Context, user *model.User) (*model.User, error)
	TransactionCreateUser(ctx context.Context, tx *query.Query, user *model.User) (*model.User, error)
	TransactionUpdateUser(ctx context.Context, tx *query.Query, userId int32, username, tel, nickname string) error
	TransactionCreateUserRole(ctx context.Context, tx *query.Query, userRole *model.UserRole) error
	TransactionDeleteUserRole(ctx context.Context, tx *query.Query, userId int32) error
	TransactionDeleteById(ctx context.Context, tx *query.Query, userId int32) error
	GetRoleByUserId(ctx context.Context, userId int32) (*account.UserRoleInfo, error)
	GetRoleIdByUserId(ctx context.Context, userId int32) (int32, error)
	IsNameTelDuplicate(ctx context.Context, username, tel string, userId int32) (bool, error)
	IsExistByRoleId(ctx context.Context, roleId int32) (bool, error)
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
	ID       int32  `json:"id"`
	Username string `json:"username" binding:"required,max=64"`
	Password string `json:"password"`
	Tel      string `json:"tel" binding:"required"`
	Nickname string `json:"nickname"`
	RoleId   int32  `json:"role_id"`
}

type UserInfo struct {
	ID        int32
	Username  string
	Tel       string
	Nickname  string
	Status    string
	RoleId    int32
	RoleName  string
	RoleCode  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type UserList struct {
	List  []*UserInfo
	Total int64
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
	if userParam.RoleId > 0 {
		isExist, err := u.userDao.IsExistByRoleId(ctx, userParam.RoleId)
		if err != nil {
			xlogger.Errorf("查询角色id存在异常: %v", err)
			return 0, errors.WithMessage(err, "查询角色id存在异常")
		}
		if !isExist {
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
		if userParam.RoleId > 0 {
			err = u.userDao.TransactionCreateUserRole(ctx, tx, &model.UserRole{
				UserID: userObj.ID,
				RoleID: userParam.RoleId,
			})
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
		xlogger.Infof("获取用户(%d)信息异常: %v", userParam.ID, err)
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
	if userParam.RoleId > 0 {
		isExist, err := u.userDao.IsExistByRoleId(ctx, userParam.RoleId)
		if err != nil {
			xlogger.Errorf("查询角色id存在异常: %v", err)
			return 0, errors.WithMessage(err, "查询角色id存在异常")
		}
		if !isExist {
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
		if userParam.RoleId > 0 {
			err = u.userDao.TransactionCreateUserRole(ctx, tx, &model.UserRole{
				UserID: userParam.ID,
				RoleID: userParam.RoleId,
			})
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
		xlogger.Infof("获取用户(%d)信息异常: %v", userId, err)
		return nil, errors.WithMessage(err, "用户信息查询失败")
	}

	// 查询角色信息
	role, err := u.userDao.GetRoleByUserId(ctx, userId)
	if err != nil {
		return nil, errors.WithMessage(err, "用户角色信息查询失败")
	}
	var roleId int32
	roleCode := ""
	roleName := ""
	if role != nil {
		roleId = role.RoleId
		roleCode = role.RoleCode
		roleName = role.RoleName
	}
	return &UserInfo{
		ID:        user.ID,
		Username:  user.Username,
		Tel:       user.Tel,
		Nickname:  *user.Nickname,
		Status:    *user.Status,
		RoleId:    roleId,
		RoleCode:  roleCode,
		RoleName:  roleName,
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
		xlogger.Infof("获取用户信息列表异常: %v", err)
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
			xlogger.Infof("用户删除异常: %v", err)
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
		xlogger.Infof("获取用户(%d)信息异常: %v", userId, err)
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
		xlogger.Infof("修改用户(%d)密码异常: %v", userId, err)
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
		xlogger.Infof("获取用户(%s)信息异常: %v", username, err)
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

// GetRoleIdByUserId 根据userId拿该用户角色
func (u *UserService) GetRoleIdByUserId(ctx context.Context, userId int32) (int32, error) {
	if userId <= 0 {
		return 0, errors.New(e.UserNotFound.GetErrMsg())
	}

	// 读缓存 user->roleId
	cacheKey := fmt.Sprintf("%s%d", constants.CacheUserRolePrefix, userId)
	if data, err := u.cache.Get(ctx, cacheKey); err == nil && data != "" {
		if roleId, strErr := strconv.Atoi(data); strErr == nil {
			return int32(roleId), nil
		}
	}

	//  缓存未命中或解析失败：查库拿 RoleId
	roleId, err := u.userDao.GetRoleIdByUserId(ctx, userId)
	if err != nil {
		xlogger.Errorf("查询用户角色id失败 uid=%d: %v", userId, err)
		return 0, errors.WithMessage(err, "查询用户角色id失败")
	}

	// 写缓存（即便 roleId=0 也写，防止下次打表）
	_ = u.cache.Set(ctx, cacheKey, strconv.Itoa(int(roleId)), constants.CacheUserRoleExpirationDay*24*time.Hour)

	return roleId, nil
}

// GetPermsListById 根据userId拿该用户所有接口权限标识
func (u *UserService) GetPermsListById(ctx context.Context, userId int32) ([]string, error) {
	if userId <= 0 {
		return nil, errors.New(e.UserNotFound.GetErrMsg())
	}

	// 根据userId拿到roleId
	roleId, err := u.GetRoleIdByUserId(ctx, userId)
	if err != nil {
		xlogger.Errorf("GetPermsListById 查询用户角色失败 uid=%d: %v", userId, err)
		return nil, err
	}

	// 如果没分配角色，直接返回空 perms
	if roleId <= 0 {
		return []string{}, nil
	}

	// 根据roleId拿到接口的perms列表
	permsList, err := u.roleService.GetRolePermsListByRuleID(ctx, roleId)
	if err != nil {
		xlogger.Errorf("GetPermsListById 获取角色权限失败 rid=%d: %v", roleId, err)
		return nil, err
	}
	return permsList, nil
}
