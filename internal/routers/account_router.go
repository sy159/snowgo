package routers

import (
	. "snowgo/internal/api/account"

	"github.com/gin-gonic/gin"
)

// 用户相关路由
func userRouters(r *gin.RouterGroup) {
	userGroup := r.Group("/account")
	{
		userGroup.GET("/user", GetUserList)
		userGroup.POST("/user", CreateUser)
		userGroup.DELETE("/user", DeleteUserById)
		userGroup.GET("/user/detail", GetUserInfo)
	}
}
