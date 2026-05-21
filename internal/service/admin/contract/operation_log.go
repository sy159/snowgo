package contract

import (
	"context"

	"snowgo/internal/dal/query"
)

// OperationLogWriter is the shared admin-service contract for synchronous audit logs.
// It belongs outside concrete domain packages so callers do not depend on system
// service implementations, while providers can implement one stable contract.
type OperationLogWriter interface {
	CreateOperationLog(ctx context.Context, tx *query.Query, input *OperationLogInput) error
}

type OperationLogInput struct {
	OperatorID   int32
	OperatorName string
	OperatorType string
	Resource     string
	ResourceID   int64
	Action       string
	TraceID      string
	BeforeData   any
	AfterData    any
	Description  string
	IP           string
}
