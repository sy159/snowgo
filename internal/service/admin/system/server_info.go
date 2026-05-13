package system

import (
	"os"
	"runtime"
	"snowgo/internal/constant"
	"syscall"
	"time"

	"snowgo/config"
	"snowgo/pkg/xenv"
	"snowgo/pkg/xruntime"
)

// ServiceInfo 服务信息
type ServiceInfo struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	Env          string `json:"env"`
	StartTime    string `json:"start_time"`
	Uptime       string `json:"uptime"`
	ReadTimeout  string `json:"read_timeout"`
	WriteTimeout string `json:"write_timeout"`
}

// GoRuntime Go 运行时信息
type GoRuntime struct {
	GoVersion  string `json:"go_version"`
	Goroutines int    `json:"goroutines"`
	MemAllocMB uint64 `json:"mem_alloc_mb"`
	MemTotalMB uint64 `json:"mem_total_mb"`
	MemSysMB   uint64 `json:"mem_sys_mb"`
	GCCount    uint32 `json:"gc_count"`
	NumCPU     int    `json:"num_cpu"`
}

// OsInfo 操作系统信息
type OsInfo struct {
	OS          string `json:"os"`
	Arch        string `json:"arch"`
	Hostname    string `json:"hostname"`
	DiskTotalGB uint64 `json:"disk_total_gb"`
	DiskUsedGB  uint64 `json:"disk_used_gb"`
	DiskFreeGB  uint64 `json:"disk_free_gb"`
}

// ServerOverview 概览数据
type ServerOverview struct {
	ServiceInfo ServiceInfo `json:"service_info"`
	GoRuntime   GoRuntime   `json:"go_runtime"`
	OsInfo      OsInfo      `json:"os_info"`
}

// GetServerOverview 获取服务概览数据（服务信息 + Go 运行时 + 操作系统信息）
func GetServerOverview() *ServerOverview {
	return &ServerOverview{
		ServiceInfo: getServiceInfo(),
		GoRuntime:   getGoRuntime(),
		OsInfo:      getOsInfo(),
	}
}

// getServiceInfo 获取服务信息（名称、版本、环境、启动时间、运行时长等）
func getServiceInfo() ServiceInfo {
	cfg := config.Get()
	srv := cfg.Application.Server
	start := xruntime.GetStartTime()

	return ServiceInfo{
		Name:         srv.Name,
		Version:      srv.Version,
		Env:          xenv.Env(),
		StartTime:    start.Format(constant.TimeFmtWithMS),
		Uptime:       time.Since(start).Round(time.Second).String(),
		ReadTimeout:  srv.ReadTimeout.String(),
		WriteTimeout: srv.WriteTimeout.String(),
	}
}

// getGoRuntime 获取 Go 运行时信息（版本、Goroutine 数、内存使用、GC 次数等）
func getGoRuntime() GoRuntime {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	return GoRuntime{
		GoVersion:  runtime.Version(),
		Goroutines: runtime.NumGoroutine(),
		MemAllocMB: mem.Alloc / 1024 / 1024,
		MemTotalMB: mem.TotalAlloc / 1024 / 1024,
		MemSysMB:   mem.Sys / 1024 / 1024,
		GCCount:    mem.NumGC,
		NumCPU:     runtime.NumCPU(),
	}
}

// getOsInfo 获取操作系统信息（OS、架构、主机名、磁盘使用情况等）
func getOsInfo() OsInfo {
	hostname, _ := os.Hostname()

	var stat syscall.Statfs_t
	// 获取当前工作目录所在磁盘分区
	err := syscall.Statfs(".", &stat)
	var totalGB, usedGB, freeGB uint64
	if err == nil {
		total := stat.Blocks * uint64(stat.Bsize)
		free := stat.Bfree * uint64(stat.Bsize)
		totalGB = total / 1024 / 1024 / 1024
		freeGB = free / 1024 / 1024 / 1024
		usedGB = totalGB - freeGB
	}

	return OsInfo{
		OS:          runtime.GOOS,
		Arch:        runtime.GOARCH,
		Hostname:    hostname,
		DiskTotalGB: totalGB,
		DiskUsedGB:  usedGB,
		DiskFreeGB:  freeGB,
	}
}
