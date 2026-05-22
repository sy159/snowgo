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
	"snowgo/pkg/xcryption"
)

func TestUserServiceCreateUserIntegration(t *testing.T) {
	deps := setupIntegrationDeps(t)
	db := deps.repo.DB()
	cleanupIntegrationTables(t, db)

	role := insertIntegrationRole(t, db, "it_user_role", "用户角色")
	service := newIntegrationUserService(deps)

	userID, err := service.CreateUser(testUserCtx(), &UserParam{
		Username: "operator",
		Password: "abc123",
		Tel:      "18100000000",
		RoleIds:  []int32{role.ID},
	})
	if err != nil {
		t.Fatalf("CreateUser expected success, got %v", err)
	}
	if userID <= 0 {
		t.Fatalf("expected created user id, got %d", userID)
	}

	var user model.SysUser
	if err := db.Where("id = ?", userID).First(&user).Error; err != nil {
		t.Fatalf("query created user: %v", err)
	}
	if user.Username != "operator" || user.Tel != "18100000000" {
		t.Fatalf("unexpected created user: %+v", user)
	}
	if user.Password == "abc123" || !xcryption.CheckPassword(user.Password, "abc123") {
		t.Fatalf("expected created user password to be bcrypt hash")
	}
	relationCount := countRows(t, db, model.TableNameSysUserRole, "user_id = ? AND role_id = ?", userID, role.ID)
	if relationCount != 1 {
		t.Fatalf("expected user role relation, got count %d", relationCount)
	}

	operationLog := queryOperationLog(t, db, constant.ResourceUser, int64(userID), constant.ActionCreate)
	if operationLog.AfterData == nil || strings.Contains(*operationLog.AfterData, "abc123") || strings.Contains(*operationLog.AfterData, "password") {
		t.Fatalf("expected operation log after_data to mask password, got %+v", operationLog.AfterData)
	}
}

func TestUserServiceUpdateUserIntegration(t *testing.T) {
	deps := setupIntegrationDeps(t)
	db := deps.repo.DB()
	cleanupIntegrationTables(t, db)

	oldRole := insertIntegrationRole(t, db, "it_old_role", "旧角色")
	newRole := insertIntegrationRole(t, db, "it_new_role", "新角色")
	user := insertIntegrationUser(t, db, "operator", "18100000000", oldRole.ID)
	cacheKey := fmt.Sprintf("%s%d", constant.CacheUserRolePrefix, user.ID)
	if err := deps.cache.Set(testUserCtx(), cacheKey, "[1]", time.Hour); err != nil {
		t.Fatalf("prime user role cache: %v", err)
	}

	service := newIntegrationUserService(deps)
	status := constant.UserStatusDisabled
	userID, err := service.UpdateUser(testUserCtx(), &UserParam{
		ID:       user.ID,
		Username: "operator_updated",
		Tel:      "18100000001",
		Status:   &status,
		RoleIds:  []int32{newRole.ID},
	})
	if err != nil {
		t.Fatalf("UpdateUser expected success, got %v", err)
	}
	if userID != user.ID {
		t.Fatalf("expected updated user id %d, got %d", user.ID, userID)
	}

	userCount := countRows(t, db, model.TableNameSysUser, "id = ? AND username = ? AND tel = ? AND status = ?", user.ID, "operator_updated", "18100000001", status)
	if userCount != 1 {
		t.Fatalf("expected user to be updated, got count %d", userCount)
	}
	oldRelationCount := countRows(t, db, model.TableNameSysUserRole, "user_id = ? AND role_id = ?", user.ID, oldRole.ID)
	if oldRelationCount != 0 {
		t.Fatalf("expected old user role relation removed, got count %d", oldRelationCount)
	}
	newRelationCount := countRows(t, db, model.TableNameSysUserRole, "user_id = ? AND role_id = ?", user.ID, newRole.ID)
	if newRelationCount != 1 {
		t.Fatalf("expected new user role relation, got count %d", newRelationCount)
	}
	operationLog := queryOperationLog(t, db, constant.ResourceUser, int64(user.ID), constant.ActionUpdate)
	if operationLog.AfterData == nil || !strings.Contains(*operationLog.AfterData, `"username": "operator_updated"`) {
		t.Fatalf("expected operation log after_data to include updated username, got %+v", operationLog.AfterData)
	}
	if _, ok, err := deps.cache.Get(testUserCtx(), cacheKey); err != nil {
		t.Fatalf("get user role cache: %v", err)
	} else if ok {
		t.Fatalf("expected user role cache %q to be invalidated", cacheKey)
	}
}

func TestUserServiceUpdateUserRollbackIntegration(t *testing.T) {
	deps := setupIntegrationDeps(t)
	db := deps.repo.DB()
	cleanupIntegrationTables(t, db)

	oldRole := insertIntegrationRole(t, db, "it_old_role", "旧角色")
	newRole := insertIntegrationRole(t, db, "it_new_role", "新角色")
	user := insertIntegrationUser(t, db, "operator", "18100000000", oldRole.ID)
	operationLogService := systemService.NewOperationLogService(deps.repo, failingOperationLogRepo{})
	roleService := newIntegrationRoleService(deps)
	service := NewUserService(deps.repo, daoAccount.NewUserDao(deps.repo), deps.cache, roleService, operationLogService)

	_, err := service.UpdateUser(testUserCtx(), &UserParam{
		ID:       user.ID,
		Username: "operator_updated",
		Tel:      "18100000001",
		RoleIds:  []int32{newRole.ID},
	})
	if !errors.Is(err, errIntegrationOperationLog) {
		t.Fatalf("UpdateUser expected operation log error, got %v", err)
	}

	userCount := countRows(t, db, model.TableNameSysUser, "id = ? AND username = ? AND tel = ?", user.ID, "operator", "18100000000")
	if userCount != 1 {
		t.Fatalf("expected user update to rollback, got count %d", userCount)
	}
	oldRelationCount := countRows(t, db, model.TableNameSysUserRole, "user_id = ? AND role_id = ?", user.ID, oldRole.ID)
	if oldRelationCount != 1 {
		t.Fatalf("expected old user role relation to remain, got count %d", oldRelationCount)
	}
	newRelationCount := countRows(t, db, model.TableNameSysUserRole, "user_id = ? AND role_id = ?", user.ID, newRole.ID)
	if newRelationCount != 0 {
		t.Fatalf("expected new user role relation to rollback, got count %d", newRelationCount)
	}
}

func TestUserServiceDeleteByIdIntegration(t *testing.T) {
	deps := setupIntegrationDeps(t)
	db := deps.repo.DB()
	cleanupIntegrationTables(t, db)

	insertIntegrationUser(t, db, "admin", "18000000000")
	role := insertIntegrationRole(t, db, "it_delete_role", "删除用户角色")
	user := insertIntegrationUser(t, db, "operator", "18100000000", role.ID)
	cacheKey := fmt.Sprintf("%s%d", constant.CacheUserRolePrefix, user.ID)
	if err := deps.cache.Set(testUserCtx(), cacheKey, "[1]", time.Hour); err != nil {
		t.Fatalf("prime user role cache: %v", err)
	}

	service := newIntegrationUserService(deps)
	if err := service.DeleteById(testUserCtx(), user.ID); err != nil {
		t.Fatalf("DeleteById expected success, got %v", err)
	}
	userCount := countRows(t, db, model.TableNameSysUser, "id = ?", user.ID)
	if userCount != 0 {
		t.Fatalf("expected user to be deleted, got count %d", userCount)
	}
	relationCount := countRows(t, db, model.TableNameSysUserRole, "user_id = ?", user.ID)
	if relationCount != 0 {
		t.Fatalf("expected user role relations to be deleted, got count %d", relationCount)
	}
	operationLog := queryOperationLog(t, db, constant.ResourceUser, int64(user.ID), constant.ActionDelete)
	if operationLog.BeforeData == nil || !strings.Contains(*operationLog.BeforeData, `"username": "operator"`) {
		t.Fatalf("expected operation log before_data to include username, got %+v", operationLog.BeforeData)
	}
	if _, ok, err := deps.cache.Get(testUserCtx(), cacheKey); err != nil {
		t.Fatalf("get user role cache: %v", err)
	} else if ok {
		t.Fatalf("expected user role cache %q to be invalidated", cacheKey)
	}
}

func TestUserServiceDeleteByIdGuardsIntegration(t *testing.T) {
	deps := setupIntegrationDeps(t)
	db := deps.repo.DB()
	cleanupIntegrationTables(t, db)

	insertIntegrationUser(t, db, "admin", "18000000000")
	service := newIntegrationUserService(deps)

	if err := service.DeleteById(testUserCtx(), 1); !errors.Is(err, ErrDeleteSelf) {
		t.Fatalf("DeleteById self expected ErrDeleteSelf, got %v", err)
	}
	userCount := countRows(t, db, model.TableNameSysUser, "id = ?", 1)
	if userCount != 1 {
		t.Fatalf("expected self user to remain, got count %d", userCount)
	}
}

func TestUserServiceResetPwdByIdIntegration(t *testing.T) {
	deps := setupIntegrationDeps(t)
	db := deps.repo.DB()
	cleanupIntegrationTables(t, db)

	user := insertIntegrationUser(t, db, "operator", "18100000000")
	service := newIntegrationUserService(deps)

	if err := service.ResetPwdById(testUserCtx(), user.ID, "new123"); err != nil {
		t.Fatalf("ResetPwdById expected success, got %v", err)
	}
	var updated model.SysUser
	if err := db.Where("id = ?", user.ID).First(&updated).Error; err != nil {
		t.Fatalf("query updated user: %v", err)
	}
	if updated.Password == user.Password || !xcryption.CheckPassword(updated.Password, "new123") {
		t.Fatalf("expected reset password to be updated bcrypt hash")
	}
	operationLog := queryOperationLog(t, db, constant.ResourceUser, int64(user.ID), constant.ActionUpdate)
	if operationLog.Description == nil || !strings.Contains(*operationLog.Description, "重置") {
		t.Fatalf("expected reset password operation log, got %+v", operationLog.Description)
	}
	if operationLog.AfterData == nil || *operationLog.AfterData != "{}" {
		t.Fatalf("expected reset password operation log after_data empty, got %+v", operationLog.AfterData)
	}
}

func TestUserServiceResetPwdByIdRollbackIntegration(t *testing.T) {
	deps := setupIntegrationDeps(t)
	db := deps.repo.DB()
	cleanupIntegrationTables(t, db)

	user := insertIntegrationUser(t, db, "operator", "18100000000")
	operationLogService := systemService.NewOperationLogService(deps.repo, failingOperationLogRepo{})
	roleService := newIntegrationRoleService(deps)
	service := NewUserService(deps.repo, daoAccount.NewUserDao(deps.repo), deps.cache, roleService, operationLogService)

	err := service.ResetPwdById(testUserCtx(), user.ID, "new123")
	if !errors.Is(err, errIntegrationOperationLog) {
		t.Fatalf("ResetPwdById expected operation log error, got %v", err)
	}
	var after model.SysUser
	if err := db.Where("id = ?", user.ID).First(&after).Error; err != nil {
		t.Fatalf("query user after rollback: %v", err)
	}
	if after.Password != user.Password {
		t.Fatalf("expected reset password to rollback")
	}
}
