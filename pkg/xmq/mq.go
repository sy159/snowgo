package xmq

import (
	"context"
)

// Producer 定义MQ生产者统一接口
type Producer interface {
	// SendMessage 同步发送消息
	SendMessage(ctx context.Context, message []byte, properties map[string]string) error
	// SendAsyncMessage 异步发送消息
	SendAsyncMessage(ctx context.Context, message []byte, properties map[string]string, callback func(messageID any, msg interface{}, err error))
	// Close 关闭生产者
	Close() error
}
