//go:build integration

package xlimiter

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"snowgo/pkg/xcache"
)

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
	cache, _ := xcache.NewRedisCache(client)
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
		cache, _ := xcache.NewRedisCache(client)
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
		cache, _ := xcache.NewRedisCache(client)
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
		cache, _ := xcache.NewRedisCache(client)
		limiter, err := NewFixedWindowLimiter(cache, "test:negative", -5, -3)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if limiter.windowSec != 60 || limiter.maxFails != 1 {
			t.Fatalf("expected defaults for negative values, got window=%d, maxFails=%d", limiter.windowSec, limiter.maxFails)
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
