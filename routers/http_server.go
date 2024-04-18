package routers

import (
	"context"
	"fmt"
	"net/http"
	"snowgo/config"
	"snowgo/utils/logger"
	"time"
)

var (
	HttpServer *http.Server
)

// StartHttpServer 初始化路由，开启http服务
func StartHttpServer() {
	// 初始化路由
	router := InitRouter()
	HttpServer = &http.Server{
		Addr:           fmt.Sprintf("%s:%d", config.ServerConf.Addr, config.ServerConf.Port),
		Handler:        router,
		ReadTimeout:    time.Duration(config.ServerConf.ReadTimeout) * time.Second,
		WriteTimeout:   time.Duration(config.ServerConf.WriteTimeout) * time.Second,
		MaxHeaderBytes: config.ServerConf.MaxHeaderMB << 20,
	}

	go func() {
		fmt.Printf("%s:%s is running on %s\n", config.ServerConf.Name, config.ServerConf.Version, HttpServer.Addr)
		if err := HttpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Server Listen: %s\n", err)
		}
	}()
}

// StopHttpServer 停止服务
func StopHttpServer() (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	// x秒内优雅关闭服务（将未处理完的请求处理完再关闭服务）
	if err := HttpServer.Shutdown(ctx); err != nil {
		logger.Fatalf("Server Shutdown: %s", err.Error())
	}
	return
}

// RestartHttpServer 重启服务
func RestartHttpServer() (err error) {
	err = StopHttpServer()
	if err == nil {
		StartHttpServer()
	}
	return
}
