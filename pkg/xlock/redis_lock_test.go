package xlock_test

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"snowgo/pkg/xlock"
)

func TestNewRedisLock_NilLogger(t *testing.T) {
	lock := xlock.NewRedisLock(nil, nil)
	if lock == nil {
		t.Fatal("expected non-nil lock with nil logger")
	}
}

func TestRedisLock(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
		Password: "",
		DB:       0,
		PoolSize: 5,
	})
	t.Cleanup(func() { _ = rdb.Close() })

	lock := xlock.NewRedisLock(rdb, nil)

	t.Run("LockWithTries", func(t *testing.T) {
		ctx := context.TODO()
		key := "test-lock-with-tries"
		_ = rdb.SetArgs(ctx, key, "locked", redis.SetArgs{Mode: "NX", TTL: 500 * time.Millisecond}).Val()

		err := lock.LockWithTries(ctx, key, 5, 6, func(isLock bool, lc xlock.LockContext) error {
			t.Logf("isLock=%t, err=%v", isLock, lc.Err())
			return nil
		})
		t.Logf("err=%v", err)
	})

	t.Run("LockWithTriesTime", func(t *testing.T) {
		ctx := context.TODO()
		key := "test-lock-with-tries-time"
		_ = rdb.SetArgs(ctx, key, "locked", redis.SetArgs{Mode: "NX", TTL: 1900 * time.Millisecond}).Val()

		err := lock.LockWithTriesTime(ctx, key, 5, 3, func(isLock bool, lc xlock.LockContext) error {
			t.Logf("isLock=%t, err=%v", isLock, lc.Err())
			return nil
		})
		t.Logf("err=%v", err)
	})

	t.Run("TryLock", func(t *testing.T) {
		ctx := context.TODO()
		key := "test-try-lock-fresh"
		err := lock.TryLock(ctx, key, 5, func(isLock bool, lc xlock.LockContext) error {
			t.Logf("isLock=%t, err=%v", isLock, lc.Err())
			return nil
		})
		t.Logf("err=%v", err)
	})

	t.Run("TryLock_Existing", func(t *testing.T) {
		ctx := context.TODO()
		key := "test-try-lock-existing"
		_ = rdb.SetArgs(ctx, key, "locked", redis.SetArgs{Mode: "NX", TTL: 1 * time.Second}).Val()

		err := lock.TryLock(ctx, key, 5, func(isLock bool, lc xlock.LockContext) error {
			t.Logf("isLock=%t, err=%v", isLock, lc.Err())
			return nil
		})
		t.Logf("err=%v", err)
	})

	t.Run("ReTryLock", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.TODO(), 3*time.Second)
		defer cancel()
		key := "test-retry-lock"
		_ = rdb.SetArgs(ctx, key, "locked", redis.SetArgs{Mode: "NX", TTL: 10 * time.Second}).Val()

		err := lock.ReTryLock(ctx, key, 5, func(lc xlock.LockContext) error {
			t.Logf("err=%v", lc.Err())
			return nil
		})
		if err == nil {
			t.Log("ReTryLock acquired lock")
		} else {
			t.Logf("ReTryLock err=%v", err)
		}
	})

	t.Run("Extend", func(t *testing.T) {
		ctx := context.TODO()
		key := "test-extend-lock"
		_ = rdb.SetArgs(ctx, key, "locked", redis.SetArgs{Mode: "NX", TTL: 1 * time.Second}).Val()

		startTime := time.Now()
		err := lock.LockWithTriesTime(ctx, key, 5, 2, func(isLock bool, lc xlock.LockContext) error {
			t.Logf("extend start: %.2fs", time.Since(startTime).Seconds())
			time.Sleep(2 * time.Second)
			t.Logf("exec cost: %.2fs", time.Since(startTime).Seconds())
			success, err := lc.Extend(ctx)
			t.Logf("Extend=%t, err=%v", success, err)
			time.Sleep(4 * time.Second)
			t.Logf("extend end: %.2fs", time.Since(startTime).Seconds())
			return nil
		})
		t.Logf("err=%v", err)
	})
}

func TestReTryLockValidation(t *testing.T) {
	lock := xlock.NewRedisLock(nil, nil)

	t.Run("fnNil", func(t *testing.T) {
		err := lock.ReTryLock(context.TODO(), "test-key", 5, nil)
		if err == nil {
			t.Fatal("expected error for nil fn")
		}
	})

	t.Run("emptyKey", func(t *testing.T) {
		err := lock.ReTryLock(context.TODO(), "", 5, func(lc xlock.LockContext) error {
			return nil
		})
		if err == nil {
			t.Fatal("expected error for empty key")
		}
	})
}

func TestRedisLockContext_Unlock(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
		Password: "",
		DB:       0,
		PoolSize: 5,
	})
	t.Cleanup(func() { _ = rdb.Close() })

	lock := xlock.NewRedisLock(rdb, nil)

	err := lock.LockWithTries(context.TODO(), "test-unlock-ctx-manual", 5, 1, func(isLock bool, lc xlock.LockContext) error {
		if !isLock {
			return nil
		}
		rlc, ok := lc.(*xlock.RedisLockContext)
		if !ok {
			t.Fatal("expected *RedisLockContext")
		}
		ok, err := rlc.Unlock(context.Background())
		if err != nil {
			t.Logf("Unlock error: %v", err)
		}
		t.Logf("Unlock result: %t", ok)
		return nil
	})
	if err != nil {
		t.Fatalf("LockWithTries error: %v", err)
	}
}

func TestLockWithTries_ExpireSecondDefault(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
		Password: "",
		DB:       0,
		PoolSize: 5,
	})
	t.Cleanup(func() { _ = rdb.Close() })

	lock := xlock.NewRedisLock(rdb, nil)

	ctx := context.TODO()
	err := lock.LockWithTries(ctx, "test-expire-default", 0, 1, func(isLock bool, lc xlock.LockContext) error {
		return nil
	})
	if err != nil {
		t.Fatalf("LockWithTries with expireSecond=0 error: %v", err)
	}
}

func TestReTryLock_ContextCanceled(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
		Password: "",
		DB:       0,
		PoolSize: 5,
	})
	t.Cleanup(func() { _ = rdb.Close() })

	lock := xlock.NewRedisLock(rdb, nil)

	ctx := context.TODO()
	done := make(chan struct{})
	go func() {
		_ = lock.LockWithTries(ctx, "test-cancel-lock", 10, 1, func(isLock bool, lc xlock.LockContext) error {
			<-done
			return nil
		})
	}()

	time.Sleep(200 * time.Millisecond)

	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel()

	err := lock.ReTryLock(cancelCtx, "test-cancel-lock", 5, func(lc xlock.LockContext) error {
		return nil
	})
	if err == nil {
		t.Fatal("expected error for canceled context")
	}

	close(done)
}

func TestLockWithTriesTime_ExpireDefault(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
		Password: "",
		DB:       0,
		PoolSize: 5,
	})
	t.Cleanup(func() { _ = rdb.Close() })

	lock := xlock.NewRedisLock(rdb, nil)

	ctx := context.TODO()
	err := lock.LockWithTriesTime(ctx, "test-time-expire-default", 0, 0, func(isLock bool, lc xlock.LockContext) error {
		return nil
	})
	if err != nil {
		t.Fatalf("LockWithTriesTime with defaults error: %v", err)
	}
}

func TestLockWithTries_ValidationErrors(t *testing.T) {
	lock := xlock.NewRedisLock(nil, nil)

	t.Run("fnNil", func(t *testing.T) {
		err := lock.LockWithTries(context.TODO(), "test-key", 5, 1, nil)
		if err == nil {
			t.Fatal("expected error for nil fn")
		}
	})

	t.Run("emptyKey", func(t *testing.T) {
		err := lock.LockWithTries(context.TODO(), "", 5, 1, func(isLock bool, lc xlock.LockContext) error {
			return nil
		})
		if err == nil {
			t.Fatal("expected error for empty key")
		}
	})
}

func TestLockWithTriesTime_ValidationErrors(t *testing.T) {
	lock := xlock.NewRedisLock(nil, nil)

	t.Run("fnNil", func(t *testing.T) {
		err := lock.LockWithTriesTime(context.TODO(), "test-key", 5, 3, nil)
		if err == nil {
			t.Fatal("expected error for nil fn")
		}
	})

	t.Run("emptyKey", func(t *testing.T) {
		err := lock.LockWithTriesTime(context.TODO(), "", 5, 3, func(isLock bool, lc xlock.LockContext) error {
			return nil
		})
		if err == nil {
			t.Fatal("expected error for empty key")
		}
	})
}

func TestReTryLock_ExpireSecondDefault(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
		Password: "",
		DB:       0,
		PoolSize: 5,
	})
	t.Cleanup(func() { _ = rdb.Close() })

	lock := xlock.NewRedisLock(rdb, nil)

	ctx := context.TODO()
	err := lock.ReTryLock(ctx, "test-expire-default", 0, func(lc xlock.LockContext) error {
		return nil
	})
	if err != nil {
		t.Fatalf("ReTryLock with expireSecond=0 error: %v", err)
	}
}
