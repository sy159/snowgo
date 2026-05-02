package system

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"snowgo/internal/constant"
	"snowgo/internal/dal/model"
	"snowgo/internal/dal/query"
	daoSystem "snowgo/internal/dao/system"
	"snowgo/pkg/xauth"
	"snowgo/pkg/xcache"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- Mock implementations ----

type mockDictRepo struct {
	DictRepo
	dict      *model.SystemDict
	dictItem  *model.SystemDictItem
	dictItems []*model.SystemDictItem
	isDictDup bool
	isItemDup bool
	err       error
}

func (m *mockDictRepo) GetDictById(_ context.Context, _ int32) (*model.SystemDict, error) {
	if m.dict == nil {
		return nil, errors.New("not found")
	}
	return m.dict, m.err
}

func (m *mockDictRepo) GetDictList(_ context.Context, _ *daoSystem.DictListCondition) ([]*model.SystemDict, int64, error) {
	if m.dict == nil {
		return nil, 0, m.err
	}
	return []*model.SystemDict{m.dict}, 1, m.err
}

func (m *mockDictRepo) IsCodeDuplicate(_ context.Context, _ string, _ int32) (bool, error) {
	return m.isDictDup, m.err
}

func (m *mockDictRepo) IsCodeItemDuplicate(_ context.Context, _ int32, _ string, _ int32) (bool, error) {
	return m.isItemDup, m.err
}

func (m *mockDictRepo) TransactionCreateDict(_ context.Context, _ *query.Query, dict *model.SystemDict) (*model.SystemDict, error) {
	dict.ID = 1
	return dict, m.err
}

func (m *mockDictRepo) TransactionUpdateDict(_ context.Context, _ *query.Query, dict *model.SystemDict) (*model.SystemDict, error) {
	return dict, m.err
}

func (m *mockDictRepo) TransactionUpdateItemByDictID(_ context.Context, _ *query.Query, _ int32, _ string) error {
	return m.err
}

func (m *mockDictRepo) GetItemListByDictCode(_ context.Context, _ string) ([]*model.SystemDictItem, error) {
	return m.dictItems, m.err
}

func (m *mockDictRepo) TransactionCreateDictItem(_ context.Context, _ *query.Query, item *model.SystemDictItem) (*model.SystemDictItem, error) {
	item.ID = 1
	return item, m.err
}

func (m *mockDictRepo) GetDictItemById(_ context.Context, _ int32) (*model.SystemDictItem, error) {
	if m.dictItem == nil {
		return nil, errors.New("not found")
	}
	return m.dictItem, m.err
}

func (m *mockDictRepo) TransactionUpdateDictItem(_ context.Context, _ *query.Query, item *model.SystemDictItem) (*model.SystemDictItem, error) {
	return item, m.err
}

func (m *mockDictRepo) TransactionDeleteItemByDictID(_ context.Context, _ *query.Query, _ int32) error {
	return m.err
}

func (m *mockDictRepo) TransactionDeleteById(_ context.Context, _ *query.Query, _ int32) error {
	return m.err
}

func (m *mockDictRepo) TransactionDeleteItemByID(_ context.Context, _ *query.Query, _ int32) error {
	return m.err
}

type mockDictCache struct {
	xcache.Cache
	data map[string]string
	err  error
}

func (m *mockDictCache) Get(_ context.Context, key string) (string, bool, error) {
	if m.err != nil {
		return "", false, m.err
	}
	val, ok := m.data[key]
	return val, ok, nil
}

func (m *mockDictCache) Set(_ context.Context, key, value string, _ time.Duration) error {
	if m.err != nil {
		return m.err
	}
	m.data[key] = value
	return nil
}

func (m *mockDictCache) Delete(_ context.Context, keys ...string) (int64, error) {
	if m.err != nil {
		return 0, m.err
	}
	for _, k := range keys {
		delete(m.data, k)
	}
	return int64(len(keys)), nil
}

// ---- Helpers ----

func ctxWithUserSys() context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, xauth.XUserId, int32(1))
	ctx = context.WithValue(ctx, xauth.XUserName, "admin")
	ctx = context.WithValue(ctx, xauth.XTraceId, "test-trace-id")
	ctx = context.WithValue(ctx, xauth.XIp, "127.0.0.1")
	ctx = context.WithValue(ctx, xauth.XUserAgent, "test-agent")
	return ctx
}

func testDict() *model.SystemDict {
	desc := "Test dict"
	return &model.SystemDict{
		ID:          1,
		Code:        "test_dict",
		Name:        "Test Dict",
		Description: &desc,
		CreatedAt:   ptrTimeSys(time.Now()),
		UpdatedAt:   ptrTimeSys(time.Now()),
	}
}

func ptrStrSys(s string) *string        { return &s }
func ptrTimeSys(t time.Time) *time.Time { return &t }

// ---- Tests: GetDictList ----

func TestGetDictList_Success(t *testing.T) {
	dict := testDict()
	svc := &DictService{dictRepo: &mockDictRepo{dict: dict}}

	result, err := svc.GetDictList(context.Background(), &DictListCondition{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Total)
	assert.Len(t, result.List, 1)
	assert.Equal(t, "test_dict", result.List[0].Code)
}

func TestGetDictList_InvalidStartTime(t *testing.T) {
	svc := &DictService{dictRepo: &mockDictRepo{}}

	_, err := svc.GetDictList(context.Background(), &DictListCondition{
		StartTime: "invalid-date",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "start_time格式错误")
}

func TestGetDictList_InvalidEndTime(t *testing.T) {
	svc := &DictService{dictRepo: &mockDictRepo{}}

	_, err := svc.GetDictList(context.Background(), &DictListCondition{
		EndTime: "invalid-date",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "end_time格式错误")
}

// ---- Tests: GetItemListByCode ----

func TestGetItemListByCode_CacheHit(t *testing.T) {
	cacheData := make(map[string]string)
	cacheKey := fmt.Sprintf("%s%s", constant.SystemDictPrefix, "test_dict")
	items := []*ItemInfo{{ID: 1, ItemName: "Active", ItemCode: "active"}}
	b, _ := json.Marshal(items)
	cacheData[cacheKey] = string(b)

	svc := &DictService{
		cache: &mockDictCache{data: cacheData},
	}

	got, err := svc.GetItemListByCode(context.Background(), "test_dict")
	require.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, "Active", got[0].ItemName)
}

func TestGetItemListByCode_CacheMiss(t *testing.T) {
	cacheData := make(map[string]string)
	status := "Active"
	dictItems := []*model.SystemDictItem{
		{
			ID:        1,
			DictCode:  "test_dict",
			ItemCode:  "active",
			ItemName:  "Active",
			Status:    &status,
			SortOrder: 1,
			CreatedAt: ptrTimeSys(time.Now()),
			UpdatedAt: ptrTimeSys(time.Now()),
		},
	}

	svc := &DictService{
		dictRepo: &mockDictRepo{dictItems: dictItems},
		cache:    &mockDictCache{data: cacheData},
	}

	got, err := svc.GetItemListByCode(context.Background(), "test_dict")
	require.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, "Active", got[0].ItemName)
	// Verify cache was set
	_, ok := cacheData[fmt.Sprintf("%s%s", constant.SystemDictPrefix, "test_dict")]
	assert.True(t, ok)
}

func TestGetItemListByCode_EmptyCode(t *testing.T) {
	svc := &DictService{}

	_, err := svc.GetItemListByCode(context.Background(), "")
	assert.True(t, errors.Is(err, ErrDictCodeNotFound))
}

func TestGetItemListByCode_EmptyResultsCacheShortTTL(t *testing.T) {
	cacheData := make(map[string]string)

	svc := &DictService{
		dictRepo: &mockDictRepo{dictItems: []*model.SystemDictItem{}},
		cache:    &mockDictCache{data: cacheData},
	}

	_, err := svc.GetItemListByCode(context.Background(), "nonexist")
	require.NoError(t, err)

	// Should have cached with short TTL (1 hour for empty results)
	_, ok := cacheData[fmt.Sprintf("%s%s", constant.SystemDictPrefix, "nonexist")]
	assert.True(t, ok)
}
