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

func TestTokenBucket_New_Boundary(t *testing.T) {
	// === Boundary values ===
	t.Run("boundary: burst=0 panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic for burst=0")
			}
		}()
		NewTokenBucket("test:zero-burst", 5, 0)
	})

	t.Run("boundary: burst=-1 panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic for burst=-1")
			}
		}()
		NewTokenBucket("test:neg-burst", 5, -1)
	})

	t.Run("boundary: limit=0 (no token refill)", func(t *testing.T) {
		defer func() { limiterMap.Range(func(k, v any) bool { limiterMap.Delete(k); return true }) }()
		tb, _ := NewTokenBucket("test:zero-limit", 0, 3)
		// burst=3 means first 3 calls should succeed, then fail (no refill with limit=0)
		for i := 0; i < 3; i++ {
			if !tb.Allow() {
				t.Fatalf("Allow %d should succeed (burst=3)", i+1)
			}
		}
		if tb.Allow() {
			t.Fatal("Allow should fail after burst exhausted (limit=0, no refill)")
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
	elapsed := time.Since(start)

	// Use a more tolerant assertion: rate=1/sec means ~1s wait, allow 50% margin
	if elapsed < 500*time.Millisecond {
		t.Fatalf("Wait should block for ~1s, returned in %v", elapsed)
	}
}

func TestTokenBucket_Wait_Canceled(t *testing.T) {
	defer func() { limiterMap.Range(func(k, v any) bool { limiterMap.Delete(k); return true }) }()

	// === Expected errors ===
	t.Run("error: already-canceled context", func(t *testing.T) {
		tb, _ := NewTokenBucket("test:canceled", 1, 1)
		_ = tb.Allow() // consume the burst
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // cancel immediately
		err := tb.Wait(ctx)
		if err == nil {
			t.Fatal("Wait with canceled context should return error")
		}
		if err != context.Canceled {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	})

	t.Run("error: deadline exceeded", func(t *testing.T) {
		tb, _ := NewTokenBucket("test:deadline", 0, 1) // limit=0 means no refill
		_ = tb.Allow()                                  // consume the burst
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		err := tb.Wait(ctx)
		if err == nil {
			t.Fatal("Wait with short deadline should return error")
		}
		// rate.Wait wraps context deadline with its own message
		if err.Error() == "" {
			t.Fatal("expected non-empty error message")
		}
	})
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
	time.Sleep(600 * time.Millisecond) // rate=10/s * 0.6s = 6 tokens expected, allow 1 margin

	success := 0
	for i := 0; i < 5; i++ {
		if tb.Allow() {
			success++
		}
	}
	if success < 4 {
		t.Fatalf("expected at least 4 tokens after refill, got %d", success)
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
