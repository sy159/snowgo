package rabbitmq

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"snowgo/pkg/xmq"
)

type consumerHandler struct {
	topic    string
	group    string
	queue    string
	handler  xmq.Handler
	meta     *consumerMeta
	cm       *connectionManager
	stopCh   chan struct{}
	wg       sync.WaitGroup
	logger   xmq.Logger
	inflight int64
}

func (h *consumerHandler) run() {
	for i := 0; i < h.meta.WorkerNum; i++ {
		h.wg.Add(1)
		go h.worker()
	}
}

func (h *consumerHandler) worker() {
	defer h.wg.Done()
	backoffBase := time.Second

	for {
		select {
		case <-h.stopCh:
			return
		default:
		}

		ch, err := h.cm.GetConsumerChannel(h.queue, h.topic, h.topic, h.meta.QoS)
		if err != nil {
			delay := backoffBase + time.Duration(rand.Intn(500))*time.Millisecond
			select {
			case <-time.After(delay):
				continue
			case <-h.stopCh:
				return
			}
		}

		closeCh := ch.NotifyClose(make(chan *amqp.Error, 1))
		go func(c *amqp.Channel) {
			err := <-closeCh
			if err != nil && h.logger != nil {
				h.logger.Warn("consumer channel closed by broker", "error", err.Error())
			}
			_ = c.Close()
		}(ch)

		msgs, err := ch.Consume(h.queue, h.group, false, false, false, false, nil)
		if err != nil {
			_ = ch.Close()
			delay := backoffBase + time.Duration(rand.Intn(500))*time.Millisecond
			select {
			case <-time.After(delay):
				continue
			case <-h.stopCh:
				return
			}
		}

	consumerLoop:
		for {
			select {
			case <-h.stopCh:
				_ = ch.Close()
				return
			case d, ok := <-msgs:
				if !ok {
					_ = ch.Close()
					break consumerLoop
				}

				atomic.AddInt64(&h.inflight, 1)
				func() {
					defer atomic.AddInt64(&h.inflight, -1)
					var hdrs map[string]interface{}
					if d.Headers != nil {
						hdrs = make(map[string]interface{}, len(d.Headers))
						for k, v := range d.Headers {
							hdrs[k] = v
						}
					}
					m := xmq.Message{
						Body:      d.Body,
						Headers:   hdrs,
						Timestamp: d.Timestamp,
						MessageId: d.MessageId,
					}
					if err := h.handler(context.Background(), m); err != nil {
						_ = d.Nack(false, false)
					} else {
						_ = d.Ack(false)
					}
				}()
			}
		}
	}
}

type Consumer struct {
	cm       *connectionManager
	cfg      *Config
	handlers sync.Map // key=topic|group, value=*consumerHandler
	stopCh   chan struct{}
	logger   xmq.Logger
	inflight int64
}

func NewConsumer(cfg *Config) (*Consumer, error) {
	cm := newConnectionManager(cfg)
	if err := cm.start(); err != nil {
		return nil, err
	}
	return &Consumer{
		cm:     cm,
		cfg:    cfg,
		stopCh: make(chan struct{}),
		logger: cfg.Logger,
	}, nil
}

func (c *Consumer) Register(topic, group string, handler xmq.Handler, opts ...interface{}) error {
	if topic == "" || group == "" || handler == nil {
		return fmt.Errorf("invalid registration")
	}
	key := topic + "|" + group
	if _, ok := c.handlers.Load(key); ok {
		return fmt.Errorf("handler duplicated")
	}
	meta := DefaultConsumerMeta()
	for _, opt := range opts {
		if fn, ok := opt.(ConsumerOption); ok {
			fn(meta)
		}
	}
	meta.Handler = handler
	h := &consumerHandler{
		topic:   topic,
		group:   group,
		queue:   fmt.Sprintf("%s.%s", topic, group),
		handler: handler,
		meta:    meta,
		cm:      c.cm,
		stopCh:  c.stopCh,
		logger:  c.logger,
	}
	c.handlers.Store(key, h)
	return nil
}

func (c *Consumer) Start(ctx context.Context) error {
	c.handlers.Range(func(_, v interface{}) bool {
		if h, ok := v.(*consumerHandler); ok {
			h.run()
		}
		return true
	})
	<-ctx.Done()
	return nil
}

func (c *Consumer) Stop(ctx context.Context) error {
	select {
	case <-c.stopCh:
	default:
		close(c.stopCh)
	}

	c.handlers.Range(func(_, v interface{}) bool {
		if h, ok := v.(*consumerHandler); ok {
			h.wg.Wait()
		}
		return true
	})

	deadline := time.After(30 * time.Second)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		var total int64
		c.handlers.Range(func(_, v interface{}) bool {
			if h, ok := v.(*consumerHandler); ok {
				total += atomic.LoadInt64(&h.inflight)
			}
			return true
		})
		if total == 0 {
			break
		}
		select {
		case <-deadline:
			return fmt.Errorf("consumer stop timeout, inflight=%d", total)
		case <-ticker.C:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return c.cm.Close(ctx)
}

func splitKey(key string) (topic, group string) {
	parts := strings.Split(key, "|")
	if len(parts) >= 2 {
		return parts[0], parts[1]
	}
	return key, ""
}
