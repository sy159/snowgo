package system

import (
	"github.com/gin-gonic/gin"
	"snowgo/internal/constants"
	"snowgo/internal/di"
	"snowgo/internal/service/log"
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
	ResourceID   int32  `json:"resource_id"`
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
	var logListReq log.OperationLogCondition
	if err := c.ShouldBindQuery(&logListReq); err != nil {
		xresponse.Fail(c, e.HttpBadRequest.GetErrCode(), err.Error())
		return
	}

	xlogger.Infof("get operation log list: %+v", logListReq)
	if logListReq.Offset < 0 {
		xresponse.FailByError(c, e.OffsetErrorRequests)
		return
	}
	if logListReq.Limit < 0 {
		xresponse.FailByError(c, e.LimitErrorRequests)
		return
	} else if logListReq.Limit == 0 {
		logListReq.Limit = constants.DefaultLimit
	}

	container := di.GetSystemContainer(c)
	res, err := container.OperationLogService.GetOperationLogList(c, &logListReq)
	if err != nil {
		xlogger.Errorf("get operation log list is err: %+v", err)
		xresponse.Fail(c, e.HttpInternalServerError.GetErrCode(), err.Error())
		return
	}
	logList := make([]*OperationLogInfo, 0, len(res.List))
	for _, operationLog := range res.List {
		logList = append(logList, &OperationLogInfo{
			ID:           operationLog.ID,
			OperatorID:   operationLog.OperatorID,
			OperatorName: operationLog.OperatorName,
			OperatorType: *operationLog.OperatorType,
			Resource:     operationLog.Resource,
			ResourceID:   operationLog.ResourceID,
			TraceID:      *operationLog.TraceID,
			Action:       *operationLog.Action,
			BeforeData:   *operationLog.BeforeData,
			AfterData:    *operationLog.AfterData,
			Description:  *operationLog.Description,
			IP:           *operationLog.IP,
			CreatedAt:    operationLog.CreatedAt.Format("2006-01-02 15:04:05.000"),
		})
	}
	xresponse.Success(c, &OperationLogList{
		Total: res.Total,
		List:  logList,
	})
}
