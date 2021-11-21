package routers

import (
	"gin-api/internal/api"

	"github.com/gin-gonic/gin"
)

// 根路由配置
func rootRouters(r *gin.Engine) {
	r.GET("/", func(c *gin.Context) {
		c.Request.URL.Path = "/index"
		r.HandleContext(c) // 内部路由重定向
		// c.Redirect(http.StatusMovedPermanently,"http://xxx.com")  // 外部路由重定向
	})
	r.GET("/index", api.Index)
}
