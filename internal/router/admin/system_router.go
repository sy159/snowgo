package admin

import (
	"github.com/gin-gonic/gin"
	"snowgo/internal/api/admin/system"
	"snowgo/internal/constant"
	"snowgo/internal/router/middleware"
)

// 系统相关路由
func systemRouters(r *gin.RouterGroup) {
	systemGroup := r.Group("/system", middleware.JWTAuth())

	logGroup := systemGroup.Group("/log")
	{
		// 操作日志
		logGroup.GET("/operation", middleware.PermissionAuth(constant.PermSystemOperationLogList), system.GetOperationLogList)
	}

	dictGroup := systemGroup.Group("/dict")
	{
		// 字典管理
		dictGroup.GET("/", middleware.PermissionAuth(constant.PermSystemDictList), system.GetDictList)
		dictGroup.POST("/", middleware.PermissionAuth(constant.PermSystemDictCreate), system.CreateDict)
		dictGroup.PUT("/", middleware.PermissionAuth(constant.PermSystemDictUpdate), system.UpdateDict)
		dictGroup.DELETE("/", middleware.PermissionAuth(constant.PermSystemDictDelete), system.DeleteDictById)
		// 字典枚举信息，// 创建字典item，权限应该跟创建字典相同
		dictGroup.GET("/item", system.GetItemListByDictCode)
		dictGroup.POST("/item", middleware.PermissionAuth(constant.PermSystemDictCreate), system.CreateItem)
		dictGroup.PUT("/item", middleware.PermissionAuth(constant.PermSystemDictUpdate), system.UpdateDictItem)
		dictGroup.DELETE("/item", middleware.PermissionAuth(constant.PermSystemDictDelete), system.DeleteDictItem)
	}
}
