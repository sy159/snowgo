package account

import (
	"errors"
	"testing"
	"time"

	"snowgo/internal/constant"
	"snowgo/internal/dal/model"
)

func TestMenuServiceEarlyValidation(t *testing.T) {
	service := &MenuService{}

	if err := service.UpdateMenu(testUserCtx(), &MenuParam{}); !errors.Is(err, ErrMenuIDInvalid) {
		t.Fatalf("UpdateMenu expected ErrMenuIDInvalid, got %v", err)
	}
	if err := service.DeleteMenuById(testUserCtx(), 0); !errors.Is(err, ErrMenuIDInvalid) {
		t.Fatalf("DeleteMenuById expected ErrMenuIDInvalid, got %v", err)
	}
}

func TestMenuServiceGetMenuTree(t *testing.T) {
	t.Run("cache hit", func(t *testing.T) {
		cache := newFakeCache()
		cache.values[constant.CacheMenuTree] = `[{"id":1,"parent_id":0,"menu_type":"Dir","name":"系统","sort_order":1,"children":[]}]`
		repo := &fakeMenuRepo{}
		service := &MenuService{menuDao: repo, cache: cache}

		got, err := service.GetMenuTree(testUserCtx())
		if err != nil {
			t.Fatalf("expected success, got %v", err)
		}
		if len(got) != 1 || got[0].Name != "系统" {
			t.Fatalf("unexpected cached tree: %+v", got)
		}
		if repo.getAllMenusNum != 0 {
			t.Fatalf("expected dao not called, got %d calls", repo.getAllMenusNum)
		}
	})

	t.Run("cache miss builds sorted tree and writes cache", func(t *testing.T) {
		path := "/system/user"
		icon := "user"
		perms := "system:user:create"
		now := time.Now()
		cache := newFakeCache()
		repo := &fakeMenuRepo{allMenus: []*model.SysMenu{
			{ID: 3, ParentID: 2, MenuType: constant.MenuTypeBtn, Name: "新增", Perms: &perms, SortOrder: 1, CreatedAt: &now, UpdatedAt: &now},
			{ID: 1, ParentID: 0, MenuType: constant.MenuTypeDir, Name: "系统", Icon: &icon, SortOrder: 2, CreatedAt: &now, UpdatedAt: &now},
			{ID: 4, ParentID: 0, MenuType: constant.MenuTypeDir, Name: "首页", SortOrder: 1, CreatedAt: &now, UpdatedAt: &now},
			{ID: 2, ParentID: 1, MenuType: constant.MenuTypeMenu, Name: "用户", Path: &path, SortOrder: 1, CreatedAt: &now, UpdatedAt: &now},
		}}
		service := &MenuService{menuDao: repo, cache: cache}

		got, err := service.GetMenuTree(testUserCtx())
		if err != nil {
			t.Fatalf("expected success, got %v", err)
		}
		if len(got) != 2 || got[0].Name != "首页" || got[1].Name != "系统" {
			t.Fatalf("expected root nodes sorted by sort_order, got %+v", got)
		}
		systemNode := got[1]
		if len(systemNode.Children) != 1 || systemNode.Children[0].Name != "用户" {
			t.Fatalf("expected system node to contain user child, got %+v", systemNode.Children)
		}
		userNode := systemNode.Children[0]
		if userNode.Path != path || len(userNode.Children) != 1 || userNode.Children[0].Perms != perms {
			t.Fatalf("unexpected nested menu data: %+v", userNode)
		}
		if cache.sets[constant.CacheMenuTree] == "" {
			t.Fatalf("expected menu tree to be cached")
		}
		wantTTL := constant.CacheMenuTreeExpirationDay * 24 * time.Hour
		if cache.expirations[constant.CacheMenuTree] != wantTTL {
			t.Fatalf("expected cache ttl %v, got %v", wantTTL, cache.expirations[constant.CacheMenuTree])
		}
	})

	t.Run("broken cache falls back to dao", func(t *testing.T) {
		cache := newFakeCache()
		cache.values[constant.CacheMenuTree] = "not-json"
		repo := &fakeMenuRepo{allMenus: []*model.SysMenu{{ID: 1, ParentID: 0, MenuType: constant.MenuTypeDir, Name: "系统"}}}
		service := &MenuService{menuDao: repo, cache: cache}

		got, err := service.GetMenuTree(testUserCtx())
		if err != nil {
			t.Fatalf("expected success, got %v", err)
		}
		if len(got) != 1 || got[0].ID != 1 {
			t.Fatalf("expected dao tree, got %+v", got)
		}
	})

	t.Run("dao error", func(t *testing.T) {
		service := &MenuService{
			menuDao: &fakeMenuRepo{allMenusErr: errTestDAO},
			cache:   newFakeCache(),
		}

		_, err := service.GetMenuTree(testUserCtx())
		if !errors.Is(err, errTestDAO) {
			t.Fatalf("expected dao error, got %v", err)
		}
	})
}
