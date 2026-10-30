//go:build integration

package account

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"

	"snowgo/internal/constant"
	"snowgo/internal/dal/model"
	daoAccount "snowgo/internal/dao/admin/account"
	systemService "snowgo/internal/service/admin/system"
	"snowgo/pkg/xcryption"
)

func insertIntegrationUserAdmin(t *testing.T, db *gorm.DB) {
	t.Helper()
	role := insertIntegrationRole(t, db, "it_super_admin", "超级管理员")
	admin := insertIntegrationUser(t, db, "admin", "18000000000", role.ID)
	if role.ID != constant.SuperAdminRoleId || admin.ID != 1 {
		t.Fatal("expected clean fixture with super role and operator ID 1")
	}
}

func TestUserRoleAssignmentIntegration(t *testing.T) {
	for _, tc := range []struct {
		name        string
		update      bool
		self        bool
		protected   bool
		grantSuper  bool
		grantExcess bool
		allowed     bool
	}{
		{name: "create cannot grant super", grantSuper: true},
		{name: "create cannot grant excess", grantExcess: true},
		{name: "update cannot grant excess", update: true, grantExcess: true},
		{name: "self escalation denied", update: true, self: true, grantSuper: true},
		{name: "cannot remove super role", update: true, protected: true},
		{name: "create subset permitted", allowed: true},
		{name: "update subset permitted", update: true, allowed: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			deps := setupIntegrationDeps(t)
			db := deps.repo.DB()
			cleanupIntegrationTables(t, db)
			super := insertIntegrationRole(t, db, "it_super", "超级管理员")
			operatorRole := insertIntegrationRole(t, db, "it_operator", "操作员")
			excess := insertIntegrationRole(t, db, "it_excess", "高权限角色")
			subset := insertIntegrationRole(t, db, "it_subset", "低权限角色")
			operator := insertIntegrationUser(t, db, "admin", "18000000000", operatorRole.ID)
			owned := insertIntegrationMenu(t, db, 0, constant.MenuTypeBtn, "已有权限")
			other := insertIntegrationMenu(t, db, 0, constant.MenuTypeBtn, "额外权限")
			insertIntegrationRoleMenu(t, db, operatorRole.ID, owned.ID)
			insertIntegrationRoleMenu(t, db, subset.ID, owned.ID)
			insertIntegrationRoleMenu(t, db, excess.ID, other.ID)
			p := &UserParam{Username: "subject", Password: "abc123", Tel: "18100000000", RoleIds: []int32{subset.ID}}
			if tc.grantSuper {
				p.RoleIds = []int32{super.ID}
			}
			if tc.grantExcess {
				p.RoleIds = []int32{excess.ID}
			}
			if tc.update {
				targetRole := subset.ID
				if tc.protected {
					targetRole = super.ID
					p.RoleIds = nil
				}
				target := operator
				if !tc.self {
					target = insertIntegrationUser(t, db, "target", "18200000000", targetRole)
				}
				p.ID = target.ID
			}
			s := newIntegrationUserService(deps)
			var id int32
			var err error
			if tc.update {
				id, err = s.UpdateUser(testUserCtx(), p)
			} else {
				id, err = s.CreateUser(testUserCtx(), p)
			}
			if !tc.allowed {
				if !errors.Is(err, ErrUserRoleNotAuthorized) {
					t.Fatalf("expected forbidden, got %v", err)
				}
				if countRows(t, db, model.TableNameSysUser, "username = ?", "subject") != 0 || countRows(t, db, model.TableNameSysOperationLog, "") != 0 {
					t.Fatal("rejected assignment persisted business changes or audit")
				}
				if tc.protected && countRows(t, db, model.TableNameSysUserRole, "user_id = ? AND role_id = ?", p.ID, super.ID) != 1 {
					t.Fatal("super role was removed")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if countRows(t, db, model.TableNameSysUserRole, "user_id = ? AND role_id = ?", id, subset.ID) != 1 || countRows(t, db, model.TableNameSysOperationLog, "resource_id = ?", id) != 1 {
				t.Fatal("authorized role assignment or audit missing")
			}
		})
	}
}

func TestValidateRefreshUserIntegration(t *testing.T) {
	deps := setupIntegrationDeps(t)
	db := deps.repo.DB()
	cleanupIntegrationTables(t, db)
	user := insertIntegrationUser(t, db, "refresh_user", "18100000000")
	s := newIntegrationUserService(deps)
	if err := s.ValidateRefreshUser(testUserCtx(), user.ID); err != nil {
		t.Fatal(err)
	}
	if err := db.Model(user).Update("status", constant.UserStatusDisabled).Error; err != nil {
		t.Fatal(err)
	}
	if err := s.ValidateRefreshUser(testUserCtx(), user.ID); !errors.Is(err, ErrRefreshUserInvalid) {
		t.Fatalf("expected disabled account rejected, got %v", err)
	}
	if err := db.Delete(user).Error; err != nil {
		t.Fatal(err)
	}
	if err := s.ValidateRefreshUser(testUserCtx(), user.ID); !errors.Is(err, ErrRefreshUserInvalid) {
		t.Fatalf("expected deleted account rejected, got %v", err)
	}
}

func TestUserServiceCreateUserIntegration(t *testing.T) {
	deps := setupIntegrationDeps(t)
	db := deps.repo.DB()
	cleanupIntegrationTables(t, db)
	insertIntegrationUserAdmin(t, db)

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

func TestUserServiceCreateUserRollbackIntegration(t *testing.T) {
	deps := setupIntegrationDeps(t)
	db := deps.repo.DB()
	cleanupIntegrationTables(t, db)
	insertIntegrationUserAdmin(t, db)

	role := insertIntegrationRole(t, db, "it_user_create_rollback_role", "用户创建回滚角色")
	operationLogService := systemService.NewOperationLogService(deps.repo, failingOperationLogRepo{})
	roleService := newIntegrationRoleService(deps)
	service := NewUserService(deps.repo, daoAccount.NewUserDao(deps.repo), deps.cache, roleService, operationLogService)

	_, err := service.CreateUser(testUserCtx(), &UserParam{
		Username: "rollback_operator",
		Password: "abc123",
		Tel:      "18100000009",
		RoleIds:  []int32{role.ID},
	})
	if !errors.Is(err, errIntegrationOperationLog) {
		t.Fatalf("CreateUser expected operation log error, got %v", err)
	}
	userCount := countRows(t, db, model.TableNameSysUser, "username = ?", "rollback_operator")
	if userCount != 0 {
		t.Fatalf("expected user create to rollback, got count %d", userCount)
	}
	relationCount := countRows(t, db, model.TableNameSysUserRole, "role_id = ?", role.ID)
	if relationCount != 0 {
		t.Fatalf("expected user role create to rollback, got count %d", relationCount)
	}
}

func TestUserServiceUpdateUserIntegration(t *testing.T) {
	deps := setupIntegrationDeps(t)
	db := deps.repo.DB()
	cleanupIntegrationTables(t, db)
	insertIntegrationUserAdmin(t, db)

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
	insertIntegrationUserAdmin(t, db)

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

func TestUserServiceDeleteByIdRollbackIntegration(t *testing.T) {
	deps := setupIntegrationDeps(t)
	db := deps.repo.DB()
	cleanupIntegrationTables(t, db)

	insertIntegrationUser(t, db, "admin", "18000000000")
	role := insertIntegrationRole(t, db, "it_delete_rollback_role", "删除用户回滚角色")
	user := insertIntegrationUser(t, db, "rollback_operator", "18100000009", role.ID)
	operationLogService := systemService.NewOperationLogService(deps.repo, failingOperationLogRepo{})
	roleService := newIntegrationRoleService(deps)
	service := NewUserService(deps.repo, daoAccount.NewUserDao(deps.repo), deps.cache, roleService, operationLogService)

	err := service.DeleteById(testUserCtx(), user.ID)
	if !errors.Is(err, errIntegrationOperationLog) {
		t.Fatalf("DeleteById expected operation log error, got %v", err)
	}
	userCount := countRows(t, db, model.TableNameSysUser, "id = ?", user.ID)
	if userCount != 1 {
		t.Fatalf("expected user delete to rollback, got count %d", userCount)
	}
	relationCount := countRows(t, db, model.TableNameSysUserRole, "user_id = ? AND role_id = ?", user.ID, role.ID)
	if relationCount != 1 {
		t.Fatalf("expected user role delete to rollback, got count %d", relationCount)
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
