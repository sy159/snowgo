package userDao

import (
	"snowgo/internal/models"
	. "snowgo/utils/database/mysql"
)

func CreateUser(user *models.User) (id uint, err error) {
	res := DB.Create(user)
	if res.Error != nil {
		return 0, res.Error
	}
	return user.ID, nil
}

func GetUserById(userId uint) (models.User, error) {
	u := models.User{}
	res := DB.Find(&u, userId).Limit(1)
	return u, res.Error
}
