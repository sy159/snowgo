package xruntime_test

import (
	"snowgo/pkg/xruntime"
	"testing"
	"time"
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
