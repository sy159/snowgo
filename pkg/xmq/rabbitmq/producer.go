package rabbitmq

import (
	"context"
	"fmt"
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

// NewProducerWithConfig 使用 ProducerConnConfig 创建 Producer 实例
// cfg.URL 必须非空，否则 panic
// cfg.Logger 若为 nil，则默认使用 nopLogger
func NewProducerWithConfig(cfg *ProducerConnConfig) (*Producer, error) {
	if cfg == nil {
		return nil, fmt.Errorf("producer config is nil")
	}
	if cfg.URL == "" {
		return nil, fmt.Errorf("producer config URL must be non-empty")
	}
	if cfg.Logger == nil {
		cfg.Logger = &nopLogger{}
	}
	cm := newProducerConnManager(cfg)
	if err := cm.Start(); err != nil {
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
		msg.MessageId = fmt.Sprintf("%d", time.Now().UnixNano())
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
	if err != nil && p.log != nil {
		p.log.Warn("publish failed",
			zap.String("exchange", exchange),
			zap.String("routing_key", routingKey),
			zap.String("message_id", msg.MessageId),
			zap.String("err", err.Error()))
	}
	return err
}

// PublishDelayed 发布延迟消息，优先使用 x-delayed-message
// 如果失败降级为 TTL+DLX
func (p *Producer) PublishDelayed(ctx context.Context, delayedExchange, routingKey string, msg *xmq.Message, delayMillis int64) error {
	if msg == nil {
		return fmt.Errorf("nil message")
	}
	if msg.Headers == nil {
		msg.Headers = make(map[string]interface{})
	}
	msg.Headers["x-delay"] = delayMillis

	// 尝试首选 publish
	err := p.Publish(ctx, delayedExchange, routingKey, msg)
	if err == nil {
		return nil
	}

	// 降级 TTL+DLX
	delete(msg.Headers, "x-delay")
	delete(msg.Headers, "expiration")
	msg.Headers["expiration"] = fmt.Sprintf("%d", delayMillis)

	return p.Publish(ctx, delayedExchange, routingKey, msg)
}

// Close 优雅关闭 producer
// 会等待所有 inflight 消息完成
func (p *Producer) Close(ctx context.Context) error {
	if p == nil || p.cm == nil {
		return nil
	}
	return p.cm.Close()
}
