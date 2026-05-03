package admin

import (
	"github.com/gin-gonic/gin"
	"snowgo/internal/api/admin/account"
	"snowgo/internal/router/middleware"
)

// Register 路由配置
func Register(r *gin.RouterGroup) {
	admin := r.Group("/admin")

	// 登录认证相关
	auth := admin.Group("/auth")
	{
		auth.POST("/login", account.Login)
		auth.POST("/refresh-token", account.RefreshToken)
		auth.POST("/logout", middleware.JWTAuth(), account.Logout)
	}

	// 受保护接口（必须登录）
	protected := admin.Group("", middleware.JWTAuth())
	{
		accountRouters(protected) // 账户相关
		systemRouters(protected)  // 系统设置相关
	}
}
