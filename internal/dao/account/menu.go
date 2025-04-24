package account

import (
	"context"
	"github.com/pkg/errors"
	"snowgo/internal/dal/model"
	"snowgo/internal/dal/repo"
)

type MenuDao struct {
	repo *repo.Repository
}

func NewMenuDao(repo *repo.Repository) *MenuDao {
	return &MenuDao{repo: repo}
}

// CreateMenu 创建菜单或按钮
func (d *MenuDao) CreateMenu(ctx context.Context, menu *model.Menu) (*model.Menu, error) {
	err := d.repo.Query().WithContext(ctx).Menu.Create(menu)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return menu, nil
}

// UpdateMenu 更新菜单
func (d *MenuDao) UpdateMenu(ctx context.Context, menu *model.Menu) (*model.Menu, error) {
	if menu.ID <= 0 {
		return nil, errors.New("菜单ID无效")
	}
	m := d.repo.Query().Menu
	err := m.WithContext(ctx).
		Where(m.ID.Eq(menu.ID)).
		Save(menu)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return menu, nil
}

// DeleteById 删除菜单
func (d *MenuDao) DeleteById(ctx context.Context, id int32) error {
	if id <= 0 {
		return errors.New("菜单ID无效")
	}
	m := d.repo.Query().Menu
	_, err := m.WithContext(ctx).
		Where(m.ID.Eq(id)).
		Delete()
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// GetById 查询单个菜单
func (d *MenuDao) GetById(ctx context.Context, id int32) (*model.Menu, error) {
	if id <= 0 {
		return nil, errors.New("菜单ID无效")
	}
	m := d.repo.Query().Menu
	menu, err := m.WithContext(ctx).
		Where(m.ID.Eq(id)).
		First()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return menu, nil
}

// GetByParentId 根据parentId获取菜单权限
func (d *MenuDao) GetByParentId(ctx context.Context, parentId int32) ([]*model.Menu, error) {
	m := d.repo.Query().Menu
	menus, err := m.WithContext(ctx).Where(m.ParentID.Eq(parentId)).Find()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return menus, nil
}

// GetAllMenus 获取所有菜单（不分页）
func (d *MenuDao) GetAllMenus(ctx context.Context) ([]*model.Menu, error) {
	menus, err := d.repo.Query().WithContext(ctx).Menu.Find()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return menus, nil
}

// IsUsedMenuByIds 判断菜单是否被使用过
func (d *MenuDao) IsUsedMenuByIds(ctx context.Context, menuIds []int32) (bool, error) {
	if len(menuIds) == 0 {
		return false, nil
	}
	m := d.repo.Query().RoleMenu
	_, err := m.WithContext(ctx).Select(m.ID).Where(m.MenuID.In(menuIds...)).First()
	if err != nil {
		return true, errors.WithStack(err)
	}
	return true, nil
}

// GetRoleIdsByIds 根据menuId拿到所有的role
func (d *MenuDao) GetRoleIdsByIds(ctx context.Context, menuId int32) ([]int32, error) {
	var roleIds []int32
	if menuId < 1 {
		return roleIds, errors.New("menuId不存在")
	}
	m := d.repo.Query().RoleMenu
	err := m.WithContext(ctx).Where(m.MenuID.Eq(menuId)).Select(m.RoleID).Pluck(m.RoleID, &roleIds)
	if err != nil {
		return roleIds, errors.WithStack(err)
	}
	return roleIds, nil
}
