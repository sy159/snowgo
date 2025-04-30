package middleware

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"snowgo/config"
	"snowgo/internal/constants"
	"snowgo/internal/di"
	"snowgo/pkg/xauth"
	"snowgo/pkg/xcolor"
	"snowgo/pkg/xdatabase/mysql"
	"snowgo/pkg/xdatabase/redis"
	e "snowgo/pkg/xerror"
	"snowgo/pkg/xlogger"
	"snowgo/pkg/xresponse"
	"strings"
	"time"
)

// 自定义一个结构体，实现 gin.ResponseWriter interface
type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

// Write 复制一份出来
func (w responseWriter) Write(b []byte) (int, error) {
	//向一个bytes.buffer中写一份数据来为获取body使用
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// AccessLogger 控制台输出访问日志，如果app配置了记录访问日志，会记录下访问日志
func AccessLogger() gin.HandlerFunc {
	return func(c *gin.Context) {

		startTime := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		method := c.Request.Method
		traceId := uuid.New().String()
		// 将请求 ID 存储到 Gin 上下文中
		c.Set(xauth.XTraceId, traceId)
		c.Set(xauth.XIp, c.ClientIP())
		c.Set(xauth.XUserAgent, c.Request.UserAgent())

		// trace_id放入header中
		c.Writer.Header().Set(xauth.XTraceId, traceId)

		// 处理ico请求，不记录日志
		if c.Request.URL.Path == "/favicon.ico" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		// 处理resp，记录resp的
		writer := &responseWriter{
			c.Writer,
			bytes.NewBuffer([]byte{}),
		}
		if config.ServerConf.EnableAccessLog {
			writer = &responseWriter{
				c.Writer,
				bytes.NewBuffer([]byte{}),
			}
			c.Writer = writer
		}

		reqBody, _ := c.GetRawData()
		// 把读取的body内容重新写入
		if len(reqBody) > 0 {
			c.Request.Body = io.NopCloser(bytes.NewBuffer(reqBody))
		}

		c.Next()

		cost := time.Since(startTime)
		bizCode := c.GetInt(xresponse.BizCode)  // 业务返回code
		bizMsg := c.GetString(xresponse.BizMsg) // 业务返回msg

		// 记录访问日志
		if config.ServerConf.EnableAccessLog {
			xlogger.Access(bizMsg,
				zap.Int("status", c.Writer.Status()),
				zap.Int("biz_code", bizCode),
				//zap.String("biz_msg", bizMsg),
				zap.String("method", method),
				zap.String("path", path),
				zap.String("query", query),
				zap.String("request_body", string(reqBody)),
				zap.String("ip", c.ClientIP()),
				zap.Duration("cost", cost),
				zap.String("res", writer.body.String()),
				zap.String("trace_id", traceId),
				zap.String("user_agent", c.Request.UserAgent()),
				zap.String("errors", c.Errors.ByType(gin.ErrorTypePrivate).String()),
			)
		} else {
			// 控制台输出访问日志
			fmt.Printf("%s %s %20s | status %3s | biz code %6s | %8v | %5s  %#v | %12s | %s\n",
				xcolor.GreenFont(fmt.Sprintf("[%s:%s]", config.ServerConf.Name, config.ServerConf.Version)),
				xcolor.YellowFont("[access] |"),
				time.Now().Format("2006-01-02 15:04:05.000"),
				xcolor.StatusCodeColor(c.Writer.Status()),
				xcolor.BizCodeColor(bizCode),
				cost,
				xcolor.MethodColor(method),
				c.Request.URL.RequestURI(),
				c.ClientIP(),
				//c.Errors.ByType(gin.ErrorTypePrivate).String(),
				bizMsg,
			)
		}
	}
}

// Recovery recover掉项目可能出现的panic(基于gin.Recovery()实现)
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				// 统一转换为error类型
				var err error
				switch v := r.(type) {
				case error:
					err = v
				default:
					err = fmt.Errorf("panic: %v", v)
				}

				// 检测 broken pipe 类错误（支持错误链）
				var brokenPipe bool
				var ne *net.OpError
				var se *os.SyscallError
				if errors.As(err, &ne) && errors.As(ne.Err, &se) {
					msg := strings.ToLower(se.Error())
					brokenPipe = strings.Contains(msg, "broken pipe") ||
						strings.Contains(msg, "connection reset by peer")
				}

				// 记录请求详情（过滤敏感头）
				httpRequest, _ := httputil.DumpRequest(c.Request, false)

				// 结构化日志字段
				logFields := []zap.Field{
					zap.Error(err), // 自动记录错误链
					zap.String("method", c.Request.Method),
					zap.String("path", c.Request.URL.Path),
					zap.String("query", c.Request.URL.RawQuery),
					zap.String("ip", c.ClientIP()),
					zap.String("trace_id", c.GetString("trace_id")),
					zap.String("user_agent", c.Request.UserAgent()),
					zap.ByteString("request", httpRequest),
				}

				if brokenPipe {
					// 连接已中断场景处理
					xlogger.Error("[Broken Connection] "+c.Request.URL.Path, logFields...)
					_ = c.Error(err) // 标记错误但不写响应
					c.Abort()
					return
				}

				// 常规 panic 处理（附加堆栈）
				logFields = append(logFields, zap.Stack("stack"))
				xlogger.Error("[Recovery from panic]", logFields...)

				// 返回标准化错误响应
				xresponse.FailByError(c, e.HttpInternalServerError)
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

// InjectContainerMiddleware 注入container
func InjectContainerMiddleware() gin.HandlerFunc {
	container := di.NewContainer(config.JwtConf, redis.RDB, mysql.DB, mysql.DbMap)
	return func(c *gin.Context) {
		c.Set(constants.CONTAINER, container)
		c.Next()
	}
}
