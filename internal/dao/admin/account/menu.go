package account

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"snowgo/internal/dal/model"
	"snowgo/internal/dal/query"
	"snowgo/internal/dal/repo"
)

type MenuDao struct {
	repo *repo.Repository
}

func NewMenuDao(repo *repo.Repository) *MenuDao {
	return &MenuDao{repo: repo}
}

// CreateMenu 创建菜单或按钮
func (d *MenuDao) CreateMenu(ctx context.Context, q *query.Query, menu *model.SysMenu) (*model.SysMenu, error) {
	err := q.WithContext(ctx).SysMenu.Create(menu)
	if err != nil {
		return nil, err
	}
	return menu, nil
}

// UpdateMenu 更新菜单
func (d *MenuDao) UpdateMenu(ctx context.Context, q *query.Query, menu *model.SysMenu) (*model.SysMenu, error) {
	if menu.ID <= 0 {
		return nil, errors.New("菜单ID无效")
	}
	err := q.WithContext(ctx).SysMenu.Where(q.SysMenu.ID.Eq(menu.ID)).Save(menu)
	if err != nil {
		return nil, err
	}
	return menu, nil
}

// DeleteById 删除菜单
func (d *MenuDao) DeleteById(ctx context.Context, q *query.Query, id int32) error {
	if id <= 0 {
		return errors.New("菜单ID无效")
	}
	_, err := q.WithContext(ctx).SysMenu.Where(q.SysMenu.ID.Eq(id)).Delete()
	if err != nil {
		return err
	}
	return nil
}

// GetById 查询单个菜单
func (d *MenuDao) GetById(ctx context.Context, q *query.Query, id int32) (*model.SysMenu, error) {
	if id <= 0 {
		return nil, errors.New("菜单ID无效")
	}
	menu, err := q.WithContext(ctx).SysMenu.Where(q.SysMenu.ID.Eq(id)).First()
	if err != nil {
		return nil, err
	}
	return menu, nil
}

// GetByParentId 根据parentId获取菜单权限
func (d *MenuDao) GetByParentId(ctx context.Context, q *query.Query, parentId int32) ([]*model.SysMenu, error) {
	menus, err := q.WithContext(ctx).SysMenu.Where(q.SysMenu.ParentID.Eq(parentId)).Find()
	if err != nil {
		return nil, err
	}
	return menus, nil
}

// GetAllMenus 获取所有菜单（不分页）
func (d *MenuDao) GetAllMenus(ctx context.Context) ([]*model.SysMenu, error) {
	menus, err := d.repo.Query().WithContext(ctx).SysMenu.Find()
	if err != nil {
		return nil, err
	}
	return menus, nil
}

// IsUsedMenuByIds 判断菜单是否被使用过
func (d *MenuDao) IsUsedMenuByIds(ctx context.Context, q *query.Query, menuIds []int32) (bool, error) {
	if len(menuIds) == 0 {
		return false, nil
	}
	m := q.SysRoleMenu
	_, err := m.WithContext(ctx).Select(m.ID).Where(m.MenuID.In(menuIds...)).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return true, err
	}
	return true, nil
}

// IsPermsExists 检查 perms 是否已存在，excludeId 用于排除自身（更新场景）
func (d *MenuDao) IsPermsExists(ctx context.Context, q *query.Query, perms string, excludeId int32) (bool, error) {
	if len(perms) == 0 {
		return false, nil
	}
	stmt := q.WithContext(ctx).SysMenu.Select(q.SysMenu.ID).Where(q.SysMenu.Perms.Eq(perms))
	if excludeId > 0 {
		stmt = stmt.Where(q.SysMenu.ID.Neq(excludeId))
	}
	_, err := stmt.First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return true, err
	}
	return true, nil
}

// IsPathExists 检查 path 是否已存在，excludeId 用于排除自身（更新场景）
func (d *MenuDao) IsPathExists(ctx context.Context, q *query.Query, path string, excludeId int32) (bool, error) {
	if len(path) == 0 {
		return false, nil
	}
	stmt := q.WithContext(ctx).SysMenu.Select(q.SysMenu.ID).Where(q.SysMenu.Path.Eq(path))
	if excludeId > 0 {
		stmt = stmt.Where(q.SysMenu.ID.Neq(excludeId))
	}
	_, err := stmt.First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return true, err
	}
	return true, nil
}

// GetRoleIdsByIds 根据menuId拿到所有的role
func (d *MenuDao) GetRoleIdsByIds(ctx context.Context, menuId int32) ([]int32, error) {
	var roleIds []int32
	if menuId < 1 {
		return roleIds, errors.New("menuId不存在")
	}
	m := d.repo.Query().SysRoleMenu
	err := m.WithContext(ctx).Where(m.MenuID.Eq(menuId)).Select(m.RoleID).Pluck(m.RoleID, &roleIds)
	if err != nil {
		return roleIds, err
	}
	return roleIds, nil
}
