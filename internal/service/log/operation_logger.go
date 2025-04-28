package log

import (
	"context"
	"encoding/json"
	"snowgo/internal/dal/model"
	"snowgo/internal/dal/query"
	"snowgo/internal/dal/repo"
)

// OperationLogRepo 定义opt log相关db操作接口
type OperationLogRepo interface {
	TransactionCreate(ctx context.Context, tx *query.Query, operationLog *model.OperationLog) (*model.OperationLog, error)
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
func (o *OperationLogService) CreateOperationLog(ctx context.Context, tx *query.Query, input OperationLogInput) error {
	beforeJSON := ""
	afterJSON := ""

	if input.BeforeData != nil {
		if b, err := json.Marshal(input.BeforeData); err == nil {
			beforeJSON = string(b)
		}
	}
	if input.AfterData != nil {
		if b, err := json.Marshal(input.AfterData); err == nil {
			afterJSON = string(b)
		}
	}

	log := &model.OperationLog{
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

	_, err := o.operationLogDao.TransactionCreate(ctx, tx, log)
	return err
}
