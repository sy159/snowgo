package api

import (
	"snowgo/pkg/logger"
	"snowgo/pkg/response"
	str "snowgo/pkg/str_tool"
	"time"

	"github.com/gin-gonic/gin"
)

// Index 首页
func Index(c *gin.Context) {
	requestId := c.GetString("request_id")
	timestamp := time.Now().Unix()
	logger.Infof("index request_id: %s time: %s\n", requestId, time.Now().Format("2006-01-02 15:04:05.000"))

	// redis测试
	//container := di.GetContainer(c)
	//indexTest, err := container.Cache.Get(c, "snow_index_test")
	//if err != nil {
	//	logger.Errorf("redis get err: %v, request_id: %s", err.Error(), requestId)
	//	response.FailByError(c, e.HttpInternalServerError)
	//	return
	//}
	//if len(indexTest) == 0 {
	//	err = container.Cache.Set(c, "snow_index_test", "snow", 5*time.Second)
	//	if err != nil {
	//		logger.Errorf("redis set err: %v, request_id: %s", err.Error(), requestId)
	//		response.FailByError(c, e.HttpInternalServerError)
	//		return
	//	}
	//	logger.Info("redis set test success")
	//} else {
	//	logger.Info("redis get test success")
	//}

	response.Success(c, gin.H{
		"requestId":  requestId,
		"timestamp":  timestamp,
		"clientIp":   c.ClientIP(),
		"random_str": str.RandStr(8, str.LowerFlag|str.UpperFlag|str.DigitFlag),
	})
	// Json会返回data，code，msg所有
	//response.Json(c, 200021, "定制化信息", nil)
	//response.JsonByError(c, e.OK, nil)
	//
	// Success固定了code跟msg，fail不会返回data，只会返回code跟msg
	//response.Success(c, gin.H{"name": "test", "age": 12})
	//response.Fail(c, 1001, "")
	//response.FailByError(c, e.OK)
}
