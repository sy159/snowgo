package account

import (
	"context"
	"github.com/pkg/errors"
	"gorm.io/gen"
	"gorm.io/gorm"
	"snowgo/internal/dal/model"
	"snowgo/internal/dal/repo"
)

// UserDao UserRepo接口实现
type UserDao struct {
	repo *repo.Repository
}

func NewUserDao(repo *repo.Repository) *UserDao {
	return &UserDao{
		repo: repo,
	}
}

type UserListCondition struct {
	Ids      []int32 `json:"ids"`
	Username string  `json:"username"`
	Tel      string  `json:"tel"`
	Nickname string  `json:"nickname"`
	Status   string  `json:"status"`
	Offset   int32   `json:"offset"`
	Limit    int32   `json:"limit"`
}

// CreateUser 创建用户
func (u *UserDao) CreateUser(ctx context.Context, user *model.User) (*model.User, error) {
	err := u.repo.Query().WithContext(ctx).User.Create(user)
	if err != nil {
		return nil, errors.WithMessage(err, "用户创建失败")
	}
	return user, nil
}

// IsNameTelDuplicate 用户名或者电话是否存在了,如果有用户id应该排除
func (u *UserDao) IsNameTelDuplicate(ctx context.Context, username, tel string, userId int32) (bool, error) {
	m := u.repo.Query().User
	query := m.WithContext(ctx).
		Select(m.ID).
		Where(m.IsDeleted.Is(false)).
		Where(m.WithContext(ctx).Or(m.Username.Eq(username)).Or(m.Tel.Eq(tel)))
	if userId > 0 {
		query = query.Where(m.ID.Neq(userId))
	}
	_, err := query.First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return true, errors.WithStack(err)
	}
	return true, nil
}

// GetUserById 查询用户by id
func (u *UserDao) GetUserById(ctx context.Context, userId int32) (*model.User, error) {
	if userId <= 0 {
		return nil, errors.New("用户id不存在")
	}
	m := u.repo.Query().User
	user, err := u.repo.Query().User.WithContext(ctx).Where(m.ID.Eq(userId), m.IsDeleted.Is(false)).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("该用户不存在")
		}
		return nil, errors.WithMessage(err, "用户查询异常")
	}
	return user, nil
}

// GetUserList 获取用户列表
func (u *UserDao) GetUserList(ctx context.Context, condition *UserListCondition) ([]*model.User, int64, error) {
	m := u.repo.Query().User
	userList, total, err := m.WithContext(ctx).
		Where(m.IsDeleted.Is(false)).
		Scopes(
			u.UserIdsScope(condition.Ids),
			u.UserNameScope(condition.Username),
			u.TelScope(condition.Tel),
			u.StatusScope(condition.Status),
			u.NickNameScope(condition.Nickname),
		).
		FindByPage(int(condition.Offset), int(condition.Limit))
	if err != nil {
		return nil, 0, errors.WithMessage(err, "用户列表查询异常")
	}
	return userList, total, nil
}

func (u *UserDao) UserIdsScope(userIds []int32) func(tx gen.Dao) gen.Dao {
	return func(tx gen.Dao) gen.Dao {
		if len(userIds) == 0 {
			return tx
		}
		m := u.repo.Query().User
		tx = tx.Where(m.ID.In(userIds...))
		return tx
	}
}

func (u *UserDao) UserNameScope(username string) func(tx gen.Dao) gen.Dao {
	return func(tx gen.Dao) gen.Dao {
		if len(username) == 0 {
			return tx
		}
		m := u.repo.Query().User
		tx = tx.Where(m.Username.Eq(username))
		return tx
	}
}

func (u *UserDao) TelScope(tel string) func(tx gen.Dao) gen.Dao {
	return func(tx gen.Dao) gen.Dao {
		if len(tel) == 0 {
			return tx
		}
		m := u.repo.Query().User
		tx = tx.Where(m.Tel.Eq(tel))
		return tx
	}
}

func (u *UserDao) StatusScope(status string) func(tx gen.Dao) gen.Dao {
	return func(tx gen.Dao) gen.Dao {
		if len(status) == 0 {
			return tx
		}
		m := u.repo.Query().User
		tx = tx.Where(m.Status.Eq(status))
		return tx
	}
}

func (u *UserDao) NickNameScope(nickname string) func(tx gen.Dao) gen.Dao {
	return func(tx gen.Dao) gen.Dao {
		if len(nickname) == 0 {
			return tx
		}
		m := u.repo.Query().User
		tx = tx.Where(m.Nickname.Like("%" + nickname + "%"))
		return tx
	}
}

// DeleteById 删除用户by id
// UpdateSimple: gen独有的，会处理零值问题、会调用Hook、并且更新时间戳
// UpdateColumnSimple：gen独有的，会处理零值问题、不是调用Hook、不会更新时间戳、性能更高，类似执行原生sql
// Save: 会处理零值问题，但是是全部更新
// Updates: 不会处理零值问题、会调用Hook、并且更新时间戳（map时会更新零值）
// UpdateColumns: 会处理零值问题、不会调用Hook、不会更新时间戳、性能更高，类似执行原生sql，跟save差不多，不过不会更新时间戳
func (u *UserDao) DeleteById(ctx context.Context, userId int32) error {
	if userId <= 0 {
		return errors.New("用户id不存在")
	}
	m := u.repo.Query().User
	_, err := u.repo.Query().User.WithContext(ctx).Where(m.ID.Eq(userId)).UpdateSimple(m.IsDeleted.Value(true))
	if err != nil {
		return errors.WithMessage(err, "用户删除异常")
	}
	return nil
}

// UpdateUser 更新用户
// UpdateSimple: gen独有的，会处理零值问题、会调用Hook、并且更新时间戳
// UpdateColumnSimple：gen独有的，会处理零值问题、不是调用Hook、不会更新时间戳、性能更高，类似执行原生sql
// Save: 会处理零值问题，但是是全部更新
// Updates: 不会处理零值问题、会调用Hook、并且更新时间戳（map时会更新零值）
// UpdateColumns: 会处理零值问题、不会调用Hook、不会更新时间戳、性能更高，类似执行原生sql，跟save差不多，不过不会更新时间戳
func (u *UserDao) UpdateUser(ctx context.Context, user *model.User) (*model.User, error) {
	if user.ID <= 0 {
		return nil, errors.New("用户id不存在")
	}
	m := u.repo.Query().User
	err := u.repo.Query().User.WithContext(ctx).Where(m.ID.Eq(user.ID)).Save(user)
	if err != nil {
		return nil, errors.WithMessage(err, "用户删除异常")
	}
	return user, nil
}
