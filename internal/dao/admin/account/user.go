package account

import (
	"context"
	"errors"
	"gorm.io/gen"
	"gorm.io/gen/field"
	"gorm.io/gorm"
	"snowgo/internal/dal/model"
	"snowgo/internal/dal/query"
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
	Status   *int8   `json:"status"`
	Offset   int32   `json:"offset"`
	Limit    int32   `json:"limit"`
}

// CreateUser 创建用户
func (u *UserDao) CreateUser(ctx context.Context, q *query.Query, user *model.SysUser) (*model.SysUser, error) {
	err := q.WithContext(ctx).SysUser.Create(user)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// UpdateUser 更新用户（根据传入的 model.SysUser 非零值字段进行部分更新）
// UpdateSimple: gen独有的，会处理零值问题、会调用Hook、并且更新时间戳
func (u *UserDao) UpdateUser(ctx context.Context, q *query.Query, user *model.SysUser) (*model.SysUser, error) {
	if user.ID <= 0 {
		return nil, errors.New("用户id不存在")
	}
	m := q.WithContext(ctx).SysUser.Where(q.SysUser.ID.Eq(user.ID))
	clauses := []field.AssignExpr{
		q.SysUser.Username.Value(user.Username),
		q.SysUser.Tel.Value(user.Tel),
	}
	if user.Nickname != nil {
		clauses = append(clauses, q.SysUser.Nickname.Value(*user.Nickname))
	}
	if user.Email != nil {
		clauses = append(clauses, q.SysUser.Email.Value(*user.Email))
	}
	if user.Remark != nil {
		clauses = append(clauses, q.SysUser.Remark.Value(*user.Remark))
	}
	if user.Status != nil {
		clauses = append(clauses, q.SysUser.Status.Value(*user.Status))
	}
	if user.UpdatedBy != nil {
		clauses = append(clauses, q.SysUser.UpdatedBy.Value(*user.UpdatedBy))
	}
	_, err := m.UpdateSimple(clauses...)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// CreateUserRole 创建用户-role关联
func (u *UserDao) CreateUserRole(ctx context.Context, q *query.Query, userRole *model.SysUserRole) error {
	err := q.WithContext(ctx).SysUserRole.Create(userRole)
	if err != nil {
		return err
	}
	return nil
}

// CreateUserRoleInBatches 创建用户-role关联
func (u *UserDao) CreateUserRoleInBatches(ctx context.Context, q *query.Query, userRoleList []*model.SysUserRole) error {
	err := q.WithContext(ctx).SysUserRole.CreateInBatches(userRoleList, 100)
	if err != nil {
		return err
	}
	return nil
}

// DeleteUserRole 删除用户与角色关联关系
func (u *UserDao) DeleteUserRole(ctx context.Context, q *query.Query, userId int32) error {
	_, err := q.WithContext(ctx).SysUserRole.Where(q.SysUserRole.UserID.Eq(userId)).Delete()
	if err != nil {
		return err
	}
	return nil
}

// DeleteById 删除用户by id（硬删除）
func (u *UserDao) DeleteById(ctx context.Context, q *query.Query, userId int32) error {
	if userId <= 0 {
		return errors.New("用户id不存在")
	}
	_, err := q.SysUser.WithContext(ctx).Where(q.SysUser.ID.Eq(userId)).Delete()
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

// IsNameTelDuplicate 用户名或者电话是否存在,如果有用户id应该排除
func (u *UserDao) IsNameTelDuplicate(ctx context.Context, username, tel string, userId int32) (bool, error) {
	m := u.repo.Query().SysUser
	userQuery := m.WithContext(ctx).
		Select(m.ID).
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
func (u *UserDao) CountRoleByIds(ctx context.Context, q *query.Query, roleIds []int32) (int64, error) {
	m := q.SysRole
	return m.WithContext(ctx).Select(m.ID).Where(m.ID.In(roleIds...)).Count()
}

// GetUserById 查询用户by id
func (u *UserDao) GetUserById(ctx context.Context, userId int32) (*model.SysUser, error) {
	if userId <= 0 {
		return nil, errors.New("用户id不存在")
	}
	m := u.repo.Query().SysUser
	user, err := m.WithContext(ctx).Where(m.ID.Eq(userId)).First()
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
	user, err := m.WithContext(ctx).Where(m.Username.Eq(username)).First()
	if err != nil {
		return nil, err
	}
	return user, nil
}

// GetUserList 获取用户列表
func (u *UserDao) GetUserList(ctx context.Context, condition *UserListCondition) ([]*model.SysUser, int64, error) {
	m := u.repo.Query().SysUser
	userList, total, err := m.WithContext(ctx).
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
		tx = tx.Where(m.Username.Like("%" + username + "%"))
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

func (u *UserDao) StatusScope(status *int8) func(tx gen.Dao) gen.Dao {
	return func(tx gen.Dao) gen.Dao {
		if status == nil {
			return tx
		}
		m := u.repo.Query().SysUser
		tx = tx.Where(m.Status.Eq(*status))
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

// ResetPwdById 重置密码
func (u *UserDao) ResetPwdById(ctx context.Context, q *query.Query, userId int32, password string) error {
	if userId <= 0 {
		return errors.New("用户id不存在")
	}
	_, err := q.WithContext(ctx).SysUser.Where(q.SysUser.ID.Eq(userId)).UpdateSimple(q.SysUser.Password.Value(password))
	if err != nil {
		return err
	}
	return nil
}
