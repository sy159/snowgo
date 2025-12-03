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

// Config 连接与行为配置
type Config struct {
	URL                   string
	ReconnectInitialDelay time.Duration
	ReconnectMaxDelay     time.Duration
	Logger                xmq.Logger

	// Producer 专用
	ProducerChannelPoolSize int
	ConfirmTimeout          time.Duration
	PublishGetTimeout       time.Duration
	NotifyBufSize           int // confirm notify buffer size per channel
}

func DefaultConfig() *Config {
	return &Config{
		URL:                   "amqp://guest:guest@localhost:5672/",
		ReconnectInitialDelay: 1 * time.Second,
		ReconnectMaxDelay:     30 * time.Second,
		Logger:                &nopLogger{},

		ProducerChannelPoolSize: runtime.NumCPU() * 2,
		ConfirmTimeout:          5 * time.Second,
		PublishGetTimeout:       3 * time.Second,
		NotifyBufSize:           256,
	}
}

// Option 函数类型
type Option func(*Config)

func WithURL(url string) Option      { return func(c *Config) { c.URL = url } }
func WithLogger(l xmq.Logger) Option { return func(c *Config) { c.Logger = l } }
func WithProducerChannelPoolSize(n int) Option {
	return func(c *Config) { c.ProducerChannelPoolSize = n }
}
func WithConfirmTimeout(d time.Duration) Option { return func(c *Config) { c.ConfirmTimeout = d } }
func WithPublishGetTimeout(d time.Duration) Option {
	return func(c *Config) { c.PublishGetTimeout = d }
}
func WithNotifyBufSize(n int) Option { return func(c *Config) { c.NotifyBufSize = n } }

type ConsumerOption func(*consumerMeta)

type consumerMeta struct {
	QoS       int
	WorkerNum int
	Handler   xmq.Handler
}

func DefaultConsumerMeta() *consumerMeta { return &consumerMeta{QoS: 16, WorkerNum: 4} }

func WithConsumerQoS(qos int) ConsumerOption     { return func(m *consumerMeta) { m.QoS = qos } }
func WithConsumerWorkerNum(n int) ConsumerOption { return func(m *consumerMeta) { m.WorkerNum = n } }

type nopLogger struct{}

func (n *nopLogger) Info(msg string, kvs ...interface{})  {}
func (n *nopLogger) Warn(msg string, kvs ...interface{})  {}
func (n *nopLogger) Error(msg string, kvs ...interface{}) {}
