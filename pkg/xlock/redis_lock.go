package xlock

import (
	"context"
	"errors"
	"time"

	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/redis/go-redis/v9"
	common "snowgo/pkg"
)

type RedisLock struct {
	rl     *redsync.Redsync
	logger Logger
}

func NewRedisLock(client *redis.Client, logger Logger) (Lock, error) {
	if client == nil {
		return nil, errors.New("redis client cannot be nil")
	}
	if logger == nil {
		logger = &defaultLogger{}
	}
	return &RedisLock{
		rl:     redsync.New(goredis.NewPool(client)),
		logger: logger,
	}, nil
}

// unlockWithTimeout 安全释放锁
func (r *RedisLock) unlockWithTimeout(mutex *redsync.Mutex, key string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ok, unLockErr := mutex.UnlockContext(ctx)
	if unLockErr != nil || !ok {
		r.logger.Errorf("[redis lock] unlock failed key=%s ok=%v err=%v\n", key, ok, unLockErr)
	} else {
		r.logger.Infof("[redis lock] lock released key=%s", key)
	}
}

func (r *RedisLock) LockWithTries(ctx context.Context, key string, expireSecond int64, tries int, fn func(isLock bool, lc LockContext) error) (err error) {
	if fn == nil {
		return errors.New("fn is empty")
	}
	if key == "" {
		return errors.New("key is empty")
	}
	if expireSecond < 1 {
		expireSecond = 1
	}
	if tries < 0 {
		tries = 0
	}
	tries += 1 // tries 表示总尝试次数，调用方传 0 表示只试一次（+1=1次）
	expiryOption := redsync.WithExpiry(time.Duration(expireSecond) * time.Second)
	triesOption := redsync.WithTries(tries)
	retryDelayOpt := redsync.WithRetryDelay(time.Millisecond * 100)
	mutex := r.rl.NewMutex(key, expiryOption, triesOption, retryDelayOpt)
	if err = mutex.LockContext(ctx); err != nil {
		return fn(false, newRedisLockContext(mutex, err))
	}
	r.logger.Infof("[redis lock] lock acquired key=%s expire=%ds tries=%d", key, expireSecond, tries)
	defer r.unlockWithTimeout(mutex, key)
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

	retryDelay := 50 * time.Millisecond
	tries := int(triesTimeSecond)*20 + 1

	expiryOption := redsync.WithExpiry(time.Duration(expireSecond) * time.Second)
	triesOption := redsync.WithTries(tries)
	retryDelayOption := redsync.WithRetryDelay(retryDelay)
	mutex := r.rl.NewMutex(key, expiryOption, triesOption, retryDelayOption)
	// WHY: 使用 lockCtx 命名避免遮蔽函数参数 ctx
	lockCtx, cancel := context.WithTimeout(ctx, time.Duration(triesTimeSecond)*time.Second)
	defer cancel()

	if err = mutex.LockContext(lockCtx); err != nil {
		return fn(false, newRedisLockContext(mutex, err))
	}
	r.logger.Infof("[redis lock] lock acquired key=%s expire=%ds triesTime=%ds", key, expireSecond, triesTimeSecond)
	defer r.unlockWithTimeout(mutex, key)

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
			// WHY: 使用 select+timer 替代 time.Sleep，使 sleep 可响应 context 取消
			jitter := time.Millisecond*50 + time.Duration(common.WeakRandInt63n(20))*time.Millisecond
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(jitter):
			}
			attempt++
			continue
		}
		break
	}
	r.logger.Infof("[redis lock] lock acquired key=%s expire=%ds", key, expireSecond)
	defer r.unlockWithTimeout(mutex, key)

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
		return false, err
	}
	return true, nil
}

func (lc *RedisLockContext) Extend(ctx context.Context) (bool, error) {
	ok, err := lc.mutex.ExtendContext(ctx)
	if err != nil || !ok {
		return false, err
	}
	return true, nil
}

func (lc *RedisLockContext) Err() error {
	return lc.err
}
