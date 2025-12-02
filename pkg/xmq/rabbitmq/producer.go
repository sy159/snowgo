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
	cm  *connectionManager
	cfg *Config
}

// Publish 同步发送消息，保证 confirm 成功
func (p *Producer) Publish(ctx context.Context, exchange, routingKey string, msg *xmq.Message) error {
	atomic.AddInt64(&p.cm.inflight, 1)
	defer atomic.AddInt64(&p.cm.inflight, -1)

	if p.cm.isClosed() {
		return xmq.ErrClosed
	}

	if p.cfg.BeforePublish != nil {
		if err := p.cfg.BeforePublish(msg); err != nil {
			return err
		}
	}

	cc, err := p.cm.getProducerChannel(ctx)
	if err != nil {
		return err
	}
	defer p.cm.putProducerChannel(cc)

	pub := amqp.Publishing{
		Body:      msg.Body,
		Headers:   amqp.Table(msg.Headers),
		Timestamp: time.Now(),
		MessageId: msg.MessageId,
	}

	return cc.publishWithConfirm(ctx, exchange, routingKey, pub, p.cfg.PublishConfirmTimeout)
}

// PublishDelayed 支持 x-delayed-message 插件
func (p *Producer) PublishDelayed(ctx context.Context, delayedExchange, routingKey string, msg *xmq.Message, delayMillis int64) error {
	if !p.cfg.EnableXDelayed {
		return fmt.Errorf("x-delayed-message disabled")
	}
	if msg.Headers == nil {
		msg.Headers = make(map[string]interface{})
	}
	msg.Headers["x-delay"] = delayMillis
	return p.Publish(ctx, delayedExchange, routingKey, msg)
}

// Close 优雅关闭，等待 inflight 消息完成
func (p *Producer) Close(ctx context.Context) error {
	if err := p.cm.close(); err != nil {
		return err
	}

	wait := time.NewTimer(10 * time.Second)
	defer wait.Stop()
	for {
		if atomic.LoadInt64(&p.cm.inflight) == 0 {
			return nil
		}
		select {
		case <-wait.C:
			return fmt.Errorf("graceful close timeout, inflight=%d", atomic.LoadInt64(&p.cm.inflight))
		default:
			time.Sleep(50 * time.Millisecond)
		}
	}
}
