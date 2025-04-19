package routers

import (
	. "snowgo/internal/api/account"

	"github.com/gin-gonic/gin"
)

// 用户相关路由
func userRouters(r *gin.RouterGroup) {
	userGroup := r.Group("/account")

	adminGroup := userGroup.Group("/admin")
	{
		adminGroup.GET("/user", GetUserList)
		adminGroup.POST("/user", CreateUser)
		adminGroup.DELETE("/user", DeleteUserById)
		adminGroup.GET("/user/detail", GetUserInfo)
	}
}
