package constants

const (
	DefaultLimit = 10 // 分页默认10条数据

	CONTAINER      = "snowgo.internal.di.container" // 注册的container名
	ActiveStatus   = "Active"
	DisabledStatus = "Disabled"

	OperatorUser   = "User"
	OperatorSystem = "System"
	OperatorJob    = "Job"
	OperatorApi    = "Api"

	ActionCreate = "Create"
	ActionUpdate = "Update"
	ActionDelete = "Delete"

	ResourceUser = "User"
	ResourceRole = "Role"
	ResourceMenu = "Menu"
)

// CacheMenuTree 缓存相关key
const (
	CacheMenuTree               = "account:menu_data"   // 菜单权限数据缓存key
	CacheMenuTreeExpirationDay  = 15                    // 菜单权限缓存天数
	CacheRolePermsPrefix        = "account:role_perms:" // 角色对应 接口权限key
	CacheRolePermsExpirationDay = 15                    // 角色对应 接口权限缓存天数
	CacheRoleMenuPrefix         = "account:role_menu:"  // 角色对应 菜单权限key
	CacheRoleMenuExpirationDay  = 15                    // 角色对应 接口权限缓存天数
	CacheUserRolePrefix         = "account:user_role:"  // 用户对应角色权限key
	CacheUserRoleExpirationDay  = 15                    // 用户对应角色权限缓存天数
	CacheRefreshJtiPrefix       = "jwt:refresh:jti:"
)

// 用户相关
const (
	UserStatusActive   = "Active"
	UserStatusDisabled = "Disabled" //被禁用

	// MenuTypeDir 菜单相关
	MenuTypeDir  = "Dir"
	MenuTypeMenu = "Menu"
	MenuTypeBtn  = "Btn"
)
