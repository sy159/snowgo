package mq

import (
	"context"
	"github.com/pkg/errors"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
)

var defaultProducer = producerOptions{
	topic:                   "default-topic",
	disableBlockIfQueueFull: false,           // 设置为false(默认值): 队列满了,会阻塞；true:队列满了,会立即返回错误
	maxPendingMessages:      10000,           // 待确定消息最大长度
	disableBatching:         true,            // true: 每一条消息单独发送；false: 合并发送
	batchingMaxPublishDelay: 2 * time.Second, // 合并发送时生效，单批次最大等待时间
	batchingMaxMessages:     5,               // 合并发送时生效，单批次最大消息数
	batchingMaxSize:         2 * 1024 * 1024, // 合并发送时生效，单批次最大消息字节
}

type Options func(producerOptions)

type producerOptions struct {
	topic                   string
	delay                   time.Duration // 消息延时时间
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

func NewPulsarProducer(url, topic string, opts ...Options) (*PulsarProducer, error) {
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
		opt(producerOptions)
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
		return nil, errors.WithMessage(err, "pulsar create producer is err")
	}

	return &PulsarProducer{
		options:  producerOptions,
		client:   client,
		producer: producer,
	}, nil
}

// WithDelayTime 设置消息延迟消费时间
func WithDelayTime(delayTime time.Duration) Options {
	return func(options producerOptions) {
		if delayTime > 0 {
			options.delay = delayTime
		}
	}
}

// WithMessageBatching 设置消息批量发送
func WithMessageBatching(disableBatching bool, maxPublishDelay time.Duration, maxMessages, maxSize uint) Options {
	return func(options producerOptions) {
		options.disableBatching = disableBatching
		options.batchingMaxPublishDelay = maxPublishDelay
		options.batchingMaxMessages = maxMessages
		options.batchingMaxSize = maxSize
	}
}

// SendMessage 同步发送消息到mq
func (p *PulsarProducer) SendMessage(ctx context.Context, message []byte, properties map[string]string) error {
	msg := &pulsar.ProducerMessage{
		Payload:    message,
		Properties: properties,
	}

	if p.options.delay > 0 {
		msg.DeliverAfter = p.options.delay
	}

	_, err := p.producer.Send(ctx, msg)
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

	if p.options.delay > 0 {
		msg.DeliverAfter = p.options.delay
	}

	p.producer.SendAsync(ctx, msg, callback)
}

func (p *PulsarProducer) Close() {
	p.producer.Close()
	p.client.Close()
}
