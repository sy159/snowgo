package account

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"snowgo/internal/constant"
	"snowgo/internal/dal/model"
	daoAccount "snowgo/internal/dao/account"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// ---- Mock implementations for RoleService ----

type mockRoleDao struct {
	RoleRepo
	codeExists bool
	role       *model.Role
	menuIds    []int32
	menus      []*model.Menu
	countMenus int64
	isUsed     bool
	err        error
}

func (m *mockRoleDao) IsCodeExists(_ context.Context, _ string, _ int32) (bool, error) {
	return m.codeExists, m.err
}

func (m *mockRoleDao) GetRoleById(_ context.Context, _ int32) (*model.Role, error) {
	if m.role == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return m.role, m.err
}

func (m *mockRoleDao) GetRoleList(_ context.Context, _ *daoAccount.RoleListCondition) ([]*model.Role, int64, error) {
	if m.role == nil {
		return nil, 0, m.err
	}
	return []*model.Role{m.role}, 1, m.err
}

func (m *mockRoleDao) GetMenuIdsByRoleId(_ context.Context, _ int32) ([]int32, error) {
	return m.menuIds, m.err
}

func (m *mockRoleDao) GetMenuListByRoleId(_ context.Context, _ int32) ([]*model.Menu, error) {
	return m.menus, m.err
}

func (m *mockRoleDao) CountMenuByIds(_ context.Context, _ []int32) (int64, error) {
	return m.countMenus, m.err
}

func (m *mockRoleDao) IsUsedUserByIds(_ context.Context, _ int32) (bool, error) {
	return m.isUsed, m.err
}

// ---- Tests: GetRoleById ----

func TestGetRoleById_Success(t *testing.T) {
	desc := "Test role"
	name := "Tester"
	role := &model.Role{
		ID:          1,
		Code:        "tester",
		Name:        &name,
		Description: &desc,
		CreatedAt:   ptrTime(time.Now()),
		UpdatedAt:   ptrTime(time.Now()),
	}
	menuIds := []int32{1, 2, 3}

	svc := &RoleService{
		roleDao: &mockRoleDao{role: role, menuIds: menuIds},
	}

	got, err := svc.GetRoleById(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, int32(1), got.ID)
	assert.Equal(t, "tester", got.Code)
	assert.Equal(t, menuIds, got.MenuIds)
}

func TestGetRoleById_InvalidID(t *testing.T) {
	svc := &RoleService{}
	_, err := svc.GetRoleById(context.Background(), -1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "角色ID无效")
}

func TestGetRoleById_NotFound(t *testing.T) {
	svc := &RoleService{roleDao: &mockRoleDao{role: nil}}

	_, err := svc.GetRoleById(context.Background(), 999)
	assert.True(t, errors.Is(err, ErrRoleNotFound))
}

// ---- Tests: ListRoles ----

func TestListRoles_Success(t *testing.T) {
	name := "Admin"
	desc := "Admin role"
	role := &model.Role{
		ID:          1,
		Code:        "admin",
		Name:        &name,
		Description: &desc,
		CreatedAt:   ptrTime(time.Now()),
		UpdatedAt:   ptrTime(time.Now()),
	}

	svc := &RoleService{roleDao: &mockRoleDao{role: role}}

	result, err := svc.ListRoles(context.Background(), &RoleListCondition{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Total)
	assert.Len(t, result.List, 1)
	assert.Equal(t, "admin", result.List[0].Code)
}

func TestListRoles_Empty(t *testing.T) {
	svc := &RoleService{roleDao: &mockRoleDao{role: nil}}

	result, err := svc.ListRoles(context.Background(), &RoleListCondition{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(0), result.Total)
	assert.Empty(t, result.List)
}

// ---- Tests: GetRoleMenuListByRuleID ----

func TestGetRoleMenuListByRuleID_CacheHit(t *testing.T) {
	cacheData := make(map[string]string)
	cacheKey := fmt.Sprintf("%s%d", constant.CacheRoleMenuPrefix, int32(1))
	b, _ := json.Marshal([]*MenuData{{ID: 1, Name: "Dashboard"}})
	cacheData[cacheKey] = string(b)

	svc := &RoleService{
		cache: &mockCache{data: cacheData},
	}

	got, err := svc.GetRoleMenuListByRuleID(context.Background(), 1)
	require.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, "Dashboard", got[0].Name)
}

func TestGetRoleMenuListByRuleID_CacheMiss(t *testing.T) {
	cacheData := make(map[string]string)
	menus := []*model.Menu{
		{
			ID:        1,
			ParentID:  0,
			MenuType:  constant.MenuTypeMenu,
			Name:      "Dashboard",
			Path:      ptrStr("/dashboard"),
			Icon:      ptrStr("dashboard"),
			Perms:     ptrStr("dashboard:view"),
			SortOrder: 1,
			CreatedAt: ptrTime(time.Now()),
			UpdatedAt: ptrTime(time.Now()),
		},
	}

	svc := &RoleService{
		roleDao: &mockRoleDao{menus: menus},
		cache:   &mockCache{data: cacheData},
	}

	got, err := svc.GetRoleMenuListByRuleID(context.Background(), 1)
	require.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, "Dashboard", got[0].Name)
	// Verify cache was written
	_, ok := cacheData[fmt.Sprintf("%s%d", constant.CacheRoleMenuPrefix, int32(1))]
	assert.True(t, ok)
}

// ---- Tests: GetRolePermsListByRuleID ----

func TestGetRolePermsListByRuleID_InvalidID(t *testing.T) {
	svc := &RoleService{}
	_, err := svc.GetRolePermsListByRuleID(context.Background(), -1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "角色ID无效")
}

func TestGetRolePermsListByRuleID_FiltersButtonsOnly(t *testing.T) {
	menus := []*model.Menu{
		{
			ID:        1,
			MenuType:  constant.MenuTypeDir,
			Name:      "System",
			Path:      ptrStr("/system"),
			Icon:      ptrStr(""),
			Perms:     ptrStr(""),
			SortOrder: 1,
			CreatedAt: ptrTime(time.Now()),
			UpdatedAt: ptrTime(time.Now()),
		},
		{
			ID:        2,
			MenuType:  constant.MenuTypeBtn,
			Name:      "Create User",
			Path:      ptrStr(""),
			Icon:      ptrStr(""),
			Perms:     ptrStr("user:create"),
			SortOrder: 1,
			CreatedAt: ptrTime(time.Now()),
			UpdatedAt: ptrTime(time.Now()),
		},
		{
			ID:        3,
			MenuType:  constant.MenuTypeBtn,
			Name:      "Delete User",
			Path:      ptrStr(""),
			Icon:      ptrStr(""),
			Perms:     ptrStr("user:delete"),
			SortOrder: 2,
			CreatedAt: ptrTime(time.Now()),
			UpdatedAt: ptrTime(time.Now()),
		},
	}

	svc := &RoleService{
		roleDao: &mockRoleDao{menus: menus},
		cache:   &mockCache{data: make(map[string]string)},
	}

	perms, err := svc.GetRolePermsListByRuleID(context.Background(), 1)
	require.NoError(t, err)
	assert.Len(t, perms, 2)
	assert.Contains(t, perms, "user:create")
	assert.Contains(t, perms, "user:delete")
}

func ptrTime(t time.Time) *time.Time { return &t }
