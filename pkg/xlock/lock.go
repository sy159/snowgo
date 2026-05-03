package xlock

import (
	"context"
	"fmt"
	"os"
)

type Logger interface {
	Infof(format string, args ...any)
	Errorf(format string, args ...any)
}

type defaultLogger struct{}

func (l *defaultLogger) Infof(format string, args ...any) {
	fmt.Printf("[lock][INFO] "+format+"\n", args...)
}
func (l *defaultLogger) Errorf(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, "[lock][ERROR] "+format+"\n", args...)
}

type Lock interface {
	// LockWithTries 重试到x次加锁
	LockWithTries(ctx context.Context, key string, expireSecond int64, tries int, fn func(isLock bool, lc LockContext) error) (err error)
	// LockWithTriesTime 重试多次s加锁
	LockWithTriesTime(ctx context.Context, key string, expireSecond int64, triesTimeSecond int64, fn func(isLock bool, lc LockContext) error) (err error)
	// TryLock 直接加锁，如果被锁了直接加锁失败
	TryLock(ctx context.Context, key string, expireSecond int64, fn func(isLock bool, lc LockContext) error) (err error)
	// ReTryLock 重试直到锁成功(不建议使用！！！)
	ReTryLock(ctx context.Context, key string, expireSecond int64, fn func(lc LockContext) error) (err error)
}

type LockContext interface {
	// Unlock 解锁
	Unlock(ctx context.Context) (bool, error)
	// Extend 续期
	Extend(ctx context.Context) (bool, error)
	// Err 错误
	Err() error
}
