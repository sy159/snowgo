package admin

import (
	"github.com/gin-gonic/gin"
	"snowgo/internal/api/admin/system"
	"snowgo/internal/constant"
	"snowgo/internal/router/middleware"
)

// 系统相关路由
func systemRouters(r *gin.RouterGroup) {
	systemGroup := r.Group("/system")

	// 服务信息（仅需 JWTAuth，不需要权限校验）
	systemGroup.GET("/info", system.GetServerInfo)

	logGroup := systemGroup.Group("/log")
	{
		// 操作日志
		logGroup.GET("/operation", middleware.PermissionAuth(constant.PermSystemOperationLogList), system.GetOperationLogList)
		// 登录日志
		logGroup.GET("/login", middleware.PermissionAuth(constant.PermSystemLoginLogList), system.GetLoginLogList)
	}

	dictGroup := systemGroup.Group("/dict")
	{
		// 字典管理
		dictGroup.GET("", middleware.PermissionAuth(constant.PermSystemDictList), system.GetDictList)
		dictGroup.POST("", middleware.PermissionAuth(constant.PermSystemDictCreate), system.CreateDict)
		dictGroup.PUT("", middleware.PermissionAuth(constant.PermSystemDictUpdate), system.UpdateDict)
		dictGroup.DELETE("/:id", middleware.PermissionAuth(constant.PermSystemDictDelete), system.DeleteDictById)
		// 字典枚举信息
		dictGroup.POST("/item", middleware.PermissionAuth(constant.PermSystemDictCreate), system.CreateItem)
		dictGroup.PUT("/item", middleware.PermissionAuth(constant.PermSystemDictUpdate), system.UpdateDictItem)
		// 字典枚举读取（仅需 JWTAuth，不需要 PermissionAuth）
		dictGroup.GET("/item/:code", system.GetItemListByDictCode)
		dictGroup.DELETE("/item/:id", middleware.PermissionAuth(constant.PermSystemDictDelete), system.DeleteDictItem)
	}
}
