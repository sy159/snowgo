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
		accountGroup.DELETE("/user", middleware.PermissionAuth(constant.PermAccountUserDelete), account.DeleteUserById)
		accountGroup.GET("/user/detail", middleware.PermissionAuth(constant.PermAccountUserDetail), account.GetUserInfo)
		accountGroup.POST("/user/pwd", middleware.PermissionAuth(constant.PermAccountUserResetPwd), account.ResetPwdById)
		accountGroup.GET("/user/permission", account.GetUserPermission)
		// 菜单权限
		accountGroup.GET("/menu", middleware.PermissionAuth(constant.PermAccountMenuList), account.GetMenuList)
		accountGroup.POST("/menu", middleware.PermissionAuth(constant.PermAccountMenuCreate), account.CreateMenu)
		accountGroup.PUT("/menu", middleware.PermissionAuth(constant.PermAccountMenuUpdate), account.UpdateMenu)
		accountGroup.DELETE("/menu", middleware.PermissionAuth(constant.PermAccountMenuDelete), account.DeleteMenuById)
		// 角色管理
		accountGroup.GET("/role", middleware.PermissionAuth(constant.PermAccountRoleList), account.GetRoleList)
		accountGroup.GET("/role/detail", middleware.PermissionAuth(constant.PermAccountRoleDetail), account.GetRoleById)
		accountGroup.POST("/role", middleware.PermissionAuth(constant.PermAccountRoleCreate), account.CreateRole)
		accountGroup.PUT("/role", middleware.PermissionAuth(constant.PermAccountRoleUpdate), account.UpdateRole)
		accountGroup.DELETE("/role", middleware.PermissionAuth(constant.PermAccountRoleDelete), account.DeleteRole)
	}
}
