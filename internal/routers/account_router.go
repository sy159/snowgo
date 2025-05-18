package routers

import (
	"github.com/gin-gonic/gin"
	. "snowgo/internal/api/account"
	"snowgo/internal/routers/middleware"
)

// 用户相关路由
func accountRouters(r *gin.RouterGroup) {
	accountGroup := r.Group("/account")

	adminGroup := accountGroup.Group("/admin", middleware.JWTAuth())
	{
		// 用户
		adminGroup.GET("/user", middleware.PermissionAuth("account:user:list"), GetUserList)
		adminGroup.POST("/user", middleware.PermissionAuth("account:user:create"), CreateUser)
		adminGroup.PUT("/user", middleware.PermissionAuth("account:user:update"), UpdateUser)
		adminGroup.DELETE("/user", middleware.PermissionAuth("account:user:delete"), DeleteUserById)
		adminGroup.GET("/user/detail", middleware.PermissionAuth("account:user:detail"), GetUserInfo)
		adminGroup.POST("/user/pwd", middleware.PermissionAuth("account:user:reset_pwd"), ResetPwdById)
		adminGroup.GET("/user/permission", GetUserPermission)
		// 菜单权限
		adminGroup.GET("/menu", middleware.PermissionAuth("account:menu:list"), GetMenuList)
		adminGroup.POST("/menu", middleware.PermissionAuth("account:menu:create"), CreateMenu)
		adminGroup.PUT("/menu", middleware.PermissionAuth("account:menu:update"), UpdateMenu)
		adminGroup.DELETE("/menu", middleware.PermissionAuth("account:menu:delete"), DeleteMenuById)
		// 角色管理
		adminGroup.GET("/role", middleware.PermissionAuth("account:role:list"), GetRoleList)
		adminGroup.GET("/role/detail", middleware.PermissionAuth("account:role:detail"), GetRoleById)
		adminGroup.POST("/role", middleware.PermissionAuth("account:role:create"), CreateRole)
		adminGroup.PUT("/role", middleware.PermissionAuth("account:role:update"), UpdateRole)
		adminGroup.DELETE("/role", middleware.PermissionAuth("account:role:delete"), DeleteRole)
	}

	authGroup := accountGroup.Group("/auth")
	{
		authGroup.POST("/login", Login)
		authGroup.POST("/refresh-token", RefreshToken)
	}
}
