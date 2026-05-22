package account

import (
	"context"
	"errors"
	"time"

	"snowgo/internal/dal/model"
	"snowgo/internal/dal/query"
	daoAccount "snowgo/internal/dao/admin/account"
	"snowgo/pkg/xauth"
)

func testUserCtx() context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, xauth.XUserId, int32(1))
	ctx = context.WithValue(ctx, xauth.XUserName, "admin")
	ctx = context.WithValue(ctx, xauth.XTraceId, "trace-test")
	ctx = context.WithValue(ctx, xauth.XIp, "127.0.0.1")
	return ctx
}

type fakeUserRepo struct {
	userByUsername    *model.SysUser
	userByUsernameErr error
	roleIds           []int32
	roleIdsErr        error
	getRoleIDsCalls   int
}

func (f *fakeUserRepo) CreateUser(context.Context, *query.Query, *model.SysUser) (*model.SysUser, error) {
	panic("not implemented")
}

func (f *fakeUserRepo) UpdateUser(context.Context, *query.Query, *model.SysUser) (*model.SysUser, error) {
	panic("not implemented")
}

func (f *fakeUserRepo) CreateUserRole(context.Context, *query.Query, *model.SysUserRole) error {
	panic("not implemented")
}

func (f *fakeUserRepo) CreateUserRoleInBatches(context.Context, *query.Query, []*model.SysUserRole) error {
	panic("not implemented")
}

func (f *fakeUserRepo) DeleteUserRole(context.Context, *query.Query, int32) error {
	panic("not implemented")
}

func (f *fakeUserRepo) DeleteById(context.Context, *query.Query, int32) error {
	panic("not implemented")
}

func (f *fakeUserRepo) GetRoleListByUserId(context.Context, int32) ([]*model.SysRole, error) {
	panic("not implemented")
}

func (f *fakeUserRepo) GetRoleIdsByUserId(context.Context, int32) ([]int32, error) {
	f.getRoleIDsCalls++
	return f.roleIds, f.roleIdsErr
}

func (f *fakeUserRepo) IsNameTelDuplicate(context.Context, string, string, int32) (bool, error) {
	panic("not implemented")
}

func (f *fakeUserRepo) IsExistByRoleId(context.Context, int32) (bool, error) {
	panic("not implemented")
}

func (f *fakeUserRepo) CountRoleByIds(context.Context, *query.Query, []int32) (int64, error) {
	panic("not implemented")
}

func (f *fakeUserRepo) GetUserById(context.Context, int32) (*model.SysUser, error) {
	panic("not implemented")
}

func (f *fakeUserRepo) GetUserByUsername(context.Context, string) (*model.SysUser, error) {
	return f.userByUsername, f.userByUsernameErr
}

func (f *fakeUserRepo) GetUserList(context.Context, *daoAccount.UserListCondition) ([]*model.SysUser, int64, error) {
	panic("not implemented")
}

func (f *fakeUserRepo) ResetPwdById(context.Context, *query.Query, int32, string) error {
	panic("not implemented")
}

type fakeCache struct {
	values      map[string]string
	sets        map[string]string
	deletes     []string
	expirations map[string]time.Duration
}

func newFakeCache() *fakeCache {
	return &fakeCache{
		values:      make(map[string]string),
		sets:        make(map[string]string),
		expirations: make(map[string]time.Duration),
	}
}

func (f *fakeCache) Eval(context.Context, string, []string, ...any) (any, error) {
	panic("not implemented")
}

func (f *fakeCache) Get(_ context.Context, key string) (string, bool, error) {
	value, ok := f.values[key]
	return value, ok, nil
}

func (f *fakeCache) Set(_ context.Context, key string, value string, expiration time.Duration) error {
	f.sets[key] = value
	f.values[key] = value
	f.expirations[key] = expiration
	return nil
}

func (f *fakeCache) Delete(_ context.Context, keys ...string) (int64, error) {
	f.deletes = append(f.deletes, keys...)
	for _, key := range keys {
		delete(f.values, key)
	}
	return int64(len(keys)), nil
}

func (f *fakeCache) IncrBy(context.Context, string, int64) (int64, error) {
	panic("not implemented")
}

func (f *fakeCache) DecrBy(context.Context, string, int64) (int64, error) {
	panic("not implemented")
}

func (f *fakeCache) HSet(context.Context, string, string, string) error {
	panic("not implemented")
}

func (f *fakeCache) HGet(context.Context, string, string) (string, bool, error) {
	panic("not implemented")
}

func (f *fakeCache) HGetAll(context.Context, string) (map[string]string, error) {
	panic("not implemented")
}

func (f *fakeCache) HDel(context.Context, string, ...string) (int64, error) {
	panic("not implemented")
}

func (f *fakeCache) HIncrBy(context.Context, string, string, int64) (int64, error) {
	panic("not implemented")
}

func (f *fakeCache) HLen(context.Context, string) (int64, error) {
	panic("not implemented")
}

func (f *fakeCache) ZAdd(context.Context, string, float64, string) error {
	panic("not implemented")
}

func (f *fakeCache) ZRem(context.Context, string, ...string) error {
	panic("not implemented")
}

func (f *fakeCache) ZRange(context.Context, string, int64, int64) ([]string, error) {
	panic("not implemented")
}

func (f *fakeCache) ZCard(context.Context, string) (int64, error) {
	panic("not implemented")
}

func (f *fakeCache) Exists(context.Context, string) (bool, error) {
	panic("not implemented")
}

func (f *fakeCache) Expire(context.Context, string, time.Duration) error {
	panic("not implemented")
}

func (f *fakeCache) TTL(context.Context, string) (time.Duration, error) {
	panic("not implemented")
}

var errTestDAO = errors.New("dao error")

type fakeRoleRepo struct {
	menuList        []*model.SysMenu
	menuListErr     error
	getMenuListCall int
}

func (f *fakeRoleRepo) IsCodeExists(context.Context, string, int32) (bool, error) {
	panic("not implemented")
}

func (f *fakeRoleRepo) CreateRole(context.Context, *query.Query, *model.SysRole) (*model.SysRole, error) {
	panic("not implemented")
}

func (f *fakeRoleRepo) UpdateRole(context.Context, *query.Query, *model.SysRole) (*model.SysRole, error) {
	panic("not implemented")
}

func (f *fakeRoleRepo) DeleteById(context.Context, *query.Query, int32) error {
	panic("not implemented")
}

func (f *fakeRoleRepo) GetRoleById(context.Context, int32) (*model.SysRole, error) {
	panic("not implemented")
}

func (f *fakeRoleRepo) GetRoleList(context.Context, *daoAccount.RoleListCondition) ([]*model.SysRole, int64, error) {
	panic("not implemented")
}

func (f *fakeRoleRepo) CreateRoleMenu(context.Context, *query.Query, []*model.SysRoleMenu) error {
	panic("not implemented")
}

func (f *fakeRoleRepo) DeleteRoleMenu(context.Context, *query.Query, int32) error {
	panic("not implemented")
}

func (f *fakeRoleRepo) IsUsedUserByIds(context.Context, *query.Query, int32) (bool, error) {
	panic("not implemented")
}

func (f *fakeRoleRepo) CountMenuByIds(context.Context, *query.Query, []int32) (int64, error) {
	panic("not implemented")
}

func (f *fakeRoleRepo) GetMenuIdsByRoleId(context.Context, int32) ([]int32, error) {
	panic("not implemented")
}

func (f *fakeRoleRepo) GetMenuPermsByRoleId(context.Context, int32) ([]string, error) {
	panic("not implemented")
}

func (f *fakeRoleRepo) GetMenuPermsByRoleIds(context.Context, []int32) ([]string, error) {
	panic("not implemented")
}

func (f *fakeRoleRepo) GetMenuListByRoleId(context.Context, int32) ([]*model.SysMenu, error) {
	f.getMenuListCall++
	return f.menuList, f.menuListErr
}

func (f *fakeRoleRepo) ListRoleMenuPerms(context.Context) ([]*daoAccount.RoleMenuPerm, error) {
	panic("not implemented")
}

func (f *fakeRoleRepo) GetUserMenuIds(context.Context, *query.Query, int32) ([]int32, error) {
	panic("not implemented")
}

func (f *fakeRoleRepo) IsSuperAdmin(context.Context, *query.Query, int32) (bool, error) {
	panic("not implemented")
}

type fakeMenuRepo struct {
	allMenus       []*model.SysMenu
	allMenusErr    error
	getAllMenusNum int
}

func (f *fakeMenuRepo) CreateMenu(context.Context, *query.Query, *model.SysMenu) (*model.SysMenu, error) {
	panic("not implemented")
}

func (f *fakeMenuRepo) UpdateMenu(context.Context, *query.Query, *model.SysMenu) (*model.SysMenu, error) {
	panic("not implemented")
}

func (f *fakeMenuRepo) DeleteById(context.Context, *query.Query, int32) error {
	panic("not implemented")
}

func (f *fakeMenuRepo) GetById(context.Context, *query.Query, int32) (*model.SysMenu, error) {
	panic("not implemented")
}

func (f *fakeMenuRepo) GetByParentId(context.Context, *query.Query, int32) ([]*model.SysMenu, error) {
	panic("not implemented")
}

func (f *fakeMenuRepo) GetAllMenus(context.Context) ([]*model.SysMenu, error) {
	f.getAllMenusNum++
	return f.allMenus, f.allMenusErr
}

func (f *fakeMenuRepo) IsUsedMenuByIds(context.Context, *query.Query, []int32) (bool, error) {
	panic("not implemented")
}

func (f *fakeMenuRepo) IsPermsExists(context.Context, *query.Query, string, int32) (bool, error) {
	panic("not implemented")
}

func (f *fakeMenuRepo) IsPathExists(context.Context, *query.Query, string, int32) (bool, error) {
	panic("not implemented")
}

func (f *fakeMenuRepo) GetRoleIdsByIds(context.Context, int32) ([]int32, error) {
	panic("not implemented")
}
