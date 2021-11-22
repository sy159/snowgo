package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"snowgo/config"
	"snowgo/routers"
	"snowgo/utils/cache/redis"
	"snowgo/utils/logger"
	"syscall"
	"time"
)

func init() {
	// 初始化zap log全局配置
	logger.InitLogger()
	// 初始化配置文件
	config.InitConf()
}

func main() {

	// 初始化redis
	redis.InitRedis()
	defer redis.RDB.Close()

	// 初始化路由
	router := routers.InitRouter()

	server := &http.Server{
		Addr:           fmt.Sprintf("%s:%d", config.ServerConf.Addr, config.ServerConf.Port),
		Handler:        router,
		ReadTimeout:    time.Duration(config.ServerConf.ReadTimeout) * time.Second,
		WriteTimeout:   time.Duration(config.ServerConf.WriteTimeout) * time.Second,
		MaxHeaderBytes: config.ServerConf.MaxHeaderMB << 20,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Server Listen: %s\n", err)
		}
	}()

	// 等待中断信号来优雅地关闭服务器，为关闭服务器操作设置一个超时
	quit := make(chan os.Signal, 1)

	// kill -2 发送 syscall.SIGINT 信号，用户发送INTR字符(Ctrl+C)触发
	// kill -3 发送 syscall.SIGQUIT 信号，用户发送QUIT字符(Ctrl+/)触发
	// kill -15 发送 syscall.SIGTERM 信号，结束程序(可以被捕获、阻塞或忽略)
	// kill -9 发送 syscall.SIGKILL 信号，但是不能被捕获，所以不需要添加它
	// signal.Notify接受信号转发给quit
	signal.Notify(quit, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGKILL)
	<-quit
	// 创建一个5秒超时的context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// x秒内优雅关闭服务（将未处理完的请求处理完再关闭服务）
	if err := server.Shutdown(ctx); err != nil {
		logger.Fatalf("Server Shutdown: %s", err.Error())
	}
}
