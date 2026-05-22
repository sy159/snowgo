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

type failingOperationLogRepo struct{}

func (failingOperationLogRepo) Create(context.Context, *query.Query, *model.SysOperationLog) (*model.SysOperationLog, error) {
	return nil, errIntegrationOperationLog
}

func (failingOperationLogRepo) GetOperationLogList(context.Context, *daoSystem.OperationLogCondition) ([]*model.SysOperationLog, int64, error) {
	panic("not implemented")
}
