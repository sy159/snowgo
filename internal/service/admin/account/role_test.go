package account

import (
	"errors"
	"testing"

	"snowgo/internal/constant"
	"snowgo/internal/dal/model"
	e "snowgo/pkg/xerror"
)

func TestRoleServiceEarlyValidation(t *testing.T) {
	service := &RoleService{}

	if err := service.UpdateRole(testUserCtx(), &RoleParam{}); !errors.Is(err, ErrRoleIDInvalid) {
		t.Fatalf("UpdateRole expected ErrRoleIDInvalid, got %v", err)
	}
	if _, err := service.GetRoleById(testUserCtx(), 0); !errors.Is(err, ErrRoleIDInvalid) {
		t.Fatalf("GetRoleById expected ErrRoleIDInvalid, got %v", err)
	}

	err := service.DeleteRole(testUserCtx(), constant.SuperAdminRoleId)
	var bizErr *e.BizError
	if !errors.As(err, &bizErr) || bizErr.Code.GetErrCode() != e.SuperAdminRoleCannotDelete.GetErrCode() {
		t.Fatalf("DeleteRole super admin expected SuperAdminRoleCannotDelete, got %v", err)
	}
}

func TestRoleServiceGetRoleMenuListByRuleID(t *testing.T) {
	cacheKey := constant.CacheRoleMenuPrefix + "2"

	t.Run("cache hit", func(t *testing.T) {
		cache := newFakeCache()
		cache.values[cacheKey] = `[{"id":1,"menu_type":"Btn","name":"Create","perms":"account:user:create"}]`
		repo := &fakeRoleRepo{}
		service := &RoleService{roleDao: repo, cache: cache}

		got, err := service.GetRoleMenuListByRuleID(testUserCtx(), 2)
		if err != nil {
			t.Fatalf("expected success, got %v", err)
		}
		if len(got) != 1 || got[0].Perms != "account:user:create" {
			t.Fatalf("unexpected cached menu list: %+v", got)
		}
		if repo.getMenuListCall != 0 {
			t.Fatalf("expected dao not called, got %d calls", repo.getMenuListCall)
		}
	})

	t.Run("cache miss reads dao and writes cache", func(t *testing.T) {
		path := "/account/user"
		perms := "account:user:create"
		cache := newFakeCache()
		repo := &fakeRoleRepo{menuList: []*model.SysMenu{
			{ID: 1, MenuType: constant.MenuTypeMenu, Name: "User", Path: &path},
			{ID: 2, MenuType: constant.MenuTypeBtn, Name: "Create", Perms: &perms},
		}}
		service := &RoleService{roleDao: repo, cache: cache}

		got, err := service.GetRoleMenuListByRuleID(testUserCtx(), 2)
		if err != nil {
			t.Fatalf("expected success, got %v", err)
		}
		if len(got) != 2 || got[0].Path != path || got[1].Perms != perms {
			t.Fatalf("unexpected dao menu list: %+v", got)
		}
		if cache.sets[cacheKey] == "" {
			t.Fatalf("expected role menu list to be cached")
		}
	})

	t.Run("broken cache falls back to dao", func(t *testing.T) {
		cache := newFakeCache()
		cache.values[cacheKey] = "not-json"
		repo := &fakeRoleRepo{menuList: []*model.SysMenu{{ID: 3, MenuType: constant.MenuTypeBtn, Name: "Delete"}}}
		service := &RoleService{roleDao: repo, cache: cache}

		got, err := service.GetRoleMenuListByRuleID(testUserCtx(), 2)
		if err != nil {
			t.Fatalf("expected success, got %v", err)
		}
		if len(got) != 1 || got[0].ID != 3 {
			t.Fatalf("expected dao menu list, got %+v", got)
		}
	})
}

func TestRoleServiceGetRolePermsListByRuleID(t *testing.T) {
	perms := "account:user:create"
	service := &RoleService{
		roleDao: &fakeRoleRepo{menuList: []*model.SysMenu{
			{ID: 1, MenuType: constant.MenuTypeDir, Name: "Account"},
			{ID: 2, MenuType: constant.MenuTypeMenu, Name: "User"},
			{ID: 3, MenuType: constant.MenuTypeBtn, Name: "Create", Perms: &perms},
			{ID: 4, MenuType: constant.MenuTypeBtn, Name: "Empty"},
		}},
		cache: newFakeCache(),
	}

	got, err := service.GetRolePermsListByRuleID(testUserCtx(), 2)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if len(got) != 1 || got[0] != perms {
		t.Fatalf("expected only button perms, got %v", got)
	}
}
