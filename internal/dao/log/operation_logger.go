package log

import (
	"context"
	"github.com/pkg/errors"
	"snowgo/internal/dal/model"
	"snowgo/internal/dal/query"
	"snowgo/internal/dal/repo"
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

// TransactionCreate 创建操作日志
func (o *OperationLogDao) TransactionCreate(ctx context.Context, tx *query.Query, operationLog *model.OperationLog) (*model.OperationLog, error) {
	err := tx.WithContext(ctx).OperationLog.Create(operationLog)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return operationLog, nil
}
