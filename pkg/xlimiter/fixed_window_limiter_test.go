package xlimiter

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// mockCache implements xcache.Cache for unit testing without Redis
type mockCache struct {
	mu       sync.Mutex
	data     map[string]string
	ttls     map[string]time.Time
	evalFn   func(ctx context.Context, script string, keys []string, args ...any) (any, error)
	deleteFn func(ctx context.Context, keys ...string) (int64, error)
}

func newMockCache() *mockCache {
	return &mockCache{
		data: make(map[string]string),
		ttls: make(map[string]time.Time),
	}
}

func (m *mockCache) Get(_ context.Context, key string) (string, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.data[key]
	return v, ok, nil
}

func (m *mockCache) Set(_ context.Context, key string, value string, expiration time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = value
	if expiration > 0 {
		m.ttls[key] = time.Now().Add(expiration)
	}
	return nil
}

func (m *mockCache) Delete(_ context.Context, keys ...string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.deleteFn != nil {
		return m.deleteFn(context.Background(), keys...)
	}
	n := int64(0)
	for _, k := range keys {
		if _, ok := m.data[k]; ok {
			delete(m.data, k)
			delete(m.ttls, k)
			n++
		}
	}
	return n, nil
}

func (m *mockCache) IncrBy(_ context.Context, key string, increment int64) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v := m.data[key]
	var cur int64
	_, _ = fmt.Sscanf(v, "%d", &cur)
	cur += increment
	m.data[key] = fmt.Sprintf("%d", cur)
	return cur, nil
}

func (m *mockCache) DecrBy(_ context.Context, key string, decrement int64) (int64, error) {
	return 0, nil
}

func (m *mockCache) HSet(_ context.Context, key string, field string, value string) error {
	return nil
}

func (m *mockCache) HGet(_ context.Context, key string, field string) (string, bool, error) {
	return "", false, nil
}

func (m *mockCache) HGetAll(_ context.Context, key string) (map[string]string, error) {
	return nil, nil
}

func (m *mockCache) HDel(_ context.Context, key string, fields ...string) (int64, error) {
	return 0, nil
}

func (m *mockCache) HIncrBy(_ context.Context, key string, field string, increment int64) (int64, error) {
	return 0, nil
}

func (m *mockCache) HLen(_ context.Context, key string) (int64, error) {
	return 0, nil
}

func (m *mockCache) ZAdd(_ context.Context, key string, score float64, member string) error {
	return nil
}

func (m *mockCache) ZRem(_ context.Context, key string, members ...string) error {
	return nil
}

func (m *mockCache) ZRange(_ context.Context, key string, start, stop int64) ([]string, error) {
	return nil, nil
}

func (m *mockCache) ZCard(_ context.Context, key string) (int64, error) {
	return 0, nil
}

func (m *mockCache) Exists(_ context.Context, key string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.data[key]
	return ok, nil
}

func (m *mockCache) Expire(_ context.Context, key string, expiration time.Duration) error {
	return nil
}

func (m *mockCache) TTL(_ context.Context, key string) (time.Duration, error) {
	return 0, nil
}

func (m *mockCache) Eval(ctx context.Context, script string, keys []string, args ...any) (any, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.evalFn != nil {
		return m.evalFn(ctx, script, keys, args...)
	}
	key := keys[0]

	maxFails := int64(3)
	windowSec := int64(60)
	if len(args) >= 1 {
		switch v := args[0].(type) {
		case string:
			_, _ = fmt.Sscanf(v, "%d", &maxFails)
		case int64:
			maxFails = v
		}
	}
	if len(args) >= 2 {
		switch v := args[1].(type) {
		case string:
			_, _ = fmt.Sscanf(v, "%d", &windowSec)
		case int64:
			windowSec = v
		}
	}

	// Simulate Redis INCR + EXPIRE
	var cur int64
	if v, ok := m.data[key]; ok {
		_, _ = fmt.Sscanf(v, "%d", &cur)
	}
	cur++
	m.data[key] = fmt.Sprintf("%d", cur)

	allowed := int64(1)
	if cur > maxFails {
		allowed = 0
	}
	return []any{allowed, cur, windowSec}, nil
}

// ============================================================
// NewFixedWindowLimiter
// ============================================================

func TestFixedWindowLimiter_New(t *testing.T) {
	// === Happy path ===
	t.Run("happy: valid parameters", func(t *testing.T) {
		cache := newMockCache()
		l, err := NewFixedWindowLimiter(cache, "test:happy", 60, 5)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if l.windowSec != 60 || l.maxFails != 5 || l.key != "test:happy" {
			t.Fatalf("wrong config: window=%d maxFails=%d key=%s", l.windowSec, l.maxFails, l.key)
		}
	})

	// === Boundary values ===
	t.Run("boundary: windowSecond=0 uses default 60", func(t *testing.T) {
		cache := newMockCache()
		l, err := NewFixedWindowLimiter(cache, "test:zero-window", 0, 5)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if l.windowSec != 60 {
			t.Fatalf("expected default windowSec=60, got %d", l.windowSec)
		}
	})

	t.Run("boundary: maxFails=0 uses default 1", func(t *testing.T) {
		cache := newMockCache()
		l, err := NewFixedWindowLimiter(cache, "test:zero-fails", 10, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if l.maxFails != 1 {
			t.Fatalf("expected default maxFails=1, got %d", l.maxFails)
		}
	})

	t.Run("boundary: negative values use defaults", func(t *testing.T) {
		cache := newMockCache()
		l, err := NewFixedWindowLimiter(cache, "test:negative", -5, -3)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if l.windowSec != 60 || l.maxFails != 1 {
			t.Fatalf("expected defaults, got window=%d maxFails=%d", l.windowSec, l.maxFails)
		}
	})

	// === Expected errors ===
	t.Run("error: nil cache returns error", func(t *testing.T) {
		_, err := NewFixedWindowLimiter(nil, "test", 10, 5)
		if err == nil {
			t.Fatal("expected error for nil cache")
		}
	})
}

// ============================================================
// Add
// ============================================================

func TestFixedWindowLimiter_Add(t *testing.T) {
	// === Happy path ===
	t.Run("happy: first request allowed", func(t *testing.T) {
		cache := newMockCache()
		l, err := NewFixedWindowLimiter(cache, "test:first", 60, 5)
		if err != nil {
			t.Fatalf("NewFixedWindowLimiter failed: %v", err)
		}
		allowed, count, ttl, err := l.Add(context.Background())
		if err != nil {
			t.Fatalf("Add error: %v", err)
		}
		if !allowed {
			t.Fatal("expected allowed=true")
		}
		if count != 1 {
			t.Fatalf("expected count=1, got %d", count)
		}
		if ttl <= 0 {
			t.Fatalf("expected positive ttl, got %v", ttl)
		}
	})

	t.Run("happy: multiple requests within limit", func(t *testing.T) {
		cache := newMockCache()
		l, err := NewFixedWindowLimiter(cache, "test:within", 60, 5)
		if err != nil {
			t.Fatalf("NewFixedWindowLimiter failed: %v", err)
		}
		ctx := context.Background()
		for i := int64(1); i <= 5; i++ {
			allowed, count, _, err := l.Add(ctx)
			if err != nil {
				t.Fatalf("Add error at %d: %v", i, err)
			}
			if !allowed {
				t.Fatalf("iteration %d: expected allowed=true, got false", i)
			}
			if count != i {
				t.Fatalf("iteration %d: expected count=%d, got %d", i, i, count)
			}
		}
	})

	// === Boundary values ===
	t.Run("boundary: maxFails=1 allows exactly one request", func(t *testing.T) {
		cache := newMockCache()
		l, err := NewFixedWindowLimiter(cache, "test:one", 60, 1)
		if err != nil {
			t.Fatalf("NewFixedWindowLimiter failed: %v", err)
		}
		ctx := context.Background()

		allowed, _, _, err := l.Add(ctx)
		if err != nil || !allowed {
			t.Fatalf("first request should be allowed: err=%v allowed=%v", err, allowed)
		}

		allowed, count, _, err := l.Add(ctx)
		if err != nil {
			t.Fatalf("second Add error: %v", err)
		}
		if allowed {
			t.Fatal("second request should be blocked")
		}
		if count != 2 {
			t.Fatalf("expected count=2, got %d", count)
		}
	})

	// === Expected errors / blocking ===
	t.Run("error: blocked after exceeding maxFails", func(t *testing.T) {
		cache := newMockCache()
		l, err := NewFixedWindowLimiter(cache, "test:blocked", 60, 3)
		if err != nil {
			t.Fatalf("NewFixedWindowLimiter failed: %v", err)
		}
		ctx := context.Background()

		for i := 0; i < 3; i++ {
			allowed, _, _, err := l.Add(ctx)
			if err != nil || !allowed {
				t.Fatalf("iteration %d: should be allowed, err=%v allowed=%v", i, err, allowed)
			}
		}

		allowed, count, _, err := l.Add(ctx)
		if err != nil {
			t.Fatalf("Add error after limit: %v", err)
		}
		if allowed {
			t.Fatal("expected allowed=false after hitting limit")
		}
		if count != 4 {
			t.Fatalf("expected count=4, got %d", count)
		}
	})
}

// ============================================================
// Reset
// ============================================================

func TestFixedWindowLimiter_Reset(t *testing.T) {
	// === Happy path ===
	t.Run("happy: reset clears counter, allows again", func(t *testing.T) {
		cache := newMockCache()
		l, err := NewFixedWindowLimiter(cache, "test:reset", 60, 1)
		if err != nil {
			t.Fatalf("NewFixedWindowLimiter failed: %v", err)
		}
		ctx := context.Background()

		// Exhaust limit
		allowed, _, _, err := l.Add(ctx)
		if err != nil || !allowed {
			t.Fatalf("first Add should succeed: err=%v allowed=%v", err, allowed)
		}
		allowed, _, _, err = l.Add(ctx)
		if err != nil || allowed {
			t.Fatalf("second Add should be blocked: err=%v allowed=%v", err, allowed)
		}

		// Reset
		err = l.Reset(ctx)
		if err != nil {
			t.Fatalf("Reset error: %v", err)
		}

		// Should be allowed again with count=1
		allowed, count, _, err := l.Add(ctx)
		if err != nil {
			t.Fatalf("Add after reset error: %v", err)
		}
		if !allowed {
			t.Fatal("expected allowed=true after reset")
		}
		if count != 1 {
			t.Fatalf("expected count=1 after reset, got %d", count)
		}
	})

	// === Boundary values ===
	t.Run("boundary: reset on empty key succeeds", func(t *testing.T) {
		cache := newMockCache()
		l, err := NewFixedWindowLimiter(cache, "test:reset-empty", 60, 5)
		if err != nil {
			t.Fatalf("NewFixedWindowLimiter failed: %v", err)
		}
		err = l.Reset(context.Background())
		if err != nil {
			t.Fatalf("Reset on empty key error: %v", err)
		}
	})

	// === Expected errors ===
	t.Run("error: concurrent access", func(t *testing.T) {
		cache := newMockCache()
		l, err := NewFixedWindowLimiter(cache, "test:concurrent", 60, 100)
		if err != nil {
			t.Fatalf("NewFixedWindowLimiter failed: %v", err)
		}

		ctx := context.Background()
		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func() {
				_, _, _, err := l.Add(ctx)
				done <- (err == nil)
			}()
		}

		for i := 0; i < 10; i++ {
			if !<-done {
				t.Fatal("concurrent Add returned error")
			}
		}
	})
}
