package xlimiter

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
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
