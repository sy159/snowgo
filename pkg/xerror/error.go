package xerror

import (
	"encoding/json"
	"sync"
)

const (
	CategoryHttp   = "http"
	CategorySystem = "system"
	CategoryAuth   = "auth"
	CategoryUser   = "user"
	CategoryRole   = "role"
	CategoryMenu   = "menu"
)

var (
	// 常用状态码

	OK                      = NewCode(CategoryHttp, 0, "success")                 // 自定义正常
	HttpOK                  = NewCode(CategoryHttp, 200, "ok")                    // http状态正常
	HttpNoContent           = NewCode(CategoryHttp, 204, "No Content")            // 无内容
	HttpMovedPermanently    = NewCode(CategoryHttp, 301, "Moved Permanently")     // 用久重定向
	HttpFound               = NewCode(CategoryHttp, 302, "Found")                 // 临时重定向
	HttpBadRequest          = NewCode(CategoryHttp, 400, "Bad Request")           // 请求数据有问题
	HttpUnauthorized        = NewCode(CategoryHttp, 401, "Unauthorized")          // 用户未认证或认证失败
	HttpForbidden           = NewCode(CategoryHttp, 403, "Forbidden")             // 用户未认证或认证失败
	HttpNotFound            = NewCode(CategoryHttp, 404, "Not Found")             // 用户未认证或认证失败
	HttpInternalServerError = NewCode(CategoryHttp, 500, "Internal Server Error") // 服务器异常
	HttpBadGateway          = NewCode(CategoryHttp, 502, "Bad Gateway")           // 网关错误
	HttpServiceUnavailable  = NewCode(CategoryHttp, 503, "Service Unavailable")   // 服务器暂时处于超负载或正在进行停机维护，现在无法处理请求
	HttpGatewayTimeout      = NewCode(CategoryHttp, 504, "Gateway Timeout")       // 网关超时

)

/*
	后面的状态码为5位数，自己根据情况定制。。。
	第一位 错误级别(2 开头表示系统相关；1 表示业务相关 等)
	第二三位表示具体模块(比如：用户01 配送员02 商家03 订单04等)
	第四五位表示模块下具体错误(比如：用户名不存在01 密码错误02 用户不存在03等)
*/

// 系统相关 2开头
var (
	TooManyRequests     = NewCode(CategorySystem, 20101, "Too Many Requests")
	KeyTooManyRequests  = NewCode(CategorySystem, 20102, "因为访问频繁，你已经被限制访问，稍后重试")
	OffsetErrorRequests = NewCode(CategorySystem, 20103, "offset必须大于等于0")
	LimitErrorRequests  = NewCode(CategorySystem, 20104, "limit必须大于0")
)

// auth相关  认证相关为101开头
var (
	TokenNotFound        = NewCode(CategoryAuth, 10101, "token不能为空")
	TokenIncorrectFormat = NewCode(CategoryAuth, 10102, "token格式错误")
	TokenInvalid         = NewCode(CategoryAuth, 10103, "token无效")
	TokenTypeError       = NewCode(CategoryAuth, 10104, "token类型必须为access")
	TokenExpired         = NewCode(CategoryAuth, 10105, "token已过期")
	TokenError           = NewCode(CategoryAuth, 10106, "token异常")
	TokenUsedError       = NewCode(CategoryAuth, 10107, "token已被使用过，不能重复使用")
	LoginLocked          = NewCode(CategoryAuth, 10108, "登录失败次数过多，请稍后再试")
	AuthError            = NewCode(CategoryAuth, 10109, "用户名或密码错误，认证失败")
)

// account相关 102开头
var (
	// UserNotFound 用户相关 102 01 - 102 19
	UserNotFound          = NewCode(CategoryUser, 10201, "用户不存在")
	UserCreateError       = NewCode(CategoryUser, 10202, "用户创建失败")
	UserUpdateError       = NewCode(CategoryUser, 10203, "用户更新失败")
	UserDeleteError       = NewCode(CategoryUser, 10204, "用户删除失败")
	UserNameTelEmptyError = NewCode(CategoryUser, 10205, "用户名或电话不能为空")
	UserNameTelExistError = NewCode(CategoryUser, 10206, "用户名或电话不能重复")
	PwdError              = NewCode(CategoryUser, 10207, "密码须为 6–32 位，并至少包含数字、字母、特殊符号（.!@#$%^&*?_~-）中的两种类型")
	UserListError         = NewCode(CategoryUser, 10208, "用户列表获取失败")
	UserInfoError         = NewCode(CategoryUser, 10209, "用户信息获取失败")
	ResetPwdError         = NewCode(CategoryUser, 10210, "重置密码失败")
	UserPermissionError   = NewCode(CategoryUser, 10211, "用户权限获取失败")

	// MenuNotFound 菜单权限相关  102 21 - 102 39
	MenuNotFound    = NewCode(CategoryMenu, 10221, "菜单不存在")
	MenuCreateError = NewCode(CategoryMenu, 10222, "菜单创建失败")
	MenuUpdateError = NewCode(CategoryMenu, 10223, "菜单更新失败")
	MenuDeleteError = NewCode(CategoryMenu, 10224, "菜单删除失败")
	MenuListError   = NewCode(CategoryMenu, 10225, "菜单列表获取失败")

	// RoleNotFound 角色相关 102 41 - 102 59
	RoleNotFound    = NewCode(CategoryRole, 10241, "角色不存在")
	RoleCreateError = NewCode(CategoryRole, 10242, "角色创建失败")
	RoleUpdateError = NewCode(CategoryRole, 10243, "角色更新失败")
	RoleDeleteError = NewCode(CategoryRole, 10244, "角色删除失败")
	RoleListError   = NewCode(CategoryRole, 10245, "角色列表获取失败")
	RoleInfoError   = NewCode(CategoryUser, 10246, "角色获取失败")
)

// 业务system相关 103开头
var (
	// LogListError 日志相关 103 01 - 103 10
	LogListError = NewCode(CategoryUser, 10301, "操作日志列表获取失败")

	// DictNotFound 字典相关 103 11 - 103 30
	DictNotFound           = NewCode(CategorySystem, 10311, "字典不存在")
	DictListError          = NewCode(CategorySystem, 10312, "字典列表获取失败")
	DictCodeExistError     = NewCode(CategorySystem, 10313, "字典编码已存在")
	DictCreateError        = NewCode(CategorySystem, 10314, "字典创建失败")
	DictUpdateError        = NewCode(CategorySystem, 10315, "字典更新失败")
	DictItemListError      = NewCode(CategorySystem, 10316, "字典枚举列表获取失败")
	DictCodeItemExistError = NewCode(CategorySystem, 10317, "字典枚举编码已存在")
)

type Code interface {
	i() // 避免被其他包实现
	GetErrCode() int
	GetErrMsg() string
	GetCategory() string
	ToString() string
	SetErrCode(int)
	SetErrMsg(string)
	SetCategory(string)
}

type code struct {
	ErrCode  int    `json:"code"`
	ErrMsg   string `json:"msg"`
	Category string `json:"category"`
}

var (
	registryMu sync.RWMutex
	registry   = make(map[int]Code)
)

// NewCode 构造错误code
func NewCode(category string, errCode int, errMsg string) Code {
	registryMu.Lock()
	defer registryMu.Unlock()

	codeInfo := &code{
		ErrCode:  errCode,
		ErrMsg:   errMsg,
		Category: category,
	}
	registry[errCode] = codeInfo
	return codeInfo
}

func GetCodes() []Code {
	registryMu.RLock()
	defer registryMu.RUnlock()

	list := make([]Code, 0, len(registry))
	for _, v := range registry {
		list = append(list, v)
	}
	return list
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

// GetCategory 获取错误类别
func (c *code) GetCategory() string {
	return c.Category
}

// SetErrCode 设置错误code
func (c *code) SetErrCode(errCode int) {
	c.ErrCode = errCode
}

// SetErrMsg 设置错误信息
func (c *code) SetErrMsg(errMsg string) {
	c.ErrMsg = errMsg
}

// SetCategory 设置错误类别
func (c *code) SetCategory(category string) {
	c.Category = category
}

// ToString 返回 JSON 格式的错误详情
func (c *code) ToString() string {
	raw, _ := json.Marshal(&c)
	return string(raw)
}
