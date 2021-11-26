package middleware

import (
	"snowgo/utils/auth/jwt"
	e "snowgo/utils/error"
	"snowgo/utils/response"
	"strings"

	"github.com/gin-gonic/gin"
)

// JWTAuth 基于JWT的认证中间件
func JWTAuth() func(c *gin.Context) {
	return func(c *gin.Context) {
		// 客户端携带Token有三种方式 1.放在请求头 2.放在请求体 3.放在URI
		// 假设Token放在Header的Authorization中，并使用Bearer开头
		authHeader := c.Request.Header.Get("Authorization")
		if authHeader == "" {
			response.FailByError(c, e.TokenNotFound)
			c.Abort()
			return
		}
		// 按空格分割
		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			response.FailByError(c, e.TokenIncorrectFormat)
			c.Abort()
			return
		}
		// parts[1]是获取到的tokenString，我们使用之前定义好的解析JWT的函数来解析它
		mc, err := jwt.ParseToken(parts[1])
		if err != nil {
			response.FailByError(c, e.TokenInvalid)
			c.Abort()
			return
		}

		// 检查token的type，refresh token不能使用
		if !mc.CheckTypeByClaims() {
			response.FailByError(c, e.TokenTypeError)
			c.Abort()
			return
		}

		// 检查token过期时间
		if !mc.CheckTimeByClaims() {
			response.FailByError(c, e.TokenExpired)
			c.Abort()
			return
		}

		// 将当前请求的username信息保存到请求的上下文c上
		c.Set("userId", mc.UserId)
		c.Set("username", mc.Username)
		c.Next() // 后续的处理函数可以用过c.Get("userId")来获取当前请求的用户信息
	}
}
