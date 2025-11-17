package xlock

import "context"

type Lock interface {
	// LockWithTries 重试到x次加锁
	LockWithTries(ctx context.Context, key string, expireSecond int64, tries int, fn func(isLock bool, LockContext LockContext) error) (err error)
	// LockWithTriesTime 重试多次s加锁
	LockWithTriesTime(ctx context.Context, key string, expireSecond int64, triesTimeSecond int64, fn func(isLock bool, LockContext LockContext) error) (err error)
	// TryLock 直接加锁，如果被锁了直接加锁失败
	TryLock(ctx context.Context, key string, expireSecond int64, fn func(isLock bool, LockContext LockContext) error) (err error)
	// ReTryLock 重试直到锁成功(不建议使用！！！)
	ReTryLock(ctx context.Context, key string, expireSecond int64, fn func(LockContext LockContext) error) (err error)
}

type LockContext interface {
	// Unlock 解锁
	Unlock(ctx context.Context) (bool, error)
	// Extend 续期
	Extend(ctx context.Context) (bool, error)
	// Err 错误
	Err() error
}
