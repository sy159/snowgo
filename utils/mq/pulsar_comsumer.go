package mq

import (
	"context"
	"fmt"
	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"snowgo/utils"
	"snowgo/utils/logger"
	"sync"
)

var defaultConsumer = consumerOptions{
	subscriptionName: "default-subscription",
	subscriptionType: pulsar.Shared,
	handler: func(ctx context.Context, message pulsar.Message) {
		fmt.Printf("Received a message: %s; properties: %+v\n", string(message.Payload()), message.Properties())
	},
	runNum:          1,
	ackWithResponse: true,
}

type consumerOptions struct {
	subscriptionName string                                // 订阅名称
	subscriptionType pulsar.SubscriptionType               // 订阅类型
	handler          func(context.Context, pulsar.Message) // 消息处理方法
	runNum           int                                   // 启动的消费者数量
	ackWithResponse  bool                                  // 默认是false; 如果是false，不需要Broker确定(低延迟)；如果是true，Broker会发送ack确定然后删除消息(保证在队列故障不会被重复消费)
}

type ConsumerOptions func(*consumerOptions)

// PulsarConsumer 消费者
type PulsarConsumer struct {
	options   consumerOptions
	client    pulsar.Client
	consumers []pulsar.Consumer
}

// NewPulsarConsumer 创建一个新的Pulsar消费者
func NewPulsarConsumer(url, topic string, handler func(context.Context, pulsar.Message), opts ...ConsumerOptions) (*PulsarConsumer, error) {
	client, err := pulsar.NewClient(pulsar.ClientOptions{
		URL: url,
	})
	if err != nil {
		return nil, errors.WithMessage(err, "pulsar client is err")
	}

	consumerOptions := defaultConsumer
	consumerOptions.handler = handler
	// 加载配置
	for _, opt := range opts {
		opt(&consumerOptions)
	}

	var consumers []pulsar.Consumer
	if consumerOptions.runNum < 1 {
		consumerOptions.runNum = 1
	}
	for i := 0; i < consumerOptions.runNum; i++ {
		consumerName := fmt.Sprintf("%s_%s_%d", topic, consumerOptions.subscriptionName, i)
		consumer, err := client.Subscribe(pulsar.ConsumerOptions{
			Topic:            topic,
			SubscriptionName: consumerOptions.subscriptionName,
			Type:             consumerOptions.subscriptionType,
			Name:             consumerName,
			AckWithResponse:  consumerOptions.ackWithResponse,
		})
		if err != nil {
			client.Close()
			return nil, errors.WithMessage(err, fmt.Sprintf("pulsar create consumer %v is error", consumerName))
		}
		consumers = append(consumers, consumer)
	}

	return &PulsarConsumer{
		options:   consumerOptions,
		client:    client,
		consumers: consumers,
	}, nil
}

// WithRunNum 设置消费数量
func WithRunNum(runNum int) ConsumerOptions {
	return func(consumerOptions *consumerOptions) {
		if runNum > 1 {
			consumerOptions.runNum = runNum
		}
	}
}

// WithSubscriptionType 默认为share
func WithSubscriptionType(subscriptionType pulsar.SubscriptionType) ConsumerOptions {
	return func(consumerOptions *consumerOptions) {
		consumerOptions.subscriptionType = subscriptionType
	}
}

// WithSubscriptionName 设置消费订阅名(相同订阅名下消息只能被其中一个消费者消费；不同订阅名信息会被多次消费)
func WithSubscriptionName(subscriptionName string) ConsumerOptions {
	return func(consumerOptions *consumerOptions) {
		consumerOptions.subscriptionName = subscriptionName
	}
}

// WithAckWithResponse 设置是否需要broker确定(false低延迟，不过可能会丢数据；true数据不会有问题，相对延迟高一点)
func WithAckWithResponse(ackWithResponse bool) ConsumerOptions {
	return func(consumerOptions *consumerOptions) {
		consumerOptions.ackWithResponse = ackWithResponse
	}
}

// Start 启动消费者并开始消费消息
func (c *PulsarConsumer) Start(ctx context.Context) {
	var wg sync.WaitGroup

	for _, consumer := range c.consumers {
		wg.Add(1)
		go func(consumer pulsar.Consumer) {
			defer wg.Done()
			consumerName := consumer.Name()
			logger.Infof("Starting consumer: %s", consumerName)

			for {
				select {
				case msg := <-consumer.Chan():
					c.options.handler(ctx, msg)
					if err := consumer.Ack(msg); err != nil {
						// TODO 增加其他的ack失败处理
						logger.Error("Failed to acknowledge message",
							zap.String("error", utils.ErrorToString(err)),
							zap.String("consumer", consumerName),
							zap.String("messageId", msg.ID().String()),
						)
					}
				case <-ctx.Done():
					logger.Infof("Stopping consumer: %s", consumerName)
					return
				}
			}
		}(consumer)
	}

	wg.Wait()
}

// Close 关闭消费者
func (c *PulsarConsumer) Close() {
	for _, consumer := range c.consumers {
		consumer.Close()
	}
	c.client.Close()
}