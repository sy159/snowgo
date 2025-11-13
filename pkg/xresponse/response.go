package xresponse

import (
	"net/http"
	"snowgo/pkg/xauth"
	e "snowgo/pkg/xerror"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	BizCode = "biz_code"
	BizMsg  = "biz_msg"
)

// String 字符串返回
func String(c *gin.Context, res string) {
	c.Set(BizCode, 0)
	c.Set(BizMsg, "")
	c.String(http.StatusOK, res)
}

// Json 统一处理格式，返回包含data
func Json(c *gin.Context, code int, msg string, data interface{}) {
	if data == nil {
		data = struct{}{}
	}
	c.Set(BizCode, code)
	c.Set(BizMsg, msg)
	c.JSON(http.StatusOK, gin.H{
		"code":      code,
		"msg":       msg,
		"data":      data,
		"timestamp": time.Now().UnixMilli(),
		"trace_id":  c.GetString(xauth.XTraceId),
	})
}

// JsonByError 统一处理格式,参数为e.Code类型，data返回
func JsonByError(c *gin.Context, code e.Code, data interface{}) {
	Json(c, code.GetErrCode(), code.GetErrMsg(), data)
}

// Success 成功返回
func Success(c *gin.Context, data interface{}) {
	Json(c, e.OK.GetErrCode(), e.OK.GetErrMsg(), data)
}

// Fail 请求异常返回，只返回code跟msg，不返回data
func Fail(c *gin.Context, errCode int, errMsg string) {
	Json(c, errCode, errMsg, nil)
}

// FailByError 请求异常返回,参数为e.Code类型，只返回code跟msg，不返回data
func FailByError(c *gin.Context, code e.Code) {
	Json(c, code.GetErrCode(), code.GetErrMsg(), nil)
}
