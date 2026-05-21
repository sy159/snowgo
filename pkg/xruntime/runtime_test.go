package xruntime_test

import (
	"sync"
	"testing"
	"time"

	"snowgo/pkg/xruntime"
)

func TestStartTime(t *testing.T) {
	// SetStartTime 应在 http_server 启动时调用
	xruntime.SetStartTime()

	start := xruntime.GetStartTime()
	if start.IsZero() {
		t.Error("GetStartTime() returned zero time")
	}

	// 验证时间在合理范围内（最近 10 秒内）
	diff := time.Since(start)
	if diff < 0 || diff > 10*time.Second {
		t.Errorf("GetStartTime() returned unexpected time: %v (since=%v)", start, diff)
	}
}

func TestStartTimeMonotonic(t *testing.T) {
	// 验证多次调用 GetStartTime 返回相同值
	xruntime.SetStartTime()
	first := xruntime.GetStartTime()
	time.Sleep(10 * time.Millisecond)
	second := xruntime.GetStartTime()

	if first != second {
		t.Errorf("GetStartTime() not monotonic: first=%v, second=%v", first, second)
	}
}

// === Additional tests ===

func TestGetStartTimeBeforeSet(t *testing.T) {
	// NOTE: This test CANNOT reliably verify GetStartTime before SetStartTime
	// in the current process, because other tests (TestStartTime, etc.) already
	// call SetStartTime which sets the package-level startTime variable.
	// The behavior is verified by the runtime.go source: a zero-initialized
	// time.Time variable returns IsZero() == true before SetStartTime is called.
	// This subtest just documents the design intent.
	t.Skip("cannot test zero-value time in multi-test process; see source comment")
}

func TestSetStartTimeMultipleCalls(t *testing.T) {
	// === Happy path ===
	t.Run("happy: SetStartTime overwrites previous value", func(t *testing.T) {
		xruntime.SetStartTime()
		first := xruntime.GetStartTime()

		time.Sleep(10 * time.Millisecond)
		xruntime.SetStartTime()
		second := xruntime.GetStartTime()

		if second.Before(first) {
			t.Errorf("second SetStartTime should be >= first: first=%v, second=%v", first, second)
		}
		// Both should be recent
		diff := time.Since(second)
		if diff < 0 || diff > 10*time.Second {
			t.Errorf("GetStartTime() returned unexpected time: %v (since=%v)", second, diff)
		}
	})
}

func TestGetStartTimeConcurrent(t *testing.T) {
	// === Happy path: concurrent reads ===
	t.Run("happy: concurrent GetStartTime calls", func(t *testing.T) {
		xruntime.SetStartTime()

		var wg sync.WaitGroup
		results := make(chan time.Time, 100)
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				results <- xruntime.GetStartTime()
			}()
		}
		wg.Wait()
		close(results)

		// All reads should return the same value
		first := <-results
		for r := range results {
			if r != first {
				t.Fatalf("concurrent GetStartTime returned different values: %v vs %v", first, r)
			}
		}
	})
}
