//go:build integration

package system

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
	gormMysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"snowgo/internal/dal/model"
	"snowgo/internal/dal/repo"
	daoSystem "snowgo/internal/dao/admin/system"
	"snowgo/pkg/xcache"
)

type integrationDeps struct {
	repo  *repo.Repository
	cache xcache.Cache
	rdb   *redis.Client
}

func setupIntegrationDeps(t *testing.T) *integrationDeps {
	t.Helper()

	db := setupTestMySQL(t)
	repository := repo.NewRepository(db, map[string]*gorm.DB{"default": db})

	rdb := setupTestRedis(t)
	cache, err := xcache.NewRedisCache(rdb)
	if err != nil {
		t.Fatalf("create redis cache: %v", err)
	}

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
		_ = rdb.Close()
	})

	return &integrationDeps{
		repo:  repository,
		cache: cache,
		rdb:   rdb,
	}
}

func setupTestMySQL(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		t.Skip("skipping MySQL integration test: MYSQL_DSN is not set")
	}
	assertTestMySQLDSN(t, dsn)

	db, err := gorm.Open(gormMysql.Open(dsn), &gorm.Config{
		SkipDefaultTransaction: true,
		Logger:                 logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Skipf("skipping MySQL integration test: cannot connect to MySQL: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(5)
	sqlDB.SetMaxIdleConns(2)
	sqlDB.SetConnMaxLifetime(time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		t.Skipf("skipping MySQL integration test: cannot ping MySQL: %v", err)
	}

	migrateIntegrationTables(t, db)
	return db
}

func assertTestMySQLDSN(t *testing.T, dsn string) {
	t.Helper()

	cfg, err := mysqlConfigFromDSN(dsn)
	if err != nil {
		t.Fatalf("invalid MYSQL_DSN: %v", err)
	}
	if cfg.DBName == "" {
		t.Fatalf("MYSQL_DSN must include a database name")
	}
	dbName := strings.ToLower(cfg.DBName)
	if !strings.Contains(dbName, "test") {
		t.Fatalf("MYSQL_DSN database must be a test database, got %q", cfg.DBName)
	}
}

func mysqlConfigFromDSN(dsn string) (*mysqlDriver.Config, error) {
	cfg, err := mysqlDriver.ParseDSN(dsn)
	if err == nil {
		return cfg, nil
	}

	u, parseErr := url.Parse(dsn)
	if parseErr != nil {
		return nil, err
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, err
	}

	cfg = mysqlDriver.NewConfig()
	if u.User != nil {
		cfg.User = u.User.Username()
		cfg.Passwd, _ = u.User.Password()
	}
	host, port, splitErr := net.SplitHostPort(u.Host)
	if splitErr != nil {
		host = u.Host
		port = "3306"
	}
	cfg.Net = "tcp"
	cfg.Addr = net.JoinHostPort(host, port)
	cfg.DBName = strings.TrimPrefix(u.Path, "/")
	return cfg, nil
}

func migrateIntegrationTables(t *testing.T, db *gorm.DB) {
	t.Helper()

	if err := db.AutoMigrate(
		&model.SysDict{},
		&model.SysDictItem{},
		&model.SysOperationLog{},
	); err != nil {
		t.Fatalf("migrate integration tables: %v", err)
	}
}

func setupTestRedis(t *testing.T) *redis.Client {
	t.Helper()
	db := 13
	if rawDB := os.Getenv("REDIS_DB"); rawDB != "" {
		var err error
		db, err = strconv.Atoi(rawDB)
		if err != nil {
			t.Fatalf("invalid REDIS_DB %q: %v", rawDB, err)
		}
	}
	client := redis.NewClient(&redis.Options{
		Addr:     redisAddr(),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       db,
		PoolSize: 5,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		t.Skipf("skipping Redis integration test: cannot connect to Redis at %s: %v", redisAddr(), err)
	}
	return client
}

func redisAddr() string {
	if addr := os.Getenv("REDIS_ADDR"); addr != "" {
		return addr
	}
	return "127.0.0.1:6379"
}

func cleanupIntegrationTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	tables := []string{
		model.TableNameSysOperationLog,
		model.TableNameSysDictItem,
		model.TableNameSysDict,
	}

	truncateIntegrationTables(t, db, tables...)
	t.Cleanup(func() {
		truncateIntegrationTables(t, db, tables...)
	})
}

func truncateIntegrationTables(t *testing.T, db *gorm.DB, tables ...string) {
	t.Helper()
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}

	if _, err := sqlDB.Exec("SET FOREIGN_KEY_CHECKS = 0"); err != nil {
		t.Fatalf("disable foreign key checks: %v", err)
	}
	for _, table := range tables {
		if _, err := sqlDB.Exec("TRUNCATE TABLE " + table); err != nil {
			t.Fatalf("truncate table %s: %v", table, err)
		}
	}
	if _, err := sqlDB.Exec("SET FOREIGN_KEY_CHECKS = 1"); err != nil {
		t.Fatalf("enable foreign key checks: %v", err)
	}
}

func newIntegrationDictService(deps *integrationDeps) *DictService {
	operationLogDao := daoSystem.NewOperationLogDao(deps.repo)
	operationLogService := NewOperationLogService(deps.repo, operationLogDao)
	return NewDictService(deps.repo, deps.cache, daoSystem.NewDictDao(deps.repo), operationLogService)
}

func insertIntegrationDict(t *testing.T, db *gorm.DB, code, name string) *model.SysDict {
	t.Helper()
	dict := &model.SysDict{
		Code: code,
		Name: name,
	}
	if err := db.Create(dict).Error; err != nil {
		t.Fatalf("insert integration dict: %v", err)
	}
	return dict
}

func insertIntegrationDictItem(t *testing.T, db *gorm.DB, dict *model.SysDict, itemName, itemCode string, sortOrder int32) *model.SysDictItem {
	t.Helper()
	status := "Active"
	item := &model.SysDictItem{
		DictID:    dict.ID,
		DictCode:  dict.Code,
		ItemName:  itemName,
		ItemCode:  itemCode,
		Status:    &status,
		SortOrder: sortOrder,
	}
	if err := db.Create(item).Error; err != nil {
		t.Fatalf("insert integration dict item: %v", err)
	}
	return item
}

func countRows(t *testing.T, db *gorm.DB, table string, where string, args ...any) int64 {
	t.Helper()

	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
	if where != "" {
		query += " WHERE " + where
	}

	var count int64
	if err := db.Raw(query, args...).Scan(&count).Error; err != nil {
		t.Fatalf("count rows from %s: %v", table, err)
	}
	return count
}

func queryOperationLog(t *testing.T, db *gorm.DB, resource string, resourceID int64, action string) *model.SysOperationLog {
	t.Helper()
	var log model.SysOperationLog
	err := db.Where("resource = ? AND resource_id = ? AND action = ?", resource, resourceID, action).First(&log).Error
	if err != nil {
		t.Fatalf("query operation log: %v", err)
	}
	return &log
}
