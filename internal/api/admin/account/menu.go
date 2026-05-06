package account

import (
	"errors"
	"github.com/gin-gonic/gin"
	"snowgo/internal/di"
	"snowgo/internal/service/admin/account"
	e "snowgo/pkg/xerror"
	"snowgo/pkg/xgin"
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
		var bizErr *e.BizError
		if errors.As(err, &bizErr) {
			xresponse.FailByError(c, bizErr.Code)
			return
		}
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
		var bizErr *e.BizError
		if errors.As(err, &bizErr) {
			xresponse.FailByError(c, bizErr.Code)
			return
		}
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
		var bizErr *e.BizError
		if errors.As(err, &bizErr) {
			xresponse.FailByError(c, bizErr.Code)
			return
		}
		xlogger.ErrorfCtx(ctx, "get menu list is err: %v", err)
		xresponse.FailByError(c, e.MenuListError)
		return
	}
	xresponse.Success(c, res)
}

// DeleteMenuById 菜单删除
func DeleteMenuById(c *gin.Context) {
	id := xgin.ParsePathID(c)
	if id < 1 {
		xresponse.FailByError(c, e.MenuNotFound)
		return
	}
	ctx := c.Request.Context()
	container := di.GetAccountContainer(c)
	err := container.MenuService.DeleteMenuById(ctx, int32(id))
	if err != nil {
		var bizErr *e.BizError
		if errors.As(err, &bizErr) {
			xresponse.FailByError(c, bizErr.Code)
			return
		}
		xlogger.ErrorfCtx(ctx, "delete menu is err: %v", err)
		xresponse.FailByError(c, e.MenuDeleteError)
		return
	}
	xresponse.Success(c, &gin.H{"id": id})
}
