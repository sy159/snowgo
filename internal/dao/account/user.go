package account

import (
	"context"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"snowgo/internal/dal/model"
	"snowgo/internal/dal/repo"
)

type UserDao struct {
	repo *repo.Repository
}

func NewUserDao() *UserDao {
	return &UserDao{
		repo: repo.NewRepository(),
	}
}

// CreateUser 创建用户
func (u *UserDao) CreateUser(ctx context.Context, user *model.User) (*model.User, error) {
	err := u.repo.Query().WithContext(ctx).User.Create(user)
	if err != nil {
		return nil, errors.WithMessage(err, "用户创建失败")
	}
	return user, nil
}

// GetUserById 查询用户by id
func (u *UserDao) GetUserById(ctx context.Context, userId int32) (*model.User, error) {
	if userId <= 0 {
		return nil, errors.New("用户id不存在")
	}
	m := u.repo.Query().User
	user, err := u.repo.Query().User.WithContext(ctx).Where(m.ID.Eq(userId)).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("该用户不存在")
		}
		return nil, errors.WithMessage(err, "用户查询异常")
	}
	return user, nil
}
