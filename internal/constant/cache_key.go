package constant

const (
	// CacheMenuTree 缓存相关key
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
