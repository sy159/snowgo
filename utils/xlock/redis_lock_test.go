package xlock_test

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"snowgo/utils/xlock"
	"testing"
	"time"
)

func TestRedisLock(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
		Password: "",
		DB:       0,
		PoolSize: 5,
	})
	defer rdb.Close()

	lock := xlock.NewRedisLock(rdb)

	t.Run("Test LockWithTries", func(t *testing.T) {
		ctx := context.TODO()
		key := "test-lock-with-tries"
		// Simulate an existing lock
		initialLock := rdb.SetNX(ctx, key, "locked", 500*time.Millisecond).Val()
		if !initialLock {
			t.Fatalf("Failed to simulate existing lock")
		}

		err := lock.LockWithTries(ctx, key, 5, 6, func(isLock bool, lc xlock.LockContext) error {
			fmt.Printf("Test LockWithTries is lock: %t, err is: %v\n", isLock, lc.Err())
			return nil
		})
		fmt.Printf("Test LockWithTries err is: %v\n", err)
	})

	t.Run("Test LockWithTriesTime", func(t *testing.T) {
		ctx := context.TODO()
		key := "test-lock-with-tries-time"

		// Simulate an existing lock
		initialLock := rdb.SetNX(ctx, key, "locked", 1900*time.Millisecond).Val()
		if !initialLock {
			t.Fatalf("Failed to simulate existing lock")
		}

		err := lock.LockWithTriesTime(ctx, key, 5, 3, func(isLock bool, lc xlock.LockContext) error {
			fmt.Printf("Test LockWithTriesTime is lock: %t, err is: %v\n", isLock, lc.Err())
			return nil
		})
		fmt.Printf("Test LockWithTriesTime err is: %v\n", err)
	})

	t.Run("Test TryLock", func(t *testing.T) {
		ctx := context.TODO()
		key := "test-try-lock"
		err := lock.TryLock(ctx, key, 5, func(isLock bool, lc xlock.LockContext) error {
			fmt.Printf("Test TryLock is lock: %t, err is: %v\n", isLock, lc.Err())
			return nil
		})
		fmt.Printf("Test TryLock err is: %v\n", err)
	})

	t.Run("Test TryLock", func(t *testing.T) {
		ctx := context.TODO()
		key := "test-try-lock"

		// Simulate an existing lock
		initialLock := rdb.SetNX(ctx, key, "locked", 1*time.Second).Val()
		if !initialLock {
			t.Fatalf("Failed to simulate existing lock")
		}

		err := lock.TryLock(ctx, key, 5, func(isLock bool, lc xlock.LockContext) error {
			fmt.Printf("Test TryLock is lock: %t, err is: %v\n", isLock, lc.Err())
			return nil
		})
		fmt.Printf("Test TryLock err is: %v\n", err)
	})

	t.Run("Test ReTryLock", func(t *testing.T) {
		ctx := context.TODO()
		key := "test-retry-lock"

		// Simulate an existing lock
		initialLock := rdb.SetNX(ctx, key, "locked", 10*time.Second).Val()
		if !initialLock {
			t.Fatalf("Failed to simulate existing lock")
		}

		err := lock.ReTryLock(ctx, key, 5, func(lc xlock.LockContext) error {
			fmt.Printf("Test ReTryLock err is: %v\n", lc.Err())
			return nil
		})
		fmt.Printf("Test ReTryLock err is: %v\n", err)
	})

	t.Run("Test Extend", func(t *testing.T) {
		ctx := context.TODO()
		key := "test-unlock"

		// Simulate an existing lock
		initialLock := rdb.SetNX(ctx, key, "locked", 1*time.Second).Val()
		if !initialLock {
			t.Fatalf("Failed to simulate existing lock")
		}

		startTime := time.Now()
		err := lock.LockWithTriesTime(ctx, key, 5, 2, func(isLock bool, lc xlock.LockContext) error {
			fmt.Printf("start func time: %fs\n", time.Since(startTime).Seconds())
			time.Sleep(2 * time.Second)
			fmt.Printf("exec cost time %fs\n", time.Since(startTime).Seconds())
			success, err := lc.Extend(ctx)
			fmt.Printf("Test Extend is: %t, err is: %v\n", success, err)
			time.Sleep(4 * time.Second)
			fmt.Printf("end func time %fs\n", time.Since(startTime).Seconds())
			return nil
		})
		fmt.Printf("Test Extend err is: %v\n", err)
	})
}
