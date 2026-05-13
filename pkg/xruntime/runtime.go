package xruntime

import "time"

var startTime time.Time

// SetStartTime 记录服务启动时间（在 http_server.go 启动时调用）
func SetStartTime() {
	startTime = time.Now()
}

// GetStartTime 获取服务启动时间
func GetStartTime() time.Time {
	return startTime
}
