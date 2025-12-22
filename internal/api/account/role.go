package account

import (
	"errors"
	"github.com/gin-gonic/gin"
	"snowgo/internal/constant"
	"snowgo/internal/di"
	"snowgo/internal/service/account"
	e "snowgo/pkg/xerror"
	"snowgo/pkg/xlogger"
	"snowgo/pkg/xresponse"
)

// RoleInfo 返回给前端的角色信息
type RoleInfo struct {
	ID          int32  `json:"id"`
	Name        string `json:"name"`
	Code        string `json:"code"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// RoleList 返回角色列表
type RoleList struct {
	List  []*RoleInfo `json:"list"`
	Total int64       `json:"total"`
}

// CreateRole 创建角色
func CreateRole(c *gin.Context) {
	var param account.RoleParam
	if err := c.ShouldBindJSON(&param); err != nil {
		xresponse.Fail(c, e.HttpBadRequest.GetErrCode(), err.Error())
		return
	}
	ctx := c.Request.Context()

	container := di.GetAccountContainer(c)
	roleID, err := container.RoleService.CreateRole(ctx, &param)
	if err != nil {
		xlogger.ErrorfCtx(ctx, "create role is err: %v", err)
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
	ctx := c.Request.Context()

	container := di.GetAccountContainer(c)
	err := container.RoleService.UpdateRole(ctx, &param)
	if err != nil {
		xlogger.ErrorfCtx(ctx, "update role is err: %v", err)
		xresponse.FailByError(c, e.RoleUpdateError)
		return
	}
	xresponse.Success(c, &gin.H{"id": param.ID})
}

// DeleteRole 删除角色
func DeleteRole(c *gin.Context) {
	var param struct {
		ID int32 `json:"id" uri:"id" form:"id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&param); err != nil {
		xresponse.FailByError(c, e.HttpBadRequest)
		return
	}
	ctx := c.Request.Context()

	container := di.GetAccountContainer(c)
	err := container.RoleService.DeleteRole(ctx, param.ID)
	if err != nil {
		xlogger.ErrorfCtx(ctx, "delete role is err: %v", err)
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
	if cond.Offset < 0 {
		xresponse.FailByError(c, e.OffsetErrorRequests)
		return
	}
	if cond.Limit < 0 {
		xresponse.FailByError(c, e.LimitErrorRequests)
		return
	} else if cond.Limit == 0 {
		cond.Limit = constant.DefaultLimit
	}
	ctx := c.Request.Context()

	container := di.GetAccountContainer(c)
	res, err := container.RoleService.ListRoles(ctx, &cond)
	if err != nil {
		xlogger.ErrorfCtx(ctx, "get role list is err: %v", err)
		xresponse.FailByError(c, e.RoleListError)
		return
	}
	roleList := make([]*RoleInfo, 0, len(res.List))
	for _, role := range res.List {
		roleList = append(roleList, &RoleInfo{
			ID:          role.ID,
			Name:        role.Name,
			Code:        role.Code,
			Description: role.Description,
			CreatedAt:   role.CreatedAt.Format(constant.TimeFmtWithMS),
			UpdatedAt:   role.UpdatedAt.Format(constant.TimeFmtWithMS),
		})
	}
	xresponse.Success(c, &RoleList{
		List:  roleList,
		Total: res.Total,
	})
}

// GetRoleById 获取角色详情（带菜单权限）
func GetRoleById(c *gin.Context) {
	var param struct {
		ID int32 `json:"id" uri:"id" form:"id" binding:"required"`
	}
	if err := c.ShouldBindQuery(&param); err != nil {
		xresponse.Fail(c, e.HttpBadRequest.GetErrCode(), err.Error())
		return
	}
	ctx := c.Request.Context()

	container := di.GetAccountContainer(c)
	role, err := container.RoleService.GetRoleById(ctx, param.ID)
	if err != nil {
		if errors.Is(err, account.ErrRoleNotFound) {
			xresponse.FailByError(c, e.RoleNotFound)
			return
		}
		xlogger.ErrorfCtx(ctx, "get role by id is err: %v", err)
		xresponse.FailByError(c, e.RoleInfoError)
		return
	}
	xresponse.Success(c, role)
}
