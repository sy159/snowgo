package constant

const (
	// PermAccountUserList 账号管理 - 用户管理
	PermAccountUserList     = "account:user:list"      // 查看用户列表
	PermAccountUserDetail   = "account:user:detail"    // 查看用户详情
	PermAccountUserCreate   = "account:user:create"    // 创建用户
	PermAccountUserUpdate   = "account:user:update"    // 更新用户信息
	PermAccountUserDelete   = "account:user:delete"    // 删除用户
	PermAccountUserResetPwd = "account:user:reset_pwd" // 重置用户密码

	// PermAccountRoleList 账号管理 - 角色管理
	PermAccountRoleList   = "account:role:list"   // 查看角色列表
	PermAccountRoleDetail = "account:role:detail" // 查看角色详情
	PermAccountRoleCreate = "account:role:create" // 创建角色
	PermAccountRoleUpdate = "account:role:update" // 更新角色信息
	PermAccountRoleDelete = "account:role:delete" // 删除角色

	// PermAccountMenuList 账号管理 - 菜单管理
	PermAccountMenuList   = "account:menu:list"   // 查看菜单列表
	PermAccountMenuCreate = "account:menu:create" // 创建菜单
	PermAccountMenuUpdate = "account:menu:update" // 更新菜单信息
	PermAccountMenuDelete = "account:menu:delete" // 删除菜单

	// PermSystemOperationLogList 系统管理 - 操作日志管理
	PermSystemOperationLogList = "system:operation-log:list" // 查看操作日志列表

	// PermSystemDictList 系统管理 - 字典管理
	PermSystemDictList   = "system:dict:list"
	PermSystemDictCreate = "system:dict:create"
	PermSystemDictUpdate = "system:dict:update"
	PermSystemDictDelete = "system:dict:delete"
)
