package system

import (
	"context"
	"github.com/pkg/errors"
	"gorm.io/gen"
	"snowgo/internal/dal/model"
	"snowgo/internal/dal/query"
	"snowgo/internal/dal/repo"
	"time"
)

// OperationLogDao 操作日志
type OperationLogDao struct {
	repo *repo.Repository
}

func NewOperationLogDao(repo *repo.Repository) *OperationLogDao {
	return &OperationLogDao{
		repo: repo,
	}
}

type OperationLogCondition struct {
	OperatorId   int32      `json:"operator_id" form:"operator_id"`
	OperatorName string     `json:"operator_name" form:"operator_name"`
	Resource     string     `json:"resource" form:"resource"`
	ResourceID   int32      `json:"resource_id" form:"resource_id"`
	Action       string     `json:"action" form:"action"`
	StartTime    *time.Time `json:"start_time" form:"start_time"`
	EndTime      *time.Time `json:"end_time" form:"end_time"`
	Offset       int32      `json:"offset" form:"offset"`
	Limit        int32      `json:"limit" form:"limit"`
}

// TransactionCreate 创建操作日志
func (o *OperationLogDao) TransactionCreate(ctx context.Context, tx *query.Query, operationLog *model.OperationLog) (*model.OperationLog, error) {
	err := tx.WithContext(ctx).OperationLog.Create(operationLog)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return operationLog, nil
}

// GetOperationLogList 操作日志列表
func (o *OperationLogDao) GetOperationLogList(ctx context.Context, condition *OperationLogCondition) ([]*model.OperationLog, int64, error) {
	m := o.repo.Query().OperationLog
	userList, total, err := m.WithContext(ctx).
		Scopes(
			o.OperatorIdScope(condition.OperatorId),
			o.OperatorNameScope(condition.OperatorName),
			o.ResourceScope(condition.Resource),
			o.ResourceIDScope(condition.ResourceID),
			o.ActionScope(condition.Action),
			o.StartTimeScope(condition.StartTime),
			o.EndTimeScope(condition.EndTime),
		).
		FindByPage(int(condition.Offset), int(condition.Limit))
	if err != nil {
		return nil, 0, errors.WithStack(err)
	}
	return userList, total, nil
}

func (o *OperationLogDao) OperatorIdScope(operatorId int32) func(tx gen.Dao) gen.Dao {
	return func(tx gen.Dao) gen.Dao {
		if operatorId <= 0 {
			return tx
		}
		m := o.repo.Query().OperationLog
		tx = tx.Where(m.OperatorID.Eq(operatorId))
		return tx
	}
}

func (o *OperationLogDao) OperatorNameScope(operatorName string) func(tx gen.Dao) gen.Dao {
	return func(tx gen.Dao) gen.Dao {
		if len(operatorName) == 0 {
			return tx
		}
		m := o.repo.Query().OperationLog
		tx = tx.Where(m.OperatorName.Eq(operatorName))
		return tx
	}
}

func (o *OperationLogDao) ResourceScope(resource string) func(tx gen.Dao) gen.Dao {
	return func(tx gen.Dao) gen.Dao {
		if len(resource) == 0 {
			return tx
		}
		m := o.repo.Query().OperationLog
		tx = tx.Where(m.Resource.Eq(resource))
		return tx
	}
}

func (o *OperationLogDao) ResourceIDScope(resourceID int32) func(tx gen.Dao) gen.Dao {
	return func(tx gen.Dao) gen.Dao {
		if resourceID <= 0 {
			return tx
		}
		m := o.repo.Query().OperationLog
		tx = tx.Where(m.ResourceID.Eq(resourceID))
		return tx
	}
}

func (o *OperationLogDao) ActionScope(action string) func(tx gen.Dao) gen.Dao {
	return func(tx gen.Dao) gen.Dao {
		if len(action) == 0 {
			return tx
		}
		m := o.repo.Query().OperationLog
		tx = tx.Where(m.Action.Eq(action))
		return tx
	}
}

func (o *OperationLogDao) StartTimeScope(starTime *time.Time) func(tx gen.Dao) gen.Dao {
	return func(tx gen.Dao) gen.Dao {
		if starTime == nil {
			return tx
		}
		m := o.repo.Query().OperationLog
		tx = tx.Where(m.CreatedAt.Gte(*starTime))
		return tx
	}
}

func (o *OperationLogDao) EndTimeScope(endTime *time.Time) func(tx gen.Dao) gen.Dao {
	return func(tx gen.Dao) gen.Dao {
		if endTime == nil {
			return tx
		}
		m := o.repo.Query().OperationLog
		tx = tx.Where(m.CreatedAt.Lte(*endTime))
		return tx
	}
}
