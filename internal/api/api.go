package api

import (
	"context"
	"snowgo/internal/di"
	"snowgo/pkg/xlogger"
	"snowgo/pkg/xresponse"
	str "snowgo/pkg/xstr_tool"
	"time"

	"github.com/gin-gonic/gin"
)

// Index 首页
func Index(c *gin.Context) {
	xresponse.Success(c, gin.H{
		"client_ip":  c.ClientIP(),
		"random_str": str.RandStr(10, str.LowerFlag|str.UpperFlag|str.DigitFlag),
	})
}

// Liveness 存活检查
func Liveness(c *gin.Context) {
	c.JSON(200, gin.H{"status": "ok"})
}

// Readiness 就绪检查
func Readiness(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	// db检查
	container := di.GetContainer(c)
	// mysql检查
	if _, err := container.GetMyDB().CheckDBAlive(ctx); err != nil {
		xlogger.ErrorfCtx(c, "db check err: %v", err.Error())
		c.JSON(503, gin.H{
			"status": "not ready",
			"error":  err.Error(),
		})
		return
	}
	// redis检查
	if _, err := container.GetRDB().Ping(ctx).Result(); err != nil {
		xlogger.ErrorfCtx(c, "redis check err: %v", err.Error())
		c.JSON(503, gin.H{
			"status": "not ready",
			"error":  err.Error(),
		})
		return
	}
	c.JSON(200, gin.H{"status": "ready"})
}
