package dal

import "snowgo/internal/dal/model"

func GetQueryModels() []interface{} {
	return []interface{}{
		&model.SysDictItem{},
		&model.SysDict{},
		&model.SysMenu{},
		&model.SysOperationLog{},
		&model.SysRoleMenu{},
		&model.SysRole{},
		&model.SysUserRole{},
		&model.SysUser{},
	}
}
