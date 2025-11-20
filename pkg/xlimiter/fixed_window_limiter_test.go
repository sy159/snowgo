package xlimiter_test

import (
	"context"
	"fmt"
	"snowgo/pkg/xcache"
	"snowgo/pkg/xlimiter"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
)

func setupTestRedis() *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
		Password: "",
		DB:       0,
		PoolSize: 5,
	})
	return client
}

func teardownTestRedis(client *redis.Client) {
	//client.FlushDB(context.Background())
	_ = client.Close()
}

func TestFixedWindowLimiter(t *testing.T) {
	client := setupTestRedis()
	defer teardownTestRedis(client)
	cache := xcache.NewRedisCache(client)
	ctx := context.Background()

	key := "login:fail:testuser"
	windowSec := int64(3) // 测试用短窗口：3秒
	maxFails := int64(3)

	limiter := xlimiter.NewFixedWindowLimiter(cache, key, windowSec, maxFails)

	// 重置确保干净
	_ = limiter.Reset(ctx)

	t.Run("Add under limit", func(t *testing.T) {
		for i := int64(1); i <= maxFails-1; i++ {
			allowed, count, ttl, err := limiter.Add(ctx)
			if err != nil {
				t.Fatalf("Add failed: %v", err)
			}
			if !allowed {
				t.Fatalf("Add unexpectedly blocked at attempt %d", i)
			}
			if count != i {
				t.Fatalf("Add count mismatch: got %d want %d", count, i)
			}
			if ttl <= 0 || ttl > time.Duration(windowSec)*time.Second {
				t.Fatalf("TTL out of range: %v", ttl)
			}
			fmt.Printf("Attempt %d: allowed=%v, count=%d, ttl=%v\n", i, allowed, count, ttl)
		}
	})

	t.Run("Add hitting limit", func(t *testing.T) {
		// 第 maxFails 次 Add，应返回 blocked
		allowed, count, ttl, err := limiter.Add(ctx)
		if err != nil {
			t.Fatalf("Add failed: %v", err)
		}
		if allowed {
			t.Fatalf("Add should be blocked at maxFails, got allowed=true")
		}
		if count != maxFails {
			t.Fatalf("Add count mismatch at limit: got %d want %d", count, maxFails)
		}
		fmt.Printf("Hit limit: allowed=%v, count=%d, ttl=%v\n", allowed, count, ttl)
	})

	t.Run("Add after window expires", func(t *testing.T) {
		time.Sleep(time.Duration(windowSec+1) * time.Second)

		allowed, count, ttl, err := limiter.Add(ctx)
		if err != nil {
			t.Fatalf("Add failed after window: %v", err)
		}
		if !allowed {
			t.Fatalf("Add should be allowed after window reset")
		}
		if count != 1 {
			t.Fatalf("Count should reset to 1 after window expired, got %d", count)
		}
		fmt.Printf("After window: allowed=%v, count=%d, ttl=%v\n", allowed, count, ttl)
	})

	t.Run("Reset limiter", func(t *testing.T) {
		err := limiter.Reset(ctx)
		if err != nil {
			t.Fatalf("Reset failed: %v", err)
		}
		allowed, count, ttl, err := limiter.Add(ctx)
		if err != nil {
			t.Fatalf("Add after reset failed: %v", err)
		}
		if !allowed || count != 1 {
			t.Fatalf("Add after reset should be allowed with count=1, got allowed=%v, count=%d", allowed, count)
		}
		fmt.Printf("After reset: allowed=%v, count=%d, ttl=%v\n", allowed, count, ttl)
	})
}
