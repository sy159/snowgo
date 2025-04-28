package dal

import "snowgo/internal/dal/model"

func GetQueryModels() []interface{} {
	return []interface{}{
		&model.Menu{},
		&model.OperationLog{},
		&model.RoleMenu{},
		&model.Role{},
		&model.UserRole{},
		&model.User{},
	}
}
