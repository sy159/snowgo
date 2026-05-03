package account

import (
	"context"
	"errors"
	"gorm.io/gen"
	"gorm.io/gorm"
	"snowgo/internal/dal/model"
	"snowgo/internal/dal/query"
	"snowgo/internal/dal/repo"
	"strconv"
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
func (u *UserDao) CreateUser(ctx context.Context, user *model.SysUser) (*model.SysUser, error) {
	err := u.repo.Query().WithContext(ctx).SysUser.Create(user)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// TransactionCreateUser 创建用户
func (u *UserDao) TransactionCreateUser(ctx context.Context, tx *query.Query, user *model.SysUser) (*model.SysUser, error) {
	err := tx.WithContext(ctx).SysUser.Create(user)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// TransactionUpdateUser 更新用户
func (u *UserDao) TransactionUpdateUser(ctx context.Context, tx *query.Query, userId int32, username, tel, nickname string) error {
	_, err := tx.WithContext(ctx).SysUser.Where(tx.SysUser.ID.Eq(userId)).UpdateSimple(
		tx.SysUser.Username.Value(username),
		tx.SysUser.Tel.Value(tel),
		tx.SysUser.Nickname.Value(nickname),
	)
	if err != nil {
		return err
	}
	return nil
}

// TransactionCreateUserRole 创建用户-rule关联
func (u *UserDao) TransactionCreateUserRole(ctx context.Context, tx *query.Query, userRole *model.SysUserRole) error {
	err := tx.WithContext(ctx).SysUserRole.Create(userRole)
	if err != nil {
		return err
	}
	return nil
}

// TransactionCreateUserRoleInBatches 创建用户-rule关联
func (u *UserDao) TransactionCreateUserRoleInBatches(ctx context.Context, tx *query.Query, userRoleList []*model.SysUserRole) error {
	err := tx.WithContext(ctx).SysUserRole.CreateInBatches(userRoleList, 100)
	if err != nil {
		return err
	}
	return nil
}

// TransactionDeleteUserRole 删除用户与角色关联关系
func (u *UserDao) TransactionDeleteUserRole(ctx context.Context, tx *query.Query, userId int32) error {
	_, err := tx.WithContext(ctx).SysUserRole.Where(tx.SysUserRole.UserID.Eq(userId)).Delete()
	if err != nil {
		return err
	}
	return nil
}

// TransactionDeleteById 删除用户
func (u *UserDao) TransactionDeleteById(ctx context.Context, tx *query.Query, userId int32) error {
	if userId <= 0 {
		return errors.New("用户id不存在")
	}
	_, err := tx.SysUser.WithContext(ctx).Where(tx.SysUser.ID.Eq(userId)).UpdateSimple(tx.SysUser.IsDeleted.Value(true))
	if err != nil {
		return err
	}
	return nil
}

func (u *UserDao) GetRoleListByUserId(ctx context.Context, userId int32) ([]*model.SysRole, error) {
	if userId <= 0 {
		return nil, errors.New("用户id不存在")
	}
	m := u.repo.Query().SysUserRole
	role := u.repo.Query().SysRole

	var userRoles []*model.SysRole
	err := m.WithContext(ctx).
		Where(m.UserID.Eq(userId)).
		LeftJoin(role, m.RoleID.EqCol(role.ID)).
		Select(role.ALL).
		Scan(&userRoles)
	return userRoles, err
}

// GetRoleIdsByUserId 只返回 roleId，如果没找到记录则返回 0
func (u *UserDao) GetRoleIdsByUserId(ctx context.Context, userId int32) ([]int32, error) {
	var roleIds []int32
	if userId <= 0 {
		return roleIds, errors.New("用户 id 不合法")
	}

	m := u.repo.Query().SysUserRole
	err := m.WithContext(ctx).Select(m.RoleID).Where(m.UserID.Eq(userId)).Scan(&roleIds)
	if err != nil {
		return roleIds, err
	}
	return roleIds, nil
}

// IsNameTelDuplicate 用户名或者电话是否存在了,如果有用户id应该排除
func (u *UserDao) IsNameTelDuplicate(ctx context.Context, username, tel string, userId int32) (bool, error) {
	m := u.repo.Query().SysUser
	userQuery := m.WithContext(ctx).
		Select(m.ID).
		Where(m.IsDeleted.Is(false)).
		Where(m.WithContext(ctx).Or(m.Username.Eq(username)).Or(m.Tel.Eq(tel)))
	if userId > 0 {
		userQuery = userQuery.Where(m.ID.Neq(userId))
	}
	_, err := userQuery.First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return true, err
	}
	return true, nil
}

// IsExistByRoleId roleId是否存在
func (u *UserDao) IsExistByRoleId(ctx context.Context, roleId int32) (bool, error) {
	if roleId < 0 {
		return true, errors.New("角色不存在")
	}
	m := u.repo.Query().SysRole
	_, err := m.WithContext(ctx).Select(m.ID).Where(m.ID.Eq(roleId)).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return true, err
	}
	return true, nil
}

// CountRoleByIds 根据role ids，获取数量
func (u *UserDao) CountRoleByIds(ctx context.Context, roleIds []int32) (int64, error) {
	m := u.repo.Query().SysRole
	return m.WithContext(ctx).Select(m.ID).Where(m.ID.In(roleIds...)).Count()
}

// GetUserById 查询用户by id
func (u *UserDao) GetUserById(ctx context.Context, userId int32) (*model.SysUser, error) {
	if userId <= 0 {
		return nil, errors.New("用户id不存在")
	}
	m := u.repo.Query().SysUser
	user, err := m.WithContext(ctx).Where(m.ID.Eq(userId), m.IsDeleted.Is(false)).First()
	if err != nil {
		return nil, err
	}
	return user, nil
}

// GetUserByUsername 查询用户by name
func (u *UserDao) GetUserByUsername(ctx context.Context, username string) (*model.SysUser, error) {
	if len(username) <= 0 {
		return nil, errors.New("用户username不存在")
	}
	m := u.repo.Query().SysUser
	user, err := m.WithContext(ctx).Where(m.Username.Eq(username), m.IsDeleted.Is(false)).First()
	if err != nil {
		return nil, err
	}
	return user, nil
}

// GetUserList 获取用户列表
func (u *UserDao) GetUserList(ctx context.Context, condition *UserListCondition) ([]*model.SysUser, int64, error) {
	m := u.repo.Query().SysUser
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
		return nil, 0, err
	}
	return userList, total, nil
}

func (u *UserDao) UserIdsScope(userIds []int32) func(tx gen.Dao) gen.Dao {
	return func(tx gen.Dao) gen.Dao {
		if len(userIds) == 0 {
			return tx
		}
		m := u.repo.Query().SysUser
		tx = tx.Where(m.ID.In(userIds...))
		return tx
	}
}

func (u *UserDao) UserNameScope(username string) func(tx gen.Dao) gen.Dao {
	return func(tx gen.Dao) gen.Dao {
		if len(username) == 0 {
			return tx
		}
		m := u.repo.Query().SysUser
		tx = tx.Where(m.Username.Eq(username))
		return tx
	}
}

func (u *UserDao) TelScope(tel string) func(tx gen.Dao) gen.Dao {
	return func(tx gen.Dao) gen.Dao {
		if len(tel) == 0 {
			return tx
		}
		m := u.repo.Query().SysUser
		tx = tx.Where(m.Tel.Eq(tel))
		return tx
	}
}

func (u *UserDao) StatusScope(status string) func(tx gen.Dao) gen.Dao {
	return func(tx gen.Dao) gen.Dao {
		if len(status) == 0 {
			return tx
		}
		m := u.repo.Query().SysUser
		statusVal, err := strconv.ParseUint(status, 10, 8)
		if err != nil {
			return tx // 非法值直接返回
		}
		tx = tx.Where(m.Status.Eq(uint8(statusVal)))
		return tx
	}
}

func (u *UserDao) NickNameScope(nickname string) func(tx gen.Dao) gen.Dao {
	return func(tx gen.Dao) gen.Dao {
		if len(nickname) == 0 {
			return tx
		}
		m := u.repo.Query().SysUser
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
	m := u.repo.Query().SysUser
	_, err := m.WithContext(ctx).Where(m.ID.Eq(userId)).UpdateSimple(m.IsDeleted.Value(true))
	if err != nil {
		return err
	}
	return nil
}

// UpdateUser 更新用户
// UpdateSimple: gen独有的，会处理零值问题、会调用Hook、并且更新时间戳
// UpdateColumnSimple：gen独有的，会处理零值问题、不是调用Hook、不会更新时间戳、性能更高，类似执行原生sql
// Save: 会处理零值问题，但是是全部更新
// Updates: 不会处理零值问题、会调用Hook、并且更新时间戳（map时会更新零值）
// UpdateColumns: 会处理零值问题、不会调用Hook、不会更新时间戳、性能更高，类似执行原生sql，跟save差不多，不过不会更新时间戳
func (u *UserDao) UpdateUser(ctx context.Context, user *model.SysUser) (*model.SysUser, error) {
	if user.ID <= 0 {
		return nil, errors.New("用户id不存在")
	}
	m := u.repo.Query().SysUser
	err := m.WithContext(ctx).Where(m.ID.Eq(user.ID)).Save(user)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// ResetPwdById 重置密码
func (u *UserDao) ResetPwdById(ctx context.Context, userId int32, password string) error {
	if userId <= 0 {
		return errors.New("用户id不存在")
	}
	m := u.repo.Query().SysUser
	_, err := m.WithContext(ctx).Where(m.ID.Eq(userId)).UpdateSimple(m.Password.Value(password))
	if err != nil {
		return err
	}
	return nil
}

// TransactionResetPwdById 事务内重置密码
func (u *UserDao) TransactionResetPwdById(ctx context.Context, tx *query.Query, userId int32, password string) error {
	if userId <= 0 {
		return errors.New("用户id不存在")
	}
	_, err := tx.WithContext(ctx).SysUser.Where(tx.SysUser.ID.Eq(userId)).UpdateSimple(tx.SysUser.Password.Value(password))
	if err != nil {
		return err
	}
	return nil
}
