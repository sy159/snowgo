package account

import (
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"snowgo/internal/dal/model"
	"snowgo/internal/dao/account"
	e "snowgo/utils/error"
	"snowgo/utils/logger"
	"snowgo/utils/response"
	"strconv"
)

type User struct {
	Username     string          `json:"username"`
	Password     string          `json:"password"`
	Tel          string          `json:"tel"`
	Sex          string          `json:"sex"` // M表示男，F表示女
	WalletAmount decimal.Decimal `json:"wallet_amount"`
}

// GetUserInfo 用户信息
func GetUserInfo(c *gin.Context) {
	userId := c.Query("id")
	userIdInt, err := strconv.Atoi(userId)
	if err != nil {
		// 转换失败，处理错误
		response.FailByError(c, e.HttpInternalServerError)
		return
	}
	userDao := account.NewUserDao()
	res, err := userDao.GetUserById(c, int32(userIdInt))
	if err != nil {
		response.Fail(c, e.UserNotFound.GetErrCode(), err.Error())
		return
	}
	response.Success(c, res)
}

func CreateUser(c *gin.Context) {
	var user User
	if err := c.ShouldBindJSON(&user); err != nil {
		response.FailByError(c, e.HttpInternalServerError)
		return
	}
	logger.Info("this is test info logo")
	userDao := account.NewUserDao()
	_, err := userDao.CreateUser(c, &model.User{
		Username:     &user.Username,
		Password:     &user.Password,
		Tel:          user.Tel,
		Sex:          &user.Sex,
		WalletAmount: &user.WalletAmount,
	})
	if err != nil {
		response.Fail(c, e.UserCreateError.GetErrCode(), err.Error())
		return
	}
	response.Success(c, nil)
}
