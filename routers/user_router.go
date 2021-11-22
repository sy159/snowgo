package routers

import (
	. "snowgo/internal/api/user"

	"github.com/gin-gonic/gin"
)

// 用户相关路由
func userRouters(r *gin.Engine) {
	userGroup := r.Group("/user")
	{
		userGroup.GET("/info", GetUserInfo)
	}
}
