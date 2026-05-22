package system

import (
	"context"
	"errors"
	"time"

	"snowgo/internal/dal/model"
	"snowgo/internal/dal/query"
	daoSystem "snowgo/internal/dao/admin/system"
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

type fakeCache struct {
	values      map[string]string
	sets        map[string]string
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

func (f *fakeCache) Delete(context.Context, ...string) (int64, error) {
	panic("not implemented")
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

type fakeDictRepo struct {
	itemList         []*model.SysDictItem
	itemListErr      error
	getItemListCalls int
}

func (f *fakeDictRepo) GetDictById(context.Context, int32) (*model.SysDict, error) {
	panic("not implemented")
}

func (f *fakeDictRepo) GetDictList(context.Context, *daoSystem.DictListCondition) ([]*model.SysDict, int64, error) {
	panic("not implemented")
}

func (f *fakeDictRepo) IsCodeDuplicate(context.Context, string, int32) (bool, error) {
	panic("not implemented")
}

func (f *fakeDictRepo) CreateDict(context.Context, *query.Query, *model.SysDict) (*model.SysDict, error) {
	panic("not implemented")
}

func (f *fakeDictRepo) UpdateDict(context.Context, *query.Query, *model.SysDict) (*model.SysDict, error) {
	panic("not implemented")
}

func (f *fakeDictRepo) DeleteById(context.Context, *query.Query, int32) error {
	panic("not implemented")
}

func (f *fakeDictRepo) DeleteItemByDictID(context.Context, *query.Query, int32) error {
	panic("not implemented")
}

func (f *fakeDictRepo) UpdateItemByDictID(context.Context, *query.Query, int32, string) error {
	panic("not implemented")
}

func (f *fakeDictRepo) GetItemListByDictCode(context.Context, string) ([]*model.SysDictItem, error) {
	f.getItemListCalls++
	return f.itemList, f.itemListErr
}

func (f *fakeDictRepo) IsCodeItemDuplicate(context.Context, int32, string, int32) (bool, error) {
	panic("not implemented")
}

func (f *fakeDictRepo) CreateDictItem(context.Context, *query.Query, *model.SysDictItem) (*model.SysDictItem, error) {
	panic("not implemented")
}

func (f *fakeDictRepo) GetDictItemById(context.Context, int32) (*model.SysDictItem, error) {
	panic("not implemented")
}

func (f *fakeDictRepo) UpdateDictItem(context.Context, *query.Query, *model.SysDictItem) (*model.SysDictItem, error) {
	panic("not implemented")
}

func (f *fakeDictRepo) DeleteItemByID(context.Context, *query.Query, int32) error {
	panic("not implemented")
}

type fakeLoginLogRepo struct {
	createdLog *model.SysLoginLog
	createErr  error
	list       []*model.SysLoginLog
	total      int64
	listErr    error
	condition  *daoSystem.LoginLogCondition
}

func (f *fakeLoginLogRepo) Create(_ context.Context, log *model.SysLoginLog) (*model.SysLoginLog, error) {
	f.createdLog = log
	return log, f.createErr
}

func (f *fakeLoginLogRepo) GetLoginLogList(_ context.Context, condition *daoSystem.LoginLogCondition) ([]*model.SysLoginLog, int64, error) {
	f.condition = condition
	return f.list, f.total, f.listErr
}
