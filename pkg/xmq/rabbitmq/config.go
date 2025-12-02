package rabbitmq

import (
	"snowgo/pkg/xmq"
	"time"
)

type Config struct {
	URL string

	// Producer
	ProducerChannelPoolSize int           // 池大小（每个 channel 串行发布）
	PublishConfirmTimeout   time.Duration // 等待 broker confirm 的超时时间
	PublishGetTimeout       time.Duration // 从池获取 channel 的超时时间

	// reconnect/backoff
	ReconnectInitialDelay time.Duration
	ReconnectMaxDelay     time.Duration

	// Delay plugin
	EnableXDelayed bool // 是否使用 x-delayed-message 插件（PublishDelayed 会基于此）
	// Optional user hook before publish (例如注入 trace_id、幂等检查等)
	BeforePublish func(msg *xmq.Message) error

	// logger
	Logger xmq.Logger

	NotifyBufSize int // confirm chan buffer
}

func defaultConfig() *Config {
	return &Config{
		URL:                     "amqp://guest:guest@localhost:5672/",
		ProducerChannelPoolSize: 8,
		PublishConfirmTimeout:   5 * time.Second,
		PublishGetTimeout:       3 * time.Second,
		ReconnectInitialDelay:   1 * time.Second,
		ReconnectMaxDelay:       30 * time.Second,
		EnableXDelayed:          true,
		NotifyBufSize:           64,
	}
}

type Option func(*Config)

func WithURL(url string) Option {
	return func(c *Config) { c.URL = url }
}
func WithLogger(l xmq.Logger) Option {
	return func(c *Config) { c.Logger = l }
}
func WithPoolSize(n int) Option {
	return func(c *Config) {
		c.ProducerChannelPoolSize = n
	}
}
func WithBeforePublish(fn func(*xmq.Message) error) Option {
	return func(c *Config) { c.BeforePublish = fn }
}
func WithEnableDelayed() Option {
	return func(c *Config) {
		c.EnableXDelayed = true
	}
}
