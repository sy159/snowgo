package xmq

import (
	"context"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	common "snowgo/pkg"
	"snowgo/pkg/xlogger"

	"time"

	"github.com/apache/pulsar-client-go/pulsar"
)

var _ Producer = (*PulsarProducer)(nil)

var (
	defaultProducer = producerOptions{
		disableBlockIfQueueFull: false,                  // 设置为false(默认值): 队列满了,会阻塞；true:队列满了,会立即返回错误
		maxPendingMessages:      10000,                  // 待确定消息最大长度
		disableBatching:         false,                  // true: 每一条消息单独发送；false: 合并发送
		batchingMaxPublishDelay: 100 * time.Millisecond, // 合并发送时生效，单批次最大等待时间
		batchingMaxMessages:     1000,                   // 合并发送时生效，单批次最大消息数
		batchingMaxSize:         2 * 1024 * 1024,        // 合并发送时生效，单批次最大消息字节
		maxRetries:              3,                      // 消息发送失败重试次数
		baseBackoffInterval:     100 * time.Millisecond, // 消息发送失败重试间隔时间
	}
	producerLogger = xlogger.NewLogger("./logs", "pulsar-producer", xlogger.WithFileMaxAgeDays(30))
	// 确保 PulsarProducer 实现 Producer 接口
)

type ProducerOptions func(*producerOptions)

type producerOptions struct {
	topic                   string
	deliverAfter            time.Duration // 消息延时时间
	disableBlockIfQueueFull bool          // 设置为false(默认值): 队列满了,会阻塞；true:队列满了,会立即返回错误
	maxPendingMessages      int           // 待确定消息最大长度
	disableBatching         bool          // true: 每一条消息单独发送；false: 合并发送
	batchingMaxPublishDelay time.Duration // 合并发送时生效，单批次最大等待时间
	batchingMaxMessages     uint          // 合并发送时生效，单批次最大消息数
	batchingMaxSize         uint          // 合并发送时生效，单批次最大消息字节
	maxRetries              int           // 消息发送失败重试次数
	baseBackoffInterval     time.Duration // 消息发送失败重试间隔时间
}

type PulsarClient struct {
	client    pulsar.Client
	producers []*PulsarProducer
}

type PulsarProducer struct {
	options  producerOptions
	producer pulsar.Producer
}

// NewPulsarClient 创建客户端
func NewPulsarClient(url string) (*PulsarClient, error) {
	if url == "" {
		return nil, errors.New("url is required")
	}
	client, err := pulsar.NewClient(pulsar.ClientOptions{
		URL:                     url,
		ConnectionTimeout:       10 * time.Second, // 连接超时时间
		OperationTimeout:        30 * time.Second, // 操作超时时间
		MaxConnectionsPerBroker: 5,
	})
	if err != nil {
		return nil, errors.WithMessage(err, "pulsar client is err")
	}

	return &PulsarClient{client: client}, nil
}

// Close 关闭客户端
func (c *PulsarClient) Close() {
	for _, p := range c.producers {
		_ = p.Close()
	}
	c.client.Close()
}

// NewProducer 创建生产者
func (c *PulsarClient) NewProducer(topic string, opts ...ProducerOptions) (*PulsarProducer, error) {
	if topic == "" {
		return nil, errors.New("topic is required")
	}

	producerOptions := defaultProducer
	producerOptions.topic = topic
	// 加载配置
	for _, opt := range opts {
		opt(&producerOptions)
	}

	producer, err := c.client.CreateProducer(pulsar.ProducerOptions{
		Topic:                   producerOptions.topic,
		DisableBlockIfQueueFull: producerOptions.disableBlockIfQueueFull,
		MaxPendingMessages:      producerOptions.maxPendingMessages,
		DisableBatching:         producerOptions.disableBatching,
		BatchingMaxPublishDelay: producerOptions.batchingMaxPublishDelay,
		BatchingMaxMessages:     producerOptions.batchingMaxMessages,
		BatchingMaxSize:         producerOptions.batchingMaxSize,
	})
	if err != nil {
		return nil, errors.WithMessage(err, "pulsar create producer is err")
	}

	newProducer := &PulsarProducer{
		options:  producerOptions,
		producer: producer,
	}
	c.producers = append(c.producers, newProducer)

	return newProducer, nil
}

// WithDeliverAfter 设置消息延迟消费时间
func WithDeliverAfter(deliverAfter time.Duration) ProducerOptions {
	return func(options *producerOptions) {
		if deliverAfter < 0 {
			deliverAfter = 0
		}
		// Pulsar限制deliverAfter不能超过15天
		maxDelay := 15 * 24 * time.Hour
		if deliverAfter > maxDelay {
			deliverAfter = maxDelay
		}
		options.deliverAfter = deliverAfter
	}
}

// WithMessageBatching 设置消息批量发送
func WithMessageBatching(disableBatching bool, maxPublishDelay time.Duration, maxMessages, maxSize uint) ProducerOptions {
	return func(options *producerOptions) {
		options.disableBatching = disableBatching
		options.batchingMaxPublishDelay = maxPublishDelay
		options.batchingMaxMessages = maxMessages
		options.batchingMaxSize = maxSize
	}
}

func WithMaxRetries(maxRetries int, baseBackoffInterval time.Duration) ProducerOptions {
	return func(options *producerOptions) {
		if maxRetries < 0 {
			maxRetries = 0
		}
		if baseBackoffInterval < 0 {
			baseBackoffInterval = 0
		}
		options.maxRetries = maxRetries
		options.baseBackoffInterval = baseBackoffInterval
	}
}

// LogMessage logs the message details.
func (p *PulsarProducer) LogMessage(ctx context.Context, messageId pulsar.MessageID, message []byte, properties map[string]string, sendTime time.Time, err error) {
	status := "success"
	errMsg := ""
	if err != nil {
		status = "fail"
		errMsg = err.Error()
	}
	messageIdStr := ""
	if messageId != nil {
		messageIdStr = messageId.String()
	}
	var traceID string
	if tid, ok := ctx.Value("trace_id").(string); ok {
		traceID = tid
	}
	producerLogger.Log(
		"pulsar producer",
		zap.String("topic", p.producer.Topic()),
		zap.String("message_id", messageIdStr),
		zap.String("message", string(message)),
		zap.Any("properties", properties),
		zap.String("created_at", sendTime.Format("2006-01-02 15:04:05.000")),
		zap.String("status", status),
		zap.String("error_msg", errMsg),
		zap.String("trace_id", traceID),
	)
}

// SendMessage 同步发送消息到mq
func (p *PulsarProducer) SendMessage(ctx context.Context, message []byte, properties map[string]string) error {
	msg := &pulsar.ProducerMessage{
		Payload:    message,
		Properties: properties,
	}

	if p.options.deliverAfter > 0 {
		msg.DeliverAfter = p.options.deliverAfter
	}

	var sendErr error
	for i := 0; i <= p.options.maxRetries; i++ {
		sendTime := time.Now()
		messageId, err := p.producer.Send(ctx, msg)
		p.LogMessage(ctx, messageId, message, properties, sendTime, err)
		sendErr = err
		if err == nil {
			break
		}
		// 失败不在最后一次则等待重试
		if i < p.options.maxRetries {
			// 指数退避
			backoff := p.options.baseBackoffInterval * (1 << i)
			// cap上限，避免等待过长
			maxBackoff := time.Second
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			// jitter：在 backoff/2 - backoff 范围内随机
			jitter := time.Duration(common.WeakRandInt63n(int64(backoff / 2)))
			time.Sleep(backoff/2 + jitter)
		}
	}
	if sendErr != nil {
		return errors.Wrap(sendErr, "pulsar send message is err")
	}
	return nil
}

// SendAsyncMessage 异步回调发送消息到mq
func (p *PulsarProducer) SendAsyncMessage(ctx context.Context, message []byte, properties map[string]string, callback func(messageID any, msg interface{}, err error)) {
	msg := &pulsar.ProducerMessage{
		Payload:    message,
		Properties: properties,
	}

	if p.options.deliverAfter > 0 {
		msg.DeliverAfter = p.options.deliverAfter
	}

	//p.producer.SendAsync(ctx, msg, callback)
	// 用匿名函数包装 Pulsar 原生回调
	p.producer.SendAsync(ctx, msg, func(msgID pulsar.MessageID, pm *pulsar.ProducerMessage, err error) {
		// 日志记录
		p.LogMessage(ctx, msgID, pm.Payload, pm.Properties, time.Now(), err)

		// 调用接口回调，转换类型为 any/interface{}
		if callback != nil {
			callback(msgID, pm, err)
		}
	})
}

// Close 关闭producer
func (p *PulsarProducer) Close() error {
	ctx := context.Background()
	err := p.producer.FlushWithCtx(ctx)
	p.producer.Close()
	return err
}
