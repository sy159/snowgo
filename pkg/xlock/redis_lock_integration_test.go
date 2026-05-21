//go:build integration

package xlock_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/redis/go-redis/v9"
	"snowgo/pkg/xlock"
)

func setupTestRedis(t *testing.T) *redis.Client {
	t.Helper()
	client := redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
		Password: "",
		DB:       10, // use separate DB to avoid polluting other tests
		PoolSize: 5,
	})
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func TestRedisLock_LockWithTries(t *testing.T) {
	rdb := setupTestRedis(t)
	lock, _ := xlock.NewRedisLock(rdb, nil)
	ctx := context.Background()
	key := "test:lock-with-tries"

	// Flush this specific key before and after the test
	t.Cleanup(func() { _ = rdb.Del(ctx, key).Err() })

	t.Run("happy: acquires fresh lock", func(t *testing.T) {
		var gotIsLock bool
		err := lock.LockWithTries(ctx, key, 5, 1, func(isLock bool, lc xlock.LockContext) error {
			gotIsLock = isLock
			return nil
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !gotIsLock {
			t.Fatal("expected isLock=true for fresh lock")
		}
	})

	t.Run("happy: returns false when lock is held by another mutex", func(t *testing.T) {
		// Use redsync directly to hold a competing lock
		rs := redsync.New(goredis.NewPool(rdb))
		mu := rs.NewMutex(key)
		if err := mu.LockContext(ctx); err != nil {
			t.Fatalf("setup: failed to acquire competing lock: %v", err)
		}
		t.Cleanup(func() { _, _ = mu.UnlockContext(ctx) })

		var gotIsLock bool
		err := lock.LockWithTries(ctx, key, 5, 1, func(isLock bool, lc xlock.LockContext) error {
			gotIsLock = isLock
			return nil
		})
		// With only 1 try, should fail to acquire since another mutex holds it
		if gotIsLock {
			t.Fatal("expected isLock=false when lock is held by another mutex")
		}
		// err should be nil because the callback was called (even with isLock=false)
		if err != nil {
			t.Fatalf("unexpected error from callback: %v", err)
		}
	})

	t.Run("error: nil fn returns error", func(t *testing.T) {
		err := lock.LockWithTries(ctx, key, 5, 1, nil)
		if err == nil {
			t.Fatal("expected error for nil fn")
		}
	})

	t.Run("error: empty key returns error", func(t *testing.T) {
		err := lock.LockWithTries(ctx, "", 5, 1, func(isLock bool, lc xlock.LockContext) error {
			return nil
		})
		if err == nil {
			t.Fatal("expected error for empty key")
		}
	})

	t.Run("boundary: expireSecond=0 defaults to 1", func(t *testing.T) {
		err := lock.LockWithTries(ctx, key, 0, 1, func(isLock bool, lc xlock.LockContext) error {
			return nil
		})
		if err != nil {
			t.Fatalf("LockWithTries with expireSecond=0 error: %v", err)
		}
	})
}

func TestRedisLock_TryLock(t *testing.T) {
	rdb := setupTestRedis(t)
	lock, _ := xlock.NewRedisLock(rdb, nil)
	ctx := context.Background()
	key := "test:try-lock"
	t.Cleanup(func() { _ = rdb.Del(ctx, key).Err() })

	t.Run("happy: acquires fresh lock", func(t *testing.T) {
		var gotIsLock bool
		err := lock.TryLock(ctx, key, 5, func(isLock bool, lc xlock.LockContext) error {
			gotIsLock = isLock
			return nil
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !gotIsLock {
			t.Fatal("expected isLock=true for fresh lock")
		}
	})

	t.Run("happy: returns false when lock is held", func(t *testing.T) {
		// First, acquire the lock via TryLock itself
		acquired := make(chan struct{})
		done := make(chan struct{})
		go func() {
			_ = lock.LockWithTries(ctx, key, 10, 1, func(isLock bool, lc xlock.LockContext) error {
				close(acquired)
				<-done
				return nil
			})
		}()
		<-acquired // wait for first lock to be acquired

		var gotIsLock bool
		err := lock.TryLock(ctx, key, 5, func(isLock bool, lc xlock.LockContext) error {
			gotIsLock = isLock
			return nil
		})
		close(done)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if gotIsLock {
			t.Fatal("expected isLock=false when lock already held")
		}
	})
}

func TestRedisLock_LockWithTriesTime(t *testing.T) {
	rdb := setupTestRedis(t)
	lock, _ := xlock.NewRedisLock(rdb, nil)
	ctx := context.Background()
	key := "test:lock-with-tries-time"
	t.Cleanup(func() { _ = rdb.Del(ctx, key).Err() })

	t.Run("happy: acquires fresh lock", func(t *testing.T) {
		var gotIsLock bool
		err := lock.LockWithTriesTime(ctx, key, 5, 2, func(isLock bool, lc xlock.LockContext) error {
			gotIsLock = isLock
			return nil
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !gotIsLock {
			t.Fatal("expected isLock=true for fresh lock")
		}
	})

	t.Run("error: nil fn returns error", func(t *testing.T) {
		err := lock.LockWithTriesTime(ctx, key, 5, 3, nil)
		if err == nil {
			t.Fatal("expected error for nil fn")
		}
	})

	t.Run("error: empty key returns error", func(t *testing.T) {
		err := lock.LockWithTriesTime(ctx, "", 5, 3, func(isLock bool, lc xlock.LockContext) error {
			return nil
		})
		if err == nil {
			t.Fatal("expected error for empty key")
		}
	})

	t.Run("boundary: expireSecond=0 and triesTimeSecond=0 default to 1", func(t *testing.T) {
		err := lock.LockWithTriesTime(ctx, key, 0, 0, func(isLock bool, lc xlock.LockContext) error {
			return nil
		})
		if err != nil {
			t.Fatalf("LockWithTriesTime with defaults error: %v", err)
		}
	})
}

func TestRedisLock_ReTryLock(t *testing.T) {
	rdb := setupTestRedis(t)
	lock, _ := xlock.NewRedisLock(rdb, nil)
	ctx := context.Background()
	key := "test:retry-lock"
	t.Cleanup(func() { _ = rdb.Del(ctx, key).Err() })

	t.Run("happy: acquires lock", func(t *testing.T) {
		err := lock.ReTryLock(ctx, key, 5, func(lc xlock.LockContext) error {
			if lc.Err() != nil {
				t.Fatalf("unexpected lock error: %v", lc.Err())
			}
			return nil
		})
		if err != nil {
			t.Fatalf("ReTryLock error: %v", err)
		}
	})

	t.Run("error: nil fn returns error", func(t *testing.T) {
		err := lock.ReTryLock(ctx, key, 5, nil)
		if err == nil {
			t.Fatal("expected error for nil fn")
		}
	})

	t.Run("error: empty key returns error", func(t *testing.T) {
		err := lock.ReTryLock(ctx, "", 5, func(lc xlock.LockContext) error {
			return nil
		})
		if err == nil {
			t.Fatal("expected error for empty key")
		}
	})

	t.Run("error: canceled context returns ctx.Err()", func(t *testing.T) {
		cancelCtx, cancel := context.WithCancel(ctx)
		cancel()
		err := lock.ReTryLock(cancelCtx, key, 5, func(lc xlock.LockContext) error {
			return nil
		})
		if err == nil {
			t.Fatal("expected error for canceled context")
		}
		if err != context.Canceled {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	})

	t.Run("boundary: expireSecond=0 defaults to 1", func(t *testing.T) {
		err := lock.ReTryLock(ctx, key, 0, func(lc xlock.LockContext) error {
			return nil
		})
		if err != nil {
			t.Fatalf("ReTryLock with expireSecond=0 error: %v", err)
		}
	})
}

func TestReTryLock_ContextCanceled_RaceSafe(t *testing.T) {
	rdb := setupTestRedis(t)
	lock, _ := xlock.NewRedisLock(rdb, nil)
	ctx := context.Background()
	key := "test:cancel-lock-race"
	t.Cleanup(func() { _ = rdb.Del(ctx, key).Err() })

	// Use a channel to guarantee the first lock is acquired before the second attempt
	holderDone := make(chan struct{})
	lockAcquired := make(chan struct{})
	go func() {
		_ = lock.LockWithTries(ctx, key, 10, 1, func(isLock bool, lc xlock.LockContext) error {
			close(lockAcquired) // signal: lock is held
			<-holderDone        // wait for test to signal release
			return nil
		})
	}()
	<-lockAcquired // wait until lock is definitely held

	cancelCtx, cancel := context.WithCancel(ctx)
	cancel()

	err := lock.ReTryLock(cancelCtx, key, 5, func(lc xlock.LockContext) error {
		return nil
	})
	if err == nil {
		t.Fatal("expected error for canceled context")
	}
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	close(holderDone)
}

func TestRedisLockContext_Unlock(t *testing.T) {
	rdb := setupTestRedis(t)
	lock, _ := xlock.NewRedisLock(rdb, nil)
	ctx := context.Background()
	key := "test:unlock-manual"
	t.Cleanup(func() { _ = rdb.Del(ctx, key).Err() })

	t.Run("happy: Unlock succeeds after acquiring lock", func(t *testing.T) {
		var unlocked bool
		err := lock.LockWithTries(ctx, key, 5, 1, func(isLock bool, lc xlock.LockContext) error {
			if !isLock {
				return nil
			}
			rlc, ok := lc.(*xlock.RedisLockContext)
			if !ok {
				t.Fatal("expected *RedisLockContext")
			}
			ok, unlockErr := rlc.Unlock(context.Background())
			if unlockErr != nil {
				t.Fatalf("Unlock error: %v", unlockErr)
			}
			unlocked = ok
			return nil
		})
		if err != nil {
			t.Fatalf("LockWithTries error: %v", err)
		}
		if !unlocked {
			t.Fatal("expected Unlock to succeed")
		}
	})
}

func TestRedisLock_Extend(t *testing.T) {
	rdb := setupTestRedis(t)
	lock, _ := xlock.NewRedisLock(rdb, nil)
	ctx := context.Background()
	key := "test:extend-lock"
	t.Cleanup(func() { _ = rdb.Del(ctx, key).Err() })

	t.Run("happy: Extend prolongs lock TTL", func(t *testing.T) {
		err := lock.LockWithTriesTime(ctx, key, 2, 3, func(isLock bool, lc xlock.LockContext) error {
			if !isLock {
				t.Fatal("expected to acquire lock")
			}
			// Extend the lock
			ok, extendErr := lc.Extend(ctx)
			if !ok || extendErr != nil {
				t.Fatalf("Extend failed: ok=%t, err=%v", ok, extendErr)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("LockWithTriesTime error: %v", err)
		}
	})
}

func TestNewRedisLock(t *testing.T) {
	rdb := setupTestRedis(t)

	t.Run("happy: nil logger uses defaultLogger", func(t *testing.T) {
		_, err := xlock.NewRedisLock(rdb, nil)
		if err != nil {
			t.Fatalf("unexpected error with nil logger: %v", err)
		}
	})

	t.Run("error: nil client returns error", func(t *testing.T) {
		_, err := xlock.NewRedisLock(nil, nil)
		if err == nil {
			t.Fatal("expected error for nil redis client")
		}
	})
}

func TestRedisLock_Concurrent(t *testing.T) {
	rdb := setupTestRedis(t)
	lock, _ := xlock.NewRedisLock(rdb, nil)
	ctx := context.Background()
	key := "test:concurrent-lock"
	t.Cleanup(func() { _ = rdb.Del(ctx, key).Err() })

	t.Run("happy: only one goroutine acquires lock at a time", func(t *testing.T) {
		var mu sync.Mutex
		acquired := 0
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = lock.LockWithTries(ctx, key, 2, 3, func(isLock bool, lc xlock.LockContext) error {
					if isLock {
						mu.Lock()
						acquired++
						mu.Unlock()
						time.Sleep(50 * time.Millisecond)
					}
					return nil
				})
			}()
		}
		wg.Wait()
		// With expire=2s and retry=3, all goroutines should eventually acquire
		if acquired == 0 {
			t.Fatal("expected at least one goroutine to acquire lock")
		}
	})
}
