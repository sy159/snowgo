package xerror

import (
	"encoding/json"
	"fmt"
	"sync"
)

const (
	CategoryHttp      = "http"       // HTTP 协议级状态
	CategorySystem    = "system"     // 基础设施级错误（限流、参数越权等）
	CategoryAuth      = "admin_auth" // Admin 认证相关
	CategoryAdminUser = "admin_user" // Admin 用户相关
	CategoryAdminMenu = "admin_menu" // Admin 菜单相关
	CategoryAdminRole = "admin_role" // Admin 角色相关
	CategoryAdminDict = "admin_dict" // Admin 字典相关
	CategoryAdminLog  = "admin_log"  // Admin 操作日志相关
)

var (
	// 常用状态码

	OK                      = NewCode(CategoryHttp, 0, "success")                 // 自定义正常
	HttpOK                  = NewCode(CategoryHttp, 200, "ok")                    // http状态正常
	HttpNoContent           = NewCode(CategoryHttp, 204, "No Content")            // 无内容
	HttpMovedPermanently    = NewCode(CategoryHttp, 301, "Moved Permanently")     // 永久重定向
	HttpFound               = NewCode(CategoryHttp, 302, "Found")                 // 临时重定向
	HttpBadRequest          = NewCode(CategoryHttp, 400, "Bad Request")           // 请求数据有问题
	HttpUnauthorized        = NewCode(CategoryHttp, 401, "Unauthorized")          // 用户未认证或认证失败
	HttpForbidden           = NewCode(CategoryHttp, 403, "Forbidden")             // 用户无权限访问
	HttpNotFound            = NewCode(CategoryHttp, 404, "Not Found")             // 请求的资源不存在
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
	TimeFormatError     = NewCode(CategorySystem, 20105, "时间格式错误，应为yyyy-MM-dd HH:mm:ss")
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
	UserNotFound          = NewCode(CategoryAdminUser, 10201, "用户不存在")
	UserCreateError       = NewCode(CategoryAdminUser, 10202, "用户创建失败")
	UserUpdateError       = NewCode(CategoryAdminUser, 10203, "用户更新失败")
	UserDeleteError       = NewCode(CategoryAdminUser, 10204, "用户删除失败")
	UserNameTelEmptyError = NewCode(CategoryAdminUser, 10205, "用户名或电话不能为空")
	UserNameTelExistError = NewCode(CategoryAdminUser, 10206, "用户名或电话不能重复")
	PwdError              = NewCode(CategoryAdminUser, 10207, "密码须为 6–32 位，并至少包含数字、字母、特殊符号（.!@#$%^&*?_~-）中的两种类型")
	UserListError         = NewCode(CategoryAdminUser, 10208, "用户列表获取失败")
	UserInfoError         = NewCode(CategoryAdminUser, 10209, "用户信息获取失败")
	ResetPwdError         = NewCode(CategoryAdminUser, 10210, "重置密码失败")
	UserPermissionError   = NewCode(CategoryAdminUser, 10211, "用户权限获取失败")
	UserRoleNotExist      = NewCode(CategoryAdminUser, 10212, "设置的角色不存在")

	// MenuNotFound 菜单权限相关  102 21 - 102 39
	MenuNotFound      = NewCode(CategoryAdminMenu, 10221, "菜单不存在")
	MenuCreateError   = NewCode(CategoryAdminMenu, 10222, "菜单创建失败")
	MenuUpdateError   = NewCode(CategoryAdminMenu, 10223, "菜单更新失败")
	MenuDeleteError   = NewCode(CategoryAdminMenu, 10224, "菜单删除失败")
	MenuListError     = NewCode(CategoryAdminMenu, 10225, "菜单列表获取失败")
	MenuPermsExist    = NewCode(CategoryAdminMenu, 10226, "权限标识已存在")
	MenuPathExist     = NewCode(CategoryAdminMenu, 10227, "菜单路径已存在")
	MenuParentInvalid = NewCode(CategoryAdminMenu, 10228, "父级菜单不存在")
	MenuParentSelf    = NewCode(CategoryAdminMenu, 10229, "父级菜单不能是自己")
	MenuHasChildren   = NewCode(CategoryAdminMenu, 10230, "存在子菜单，无法删除")
	MenuUsedByRole    = NewCode(CategoryAdminMenu, 10231, "该菜单权限已被使用，无法删除")
	MenuIDInvalid     = NewCode(CategoryAdminMenu, 10232, "菜单ID无效")

	// RoleNotFound 角色相关 102 41 - 102 59
	RoleNotFound     = NewCode(CategoryAdminRole, 10241, "角色不存在")
	RoleCreateError  = NewCode(CategoryAdminRole, 10242, "角色创建失败")
	RoleUpdateError  = NewCode(CategoryAdminRole, 10243, "角色更新失败")
	RoleDeleteError  = NewCode(CategoryAdminRole, 10244, "角色删除失败")
	RoleListError    = NewCode(CategoryAdminRole, 10245, "角色列表获取失败")
	RoleInfoError    = NewCode(CategoryAdminRole, 10246, "角色获取失败")
	RoleCodeExist    = NewCode(CategoryAdminRole, 10247, "角色编码已存在")
	RoleUsed         = NewCode(CategoryAdminRole, 10248, "该角色已被使用，无法删除")
	RoleIDInvalid    = NewCode(CategoryAdminRole, 10249, "角色ID无效")
	RoleMenuNotExist = NewCode(CategoryAdminRole, 10250, "设置的菜单不存在")
)

// 业务system相关 103开头
var (
	// LogListError 日志相关 103 01 - 103 10
	LogListError = NewCode(CategoryAdminLog, 10301, "操作日志列表获取失败")

	// DictNotFound 字典相关 103 11 - 103 30
	DictNotFound           = NewCode(CategoryAdminDict, 10311, "字典不存在")
	DictListError          = NewCode(CategoryAdminDict, 10312, "字典列表获取失败")
	DictCodeExistError     = NewCode(CategoryAdminDict, 10313, "字典编码已存在")
	DictCreateError        = NewCode(CategoryAdminDict, 10314, "字典创建失败")
	DictUpdateError        = NewCode(CategoryAdminDict, 10315, "字典更新失败")
	DictDeleteError        = NewCode(CategoryAdminDict, 10316, "字典删除失败")
	DictItemListError      = NewCode(CategoryAdminDict, 10317, "字典枚举列表获取失败")
	DictCodeItemExistError = NewCode(CategoryAdminDict, 10318, "字典枚举编码已存在")
	DictItemNotFound       = NewCode(CategoryAdminDict, 10319, "字典枚举不存在")
	DictItemCreateError    = NewCode(CategoryAdminDict, 10320, "字典枚举创建失败")
	DictItemUpdateError    = NewCode(CategoryAdminDict, 10321, "字典枚举更新失败")
	DictItemDeleteError    = NewCode(CategoryAdminDict, 10322, "字典枚举删除失败")
)

// Code 错误码接口，错误码一旦创建即为不可变常量
type Code interface {
	i() // 避免被其他包实现
	error
	GetErrCode() int
	GetErrMsg() string
	GetCategory() string
	ToString() string
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

	if _, exists := registry[errCode]; exists {
		panic(fmt.Sprintf("duplicate error code registered: %d", errCode))
	}

	codeInfo := &code{
		ErrCode:  errCode,
		ErrMsg:   errMsg,
		Category: category,
	}
	registry[errCode] = codeInfo
	return codeInfo
}

// GetCodes 返回所有注册的错误码
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

// Error 实现 error 接口
func (c *code) Error() string {
	return c.ErrMsg
}

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

// ToString 返回 JSON 格式的错误详情
func (c *code) ToString() string {
	raw, err := json.Marshal(c)
	if err != nil {
		return fmt.Sprintf(`{"code":%d,"msg":%q,"category":%q}`, c.ErrCode, c.ErrMsg, c.Category)
	}
	return string(raw)
}

// BizError 携带 xerror Code 的业务错误，Service 层用 NewBizError 定义 sentinel
// API 层用 errors.As 提取 Code 统一响应，无需逐个 errors.Is 映射
type BizError struct {
	Code  Code
	cause error
}

// NewBizError 创建携带 xerror Code 的业务错误
func NewBizError(code Code) *BizError {
	return &BizError{Code: code}
}

// WrapBizError 创建携带 xerror Code 并包装底层错误的业务错误
func WrapBizError(code Code, cause error) *BizError {
	return &BizError{Code: code, cause: cause}
}

func (e *BizError) Error() string {
	if e.cause != nil {
		return e.Code.GetErrMsg() + ": " + e.cause.Error()
	}
	return e.Code.GetErrMsg()
}

func (e *BizError) Unwrap() error {
	return e.cause
}
