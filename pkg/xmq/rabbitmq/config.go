package rabbitmq

import (
	"snowgo/pkg/xmq"
	"time"

	"go.uber.org/zap"
)

// 内置 nopLogger：实现 xmq.Logger，默认不输出，避免外层未传 logger 导致 nil panic
type nopLogger struct{}

func (n *nopLogger) Info(msg string, fields ...zap.Field)  {}
func (n *nopLogger) Warn(msg string, fields ...zap.Field)  {}
func (n *nopLogger) Error(msg string, fields ...zap.Field) {}

// ProducerConfig 专注于 producer 的配置（url 必传）
type ProducerConfig struct {
	URL                         string
	ProducerChannelPoolSize     int
	PublishGetTimeout           time.Duration
	ConfirmTimeout              time.Duration
	PutBackTimeout              time.Duration
	ReconnectInitialDelay       time.Duration
	ReconnectMaxDelay           time.Duration
	ConsecutiveTimeoutThreshold int32
	Logger                      xmq.Logger
}

type ProducerOption func(*ProducerConfig)

func NewProducerConfig(url string, opts ...ProducerOption) *ProducerConfig {
	if url == "" {
		panic("rabbitmq: NewProducerConfig url must be provided")
	}
	cfg := &ProducerConfig{
		URL:                         url,
		ProducerChannelPoolSize:     16,
		PublishGetTimeout:           3 * time.Second,
		ConfirmTimeout:              5 * time.Second,
		PutBackTimeout:              800 * time.Millisecond,
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

func WithPoolSize(n int) ProducerOption {
	return func(c *ProducerConfig) {
		if n > 0 {
			c.ProducerChannelPoolSize = n
		}
	}
}
func WithPublishGetTimeout(d time.Duration) ProducerOption {
	return func(c *ProducerConfig) {
		if d > 0 {
			c.PublishGetTimeout = d
		}
	}
}
func WithConfirmTimeout(d time.Duration) ProducerOption {
	return func(c *ProducerConfig) {
		if d > 0 {
			c.ConfirmTimeout = d
		}
	}
}
func WithPutBackTimeout(d time.Duration) ProducerOption {
	return func(c *ProducerConfig) {
		if d > 0 {
			c.PutBackTimeout = d
		}
	}
}
func WithProducerReconnectInitialDelay(d time.Duration) ProducerOption {
	return func(c *ProducerConfig) {
		if d > 0 {
			c.ReconnectInitialDelay = d
		}
	}
}
func WithProducerReconnectMaxDelay(d time.Duration) ProducerOption {
	return func(c *ProducerConfig) {
		if d > 0 {
			c.ReconnectMaxDelay = d
		}
	}
}
func WithConsecTimeoutThreshold(n int32) ProducerOption {
	return func(c *ProducerConfig) {
		if n > 0 {
			c.ConsecutiveTimeoutThreshold = n
		}
	}
}
func WithProducerLogger(l xmq.Logger) ProducerOption {
	return func(c *ProducerConfig) {
		if l != nil {
			c.Logger = l
		}
	}
}

// ------------------------------------------------------------------

// ConsumerConfig 专注 consumer 的配置（url 必传）
type ConsumerConfig struct {
	URL                   string
	ConsumerRetryLimit    int
	ConsumerBackoffBase   time.Duration
	ReconnectInitialDelay time.Duration
	ReconnectMaxDelay     time.Duration

	Logger xmq.Logger
}

type ConsumerOption func(*ConsumerConfig)

func NewConsumerConfig(url string, opts ...ConsumerOption) *ConsumerConfig {
	if url == "" {
		panic("rabbitmq: NewConsumerConfig url must be provided")
	}
	cfg := &ConsumerConfig{
		URL:                   url,
		ConsumerRetryLimit:    5,
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

func WithRetryLimit(n int) ConsumerOption {
	return func(c *ConsumerConfig) {
		if n >= 0 {
			c.ConsumerRetryLimit = n
		}
	}
}
func WithBackoffBase(d time.Duration) ConsumerOption {
	return func(c *ConsumerConfig) {
		if d > 0 {
			c.ConsumerBackoffBase = d
		}
	}
}
func WithConsumerReconnectInitialDelay(d time.Duration) ConsumerOption {
	return func(c *ConsumerConfig) {
		if d > 0 {
			c.ReconnectInitialDelay = d
		}
	}
}
func WithConsumerReconnectMaxDelay(d time.Duration) ConsumerOption {
	return func(c *ConsumerConfig) {
		if d > 0 {
			c.ReconnectMaxDelay = d
		}
	}
}
func WithConsumerLogger(l xmq.Logger) ConsumerOption {
	return func(c *ConsumerConfig) {
		if l != nil {
			c.Logger = l
		}
	}
}
