package router

import (
	"github.com/gin-gonic/gin"
	. "snowgo/internal/api/system"
	"snowgo/internal/constant"
	"snowgo/internal/router/middleware"
)

// 系统相关路由
func systemRouters(r *gin.RouterGroup) {
	systemGroup := r.Group("/system", middleware.JWTAuth())

	logGroup := systemGroup.Group("/log")
	{
		// 操作日志
		logGroup.GET("/operation", middleware.PermissionAuth(constant.PermSystemOperationLogList), GetOperationLogList)
	}

	dictGroup := systemGroup.Group("/dict")
	{
		// 字典管理
		dictGroup.GET("/", middleware.PermissionAuth(constant.PermSystemDictList), GetDictList)
		dictGroup.POST("/", middleware.PermissionAuth(constant.PermSystemDictCreate), CreateDict)
		dictGroup.PUT("/", middleware.PermissionAuth(constant.PermSystemDictUpdate), UpdateDict)
	}
}
