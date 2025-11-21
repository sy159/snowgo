package xlock

import (
	"context"
	v8 "github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v8"
	"github.com/pkg/errors"
	"math/rand"
	"time"
)

type RedisLock struct {
	rl     *redsync.Redsync
	logger Logger
}

func NewRedisLock(client *v8.Client, logger Logger) Lock {
	if logger == nil {
		logger = &defaultLogger{} // fmt.Printf 封装
	}
	return &RedisLock{
		rl:     redsync.New(goredis.NewPool(client)),
		logger: logger,
	}
}

func (r *RedisLock) LockWithTries(ctx context.Context, key string, expireSecond int64, tries int, fn func(isLock bool, lc LockContext) error) (err error) {
	tries += 1 // 重试次数需要+1
	if fn == nil {
		return errors.New("fn is empty")
	}
	if key == "" {
		return errors.New("key is empty")
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
	r.logger.Infof("[redis lock] lock acquired key=%s expire=%ds tries=%d", key, expireSecond, tries)
	// 释放锁
	defer func() {
		// 解锁使用 background context，避免外部 ctx被 cancel导致Unlock失败
		ok, unLockErr := mutex.UnlockContext(context.Background())
		if unLockErr != nil || !ok {
			r.logger.Errorf("[redis lock] unlock failed key=%s ok=%v err=%v\n", key, ok, unLockErr)
		} else {
			r.logger.Infof("[redis lock] lock released key=%s", key)
		}
	}()
	return fn(true, newRedisLockContext(mutex, nil))
}

func (r *RedisLock) LockWithTriesTime(ctx context.Context, key string, expireSecond int64, triesTimeSecond int64, fn func(isLock bool, lc LockContext) error) (err error) {
	if fn == nil {
		return errors.New("fn is empty")
	}
	if key == "" {
		return errors.New("key is empty")
	}
	if expireSecond < 1 {
		expireSecond = 1
	}
	if triesTimeSecond < 1 {
		triesTimeSecond = 1
	}

	// 间隔时间50ms，一秒就可以重试20次, 要保证大于锁重试时间，所以+1
	retryDelay := 50 * time.Millisecond
	tries := int(triesTimeSecond)*20 + 1

	expiryOption := redsync.WithExpiry(time.Duration(expireSecond) * time.Second)
	triesOption := redsync.WithTries(tries)                // 重试次数
	retryDelayOption := redsync.WithRetryDelay(retryDelay) // 重试间隔时间
	// 重试次数 * 间隔时间 > 锁重试时间，通过超时上下文管理锁重试时间
	mutex := r.rl.NewMutex(key, expiryOption, triesOption, retryDelayOption)
	ctx, cancel := context.WithTimeout(ctx, time.Duration(triesTimeSecond)*time.Second)
	defer cancel()

	if err = mutex.LockContext(ctx); err != nil {
		return fn(false, newRedisLockContext(mutex, err))
	}
	r.logger.Infof("[redis lock] lock acquired key=%s expire=%ds triesTime=%ds", key, expireSecond, triesTimeSecond)
	// 释放锁
	defer func() {
		// 解锁使用 background context，避免外部 ctx被 cancel导致Unlock失败
		ok, unLockErr := mutex.UnlockContext(context.Background())
		if unLockErr != nil || !ok {
			r.logger.Errorf("[redis lock] unlock failed key=%s ok=%v err=%v\n", key, ok, unLockErr)
		} else {
			r.logger.Infof("[redis lock] lock released key=%s", key)
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
	if key == "" {
		return errors.New("key is empty")
	}
	if expireSecond < 1 {
		expireSecond = 1
	}
	expiryOption := redsync.WithExpiry(time.Duration(expireSecond) * time.Second)
	mutex := r.rl.NewMutex(key, expiryOption)
	attempt := 0
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err = mutex.LockContext(ctx); err != nil {
			r.logger.Errorf("[redis lock] redis retry lock failed key=%s attempt=%d err=%v", key, attempt, err)
			time.Sleep(time.Millisecond*50 + time.Duration(rand.Intn(20))*time.Millisecond) // nolint:gosec
			attempt++
			continue
		}
		break
	}
	r.logger.Infof("[redis lock] lock acquired key=%s expire=%ds", key, expireSecond)
	// 释放锁
	defer func() {
		// 解锁使用 background context，避免外部 ctx被 cancel导致Unlock失败
		ok, unLockErr := mutex.UnlockContext(context.Background())
		if unLockErr != nil || !ok {
			r.logger.Errorf("[redis lock] unlock failed key=%s ok=%v err=%v\n", key, ok, unLockErr)
		} else {
			r.logger.Infof("[redis lock] lock released key=%s", key)
		}
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
	ok, err := lc.mutex.UnlockContext(ctx)
	if err != nil || !ok {
		return false, errors.WithStack(err)
	}
	return true, nil
}

func (lc *RedisLockContext) Extend(ctx context.Context) (bool, error) {
	ok, err := lc.mutex.ExtendContext(ctx)
	if err != nil || !ok {
		return false, errors.WithStack(err)
	}
	return true, nil
}

func (lc *RedisLockContext) Err() error {
	return lc.err
}
