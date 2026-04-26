package router

import (
	"github.com/gin-gonic/gin"
	"snowgo/internal/api/account"
	"snowgo/internal/constant"
	"snowgo/internal/router/middleware"
)

// 用户相关路由
func accountRouters(r *gin.RouterGroup) {
	accountGroup := r.Group("/account")

	adminGroup := accountGroup.Group("/admin", middleware.JWTAuth())
	{
		// 用户
		adminGroup.GET("/user", middleware.PermissionAuth(constant.PermAccountUserList), account.GetUserList)
		adminGroup.POST("/user", middleware.PermissionAuth(constant.PermAccountUserCreate), account.CreateUser)
		adminGroup.PUT("/user", middleware.PermissionAuth(constant.PermAccountUserUpdate), account.UpdateUser)
		adminGroup.DELETE("/user", middleware.PermissionAuth(constant.PermAccountUserDelete), account.DeleteUserById)
		adminGroup.GET("/user/detail", middleware.PermissionAuth(constant.PermAccountUserDetail), account.GetUserInfo)
		adminGroup.POST("/user/pwd", middleware.PermissionAuth(constant.PermAccountUserResetPwd), account.ResetPwdById)
		adminGroup.GET("/user/permission", account.GetUserPermission)
		// 菜单权限
		adminGroup.GET("/menu", middleware.PermissionAuth(constant.PermAccountMenuList), account.GetMenuList)
		adminGroup.POST("/menu", middleware.PermissionAuth(constant.PermAccountMenuCreate), account.CreateMenu)
		adminGroup.PUT("/menu", middleware.PermissionAuth(constant.PermAccountMenuUpdate), account.UpdateMenu)
		adminGroup.DELETE("/menu", middleware.PermissionAuth(constant.PermAccountMenuDelete), account.DeleteMenuById)
		// 角色管理
		adminGroup.GET("/role", middleware.PermissionAuth(constant.PermAccountRoleList), account.GetRoleList)
		adminGroup.GET("/role/detail", middleware.PermissionAuth(constant.PermAccountRoleDetail), account.GetRoleById)
		adminGroup.POST("/role", middleware.PermissionAuth(constant.PermAccountRoleCreate), account.CreateRole)
		adminGroup.PUT("/role", middleware.PermissionAuth(constant.PermAccountRoleUpdate), account.UpdateRole)
		adminGroup.DELETE("/role", middleware.PermissionAuth(constant.PermAccountRoleDelete), account.DeleteRole)
	}

	authGroup := accountGroup.Group("/auth")
	{
		authGroup.POST("/login", account.Login)
		authGroup.POST("/refresh-token", account.RefreshToken)
		authGroup.POST("/logout", middleware.JWTAuth(), account.Logout)
	}
}
