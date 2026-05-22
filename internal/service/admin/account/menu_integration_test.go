//go:build integration

package account

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"snowgo/internal/constant"
	"snowgo/internal/dal/model"
	daoAccount "snowgo/internal/dao/admin/account"
	systemService "snowgo/internal/service/admin/system"
)

func TestMenuServiceCreateMenuIntegration(t *testing.T) {
	deps := setupIntegrationDeps(t)
	db := deps.repo.DB()
	cleanupIntegrationTables(t, db)

	parent := insertIntegrationMenu(t, db, 0, constant.MenuTypeDir, "系统管理")
	cacheKey := constant.CacheMenuTree
	if err := deps.cache.Set(testUserCtx(), cacheKey, `[{"id":1}]`, time.Hour); err != nil {
		t.Fatalf("prime menu tree cache: %v", err)
	}

	path := "/system/user"
	icon := "user"
	perms := "system:user:list"
	service := newIntegrationMenuService(deps)
	menuID, err := service.CreateMenu(testUserCtx(), &MenuParam{
		ParentID:  parent.ID,
		MenuType:  constant.MenuTypeMenu,
		Name:      "用户管理",
		Path:      &path,
		Icon:      &icon,
		Perms:     &perms,
		SortOrder: 2,
	})
	if err != nil {
		t.Fatalf("CreateMenu expected success, got %v", err)
	}
	if menuID <= 0 {
		t.Fatalf("expected created menu id, got %d", menuID)
	}
	menuCount := countRows(t, db, model.TableNameSysMenu, "id = ? AND parent_id = ? AND name = ? AND path = ? AND perms = ? AND sort_order = ?", menuID, parent.ID, "用户管理", path, perms, 2)
	if menuCount != 1 {
		t.Fatalf("expected menu to be created, got count %d", menuCount)
	}
	operationLog := queryOperationLog(t, db, constant.ResourceMenu, int64(menuID), constant.ActionCreate)
	if operationLog.AfterData == nil || !strings.Contains(*operationLog.AfterData, `"name": "用户管理"`) {
		t.Fatalf("expected operation log after_data to include menu name, got %+v", operationLog.AfterData)
	}
	if _, ok, err := deps.cache.Get(testUserCtx(), cacheKey); err != nil {
		t.Fatalf("get menu tree cache: %v", err)
	} else if ok {
		t.Fatalf("expected menu tree cache %q to be invalidated", cacheKey)
	}
}

func TestMenuServiceCreateMenuGuardsIntegration(t *testing.T) {
	deps := setupIntegrationDeps(t)
	db := deps.repo.DB()
	cleanupIntegrationTables(t, db)

	existingPath := "/system/user"
	existingPerms := "system:user:list"
	existing := insertIntegrationMenu(t, db, 0, constant.MenuTypeMenu, "已有菜单")
	if err := db.Model(&model.SysMenu{}).Where("id = ?", existing.ID).Updates(map[string]any{
		"path":  existingPath,
		"perms": existingPerms,
	}).Error; err != nil {
		t.Fatalf("update existing menu path/perms: %v", err)
	}

	service := newIntegrationMenuService(deps)
	path := "/missing-parent"
	perms := "missing:parent"
	_, err := service.CreateMenu(testUserCtx(), &MenuParam{
		ParentID:  999,
		MenuType:  constant.MenuTypeMenu,
		Name:      "不存在父级菜单",
		Path:      &path,
		Perms:     &perms,
		SortOrder: 1,
	})
	if !errors.Is(err, ErrMenuParentInvalid) {
		t.Fatalf("CreateMenu missing parent expected ErrMenuParentInvalid, got %v", err)
	}
	menuCount := countRows(t, db, model.TableNameSysMenu, "name = ?", "不存在父级菜单")
	if menuCount != 0 {
		t.Fatalf("expected menu not created when parent missing, got count %d", menuCount)
	}

	newPath := "/system/role"
	_, err = service.CreateMenu(testUserCtx(), &MenuParam{
		MenuType:  constant.MenuTypeMenu,
		Name:      "重复权限菜单",
		Path:      &newPath,
		Perms:     &existingPerms,
		SortOrder: 1,
	})
	if !errors.Is(err, ErrMenuPermsExist) {
		t.Fatalf("CreateMenu duplicate perms expected ErrMenuPermsExist, got %v", err)
	}
	menuCount = countRows(t, db, model.TableNameSysMenu, "name = ?", "重复权限菜单")
	if menuCount != 0 {
		t.Fatalf("expected menu not created when perms duplicate, got count %d", menuCount)
	}

	newPerms := "system:role:list"
	_, err = service.CreateMenu(testUserCtx(), &MenuParam{
		MenuType:  constant.MenuTypeMenu,
		Name:      "重复路径菜单",
		Path:      &existingPath,
		Perms:     &newPerms,
		SortOrder: 1,
	})
	if !errors.Is(err, ErrMenuPathExist) {
		t.Fatalf("CreateMenu duplicate path expected ErrMenuPathExist, got %v", err)
	}
	menuCount = countRows(t, db, model.TableNameSysMenu, "name = ?", "重复路径菜单")
	if menuCount != 0 {
		t.Fatalf("expected menu not created when path duplicate, got count %d", menuCount)
	}
}

func TestMenuServiceCreateMenuRollbackIntegration(t *testing.T) {
	deps := setupIntegrationDeps(t)
	db := deps.repo.DB()
	cleanupIntegrationTables(t, db)

	operationLogService := systemService.NewOperationLogService(deps.repo, failingOperationLogRepo{})
	service := NewMenuService(deps.repo, deps.cache, daoAccount.NewMenuDao(deps.repo), operationLogService)
	path := "/rollback/menu"
	perms := "rollback:menu"
	_, err := service.CreateMenu(testUserCtx(), &MenuParam{
		MenuType:  constant.MenuTypeMenu,
		Name:      "待回滚菜单",
		Path:      &path,
		Perms:     &perms,
		SortOrder: 1,
	})
	if !errors.Is(err, errIntegrationOperationLog) {
		t.Fatalf("CreateMenu expected operation log error, got %v", err)
	}
	menuCount := countRows(t, db, model.TableNameSysMenu, "name = ?", "待回滚菜单")
	if menuCount != 0 {
		t.Fatalf("expected menu create to rollback, got count %d", menuCount)
	}
}

func TestMenuServiceUpdateMenuIntegration(t *testing.T) {
	deps := setupIntegrationDeps(t)
	db := deps.repo.DB()
	cleanupIntegrationTables(t, db)

	oldPath := "/system/user"
	oldPerms := "system:user:list"
	parent := insertIntegrationMenu(t, db, 0, constant.MenuTypeDir, "系统管理")
	menu := insertIntegrationMenu(t, db, 0, constant.MenuTypeMenu, "用户管理")
	if err := db.Model(&model.SysMenu{}).Where("id = ?", menu.ID).Updates(map[string]any{
		"path":  oldPath,
		"perms": oldPerms,
	}).Error; err != nil {
		t.Fatalf("update existing menu path/perms: %v", err)
	}
	role := insertIntegrationRole(t, db, "it_menu_update_role", "菜单更新角色")
	insertIntegrationRoleMenu(t, db, role.ID, menu.ID)

	menuCacheKey := constant.CacheMenuTree
	roleCacheKey := fmt.Sprintf("%s%d", constant.CacheRoleMenuPrefix, role.ID)
	if err := deps.cache.Set(testUserCtx(), menuCacheKey, `[{"id":1}]`, time.Hour); err != nil {
		t.Fatalf("prime menu tree cache: %v", err)
	}
	if err := deps.cache.Set(testUserCtx(), roleCacheKey, `[{"id":1}]`, time.Hour); err != nil {
		t.Fatalf("prime role menu cache: %v", err)
	}

	newPath := "/system/user-updated"
	newPerms := "system:user:update"
	service := newIntegrationMenuService(deps)
	err := service.UpdateMenu(testUserCtx(), &MenuParam{
		ID:        menu.ID,
		ParentID:  parent.ID,
		MenuType:  constant.MenuTypeMenu,
		Name:      "用户管理更新",
		Path:      &newPath,
		Perms:     &newPerms,
		SortOrder: 3,
	})
	if err != nil {
		t.Fatalf("UpdateMenu expected success, got %v", err)
	}
	menuCount := countRows(t, db, model.TableNameSysMenu, "id = ? AND parent_id = ? AND name = ? AND path = ? AND perms = ? AND sort_order = ?", menu.ID, parent.ID, "用户管理更新", newPath, newPerms, 3)
	if menuCount != 1 {
		t.Fatalf("expected menu to be updated, got count %d", menuCount)
	}
	operationLog := queryOperationLog(t, db, constant.ResourceMenu, int64(menu.ID), constant.ActionUpdate)
	if operationLog.BeforeData == nil || !strings.Contains(*operationLog.BeforeData, `"name": "用户管理"`) {
		t.Fatalf("expected operation log before_data to include old menu name, got %+v", operationLog.BeforeData)
	}
	if operationLog.AfterData == nil || !strings.Contains(*operationLog.AfterData, `"name": "用户管理更新"`) {
		t.Fatalf("expected operation log after_data to include updated menu name, got %+v", operationLog.AfterData)
	}
	for _, key := range []string{menuCacheKey, roleCacheKey} {
		if _, ok, err := deps.cache.Get(testUserCtx(), key); err != nil {
			t.Fatalf("get cache %s: %v", key, err)
		} else if ok {
			t.Fatalf("expected cache %q to be invalidated", key)
		}
	}
}

func TestMenuServiceUpdateMenuGuardsIntegration(t *testing.T) {
	deps := setupIntegrationDeps(t)
	db := deps.repo.DB()
	cleanupIntegrationTables(t, db)

	existingPath := "/system/user"
	existingPerms := "system:user:list"
	existing := insertIntegrationMenu(t, db, 0, constant.MenuTypeMenu, "已有菜单")
	if err := db.Model(&model.SysMenu{}).Where("id = ?", existing.ID).Updates(map[string]any{
		"path":  existingPath,
		"perms": existingPerms,
	}).Error; err != nil {
		t.Fatalf("update existing menu path/perms: %v", err)
	}
	menu := insertIntegrationMenu(t, db, 0, constant.MenuTypeMenu, "待更新菜单")
	service := newIntegrationMenuService(deps)

	path := "/self-parent"
	perms := "self:parent"
	err := service.UpdateMenu(testUserCtx(), &MenuParam{
		ID:       menu.ID,
		ParentID: menu.ID,
		MenuType: constant.MenuTypeMenu,
		Name:     "自引用菜单",
		Path:     &path,
		Perms:    &perms,
	})
	if !errors.Is(err, ErrMenuParentSelf) {
		t.Fatalf("UpdateMenu self parent expected ErrMenuParentSelf, got %v", err)
	}
	menuCount := countRows(t, db, model.TableNameSysMenu, "id = ? AND name = ?", menu.ID, "待更新菜单")
	if menuCount != 1 {
		t.Fatalf("expected menu to remain after self parent failure, got count %d", menuCount)
	}

	err = service.UpdateMenu(testUserCtx(), &MenuParam{
		ID:       menu.ID,
		ParentID: 999,
		MenuType: constant.MenuTypeMenu,
		Name:     "不存在父级更新",
		Path:     &path,
		Perms:    &perms,
	})
	if !errors.Is(err, ErrMenuParentInvalid) {
		t.Fatalf("UpdateMenu missing parent expected ErrMenuParentInvalid, got %v", err)
	}

	newPath := "/system/role"
	err = service.UpdateMenu(testUserCtx(), &MenuParam{
		ID:       menu.ID,
		MenuType: constant.MenuTypeMenu,
		Name:     "重复权限更新",
		Path:     &newPath,
		Perms:    &existingPerms,
	})
	if !errors.Is(err, ErrMenuPermsExist) {
		t.Fatalf("UpdateMenu duplicate perms expected ErrMenuPermsExist, got %v", err)
	}

	newPerms := "system:role:list"
	err = service.UpdateMenu(testUserCtx(), &MenuParam{
		ID:       menu.ID,
		MenuType: constant.MenuTypeMenu,
		Name:     "重复路径更新",
		Path:     &existingPath,
		Perms:    &newPerms,
	})
	if !errors.Is(err, ErrMenuPathExist) {
		t.Fatalf("UpdateMenu duplicate path expected ErrMenuPathExist, got %v", err)
	}

	menuCount = countRows(t, db, model.TableNameSysMenu, "id = ? AND name = ?", menu.ID, "待更新菜单")
	if menuCount != 1 {
		t.Fatalf("expected menu to remain after guard failures, got count %d", menuCount)
	}
}

func TestMenuServiceUpdateMenuRollbackIntegration(t *testing.T) {
	deps := setupIntegrationDeps(t)
	db := deps.repo.DB()
	cleanupIntegrationTables(t, db)

	oldPath := "/rollback/menu"
	oldPerms := "rollback:menu"
	menu := insertIntegrationMenu(t, db, 0, constant.MenuTypeMenu, "待回滚菜单")
	if err := db.Model(&model.SysMenu{}).Where("id = ?", menu.ID).Updates(map[string]any{
		"path":  oldPath,
		"perms": oldPerms,
	}).Error; err != nil {
		t.Fatalf("update existing menu path/perms: %v", err)
	}
	operationLogService := systemService.NewOperationLogService(deps.repo, failingOperationLogRepo{})
	service := NewMenuService(deps.repo, deps.cache, daoAccount.NewMenuDao(deps.repo), operationLogService)
	newPath := "/rollback/menu-updated"
	newPerms := "rollback:menu:update"
	err := service.UpdateMenu(testUserCtx(), &MenuParam{
		ID:        menu.ID,
		MenuType:  constant.MenuTypeMenu,
		Name:      "回滚后的菜单",
		Path:      &newPath,
		Perms:     &newPerms,
		SortOrder: 5,
	})
	if !errors.Is(err, errIntegrationOperationLog) {
		t.Fatalf("UpdateMenu expected operation log error, got %v", err)
	}
	menuCount := countRows(t, db, model.TableNameSysMenu, "id = ? AND name = ? AND path = ? AND perms = ?", menu.ID, "待回滚菜单", oldPath, oldPerms)
	if menuCount != 1 {
		t.Fatalf("expected menu update to rollback, got count %d", menuCount)
	}
}

func TestMenuServiceDeleteMenuByIdIntegration(t *testing.T) {
	deps := setupIntegrationDeps(t)
	db := deps.repo.DB()
	cleanupIntegrationTables(t, db)

	menu := insertIntegrationMenu(t, db, 0, constant.MenuTypeMenu, "待删除菜单")
	cacheKey := constant.CacheMenuTree
	if err := deps.cache.Set(testUserCtx(), cacheKey, `[{"id":1}]`, time.Hour); err != nil {
		t.Fatalf("prime menu tree cache: %v", err)
	}

	service := newIntegrationMenuService(deps)
	err := service.DeleteMenuById(testUserCtx(), menu.ID)
	if err != nil {
		t.Fatalf("DeleteMenuById expected success, got %v", err)
	}

	menuCount := countRows(t, db, model.TableNameSysMenu, "id = ?", menu.ID)
	if menuCount != 0 {
		t.Fatalf("expected menu to be deleted, got count %d", menuCount)
	}
	operationLog := queryOperationLog(t, db, constant.ResourceMenu, int64(menu.ID), constant.ActionDelete)
	if operationLog.BeforeData == nil || !strings.Contains(*operationLog.BeforeData, `"name": "待删除菜单"`) {
		t.Fatalf("expected operation log before_data to include deleted menu, got %+v", operationLog.BeforeData)
	}
	if _, ok, err := deps.cache.Get(testUserCtx(), cacheKey); err != nil {
		t.Fatalf("get menu tree cache: %v", err)
	} else if ok {
		t.Fatalf("expected menu tree cache %q to be invalidated", cacheKey)
	}
}

func TestMenuServiceDeleteMenuByIdGuardsIntegration(t *testing.T) {
	deps := setupIntegrationDeps(t)
	db := deps.repo.DB()
	cleanupIntegrationTables(t, db)

	parent := insertIntegrationMenu(t, db, 0, constant.MenuTypeMenu, "父菜单")
	insertIntegrationMenu(t, db, parent.ID, constant.MenuTypeBtn, "子按钮")
	usedMenu := insertIntegrationMenu(t, db, 0, constant.MenuTypeMenu, "已绑定菜单")
	role := insertIntegrationRole(t, db, "it_menu_role", "菜单角色")
	insertIntegrationRoleMenu(t, db, role.ID, usedMenu.ID)
	service := newIntegrationMenuService(deps)

	if err := service.DeleteMenuById(testUserCtx(), parent.ID); !errors.Is(err, ErrMenuHasChildren) {
		t.Fatalf("DeleteMenuById parent expected ErrMenuHasChildren, got %v", err)
	}
	parentCount := countRows(t, db, model.TableNameSysMenu, "id = ?", parent.ID)
	if parentCount != 1 {
		t.Fatalf("expected parent menu to remain, got count %d", parentCount)
	}

	if err := service.DeleteMenuById(testUserCtx(), usedMenu.ID); !errors.Is(err, ErrMenuUsedByRole) {
		t.Fatalf("DeleteMenuById used menu expected ErrMenuUsedByRole, got %v", err)
	}
	usedMenuCount := countRows(t, db, model.TableNameSysMenu, "id = ?", usedMenu.ID)
	if usedMenuCount != 1 {
		t.Fatalf("expected used menu to remain, got count %d", usedMenuCount)
	}
}

func TestMenuServiceDeleteMenuByIdRollbackIntegration(t *testing.T) {
	deps := setupIntegrationDeps(t)
	db := deps.repo.DB()
	cleanupIntegrationTables(t, db)

	menu := insertIntegrationMenu(t, db, 0, constant.MenuTypeMenu, "待回滚菜单")
	operationLogService := systemService.NewOperationLogService(deps.repo, failingOperationLogRepo{})
	service := NewMenuService(deps.repo, deps.cache, daoAccount.NewMenuDao(deps.repo), operationLogService)

	err := service.DeleteMenuById(testUserCtx(), menu.ID)
	if !errors.Is(err, errIntegrationOperationLog) {
		t.Fatalf("DeleteMenuById expected operation log error, got %v", err)
	}
	menuCount := countRows(t, db, model.TableNameSysMenu, "id = ?", menu.ID)
	if menuCount != 1 {
		t.Fatalf("expected menu delete to rollback, got count %d", menuCount)
	}
}
