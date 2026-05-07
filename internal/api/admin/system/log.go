package system

import (
	"errors"
	"github.com/gin-gonic/gin"
	"snowgo/internal/constant"
	"snowgo/internal/di"
	"snowgo/internal/service/admin/system"
	common "snowgo/pkg"
	e "snowgo/pkg/xerror"
	"snowgo/pkg/xlogger"
	"snowgo/pkg/xresponse"
)

type OperationLogInfo struct {
	ID           int64  `json:"id"`
	OperatorID   int32  `json:"operator_id"`
	OperatorName string `json:"operator_name"`
	OperatorType string `json:"operator_type"`
	Resource     string `json:"resource"`
	ResourceID   int64  `json:"resource_id"`
	Action       string `json:"action"`
	TraceID      string `json:"trace_id"`
	BeforeData   string `json:"before_data"`
	AfterData    string `json:"after_data"`
	Description  string `json:"description"`
	IP           string `json:"ip"`
	CreatedAt    string `json:"created_at"`
}

type OperationLogList struct {
	List  []*OperationLogInfo `json:"list"`
	Total int64               `json:"total"`
}

// GetOperationLogList 操作日志列表
func GetOperationLogList(c *gin.Context) {
	var logListReq system.OperationLogCondition
	if err := c.ShouldBindQuery(&logListReq); err != nil {
		xresponse.Fail(c, e.HttpBadRequest.GetErrCode(), err.Error())
		return
	}
	ctx := c.Request.Context()

	//xlogger.InfofCtx(ctx, "get operation log list: %+v", logListReq)
	if logListReq.Offset < 0 {
		xresponse.FailByError(c, e.OffsetErrorRequests)
		return
	}
	if logListReq.Limit < 0 {
		xresponse.FailByError(c, e.LimitErrorRequests)
		return
	} else if logListReq.Limit == 0 {
		logListReq.Limit = constant.DefaultLimit
	} else if logListReq.Limit > constant.MaxLimit {
		logListReq.Limit = constant.MaxLimit
	}

	container := di.GetSystemContainer(c)
	res, err := container.OperationLogService.GetOperationLogList(ctx, &logListReq)
	if err != nil {
		var bizErr *e.BizError
		if errors.As(err, &bizErr) {
			xresponse.FailByError(c, bizErr.Code)
			return
		}
		xlogger.ErrorfCtx(ctx, "get operation log list is err: %v", err)
		xresponse.FailByError(c, e.LogListError)
		return
	}
	logList := make([]*OperationLogInfo, 0, len(res.List))
	for _, operationLog := range res.List {
		logList = append(logList, &OperationLogInfo{
			ID:           operationLog.ID,
			OperatorID:   operationLog.OperatorID,
			OperatorName: operationLog.OperatorName,
			OperatorType: common.DerefOrZero(operationLog.OperatorType),
			Resource:     operationLog.Resource,
			ResourceID:   operationLog.ResourceID,
			TraceID:      common.DerefOrZero(operationLog.TraceID),
			Action:       common.DerefOrZero(operationLog.Action),
			BeforeData:   common.DerefOrZero(operationLog.BeforeData),
			AfterData:    common.DerefOrZero(operationLog.AfterData),
			Description:  common.DerefOrZero(operationLog.Description),
			IP:           common.DerefOrZero(operationLog.IP),
			CreatedAt:    operationLog.CreatedAt.Format(constant.TimeFmtWithMS),
		})
	}
	xresponse.Success(c, &OperationLogList{
		Total: res.Total,
		List:  logList,
	})
}

type LoginLogInfo struct {
	ID        int32  `json:"id"`
	UserID    int32  `json:"user_id"`
	Username  string `json:"username"`
	IP        string `json:"ip"`
	Status    bool   `json:"status"`
	Message   string `json:"message"`
	UserAgent string `json:"user_agent"`
	CreatedAt string `json:"created_at"`
}

type LoginLogList struct {
	List  []*LoginLogInfo `json:"list"`
	Total int64           `json:"total"`
}

// GetLoginLogList 登录日志列表
func GetLoginLogList(c *gin.Context) {
	var logListReq system.LoginLogCondition
	if err := c.ShouldBindQuery(&logListReq); err != nil {
		xresponse.Fail(c, e.HttpBadRequest.GetErrCode(), err.Error())
		return
	}
	ctx := c.Request.Context()

	if logListReq.Offset < 0 {
		xresponse.FailByError(c, e.OffsetErrorRequests)
		return
	}
	if logListReq.Limit < 0 {
		xresponse.FailByError(c, e.LimitErrorRequests)
		return
	} else if logListReq.Limit == 0 {
		logListReq.Limit = constant.DefaultLimit
	} else if logListReq.Limit > constant.MaxLimit {
		logListReq.Limit = constant.MaxLimit
	}

	container := di.GetSystemContainer(c)
	res, err := container.LoginLogService.GetLoginLogList(ctx, &logListReq)
	if err != nil {
		var bizErr *e.BizError
		if errors.As(err, &bizErr) {
			xresponse.FailByError(c, bizErr.Code)
			return
		}
		xlogger.ErrorfCtx(ctx, "get login log list is err: %v", err)
		xresponse.FailByError(c, e.LoginLogListError)
		return
	}
	logList := make([]*LoginLogInfo, 0, len(res.List))
	for _, loginLog := range res.List {
		info := &LoginLogInfo{
			ID:       int32(loginLog.ID),
			UserID:   loginLog.UserID,
			Username: loginLog.Username,
			IP:       loginLog.IP,
			Status:   loginLog.Status,
		}
		if loginLog.Message != nil {
			info.Message = *loginLog.Message
		}
		if loginLog.UserAgent != nil {
			info.UserAgent = *loginLog.UserAgent
		}
		if loginLog.CreatedAt != nil {
			info.CreatedAt = loginLog.CreatedAt.Format(constant.TimeFmtWithMS)
		}
		logList = append(logList, info)
	}
	xresponse.Success(c, &LoginLogList{
		Total: res.Total,
		List:  logList,
	})
}
