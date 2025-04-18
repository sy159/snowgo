package xlimiter

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"
)

var (
	// prefixKey 区分不同限流器的前缀
	prefixKey = "token_bucket:"
	// limiterMap 缓存所有限流器实例，线程安全
	limiterMap sync.Map // map[string]*BucketLimiter

	// cleanupOnce 确保清理任务只启动一次
	cleanupOnce sync.Once
	// 默认清理间隔和空闲阈值
	defaultCleanupInterval = 10 * time.Minute
	defaultIdleThreshold   = 30 * time.Minute
)

// BucketLimiter 是基于令牌桶算法的内存限流器，支持动态设置速率和突发容量。
// 适用场景：API QPS 控制、用户行为限流、任务速率限制等
type BucketLimiter struct {
	key         string        // 限流器唯一标识（prefixKey + 自定义 key）
	lastGetTime atomic.Int64  // 最后一次成功获取令牌的时间戳（UnixNano）
	limiter     *rate.Limiter // Go 标准库令牌桶
}

// BucketLimit 表示每秒产生的令牌数，等价于 rate.Limit
type BucketLimit = rate.Limit

// BucketEvery 根据两次事件的最小间隔时间，计算令牌桶速率。
// 例如：BucketEvery(100ms) 等同 10 次/秒。
func BucketEvery(interval time.Duration) BucketLimit {
	return rate.Every(interval)
}

// NewTokenBucket 创建或获取限流器，并在首次调用时启动自动清理任务。
// 参数：key（自定义分类 key）、limit（令牌生成速率, 每秒多少个）、burst（桶容量）
// 返回：限流器实例，以及是否首次创建。
func NewTokenBucket(key string, limit BucketLimit, burst int) (*BucketLimiter, bool) {
	// 启动后台清理
	ensureCleanup()

	fullKey := prefixKey + key
	if l, ok := limiterMap.Load(fullKey); ok {
		return l.(*BucketLimiter), false
	}
	bl := &BucketLimiter{
		key:     fullKey,
		limiter: rate.NewLimiter(limit, burst),
	}
	bl.lastGetTime.Store(time.Now().UnixNano())
	limiterMap.Store(fullKey, bl)
	return bl, true
}

// Allow 尝试立即获取令牌，成功返回 true（非阻塞），并更新时间戳。
func (bl *BucketLimiter) Allow() bool {
	if bl.limiter.Allow() {
		bl.lastGetTime.Store(time.Now().UnixNano())
		return true
	}
	return false
}

// Wait 阻塞直到获取令牌或 context 超时，成功时更新时间戳。
func (bl *BucketLimiter) Wait(ctx context.Context) error {
	err := bl.limiter.Wait(ctx)
	if err == nil {
		bl.lastGetTime.Store(time.Now().UnixNano())
	}
	return err
}

// Reserve 预留一个令牌，由调用方决定是否取消。预留成功时更新时间戳。
func (bl *BucketLimiter) Reserve() *rate.Reservation {
	r := bl.limiter.Reserve()
	if r.OK() {
		bl.lastGetTime.Store(time.Now().UnixNano())
	}
	return r
}

// SetLimit 动态修改令牌生成速率。
func (bl *BucketLimiter) SetLimit(newLimit rate.Limit) {
	bl.limiter.SetLimit(newLimit)
}

// SetBurst 动态修改桶容量。
func (bl *BucketLimiter) SetBurst(newBurst int) {
	bl.limiter.SetBurst(newBurst)
}

// Close 注销限流器，从全局缓存中移除。
func (bl *BucketLimiter) Close() {
	limiterMap.Delete(bl.key)
}

// ensureCleanup 启动后台清理协程，仅执行一次。清理超过空闲阈值的限流器。
func ensureCleanup() {
	cleanupOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(defaultCleanupInterval)
			defer ticker.Stop()
			for range ticker.C {
				now := time.Now().UnixNano()
				limiterMap.Range(func(k, v interface{}) bool {
					bl := v.(*BucketLimiter)
					if now-bl.lastGetTime.Load() > defaultIdleThreshold.Nanoseconds() {
						limiterMap.Delete(k)
					}
					return true
				})
			}
		}()
	})
}
