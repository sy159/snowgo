package response

import (
	e "gin-api/utils/error"
	"github.com/gin-gonic/gin"
	"net/http"
)

// String 字符串返回
func String(c *gin.Context, res string) {
	c.String(http.StatusOK, res)
}

// Json 统一处理格式，返回包含data
func Json(c *gin.Context, code int, msg string, data interface{}) {
	if data == nil {
		data = struct{}{}
	}
	c.JSON(http.StatusOK, gin.H{
		"code": code,
		"msg":  msg,
		"data": data,
	})
}

// JsonByError 统一处理格式,参数为e.Code类型，data返回
func JsonByError(c *gin.Context, code e.Code, data interface{}) {
	if data == nil {
		data = struct{}{}
	}
	c.JSON(http.StatusOK, gin.H{
		"code": code.GetErrCode(),
		"msg":  code.GetErrMsg(),
		"data": data,
	})
}

// Success 成功返回
func Success(c *gin.Context, data interface{}) {
	Json(c, e.OK.GetErrCode(), e.OK.GetErrMsg(), data)
}

// Fail 请求异常返回，只返回code跟msg，不返回data
func Fail(c *gin.Context, errCode int, errMsg string) {
	c.JSON(http.StatusOK, gin.H{
		"code": errCode,
		"msg":  errMsg,
	})
}

// FailByError 请求异常返回,参数为e.Code类型，只返回code跟msg，不返回data
func FailByError(c *gin.Context, code e.Code) {
	c.JSON(http.StatusOK, gin.H{
		"code": code.GetErrCode(),
		"msg":  code.GetErrMsg(),
	})
}
