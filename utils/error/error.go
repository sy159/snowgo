package e

import "encoding/json"

var (
	// 常用状态码

	OK                      = NewCode(0, "success")                 // 自定义正常
	HttpOK                  = NewCode(200, "ok")                    // http状态正常
	HttpNoContent           = NewCode(204, "No Content")            // 无内容
	HttpMovedPermanently    = NewCode(301, "Moved Permanently")     // 用久重定向
	HttpFound               = NewCode(302, "Found")                 // 临时重定向
	HttpBadRequest          = NewCode(400, "Bad Request")           // 请求数据有问题
	HttpUnauthorized        = NewCode(401, "Unauthorized")          // 用户未认证或认证失败
	HttpForbidden           = NewCode(403, "Forbidden")             // 用户未认证或认证失败
	HttpNotFound            = NewCode(404, "Not Found")             // 用户未认证或认证失败
	HttpInternalServerError = NewCode(500, "Internal Server Error") // 服务器异常
	HttpBadGateway          = NewCode(502, "Bad Gateway")           // 网关错误
	HttpServiceUnavailable  = NewCode(503, "Service Unavailable")   // 服务器暂时处于超负载或正在进行停机维护，现在无法处理请求
	HttpGatewayTimeout      = NewCode(504, "Gateway Timeout")       // 网关超时

)

/*
	后面的状态码为5位数，自己根据情况定制。。。
	第一位 错误级别(1 开头表示系统相关；2 表示业务相关 等)
	第二三位表示具体模块(比如：用户01 配送员02 商家03 订单04等)
	第四五位表示模块下具体错误(比如：用户名不存在01 密码错误02 用户不存在03等)
*/

// auth相关  认证相关为101开头
var (
	TokenNotFound        = NewCode(10101, "token不能为空")
	TokenIncorrectFormat = NewCode(10102, "token格式错误")
	TokenInvalid         = NewCode(10103, "token无效")
	TokenTypeError       = NewCode(10104, "token类型必须为access")
	TokenExpired         = NewCode(10105, "token已过期")
)

type Code interface {
	i() // 避免被其他包实现
	GetErrCode() int
	GetErrMsg() string
	ToString() string
}

type code struct {
	ErrCode int    `json:"code"`
	ErrMsg  string `json:"msg"`
}

// NewCode 构造错误code
func NewCode(errCode int, errMsg string) Code {
	return &code{
		ErrCode: errCode,
		ErrMsg:  errMsg,
	}
}

func (c *code) i() {}

// GetErrCode 获取错误code
func (c *code) GetErrCode() int {
	return c.ErrCode
}

// GetErrMsg 获取错误信息
func (c *code) GetErrMsg() string {
	return c.ErrMsg
}

// ToString 返回 JSON 格式的错误详情
func (c *code) ToString() string {
	raw, _ := json.Marshal(&c)
	return string(raw)
}
