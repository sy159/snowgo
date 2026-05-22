//go:build integration

package account

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"snowgo/internal/constant"
	"snowgo/internal/dal/model"
	"snowgo/internal/dal/query"
	daoAccount "snowgo/internal/dao/admin/account"
	daoSystem "snowgo/internal/dao/admin/system"
	systemService "snowgo/internal/service/admin/system"
)

var errIntegrationOperationLog = errors.New("operation log integration error")

func TestRoleServiceUpdateRoleIntegration(t *testing.T) {
	deps := setupIntegrationDeps(t)
	db := deps.repo.DB()
	cleanupIntegrationTables(t, db)

	insertIntegrationRole(t, db, "super_admin", "超级管理员")
	insertIntegrationUser(t, db, "admin", "18000000000", constant.SuperAdminRoleId)
	role := insertIntegrationRole(t, db, "it_role", "集成测试角色")
	oldMenu := insertIntegrationMenu(t, db, 0, constant.MenuTypeMenu, "旧菜单")
	newMenu := insertIntegrationMenu(t, db, 0, constant.MenuTypeMenu, "新菜单")
	insertIntegrationRoleMenu(t, db, role.ID, oldMenu.ID)

	cacheKey := fmt.Sprintf("%s%d", constant.CacheRoleMenuPrefix, role.ID)
	if err := deps.cache.Set(testUserCtx(), cacheKey, `[{"id":1}]`, time.Hour); err != nil {
		t.Fatalf("prime role cache: %v", err)
	}

	service := newIntegrationRoleService(deps)
	err := service.UpdateRole(testUserCtx(), &RoleParam{
		ID:          role.ID,
		Name:        "更新后的角色",
		Code:        "it_role_updated",
		Description: "updated",
		MenuIds:     []int32{newMenu.ID},
	})
	if err != nil {
		t.Fatalf("UpdateRole expected success, got %v", err)
	}

	roleCount := countRows(t, db, model.TableNameSysRole, "id = ? AND code = ?", role.ID, "it_role_updated")
	if roleCount != 1 {
		t.Fatalf("expected role to be updated, got count %d", roleCount)
	}
	oldRelationCount := countRows(t, db, model.TableNameSysRoleMenu, "role_id = ? AND menu_id = ?", role.ID, oldMenu.ID)
	if oldRelationCount != 0 {
		t.Fatalf("expected old role menu relation removed, got count %d", oldRelationCount)
	}
	newRelationCount := countRows(t, db, model.TableNameSysRoleMenu, "role_id = ? AND menu_id = ?", role.ID, newMenu.ID)
	if newRelationCount != 1 {
		t.Fatalf("expected new role menu relation, got count %d", newRelationCount)
	}

	operationLog := queryOperationLog(t, db, constant.ResourceRole, int64(role.ID), constant.ActionUpdate)
	if operationLog.AfterData == nil || !strings.Contains(*operationLog.AfterData, `"code": "it_role_updated"`) {
		t.Fatalf("expected operation log after_data to include role code, got %+v", operationLog.AfterData)
	}
	if _, ok, err := deps.cache.Get(testUserCtx(), cacheKey); err != nil {
		t.Fatalf("get role cache: %v", err)
	} else if ok {
		t.Fatalf("expected role cache %q to be invalidated", cacheKey)
	}
}

func TestRoleServiceCreateRoleIntegration(t *testing.T) {
	deps := setupIntegrationDeps(t)
	db := deps.repo.DB()
	cleanupIntegrationTables(t, db)

	insertIntegrationRole(t, db, "super_admin", "超级管理员")
	insertIntegrationUser(t, db, "admin", "18000000000", constant.SuperAdminRoleId)
	menu := insertIntegrationMenu(t, db, 0, constant.MenuTypeMenu, "角色菜单")
	insertIntegrationRoleMenu(t, db, constant.SuperAdminRoleId, menu.ID)

	service := newIntegrationRoleService(deps)
	roleID, err := service.CreateRole(testUserCtx(), &RoleParam{
		Name:        "新角色",
		Code:        "it_role_create",
		Description: "created",
		MenuIds:     []int32{menu.ID},
	})
	if err != nil {
		t.Fatalf("CreateRole expected success, got %v", err)
	}
	if roleID <= 0 {
		t.Fatalf("expected created role id, got %d", roleID)
	}
	roleCount := countRows(t, db, model.TableNameSysRole, "id = ? AND code = ?", roleID, "it_role_create")
	if roleCount != 1 {
		t.Fatalf("expected created role, got count %d", roleCount)
	}
	relationCount := countRows(t, db, model.TableNameSysRoleMenu, "role_id = ? AND menu_id = ?", roleID, menu.ID)
	if relationCount != 1 {
		t.Fatalf("expected created role menu relation, got count %d", relationCount)
	}
	operationLog := queryOperationLog(t, db, constant.ResourceRole, int64(roleID), constant.ActionCreate)
	if operationLog.AfterData == nil || !strings.Contains(*operationLog.AfterData, `"code": "it_role_create"`) {
		t.Fatalf("expected operation log after_data to include role code, got %+v", operationLog.AfterData)
	}
}

func TestRoleServiceCreateRoleGuardsIntegration(t *testing.T) {
	deps := setupIntegrationDeps(t)
	db := deps.repo.DB()
	cleanupIntegrationTables(t, db)

	insertIntegrationRole(t, db, "super_admin", "超级管理员")
	insertIntegrationUser(t, db, "admin", "18000000000", constant.SuperAdminRoleId)
	service := newIntegrationRoleService(deps)

	_, err := service.CreateRole(testUserCtx(), &RoleParam{
		Name:    "不存在菜单角色",
		Code:    "it_missing_menu",
		MenuIds: []int32{999},
	})
	if !errors.Is(err, ErrRoleMenuNotExist) {
		t.Fatalf("CreateRole missing menu expected ErrRoleMenuNotExist, got %v", err)
	}
	roleCount := countRows(t, db, model.TableNameSysRole, "code = ?", "it_missing_menu")
	if roleCount != 0 {
		t.Fatalf("expected role not created when menu missing, got count %d", roleCount)
	}
}

func TestRoleServiceDeleteRoleIntegration(t *testing.T) {
	deps := setupIntegrationDeps(t)
	db := deps.repo.DB()
	cleanupIntegrationTables(t, db)

	insertIntegrationRole(t, db, "super_admin", "超级管理员")
	role := insertIntegrationRole(t, db, "it_role_delete", "待删除角色")
	menu := insertIntegrationMenu(t, db, 0, constant.MenuTypeMenu, "角色菜单")
	insertIntegrationRoleMenu(t, db, role.ID, menu.ID)
	cacheKey := fmt.Sprintf("%s%d", constant.CacheRoleMenuPrefix, role.ID)
	if err := deps.cache.Set(testUserCtx(), cacheKey, `[{"id":1}]`, time.Hour); err != nil {
		t.Fatalf("prime role cache: %v", err)
	}

	service := newIntegrationRoleService(deps)
	if err := service.DeleteRole(testUserCtx(), role.ID); err != nil {
		t.Fatalf("DeleteRole expected success, got %v", err)
	}
	roleCount := countRows(t, db, model.TableNameSysRole, "id = ?", role.ID)
	if roleCount != 0 {
		t.Fatalf("expected role to be deleted, got count %d", roleCount)
	}
	relationCount := countRows(t, db, model.TableNameSysRoleMenu, "role_id = ?", role.ID)
	if relationCount != 0 {
		t.Fatalf("expected role menu relations to be deleted, got count %d", relationCount)
	}
	operationLog := queryOperationLog(t, db, constant.ResourceRole, int64(role.ID), constant.ActionDelete)
	if operationLog.BeforeData == nil || !strings.Contains(*operationLog.BeforeData, `"code": "it_role_delete"`) {
		t.Fatalf("expected operation log before_data to include role code, got %+v", operationLog.BeforeData)
	}
	if _, ok, err := deps.cache.Get(testUserCtx(), cacheKey); err != nil {
		t.Fatalf("get role cache: %v", err)
	} else if ok {
		t.Fatalf("expected role cache %q to be invalidated", cacheKey)
	}
}

func TestRoleServiceDeleteRoleGuardsIntegration(t *testing.T) {
	deps := setupIntegrationDeps(t)
	db := deps.repo.DB()
	cleanupIntegrationTables(t, db)

	insertIntegrationRole(t, db, "super_admin", "超级管理员")
	role := insertIntegrationRole(t, db, "it_role_used", "被使用角色")
	insertIntegrationUser(t, db, "operator", "18100000000", role.ID)
	service := newIntegrationRoleService(deps)

	if err := service.DeleteRole(testUserCtx(), role.ID); !errors.Is(err, ErrRoleUsed) {
		t.Fatalf("DeleteRole used role expected ErrRoleUsed, got %v", err)
	}
	roleCount := countRows(t, db, model.TableNameSysRole, "id = ?", role.ID)
	if roleCount != 1 {
		t.Fatalf("expected used role to remain, got count %d", roleCount)
	}
}

func TestRoleServiceUpdateRoleRollbackIntegration(t *testing.T) {
	deps := setupIntegrationDeps(t)
	db := deps.repo.DB()
	cleanupIntegrationTables(t, db)

	insertIntegrationRole(t, db, "super_admin", "超级管理员")
	insertIntegrationUser(t, db, "admin", "18000000000", constant.SuperAdminRoleId)
	role := insertIntegrationRole(t, db, "it_role_rollback", "待回滚角色")
	oldMenu := insertIntegrationMenu(t, db, 0, constant.MenuTypeMenu, "旧菜单")
	newMenu := insertIntegrationMenu(t, db, 0, constant.MenuTypeMenu, "新菜单")
	insertIntegrationRoleMenu(t, db, role.ID, oldMenu.ID)

	operationLogService := systemService.NewOperationLogService(deps.repo, failingOperationLogRepo{})
	service := NewRoleService(deps.repo, daoAccount.NewRoleDao(deps.repo), deps.cache, operationLogService)
	err := service.UpdateRole(testUserCtx(), &RoleParam{
		ID:      role.ID,
		Name:    "回滚后的角色",
		Code:    "it_role_rollback_updated",
		MenuIds: []int32{newMenu.ID},
	})
	if !errors.Is(err, errIntegrationOperationLog) {
		t.Fatalf("UpdateRole expected operation log error, got %v", err)
	}

	roleCount := countRows(t, db, model.TableNameSysRole, "id = ? AND code = ?", role.ID, "it_role_rollback")
	if roleCount != 1 {
		t.Fatalf("expected role update to rollback, got count %d", roleCount)
	}
	oldRelationCount := countRows(t, db, model.TableNameSysRoleMenu, "role_id = ? AND menu_id = ?", role.ID, oldMenu.ID)
	if oldRelationCount != 1 {
		t.Fatalf("expected old role menu relation to remain, got count %d", oldRelationCount)
	}
	newRelationCount := countRows(t, db, model.TableNameSysRoleMenu, "role_id = ? AND menu_id = ?", role.ID, newMenu.ID)
	if newRelationCount != 0 {
		t.Fatalf("expected new role menu relation to rollback, got count %d", newRelationCount)
	}
}

type failingOperationLogRepo struct{}

func (failingOperationLogRepo) Create(context.Context, *query.Query, *model.SysOperationLog) (*model.SysOperationLog, error) {
	return nil, errIntegrationOperationLog
}

func (failingOperationLogRepo) GetOperationLogList(context.Context, *daoSystem.OperationLogCondition) ([]*model.SysOperationLog, int64, error) {
	panic("not implemented")
}
