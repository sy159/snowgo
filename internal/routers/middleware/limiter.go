package middleware

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	e "snowgo/pkg/xerror"
	"snowgo/pkg/xlimiter"
	"snowgo/pkg/xresponse"
	"time"
)

// AccessLimiter 可用于路由访问限流，可设置等待时间
func AccessLimiter(key string, r xlimiter.BucketLimit, b int, waitTime time.Duration) gin.HandlerFunc {
	bucketLimiter, _ := xlimiter.NewTokenBucket(key, r, b)
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c, waitTime)
		defer cancel()
		if err := bucketLimiter.Wait(ctx); err != nil {
			// 这里先不处理日志了，如果返回错误就直接 429
			xresponse.FailByError(c, e.TooManyRequests)
			c.Abort()
			return
		}
		c.Next()
	}
}

// KeyLimiter 可用于ip，用户等标示性限流
func KeyLimiter(r xlimiter.BucketLimit, b int) gin.HandlerFunc {
	return func(c *gin.Context) {
		// key 除了ip 之外也可以是其他的，例如user name等根据需求调整
		key := c.ClientIP()
		fmt.Println(key)

		bucketLimiter, _ := xlimiter.NewTokenBucket(key, r, b)
		if !bucketLimiter.Allow() {
			xresponse.FailByError(c, e.KeyTooManyRequests)
			c.Abort()
			return
		}
		c.Next()
	}
}
