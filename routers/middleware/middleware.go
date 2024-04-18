package middleware

import (
	"fmt"
	"github.com/google/uuid"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"snowgo/config"
	"snowgo/utils"
	"snowgo/utils/color"
	e "snowgo/utils/error"
	"snowgo/utils/logger"
	"snowgo/utils/response"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
)

// AccessLogger 控制台输出访问日志，如果app配置了记录访问日志，会记录下访问日志
func AccessLogger() gin.HandlerFunc {
	return func(c *gin.Context) {

		startTime := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		method := c.Request.Method
		requestID := uuid.New().String()
		// 将请求 ID 存储到 Gin 上下文中
		c.Set("request_id", requestID)

		// 处理ico请求，不记录日志
		if c.Request.URL.Path == "/favicon.ico" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()

		cost := time.Since(startTime)
		// 控制台输出访问日志
		fmt.Printf("[%s] %25s | %4s | %14v | %5s  %#v | %12s\n%s",
			config.ServerConf.Name,
			time.Now().Format("2006-01-02 15:04:05.000"),
			color.StatusCodeColor(c.Writer.Status()),
			cost,
			color.MethodColor(method),
			c.Request.URL.RequestURI(),
			c.ClientIP(),
			c.Errors.ByType(gin.ErrorTypePrivate).String(),
		)

		// 记录访问日志
		if config.ServerConf.AccessLog {
			logger.Access(path,
				zap.Int("status", c.Writer.Status()),
				zap.String("method", method),
				zap.String("path", path),
				zap.String("query", query),
				zap.String("ip", c.ClientIP()),
				zap.Duration("cost", cost),
				zap.String("request_id", requestID),
				zap.String("user-agent", c.Request.UserAgent()),
				zap.String("errors", c.Errors.ByType(gin.ErrorTypePrivate).String()),
			)
		}
	}
}

// Recovery recover掉项目可能出现的panic(基于gin.Recovery()实现)
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Check for a broken connection, as it is not really a
				// condition that warrants a panic stack trace.
				var brokenPipe bool
				if ne, ok := err.(*net.OpError); ok {
					if se, ok := ne.Err.(*os.SyscallError); ok {
						if strings.Contains(strings.ToLower(se.Error()), "broken pipe") || strings.Contains(strings.ToLower(se.Error()), "connection reset by peer") {
							brokenPipe = true
						}
					}
				}

				if brokenPipe {
					httpRequest, _ := httputil.DumpRequest(c.Request, false)
					logger.Error(c.Request.URL.Path,
						zap.Any("error", err),
						zap.String("request", string(httpRequest)),
					)
					// If the connection is dead, we can't write a status to it.
					_ = c.Error(err.(error)) // nolint: err check
					c.Abort()
					return
				}

				logger.Error("[Recovery from panic]",
					zap.String("error", utils.ErrorToString(err)),
					zap.String("method", c.Request.Method),
					zap.String("path", c.Request.URL.Path),
					zap.String("query", c.Request.URL.RawQuery),
					zap.String("ip", c.ClientIP()),
					zap.String("request_id", c.GetString("request_id")),
					zap.String("user-agent", c.Request.UserAgent()),
				)
				//c.AbortWithStatus(http.StatusInternalServerError)  // 直接状态码为500
				response.FailByError(c, e.HttpInternalServerError)
				c.Abort()
			}
		}()
		c.Next()
	}
}

// Cors 前后端跨域设置
func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		origin := c.Request.Header.Get("Origin") //请求头部
		if origin != "" {
			// 服务端允许的地址
			c.Header("Access-Control-Allow-Origin", "*")
			// 服务端支持的所有跨域请求的方法
			c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, DELETE,UPDATE")
			//允许跨域设置可以返回其他子段(可根据需求添加)
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Length, X-CSRF-Token, Token, Session, X_Requested_With ,Accept, Origin, Host, Connection, Accept-Encoding, Accept-Language, DNT, X-CustomHeader, Keep-Alive, User-Agent, X-Requested-With, If-Modified-Since, Cache-Control, Content-Type, Pragma")
			// 允许浏览器（客户端）可以解析的头部
			c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Content-Type")
			//允许客户端传递校验信息比如 cookie
			c.Header("Access-Control-Allow-Credentials", "true")

		}

		//允许类型校验 放行所有OPTIONS方法，因为有的模板是要请求两次的
		if method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
		}

		c.Next()
	}
}
