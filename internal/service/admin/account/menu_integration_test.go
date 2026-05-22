//go:build integration

package account

import (
	"errors"
	"strings"
	"testing"
	"time"

	"snowgo/internal/constant"
	"snowgo/internal/dal/model"
	daoAccount "snowgo/internal/dao/admin/account"
	systemService "snowgo/internal/service/admin/system"
)

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
