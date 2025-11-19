package router

import (
	"github.com/gin-contrib/pprof"
	"snowgo/config"
	"snowgo/internal/api"
	"snowgo/internal/di"
	"snowgo/internal/router/middleware"
	"snowgo/pkg/xenv"
	e "snowgo/pkg/xerror"
	"snowgo/pkg/xresponse"

	"github.com/gin-gonic/gin"
)

type option func(*gin.RouterGroup)

// 根据启动配置设置运行的mode
func setMode() {
	if xenv.Dev() {
		gin.SetMode(gin.DebugMode)
	} else if xenv.Uat() {
		gin.SetMode(gin.TestMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
}

// 中间件注册使用
func loadMiddleWare(router *gin.Engine, container *di.Container) {
	router.Use(middleware.AccessLogger(), middleware.Recovery())
	//router.Use(middleware.Cors())
	// 依赖注入
	router.Use(middleware.InjectContainerMiddleware(container))
}

// 注册所有路由
func loadRouter(router *gin.Engine) {
	// 统一处理404页面
	router.NoRoute(func(c *gin.Context) {
		xresponse.FailByError(c, e.HttpNotFound)
	})

	// 注册pprof路由(白名单访问)
	cfg := config.Get()
	if cfg.Application.EnablePprof {
		// 只允许内网或指定网段访问
		iPWhitelist := []string{"127.0.0.1/32", "192.168.0.0/16"}
		pprofGroup := router.Group("", middleware.IPWhiteList(iPWhitelist))
		pprof.Register(pprofGroup)
	}

	// 注册健康检查
	router.GET("/healthz", api.Liveness)
	router.GET("/readyz", api.Readiness)

	// 创建根路由组，并添加前缀
	apiGroup := router.Group("/api")

	rootRouters(apiGroup) // 根目录下路由
	options := []option{  // 根据不同分组注册路由
		accountRouters, // 用户相关路由
		systemRouters,  // 系统相关路由
	}

	// 注册其他分组下的路由
	for _, opt := range options {
		opt(apiGroup)
	}
}

// InitRouter 初始化路由
func InitRouter(container *di.Container) *gin.Engine {
	// 设置模式
	setMode()
	// 创建引擎
	router := gin.New()
	// 中间件注册
	loadMiddleWare(router, container)
	// 路由注册
	loadRouter(router)
	return router
}
