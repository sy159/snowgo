package xmq

import (
	"context"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"snowgo/pkg/xlogger"

	"time"

	"github.com/apache/pulsar-client-go/pulsar"
)

var (
	defaultProducer = producerOptions{
		topic:                   "default-topic",
		disableBlockIfQueueFull: false,           // 设置为false(默认值): 队列满了,会阻塞；true:队列满了,会立即返回错误
		maxPendingMessages:      10000,           // 待确定消息最大长度
		disableBatching:         true,            // true: 每一条消息单独发送；false: 合并发送
		batchingMaxPublishDelay: 2 * time.Second, // 合并发送时生效，单批次最大等待时间
		batchingMaxMessages:     5,               // 合并发送时生效，单批次最大消息数
		batchingMaxSize:         2 * 1024 * 1024, // 合并发送时生效，单批次最大消息字节
	}
	producerLogger = xlogger.NewLogger("pulsar-producer", xlogger.WithFileMaxAgeDays(7))
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
}

type PulsarProducer struct {
	options  producerOptions
	client   pulsar.Client
	producer pulsar.Producer
}

func NewPulsarProducer(url, topic string, opts ...ProducerOptions) (*PulsarProducer, error) {
	client, err := pulsar.NewClient(pulsar.ClientOptions{
		URL: url,
	})
	if err != nil {
		return nil, errors.WithMessage(err, "pulsar client is err")
	}

	producerOptions := defaultProducer
	producerOptions.topic = topic
	// 加载配置
	for _, opt := range opts {
		opt(&producerOptions)
	}

	producer, err := client.CreateProducer(pulsar.ProducerOptions{
		Topic:                   producerOptions.topic,
		DisableBlockIfQueueFull: producerOptions.disableBlockIfQueueFull,
		MaxPendingMessages:      producerOptions.maxPendingMessages,
		DisableBatching:         producerOptions.disableBatching,
		BatchingMaxPublishDelay: producerOptions.batchingMaxPublishDelay,
		BatchingMaxMessages:     producerOptions.batchingMaxMessages,
		BatchingMaxSize:         producerOptions.batchingMaxSize,
	})
	if err != nil {
		client.Close()
		return nil, errors.WithMessage(err, "pulsar create producer is err")
	}

	return &PulsarProducer{
		options:  producerOptions,
		client:   client,
		producer: producer,
	}, nil
}

// WithDeliverAfter 设置消息延迟消费时间
func WithDeliverAfter(deliverAfter time.Duration) ProducerOptions {
	return func(options *producerOptions) {
		if deliverAfter > 0 {
			options.deliverAfter = deliverAfter
		}
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

// LogMessage logs the message details.
func (p *PulsarProducer) LogMessage(messageId pulsar.MessageID, message []byte, properties map[string]string, sendTime time.Time, err error) {
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
	producerLogger.Log(
		"pulsar producer",
		zap.String("topic", p.producer.Topic()),
		zap.String("message_id", messageIdStr),
		zap.String("message", string(message)),
		zap.Any("properties", properties),
		zap.String("created_at", sendTime.Format("2006-01-02 15:04:05.000")),
		zap.String("status", status),
		zap.String("error_msg", errMsg),
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

	sendTime := time.Now()
	messageId, err := p.producer.Send(ctx, msg)
	p.LogMessage(messageId, message, properties, sendTime, err)
	if err != nil {
		return errors.Wrap(err, "pulsar send message is err")
	}
	return nil
}

// SendAsyncMessage 异步回调发送消息到mq
func (p *PulsarProducer) SendAsyncMessage(ctx context.Context, message []byte, properties map[string]string, callback func(pulsar.MessageID, *pulsar.ProducerMessage, error)) {
	msg := &pulsar.ProducerMessage{
		Payload:    message,
		Properties: properties,
	}

	if p.options.deliverAfter > 0 {
		msg.DeliverAfter = p.options.deliverAfter
	}

	p.producer.SendAsync(ctx, msg, callback)
}

func (p *PulsarProducer) Close() {
	p.producer.Close()
	p.client.Close()
}
