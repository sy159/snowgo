package api

import (
	e "gin-api/utils/error"
	"gin-api/utils/response"
	"github.com/gin-gonic/gin"
	"time"
)

// Index 首页
func Index(c *gin.Context) {
	//logger.Debug("test")
	//logger.Info("info")
	//logger.Error("error")

	time.Sleep(100 * time.Millisecond)
	// Json会返回data，code，msg所有
	response.Json(c, 200021, "定制化信息", nil)
	response.JsonByError(c, e.OK, nil)

	// Success固定了code跟msg，fail不会返回data，只会返回code跟msg
	response.Success(c, gin.H{"name": "test", "age": 12})
	response.Fail(c, 1001, "")
	response.FailByError(c, e.OK)
}
