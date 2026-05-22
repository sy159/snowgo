package system

import (
	"errors"
	"testing"
	"time"

	"snowgo/internal/constant"
	"snowgo/internal/dal/model"
)

func TestDictServiceEarlyValidation(t *testing.T) {
	service := &DictService{}

	if _, err := service.GetDictList(testUserCtx(), &DictListCondition{StartTime: "bad-time"}); !errors.Is(err, ErrTimeFormat) {
		t.Fatalf("GetDictList expected ErrTimeFormat, got %v", err)
	}
	if _, err := service.UpdateDict(testUserCtx(), &DictParam{}); !errors.Is(err, ErrDictCodeNotFound) {
		t.Fatalf("UpdateDict expected ErrDictCodeNotFound, got %v", err)
	}
	if err := service.DeleteById(testUserCtx(), 0); !errors.Is(err, ErrDictCodeNotFound) {
		t.Fatalf("DeleteById expected ErrDictCodeNotFound, got %v", err)
	}
	if _, err := service.GetItemListByCode(testUserCtx(), ""); !errors.Is(err, ErrDictCodeNotFound) {
		t.Fatalf("GetItemListByCode expected ErrDictCodeNotFound, got %v", err)
	}
	if _, err := service.UpdateItem(testUserCtx(), &DictItemParam{}); !errors.Is(err, ErrDictCodeItemNotFound) {
		t.Fatalf("UpdateItem expected ErrDictCodeItemNotFound, got %v", err)
	}
	if err := service.DeleteItemById(testUserCtx(), 0); !errors.Is(err, ErrDictCodeItemNotFound) {
		t.Fatalf("DeleteItemById expected ErrDictCodeItemNotFound, got %v", err)
	}
}

func TestDictServiceGetItemListByCode(t *testing.T) {
	cacheKey := constant.SystemDictPrefix + "status"

	t.Run("cache hit", func(t *testing.T) {
		cache := newFakeCache()
		cache.values[cacheKey] = `[{"id":1,"item_name":"启用","item_code":"Active","sort_order":1}]`
		repo := &fakeDictRepo{}
		service := &DictService{dictRepo: repo, cache: cache}

		got, err := service.GetItemListByCode(testUserCtx(), "status")
		if err != nil {
			t.Fatalf("expected success, got %v", err)
		}
		if len(got) != 1 || got[0].ItemCode != "Active" || got[0].ItemName != "启用" {
			t.Fatalf("unexpected cached item list: %+v", got)
		}
		if repo.getItemListCalls != 0 {
			t.Fatalf("expected dao not called, got %d calls", repo.getItemListCalls)
		}
	})

	t.Run("cache miss reads dao and writes cache", func(t *testing.T) {
		status := constant.ActiveStatus
		cache := newFakeCache()
		repo := &fakeDictRepo{itemList: []*model.SysDictItem{
			{ID: 1, ItemName: "启用", ItemCode: "Active", Status: &status, SortOrder: 1},
			{ID: 2, ItemName: "禁用", ItemCode: "Disabled", Status: &status, SortOrder: 2},
		}}
		service := &DictService{dictRepo: repo, cache: cache}

		got, err := service.GetItemListByCode(testUserCtx(), "status")
		if err != nil {
			t.Fatalf("expected success, got %v", err)
		}
		if len(got) != 2 || got[0].ItemCode != "Active" || got[1].ItemCode != "Disabled" {
			t.Fatalf("unexpected dao item list: %+v", got)
		}
		if repo.getItemListCalls != 1 {
			t.Fatalf("expected one dao call, got %d", repo.getItemListCalls)
		}
		if cache.sets[cacheKey] == "" {
			t.Fatalf("expected item list to be cached")
		}
		wantTTL := constant.SystemDictExpirationDay * 24 * time.Hour
		if cache.expirations[cacheKey] != wantTTL {
			t.Fatalf("expected cache ttl %v, got %v", wantTTL, cache.expirations[cacheKey])
		}
	})

	t.Run("empty dao result caches short ttl", func(t *testing.T) {
		cache := newFakeCache()
		repo := &fakeDictRepo{}
		service := &DictService{dictRepo: repo, cache: cache}

		got, err := service.GetItemListByCode(testUserCtx(), "status")
		if err != nil {
			t.Fatalf("expected success, got %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("expected empty item list, got %+v", got)
		}
		if cache.sets[cacheKey] != "[]" {
			t.Fatalf("expected empty list cached as [], got %q", cache.sets[cacheKey])
		}
		if cache.expirations[cacheKey] != time.Hour {
			t.Fatalf("expected empty result ttl %v, got %v", time.Hour, cache.expirations[cacheKey])
		}
	})

	t.Run("broken cache falls back to dao", func(t *testing.T) {
		cache := newFakeCache()
		cache.values[cacheKey] = "not-json"
		repo := &fakeDictRepo{itemList: []*model.SysDictItem{{ID: 3, ItemName: "未知", ItemCode: "Unknown"}}}
		service := &DictService{dictRepo: repo, cache: cache}

		got, err := service.GetItemListByCode(testUserCtx(), "status")
		if err != nil {
			t.Fatalf("expected success, got %v", err)
		}
		if len(got) != 1 || got[0].ID != 3 {
			t.Fatalf("expected dao item list, got %+v", got)
		}
	})

	t.Run("dao error", func(t *testing.T) {
		service := &DictService{
			dictRepo: &fakeDictRepo{itemListErr: errTestDAO},
			cache:    newFakeCache(),
		}

		_, err := service.GetItemListByCode(testUserCtx(), "status")
		if !errors.Is(err, errTestDAO) {
			t.Fatalf("expected dao error, got %v", err)
		}
	})
}
