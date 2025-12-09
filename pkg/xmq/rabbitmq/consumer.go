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

// 注册单元
type consumerUnit struct {
	exchange   string
	routingKey string
	queue      string
	handler    xmq.Handler
	meta       *consumerMeta
}

type consumerMeta struct {
	Prefetch   int // Qos prefetch count，未ack消息上限，针对的是单个consumer，不是单个cannel
	WorkerNum  int // 消费数量
	RetryLimit int // 消费失败重试次数
}

func defaultMeta() *consumerMeta { return &consumerMeta{Prefetch: 4, WorkerNum: 2, RetryLimit: 3} }

// ConsumerOption 函数类型
type ConsumerOption func(*consumerMeta)

func WithPrefetch(n int) ConsumerOption   { return func(m *consumerMeta) { m.Prefetch = n } }
func WithWorkerNum(n int) ConsumerOption  { return func(m *consumerMeta) { m.WorkerNum = n } }
func WithRetryLimit(n int) ConsumerOption { return func(m *consumerMeta) { m.RetryLimit = n } }

// Consumer 封装 consumerConnManager 和 worker 管理
type Consumer struct {
	cm     *consumerConnManager
	cfg    *ConsumerConnConfig
	logger xmq.Logger

	stopped int32
	wg      sync.WaitGroup
	units   sync.Map
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
	return &Consumer{
		cm:     cm,
		cfg:    cfg,
		logger: cfg.Logger,
	}, nil
}

// StartWorker 启动 workerCount 个 worker，每个 worker 拥有独立 channel。
// prefetch: Qos prefetch count
// handler: 业务函数，返回 error 则触发受控重试
func (c *Consumer) startWorker(ctx context.Context, queue string, prefetch int, workerCount int, retryLimit int, handler xmq.Handler) error {
	if c == nil || c.cm == nil {
		return fmt.Errorf("consumer not initialized")
	}
	for i := 0; i < workerCount; i++ {
		c.wg.Add(1)
		go c.workerLoop(ctx, queue, prefetch, i, retryLimit, handler)
	}
	return nil
}

// workerLoop 每个 worker 的执行体
func (c *Consumer) workerLoop(ctx context.Context, queue string, prefetch, workerID, retryLimit int, handler xmq.Handler) {
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

		_ = ch.Qos(prefetch, 0, false)
		consumerTag := fmt.Sprintf("%s-%d", queue, workerID)
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

					err := handler(ctx, m)
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

						if retry <= retryLimit {
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

// Register 消费注册
func (c *Consumer) Register(ctx context.Context, exchange, routingKey string, handler xmq.Handler, opts ...interface{}) error {
	if exchange == "" || routingKey == "" || handler == nil {
		return fmt.Errorf("register: exchange/routingKey/handler must be non-empty")
	}
	key := exchange + "|" + routingKey
	meta := defaultMeta()
	for _, o := range opts {
		if fn, ok := o.(ConsumerOption); ok {
			fn(meta)
		}
	}
	if _, loaded := c.units.LoadOrStore(key, &consumerUnit{
		exchange:   exchange,
		routingKey: routingKey,
		queue:      fmt.Sprintf("%s.%s", exchange, routingKey),
		handler:    handler,
		meta:       meta,
	}); loaded {
		return fmt.Errorf("register: duplicate handler for %s", key)
	}
	return nil
}

// Start 统一启动所有注册组（阻塞直到 ctx 取消）
func (c *Consumer) Start(ctx context.Context) error {
	var err error
	c.units.Range(func(_, v interface{}) bool {
		u := v.(*consumerUnit)
		if e := c.startWorker(ctx, u.queue, u.meta.Prefetch, u.meta.WorkerNum, u.meta.RetryLimit, u.handler); e != nil {
			err = e
			return false
		}
		return true
	})
	if err != nil {
		return err
	}
	<-ctx.Done()
	return nil
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
	c.cm.Close(ctx)
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
