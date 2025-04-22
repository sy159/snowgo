package routers

import (
	. "snowgo/internal/api/account"

	"github.com/gin-gonic/gin"
)

// 用户相关路由
func accountRouters(r *gin.RouterGroup) {
	accountGroup := r.Group("/account")

	adminGroup := accountGroup.Group("/admin")
	{
		// 用户
		adminGroup.GET("/user", GetUserList)
		adminGroup.POST("/user", CreateUser)
		adminGroup.PUT("/user", UpdateUser)
		adminGroup.DELETE("/user", DeleteUserById)
		adminGroup.GET("/user/detail", GetUserInfo)
		adminGroup.POST("/user/pwd", ResetPwdById)
		// 菜单权限
		adminGroup.GET("/menu", GetMenuList)
		adminGroup.POST("/menu", CreateMenu)
		adminGroup.PUT("/menu", UpdateMenu)
		adminGroup.DELETE("/menu", DeleteMenuById)
		// 角色管理
		adminGroup.GET("/role", GetRoleList)
		adminGroup.GET("/role/detail", GetRoleById)
		adminGroup.POST("/role", CreateRole)
		adminGroup.PUT("/role", UpdateRole)
		adminGroup.DELETE("/role", DeleteRole)
	}
}
