package constants

const (
	CONTAINER = "snowgo.internal.di.container" // 注册的container名
)

// CacheMenuTree 缓存相关key
const (
	CacheMenuTree         = "account:menu_data"
	CacheRolePermsMap     = "account:role_perms_map"
	CacheRefreshJtiPrefix = "jwt:refresh:jti:"
)

// 用户相关
const (
	UserStatusActive   = "Active"
	UserStatusDisabled = "Disabled" //被禁用

	ActiveStatus   = "Active"
	DisabledStatus = "Disabled"

	// MenuTypeDir 菜单相关
	MenuTypeDir  = "Dir"
	MenuTypeMenu = "Menu"
	MenuTypeBtn  = "Btn"
)
