package system

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"snowgo/internal/constant"
	"snowgo/internal/dal/model"
	"snowgo/internal/dal/query"
	"snowgo/internal/dal/repo"
	daoSystem "snowgo/internal/dao/admin/system"
	"snowgo/internal/service/admin/contract"
	"snowgo/pkg/xlogger"
	"strings"
	"time"
)

// OperationLogRepo 定义opt log相关db操作接口
type OperationLogRepo interface {
	Create(ctx context.Context, q *query.Query, operationLog *model.SysOperationLog) (*model.SysOperationLog, error)
	GetOperationLogList(ctx context.Context, condition *daoSystem.OperationLogCondition) ([]*model.SysOperationLog, int64, error)
}

var _ contract.OperationLogWriter = (*OperationLogService)(nil)

var (
	auditDropFields = map[string]struct{}{
		"password":      {},
		"pass":          {},
		"pwd":           {},
		"token":         {},
		"access_token":  {},
		"refresh_token": {},
		"secret":        {},
		"jwt_secret":    {},
	}
)

// OperationLogInput keeps system package callers on local operation-log terminology
// while sharing the cross-package contract type.
type OperationLogInput = contract.OperationLogInput

type OperationLogCondition struct {
	OperatorId   int32  `json:"operator_id" form:"operator_id"`
	OperatorName string `json:"operator_name" form:"operator_name"`
	Resource     string `json:"resource" form:"resource"`
	ResourceID   int64  `json:"resource_id" form:"resource_id"`
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
	ResourceID   int64
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
	beforeJSON := marshalAuditData(input.BeforeData)
	afterJSON := marshalAuditData(input.AfterData)

	operationLog := &model.SysOperationLog{
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

	_, err := o.operationLogDao.Create(ctx, tx, operationLog)
	return err
}

// GetOperationLogList 获取操作日志列表信息
func (o *OperationLogService) GetOperationLogList(ctx context.Context, condition *OperationLogCondition) (*OperationLogList, error) {
	var startTimePtr *time.Time
	var endTimePtr *time.Time
	if condition.StartTime != "" {
		t, err := time.ParseInLocation(constant.TimeFmtWithS, condition.StartTime, time.Local)
		if err != nil {
			return nil, ErrTimeFormat
		}
		startTimePtr = &t
	}
	if condition.EndTime != "" {
		t, err := time.ParseInLocation(constant.TimeFmtWithS, condition.EndTime, time.Local)
		if err != nil {
			return nil, ErrTimeFormat
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
		return nil, fmt.Errorf("操作日志信息列表查询失败: %w", err)
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

// marshalAuditData serializes snapshots after dropping high-risk credential fields.
func marshalAuditData(v any) string {
	if v == nil || v == "" {
		return "{}"
	}
	data, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	var raw any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return "{}"
	}
	data, err = json.Marshal(sanitizeAuditValue(raw))
	if err != nil {
		return "{}"
	}
	return string(data)
}

func shouldDropAuditField(key string) bool {
	var b strings.Builder
	for i, r := range key {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				b.WriteByte('_')
			}
			b.WriteRune(r + ('a' - 'A'))
			continue
		}
		if r == '-' || r == ' ' {
			b.WriteByte('_')
			continue
		}
		b.WriteRune(r)
	}
	_, drop := auditDropFields[b.String()]
	return drop
}

func sanitizeAuditValue(v any) any {
	switch val := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(val))
		for k, item := range val {
			if shouldDropAuditField(k) {
				continue
			}
			out[k] = sanitizeAuditValue(item)
		}
		return out
	case []any:
		for i, item := range val {
			val[i] = sanitizeAuditValue(item)
		}
		return val
	default:
		return v
	}
}
