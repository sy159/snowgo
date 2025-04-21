package account

import (
	"github.com/gin-gonic/gin"
	"snowgo/internal/di"
	"snowgo/internal/service/account"
	e "snowgo/pkg/xerror"
	"snowgo/pkg/xlogger"
	"snowgo/pkg/xresponse"
)

// CreateRole 创建角色
func CreateRole(c *gin.Context) {
	var param account.RoleParam
	if err := c.ShouldBindJSON(&param); err != nil {
		xresponse.Fail(c, e.HttpBadRequest.GetErrCode(), err.Error())
		return
	}
	xlogger.Infof("create role: %+v", param)

	container := di.GetAccountContainer(c)
	roleID, err := container.RoleService.CreateRole(c, &param)
	if err != nil {
		xlogger.Errorf("create role is err: %+v", err)
		xresponse.FailByError(c, e.RoleCreateError)
		return
	}
	xresponse.Success(c, &gin.H{"id": roleID})
}

// UpdateRole 更新角色
func UpdateRole(c *gin.Context) {
	var param account.RoleParam
	if err := c.ShouldBindJSON(&param); err != nil {
		xresponse.Fail(c, e.HttpBadRequest.GetErrCode(), err.Error())
		return
	}
	xlogger.Infof("update role: %+v", param)

	container := di.GetAccountContainer(c)
	err := container.RoleService.UpdateRole(c, &param)
	if err != nil {
		xlogger.Errorf("update role is err: %+v", err)
		xresponse.FailByError(c, e.RoleUpdateError)
		return
	}
	xresponse.Success(c, &gin.H{"id": param.ID})
}

// DeleteRole 删除角色
func DeleteRole(c *gin.Context) {
	var param struct {
		ID int32 `json:"id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&param); err != nil {
		xresponse.FailByError(c, e.HttpBadRequest)
		return
	}
	xlogger.Infof("delete role: %+v", param.ID)

	container := di.GetAccountContainer(c)
	err := container.RoleService.DeleteRole(c, param.ID)
	if err != nil {
		xlogger.Errorf("delete role is err: %+v", err)
		xresponse.FailByError(c, e.RoleDeleteError)
		return
	}
	xresponse.Success(c, &gin.H{"id": param.ID})
}

// GetRoleList 获取角色列表
func GetRoleList(c *gin.Context) {
	var cond account.RoleListCondition
	if err := c.ShouldBindQuery(&cond); err != nil {
		xresponse.Fail(c, e.HttpBadRequest.GetErrCode(), err.Error())
		return
	}
	xlogger.Infof("get role list: %+v", cond)
	if cond.Offset < 0 {
		xresponse.FailByError(c, e.OffsetErrorRequests)
		return
	}
	if cond.Limit < 0 {
		xresponse.FailByError(c, e.LimitErrorRequests)
		return
	} else if cond.Limit == 0 {
		cond.Limit = 10 // 默认长度为10
	}

	container := di.GetAccountContainer(c)
	list, err := container.RoleService.ListRoles(c, &cond)
	if err != nil {
		xlogger.Errorf("get role list is err: %+v", err)
		xresponse.FailByError(c, e.HttpInternalServerError)
		return
	}
	xresponse.Success(c, list)
}

// GetRoleById 获取角色详情（带菜单权限）
func GetRoleById(c *gin.Context) {
	var param struct {
		ID int32 `form:"id" binding:"required"`
	}
	if err := c.ShouldBindQuery(&param); err != nil {
		xresponse.Fail(c, e.HttpBadRequest.GetErrCode(), err.Error())
		return
	}
	xlogger.Infof("get role by id: %+v", param.ID)

	container := di.GetAccountContainer(c)
	role, err := container.RoleService.GetRoleById(c, param.ID)
	if err != nil {
		xlogger.Errorf("get role by id is err: %+v", err)
		xresponse.FailByError(c, e.RoleNotFound)
		return
	}
	xresponse.Success(c, role)
}
