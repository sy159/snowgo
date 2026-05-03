package account

import (
	"context"
	"errors"
	"snowgo/internal/constant"
	"snowgo/internal/dal/model"
	"snowgo/internal/dal/query"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- Mock implementations for MenuService ----

type mockMenuDao struct {
	MenuRepo
	menu        *model.Menu
	menus       []*model.Menu
	parentMenus []*model.Menu
	roleIds     []int32
	isUsed      bool
	err         error
}

func (m *mockMenuDao) GetById(_ context.Context, _ int32) (*model.Menu, error) {
	if m.menu == nil {
		return nil, errors.New("not found")
	}
	return m.menu, m.err
}

func (m *mockMenuDao) GetByParentId(_ context.Context, _ int32) ([]*model.Menu, error) {
	return m.parentMenus, m.err
}

func (m *mockMenuDao) GetAllMenus(_ context.Context) ([]*model.Menu, error) {
	return m.menus, m.err
}

func (m *mockMenuDao) IsUsedMenuByIds(_ context.Context, _ []int32) (bool, error) {
	return m.isUsed, m.err
}

func (m *mockMenuDao) GetRoleIdsByIds(_ context.Context, _ int32) ([]int32, error) {
	return m.roleIds, m.err
}

func (m *mockMenuDao) TransactionCreateMenu(_ context.Context, _ *query.Query, menu *model.Menu) (*model.Menu, error) {
	menu.ID = 1
	return menu, nil
}

func (m *mockMenuDao) TransactionUpdateMenu(_ context.Context, _ *query.Query, menu *model.Menu) (*model.Menu, error) {
	return menu, nil
}

func (m *mockMenuDao) TransactionDeleteById(_ context.Context, _ *query.Query, _ int32) error {
	return nil
}

// ---- Tests: GetMenuTree ----

func TestGetMenuTree_CacheHit(t *testing.T) {
	cacheData := make(map[string]string)
	cacheData[constant.CacheMenuTree] = `[{"id":1,"parent_id":0,"menu_type":"Dir","name":"System","path":"/system","icon":"","perms":"","sort_order":1,"children":[]}]`

	svc := &MenuService{
		cache: &mockCache{data: cacheData},
	}

	got, err := svc.GetMenuTree(context.Background())
	require.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, "System", got[0].Name)
}

func TestGetMenuTree_BuildsTree(t *testing.T) {
	cacheData := make(map[string]string)
	now := time.Now()
	menus := []*model.Menu{
		{
			ID:        1,
			ParentID:  0,
			MenuType:  constant.MenuTypeDir,
			Name:      "System",
			Path:      ptrStr("/system"),
			Icon:      ptrStr("system"),
			Perms:     ptrStr(""),
			SortOrder: 1,
			CreatedAt: &now,
			UpdatedAt: &now,
		},
		{
			ID:        2,
			ParentID:  1,
			MenuType:  constant.MenuTypeMenu,
			Name:      "User Management",
			Path:      ptrStr("/system/user"),
			Icon:      ptrStr("user"),
			Perms:     ptrStr("user:list"),
			SortOrder: 1,
			CreatedAt: &now,
			UpdatedAt: &now,
		},
	}

	svc := &MenuService{
		menuDao: &mockMenuDao{menus: menus},
		cache:   &mockCache{data: cacheData},
	}

	got, err := svc.GetMenuTree(context.Background())
	require.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, "System", got[0].Name)
	assert.Len(t, got[0].Children, 1)
	assert.Equal(t, "User Management", got[0].Children[0].Name)
}

func TestGetMenuTree_SortsByOrder(t *testing.T) {
	cacheData := make(map[string]string)
	now := time.Now()
	menus := []*model.Menu{
		{
			ID: 2, ParentID: 0, MenuType: constant.MenuTypeMenu,
			Name: "Second", Path: ptrStr("/second"), Icon: ptrStr(""),
			Perms: ptrStr(""), SortOrder: 2, CreatedAt: &now, UpdatedAt: &now,
		},
		{
			ID: 1, ParentID: 0, MenuType: constant.MenuTypeMenu,
			Name: "First", Path: ptrStr("/first"), Icon: ptrStr(""),
			Perms: ptrStr(""), SortOrder: 1, CreatedAt: &now, UpdatedAt: &now,
		},
	}

	svc := &MenuService{
		menuDao: &mockMenuDao{menus: menus},
		cache:   &mockCache{data: cacheData},
	}

	got, err := svc.GetMenuTree(context.Background())
	require.NoError(t, err)
	assert.Len(t, got, 2)
	assert.Equal(t, "First", got[0].Name)
	assert.Equal(t, "Second", got[1].Name)
}

func TestGetMenuTree_OrphanNodesGoToRoot(t *testing.T) {
	cacheData := make(map[string]string)
	now := time.Now()
	menus := []*model.Menu{
		{
			ID: 1, ParentID: 999, MenuType: constant.MenuTypeMenu,
			Name: "Orphan", Path: ptrStr("/orphan"), Icon: ptrStr(""),
			Perms: ptrStr(""), SortOrder: 1, CreatedAt: &now, UpdatedAt: &now,
		},
	}

	svc := &MenuService{
		menuDao: &mockMenuDao{menus: menus},
		cache:   &mockCache{data: cacheData},
	}

	got, err := svc.GetMenuTree(context.Background())
	require.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, "Orphan", got[0].Name)
}

func TestGetMenuTree_CachesResult(t *testing.T) {
	cacheData := make(map[string]string)
	now := time.Now()
	menus := []*model.Menu{
		{
			ID: 1, ParentID: 0, MenuType: constant.MenuTypeDir,
			Name: "Test", Path: ptrStr("/test"), Icon: ptrStr(""),
			Perms: ptrStr(""), SortOrder: 1, CreatedAt: &now, UpdatedAt: &now,
		},
	}

	svc := &MenuService{
		menuDao: &mockMenuDao{menus: menus},
		cache:   &mockCache{data: cacheData},
	}

	_, err := svc.GetMenuTree(context.Background())
	require.NoError(t, err)

	cached, ok := cacheData[constant.CacheMenuTree]
	assert.True(t, ok)
	assert.NotEmpty(t, cached)
}

func TestGetMenuTree_EmptyMenus(t *testing.T) {
	cacheData := make(map[string]string)

	svc := &MenuService{
		menuDao: &mockMenuDao{menus: []*model.Menu{}},
		cache:   &mockCache{data: cacheData},
	}

	got, err := svc.GetMenuTree(context.Background())
	require.NoError(t, err)
	assert.Empty(t, got)
}

// ---- Tests: CreateMenu ----

func TestCreateMenu_InvalidParent(t *testing.T) {
	logWriter := &mockLogWriter{}
	svc := &MenuService{
		menuDao:    &mockMenuDao{menu: nil},
		logService: logWriter,
	}

	_, err := svc.CreateMenu(testCtx(), &MenuParam{
		ParentID: 999, MenuType: constant.MenuTypeMenu, Name: "Child",
		Path: "/child", Icon: "", Perms: "", SortOrder: 1,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "父级菜单不存在")
	assert.Equal(t, 0, logWriter.callCount)
}

// ---- Tests: UpdateMenu ----

func TestUpdateMenu_InvalidID(t *testing.T) {
	svc := &MenuService{}
	err := svc.UpdateMenu(testCtx(), &MenuParam{
		ID: -1, MenuType: constant.MenuTypeMenu, Name: "X",
		Path: "/x", Icon: "", Perms: "", SortOrder: 1,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "菜单ID无效")
}

func TestUpdateMenu_ParentIsSelf(t *testing.T) {
	now := time.Now()
	svc := &MenuService{
		menuDao: &mockMenuDao{menu: &model.Menu{
			ID: 1, MenuType: constant.MenuTypeMenu, Name: "X",
			CreatedAt: &now, UpdatedAt: &now,
		}},
	}
	err := svc.UpdateMenu(testCtx(), &MenuParam{
		ID: 1, ParentID: 1, MenuType: constant.MenuTypeMenu, Name: "X",
		Path: "/x", Icon: "", Perms: "", SortOrder: 1,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "父级菜单不能是自己")
}

func TestUpdateMenu_NotFound(t *testing.T) {
	svc := &MenuService{
		menuDao: &mockMenuDao{menu: nil},
	}
	err := svc.UpdateMenu(testCtx(), &MenuParam{
		ID: 1, MenuType: constant.MenuTypeMenu, Name: "X",
		Path: "/x", Icon: "", Perms: "", SortOrder: 1,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "菜单不存在")
}

// ---- Tests: DeleteMenuById ----

func TestDeleteMenuById_InvalidID(t *testing.T) {
	svc := &MenuService{}
	err := svc.DeleteMenuById(testCtx(), -1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "菜单ID无效")
}

func TestDeleteMenuById_HasChildren(t *testing.T) {
	now := time.Now()
	svc := &MenuService{
		menuDao: &mockMenuDao{
			menu:        &model.Menu{ID: 1, CreatedAt: &now, UpdatedAt: &now},
			parentMenus: []*model.Menu{{ID: 2, ParentID: 1}},
		},
	}
	err := svc.DeleteMenuById(testCtx(), 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "存在子菜单")
}

func TestDeleteMenuById_UsedByRole(t *testing.T) {
	now := time.Now()
	svc := &MenuService{
		menuDao: &mockMenuDao{
			menu:        &model.Menu{ID: 1, CreatedAt: &now, UpdatedAt: &now},
			parentMenus: []*model.Menu{},
			isUsed:      true,
			roleIds:     []int32{1},
		},
	}
	err := svc.DeleteMenuById(testCtx(), 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "该菜单权限已被使用")
}
