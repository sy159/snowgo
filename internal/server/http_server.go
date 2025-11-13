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
	cfg := config.Get()
	HttpServer = &http.Server{
		Addr:           fmt.Sprintf("%s:%d", cfg.Application.Server.Addr, cfg.Application.Server.Port),
		Handler:        router,
		ReadTimeout:    time.Duration(cfg.Application.Server.ReadTimeout) * time.Second,
		WriteTimeout:   time.Duration(cfg.Application.Server.WriteTimeout) * time.Second,
		MaxHeaderBytes: cfg.Application.Server.MaxHeaderMB << 20,
	}

	go func() {
		banner := `
     _______..__   __.   ______   ____    __    ____  _______   ______   
    /       ||  \ |  |  /  __  \  \   \  /  \  /   / /  _____| /  __  \  
   |   (----` + "`" + `|   \|  | |  |  |  |  \   \/    \/   / |  |  __  |  |  |  | 
    \   \    |  . ` + "`" + `  | |  |  |  |   \            /  |  | |_ | |  |  |  | 
.----)   |   |  |\   | |  ` + "`" + `--'  |    \    /\    /   |  |__| | |  ` + "`" + `--'  | 
|_______/    |__| \__|  \______/      \__/  \__/     \______|  \______/  
`
		fmt.Printf("%s\n", xcolor.GreenFont(banner))
		fmt.Printf("%s %s %s is running on %s %s log mode %s \n",
			xcolor.GreenFont(fmt.Sprintf("[%s:%s]", cfg.Application.Server.Name, cfg.Application.Server.Version)),
			xcolor.GreenFont("|"),
			xcolor.PurpleFont(fmt.Sprintf("http://%s", HttpServer.Addr)),
			xcolor.RedBackground(xenv.Env()),
			xcolor.GreenFont("|"),
			xcolor.BlueFont(cfg.Log.Writer))

		if err := HttpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			xlogger.Panicf("Server Listen: %s\n", err)
		}
	}()
}

// StopHttpServer 停止服务
func StopHttpServer() (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	// x秒内优雅关闭服务（将未处理完的请求处理完再关闭服务）
	if err := HttpServer.Shutdown(ctx); err != nil {
		xlogger.Panicf("Server Shutdown: %s", err.Error())
	}
	xlogger.Info("Server Shutdown...")
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
