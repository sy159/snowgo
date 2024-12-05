package routers

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"snowgo/config"
	"snowgo/internal/routers/middleware"
	"snowgo/utils/env"
	e "snowgo/utils/error"
	"snowgo/utils/response"

	"github.com/gin-gonic/gin"
)

type option func(*gin.RouterGroup)

// 根据启动配置设置运行的mode
func setMode() {
	if env.Dev() {
		gin.SetMode(gin.DebugMode)
	} else if env.Uat() {
		gin.SetMode(gin.TestMode)
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

	// 注册pprof路由
	if config.ServerConf.EnablePprof {
		router.GET("/debug/pprof/*any", gin.WrapH(http.DefaultServeMux))
	}

	// 创建根路由组，并添加前缀
	apiGroup := router.Group(fmt.Sprintf("/api/%s", config.ServerConf.Version))
	rootRouters(apiGroup) // 根目录下路由
	options := []option{  // 根据不同分组注册路由
		userRouters, // 用户相关路由
	}

	// 注册其他分组下的路由
	for _, opt := range options {
		opt(apiGroup)
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
