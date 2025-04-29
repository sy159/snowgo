package routers

import (
	"github.com/gin-gonic/gin"
	. "snowgo/internal/api/system"
)

// 系统相关路由
func systemRouters(r *gin.RouterGroup) {
	accountGroup := r.Group("/system")

	adminGroup := accountGroup.Group("/log")
	{
		// 操作日志
		adminGroup.GET("/operation", GetOperationLogList)
	}

}
