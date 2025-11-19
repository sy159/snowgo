package constant

// 常用
const (
	DefaultLimit = 10 // 分页默认10条数据

	CONTAINER = "snowgo.internal.di.container" // 注册的container名

	// ActiveStatus 状态
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

// 用户相关
const (
	UserStatusActive   = "Active"
	UserStatusDisabled = "Disabled" //被禁用

	// MenuTypeDir 菜单相关
	MenuTypeDir  = "Dir"
	MenuTypeMenu = "Menu"
	MenuTypeBtn  = "Btn"
)
