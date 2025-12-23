package system

import (
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
	Status      string `json:"status"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type DictList struct {
	List  []*DictInfo `json:"list"`
	Total int64       `json:"total"`
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
		xresponse.FailByError(c, e.LogListError)
		return
	}
	dictList := make([]*DictInfo, 0, len(res.List))
	for _, dict := range res.List {
		dictList = append(dictList, &DictInfo{
			ID:        dict.ID,
			Code:      dict.Code,
			Name:      dict.Name,
			Status:    *dict.Status,
			CreatedAt: dict.CreatedAt.Format(constant.TimeFmtWithMS),
			UpdatedAt: dict.UpdatedAt.Format(constant.TimeFmtWithMS),
		})
	}
	xresponse.Success(c, &DictList{
		Total: res.Total,
		List:  dictList,
	})
}
