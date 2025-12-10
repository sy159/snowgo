package rabbitmq

import (
	"context"
	"fmt"
	common "snowgo/pkg"
	"snowgo/pkg/xmq"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// Producer 对外生产者 API 封装
type Producer struct {
	cm  *producerConnManager
	cfg *ProducerConnConfig
	log xmq.Logger
}

// NewProducer 使用 ProducerConnConfig 创建 Producer 实例
func NewProducer(ctx context.Context, cfg *ProducerConnConfig) (*Producer, error) {
	if cfg == nil {
		return nil, fmt.Errorf("producer config is nil")
	}
	if cfg.URL == "" {
		return nil, fmt.Errorf("producer config URL must be non-empty")
	}
	if cfg.Logger == nil {
		cfg.Logger = &nopLogger{}
	}
	cm := newProducerConnManager(ctx, cfg)
	if err := cm.Start(ctx); err != nil {
		cm.logger.Error(
			ctx,
			"producer conn fail",
			zap.String("event", xmq.EventProducerConnection),
			zap.Error(err),
		)
		return nil, fmt.Errorf("producer start failed: %w", err)
	}
	return &Producer{
		cm:  cm,
		cfg: cfg,
		log: cfg.Logger,
	}, nil
}

// Publish 同步发布一条消息，等待 broker confirm
func (p *Producer) Publish(ctx context.Context, exchange, routingKey string, msg *xmq.Message) error {
	if p == nil || p.cm == nil {
		return xmq.ErrClosed
	}
	if msg == nil {
		return fmt.Errorf("nil message")
	}

	if msg.MessageId == "" {
		msg.MessageId = common.GenerateID()
	}
	msg.Timestamp = time.Now()

	pub := amqp.Publishing{
		Body:         msg.Body,
		Headers:      amqp.Table(msg.Headers),
		Timestamp:    msg.Timestamp,
		MessageId:    msg.MessageId,
		DeliveryMode: amqp.Persistent,
	}

	// Publish 内部已经管理 inflight
	err := p.cm.Publish(ctx, exchange, routingKey, pub)

	status := "success"
	if err != nil {
		status = "fail"
	}
	// 记录所有发送的业务消息
	p.log.Info(
		ctx,
		"publish msg",
		zap.String("event", xmq.EventPublish),
		zap.String("exchange", exchange),
		zap.String("routing_key", routingKey),
		zap.String("message_id", msg.MessageId),
		zap.String("message_body", string(msg.Body)),
		zap.Any("message_header", msg.Headers),
		zap.Any("status", status),
		zap.Error(err),
	)

	return err
}

// PublishDelayed 发布延迟消息，优先使用 x-delayed-message
func (p *Producer) PublishDelayed(ctx context.Context, delayedExchange, routingKey string, msg *xmq.Message, delayMillis int64) error {
	if msg == nil {
		return fmt.Errorf("nil message")
	}
	// 先尝试插件方案（x-delay）,rabbitmq已经安装rabbitmq_delayed_message_exchange插件，可直接实现延时
	if msg.Headers == nil {
		msg.Headers = make(map[string]interface{})
	}
	msg.Headers["x-delay"] = delayMillis

	// 尝试首选 publish
	err := p.Publish(ctx, delayedExchange, routingKey, msg)
	return err
	// 降级 TTL+DLX，根据情况实现
}

// Close 优雅关闭 producer
// 会等待所有 inflight 消息完成
func (p *Producer) Close(ctx context.Context) error {
	if p == nil || p.cm == nil {
		return nil
	}
	return p.cm.Close(ctx)
}
