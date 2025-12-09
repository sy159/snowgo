package rabbitmq

import (
	"context"
	"fmt"
	"os"
	"snowgo/pkg/xmq"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// 注册单元
type consumerUnit struct {
	queue   string
	handler xmq.Handler
	meta    *consumerMeta
}

type consumerMeta struct {
	Prefetch       int           // Qos prefetch count，未ack消息上限，针对的是单个consumer，不是单个cancel
	WorkerNum      int           // 消费数量
	RetryLimit     int           // 消费失败重试次数
	HandlerTimeout time.Duration // handler超时,防止一直阻塞
}

func defaultMeta() *consumerMeta {
	return &consumerMeta{Prefetch: 4, WorkerNum: 4, RetryLimit: 3, HandlerTimeout: 60 * time.Second}
}

// ConsumerOption 函数类型
type ConsumerOption func(*consumerMeta)

// WithPrefetch 设置 Qos prefetch count
func WithPrefetch(n int) ConsumerOption { return func(m *consumerMeta) { m.Prefetch = n } }

// WithWorkerNum 设置消费者的工作协程数量
func WithWorkerNum(n int) ConsumerOption { return func(m *consumerMeta) { m.WorkerNum = n } }

// WithRetryLimit 设置消费失败重试次数
func WithRetryLimit(n int) ConsumerOption { return func(m *consumerMeta) { m.RetryLimit = n } }

// WithHandlerTimeout 设置handler超时
func WithHandlerTimeout(n time.Duration) ConsumerOption {
	return func(m *consumerMeta) { m.HandlerTimeout = n }
}

// Consumer 封装 consumerConnManager 和 worker 管理
type Consumer struct {
	cm     *consumerConnManager
	cfg    *ConsumerConnConfig
	logger xmq.Logger

	stopped    int32
	wg         sync.WaitGroup
	units      sync.Map
	instanceID string // 实例ID
}

// NewConsumer 使用 ConsumerConfig 创建 Consumer 实例
func NewConsumer(ctx context.Context, cfg *ConsumerConnConfig) (*Consumer, error) {
	if cfg == nil {
		return nil, fmt.Errorf("consumer config is nil")
	}
	if cfg.URL == "" {
		return nil, fmt.Errorf("consumer config URL must be non-empty")
	}
	if cfg.Logger == nil {
		cfg.Logger = &nopLogger{}
	}
	cm := newConsumerConnManager(ctx, cfg)
	if err := cm.Start(ctx); err != nil {
		cm.logger.Error(
			ctx,
			"consumer conn fail",
			zap.String("event", xmq.EventConsumerConnection),
			zap.Error(err),
		)
		return nil, fmt.Errorf("consumer start failed: %w", err)
	}
	hn, _ := os.Hostname() // 忽略错误，若错误可退回 uuid
	instanceID := fmt.Sprintf("%s_%d", hn, os.Getpid())
	return &Consumer{
		cm:         cm,
		cfg:        cfg,
		logger:     cfg.Logger,
		instanceID: instanceID,
	}, nil
}

// StartWorker 启动 workerCount 个 worker，每个 worker 拥有独立 channel。
// prefetch: Qos prefetch count
// handler: 业务函数，返回 error 则触发受控重试
func (c *Consumer) startWorker(ctx context.Context, queue string, prefetch int, workerCount int, retryLimit int, handlerTimeout time.Duration, handler xmq.Handler) error {
	if c == nil || c.cm == nil {
		return fmt.Errorf("consumer not initialized")
	}
	for i := 0; i < workerCount; i++ {
		go func(idx int) {
			c.wg.Add(1)
			defer c.wg.Done()
			c.workerLoop(ctx, queue, prefetch, idx, retryLimit, handlerTimeout, handler)
		}(i)
	}
	return nil
}

// workerLoop 每个 worker 的执行体
func (c *Consumer) workerLoop(ctx context.Context, queue string, prefetch, workerID, retryLimit int, handlerTimeout time.Duration, handler xmq.Handler) {
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
		ch, err := c.cm.GetConsumerChannel(ctx)
		if err != nil {
			c.logger.Warn(
				ctx,
				fmt.Sprintf("consumer get channel failed, err is: %s", err),
				zap.String("event", xmq.EventConsumerChannel),
			)
			time.Sleep(backoff)
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}

		// 设置Qos
		if err := ch.Qos(prefetch, 0, false); err != nil {
			_ = ch.Close()
			c.logger.Warn(
				ctx,
				fmt.Sprintf("ch.Qos failed queue=%s err=%v", queue, err),
				zap.String("event", xmq.EventConsumerChannel),
			)
			time.Sleep(backoff)
			if backoff < maxBackoff {
				backoff *= 2
			}
			continue
		}
		consumerTag := fmt.Sprintf("%s-%d-%s", queue, workerID, c.instanceID)
		msgs, err := ch.Consume(queue, consumerTag, false, false, false, false, nil)
		if err != nil {
			_ = ch.Close()
			c.logger.Warn(
				ctx,
				fmt.Sprintf("consumer start consume failed queue is %s, err is: %s", queue, err),
				zap.String("event", xmq.EventConsumerConsume),
			)
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
							c.logger.Error(
								ctx,
								fmt.Sprintf("consumer handler panic， queue: %s, workID: %d", queue, workerID),
								zap.String("event", xmq.EventConsumerConsume),
								zap.Any("error", r),
							)
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

					handlerCtx, cancel := context.WithTimeout(ctx, handlerTimeout)
					defer cancel()
					err := handler(handlerCtx, m)
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
						if retry <= retryLimit {
							if nackErr := d.Nack(false, true); nackErr != nil {
								c.logger.Warn(
									handlerCtx,
									fmt.Sprintf("d.Nack failed queue=%s message_id=%s err=%v", queue, d.MessageId, nackErr),
									zap.String("event", xmq.EventConsumerConsume),
								)
							}
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

// Register 消费注册
func (c *Consumer) Register(ctx context.Context, queue string, handler xmq.Handler, opts ...interface{}) error {
	if queue == "" || handler == nil {
		return fmt.Errorf("register: queue/handler must be non-empty")
	}
	meta := defaultMeta()
	for _, o := range opts {
		if fn, ok := o.(ConsumerOption); ok {
			fn(meta)
		}
	}
	if _, loaded := c.units.LoadOrStore(queue, &consumerUnit{
		queue:   queue,
		handler: handler,
		meta:    meta,
	}); loaded {
		return fmt.Errorf("register: duplicate handler for %s", queue)
	}
	return nil
}

// Start 统一启动所有注册组（阻塞直到 ctx 取消）
// 若任一队列启动失败，已启动的 workers 会被尝试停止（优雅回滚），然后返回 error
func (c *Consumer) Start(ctx context.Context) error {
	var (
		err         error
		startedKeys []string
	)

	// iterate registrations and start workers; if any fails, break and stop started ones
	c.units.Range(func(k, v interface{}) bool {
		u := v.(*consumerUnit)
		if e := c.startWorker(ctx, u.queue, u.meta.Prefetch, u.meta.WorkerNum, u.meta.RetryLimit,
			u.meta.HandlerTimeout, u.handler); e != nil {
			err = fmt.Errorf("start worker for queue %s failed: %w", u.queue, e)
			return false
		}
		// record started key for potential rollback
		if ks, ok := k.(string); ok {
			startedKeys = append(startedKeys, ks)
		}
		return true
	})

	if err != nil {
		// 如果有消费启动失败，则尝试停止已启动的 worker
		c.logger.Warn(
			ctx,
			fmt.Sprintf("start consumer error %s, rolling back %d started queues: %s",
				err, len(startedKeys), strings.Join(startedKeys, ", ")),
			zap.String("event", xmq.EventConsumerConnection),
		)
		// Use a short timeout context for rollback to avoid blocking forever
		rollbackCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = c.Stop(rollbackCtx)
		return err
	}
	<-ctx.Done()
	return nil
}

// Stop 优雅停止：关闭连接管理，并等待 worker goroutine 完全退出 当ctx 被取消时，会等待一个短暂的时间让wg归零后再返回（尽可能清理资源）
func (c *Consumer) Stop(ctx context.Context) error {
	if c == nil {
		return nil
	}
	if !atomic.CompareAndSwapInt32(&c.stopped, 0, 1) {
		return nil
	}

	// 关闭 consumerConnManager（触发 channel/connection close）
	c.cm.Close(ctx)

	// 等待 wg 或 ctx.Done；当 ctx.Done 触发时，仍然尝试再等待一个 gracePeriod
	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		// ctx cancelled; still wait up to 5s for cleanup
		timer := time.NewTimer(5 * time.Second)
		select {
		case <-done:
			timer.Stop()
			return nil
		case <-timer.C:
			// timed out waiting for wg after ctx cancelled
			return ctx.Err()
		}
	}
}
