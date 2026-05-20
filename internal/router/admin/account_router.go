package admin

import (
	"github.com/gin-gonic/gin"
	"snowgo/internal/api/admin/account"
	"snowgo/internal/constant"
	"snowgo/internal/router/middleware"
)

// 用户相关路由
func accountRouters(r *gin.RouterGroup) {
	accountGroup := r.Group("/account")
	{
		// 用户
		accountGroup.GET("/user", middleware.PermissionAuth(constant.PermAccountUserList), account.GetUserList)
		accountGroup.POST("/user", middleware.PermissionAuth(constant.PermAccountUserCreate), account.CreateUser)
		accountGroup.PUT("/user", middleware.PermissionAuth(constant.PermAccountUserUpdate), account.UpdateUser)
		// 当前登录用户权限（仅需 JWTAuth，不需要 PermissionAuth）
		accountGroup.GET("/user/permission", account.GetUserPermission)
		accountGroup.POST("/user/pwd", middleware.PermissionAuth(constant.PermAccountUserResetPwd), account.ResetPwdById)
		accountGroup.DELETE("/user/:id", middleware.PermissionAuth(constant.PermAccountUserDelete), account.DeleteUserById)
		accountGroup.GET("/user/:id", middleware.PermissionAuth(constant.PermAccountUserDetail), account.GetUserInfo)
		// 菜单权限
		accountGroup.GET("/menu", middleware.PermissionAuth(constant.PermAccountMenuList), account.GetMenuList)
		accountGroup.POST("/menu", middleware.PermissionAuth(constant.PermAccountMenuCreate), account.CreateMenu)
		accountGroup.PUT("/menu", middleware.PermissionAuth(constant.PermAccountMenuUpdate), account.UpdateMenu)
		accountGroup.DELETE("/menu/:id", middleware.PermissionAuth(constant.PermAccountMenuDelete), account.DeleteMenuById)
		// 角色管理
		accountGroup.GET("/role", middleware.PermissionAuth(constant.PermAccountRoleList), account.GetRoleList)
		accountGroup.POST("/role", middleware.PermissionAuth(constant.PermAccountRoleCreate), account.CreateRole)
		accountGroup.PUT("/role", middleware.PermissionAuth(constant.PermAccountRoleUpdate), account.UpdateRole)
		accountGroup.GET("/role/:id", middleware.PermissionAuth(constant.PermAccountRoleDetail), account.GetRoleById)
		accountGroup.DELETE("/role/:id", middleware.PermissionAuth(constant.PermAccountRoleDelete), account.DeleteRole)
	}
}
