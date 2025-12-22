package account

import (
	"github.com/gin-gonic/gin"
	"snowgo/internal/di"
	"snowgo/internal/service/account"
	e "snowgo/pkg/xerror"
	"snowgo/pkg/xlogger"
	"snowgo/pkg/xresponse"
)

type MenuInfo struct {
	ID        int32       `json:"id"`
	ParentID  int32       `json:"parent_id"`
	MenuType  string      `json:"menu_type"`
	Name      string      `json:"name"`
	Path      string      `json:"path"`
	Icon      string      `json:"icon"`
	Perms     string      `json:"perms"`
	SortOrder int32       `json:"sort_order"`
	Status    string      `json:"status"`
	CreatedAt string      `json:"created_at"`
	UpdatedAt string      `json:"updated_at"`
	Children  []*MenuInfo `json:"children"`
}

// CreateMenu 创建菜单权限
func CreateMenu(c *gin.Context) {
	var menuParam account.MenuParam
	if err := c.ShouldBindJSON(&menuParam); err != nil {
		xresponse.Fail(c, e.HttpBadRequest.GetErrCode(), err.Error())
		return
	}
	ctx := c.Request.Context()

	container := di.GetAccountContainer(c)
	menuId, err := container.MenuService.CreateMenu(ctx, &menuParam)
	if err != nil {
		xlogger.ErrorfCtx(ctx, "create menu is err: %v", err)
		xresponse.FailByError(c, e.MenuCreateError)
		return
	}
	xresponse.Success(c, &gin.H{"id": menuId})
}

// UpdateMenu 更新菜单权限
func UpdateMenu(c *gin.Context) {
	var menuParam account.MenuParam
	if err := c.ShouldBindJSON(&menuParam); err != nil {
		xresponse.Fail(c, e.HttpBadRequest.GetErrCode(), err.Error())
		return
	}
	ctx := c.Request.Context()

	container := di.GetAccountContainer(c)
	err := container.MenuService.UpdateMenu(ctx, &menuParam)
	if err != nil {
		xlogger.ErrorfCtx(ctx, "update menu is err: %v", err)
		xresponse.FailByError(c, e.MenuUpdateError)
		return
	}
	xresponse.Success(c, &gin.H{"id": menuParam.ID})
}

// GetMenuList 菜单信息列表
func GetMenuList(c *gin.Context) {
	container := di.GetAccountContainer(c)
	ctx := c.Request.Context()
	res, err := container.MenuService.GetMenuTree(ctx)
	if err != nil {
		xlogger.ErrorfCtx(ctx, "get user list is err: %v", err)
		xresponse.FailByError(c, e.MenuListError)
		return
	}
	xresponse.Success(c, res)
}

// DeleteMenuById 菜单删除
func DeleteMenuById(c *gin.Context) {
	var menuParam account.MenuInfo
	if err := c.ShouldBindJSON(&menuParam); err != nil {
		xresponse.Fail(c, e.HttpBadRequest.GetErrCode(), err.Error())
		return
	}
	if menuParam.ID < 1 {
		xresponse.FailByError(c, e.MenuNotFound)
		return
	}
	ctx := c.Request.Context()
	container := di.GetAccountContainer(c)
	err := container.MenuService.DeleteMenuById(ctx, menuParam.ID)
	if err != nil {
		xlogger.ErrorfCtx(ctx, "delete menu is err: %v", err)
		xresponse.FailByError(c, e.MenuDeleteError)
		return
	}
	xresponse.Success(c, &gin.H{"id": menuParam.ID})
}
