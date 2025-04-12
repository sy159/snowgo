package account

import (
	"github.com/gin-gonic/gin"
	"snowgo/internal/di"
	"snowgo/internal/service/account"
	e "snowgo/utils/error"
	"snowgo/utils/logger"
	"snowgo/utils/response"
	"strconv"
)

type UserInfo struct {
	ID           int32   `json:"id"`
	Username     string  `json:"username"`
	Tel          string  `json:"tel"`
	Sex          string  `json:"sex"` // M表示男，F表示女
	WalletAmount float64 `json:"wallet_amount"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

type UserList struct {
	List  []*UserInfo `json:"list"`
	Total int64       `json:"total"`
}

// CreateUser 创建用户
func CreateUser(c *gin.Context) {
	var user account.User
	if err := c.ShouldBindJSON(&user); err != nil {
		response.FailByError(c, e.HttpInternalServerError)
		return
	}
	logger.Infof("create user: %+v", user)

	if user.Username == "" || user.Tel == "" {
		response.FailByError(c, e.UserNameTelEmptyError)
		return
	}
	container := di.GetContainer(c)
	userId, err := container.UserService.CreateUser(c, &user)
	if err != nil {
		if err.Error() == e.UserNameTelExistError.GetErrMsg() {
			response.FailByError(c, e.UserNameTelExistError)
			return
		}
		logger.Errorf("create user info is err: %+v", err)
		response.FailByError(c, e.UserCreateError)
		return
	}
	response.Success(c, &gin.H{"id": userId})
}

// GetUserInfo 用户信息
func GetUserInfo(c *gin.Context) {
	userIdStr := c.Query("id")
	userId, err := strconv.Atoi(userIdStr)
	if err != nil {
		// 转换失败，处理错误
		response.FailByError(c, e.HttpInternalServerError)
		return
	}
	logger.Infof("get user info by id: %+v", userId)
	container := di.GetContainer(c)
	user, err := container.UserService.GetUserById(c, int32(userId))
	if err != nil {
		logger.Errorf("get user info is err: %+v", err)
		response.Fail(c, e.UserNotFound.GetErrCode(), err.Error())
		return
	}
	response.Success(c, &UserInfo{
		ID:           user.ID,
		Username:     user.Username,
		Tel:          user.Tel,
		Sex:          user.Sex,
		WalletAmount: user.WalletAmount.InexactFloat64(),
		CreatedAt:    user.CreatedAt.Format("2006-01-02 15:04:05.000"),
		UpdatedAt:    user.UpdatedAt.Format("2006-01-02 15:04:05.000"),
	})
}

// GetUserList 用户信息列表
func GetUserList(c *gin.Context) {
	var userListReq account.UserListCondition
	if err := c.ShouldBindQuery(&userListReq); err != nil {
		response.FailByError(c, e.HttpInternalServerError)
		return
	}
	logger.Infof("get user list: %+v", userListReq)
	if userListReq.Offset < 0 {
		response.FailByError(c, e.OffsetErrorRequests)
		return
	}
	if userListReq.Limit < 0 {
		response.FailByError(c, e.LimitErrorRequests)
		return
	} else if userListReq.Limit == 0 {
		userListReq.Limit = 10 // 默认长度为10
	}

	container := di.GetContainer(c)
	res, err := container.UserService.GetUserList(c, &userListReq)
	if err != nil {
		logger.Errorf("get user list is err: %+v", err)
		response.FailByError(c, e.HttpInternalServerError)
		return
	}
	userList := make([]*UserInfo, 0, len(res.List))
	for _, user := range res.List {
		userList = append(userList, &UserInfo{
			ID:           user.ID,
			Username:     user.Username,
			Tel:          user.Tel,
			Sex:          user.Sex,
			WalletAmount: user.WalletAmount.InexactFloat64(),
			CreatedAt:    user.CreatedAt.Format("2006-01-02 15:04:05.000"),
			UpdatedAt:    user.UpdatedAt.Format("2006-01-02 15:04:05.000"),
		})
	}
	response.Success(c, &UserList{
		Total: res.Total,
		List:  userList,
	})
}

// DeleteUserById 用户删除
func DeleteUserById(c *gin.Context) {
	var user UserInfo
	if err := c.ShouldBindJSON(&user); err != nil {
		response.FailByError(c, e.HttpInternalServerError)
		return
	}
	logger.Infof("delete user info by id: %+v", user.ID)
	if user.ID < 1 {
		response.FailByError(c, e.UserNotFound)
		return
	}
	container := di.GetContainer(c)
	err := container.UserService.DeleteById(c, user.ID)
	if err != nil {
		logger.Errorf("delete user is err: %+v", err)
		response.Fail(c, e.UserNotFound.GetErrCode(), err.Error())
		return
	}
	response.Success(c, &gin.H{"id": user.ID})
}
