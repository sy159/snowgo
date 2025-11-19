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
}
