package rabbitmq

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"snowgo/pkg/xmq"
	"sync"
	"sync/atomic"
	"time"
)

// consumerHandler 处理单 topic+group 的消息
type consumerHandler struct {
	topic    string
	group    string
	queue    string
	handler  xmq.Handler
	connMgr  *connectionManager
	stopCh   chan struct{}
	wg       *sync.WaitGroup
	logger   xmq.Logger
	inflight *int64
}

// runWorker 启动消费 goroutine
func (h *consumerHandler) runWorker() {
	defer h.wg.Done()

	backoff := 500 * time.Millisecond
	maxBackoff := 30 * time.Second

	for {
		select {
		case <-h.stopCh:
			return
		default:
		}

		ch, err := h.connMgr.getConsumerChannel(h.queue, "", "", 1)
		if err != nil {
			h.logger.Warn("get consumer channel failed, retrying", zap.Error(err))
			time.Sleep(backoff)
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}
		backoff = 500 * time.Millisecond // 成功获取 channel，重置 backoff

		msgs, err := ch.Consume(
			h.queue,
			"",
			false, // autoAck false
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			h.logger.Warn("consume failed, retrying", zap.Error(err))
			ch.Close()
			time.Sleep(backoff)
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}

	consumeLoop:
		for {
			select {
			case <-h.stopCh:
				ch.Close()
				return
			case msg, ok := <-msgs:
				if !ok {
					break consumeLoop
				}
				atomic.AddInt64(h.inflight, 1)
				func() {
					defer atomic.AddInt64(h.inflight, -1)
					m := xmq.Message{
						Body:      msg.Body,
						Headers:   msg.Headers,
						Timestamp: msg.Timestamp,
						MessageId: msg.MessageId,
					}
					if err := h.handler(context.Background(), m); err != nil {
						_ = msg.Nack(false, true) // 失败重试
					} else {
						_ = msg.Ack(false)
					}
				}()
			}
		}
		ch.Close()
	}
}

// Consumer 实现 xmq.Consumer
type Consumer struct {
	cfg      *Config
	connMgr  *connectionManager
	handlers sync.Map // key = topic_group, value = *consumerHandler
	stopCh   chan struct{}
	wg       sync.WaitGroup
	logger   xmq.Logger
	inflight int64
}

func NewConsumer(cfg *Config, connMgr *connectionManager) *Consumer {
	return &Consumer{
		cfg:     cfg,
		connMgr: connMgr,
		stopCh:  make(chan struct{}),
		logger:  cfg.Logger,
	}
}

// Register 注册 topic + group
func (c *Consumer) Register(topic, group string, handler xmq.Handler) error {
	if topic == "" || group == "" || handler == nil {
		return fmt.Errorf("invalid consumer registration")
	}
	key := fmt.Sprintf("%s_%s", topic, group)
	queueName := key

	ch := &consumerHandler{
		topic:    topic,
		group:    group,
		queue:    queueName,
		handler:  handler,
		connMgr:  c.connMgr,
		stopCh:   c.stopCh,
		wg:       &c.wg,
		logger:   c.logger,
		inflight: &c.inflight,
	}
	c.handlers.Store(key, ch)
	return nil
}

// Start 启动所有注册的 consumerHandler
func (c *Consumer) Start(ctx context.Context) error {
	c.handlers.Range(func(key, value any) bool {
		h := value.(*consumerHandler)
		c.wg.Add(1)
		go h.runWorker()
		return true
	})
	return nil
}

// Stop 优雅关闭，等待 inflight 消息处理完成
func (c *Consumer) Stop(ctx context.Context) error {
	close(c.stopCh)

	doneCh := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(doneCh)
	}()

	if ctx == nil {
		ctx = context.Background()
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-doneCh:
		return nil
	}
}
