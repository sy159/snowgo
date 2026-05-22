package account

import (
	"context"

	"snowgo/pkg/xauth"
)

func testUserCtx() context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, xauth.XUserId, int32(1))
	ctx = context.WithValue(ctx, xauth.XUserName, "admin")
	ctx = context.WithValue(ctx, xauth.XTraceId, "trace-test")
	ctx = context.WithValue(ctx, xauth.XIp, "127.0.0.1")
	return ctx
}
