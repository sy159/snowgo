package router

import (
	"github.com/gin-gonic/gin"
	. "snowgo/internal/api/account"
	"snowgo/internal/constant"
	"snowgo/internal/router/middleware"
)

// 用户相关路由
func accountRouters(r *gin.RouterGroup) {
	accountGroup := r.Group("/account")

	adminGroup := accountGroup.Group("/admin", middleware.JWTAuth())
	{
		// 用户
		adminGroup.GET("/user", middleware.PermissionAuth(constant.PermAccountUserList), GetUserList)
		adminGroup.POST("/user", middleware.PermissionAuth(constant.PermAccountUserCreate), CreateUser)
		adminGroup.PUT("/user", middleware.PermissionAuth(constant.PermAccountUserUpdate), UpdateUser)
		adminGroup.DELETE("/user", middleware.PermissionAuth(constant.PermAccountUserDelete), DeleteUserById)
		adminGroup.GET("/user/detail", middleware.PermissionAuth(constant.PermAccountUserDetail), GetUserInfo)
		adminGroup.POST("/user/pwd", middleware.PermissionAuth(constant.PermAccountUserResetPwd), ResetPwdById)
		adminGroup.GET("/user/permission", GetUserPermission)
		// 菜单权限
		adminGroup.GET("/menu", middleware.PermissionAuth(constant.PermAccountMenuList), GetMenuList)
		adminGroup.POST("/menu", middleware.PermissionAuth(constant.PermAccountMenuCreate), CreateMenu)
		adminGroup.PUT("/menu", middleware.PermissionAuth(constant.PermAccountMenuUpdate), UpdateMenu)
		adminGroup.DELETE("/menu", middleware.PermissionAuth(constant.PermAccountMenuDelete), DeleteMenuById)
		// 角色管理
		adminGroup.GET("/role", middleware.PermissionAuth(constant.PermAccountRoleList), GetRoleList)
		adminGroup.GET("/role/detail", middleware.PermissionAuth(constant.PermAccountRoleDetail), GetRoleById)
		adminGroup.POST("/role", middleware.PermissionAuth(constant.PermAccountRoleCreate), CreateRole)
		adminGroup.PUT("/role", middleware.PermissionAuth(constant.PermAccountRoleUpdate), UpdateRole)
		adminGroup.DELETE("/role", middleware.PermissionAuth(constant.PermAccountRoleDelete), DeleteRole)
	}

	authGroup := accountGroup.Group("/auth")
	{
		authGroup.POST("/login", Login)
		authGroup.POST("/refresh-token", RefreshToken)
		authGroup.POST("/logout", middleware.JWTAuth(), Logout)
	}
}
