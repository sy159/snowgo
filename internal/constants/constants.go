package constants

const (
	CONTAINER = "snowgo.internal.di.container" // 注册的container名
)

// CacheMenuTree 缓存相关key
const (
	CacheMenuTree               = "account:menu_data"   // 菜单权限数据缓存key
	CacheMenuTreeExpirationDay  = 15                    // 菜单权限缓存天数
	CacheRolePermsPrefix        = "account:role_perms:" // 角色对应 接口权限key
	CacheRolePermsExpirationDay = 15                    // 角色对应 接口权限缓存天数
	CacheRefreshJtiPrefix       = "jwt:refresh:jti:"
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
