package main

import (
	"context"
	"os"
	"os/signal"
	"snowgo/config"
	"snowgo/internal/di"
	"snowgo/internal/server"
	"snowgo/pkg/xenv"
	"snowgo/pkg/xlogger"
	"snowgo/pkg/xtrace"
	"syscall"
	"time"
)

func init() {
	// 初始化配置文件
	config.Init("./config")

	// 初始化zap log全局配置
	xlogger.Init("./logs")
}

func main() {
	// 获取配置
	cfg := config.Get()
	// 链路注入
	var traceShutdown func(context.Context) error
	if cfg.Application.EnableTrace {
		traceShutdown = xtrace.InitTracer(
			cfg.Application.Server.Name,
			cfg.Application.Server.Version,
			xenv.Env(),
			cfg.Application.TempoEndpoint,
		)
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
			defer cancel()
			if err := traceShutdown(ctx); err != nil {
				xlogger.Errorf("tracer shutdown failed: %v", err)
			}
		}()
	}

	// 手动注入依赖
	container, err := di.NewContainer(
		di.WithJWT(cfg.Jwt),
		di.WithMySQL(cfg.Mysql, cfg.OtherDB),
		di.WithRedis(cfg.Redis),
	)
	if err != nil {
		xlogger.Fatalf("new container failed: %v", err)
	}

	// 启动服务
	server.StartHttpServer(container)

	// 等待中断信号来优雅地关闭服务器，为关闭服务器操作设置一个超时
	// kill -2 发送 syscall.SIGINT 信号，用户发送INTR字符(Ctrl+C)触发
	// kill -3 发送 syscall.SIGQUIT 信号，用户发送QUIT字符(Ctrl+/)触发
	// kill -15 发送 syscall.SIGTERM 信号，结束程序(可以被捕获、阻塞或忽略)
	// kill -9 发送 syscall.SIGKILL 信号，但是不能被捕获，所以不需要添加它
	quit := make(chan os.Signal, 1)
	// signal.Notify接受信号转发给quit
	signal.Notify(quit, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
	<-quit

	// 关闭服务
	_ = server.StopHttpServer()

	// 注入关闭
	if err := container.Close(); err != nil {
		xlogger.Errorf("container close error: %v", err)
	}
}
