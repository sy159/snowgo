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
	xlogger.Infof("index traceId: %s time: %s\n", time.Now().Format("2006-01-02 15:04:05.000"))

	// redis测试
	//container := di.GetContainer(c)
	//indexTest, err := container.Cache.Get(c, "snow_index_test")
	//if err != nil {
	//	xlogger.Errorf("redis get err: %v, request_id: %s", err.Error(), requestId)
	//	xresponse.FailByError(c, e.HttpInternalServerError)
	//	return
	//}
	//if len(indexTest) == 0 {
	//	err = container.Cache.Set(c, "snow_index_test", "snow", 5*time.Second)
	//	if err != nil {
	//		xlogger.Errorf("redis set err: %v, request_id: %s", err.Error(), requestId)
	//		xresponse.FailByError(c, e.HttpInternalServerError)
	//		return
	//	}
	//	xlogger.Info("redis set test success")
	//} else {
	//	xlogger.Info("redis get test success")
	//}

	xresponse.Success(c, gin.H{
		"client_ip":  c.ClientIP(),
		"random_str": str.RandStr(10, str.LowerFlag|str.UpperFlag|str.DigitFlag),
	})
	// Json会返回data，code，msg所有
	//xresponse.Json(c, 200021, "定制化信息", nil)
	//xresponse.JsonByError(c, e.OK, nil)
	//
	// Success固定了code跟msg，fail不会返回data，只会返回code跟msg
	//xresponse.Success(c, gin.H{"name": "test", "age": 12})
	//xresponse.Fail(c, 1001, "")
	//xresponse.FailByError(c, e.OK)
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
		xlogger.Errorf("db check err: %v", err.Error())
		c.JSON(503, gin.H{
			"status": "not ready",
			"error":  err.Error(),
		})
		return
	}
	// redis检查
	if _, err := container.GetRDB().Ping(ctx).Result(); err != nil {
		xlogger.Errorf("redis check err: %v", err.Error())
		c.JSON(503, gin.H{
			"status": "not ready",
			"error":  err.Error(),
		})
		return
	}
	c.JSON(200, gin.H{"status": "ready"})
}
