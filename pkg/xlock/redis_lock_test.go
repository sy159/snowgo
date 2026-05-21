package xlock

import (
	"testing"
)

// ============================================================
// NewRedisLock
// ============================================================

func TestNewRedisLock(t *testing.T) {
	// === Expected errors ===
	t.Run("error: nil client returns error", func(t *testing.T) {
		_, err := NewRedisLock(nil, nil)
		if err == nil {
			t.Fatal("expected error for nil redis client")
		}
	})

	// === Happy path ===
	t.Run("happy: nil logger uses defaultLogger", func(t *testing.T) {
		// NewRedisLock with nil client fails before logger check,
		// so we verify defaultLogger independently below.
		// The integration with a real Redis client is in redis_lock_test.go.
	})
}

// ============================================================
// defaultLogger
// ============================================================

func TestDefaultLogger(t *testing.T) {
	// === Happy path ===
	t.Run("happy: Infof does not panic", func(t *testing.T) {
		l := &defaultLogger{}
		l.Infof("test %s %d", "hello", 42)
	})

	t.Run("happy: Errorf does not panic", func(t *testing.T) {
		l := &defaultLogger{}
		l.Errorf("error %s", "boom")
	})

	// === Boundary values ===
	t.Run("boundary: empty format string", func(t *testing.T) {
		l := &defaultLogger{}
		l.Infof("")
		l.Errorf("")
	})

	t.Run("boundary: no args", func(t *testing.T) {
		l := &defaultLogger{}
		l.Infof("static message")
		l.Errorf("static error")
	})
}

// ============================================================
// RedisLockContext
// ============================================================

func TestRedisLockContext_Err(t *testing.T) {
	// === Happy path ===
	t.Run("happy: Err returns nil when no error", func(t *testing.T) {
		lc := newRedisLockContext(nil, nil)
		if lc.Err() != nil {
			t.Fatalf("expected nil error, got %v", lc.Err())
		}
	})

	t.Run("happy: Err returns the underlying error", func(t *testing.T) {
		expectedErr := errTest
		lc := newRedisLockContext(nil, expectedErr)
		if lc.Err() != expectedErr {
			t.Fatalf("expected error %v, got %v", expectedErr, lc.Err())
		}
	})
}

// errTest is a sentinel for testing
var errTest = &testError{"test lock error"}

type testError struct{ msg string }

func (e *testError) Error() string { return e.msg }
