package account

import (
	"github.com/gin-gonic/gin"
	"snowgo/internal/constants"
	"snowgo/internal/di"
	"snowgo/internal/service/account"
	e "snowgo/pkg/xerror"
	"snowgo/pkg/xlogger"
	"snowgo/pkg/xresponse"
)

type UserListInfo struct {
	ID        int32  `json:"id"`
	Username  string `json:"username"`
	Tel       string `json:"tel"`
	Nickname  string `json:"nickname"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type UserInfo struct {
	ID        int32       `json:"id"`
	Username  string      `json:"username"`
	Tel       string      `json:"tel"`
	Nickname  string      `json:"nickname"`
	Status    string      `json:"status"`
	CreatedAt string      `json:"created_at"`
	UpdatedAt string      `json:"updated_at"`
	RoleList  []*UserRole `json:"role_list"`
}

type UserRole struct {
	ID   int32  `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
}

type UserList struct {
	List  []*UserListInfo `json:"list"`
	Total int64           `json:"total"`
}

// CreateUser 创建用户
func CreateUser(c *gin.Context) {
	var user account.UserParam
	if err := c.ShouldBindJSON(&user); err != nil {
		xresponse.Fail(c, e.HttpBadRequest.GetErrCode(), err.Error())
		return
	}
	xlogger.Infof("create user: (用户名: %s, 电话: %s, 昵称: %s, 角色: %v)",
		user.Username, user.Tel, user.Nickname, user.RoleIds)

	// 可以额外校验
	if user.Username == "" || user.Tel == "" {
		xresponse.FailByError(c, e.UserNameTelEmptyError)
		return
	}

	container := di.GetContainer(c)
	userId, err := container.UserService.CreateUser(c, &user)
	if err != nil {
		if err.Error() == e.UserNameTelExistError.GetErrMsg() {
			xresponse.FailByError(c, e.UserNameTelExistError)
			return
		}
		xlogger.Errorf("create user info is err: %+v", err)
		xresponse.Fail(c, e.UserCreateError.GetErrCode(), err.Error())
		return
	}
	xresponse.Success(c, &gin.H{"id": userId})
}

// UpdateUser 更新用户
func UpdateUser(c *gin.Context) {
	var user account.UserParam
	if err := c.ShouldBindJSON(&user); err != nil {
		xresponse.Fail(c, e.HttpBadRequest.GetErrCode(), err.Error())
		return
	}
	xlogger.Infof("update user: %+v", user)

	container := di.GetContainer(c)
	userId, err := container.UserService.UpdateUser(c, &user)
	if err != nil {
		if err.Error() == e.UserNameTelExistError.GetErrMsg() {
			xresponse.FailByError(c, e.UserNameTelExistError)
			return
		}
		xlogger.Errorf("update user info is err: %+v", err)
		xresponse.Fail(c, e.UserUpdateError.GetErrCode(), err.Error())
		return
	}
	xresponse.Success(c, &gin.H{"id": userId})
}

// GetUserInfo 用户信息
func GetUserInfo(c *gin.Context) {
	var param struct {
		ID int32 `json:"id" uri:"id" form:"id" binding:"required"`
	}
	if err := c.ShouldBindQuery(&param); err != nil {
		xresponse.Fail(c, e.HttpBadRequest.GetErrCode(), err.Error())
		return
	}

	xlogger.Infof("get user info by id: %v", param.ID)
	container := di.GetContainer(c)
	user, err := container.UserService.GetUserById(c, param.ID)
	if err != nil {
		xlogger.Errorf("get user info is err: %+v", err)
		xresponse.Fail(c, e.UserNotFound.GetErrCode(), err.Error())
		return
	}
	roleList := make([]*UserRole, 0, len(user.RoleList))
	for _, role := range user.RoleList {
		roleList = append(roleList, &UserRole{
			ID:   role.ID,
			Name: role.Name,
			Code: role.Code,
		})
	}

	xresponse.Success(c, &UserInfo{
		ID:        user.ID,
		Username:  user.Username,
		Tel:       user.Tel,
		Nickname:  user.Nickname,
		Status:    user.Status,
		CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05.000"),
		UpdatedAt: user.UpdatedAt.Format("2006-01-02 15:04:05.000"),
		RoleList:  roleList,
	})
}

// GetUserList 用户信息列表
func GetUserList(c *gin.Context) {
	var userListReq account.UserListCondition
	if err := c.ShouldBindQuery(&userListReq); err != nil {
		xresponse.Fail(c, e.HttpBadRequest.GetErrCode(), err.Error())
		return
	}
	xlogger.Infof("get user list: %+v", userListReq)
	if userListReq.Offset < 0 {
		xresponse.FailByError(c, e.OffsetErrorRequests)
		return
	}
	if userListReq.Limit < 0 {
		xresponse.FailByError(c, e.LimitErrorRequests)
		return
	} else if userListReq.Limit == 0 {
		userListReq.Limit = constants.DefaultLimit
	}

	container := di.GetContainer(c)
	res, err := container.UserService.GetUserList(c, &userListReq)
	if err != nil {
		xlogger.Errorf("get user list is err: %+v", err)
		xresponse.Fail(c, e.HttpInternalServerError.GetErrCode(), err.Error())
		return
	}
	userList := make([]*UserListInfo, 0, len(res.List))
	for _, user := range res.List {
		userList = append(userList, &UserListInfo{
			ID:        user.ID,
			Username:  user.Username,
			Tel:       user.Tel,
			Nickname:  user.Nickname,
			Status:    user.Status,
			CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05.000"),
			UpdatedAt: user.UpdatedAt.Format("2006-01-02 15:04:05.000"),
		})
	}
	xresponse.Success(c, &UserList{
		Total: res.Total,
		List:  userList,
	})
}

// DeleteUserById 用户删除
func DeleteUserById(c *gin.Context) {
	var user UserInfo
	if err := c.ShouldBindJSON(&user); err != nil {
		xresponse.FailByError(c, e.HttpBadRequest)
		return
	}
	xlogger.Infof("delete user info by id: %+v", user.ID)
	if user.ID < 1 {
		xresponse.FailByError(c, e.UserNotFound)
		return
	}
	container := di.GetContainer(c)
	err := container.UserService.DeleteById(c, user.ID)
	if err != nil {
		xlogger.Errorf("delete user is err: %+v", err)
		xresponse.Fail(c, e.UserDeleteError.GetErrCode(), err.Error())
		return
	}
	xresponse.Success(c, &gin.H{"id": user.ID})
}

func ResetPwdById(c *gin.Context) {
	var param struct {
		ID       int32  `json:"id" form:"id" binding:"required"`
		Password string `json:"password" form:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&param); err != nil {
		xresponse.Fail(c, e.HttpBadRequest.GetErrCode(), err.Error())
		return
	}
	xlogger.Infof("reset user pwd by id: %+v", param.ID)
	if param.ID < 1 {
		xresponse.FailByError(c, e.UserNotFound)
		return
	}
	container := di.GetContainer(c)
	err := container.UserService.ResetPwdById(c, param.ID, param.Password)
	if err != nil {
		xlogger.Errorf("delete user is err: %+v", err)
		xresponse.Fail(c, e.UserNotFound.GetErrCode(), err.Error())
		return
	}
	xresponse.Success(c, &gin.H{"id": param.ID})
}
