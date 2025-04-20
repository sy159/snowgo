package account

import (
	"context"
	"github.com/pkg/errors"
	"snowgo/internal/constants"
	"snowgo/internal/dal/model"
	"snowgo/internal/dal/repo"
	"snowgo/internal/dao/account"
	"snowgo/pkg/xcache"
	"snowgo/pkg/xcryption"
	e "snowgo/pkg/xerror"
	"snowgo/pkg/xlogger"
	"time"
)

// UserRepo 定义User相关db操作接口
type UserRepo interface {
	CreateUser(ctx context.Context, user *model.User) (*model.User, error)
	IsNameTelDuplicate(ctx context.Context, username, tel string, userId int32) (bool, error)
	GetUserById(ctx context.Context, userId int32) (*model.User, error)
	GetUserList(ctx context.Context, condition *account.UserListCondition) ([]*model.User, int64, error)
	DeleteById(ctx context.Context, userId int32) error
}

type UserService struct {
	db      *repo.Repository
	userDao UserRepo
	cache   xcache.Cache
}

func NewUserService(db *repo.Repository, userDao UserRepo, cache xcache.Cache) *UserService {
	return &UserService{
		db:      db,
		cache:   cache,
		userDao: userDao,
	}
}

type User struct {
	Username string `json:"username" binding:"required,max=64"`
	Password string `json:"password"`
	Tel      string `json:"tel" binding:"required"`
	Nickname string `json:"nickname"`
}

type UserInfo struct {
	ID        int32
	Username  string
	Tel       string
	Nickname  string
	Status    string
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
func (u *UserService) CreateUser(ctx context.Context, user *User) (int32, error) {
	// 检查用户名，或者电话是否存在
	isDuplicate, err := u.userDao.IsNameTelDuplicate(ctx, user.Username, user.Tel, 0)
	if err != nil {
		xlogger.Errorf("查询用户名或电话是否存在异常: %v", err)
		return 0, errors.WithMessage(err, "查询用户名或电话是否存在异常")
	}
	if isDuplicate {
		return 0, errors.New(e.UserNameTelExistError.GetErrMsg())
	}
	// 创建用户
	pwd, err := xcryption.HashPassword(user.Password)
	if err != nil {
		xlogger.Errorf("密码加密异常: %v", err)
		return 0, errors.WithMessage(err, "密码加密异常")
	}
	activeStatus := constants.UserStatusActive
	userObj, err := u.userDao.CreateUser(ctx, &model.User{
		Username:  user.Username,
		Password:  pwd,
		Tel:       user.Tel,
		Nickname:  &user.Nickname,
		Status:    &activeStatus,
		IsDeleted: false,
	})
	if err != nil {
		xlogger.Errorf("用户创建失败: %+v err: %v", user, err)
		return 0, errors.WithMessage(err, "用户创建失败")
	}
	return userObj.ID, nil
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
	if userId <= 0 {
		return errors.New(e.UserNotFound.GetErrMsg())
	}
	err := u.userDao.DeleteById(ctx, userId)
	if err != nil {
		xlogger.Infof("用户删除异常: %v", err)
		return errors.WithMessage(err, "用户删除异常")
	}
	return nil
}
