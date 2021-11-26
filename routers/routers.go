package routers

import (
	"snowgo/config"
	"snowgo/routers/middleware"
	e "snowgo/utils/error"
	"snowgo/utils/response"

	"github.com/gin-gonic/gin"
)

type option func(*gin.Engine)

// 根据启动配置设置运行的mode
func setMode() {
	if config.ServerConf.IsDebug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
}

// 中间件注册使用
func loadMiddleWare(router *gin.Engine) {
	router.Use(middleware.AccessLogger(), middleware.Recovery())
	router.Use(middleware.Cors())
}

// 注册所有路由
func loadRouter(router *gin.Engine) {
	// 统一处理404页面
	router.NoRoute(func(c *gin.Context) {
		response.FailByError(c, e.HttpNotFound)
	})

	rootRouters(router)  // 根目录下路由
	options := []option{ // 根据不同分组注册路由
		userRouters, // 用户相关路由
	}

	// 注册其他分组下的路由
	for _, opt := range options {
		opt(router)
	}
}

// InitRouter 初始化路由
func InitRouter() *gin.Engine {
	// 设置模式
	setMode()
	// 创建引擎
	router := gin.New()
	// 中间件注册
	loadMiddleWare(router)
	// 路由注册
	loadRouter(router)
	return router
}
