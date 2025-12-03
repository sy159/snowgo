package xmq

import (
	"context"
	"errors"
	"time"
)

var (
	ErrClosed            = errors.New("mq client closed")
	ErrPublishTimeout    = errors.New("mq publish confirm timeout")
	ErrPublishNack       = errors.New("mq publish nack from broker")
	ErrNoConnection      = errors.New("mq no connection")
	ErrGetChannelTimeout = errors.New("mq get producer channel timeout")
	ErrChannelExhausted  = errors.New("mq consumer channels exhausted")
)

// Message 消息结构体
type Message struct {
	Body      []byte                 // 业务数据
	Headers   map[string]interface{} // 自定义 key/value（可用于存放 trace_id、业务字段等）
	Timestamp time.Time              // 发送时间（在 Publish 前会默认填充）
	MessageId string                 // 业务唯一 ID（用于幂等/追踪）
}

type Logger interface {
	Info(msg string, keysAndValues ...interface{})
	Warn(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
}

// Producer 生产者
type Producer interface {
	// Publish 同步发布，成功返回 nil；失败返回 error（带确认超时、nack）。
	Publish(ctx context.Context, exchange, routingKey string, message *Message) error

	// PublishDelayed 使用 x-delayed-message 插件将消息延时投递到指定 delayed exchange。
	PublishDelayed(ctx context.Context, delayedExchange, routingKey string, message *Message, delayMillis int64) error

	// Close 优雅关闭：等待 inflight 消息确认后释放资源。
	Close(ctx context.Context) error
}

// Handler 业务唯一需要实现的函数。 返回 nil 表示成功，框架自动 ACK；返回error框架自动 NACK 并重试。
type Handler func(ctx context.Context, msg Message) error

// Consumer 业务注册 Handler，框架负责 ACK/重试
type Consumer interface {
	Register(topic string, group string, handler Handler, opts ...interface{}) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}
