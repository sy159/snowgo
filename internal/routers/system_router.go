package routers

import (
	"github.com/gin-gonic/gin"
	. "snowgo/internal/api/system"
	"snowgo/internal/routers/middleware"
)

// 系统相关路由
func systemRouters(r *gin.RouterGroup) {
	accountGroup := r.Group("/system", middleware.JWTAuth())

	adminGroup := accountGroup.Group("/log")
	{
		// 操作日志
		adminGroup.GET("/operation", middleware.PermissionAuth("system:operation-log:list"), GetOperationLogList)
	}
}
