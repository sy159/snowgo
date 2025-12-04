package rabbitmq

import (
	"context"
	"fmt"
	"snowgo/pkg/xmq"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// Consumer 封装 consumerConnManager 和 worker 管理
type Consumer struct {
	cm     *consumerConnManager
	cfg    *ConsumerConfig
	logger xmq.Logger

	stopped int32
	wg      sync.WaitGroup
}

// NewConsumerWithConfig 使用 ConsumerConfig 创建 Consumer 实例
func NewConsumerWithConfig(cfg *ConsumerConfig) (*Consumer, error) {
	if cfg == nil {
		return nil, fmt.Errorf("consumer config is nil")
	}
	if cfg.URL == "" {
		return nil, fmt.Errorf("consumer config URL must be non-empty")
	}
	if cfg.Logger == nil {
		cfg.Logger = &nopLogger{}
	}
	cm := newConsumerConnManager(cfg)
	if err := cm.Start(); err != nil {
		return nil, fmt.Errorf("consumer start failed: %w", err)
	}
	return &Consumer{
		cm:     cm,
		cfg:    cfg,
		logger: cfg.Logger,
	}, nil
}

// StartWorker 启动 workerCount 个 worker，每个 worker 拥有独立 channel。
// prefetch: Qos prefetch count
// handler: 业务函数，返回 error 则触发受控重试
func (c *Consumer) StartWorker(ctx context.Context, queue string, prefetch int, workerCount int, handler xmq.Handler) error {
	if c == nil || c.cm == nil {
		return fmt.Errorf("consumer not initialized")
	}
	for i := 0; i < workerCount; i++ {
		c.wg.Add(1)
		go c.workerLoop(ctx, queue, prefetch, i, handler)
	}
	return nil
}

// workerLoop 每个 worker 的执行体
func (c *Consumer) workerLoop(ctx context.Context, queue string, prefetch, workerID int, handler xmq.Handler) {
	defer c.wg.Done()

	backoff := c.cfg.ConsumerBackoffBase
	if backoff <= 0 {
		backoff = 1 * time.Second
	}
	maxBackoff := c.cfg.ReconnectMaxDelay
	if maxBackoff <= 0 {
		maxBackoff = 30 * time.Second
	}

	for {
		// stop check
		select {
		case <-ctx.Done():
			return
		default:
		}

		// 获取独立 channel
		ch, err := c.cm.GetConsumerChannel()
		if err != nil {
			if c.logger != nil {
				c.logger.Warn("consumer get channel failed", zap.String("error", err.Error()))
			}
			time.Sleep(backoff)
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}

		_ = ch.Qos(prefetch, 0, false)
		consumerTag := fmt.Sprintf("%s-%d", queue, workerID)
		msgs, err := ch.Consume(queue, consumerTag, false, false, false, false, nil)
		if err != nil {
			_ = ch.Close()
			if c.logger != nil {
				c.logger.Warn("consumer start consume failed", zap.String("queue", queue), zap.String("error", err.Error()))
			}
			time.Sleep(backoff)
			if backoff < maxBackoff {
				backoff *= 2
			}
			continue
		}

		// 成功进入消费循环，重置 backoff
		backoff = c.cfg.ConsumerBackoffBase

	consumerLoop:
		for {
			select {
			case <-ctx.Done():
				_ = ch.Close()
				return
			case d, ok := <-msgs:
				if !ok {
					_ = ch.Close()
					break consumerLoop
				}

				func() {
					defer func() {
						if r := recover(); r != nil {
							_ = d.Nack(false, true)
							if c.logger != nil {
								c.logger.Error("consumer handler panic", zap.Any("panic", r))
							}
						}
					}()
					hdrs := map[string]interface{}{}
					for k, v := range d.Headers {
						hdrs[k] = v
					}
					m := xmq.Message{
						Body:      d.Body,
						Headers:   hdrs,
						Timestamp: d.Timestamp,
						MessageId: d.MessageId,
					}

					err := handler(context.Background(), m)
					if err != nil {
						// 处理受控重试
						retry := 0
						if v, ok := d.Headers["x-retry-count"]; ok {
							switch vv := v.(type) {
							case int:
								retry = vv
							case int32:
								retry = int(vv)
							case int64:
								retry = int(vv)
							case string:
								if n, e := strconv.Atoi(vv); e == nil {
									retry = n
								}
							}
						}
						retry++
						// 写回 header
						d.Headers["x-retry-count"] = retry

						if retry <= c.cfg.ConsumerRetryLimit {
							_ = d.Nack(false, true)
							time.Sleep(100 * time.Millisecond) // 避免 tight loop
						} else {
							_ = d.Nack(false, false) // 超过限制交由 broker DLX 或丢弃
						}
					} else {
						_ = d.Ack(false)
					}
				}()
			}
		}

		// channel 出现问题，关闭并短暂等待后循环重建
		_ = ch.Close()
		time.Sleep(1 * time.Second)
	}
}

// Stop 优雅停止：关闭连接管理，并等待 worker goroutine 完全退出
func (c *Consumer) Stop(ctx context.Context) error {
	if c == nil {
		return nil
	}
	if !atomic.CompareAndSwapInt32(&c.stopped, 0, 1) {
		return nil
	}
	// 关闭 consumerConnManager
	c.cm.Close()
	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}
