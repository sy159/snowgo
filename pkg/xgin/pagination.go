package xgin

import (
	"snowgo/internal/constant"
	e "snowgo/pkg/xerror"
)

type paginationNumber interface {
	~int | ~int32 | ~int64
}

// NormalizePagination validates offset and limit, applying the default limit when omitted.
func NormalizePagination[T paginationNumber](offset T, limit *T) e.Code {
	if offset < 0 {
		return e.OffsetErrorRequests
	}
	if *limit < 0 {
		return e.LimitErrorRequests
	}
	if *limit == 0 {
		*limit = T(constant.DefaultLimit)
		return nil
	}
	if *limit > T(constant.MaxLimit) {
		return e.LimitExceededErrorRequests
	}
	return nil
}
