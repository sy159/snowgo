package api

import (
	"context"
	"encoding/json"
	"snowgo/internal/constant"
	"snowgo/internal/di"
	"snowgo/pkg/xerror"
	"snowgo/pkg/xlogger"
	"snowgo/pkg/xmq"
	"snowgo/pkg/xresponse"
	str "snowgo/pkg/xstr_tool"
	"strconv"
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

// PublishMessage 发送消息
func PublishMessage(c *gin.Context) {
	container := di.GetContainer(c)
	delayMs, _ := strconv.ParseInt(c.Query("delay"), 10, 64)

	body := struct {
		ClientIP  string `json:"client_ip"`
		Timestamp int64  `json:"timestamp"`
	}{
		ClientIP:  c.ClientIP(),
		Timestamp: time.Now().UTC().UnixMilli(),
	}
	bodyBytes, _ := json.Marshal(body)
	msg := &xmq.Message{
		Body:    bodyBytes,
		Headers: map[string]interface{}{"source": "http-api"},
	}

	// 如果是延时消息
	if delayMs > 0 {
		err := container.Producer.PublishDelayed(c.Request.Context(), constant.DelayedExchange, constant.ExampleDelayedRoutingKey, msg, delayMs)
		if err != nil {
			xlogger.ErrorfCtx(c.Request.Context(), "publish delay message error: %s", err.Error())
			xresponse.FailByError(c, xerror.HttpInternalServerError)
			return
		}
	} else {
		err := container.Producer.Publish(c.Request.Context(), constant.NormalExchange, constant.ExampleNormalRoutingKey, msg)
		if err != nil {
			xlogger.ErrorfCtx(c.Request.Context(), "publish message error: %s", err.Error())
			xresponse.FailByError(c, xerror.HttpInternalServerError)
			return
		}
	}
	xresponse.Success(c, nil)
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
