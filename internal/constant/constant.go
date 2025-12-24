package constant

// 常用
const (
	DefaultLimit = 10 // 分页默认10条数据

	TimeFmtWithMS = "2006-01-02 15:04:05.000"
	TimeFmtWithS  = "2006-01-02 15:04:05"
	TimeFmtWithM  = "2006-01-02 15:04"
	TimeFmtWithH  = "2006-01-02 15"

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
	ResourceDict = "Dict"
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
