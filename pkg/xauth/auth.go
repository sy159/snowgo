package xauth

import (
	"context"
	e "snowgo/pkg/xerror"
)

type contextKey string

const (
	XTraceId       contextKey = "X-Trace-Id"
	XTraceIDHeader string     = "X-Trace-Id"
	XIp            contextKey = "X-Client-Ip"
	XUserAgent     contextKey = "X-User-Agent"
	XUserId        contextKey = "X-User-Id"
	XUserName      contextKey = "X-User-Name"
	XSessionId     contextKey = "X-Session-Id"
)

type Context struct {
	TraceId   string
	IP        string
	UserAgent string
}

type UserContext struct {
	Context
	UserId    int32
	Username  string
	SessionId string
}

func GetContext(ctx context.Context) *Context {
	traceId, _ := ctx.Value(XTraceId).(string)
	ip, _ := ctx.Value(XIp).(string)
	userAgent, _ := ctx.Value(XUserAgent).(string)
	return &Context{
		TraceId:   traceId,
		IP:        ip,
		UserAgent: userAgent,
	}
}

// GetUserContext 获取登录的ctx
func GetUserContext(ctx context.Context) (*UserContext, error) {
	userId, ok := ctx.Value(XUserId).(int32)
	if !ok || userId <= 0 {
		return nil, e.HttpForbidden
	}
	traceId, _ := ctx.Value(XTraceId).(string)
	ip, _ := ctx.Value(XIp).(string)
	userAgent, _ := ctx.Value(XUserAgent).(string)
	username, _ := ctx.Value(XUserName).(string)
	sessionId, _ := ctx.Value(XSessionId).(string)
	return &UserContext{
		Context: Context{
			TraceId:   traceId,
			IP:        ip,
			UserAgent: userAgent,
		},
		UserId:    userId,
		Username:  username,
		SessionId: sessionId,
	}, nil
}
