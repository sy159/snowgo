package xgin_test

import (
	"testing"

	"snowgo/internal/constant"
	e "snowgo/pkg/xerror"
	"snowgo/pkg/xgin"
)

func TestNormalizePagination(t *testing.T) {
	testNormalizePagination(t, int(-1), int(10), e.OffsetErrorRequests, int(10))
	testNormalizePagination(t, int64(0), int64(constant.MaxLimit+1), e.LimitExceededErrorRequests, int64(constant.MaxLimit+1))
	testNormalizePagination(t, int64(0), int64(0), nil, int64(constant.DefaultLimit))

	tests := []struct {
		name      string
		offset    int32
		limit     int32
		wantCode  e.Code
		wantLimit int32
	}{
		{name: "negative offset", offset: -1, limit: 10, wantCode: e.OffsetErrorRequests, wantLimit: 10},
		{name: "negative limit", offset: 0, limit: -1, wantCode: e.LimitErrorRequests, wantLimit: -1},
		{name: "default limit", offset: 0, limit: 0, wantLimit: constant.DefaultLimit},
		{name: "max boundary", offset: 0, limit: constant.MaxLimit, wantLimit: constant.MaxLimit},
		{name: "over max", offset: 0, limit: constant.MaxLimit + 1, wantCode: e.LimitExceededErrorRequests, wantLimit: constant.MaxLimit + 1},
		{name: "normal", offset: 20, limit: 50, wantLimit: 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			offset, limit := tt.offset, tt.limit
			code := xgin.NormalizePagination(offset, &limit)
			if code != tt.wantCode {
				t.Fatalf("code = %v, want %v", code, tt.wantCode)
			}
			if limit != tt.wantLimit {
				t.Fatalf("limit = %d, want %d", limit, tt.wantLimit)
			}
		})
	}
}

type paginationNumber interface {
	~int | ~int32 | ~int64
}

func testNormalizePagination[T paginationNumber](t *testing.T, offset, limit T, wantCode e.Code, wantLimit T) {
	t.Helper()
	code := xgin.NormalizePagination(offset, &limit)
	if code != wantCode {
		t.Fatalf("code = %v, want %v", code, wantCode)
	}
	if limit != wantLimit {
		t.Fatalf("limit = %d, want %d", limit, wantLimit)
	}
}
