package server

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"net/http"
	"snowgo/config"
	"snowgo/internal/routers"
	"snowgo/pkg/xcolor"
	"snowgo/pkg/xenv"
	"snowgo/pkg/xlogger"
	"time"
)

var (
	HttpServer *http.Server
)

// StartHttpServer 初始化路由，开启http服务
func StartHttpServer() {
	// 初始化路由
	router := routers.InitRouter()
	HttpServer = &http.Server{
		Addr:           fmt.Sprintf("%s:%d", config.ServerConf.Addr, config.ServerConf.Port),
		Handler:        router,
		ReadTimeout:    time.Duration(config.ServerConf.ReadTimeout) * time.Second,
		WriteTimeout:   time.Duration(config.ServerConf.WriteTimeout) * time.Second,
		MaxHeaderBytes: config.ServerConf.MaxHeaderMB << 20,
	}

	go func() {
		fmt.Printf("%s %s is running on %s... log wirter %s \n",
			xcolor.BlueFont(fmt.Sprintf("[%s:%s]", config.ServerConf.Name, config.ServerConf.Version)),
			xcolor.RedBackground(xenv.Env()),
			HttpServer.Addr,
			xcolor.GreenFont(config.LogConf.Writer))
		if err := HttpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			xlogger.Fatalf("Server Listen: %s\n", err)
		}
	}()
}

// StopHttpServer 停止服务
func StopHttpServer() (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	// x秒内优雅关闭服务（将未处理完的请求处理完再关闭服务）
	if err := HttpServer.Shutdown(ctx); err != nil {
		xlogger.Fatalf("Server Shutdown: %s", err.Error())
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
