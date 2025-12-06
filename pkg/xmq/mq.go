package xmq

import (
	"context"
	"errors"
	"go.uber.org/zap"
	"time"
)

var (
	ErrClosed                    = errors.New("mq client closed")
	ErrProducerConnManagerClosed = errors.New("mq producer conn manager closed")
	ErrPublishTimeout            = errors.New("mq publish confirm timeout")
	ErrPublishNack               = errors.New("mq publish nack from broker")
	ErrNoConnection              = errors.New("mq no connection")
	ErrGetChannelTimeout         = errors.New("mq get producer channel timeout")
)

const (
	// EventProducerConnection Producer Connection & Channel & Confirm
	EventProducerConnection      = "producer_connection"       // 生产者建立连接
	EventProducerReconnection    = "producer_reconnection"     // 生产者重新建立连接
	EventProducerCloseConnection = "producer_close_connection" // 生产者连接关闭
	EventProducerChannel         = "producer_channel"          //  producer channel
	EventProducerConfirm         = "producer_confirm"          // producer confirm

	// EventPublish Publish
	EventPublish = "publish_msg" // 发布消息

	// EventConsumerConnection Consumer
	EventConsumerConnection      = "consumer_connection"       // consumer建立连接
	EventConsumerReconnection    = "consumer_reconnection"     // consumer重新建立连接
	EventConsumerCloseConnection = "consumer_close_connection" // consumer连接关闭
	EventConsumerChannel         = "consumer_channel"          //  consumer channel
	EventConsumerConsume         = "consumer_Consume"          //  consumer consume
)

// Message 消息结构体
type Message struct {
	Body      []byte                 // 业务数据
	Headers   map[string]interface{} // 自定义 key/value（可用于存放 trace_id、业务字段等）
	Timestamp time.Time              // 发送时间（在 Publish 前会默认填充）
	MessageId string                 // 业务唯一 ID（用于幂等/追踪）
}

type Logger interface {
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
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
	Register(ctx context.Context, exchange string, routingKey string, handler Handler, opts ...interface{}) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}
