//go:build integration

package system

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"snowgo/internal/constant"
	"snowgo/internal/dal/model"
	"snowgo/internal/dal/query"
	daoSystem "snowgo/internal/dao/admin/system"
)

var errIntegrationOperationLog = errors.New("operation log integration error")

func TestDictServiceCreateDictIntegration(t *testing.T) {
	deps := setupIntegrationDeps(t)
	db := deps.repo.DB()
	cleanupIntegrationTables(t, db)

	service := newIntegrationDictService(deps)
	description := "集成测试字典"
	dictID, err := service.CreateDict(testUserCtx(), &DictParam{
		Code:        "it_create",
		Name:        "创建字典",
		Description: &description,
	})
	if err != nil {
		t.Fatalf("CreateDict expected success, got %v", err)
	}
	if dictID <= 0 {
		t.Fatalf("expected created dict id, got %d", dictID)
	}
	dictCount := countRows(t, db, model.TableNameSysDict, "id = ? AND code = ? AND name = ? AND description = ?", dictID, "it_create", "创建字典", description)
	if dictCount != 1 {
		t.Fatalf("expected dict to be created, got count %d", dictCount)
	}
	operationLog := queryOperationLog(t, db, constant.ResourceDict, int64(dictID), constant.ActionCreate)
	if operationLog.AfterData == nil || !strings.Contains(*operationLog.AfterData, `"code": "it_create"`) {
		t.Fatalf("expected operation log after_data to include dict code, got %+v", operationLog.AfterData)
	}
}

func TestDictServiceCreateDictDuplicateIntegration(t *testing.T) {
	deps := setupIntegrationDeps(t)
	db := deps.repo.DB()
	cleanupIntegrationTables(t, db)

	insertIntegrationDict(t, db, "it_duplicate_dict", "已有字典")
	service := newIntegrationDictService(deps)
	_, err := service.CreateDict(testUserCtx(), &DictParam{
		Code: "it_duplicate_dict",
		Name: "重复字典",
	})
	if !errors.Is(err, ErrDictCodeExist) {
		t.Fatalf("CreateDict duplicate expected ErrDictCodeExist, got %v", err)
	}
	dictCount := countRows(t, db, model.TableNameSysDict, "code = ?", "it_duplicate_dict")
	if dictCount != 1 {
		t.Fatalf("expected duplicate dict not created, got count %d", dictCount)
	}
	logCount := countRows(t, db, model.TableNameSysOperationLog, "resource = ? AND action = ?", constant.ResourceDict, constant.ActionCreate)
	if logCount != 0 {
		t.Fatalf("expected no operation log after duplicate failure, got %d", logCount)
	}
}

func TestDictServiceCreateDictRollbackIntegration(t *testing.T) {
	deps := setupIntegrationDeps(t)
	db := deps.repo.DB()
	cleanupIntegrationTables(t, db)

	operationLogService := NewOperationLogService(deps.repo, failingOperationLogRepo{})
	service := NewDictService(deps.repo, deps.cache, daoSystem.NewDictDao(deps.repo), operationLogService)
	_, err := service.CreateDict(testUserCtx(), &DictParam{
		Code: "it_rollback_dict",
		Name: "回滚字典",
	})
	if !errors.Is(err, errIntegrationOperationLog) {
		t.Fatalf("CreateDict expected operation log error, got %v", err)
	}
	dictCount := countRows(t, db, model.TableNameSysDict, "code = ?", "it_rollback_dict")
	if dictCount != 0 {
		t.Fatalf("expected dict create to rollback, got count %d", dictCount)
	}
}

func TestDictServiceCreateItemIntegration(t *testing.T) {
	deps := setupIntegrationDeps(t)
	db := deps.repo.DB()
	cleanupIntegrationTables(t, db)

	service := newIntegrationDictService(deps)
	ctx := testUserCtx()
	dict := insertIntegrationDict(t, db, "it_status", "集成测试状态")
	cacheKey := constant.SystemDictPrefix + dict.Code
	if err := deps.cache.Set(ctx, cacheKey, `[{"item_code":"stale"}]`, time.Hour); err != nil {
		t.Fatalf("prime dict cache: %v", err)
	}

	itemID, err := service.CreateItem(ctx, &DictItemParam{
		DictID:    dict.ID,
		ItemName:  "启用",
		ItemCode:  "Active",
		SortOrder: 1,
	})
	if err != nil {
		t.Fatalf("CreateItem expected success, got %v", err)
	}

	if itemID <= 0 {
		t.Fatalf("expected created item id, got %d", itemID)
	}
	itemCount := countRows(t, db, model.TableNameSysDictItem, "id = ? AND dict_id = ? AND dict_code = ? AND item_code = ?", itemID, dict.ID, dict.Code, "Active")
	if itemCount != 1 {
		t.Fatalf("expected created dict item, got count %d", itemCount)
	}

	operationLog := queryOperationLog(t, db, constant.ResourceDictItem, int64(itemID), constant.ActionCreate)
	if operationLog.OperatorID != 1 || operationLog.OperatorName != "admin" {
		t.Fatalf("unexpected operation log operator: %+v", operationLog)
	}
	if operationLog.AfterData == nil || !strings.Contains(*operationLog.AfterData, `"item_code": "Active"`) {
		t.Fatalf("expected operation log after_data to include item_code, got %+v", operationLog.AfterData)
	}

	if _, ok, err := deps.cache.Get(ctx, cacheKey); err != nil {
		t.Fatalf("get dict cache: %v", err)
	} else if ok {
		t.Fatalf("expected dict cache %q to be invalidated", cacheKey)
	}
}

func TestDictServiceCreateItemDuplicateIntegration(t *testing.T) {
	deps := setupIntegrationDeps(t)
	db := deps.repo.DB()
	cleanupIntegrationTables(t, db)

	service := newIntegrationDictService(deps)
	ctx := testUserCtx()
	dict := insertIntegrationDict(t, db, "it_duplicate", "集成测试重复值")

	if _, err := service.CreateItem(ctx, &DictItemParam{
		DictID:   dict.ID,
		ItemName: "启用",
		ItemCode: "Active",
	}); err != nil {
		t.Fatalf("first CreateItem expected success, got %v", err)
	}

	_, err := service.CreateItem(ctx, &DictItemParam{
		DictID:   dict.ID,
		ItemName: "启用重复",
		ItemCode: "Active",
	})
	if !errors.Is(err, ErrDictItemCodeExist) {
		t.Fatalf("duplicate CreateItem expected ErrDictItemCodeExist, got %v", err)
	}

	itemCount := countRows(t, db, model.TableNameSysDictItem, "dict_id = ? AND item_code = ?", dict.ID, "Active")
	if itemCount != 1 {
		t.Fatalf("expected only one dict item after duplicate failure, got %d", itemCount)
	}
	logCount := countRows(t, db, model.TableNameSysOperationLog, "resource = ? AND action = ?", constant.ResourceDictItem, constant.ActionCreate)
	if logCount != 1 {
		t.Fatalf("expected only first create operation log, got %d", logCount)
	}
}

func TestDictServiceCreateItemRollbackIntegration(t *testing.T) {
	deps := setupIntegrationDeps(t)
	db := deps.repo.DB()
	cleanupIntegrationTables(t, db)

	operationLogService := NewOperationLogService(deps.repo, failingOperationLogRepo{})
	service := NewDictService(deps.repo, deps.cache, daoSystem.NewDictDao(deps.repo), operationLogService)
	ctx := testUserCtx()
	dict := insertIntegrationDict(t, db, "it_rollback", "集成测试回滚")

	_, err := service.CreateItem(ctx, &DictItemParam{
		DictID:    dict.ID,
		ItemName:  "待回滚",
		ItemCode:  "Rollback",
		SortOrder: 1,
	})
	if !errors.Is(err, errIntegrationOperationLog) {
		t.Fatalf("CreateItem expected operation log error, got %v", err)
	}

	itemCount := countRows(t, db, model.TableNameSysDictItem, "dict_id = ? AND item_code = ?", dict.ID, "Rollback")
	if itemCount != 0 {
		t.Fatalf("expected dict item insert to rollback, got count %d", itemCount)
	}
	logCount := countRows(t, db, model.TableNameSysOperationLog, "resource = ? AND action = ?", constant.ResourceDictItem, constant.ActionCreate)
	if logCount != 0 {
		t.Fatalf("expected no operation log after rollback, got %d", logCount)
	}
}

func TestDictServiceUpdateDictIntegration(t *testing.T) {
	deps := setupIntegrationDeps(t)
	db := deps.repo.DB()
	cleanupIntegrationTables(t, db)

	service := newIntegrationDictService(deps)
	ctx := testUserCtx()
	dict := insertIntegrationDict(t, db, "it_old_code", "旧字典")
	insertIntegrationDictItem(t, db, dict, "启用", "Active", 1)
	oldCacheKey := constant.SystemDictPrefix + "it_old_code"
	newCacheKey := constant.SystemDictPrefix + "it_new_code"
	if err := deps.cache.Set(ctx, oldCacheKey, `[{"item_code":"old"}]`, time.Hour); err != nil {
		t.Fatalf("prime old dict cache: %v", err)
	}
	if err := deps.cache.Set(ctx, newCacheKey, `[{"item_code":"new"}]`, time.Hour); err != nil {
		t.Fatalf("prime new dict cache: %v", err)
	}

	dictID, err := service.UpdateDict(ctx, &DictParam{
		ID:   dict.ID,
		Code: "it_new_code",
		Name: "新字典",
	})
	if err != nil {
		t.Fatalf("UpdateDict expected success, got %v", err)
	}
	if dictID != dict.ID {
		t.Fatalf("expected updated dict id %d, got %d", dict.ID, dictID)
	}
	dictCount := countRows(t, db, model.TableNameSysDict, "id = ? AND code = ? AND name = ?", dict.ID, "it_new_code", "新字典")
	if dictCount != 1 {
		t.Fatalf("expected dict to be updated, got count %d", dictCount)
	}
	itemCount := countRows(t, db, model.TableNameSysDictItem, "dict_id = ? AND dict_code = ?", dict.ID, "it_new_code")
	if itemCount != 1 {
		t.Fatalf("expected dict item dict_code to be updated, got count %d", itemCount)
	}
	operationLog := queryOperationLog(t, db, constant.ResourceDict, int64(dict.ID), constant.ActionUpdate)
	if operationLog.AfterData == nil || !strings.Contains(*operationLog.AfterData, `"code": "it_new_code"`) {
		t.Fatalf("expected operation log after_data to include updated code, got %+v", operationLog.AfterData)
	}
	for _, key := range []string{oldCacheKey, newCacheKey} {
		if _, ok, err := deps.cache.Get(ctx, key); err != nil {
			t.Fatalf("get dict cache %s: %v", key, err)
		} else if ok {
			t.Fatalf("expected dict cache %q to be invalidated", key)
		}
	}
}

func TestDictServiceUpdateDictRollbackIntegration(t *testing.T) {
	deps := setupIntegrationDeps(t)
	db := deps.repo.DB()
	cleanupIntegrationTables(t, db)

	dict := insertIntegrationDict(t, db, "it_update_rollback", "旧字典")
	insertIntegrationDictItem(t, db, dict, "启用", "Active", 1)
	operationLogService := NewOperationLogService(deps.repo, failingOperationLogRepo{})
	service := NewDictService(deps.repo, deps.cache, daoSystem.NewDictDao(deps.repo), operationLogService)

	_, err := service.UpdateDict(testUserCtx(), &DictParam{
		ID:   dict.ID,
		Code: "it_update_rollback_new",
		Name: "新字典",
	})
	if !errors.Is(err, errIntegrationOperationLog) {
		t.Fatalf("UpdateDict expected operation log error, got %v", err)
	}
	dictCount := countRows(t, db, model.TableNameSysDict, "id = ? AND code = ? AND name = ?", dict.ID, "it_update_rollback", "旧字典")
	if dictCount != 1 {
		t.Fatalf("expected dict update to rollback, got count %d", dictCount)
	}
	itemCount := countRows(t, db, model.TableNameSysDictItem, "dict_id = ? AND dict_code = ?", dict.ID, "it_update_rollback")
	if itemCount != 1 {
		t.Fatalf("expected dict item code update to rollback, got count %d", itemCount)
	}
}

func TestDictServiceDeleteByIdIntegration(t *testing.T) {
	deps := setupIntegrationDeps(t)
	db := deps.repo.DB()
	cleanupIntegrationTables(t, db)

	service := newIntegrationDictService(deps)
	ctx := testUserCtx()
	dict := insertIntegrationDict(t, db, "it_delete", "待删除字典")
	insertIntegrationDictItem(t, db, dict, "启用", "Active", 1)
	cacheKey := constant.SystemDictPrefix + dict.Code
	if err := deps.cache.Set(ctx, cacheKey, `[{"item_code":"stale"}]`, time.Hour); err != nil {
		t.Fatalf("prime dict cache: %v", err)
	}

	if err := service.DeleteById(ctx, dict.ID); err != nil {
		t.Fatalf("DeleteById expected success, got %v", err)
	}
	dictCount := countRows(t, db, model.TableNameSysDict, "id = ?", dict.ID)
	if dictCount != 0 {
		t.Fatalf("expected dict to be deleted, got count %d", dictCount)
	}
	itemCount := countRows(t, db, model.TableNameSysDictItem, "dict_id = ?", dict.ID)
	if itemCount != 0 {
		t.Fatalf("expected dict items to be deleted, got count %d", itemCount)
	}
	operationLog := queryOperationLog(t, db, constant.ResourceDict, int64(dict.ID), constant.ActionDelete)
	if operationLog.BeforeData == nil || !strings.Contains(*operationLog.BeforeData, `"code": "it_delete"`) {
		t.Fatalf("expected operation log before_data to include dict code, got %+v", operationLog.BeforeData)
	}
	if _, ok, err := deps.cache.Get(ctx, cacheKey); err != nil {
		t.Fatalf("get dict cache: %v", err)
	} else if ok {
		t.Fatalf("expected dict cache %q to be invalidated", cacheKey)
	}
}

func TestDictServiceDeleteByIdRollbackIntegration(t *testing.T) {
	deps := setupIntegrationDeps(t)
	db := deps.repo.DB()
	cleanupIntegrationTables(t, db)

	dict := insertIntegrationDict(t, db, "it_delete_rollback", "删除回滚字典")
	item := insertIntegrationDictItem(t, db, dict, "启用", "Active", 1)
	operationLogService := NewOperationLogService(deps.repo, failingOperationLogRepo{})
	service := NewDictService(deps.repo, deps.cache, daoSystem.NewDictDao(deps.repo), operationLogService)

	err := service.DeleteById(testUserCtx(), dict.ID)
	if !errors.Is(err, errIntegrationOperationLog) {
		t.Fatalf("DeleteById expected operation log error, got %v", err)
	}
	dictCount := countRows(t, db, model.TableNameSysDict, "id = ?", dict.ID)
	if dictCount != 1 {
		t.Fatalf("expected dict delete to rollback, got count %d", dictCount)
	}
	itemCount := countRows(t, db, model.TableNameSysDictItem, "id = ?", item.ID)
	if itemCount != 1 {
		t.Fatalf("expected dict item delete to rollback, got count %d", itemCount)
	}
}

func TestDictServiceUpdateItemIntegration(t *testing.T) {
	deps := setupIntegrationDeps(t)
	db := deps.repo.DB()
	cleanupIntegrationTables(t, db)

	service := newIntegrationDictService(deps)
	ctx := testUserCtx()
	dict := insertIntegrationDict(t, db, "it_item_update", "字典项更新")
	item := insertIntegrationDictItem(t, db, dict, "启用", "Active", 1)
	status := constant.DisabledStatus
	cacheKey := constant.SystemDictPrefix + dict.Code
	if err := deps.cache.Set(ctx, cacheKey, `[{"item_code":"stale"}]`, time.Hour); err != nil {
		t.Fatalf("prime dict cache: %v", err)
	}

	itemID, err := service.UpdateItem(ctx, &DictItemParam{
		ID:        item.ID,
		DictID:    dict.ID,
		ItemName:  "禁用",
		ItemCode:  "Disabled",
		Status:    &status,
		SortOrder: 9,
	})
	if err != nil {
		t.Fatalf("UpdateItem expected success, got %v", err)
	}
	if itemID != item.ID {
		t.Fatalf("expected updated item id %d, got %d", item.ID, itemID)
	}
	itemCount := countRows(t, db, model.TableNameSysDictItem, "id = ? AND item_name = ? AND item_code = ? AND status = ? AND sort_order = ?", item.ID, "禁用", "Disabled", status, 9)
	if itemCount != 1 {
		t.Fatalf("expected dict item to be updated, got count %d", itemCount)
	}
	operationLog := queryOperationLog(t, db, constant.ResourceDictItem, int64(item.ID), constant.ActionUpdate)
	if operationLog.AfterData == nil || !strings.Contains(*operationLog.AfterData, `"item_code": "Disabled"`) {
		t.Fatalf("expected operation log after_data to include item code, got %+v", operationLog.AfterData)
	}
	if _, ok, err := deps.cache.Get(ctx, cacheKey); err != nil {
		t.Fatalf("get dict cache: %v", err)
	} else if ok {
		t.Fatalf("expected dict cache %q to be invalidated", cacheKey)
	}
}

func TestDictServiceUpdateItemRollbackIntegration(t *testing.T) {
	deps := setupIntegrationDeps(t)
	db := deps.repo.DB()
	cleanupIntegrationTables(t, db)

	dict := insertIntegrationDict(t, db, "it_item_update_rollback", "字典项更新回滚")
	item := insertIntegrationDictItem(t, db, dict, "启用", "Active", 1)
	status := constant.DisabledStatus
	operationLogService := NewOperationLogService(deps.repo, failingOperationLogRepo{})
	service := NewDictService(deps.repo, deps.cache, daoSystem.NewDictDao(deps.repo), operationLogService)

	_, err := service.UpdateItem(testUserCtx(), &DictItemParam{
		ID:        item.ID,
		DictID:    dict.ID,
		ItemName:  "禁用",
		ItemCode:  "Disabled",
		Status:    &status,
		SortOrder: 9,
	})
	if !errors.Is(err, errIntegrationOperationLog) {
		t.Fatalf("UpdateItem expected operation log error, got %v", err)
	}
	itemCount := countRows(t, db, model.TableNameSysDictItem, "id = ? AND item_name = ? AND item_code = ? AND sort_order = ?", item.ID, "启用", "Active", 1)
	if itemCount != 1 {
		t.Fatalf("expected dict item update to rollback, got count %d", itemCount)
	}
}

func TestDictServiceDeleteItemByIdIntegration(t *testing.T) {
	deps := setupIntegrationDeps(t)
	db := deps.repo.DB()
	cleanupIntegrationTables(t, db)

	service := newIntegrationDictService(deps)
	ctx := testUserCtx()
	dict := insertIntegrationDict(t, db, "it_item_delete", "字典项删除")
	item := insertIntegrationDictItem(t, db, dict, "启用", "Active", 1)
	cacheKey := constant.SystemDictPrefix + dict.Code
	if err := deps.cache.Set(ctx, cacheKey, `[{"item_code":"stale"}]`, time.Hour); err != nil {
		t.Fatalf("prime dict cache: %v", err)
	}

	if err := service.DeleteItemById(ctx, item.ID); err != nil {
		t.Fatalf("DeleteItemById expected success, got %v", err)
	}
	itemCount := countRows(t, db, model.TableNameSysDictItem, "id = ?", item.ID)
	if itemCount != 0 {
		t.Fatalf("expected dict item to be deleted, got count %d", itemCount)
	}
	operationLog := queryOperationLog(t, db, constant.ResourceDictItem, int64(item.ID), constant.ActionDelete)
	if operationLog.BeforeData == nil || !strings.Contains(*operationLog.BeforeData, `"item_code": "Active"`) {
		t.Fatalf("expected operation log before_data to include item code, got %+v", operationLog.BeforeData)
	}
	if _, ok, err := deps.cache.Get(ctx, cacheKey); err != nil {
		t.Fatalf("get dict cache: %v", err)
	} else if ok {
		t.Fatalf("expected dict cache %q to be invalidated", cacheKey)
	}
}

func TestDictServiceDeleteItemByIdRollbackIntegration(t *testing.T) {
	deps := setupIntegrationDeps(t)
	db := deps.repo.DB()
	cleanupIntegrationTables(t, db)

	dict := insertIntegrationDict(t, db, "it_item_delete_rollback", "字典项删除回滚")
	item := insertIntegrationDictItem(t, db, dict, "启用", "Active", 1)
	operationLogService := NewOperationLogService(deps.repo, failingOperationLogRepo{})
	service := NewDictService(deps.repo, deps.cache, daoSystem.NewDictDao(deps.repo), operationLogService)

	err := service.DeleteItemById(testUserCtx(), item.ID)
	if !errors.Is(err, errIntegrationOperationLog) {
		t.Fatalf("DeleteItemById expected operation log error, got %v", err)
	}
	itemCount := countRows(t, db, model.TableNameSysDictItem, "id = ?", item.ID)
	if itemCount != 1 {
		t.Fatalf("expected dict item delete to rollback, got count %d", itemCount)
	}
}

type failingOperationLogRepo struct{}

func (failingOperationLogRepo) Create(context.Context, *query.Query, *model.SysOperationLog) (*model.SysOperationLog, error) {
	return nil, errIntegrationOperationLog
}

func (failingOperationLogRepo) GetOperationLogList(context.Context, *daoSystem.OperationLogCondition) ([]*model.SysOperationLog, int64, error) {
	panic("not implemented")
}
