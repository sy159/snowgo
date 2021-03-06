package routers

import (
	. "snowgo/internal/api/account"

	"github.com/gin-gonic/gin"
)

// 用户相关路由
func userRouters(r *gin.Engine) {
	userGroup := r.Group("/account")
	{
		userGroup.POST("/login", Login)
		userGroup.GET("/user/info", GetUserInfo)
	}
}
