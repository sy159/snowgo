package api

import (
	"snowgo/pkg/xlogger"
	"snowgo/pkg/xresponse"
	str "snowgo/pkg/xstr_tool"
	"time"

	"github.com/gin-gonic/gin"
)

// Index 首页
func Index(c *gin.Context) {
	requestId := c.GetString("request_id")
	timestamp := time.Now().Unix()
	xlogger.Infof("index request_id: %s time: %s\n", requestId, time.Now().Format("2006-01-02 15:04:05.000"))

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
		"requestId":  requestId,
		"timestamp":  timestamp,
		"clientIp":   c.ClientIP(),
		"random_str": str.RandStr(8, str.LowerFlag|str.UpperFlag|str.DigitFlag),
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
