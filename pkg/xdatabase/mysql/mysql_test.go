package mysql

import (
	"errors"
	"strings"
	"testing"
	"time"

	"snowgo/config"

	mysqlDriver "github.com/go-sql-driver/mysql"
)

// ============================================================
// processConfig
// ============================================================

func TestProcessConfig(t *testing.T) {
	// === Happy path ===
	t.Run("happy: full config preserved", func(t *testing.T) {
		cfg := processConfig(config.MysqlConfig{
			MaxOpenConns:         200,
			MaxIdleConns:         100,
			ConnMaxLifeTime:      120 * time.Minute,
			ConnMaxIdleTime:      30 * time.Minute,
			SlowSqlThresholdTime: 5 * time.Second,
		})
		if cfg.MaxOpenConns != 200 || cfg.MaxIdleConns != 100 {
			t.Fatalf("expected open=200 idle=100, got open=%d idle=%d", cfg.MaxOpenConns, cfg.MaxIdleConns)
		}
	})

	// === Boundary values ===
	t.Run("boundary: zero open conns defaults to 100", func(t *testing.T) {
		cfg := processConfig(config.MysqlConfig{})
		if cfg.MaxOpenConns != 100 {
			t.Fatalf("expected default MaxOpenConns=100, got %d", cfg.MaxOpenConns)
		}
	})

	t.Run("boundary: zero idle conns defaults to CPU*2+1", func(t *testing.T) {
		cfg := processConfig(config.MysqlConfig{MaxOpenConns: 50})
		if cfg.MaxIdleConns <= 0 || cfg.MaxIdleConns > cfg.MaxOpenConns {
			t.Fatalf("default MaxIdleConns should be positive and <= MaxOpenConns, got %d (open=%d)", cfg.MaxIdleConns, cfg.MaxOpenConns)
		}
	})

	t.Run("boundary: idle capped to open when exceeds", func(t *testing.T) {
		cfg := processConfig(config.MysqlConfig{MaxOpenConns: 50, MaxIdleConns: 200})
		if cfg.MaxIdleConns != 50 {
			t.Fatalf("expected MaxIdleConns capped to MaxOpenConns=50, got %d", cfg.MaxIdleConns)
		}
	})

	t.Run("boundary: zero conn max life defaults to 60min", func(t *testing.T) {
		cfg := processConfig(config.MysqlConfig{})
		if cfg.ConnMaxLifeTime.Minutes() != 60 {
			t.Fatalf("expected default ConnMaxLifeTime=60min, got %v", cfg.ConnMaxLifeTime)
		}
	})

	t.Run("boundary: zero conn max idle defaults to 10min", func(t *testing.T) {
		cfg := processConfig(config.MysqlConfig{})
		if cfg.ConnMaxIdleTime.Minutes() != 10 {
			t.Fatalf("expected default ConnMaxIdleTime=10min, got %v", cfg.ConnMaxIdleTime)
		}
	})

	t.Run("boundary: idle capped to life when exceeds", func(t *testing.T) {
		cfg := processConfig(config.MysqlConfig{ConnMaxLifeTime: 30 * time.Minute, ConnMaxIdleTime: 60 * time.Minute})
		if cfg.ConnMaxIdleTime.Minutes() != 30 {
			t.Fatalf("expected ConnMaxIdleTime capped to ConnMaxLifeTime=30min, got %v", cfg.ConnMaxIdleTime)
		}
	})

	t.Run("boundary: zero slow threshold defaults to 2s", func(t *testing.T) {
		cfg := processConfig(config.MysqlConfig{})
		if cfg.SlowSqlThresholdTime.Seconds() != 2 {
			t.Fatalf("expected default SlowSqlThresholdTime=2s, got %v", cfg.SlowSqlThresholdTime)
		}
	})

	t.Run("boundary: negative values use defaults", func(t *testing.T) {
		cfg := processConfig(config.MysqlConfig{
			MaxOpenConns:    -1,
			MaxIdleConns:    -5,
			ConnMaxLifeTime: -10 * time.Minute,
			ConnMaxIdleTime: -20 * time.Minute,
		})
		if cfg.MaxOpenConns != 100 {
			t.Fatalf("expected default MaxOpenConns for negative, got %d", cfg.MaxOpenConns)
		}
	})
}

// ============================================================
// ensureTimeout
// ============================================================

func TestEnsureTimeout(t *testing.T) {
	// === Happy path ===
	t.Run("happy: adds timeout params to clean DSN", func(t *testing.T) {
		dsn := "user:pass@tcp(127.0.0.1:3306)/dbname?charset=utf8mb4&parseTime=True"
		result := ensureTimeout(dsn)
		if !containsParam(result, "timeout=5s") {
			t.Fatalf("expected timeout=5s in %s", result)
		}
		if !containsParam(result, "readTimeout=10s") {
			t.Fatalf("expected readTimeout=10s in %s", result)
		}
		if !containsParam(result, "writeTimeout=10s") {
			t.Fatalf("expected writeTimeout=10s in %s", result)
		}
	})

	// === Boundary values ===
	t.Run("boundary: preserves existing timeout params", func(t *testing.T) {
		dsn := "user:pass@tcp(127.0.0.1:3306)/dbname?timeout=30s&readTimeout=20s&writeTimeout=20s"
		result := ensureTimeout(dsn)
		if !containsParam(result, "timeout=30s") {
			t.Fatalf("expected existing timeout=30s preserved in %s", result)
		}
	})

	t.Run("boundary: partial params already set", func(t *testing.T) {
		dsn := "user:pass@tcp(127.0.0.1:3306)/dbname?timeout=30s"
		result := ensureTimeout(dsn)
		if !containsParam(result, "timeout=30s") {
			t.Fatalf("expected existing timeout=30s preserved in %s", result)
		}
		if !containsParam(result, "readTimeout=10s") {
			t.Fatalf("expected default readTimeout=10s in %s", result)
		}
	})

	// === Expected errors ===
	t.Run("error: invalid DSN returns unchanged", func(t *testing.T) {
		// ensureTimeout returns original on parse error
		result := ensureTimeout("://invalid")
		if result != "://invalid" {
			t.Fatalf("expected invalid DSN returned unchanged, got %s", result)
		}
	})
}

// ============================================================
// IsDuplicateKeyErr
// ============================================================

func TestIsDuplicateKeyErr(t *testing.T) {
	// === Happy path ===
	t.Run("happy: detects Error 1062", func(t *testing.T) {
		err := &mysqlDriver.MySQLError{Number: 1062, Message: "Duplicate entry"}
		if !IsDuplicateKeyErr(err) {
			t.Fatal("expected IsDuplicateKeyErr to return true for Error 1062")
		}
	})

	// === Boundary values ===
	t.Run("boundary: returns false for nil", func(t *testing.T) {
		if IsDuplicateKeyErr(nil) {
			t.Fatal("expected false for nil error")
		}
	})

	t.Run("boundary: returns false for non-MySQLError", func(t *testing.T) {
		if IsDuplicateKeyErr(errors.New("some error")) {
			t.Fatal("expected false for generic error")
		}
	})

	// === Expected errors ===
	t.Run("error: other MySQL error codes", func(t *testing.T) {
		for _, num := range []uint16{1045, 1049, 1064, 1146, 2002} {
			err := &mysqlDriver.MySQLError{Number: num, Message: "other"}
			if IsDuplicateKeyErr(err) {
				t.Fatalf("expected false for MySQL error %d", num)
			}
		}
	})

	t.Run("happy: wrapped error detection via errors.As", func(t *testing.T) {
		inner := &mysqlDriver.MySQLError{Number: 1062, Message: "Duplicate"}
		wrapped := &myError{msg: "wrapped", cause: inner}
		if !IsDuplicateKeyErr(wrapped) {
			t.Fatal("expected true for wrapped MySQLError 1062")
		}
	})
}

// myError is a simple wrapper to test errors.As traversal
type myError struct {
	msg   string
	cause error
}

func (e *myError) Error() string { return e.msg }
func (e *myError) Unwrap() error { return e.cause }

// ============================================================
// Helpers
// ============================================================

func containsParam(dsn, param string) bool {
	return len(dsn) > 0 && len(param) > 0 &&
		(strings.Contains(dsn, "?"+param) || strings.Contains(dsn, "&"+param) || strings.Contains(dsn, param+"&"))
}
