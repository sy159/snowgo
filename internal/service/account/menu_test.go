package account

import (
	"context"
	"errors"
	"snowgo/internal/constant"
	"snowgo/internal/dal/model"
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
	assert.Len(t, got, 1) // One root node
	assert.Equal(t, "System", got[0].Name)
	assert.Len(t, got[0].Children, 1) // One child
	assert.Equal(t, "User Management", got[0].Children[0].Name)
}

func TestGetMenuTree_SortsByOrder(t *testing.T) {
	cacheData := make(map[string]string)
	now := time.Now()
	menus := []*model.Menu{
		{
			ID:        2,
			ParentID:  0,
			MenuType:  constant.MenuTypeMenu,
			Name:      "Second",
			Path:      ptrStr("/second"),
			Icon:      ptrStr(""),
			Perms:     ptrStr(""),
			SortOrder: 2,
			CreatedAt: &now,
			UpdatedAt: &now,
		},
		{
			ID:        1,
			ParentID:  0,
			MenuType:  constant.MenuTypeMenu,
			Name:      "First",
			Path:      ptrStr("/first"),
			Icon:      ptrStr(""),
			Perms:     ptrStr(""),
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
	assert.Len(t, got, 2)
	// Should be sorted by sort_order
	assert.Equal(t, "First", got[0].Name)
	assert.Equal(t, "Second", got[1].Name)
}

func TestGetMenuTree_OrphanNodesGoToRoot(t *testing.T) {
	cacheData := make(map[string]string)
	now := time.Now()
	menus := []*model.Menu{
		{
			ID:        1,
			ParentID:  999, // parent doesn't exist
			MenuType:  constant.MenuTypeMenu,
			Name:      "Orphan",
			Path:      ptrStr("/orphan"),
			Icon:      ptrStr(""),
			Perms:     ptrStr(""),
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
	assert.Len(t, got, 1) // Orphan becomes root
	assert.Equal(t, "Orphan", got[0].Name)
}

func TestGetMenuTree_CachesResult(t *testing.T) {
	cacheData := make(map[string]string)
	now := time.Now()
	menus := []*model.Menu{
		{
			ID:        1,
			ParentID:  0,
			MenuType:  constant.MenuTypeDir,
			Name:      "Test",
			Path:      ptrStr("/test"),
			Icon:      ptrStr(""),
			Perms:     ptrStr(""),
			SortOrder: 1,
			CreatedAt: &now,
			UpdatedAt: &now,
		},
	}

	svc := &MenuService{
		menuDao: &mockMenuDao{menus: menus},
		cache:   &mockCache{data: cacheData},
	}

	_, err := svc.GetMenuTree(context.Background())
	require.NoError(t, err)

	// Verify cache was set
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
