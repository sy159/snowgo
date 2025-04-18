package xlimiter

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestNewTokenBucket(t *testing.T) {
	tb1, first := NewTokenBucket("user:123", 5, 10)
	tb2, second := NewTokenBucket("user:123", 5, 10)

	if !first || second {
		t.Errorf("Expected first=true, second=false, got first=%v, second=%v", first, second)
	}

	if tb1 != tb2 {
		t.Errorf("Expected same limiter instance for same key")
	}
}

func TestBucketEvery(t *testing.T) {
	// 每 100ms 产生一个令牌，相当于每秒 10 次
	limit := BucketEvery(100 * time.Millisecond)
	tb, _ := NewTokenBucket("test:every", limit, 1)

	// 一开始允许一次（桶容量为 1）
	if !tb.Allow() {
		t.Error("Expected first Allow to succeed due to burst")
	}

	// 马上再调用，应该失败（100ms 还没到，没 token）
	if tb.Allow() {
		t.Error("Expected second Allow to fail due to token not yet generated")
	}

	// 等待足够的时间让 token 生成
	time.Sleep(110 * time.Millisecond)

	if !tb.Allow() {
		t.Error("Expected Allow to succeed after waiting for token")
	}
}

func TestAllow(t *testing.T) {
	tb, _ := NewTokenBucket("test:allow", 10, 2)

	first := tb.Allow()
	second := tb.Allow()
	// 初始 burst 应该允许 2 次
	if !first || !second {
		t.Errorf("Expected first two Allow() calls to succeed")
	}

	// 第三次立即失败
	if tb.Allow() {
		t.Errorf("Expected third Allow() call to fail due to burst exhausted")
	}
}

func TestWait(t *testing.T) {
	tb, _ := NewTokenBucket("test:wait", 1, 1)

	_ = tb.Wait(context.Background()) // 第一个令牌
	start := time.Now()
	_ = tb.Wait(context.Background()) // 第二个令牌需要等 1s

	if time.Since(start) < 900*time.Millisecond {
		t.Errorf("Expected Wait() to block for ~1s, but returned too early")
	}
}

func TestReserve(t *testing.T) {
	tb, _ := NewTokenBucket("test:reserve", 1, 1)

	res := tb.Reserve()
	if !res.OK() {
		t.Errorf("Expected Reserve to succeed")
	}

	delay := res.Delay()
	if delay < 0 {
		t.Errorf("Expected non-negative delay")
	}
}

func TestSetLimitAndBurst(t *testing.T) {
	tb, _ := NewTokenBucket("test:set", 1, 1)

	// 先把初始的 token 消耗掉
	tb.Allow()

	// 设置新速率为 10 token/sec，桶容量为 5
	tb.SetLimit(10)
	tb.SetBurst(5)

	// 等待 500ms，令牌桶应该可以生成至少 5 个令牌
	time.Sleep(500 * time.Millisecond)

	// 连续 5 次 Allow 应该都成功
	success := 0
	for i := 0; i < 5; i++ {
		if tb.Allow() {
			success++
		}
	}
	if success != 5 {
		t.Errorf("Expected 5 tokens after refill, got %d", success)
	}
}

func TestClose(t *testing.T) {
	tb, _ := NewTokenBucket("test:close", 1, 1)
	tb.Close()

	if _, ok := limiterMap.Load(tb.key); ok {
		t.Errorf("Expected limiter to be removed after Close()")
	}
}

//func TestAutoCleanup(t *testing.T) {
//	tb, _ := NewTokenBucket("test:cleanup", 1, 1)
//	tb.lastGetTime.Store(time.Now().Add(-1 * time.Hour).UnixNano()) // 模拟过期时间
//
//	// 提前触发一次清理（等待清理周期）
//	time.Sleep(defaultCleanupInterval + 1*time.Second)
//
//	if _, ok := limiterMap.Load(tb.key); ok {
//		t.Errorf("Expected limiter to be auto-cleaned")
//	}
//}

func TestConcurrentAccess(t *testing.T) {
	tb, _ := NewTokenBucket("test:concurrent", 100, 100)

	var wg sync.WaitGroup
	successCount := int32(0)

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if tb.Allow() {
				successCount++
			}
		}()
	}

	wg.Wait()

	if successCount == 0 {
		t.Errorf("Expected at least some Allow() calls to succeed concurrently")
	}
}
