package rabbitmq

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"snowgo/pkg/xmq"
)

// Producer 生产者实例
type Producer struct {
	cm *connectionManager
}

// NewProducer 新建 Producer（底层会启动连接与 Channel 池）
func NewProducer(cfg *Config) (*Producer, error) {
	if cfg.Role != RoleProducer {
		return nil, fmt.Errorf("config role must be producer")
	}
	cm := newConnectionManager(cfg)
	if err := cm.start(); err != nil {
		return nil, err
	}
	return &Producer{cm: cm}, nil
}

// Publish 同步发布
func (p *Producer) Publish(ctx context.Context, exchange, routingKey string, msg *xmq.Message) error {
	return p.publish(ctx, exchange, routingKey, msg)
}

// PublishDelayed 先尝试 x-delay，失败降级 TTL+DLX
func (p *Producer) PublishDelayed(ctx context.Context, exchange, routingKey string, msg *xmq.Message, delayMillis int64) error {
	if msg.Headers == nil {
		msg.Headers = make(map[string]interface{})
	}
	msg.Headers["x-delay"] = delayMillis
	if err := p.publish(ctx, exchange, routingKey, msg); err == nil {
		return nil
	}
	// 降级
	delete(msg.Headers, "x-delay")
	return p.publishWithTTL(ctx, exchange, routingKey, msg, delayMillis)
}

/* ---------- 内部通用 publish ---------- */

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

	atomic.AddInt64(&p.cm.inflight, 1)
	defer atomic.AddInt64(&p.cm.inflight, -1)

	pub := amqp.Publishing{
		Body:         msg.Body,
		Headers:      amqp.Table(msg.Headers),
		Timestamp:    msg.Timestamp,
		MessageId:    msg.MessageId,
		DeliveryMode: amqp.Persistent,
	}
	return cc.publishWithConfirm(ctx, exchange, routingKey, pub, p.cm.cfg.ConfirmTimeout, p.cm)
}

func (p *Producer) publishWithTTL(ctx context.Context, exchange, routingKey string, msg *xmq.Message, ttlMillis int64) error {
	if msg.Headers == nil {
		msg.Headers = make(map[string]interface{})
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

	atomic.AddInt64(&p.cm.inflight, 1)
	defer atomic.AddInt64(&p.cm.inflight, -1)

	pub := amqp.Publishing{
		Body:         msg.Body,
		Headers:      amqp.Table(msg.Headers),
		Timestamp:    msg.Timestamp,
		MessageId:    msg.MessageId,
		DeliveryMode: amqp.Persistent,
		Expiration:   fmt.Sprintf("%d", ttlMillis),
	}
	return cc.publishWithConfirm(ctx, exchange, routingKey, pub, p.cm.cfg.ConfirmTimeout, p.cm)
}

// Close 优雅关闭
func (p *Producer) Close(ctx context.Context) error {
	return p.cm.Close(ctx)
}
