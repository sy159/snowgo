package system

import (
	"context"
	"encoding/json"
	"github.com/pkg/errors"
	"snowgo/internal/constant"
	"snowgo/internal/dal/model"
	"snowgo/internal/dal/query"
	"snowgo/internal/dal/repo"
	daoSystem "snowgo/internal/dao/system"
	"snowgo/pkg/xlogger"
	"time"
)

// OperationLogRepo 定义opt log相关db操作接口
type OperationLogRepo interface {
	TransactionCreate(ctx context.Context, tx *query.Query, operationLog *model.OperationLog) (*model.OperationLog, error)
	GetOperationLogList(ctx context.Context, condition *daoSystem.OperationLogCondition) ([]*model.OperationLog, int64, error)
}

type OperationLogInput struct {
	OperatorID   int32
	OperatorName string
	OperatorType string
	Resource     string
	ResourceID   int32
	Action       string // "Create", "Update", "Delete"
	TraceID      string
	BeforeData   any // 结构体或 map，将会序列化为 JSON
	AfterData    any
	Description  string
	IP           string
}

type OperationLogCondition struct {
	OperatorId   int32  `json:"operator_id" form:"operator_id"`
	OperatorName string `json:"operator_name" form:"operator_name"`
	Resource     string `json:"resource" form:"resource"`
	ResourceID   int32  `json:"resource_id" form:"resource_id"`
	Action       string `json:"action" form:"action"`
	StartTime    string `json:"start_time" form:"start_time"`
	EndTime      string `json:"end_time" form:"end_time"`
	Offset       int32  `json:"offset" form:"offset"`
	Limit        int32  `json:"limit" form:"limit"`
}

type OperationLog struct {
	ID           int64
	OperatorID   int32
	OperatorName string
	OperatorType *string
	Resource     string
	ResourceID   int32
	Action       *string
	TraceID      *string
	BeforeData   *string
	AfterData    *string
	Description  *string
	IP           *string
	CreatedAt    *time.Time
}

type OperationLogList struct {
	List  []*OperationLog
	Total int64
}

type OperationLogService struct {
	db              *repo.Repository
	operationLogDao OperationLogRepo
}

func NewOperationLogService(db *repo.Repository, operationLogDao OperationLogRepo) *OperationLogService {
	return &OperationLogService{
		db:              db,
		operationLogDao: operationLogDao,
	}
}

// CreateOperationLog 记录一条操作日志
func (o *OperationLogService) CreateOperationLog(ctx context.Context, tx *query.Query, input *OperationLogInput) error {
	beforeJSON := "{}"
	afterJSON := "{}"

	if input.BeforeData != nil && input.BeforeData != "" {
		if b, err := json.Marshal(input.BeforeData); err == nil {
			beforeJSON = string(b)
		}
	}
	if input.AfterData != nil && input.AfterData != "" {
		if b, err := json.Marshal(input.AfterData); err == nil {
			afterJSON = string(b)
		}
	}

	operationLog := &model.OperationLog{
		OperatorID:   input.OperatorID,
		OperatorName: input.OperatorName,
		OperatorType: &input.OperatorType,
		Resource:     input.Resource,
		ResourceID:   input.ResourceID,
		Action:       &input.Action,
		TraceID:      &input.TraceID,
		BeforeData:   &beforeJSON,
		AfterData:    &afterJSON,
		Description:  &input.Description,
		IP:           &input.IP,
	}

	_, err := o.operationLogDao.TransactionCreate(ctx, tx, operationLog)
	return err
}

// GetOperationLogList 获取操作日志列表信息
func (o *OperationLogService) GetOperationLogList(ctx context.Context, condition *OperationLogCondition) (*OperationLogList, error) {
	var startTimePtr *time.Time
	var endTimePtr *time.Time
	if condition.StartTime != "" {
		t, err := time.ParseInLocation(constant.TimeFmtWithS, condition.StartTime, time.Local)
		if err != nil {
			return nil, errors.New("start_time格式错误，应为yyyy-MM-dd HH:mm:ss")
		}
		startTimePtr = &t
	}
	if condition.EndTime != "" {
		t, err := time.ParseInLocation(constant.TimeFmtWithS, condition.EndTime, time.Local)
		if err != nil {
			return nil, errors.New("end_time格式错误，应为yyyy-MM-dd HH:mm:ss")
		}
		endTimePtr = &t
	}
	operationLogList, total, err := o.operationLogDao.GetOperationLogList(ctx, &daoSystem.OperationLogCondition{
		OperatorId:   condition.OperatorId,
		OperatorName: condition.OperatorName,
		Resource:     condition.Resource,
		ResourceID:   condition.ResourceID,
		Action:       condition.Action,
		StartTime:    startTimePtr,
		EndTime:      endTimePtr,
		Offset:       condition.Offset,
		Limit:        condition.Limit,
	})
	if err != nil {
		xlogger.ErrorfCtx(ctx, "获取操作日志信息列表异常: %v", err)
		return nil, errors.WithMessage(err, "操作日志信息列表查询失败")
	}
	logList := make([]*OperationLog, 0, len(operationLogList))
	for _, operationLog := range operationLogList {
		logList = append(logList, &OperationLog{
			ID:           operationLog.ID,
			OperatorID:   operationLog.OperatorID,
			OperatorName: operationLog.OperatorName,
			OperatorType: operationLog.OperatorType,
			Resource:     operationLog.Resource,
			ResourceID:   operationLog.ResourceID,
			Action:       operationLog.Action,
			TraceID:      operationLog.TraceID,
			BeforeData:   operationLog.BeforeData,
			AfterData:    operationLog.AfterData,
			Description:  operationLog.Description,
			IP:           operationLog.IP,
			CreatedAt:    operationLog.CreatedAt,
		})
	}
	return &OperationLogList{List: logList, Total: total}, nil
}
