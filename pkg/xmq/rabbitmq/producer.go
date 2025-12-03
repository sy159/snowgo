package rabbitmq

import (
	"context"
	"fmt"
	"snowgo/pkg/xmq"
	"sync/atomic"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Producer struct {
	cm *connectionManager
}

func NewProducer(cfg *Config) (*Producer, error) {
	cm := newConnectionManager(cfg)
	if err := cm.start(); err != nil {
		return nil, err
	}
	return &Producer{cm: cm}, nil
}

// 普通 publish
func (p *Producer) Publish(ctx context.Context, exchange, routingKey string, msg *xmq.Message) error {
	return p.publish(ctx, exchange, routingKey, msg)
}

// 延迟消息 publish，优先使用 x-delay，失败降级 TTL+DLX
func (p *Producer) PublishDelayed(ctx context.Context, delayedExchange, routingKey string, msg *xmq.Message, delayMillis int64) error {
	if msg.Headers == nil {
		msg.Headers = make(map[string]interface{})
	}
	msg.Headers["x-delay"] = delayMillis
	err := p.publish(ctx, delayedExchange, routingKey, msg)
	if err == nil {
		return nil
	}
	// 降级 TTL+DLX
	delete(msg.Headers, "x-delay")
	return p.publishWithTTL(ctx, delayedExchange, routingKey, msg, delayMillis)
}

// TTL publish（降级）
func (p *Producer) publishWithTTL(ctx context.Context, exchange, routingKey string, msg *xmq.Message, ttlMillis int64) error {
	if msg.Headers == nil {
		msg.Headers = make(map[string]interface{})
	}

	cc, err := p.cm.GetProducerChannel(ctx)
	if err != nil {
		return err
	}
	defer p.cm.PutProducerChannel(cc)

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
		Expiration:   fmt.Sprintf("%d", ttlMillis),
	}
	return cc.publishWithConfirm(ctx, exchange, routingKey, pub, p.cm.cfg.ConfirmTimeout)
}

// 内部通用 publish
func (p *Producer) publish(ctx context.Context, exchange, routingKey string, msg *xmq.Message) error {
	if p.cm.isClosed() {
		return xmq.ErrClosed
	}
	if msg.MessageId == "" {
		msg.MessageId = fmt.Sprintf("%d", time.Now().UnixNano())
	}
	msg.Timestamp = time.Now()

	cc, err := p.cm.GetProducerChannel(ctx)
	if err != nil {
		return err
	}
	defer p.cm.PutProducerChannel(cc)

	// 计数 inflight
	atomic.AddInt64(&p.cm.inflight, 1)
	defer atomic.AddInt64(&p.cm.inflight, -1)

	pub := amqp.Publishing{
		Body:         msg.Body,
		Headers:      amqp.Table(msg.Headers),
		Timestamp:    msg.Timestamp,
		MessageId:    msg.MessageId,
		DeliveryMode: amqp.Persistent,
	}
	return cc.publishWithConfirm(ctx, exchange, routingKey, pub, p.cm.cfg.ConfirmTimeout)
}

// 关闭 producer
func (p *Producer) Close(ctx context.Context) error {
	return p.cm.Close(ctx)
}
