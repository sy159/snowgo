package routers

import (
	"github.com/gin-gonic/gin"
	. "snowgo/internal/api/account"
	"snowgo/internal/constants"
	"snowgo/internal/routers/middleware"
)

// 用户相关路由
func accountRouters(r *gin.RouterGroup) {
	accountGroup := r.Group("/account")

	adminGroup := accountGroup.Group("/admin", middleware.JWTAuth())
	{
		// 用户
		adminGroup.GET("/user", middleware.PermissionAuth(constants.PermAccountUserList), GetUserList)
		adminGroup.POST("/user", middleware.PermissionAuth(constants.PermAccountUserCreate), CreateUser)
		adminGroup.PUT("/user", middleware.PermissionAuth(constants.PermAccountUserUpdate), UpdateUser)
		adminGroup.DELETE("/user", middleware.PermissionAuth(constants.PermAccountUserDelete), DeleteUserById)
		adminGroup.GET("/user/detail", middleware.PermissionAuth(constants.PermAccountUserDetail), GetUserInfo)
		adminGroup.POST("/user/pwd", middleware.PermissionAuth(constants.PermAccountUserResetPwd), ResetPwdById)
		adminGroup.GET("/user/permission", GetUserPermission)
		// 菜单权限
		adminGroup.GET("/menu", middleware.PermissionAuth(constants.PermAccountMenuList), GetMenuList)
		adminGroup.POST("/menu", middleware.PermissionAuth(constants.PermAccountMenuCreate), CreateMenu)
		adminGroup.PUT("/menu", middleware.PermissionAuth(constants.PermAccountMenuUpdate), UpdateMenu)
		adminGroup.DELETE("/menu", middleware.PermissionAuth(constants.PermAccountMenuDelete), DeleteMenuById)
		// 角色管理
		adminGroup.GET("/role", middleware.PermissionAuth(constants.PermAccountRoleList), GetRoleList)
		adminGroup.GET("/role/detail", middleware.PermissionAuth(constants.PermAccountRoleDetail), GetRoleById)
		adminGroup.POST("/role", middleware.PermissionAuth(constants.PermAccountRoleCreate), CreateRole)
		adminGroup.PUT("/role", middleware.PermissionAuth(constants.PermAccountRoleUpdate), UpdateRole)
		adminGroup.DELETE("/role", middleware.PermissionAuth(constants.PermAccountRoleDelete), DeleteRole)
	}

	authGroup := accountGroup.Group("/auth")
	{
		authGroup.POST("/login", Login)
		authGroup.POST("/refresh-token", RefreshToken)
	}
}
