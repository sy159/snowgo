package xlock

import (
	"context"
	"fmt"
	v8 "github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v8"
	"github.com/pkg/errors"
	"time"
)

type RedisLock struct {
	rl *redsync.Redsync
}

func NewRedisLock(client *v8.Client) Lock {
	return &RedisLock{
		rl: redsync.New(goredis.NewPool(client)),
	}
}

func (r *RedisLock) LockWithTries(ctx context.Context, key string, expireSecond int64, tries int, fn func(isLock bool, lc LockContext) error) (err error) {
	tries += 1 // 重试次数需要+1
	if fn == nil {
		return errors.New("fn is empty")
	}
	if expireSecond < 1 {
		expireSecond = 1
	}
	if tries < 1 {
		tries = 1
	}
	expiryOption := redsync.WithExpiry(time.Duration(expireSecond) * time.Second)
	triesOption := redsync.WithTries(tries)
	retryDelayOpt := redsync.WithRetryDelay(time.Millisecond * 100) // 重试间隔时间
	mutex := r.rl.NewMutex(key, expiryOption, triesOption, retryDelayOpt)
	if err = mutex.LockContext(ctx); err != nil {
		return fn(false, newRedisLockContext(mutex, err))
	}
	defer func() {
		if !errors.Is(err, redsync.ErrFailed) {
			_, _ = mutex.UnlockContext(ctx)
		}
	}()
	return fn(true, newRedisLockContext(mutex, nil))
}

func (r *RedisLock) LockWithTriesTime(ctx context.Context, key string, expireSecond int64, triesTimeSecond int64, fn func(isLock bool, lc LockContext) error) (err error) {
	if fn == nil {
		return errors.New("fn is empty")
	}
	if expireSecond < 1 {
		expireSecond = 1
	}
	if triesTimeSecond < 1 {
		triesTimeSecond = 1
	}

	expiryOption := redsync.WithExpiry(time.Duration(expireSecond) * time.Second)
	triesOption := redsync.WithTries(int(triesTimeSecond) * 202)      // 重试次数
	retryDelayOption := redsync.WithRetryDelay(time.Millisecond * 50) // 重试间隔时间
	// 重试次数 * 间隔时间 > 锁重试时间，通过超时上下文管理锁重试时间
	mutex := r.rl.NewMutex(key, expiryOption, triesOption, retryDelayOption)
	ctx, cancel := context.WithTimeout(ctx, time.Duration(triesTimeSecond)*time.Second)
	defer cancel()

	if err = mutex.LockContext(ctx); err != nil {
		return fn(false, newRedisLockContext(mutex, err))
	}

	defer func() {
		if !errors.Is(err, redsync.ErrFailed) {
			_, _ = mutex.UnlockContext(ctx)
		}
	}()

	return fn(true, newRedisLockContext(mutex, nil))
}

func (r *RedisLock) TryLock(ctx context.Context, key string, expireSecond int64, fn func(isLock bool, lc LockContext) error) (err error) {
	return r.LockWithTries(ctx, key, expireSecond, 0, fn)
}

func (r *RedisLock) ReTryLock(ctx context.Context, key string, expireSecond int64, fn func(lc LockContext) error) (err error) {
	if fn == nil {
		return errors.New("fn is empty")
	}
	if expireSecond < 1 {
		expireSecond = 1
	}
	expiryOption := redsync.WithExpiry(time.Duration(expireSecond) * time.Second)
	mutex := r.rl.NewMutex(key, expiryOption)
	for {
		if err = mutex.LockContext(ctx); err != nil {
			fmt.Printf("redis retry lock is err: %v\n", err)
			continue
		}
		break
	}
	defer func() {
		_, _ = mutex.UnlockContext(ctx)
	}()
	return fn(newRedisLockContext(mutex, nil))
}

type RedisLockContext struct {
	mutex *redsync.Mutex
	err   error
}

func newRedisLockContext(mutex *redsync.Mutex, err error) LockContext {
	return &RedisLockContext{mutex: mutex, err: err}
}

func (lc *RedisLockContext) Unlock(ctx context.Context) (bool, error) {
	_, err := lc.mutex.UnlockContext(ctx)
	if err != nil {
		return false, errors.WithStack(err)
	}
	return true, nil
}

func (lc *RedisLockContext) Extend(ctx context.Context) (bool, error) {
	_, err := lc.mutex.ExtendContext(ctx)
	if err != nil {
		return false, errors.WithStack(err)
	}
	return true, nil
}

func (lc *RedisLockContext) Err() error {
	return lc.err
}
