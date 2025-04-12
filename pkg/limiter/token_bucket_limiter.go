package limiter

import (
	"context"
	"golang.org/x/time/rate"
	"sync"
	"time"
)

var (
	prefixKey  = "token_bucket:"
	limiterMap sync.Map
)

type bucketLimiter struct {
	key         string // 限流key
	lastGetTime int64  //上一次获取token的时间戳
	limiter     *rate.Limiter
}

type BucketLimit = rate.Limit

// BucketEvery converts a minimum time interval between events to a Limit.
func BucketEvery(interval time.Duration) BucketLimit {
	return rate.Every(interval)
}

// NewTokenBucket key限流器名称，r限流器配置，burst桶容量，
func NewTokenBucket(key string, r BucketLimit, burst int) (*bucketLimiter, bool) {
	key = prefixKey + key

	l, ok := limiterMap.Load(key)
	if ok {
		return l.(*bucketLimiter), false
	}
	limiter := &bucketLimiter{key, time.Now().Unix(), rate.NewLimiter(r, burst)}
	limiterMap.Store(key, limiter)
	return limiter, true
}

// Allow 判断是否有空余
func (l *bucketLimiter) Allow() bool {
	l.lastGetTime = time.Now().Unix()
	return l.limiter.Allow()
}

// Wait 等待xx获取
func (l *bucketLimiter) Wait(ctx context.Context) error {
	l.lastGetTime = time.Now().Unix()
	return l.limiter.Wait(ctx)
}

// Cancel 把拿到的令牌放回去
func (l *bucketLimiter) Cancel() {
	l.limiter.Reserve().Cancel()
}

// SetLimit 重新设置限流器
func (l *bucketLimiter) SetLimit(newLimit rate.Limit) {
	l.limiter.SetLimit(newLimit)
}

// SetBurst 重新设置桶的大小
func (l *bucketLimiter) SetBurst(newBurst int) {
	l.limiter.SetBurst(newBurst)
}

func (l *bucketLimiter) CloseTokenBucket() {
	limiterMap.Delete(l.key)
}
