package xlimiter

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"snowgo/pkg/xcache"
)

// ========================
// Token Bucket Limiter Tests
// ========================

func TestTokenBucket_New(t *testing.T) {
	// Clean up any leftover from previous runs
	defer func() { limiterMap.Range(func(k, v any) bool { limiterMap.Delete(k); return true }) }()

	t.Run("first creation returns true", func(t *testing.T) {
		tb, first := NewTokenBucket("test:first", 5, 10)
		if !first {
			t.Fatal("expected first=true")
		}
		if tb == nil {
			t.Fatal("expected non-nil limiter")
		}
	})

	t.Run("same key returns existing", func(t *testing.T) {
		tb1, first := NewTokenBucket("test:same", 5, 10)
		tb2, second := NewTokenBucket("test:same", 5, 10)
		if !first || second {
			t.Fatalf("expected first=true, second=false")
		}
		if tb1 != tb2 {
			t.Fatal("expected same limiter instance for same key")
		}
	})
}

func TestTokenBucket_Allow(t *testing.T) {
	defer func() { limiterMap.Range(func(k, v any) bool { limiterMap.Delete(k); return true }) }()

	t.Run("burst allows initial requests", func(t *testing.T) {
		tb, _ := NewTokenBucket("test:allow", 10, 2)
		if !tb.Allow() {
			t.Fatal("first Allow() call should succeed")
		}
		if !tb.Allow() {
			t.Fatal("second Allow() call should succeed")
		}
		if tb.Allow() {
			t.Fatal("third Allow() should fail (burst exhausted)")
		}
	})

	t.Run("every rate limit", func(t *testing.T) {
		limit := BucketEvery(100 * time.Millisecond)
		tb, _ := NewTokenBucket("test:every", limit, 1)

		if !tb.Allow() {
			t.Fatal("first Allow should succeed due to burst")
		}
		if tb.Allow() {
			t.Fatal("second Allow should fail (token not yet generated)")
		}

		time.Sleep(110 * time.Millisecond)
		if !tb.Allow() {
			t.Fatal("Allow should succeed after waiting for token")
		}
	})
}

func TestTokenBucket_Wait(t *testing.T) {
	defer func() { limiterMap.Range(func(k, v any) bool { limiterMap.Delete(k); return true }) }()

	tb, _ := NewTokenBucket("test:wait", 1, 1)
	_ = tb.Wait(context.Background())
	start := time.Now()
	_ = tb.Wait(context.Background())

	if time.Since(start) < 900*time.Millisecond {
		t.Fatalf("Wait should block for ~1s, returned in %v", time.Since(start))
	}
}

func TestTokenBucket_Reserve(t *testing.T) {
	defer func() { limiterMap.Range(func(k, v any) bool { limiterMap.Delete(k); return true }) }()

	tb, _ := NewTokenBucket("test:reserve", 1, 1)
	res := tb.Reserve()
	if !res.OK() {
		t.Fatal("expected Reserve to succeed")
	}
	if res.Delay() < 0 {
		t.Fatal("expected non-negative delay")
	}
}

func TestTokenBucket_SetLimitAndBurst(t *testing.T) {
	defer func() { limiterMap.Range(func(k, v any) bool { limiterMap.Delete(k); return true }) }()

	tb, _ := NewTokenBucket("test:set", 1, 1)
	tb.Allow() // consume the burst

	tb.SetLimit(10)
	tb.SetBurst(5)
	time.Sleep(500 * time.Millisecond)

	success := 0
	for i := 0; i < 5; i++ {
		if tb.Allow() {
			success++
		}
	}
	if success != 5 {
		t.Fatalf("expected 5 tokens after refill, got %d", success)
	}
}

func TestTokenBucket_Close(t *testing.T) {
	tb, _ := NewTokenBucket("test:close", 1, 1)
	tb.Close()

	if _, ok := limiterMap.Load(tb.key); ok {
		t.Fatal("expected limiter to be removed after Close()")
	}
}

func TestTokenBucket_ConcurrentAccess(t *testing.T) {
	defer func() { limiterMap.Range(func(k, v any) bool { limiterMap.Delete(k); return true }) }()

	tb, _ := NewTokenBucket("test:concurrent", 100, 100)

	var wg sync.WaitGroup
	successCount := int32(0)
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if tb.Allow() {
				atomic.AddInt32(&successCount, 1)
			}
		}()
	}
	wg.Wait()

	if successCount == 0 {
		t.Fatal("expected at least some Allow() calls to succeed concurrently")
	}
}

// ========================
// Fixed Window Limiter Tests
// ========================

func setupTestRedis(t *testing.T) *redis.Client {
	t.Helper()
	client := redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
		Password: "",
		DB:       0,
		PoolSize: 5,
	})
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func TestFixedWindowLimiter_Add(t *testing.T) {
	client := setupTestRedis(t)
	cache := xcache.NewRedisCache(client)
	ctx := context.Background()

	key := "test:fw:add"
	windowSec := int64(3)
	maxFails := int64(3)

	limiter, err := NewFixedWindowLimiter(cache, key, windowSec, maxFails)
	if err != nil {
		t.Fatalf("NewFixedWindowLimiter failed: %v", err)
	}
	_ = limiter.Reset(ctx)

	t.Run("under limit", func(t *testing.T) {
		for i := int64(1); i <= maxFails; i++ {
			allowed, count, ttl, err := limiter.Add(ctx)
			if err != nil {
				t.Fatalf("Add failed: %v", err)
			}
			if !allowed {
				t.Fatalf("Add unexpectedly blocked at attempt %d", i)
			}
			if count != i {
				t.Fatalf("count mismatch: got %d want %d", count, i)
			}
			if ttl <= 0 || ttl > time.Duration(windowSec)*time.Second {
				t.Fatalf("TTL out of range: %v", ttl)
			}
		}
	})

	t.Run("hit limit", func(t *testing.T) {
		allowed, count, _, err := limiter.Add(ctx)
		if err != nil {
			t.Fatalf("Add failed: %v", err)
		}
		if allowed {
			t.Fatal("Add should be blocked at maxFails+1")
		}
		if count != maxFails+1 {
			t.Fatalf("count mismatch at limit: got %d want %d", count, maxFails+1)
		}
	})

	t.Run("window expires", func(t *testing.T) {
		time.Sleep(time.Duration(windowSec+1) * time.Second)

		allowed, count, _, err := limiter.Add(ctx)
		if err != nil {
			t.Fatalf("Add failed after window: %v", err)
		}
		if !allowed {
			t.Fatal("Add should be allowed after window reset")
		}
		if count != 1 {
			t.Fatalf("count should reset to 1, got %d", count)
		}
	})

	t.Run("reset", func(t *testing.T) {
		err := limiter.Reset(ctx)
		if err != nil {
			t.Fatalf("Reset failed: %v", err)
		}
		allowed, count, _, err := limiter.Add(ctx)
		if err != nil {
			t.Fatalf("Add after reset failed: %v", err)
		}
		if !allowed || count != 1 {
			t.Fatalf("Add after reset should be allowed with count=1, got allowed=%v, count=%d", allowed, count)
		}
	})
}

func TestFixedWindowLimiter_Defaults(t *testing.T) {
	t.Run("nil cache returns error", func(t *testing.T) {
		_, err := NewFixedWindowLimiter(nil, "test:nil-cache", 0, 5)
		if err == nil {
			t.Fatal("expected error for nil cache")
		}
	})

	t.Run("default windowSecond", func(t *testing.T) {
		client := setupTestRedis(t)
		cache := xcache.NewRedisCache(client)
		limiter, err := NewFixedWindowLimiter(cache, "test:default-window", 0, 5)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if limiter.windowSec != 60 {
			t.Fatalf("expected default windowSecond=60, got %d", limiter.windowSec)
		}
	})

	t.Run("default maxFails", func(t *testing.T) {
		client := setupTestRedis(t)
		cache := xcache.NewRedisCache(client)
		limiter, err := NewFixedWindowLimiter(cache, "test:default-fails", 10, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if limiter.maxFails != 1 {
			t.Fatalf("expected default maxFails=1, got %d", limiter.maxFails)
		}
	})

	t.Run("negative values use defaults", func(t *testing.T) {
		client := setupTestRedis(t)
		cache := xcache.NewRedisCache(client)
		limiter, err := NewFixedWindowLimiter(cache, "test:negative", -5, -3)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if limiter.windowSec != 60 || limiter.maxFails != 1 {
			t.Fatalf("expected defaults for negative values, got window=%d, maxFails=%d", limiter.windowSec, limiter.maxFails)
		}
	})
}

func TestParseRedisInt(t *testing.T) {
	t.Run("int64", func(t *testing.T) {
		v, err := parseRedisInt(int64(42))
		if err != nil || v != 42 {
			t.Fatalf("parseRedisInt(int64(42)) = %d, %v", v, err)
		}
	})

	t.Run("string", func(t *testing.T) {
		v, err := parseRedisInt("42")
		if err != nil || v != 42 {
			t.Fatalf("parseRedisInt(\"42\") = %d, %v", v, err)
		}
	})

	t.Run("bytes", func(t *testing.T) {
		v, err := parseRedisInt([]byte("42"))
		if err != nil || v != 42 {
			t.Fatalf("parseRedisInt([]byte(\"42\")) = %d, %v", v, err)
		}
	})

	t.Run("unsupported type", func(t *testing.T) {
		_, err := parseRedisInt(float64(42.0))
		if err == nil {
			t.Fatal("expected error for unsupported type")
		}
	})

	t.Run("negative int64", func(t *testing.T) {
		v, err := parseRedisInt(int64(-2))
		if err != nil || v != -2 {
			t.Fatalf("parseRedisInt(int64(-2)) = %d, %v", v, err)
		}
	})

	t.Run("negative string", func(t *testing.T) {
		v, err := parseRedisInt("-5")
		if err != nil || v != -5 {
			t.Fatalf("parseRedisInt(\"-5\") = %d, %v", v, err)
		}
	})

	t.Run("empty string", func(t *testing.T) {
		_, err := parseRedisInt("")
		if err == nil {
			t.Fatal("expected error for empty string")
		}
	})
}

func TestFixedWindowLimiter_NilCache(t *testing.T) {
	t.Run("NewFixedWindowLimiter returns error", func(t *testing.T) {
		_, err := NewFixedWindowLimiter(nil, "test:nil-cache", 5, 3)
		if err == nil {
			t.Fatal("expected error for nil cache")
		}
	})
}
