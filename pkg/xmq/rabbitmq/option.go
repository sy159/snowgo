package rabbitmq

import (
	"runtime"
	"time"

	"snowgo/pkg/xmq"
)

// Role 角色常量（将来可根据 Role 做不同初始化）
type Role string

const (
	RoleProducer Role = "producer"
	RoleConsumer Role = "consumer"
)

// Config 全量配置
type Config struct {
	URL    string
	Logger xmq.Logger
	Role   Role // 新建连接时必须指定角色

	// 公共重连
	ReconnectInitialDelay time.Duration
	ReconnectMaxDelay     time.Duration

	// Producer 专用
	ProducerChannelPoolSize int
	ConfirmTimeout          time.Duration
	PublishGetTimeout       time.Duration

	// Consumer 专用
	MaxChannelsPerConn int // 单连接最大 Channel 数，防止耗尽
}

func DefaultConfig() *Config {
	return &Config{
		URL:                   "amqp://guest:guest@localhost:5672/",
		Logger:                &nopLogger{},
		ReconnectInitialDelay: 1 * time.Second,
		ReconnectMaxDelay:     30 * time.Second,

		// Producer
		ProducerChannelPoolSize: runtime.NumCPU() * 2,
		ConfirmTimeout:          5 * time.Second,
		PublishGetTimeout:       3 * time.Second,

		// Consumer
		MaxChannelsPerConn: 2047, // 默认 rabbitmq 单连接上限
	}
}

// Option 函数
type Option func(*Config)

func WithURL(url string) Option      { return func(c *Config) { c.URL = url } }
func WithLogger(l xmq.Logger) Option { return func(c *Config) { c.Logger = l } }
func WithRole(r Role) Option         { return func(c *Config) { c.Role = r } }
func WithProducerChannelPoolSize(n int) Option {
	return func(c *Config) { c.ProducerChannelPoolSize = n }
}
func WithConfirmTimeout(d time.Duration) Option { return func(c *Config) { c.ConfirmTimeout = d } }
func WithPublishGetTimeout(d time.Duration) Option {
	return func(c *Config) { c.PublishGetTimeout = d }
}
func WithMaxChannelsPerConn(n int) Option { return func(c *Config) { c.MaxChannelsPerConn = n } }
func WithReconnectDelay(init, max time.Duration) Option {
	return func(c *Config) {
		c.ReconnectInitialDelay = init
		c.ReconnectMaxDelay = max
	}
}

type nopLogger struct{}

func (n *nopLogger) Info(msg string, kvs ...interface{})  {}
func (n *nopLogger) Warn(msg string, kvs ...interface{})  {}
func (n *nopLogger) Error(msg string, kvs ...interface{}) {}
