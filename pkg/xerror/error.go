package xerror

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

// 系统相关 101开头
var (
	TooManyRequests     = NewCode(10101, "Too Many Requests")
	KeyTooManyRequests  = NewCode(10102, "因为访问频繁，你已经被限制访问，稍后重试")
	OffsetErrorRequests = NewCode(10103, "offset必须大于等于0")
	LimitErrorRequests  = NewCode(10104, "limit必须大于0")
)

// auth相关  认证相关为102开头
var (
	TokenNotFound        = NewCode(10201, "token不能为空")
	TokenIncorrectFormat = NewCode(10202, "token格式错误")
	TokenInvalid         = NewCode(10203, "token无效")
	TokenTypeError       = NewCode(10204, "token类型必须为access")
	TokenExpired         = NewCode(10205, "token已过期")
	TokenError           = NewCode(10206, "token异常")
	TokenUseDError       = NewCode(10207, "token已被删除")
	LoginLocked          = NewCode(10208, "登录失败次数过多，请稍后再试")
)

// account相关 103开头
var (
	// UserNotFound 用户相关
	UserNotFound          = NewCode(10301, "用户不存在")
	UserCreateError       = NewCode(10302, "用户创建失败")
	UserUpdateError       = NewCode(10303, "用户更新失败")
	UserDeleteError       = NewCode(10304, "用户更新失败")
	UserNameTelEmptyError = NewCode(10305, "用户名或电话不能为空")
	UserNameTelExistError = NewCode(10306, "用户名或电话已存在")
	AuthError             = NewCode(10307, "用户名或密码错误")
	PwdError              = NewCode(10308, "密码须为 6–32 位，并至少包含数字、字母、特殊符号（.!@#$%^&*?_~-）中的两种类型")

	// MenuNotFound 菜单权限相关
	MenuNotFound    = NewCode(10311, "菜单不存在")
	MenuCreateError = NewCode(10312, "菜单创建失败")
	MenuUpdateError = NewCode(10313, "菜单更新失败")

	// RoleNotFound 角色相关
	RoleNotFound    = NewCode(10321, "角色不存在")
	RoleCreateError = NewCode(10322, "角色创建失败")
	RoleUpdateError = NewCode(10323, "角色更新失败")
	RoleDeleteError = NewCode(10324, "角色删除失败")
)

type Code interface {
	i() // 避免被其他包实现
	GetErrCode() int
	GetErrMsg() string
	ToString() string
	SetErrCode(int)
	SetErrMsg(string)
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

// SetErrCode 设置错误code
func (c *code) SetErrCode(errCode int) {
	c.ErrCode = errCode
}

// SetErrMsg 设置错误信息
func (c *code) SetErrMsg(errMsg string) {
	c.ErrMsg = errMsg
}

// ToString 返回 JSON 格式的错误详情
func (c *code) ToString() string {
	raw, _ := json.Marshal(&c)
	return string(raw)
}
