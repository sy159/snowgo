package middleware

import (
	"github.com/pkg/errors"
	"snowgo/config"
	"snowgo/internal/di"
	"snowgo/pkg/xauth"
	"snowgo/pkg/xauth/jwt"
	e "snowgo/pkg/xerror"
	"snowgo/pkg/xresponse"
	"strings"

	"github.com/gin-gonic/gin"
)

// JWTAuth 基于JWT的认证中间件
func JWTAuth() func(c *gin.Context) {
	cfg := config.Get()
	jwtManager := jwt.NewJwtManager(&jwt.Config{
		JwtSecret:             cfg.Jwt.JwtSecret,
		Issuer:                cfg.Jwt.Issuer,
		AccessExpirationTime:  cfg.Jwt.AccessExpirationTime,
		RefreshExpirationTime: cfg.Jwt.RefreshExpirationTime,
	})
	return func(c *gin.Context) {
		// 客户端携带Token有三种方式 1.放在请求头 2.放在请求体 3.放在URI
		// 假设Token放在Header的Authorization中，并使用Bearer开头
		authHeader := c.Request.Header.Get("Authorization")
		if authHeader == "" {
			xresponse.FailByError(c, e.TokenNotFound)
			c.Abort()
			return
		}
		// 按空格分割
		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			xresponse.FailByError(c, e.TokenIncorrectFormat)
			c.Abort()
			return
		}
		// parts[1]是获取到的tokenString，我们使用之前定义好的解析JWT的函数来解析它
		mc, err := jwtManager.ParseToken(parts[1])
		if err != nil {
			xresponse.Fail(c, e.TokenInvalid.GetErrCode(), err.Error())
			c.Abort()
			return
		}

		// 检查token的过期时间，以及type
		if err := mc.ValidAccessToken(); err != nil {
			if errors.Is(err, jwt.ErrInvalidTokenType) {
				xresponse.FailByError(c, e.TokenTypeError)
				c.Abort()
				return
			}
			xresponse.FailByError(c, e.TokenExpired)
			c.Abort()
			return
		}

		// 将当前请求的username信息保存到请求的上下文c上
		c.Set(xauth.XUserId, mc.UserId)
		c.Set(xauth.XUserName, mc.Username)
		c.Next() // 后续的处理函数可以用过c.Get("userId")来获取当前请求的用户信息
	}
}

// PermissionAuth 接口权限校验
func PermissionAuth(requiredPerm string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 拿到 userId
		uidIfc, exists := c.Get(xauth.XUserId)
		if !exists {
			xresponse.FailByError(c, e.HttpUnauthorized)
			c.Abort()
			return
		}
		userId := uidIfc.(int64)

		container := di.GetContainer(c)
		// 拿该用户的perms列表
		perms, err := container.UserService.GetPermsListById(c, int32(userId))
		if err != nil {
			xresponse.FailByError(c, e.HttpInternalServerError)
			c.Abort()
			return
		}

		// 校验是否有接口权限
		allowed := false
		for _, p := range perms {
			if p == requiredPerm {
				allowed = true
				break
			}
		}
		if !allowed {
			xresponse.FailByError(c, e.HttpForbidden)
			c.Abort()
			return
		}

		c.Next()
	}
}
