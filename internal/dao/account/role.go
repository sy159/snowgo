package account

import (
	"context"
	"github.com/pkg/errors"
	"gorm.io/gen"
	"gorm.io/gorm"
	"snowgo/internal/dal/model"
	"snowgo/internal/dal/query"
	"snowgo/internal/dal/repo"
	"time"
)

type RoleDao struct {
	repo *repo.Repository
}

func NewRoleDao(repo *repo.Repository) *RoleDao {
	return &RoleDao{repo: repo}
}

type RoleListCondition struct {
	Ids    []int32 `json:"ids"`
	Name   string  `json:"name"`
	Code   string  `json:"code"`
	Status string  `json:"status"`
	Offset int32   `json:"offset"`
	Limit  int32   `json:"limit"`
}

// RoleMenuPerm 用于接收 JOIN 查询结果
type RoleMenuPerm struct {
	RoleID int32  `gorm:"column:role_id"`
	Perms  string `gorm:"column:perms"`
}

type RoleMenuInfo struct {
	RoleID    int32     `gorm:"column:role_id" json:"role_id"`
	ID        int32     `json:"id"`
	ParentID  int32     `json:"parent_id"`
	MenuType  string    `json:"menu_type"`
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	Icon      string    `json:"icon"`
	Perms     string    `json:"perms"`
	OrderNum  int32     `json:"order_num"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (r *RoleDao) IsCodeExists(ctx context.Context, code string, roleId int32) (bool, error) {
	m := r.repo.Query().Role
	roleQuery := m.WithContext(ctx).Select(m.ID).Where(m.Code.Eq(code))
	if roleId > 0 {
		roleQuery = roleQuery.Where(m.ID.Neq(roleId))
	}
	_, err := roleQuery.First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return true, errors.WithStack(err)
	}
	return true, nil
}

// CreateRole 创建角色
func (r *RoleDao) CreateRole(ctx context.Context, role *model.Role) (*model.Role, error) {
	err := r.repo.Query().WithContext(ctx).Role.Create(role)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return role, nil
}

// TransactionCreateRole 事务创建角色
func (r *RoleDao) TransactionCreateRole(ctx context.Context, tx *query.Query, role *model.Role) (*model.Role, error) {
	err := tx.WithContext(ctx).Role.Create(role)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return role, nil
}

// UpdateRole 更新角色
func (r *RoleDao) UpdateRole(ctx context.Context, role *model.Role) (*model.Role, error) {
	if role.ID <= 0 {
		return nil, errors.New("角色id不存在")
	}
	m := r.repo.Query().Role
	err := m.WithContext(ctx).Where(m.ID.Eq(role.ID)).Save(role)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return role, nil
}

// DeleteById 删除角色
func (r *RoleDao) DeleteById(ctx context.Context, roleId int32) error {
	if roleId <= 0 {
		return errors.New("角色id不存在")
	}
	m := r.repo.Query().Role
	_, err := m.WithContext(ctx).Where(m.ID.Eq(roleId)).Delete()
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// GetRoleById 查询角色
func (r *RoleDao) GetRoleById(ctx context.Context, roleId int32) (*model.Role, error) {
	if roleId <= 0 {
		return nil, errors.New("角色id无效")
	}
	m := r.repo.Query().Role
	role, err := m.WithContext(ctx).Where(m.ID.Eq(roleId)).First()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return role, nil
}

// GetRoleList 角色列表
func (r *RoleDao) GetRoleList(ctx context.Context, cond *RoleListCondition) ([]*model.Role, int64, error) {
	m := r.repo.Query().Role
	list, total, err := m.WithContext(ctx).
		Scopes(
			r.NameScope(cond.Name),
			r.CodeScope(cond.Code),
			r.IdsScope(cond.Ids),
		).FindByPage(int(cond.Offset), int(cond.Limit))
	if err != nil {
		return nil, 0, errors.WithStack(err)
	}
	return list, total, nil
}

func (r *RoleDao) NameScope(name string) func(tx gen.Dao) gen.Dao {
	return func(tx gen.Dao) gen.Dao {
		if name == "" {
			return tx
		}
		m := r.repo.Query().Role
		return tx.Where(m.Name.Like("%" + name + "%"))
	}
}

func (r *RoleDao) CodeScope(code string) func(tx gen.Dao) gen.Dao {
	return func(tx gen.Dao) gen.Dao {
		if code == "" {
			return tx
		}
		m := r.repo.Query().Role
		return tx.Where(m.Code.Eq(code))
	}
}

func (r *RoleDao) IdsScope(ids []int32) func(tx gen.Dao) gen.Dao {
	return func(tx gen.Dao) gen.Dao {
		if len(ids) == 0 {
			return tx
		}
		m := r.repo.Query().Role
		return tx.Where(m.ID.In(ids...))
	}
}

// TransactionCreateRoleMenu 创建角色与菜单关联关系
func (r *RoleDao) TransactionCreateRoleMenu(ctx context.Context, tx *query.Query, roleMenuList []*model.RoleMenu) error {
	err := tx.WithContext(ctx).RoleMenu.CreateInBatches(roleMenuList, 1000)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// TransactionDeleteRoleMenu 删除角色与菜单关联关系
func (r *RoleDao) TransactionDeleteRoleMenu(ctx context.Context, tx *query.Query, roleId int32) error {
	_, err := tx.WithContext(ctx).RoleMenu.Where(tx.RoleMenu.RoleID.Eq(roleId)).Delete()
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// TransactionDeleteById 删除角色
func (r *RoleDao) TransactionDeleteById(ctx context.Context, tx *query.Query, roleId int32) error {
	if roleId <= 0 {
		return errors.New("角色id不存在")
	}
	_, err := tx.WithContext(ctx).Role.Where(tx.Role.ID.Eq(roleId)).Delete()
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// IsUsedUserByIds 判断角色是否被使用过
func (r *RoleDao) IsUsedUserByIds(ctx context.Context, roleId int32) (bool, error) {
	m := r.repo.Query().UserRole
	_, err := m.WithContext(ctx).Select(m.ID).Where(m.RoleID.Eq(roleId)).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return true, errors.WithStack(err)
	}
	return true, nil
}

// CountMenuByIds 根据菜单ids，获取数量
func (r *RoleDao) CountMenuByIds(ctx context.Context, ids []int32) (int64, error) {
	m := r.repo.Query().Menu
	return m.WithContext(ctx).Where(m.ID.In(ids...)).Count()
}

// GetMenuIdsByRoleId 根据roleId 获取关联的菜单id
func (r *RoleDao) GetMenuIdsByRoleId(ctx context.Context, roleId int32) ([]int32, error) {
	m := r.repo.Query().RoleMenu
	menuIds := make([]int32, 0, 10)
	err := m.WithContext(ctx).Select(m.MenuID).Where(m.RoleID.Eq(roleId)).Pluck(m.MenuID, &menuIds)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return menuIds, nil
}

// GetMenuPermsByRoleId 根据roleId 获取关联的菜单perms
func (r *RoleDao) GetMenuPermsByRoleId(ctx context.Context, roleId int32) ([]string, error) {
	m := r.repo.Query().RoleMenu
	menu := r.repo.Query().Menu
	menuPermsList := make([]string, 0, 10)
	err := m.WithContext(ctx).
		Join(menu, m.MenuID.EqCol(menu.ID)).
		Where(m.RoleID.Eq(roleId), menu.Perms.IsNotNull(), menu.Perms.Neq("")).
		Select(menu.Perms).
		Pluck(menu.Perms, &menuPermsList)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return menuPermsList, nil
}

// GetMenuListByRoleId 根据roleId 获取关联的菜单
func (r *RoleDao) GetMenuListByRoleId(ctx context.Context, roleId int32) ([]*model.Menu, error) {
	m := r.repo.Query().RoleMenu
	menu := r.repo.Query().Menu
	menuList := make([]*model.Menu, 0, 10)
	err := m.WithContext(ctx).
		Join(menu, m.MenuID.EqCol(menu.ID)).
		Where(m.RoleID.Eq(roleId)).
		Select(menu.ALL).
		Scan(&menuList)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return menuList, nil
}

// ListRoleMenuPerms 查询每个角色绑定的接口perms权限名
func (r *RoleDao) ListRoleMenuPerms(ctx context.Context) ([]*RoleMenuPerm, error) {
	var rows []*RoleMenuPerm
	m := r.repo.Query().RoleMenu
	menu := r.repo.Query().Menu
	err := m.WithContext(ctx).
		Join(menu, m.MenuID.EqCol(menu.ID)).
		Where(menu.Perms.IsNotNull()).
		Select(m.RoleID, menu.Perms).
		Scan(&rows)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return rows, nil
}
