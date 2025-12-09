package rabbitmq

import (
	"context"
	"snowgo/pkg/xmq"
	"time"

	"go.uber.org/zap"
)

// 内置 nopLogger：实现 xmq.Logger，默认不输出，避免外层未传 logger 导致 nil panic
type nopLogger struct{}

func (n *nopLogger) Info(ctx context.Context, msg string, fields ...zap.Field)  {}
func (n *nopLogger) Warn(ctx context.Context, msg string, fields ...zap.Field)  {}
func (n *nopLogger) Error(ctx context.Context, msg string, fields ...zap.Field) {}

// ProducerConnConfig producer链接 的配置（url 必传）
type ProducerConnConfig struct {
	URL                         string        // 必填：RabbitMQ 连接 URL，例如 "amqp://user:pass@host:port/vhost"
	ProducerChannelPoolSize     int           // 生产者 channel 池大小，决定并发发布能力，默认 8
	PublishGetTimeout           time.Duration // 获取可用 channel 的超时时间，超过返回 ErrGetChannelTimeout，默认 3s
	ConfirmTimeout              time.Duration // 等待 broker confirm 的超时时间，每条消息等待 confirm 的最大时长，默认 5s
	ReconnectInitialDelay       time.Duration // 连接失败或断开后的初始重连延迟（指数回退），默认 500ms
	ReconnectMaxDelay           time.Duration // 重连延迟的最大值（指数回退上限），默认 30s
	ConsecutiveTimeoutThreshold int32         // 连续 confirm 超时计数阈值，达到后会标记 channel 为 broken 并回收，默认 3
	Logger                      xmq.Logger    // 日志接口，推荐传入 zap.Logger 封装的 xmq.Logger，否则使用默认 nopLogger（不输出日志）
}

type ProducerConnOption func(*ProducerConnConfig)

func NewProducerConnConfig(url string, opts ...ProducerConnOption) *ProducerConnConfig {
	if url == "" {
		panic("rabbitmq: NewProducerConnConfig url must be provided")
	}
	cfg := &ProducerConnConfig{
		URL:                         url,
		ProducerChannelPoolSize:     8,
		PublishGetTimeout:           3 * time.Second,
		ConfirmTimeout:              5 * time.Second,
		ReconnectInitialDelay:       500 * time.Millisecond,
		ReconnectMaxDelay:           30 * time.Second,
		ConsecutiveTimeoutThreshold: 3,
		Logger:                      &nopLogger{}, // 默认 nopLogger，避免 nil
	}
	for _, o := range opts {
		o(cfg)
	}
	return cfg
}

// WithChannelPoolSize 生产者 channel 池大小，决定并发发布能力，默认 8
func WithChannelPoolSize(n int) ProducerConnOption {
	return func(c *ProducerConnConfig) {
		if n > 0 {
			c.ProducerChannelPoolSize = n
		}
	}
}

// WithPublishGetTimeout 获取可用 channel 的超时时间，超过返回 ErrGetChannelTimeout，默认 3s
func WithPublishGetTimeout(d time.Duration) ProducerConnOption {
	return func(c *ProducerConnConfig) {
		if d > 0 {
			c.PublishGetTimeout = d
		}
	}
}

// WithConfirmTimeout 等待 broker confirm 的超时时间，每条消息等待 confirm 的最大时长，默认 5s
func WithConfirmTimeout(d time.Duration) ProducerConnOption {
	return func(c *ProducerConnConfig) {
		if d > 0 {
			c.ConfirmTimeout = d
		}
	}
}

// WithProducerReconnectInitialDelay 连接失败或断开后的初始重连延迟（指数回退），默认 500ms
func WithProducerReconnectInitialDelay(d time.Duration) ProducerConnOption {
	return func(c *ProducerConnConfig) {
		if d > 0 {
			c.ReconnectInitialDelay = d
		}
	}
}

// WithProducerReconnectMaxDelay 重连延迟的最大值（指数回退上限），默认 30s
func WithProducerReconnectMaxDelay(d time.Duration) ProducerConnOption {
	return func(c *ProducerConnConfig) {
		if d > 0 {
			c.ReconnectMaxDelay = d
		}
	}
}

// WithConsecTimeoutThreshold 连续 confirm 超时计数阈值，达到后会标记 channel 为 broken 并回收，默认 3
func WithConsecTimeoutThreshold(n int32) ProducerConnOption {
	return func(c *ProducerConnConfig) {
		if n > 0 {
			c.ConsecutiveTimeoutThreshold = n
		}
	}
}

// WithProducerLogger 生产日志
func WithProducerLogger(l xmq.Logger) ProducerConnOption {
	return func(c *ProducerConnConfig) {
		if l != nil {
			c.Logger = l
		}
	}
}

// ConsumerConnConfig producer链接 的配置（url 必传）
type ConsumerConnConfig struct {
	URL                   string
	ConsumerBackoffBase   time.Duration // 消费获取 channel 失败时的基础重试间隔
	ReconnectInitialDelay time.Duration // 连接失败或断开后的初始重连延迟（指数回退），默认 500ms
	ReconnectMaxDelay     time.Duration // 重连延迟的最大值（指数回退上限），默认 30s

	Logger xmq.Logger
}

type ConsumerConnOption func(*ConsumerConnConfig)

func NewConsumerConnConfig(url string, opts ...ConsumerConnOption) *ConsumerConnConfig {
	if url == "" {
		panic("rabbitmq: NewConsumerConnConfig url must be provided")
	}
	cfg := &ConsumerConnConfig{
		URL:                   url,
		ConsumerBackoffBase:   1 * time.Second,
		ReconnectInitialDelay: 500 * time.Millisecond,
		ReconnectMaxDelay:     30 * time.Second,
		Logger:                &nopLogger{},
	}
	for _, o := range opts {
		o(cfg)
	}
	return cfg
}

// WithBackoffBase 消费失败或获取 channel 失败时的基础重试间隔
func WithBackoffBase(d time.Duration) ConsumerConnOption {
	return func(c *ConsumerConnConfig) {
		if d > 0 {
			c.ConsumerBackoffBase = d
		}
	}
}

// WithConsumerReconnectInitialDelay 连接失败或断开后的初始重连延迟（指数回退），默认 500ms
func WithConsumerReconnectInitialDelay(d time.Duration) ConsumerConnOption {
	return func(c *ConsumerConnConfig) {
		if d > 0 {
			c.ReconnectInitialDelay = d
		}
	}
}

// WithConsumerReconnectMaxDelay 重连延迟的最大值（指数回退上限），默认 30s
func WithConsumerReconnectMaxDelay(d time.Duration) ConsumerConnOption {
	return func(c *ConsumerConnConfig) {
		if d > 0 {
			c.ReconnectMaxDelay = d
		}
	}
}

// WithConsumerLogger 消费日志
func WithConsumerLogger(l xmq.Logger) ConsumerConnOption {
	return func(c *ConsumerConnConfig) {
		if l != nil {
			c.Logger = l
		}
	}
}
