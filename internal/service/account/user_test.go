package account

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"snowgo/internal/constant"
	"snowgo/internal/dal/model"
	"snowgo/internal/dal/query"
	daoAccount "snowgo/internal/dao/account"
	"snowgo/pkg/xcache"
	"snowgo/pkg/xcryption"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// ---- Mock implementations ----

type mockUserDao struct {
	UserRepo
	isDuplicate bool
	user        *model.User
	roles       []*model.Role
	roleIds     []int32
	countRoles  int64
	err         error
}

func (m *mockUserDao) IsNameTelDuplicate(_ context.Context, _, _ string, _ int32) (bool, error) {
	return m.isDuplicate, m.err
}

func (m *mockUserDao) CountRoleByIds(_ context.Context, _ []int32) (int64, error) {
	return m.countRoles, m.err
}

func (m *mockUserDao) GetUserById(_ context.Context, _ int32) (*model.User, error) {
	if m.user == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return m.user, m.err
}

func (m *mockUserDao) GetUserByUsername(_ context.Context, _ string) (*model.User, error) {
	if m.user == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return m.user, m.err
}

func (m *mockUserDao) GetRoleListByUserId(_ context.Context, _ int32) ([]*model.Role, error) {
	return m.roles, m.err
}

func (m *mockUserDao) GetRoleIdsByUserId(_ context.Context, _ int32) ([]int32, error) {
	return m.roleIds, m.err
}

func (m *mockUserDao) GetUserList(_ context.Context, _ *daoAccount.UserListCondition) ([]*model.User, int64, error) {
	if m.user == nil {
		return nil, 0, m.err
	}
	return []*model.User{m.user}, 1, m.err
}

func (m *mockUserDao) TransactionCreateUser(_ context.Context, _ *query.Query, user *model.User) (*model.User, error) {
	user.ID = 1
	return user, m.err
}

func (m *mockUserDao) TransactionUpdateUser(_ context.Context, _ *query.Query, _ int32, _, _, _ string) error {
	return m.err
}

func (m *mockUserDao) TransactionCreateUserRoleInBatches(_ context.Context, _ *query.Query, _ []*model.UserRole) error {
	return m.err
}

func (m *mockUserDao) TransactionDeleteUserRole(_ context.Context, _ *query.Query, _ int32) error {
	return m.err
}

func (m *mockUserDao) TransactionDeleteById(_ context.Context, _ *query.Query, _ int32) error {
	return m.err
}

func (m *mockUserDao) ResetPwdById(_ context.Context, _ int32, _ string) error {
	return m.err
}

type mockCache struct {
	xcache.Cache
	data map[string]string
	err  error
}

func (m *mockCache) Get(_ context.Context, key string) (string, bool, error) {
	if m.err != nil {
		return "", false, m.err
	}
	val, ok := m.data[key]
	return val, ok, nil
}

func (m *mockCache) Set(_ context.Context, key, value string, _ time.Duration) error {
	if m.err != nil {
		return m.err
	}
	m.data[key] = value
	return nil
}

func (m *mockCache) Delete(_ context.Context, keys ...string) (int64, error) {
	if m.err != nil {
		return 0, m.err
	}
	for _, k := range keys {
		delete(m.data, k)
	}
	return int64(len(keys)), nil
}

// ---- Helpers ----

func testUser() *model.User {
	nickname := "Test User"
	status := constant.UserStatusActive
	now := time.Now()
	return &model.User{
		ID:        1,
		Username:  "testuser",
		Password:  "$2a$10$dummyhash",
		Tel:       "13800138000",
		Nickname:  &nickname,
		Status:    &status,
		CreatedAt: &now,
		UpdatedAt: &now,
	}
}

// ---- Tests: Authenticate ----

func TestAuthenticate_Success(t *testing.T) {
	pw, _ := xcryption.HashPassword("password123")
	u := testUser()
	u.Password = pw

	svc := &UserService{userDao: &mockUserDao{user: u}}

	got, err := svc.Authenticate(context.Background(), "testuser", "password123")
	require.NoError(t, err)
	assert.Equal(t, int32(1), got.ID)
	assert.Equal(t, "testuser", got.Username)
}

func TestAuthenticate_WrongPassword(t *testing.T) {
	pw, _ := xcryption.HashPassword("correct")
	u := testUser()
	u.Password = pw

	svc := &UserService{userDao: &mockUserDao{user: u}}

	_, err := svc.Authenticate(context.Background(), "testuser", "wrong")
	assert.True(t, errors.Is(err, ErrAuth))
}

func TestAuthenticate_UserNotFound(t *testing.T) {
	svc := &UserService{userDao: &mockUserDao{user: nil}}

	_, err := svc.Authenticate(context.Background(), "nonexistent", "any")
	assert.Error(t, err)
}

// ---- Tests: GetUserById ----

func TestGetUserById_Success(t *testing.T) {
	u := testUser()
	roles := []*model.Role{{ID: 1, Code: "admin", Name: ptrStr("Admin")}}

	svc := &UserService{userDao: &mockUserDao{user: u, roles: roles}}

	got, err := svc.GetUserById(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, int32(1), got.ID)
	assert.Equal(t, "Test User", got.Nickname)
	assert.Len(t, got.RoleList, 1)
	assert.Equal(t, "admin", got.RoleList[0].Code)
}

func TestGetUserById_NotFound(t *testing.T) {
	svc := &UserService{userDao: &mockUserDao{user: nil}}

	_, err := svc.GetUserById(context.Background(), 999)
	assert.True(t, errors.Is(err, ErrUserNotFound))
}

func TestGetUserById_InvalidID(t *testing.T) {
	svc := &UserService{}
	_, err := svc.GetUserById(context.Background(), -1)
	assert.True(t, errors.Is(err, ErrUserNotFound))
}

// ---- Tests: GetUserList ----

func TestGetUserList_Success(t *testing.T) {
	u := testUser()
	svc := &UserService{userDao: &mockUserDao{user: u}}

	result, err := svc.GetUserList(context.Background(), &UserListCondition{Limit: 10, Offset: 0})
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Total)
	assert.Len(t, result.List, 1)
}

// ---- Tests: GetRoleIdsByUserId ----

func TestGetRoleIdsByUserId_CacheHit(t *testing.T) {
	roleIds := []int32{1, 2}
	cacheData := make(map[string]string)
	cacheKey := fmt.Sprintf("%s%d", constant.CacheUserRolePrefix, int32(1))
	b, _ := json.Marshal(roleIds)
	cacheData[cacheKey] = string(b)

	svc := &UserService{userDao: &mockUserDao{}, cache: &mockCache{data: cacheData}}

	got, err := svc.GetRoleIdsByUserId(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, []int32{1, 2}, got)
}

func TestGetRoleIdsByUserId_CacheMiss(t *testing.T) {
	roleIds := []int32{3, 4}
	cacheData := make(map[string]string)

	svc := &UserService{
		userDao: &mockUserDao{roleIds: roleIds},
		cache:   &mockCache{data: cacheData},
	}

	got, err := svc.GetRoleIdsByUserId(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, []int32{3, 4}, got)
	// Verify cache was written
	_, ok := cacheData[fmt.Sprintf("%s%d", constant.CacheUserRolePrefix, int32(1))]
	assert.True(t, ok)
}

func TestGetRoleIdsByUserId_InvalidID(t *testing.T) {
	svc := &UserService{}
	_, err := svc.GetRoleIdsByUserId(context.Background(), -1)
	assert.True(t, errors.Is(err, ErrUserNotFound))
}

// ---- Tests: GetPermsListById ----

func TestGetPermsListById_NoRolesReturnsEmpty(t *testing.T) {
	svc := &UserService{
		userDao: &mockUserDao{roleIds: []int32{}},
		cache:   &mockCache{data: make(map[string]string)},
	}

	perms, err := svc.GetPermsListById(context.Background(), 1)
	require.NoError(t, err)
	assert.Empty(t, perms)
}

func TestGetRoleIdsByUserId_InvalidUserID(t *testing.T) {
	svc := &UserService{}
	_, err := svc.GetRoleIdsByUserId(context.Background(), 0)
	assert.True(t, errors.Is(err, ErrUserNotFound))
}

// ---- Helpers ----

func ptrStr(s string) *string { return &s }
