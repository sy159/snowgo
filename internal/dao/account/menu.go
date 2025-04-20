package account

import (
	"context"
	"github.com/pkg/errors"
	"gorm.io/gorm"
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
		return nil, errors.WithMessage(err, "菜单创建失败")
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
		return nil, errors.WithMessage(err, "菜单更新失败")
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
		return errors.WithMessage(err, "菜单删除失败")
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("菜单不存在")
		}
		return nil, errors.WithMessage(err, "菜单查询失败")
	}
	return menu, nil
}

// GetByParentId 根据parentId获取菜单权限
func (d *MenuDao) GetByParentId(ctx context.Context, parentId int32) ([]*model.Menu, error) {
	m := d.repo.Query().Menu
	menus, err := m.WithContext(ctx).Where(m.ParentID.Eq(parentId)).Find()
	if err != nil {
		return nil, err
	}
	return menus, nil
}

// GetAllMenus 获取所有菜单（不分页）
func (d *MenuDao) GetAllMenus(ctx context.Context) ([]*model.Menu, error) {
	menus, err := d.repo.Query().WithContext(ctx).Menu.Find()
	if err != nil {
		return nil, err
	}
	return menus, nil
}

// IsUsedMenuByIds 判断菜单是否被使用过
func (d *MenuDao) IsUsedMenuByIds(ctx context.Context, MenuIds []int32) (bool, error) {
	if len(MenuIds) == 0 {
		return false, nil
	}
	m := d.repo.Query().RoleMenu
	_, err := m.WithContext(ctx).Select(m.ID).Where(m.MenuID.In(MenuIds...)).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return true, errors.WithStack(err)
	}
	return true, nil
}
