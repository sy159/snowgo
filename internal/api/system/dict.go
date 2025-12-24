package system

import (
	"errors"
	"github.com/gin-gonic/gin"
	"snowgo/internal/constant"
	"snowgo/internal/di"
	"snowgo/internal/service/system"
	e "snowgo/pkg/xerror"
	"snowgo/pkg/xlogger"
	"snowgo/pkg/xresponse"
)

type DictInfo struct {
	ID          int32  `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type DictList struct {
	List  []*DictInfo `json:"list"`
	Total int64       `json:"total"`
}

type ItemInfo struct {
	ID          int32  `json:"id"`
	ItemName    string `json:"item_name"`   // 枚举显示名称
	ItemCode    string `json:"item_code"`   // 枚举值编码
	Status      string `json:"status"`      // 状态：Active 启用，Disabled 禁用
	SortOrder   int32  `json:"sort_order"`  // 排序号
	Description string `json:"description"` // 描述
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

func GetDictList(c *gin.Context) {
	var dictListReq system.DictListCondition
	if err := c.ShouldBindQuery(&dictListReq); err != nil {
		xresponse.Fail(c, e.HttpBadRequest.GetErrCode(), err.Error())
		return
	}
	if dictListReq.Offset < 0 {
		xresponse.FailByError(c, e.OffsetErrorRequests)
		return
	}
	if dictListReq.Limit < 0 {
		xresponse.FailByError(c, e.LimitErrorRequests)
		return
	} else if dictListReq.Limit == 0 {
		dictListReq.Limit = constant.DefaultLimit
	}
	ctx := c.Request.Context()

	container := di.GetSystemContainer(c)
	res, err := container.DictService.GetDictList(ctx, &dictListReq)
	if err != nil {
		xlogger.ErrorfCtx(ctx, "get system dict list is err: %v", err)
		xresponse.FailByError(c, e.DictListError)
		return
	}
	dictList := make([]*DictInfo, 0, len(res.List))
	for _, dict := range res.List {
		dictList = append(dictList, &DictInfo{
			ID:          dict.ID,
			Code:        dict.Code,
			Name:        dict.Name,
			Description: *dict.Description,
			CreatedAt:   dict.CreatedAt.Format(constant.TimeFmtWithMS),
			UpdatedAt:   dict.UpdatedAt.Format(constant.TimeFmtWithMS),
		})
	}
	xresponse.Success(c, &DictList{
		Total: res.Total,
		List:  dictList,
	})
}

// CreateDict 创建字典
func CreateDict(c *gin.Context) {
	var dict system.DictParam
	if err := c.ShouldBindJSON(&dict); err != nil {
		xresponse.Fail(c, e.HttpBadRequest.GetErrCode(), err.Error())
		return
	}

	ctx := c.Request.Context()

	container := di.GetSystemContainer(c)
	dictId, err := container.DictService.CreateDict(ctx, &dict)
	if err != nil {
		if errors.Is(err, system.ErrDictCodeExist) {
			xresponse.FailByError(c, e.DictCodeExistError)
			return
		}
		xlogger.ErrorfCtx(ctx, "create system dict is err: %v", err)
		xresponse.FailByError(c, e.DictCreateError)
		return
	}
	xresponse.Success(c, &gin.H{"id": dictId})
}

// UpdateDict 更新字典
func UpdateDict(c *gin.Context) {
	var dict system.DictParam
	if err := c.ShouldBindJSON(&dict); err != nil {
		xresponse.Fail(c, e.HttpBadRequest.GetErrCode(), err.Error())
		return
	}

	ctx := c.Request.Context()

	container := di.GetSystemContainer(c)
	dictId, err := container.DictService.UpdateDict(ctx, &dict)
	if err != nil {
		if errors.Is(err, system.ErrDictCodeNotFound) {
			xresponse.FailByError(c, e.DictNotFound)
			return
		}
		if errors.Is(err, system.ErrDictCodeExist) {
			xresponse.FailByError(c, e.DictCodeExistError)
			return
		}
		xlogger.ErrorfCtx(ctx, "update system dict is err: %v", err)
		xresponse.FailByError(c, e.DictUpdateError)
		return
	}
	xresponse.Success(c, &gin.H{"id": dictId})
}

// DeleteDictById 用户删除
func DeleteDictById(c *gin.Context) {
	var param struct {
		ID int32 `json:"id" uri:"id" form:"id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&param); err != nil {
		xresponse.Fail(c, e.HttpBadRequest.GetErrCode(), err.Error())
		return
	}
	if param.ID < 1 {
		xresponse.FailByError(c, e.DictNotFound)
		return
	}
	ctx := c.Request.Context()
	container := di.GetSystemContainer(c)
	err := container.DictService.DeleteById(ctx, param.ID)
	if err != nil {
		xlogger.ErrorfCtx(ctx, "delete dict is err: %v", err)
		xresponse.FailByError(c, e.UserDeleteError)
		return
	}
	xresponse.Success(c, &gin.H{"id": param.ID})
}

// GetItemListByDictCode 根据字典code获取item列表
func GetItemListByDictCode(c *gin.Context) {
	var param struct {
		Code string `json:"code" uri:"code" form:"code" binding:"required"`
	}
	if err := c.ShouldBindQuery(&param); err != nil {
		xresponse.Fail(c, e.HttpBadRequest.GetErrCode(), err.Error())
		return
	}
	ctx := c.Request.Context()

	container := di.GetSystemContainer(c)
	itemList, err := container.DictService.GetItemListByCode(ctx, param.Code)
	if err != nil {
		xlogger.ErrorfCtx(ctx, "get system item list is err: %v", err)
		xresponse.FailByError(c, e.DictItemListError)
		return
	}
	itemInfoList := make([]*ItemInfo, 0, len(itemList))
	for _, item := range itemList {
		itemInfoList = append(itemInfoList, &ItemInfo{
			ID:          item.ID,
			ItemName:    item.ItemName,
			ItemCode:    item.ItemCode,
			Status:      *item.Status,
			SortOrder:   item.SortOrder,
			Description: *item.Description,
			CreatedAt:   item.CreatedAt.Format(constant.TimeFmtWithMS),
			UpdatedAt:   item.UpdatedAt.Format(constant.TimeFmtWithMS),
		})
	}
	xresponse.Success(c, itemInfoList)
}
