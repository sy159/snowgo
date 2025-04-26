package xauth

import (
	"context"
	"github.com/pkg/errors"
	e "snowgo/pkg/xerror"
)

const (
	XTraceId   = "X-Trace-Id"
	XIp        = "X-Client-Ip"
	XUserAgent = "X-User-Agent"
	XUserId    = "X-User-Id"
	XUserName  = "X-User-Name"
)

type Context struct {
	TraceId   string
	IP        string
	UserAgent string
}

type UserContext struct {
	TraceId   string
	IP        string
	UserAgent string
	UserId    int64
	Username  string
}

func GetContext(ctx context.Context) *Context {
	traceId, _ := ctx.Value(XTraceId).(string)
	iP, _ := ctx.Value(XIp).(string)
	userAgent, _ := ctx.Value(XUserAgent).(string)
	return &Context{
		TraceId:   traceId,
		IP:        iP,
		UserAgent: userAgent,
	}
}

// GetUserContext 获取登录的ctx
func GetUserContext(ctx context.Context) (*UserContext, error) {
	userId, ok := ctx.Value(XUserId).(int64)
	if !ok || userId <= 0 {
		return nil, errors.New(e.HttpForbidden.GetErrMsg())
	}
	traceId, _ := ctx.Value(XTraceId).(string)
	iP, _ := ctx.Value(XIp).(string)
	userAgent, _ := ctx.Value(XUserAgent).(string)
	username, _ := ctx.Value(XUserName).(string)
	return &UserContext{
		TraceId:   traceId,
		IP:        iP,
		UserAgent: userAgent,
		UserId:    userId,
		Username:  username,
	}, nil
}
