package dal

import "snowgo/internal/dal/model"

func GetQueryModels() []interface{} {
	return []interface{}{
		&model.User{},
	}
}
