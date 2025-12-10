package rabbitmq

import (
	"context"
	"fmt"
	"os"
	"snowgo/pkg/xmq"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// consumerUnit 表示注册的队列消费单元
type consumerUnit struct {
	queue         string
	handler       xmq.Handler
	meta          *xmq.ConsumerMeta
	notFoundCount int32 // 连续 404 / 队列不存在计数
}

func defaultMeta() *xmq.ConsumerMeta {
	return &xmq.ConsumerMeta{
		Prefetch:       4,
		WorkerNum:      4,
		RetryLimit:     2,
		HandlerTimeout: 60 * time.Second,
	}
}

// Consumer 结构
type Consumer struct {
	cm     *consumerConnManager
	cfg    *ConsumerConnConfig
	logger xmq.Logger

	stopped    int32
	wg         sync.WaitGroup
	units      sync.Map // key: queue string -> *consumerUnit
	instanceID string

	stopCtx    context.Context
	stopCancel context.CancelFunc
}

const (
	defaultMaxNotFound = 5
	maxConsumerTagLen  = 250 // AMQP限制最多255
	defaultBackoff     = 1 * time.Second
	defaultMaxBackoff  = 30 * time.Second
)

// sanitizeInstanceID 简单清洗
func sanitizeInstanceID(s string) string {
	s = strings.ReplaceAll(s, ":", "_")
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "\\", "_")
	s = strings.ReplaceAll(s, " ", "_")
	return s
}

// NewConsumer 创建
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
		cm.logger.Error(ctx,
			"consumer conn fail",
			zap.String("event", xmq.EventConsumerConnection),
			zap.Error(err),
		)
		return nil, fmt.Errorf("consumer start failed: %w", err)
	}
	hn, _ := os.Hostname()
	instanceID := sanitizeInstanceID(fmt.Sprintf("%s_%d", hn, os.Getpid()))
	return &Consumer{
		cm:         cm,
		cfg:        cfg,
		logger:     cfg.Logger,
		instanceID: instanceID,
	}, nil
}

// startWorker spawn worker goroutine
func (c *Consumer) startWorker(ctx context.Context, u *consumerUnit) error {
	if c == nil || c.cm == nil {
		return fmt.Errorf("consumer not initialized")
	}
	for i := 0; i < u.meta.WorkerNum; i++ {
		idx := i
		// add BEFORE launching goroutine => 避免 wg race
		c.wg.Add(1)
		go func(unit *consumerUnit, workerIdx int) {
			defer c.wg.Done()
			c.workerLoop(ctx, unit, workerIdx)
		}(u, idx)
	}
	return nil
}

func sleepWithContext(ctx context.Context, d time.Duration) {
	if d <= 0 {
		return
	}
	select {
	case <-ctx.Done():
		return
	case <-time.After(d):
	}
}

// workerLoop
func (c *Consumer) workerLoop(ctx context.Context, unit *consumerUnit, workerID int) {
	c.logger.Info(ctx,
		fmt.Sprintf("queue %s consumer work %d is start...", unit.queue, workerID),
		zap.String("event", xmq.EventConsumerConsume),
	)
	meta := unit.meta

	initialBackoff := c.cfg.ConsumerBackoffBase
	if initialBackoff <= 0 {
		initialBackoff = defaultBackoff
	}
	backoff := initialBackoff
	maxBackoff := c.cfg.ReconnectMaxDelay
	if maxBackoff <= 0 {
		maxBackoff = defaultMaxBackoff
	}

	for {
		// stop
		select {
		case <-ctx.Done():
			return
		default:
		}

		ch, err := c.cm.GetConsumerChannel(ctx)
		if err != nil {
			c.logger.Warn(ctx,
				fmt.Sprintf("consumer get channel failed err=%v", err),
				zap.String("event", xmq.EventConsumerChannel),
			)
			//time.Sleep(backoff)
			sleepWithContext(ctx, backoff)
			backoff = growBackoff(backoff, maxBackoff)
			continue
		}

		// Qos
		if err := ch.Qos(meta.Prefetch, 0, false); err != nil {
			_ = ch.Close()
			c.logger.Warn(ctx,
				fmt.Sprintf("ch set qos failed queue=%s err=%v", unit.queue, err),
				zap.String("event", xmq.EventConsumerChannel),
			)
			//time.Sleep(backoff)
			sleepWithContext(ctx, backoff)
			backoff = growBackoff(backoff, maxBackoff)
			continue
		}

		// consumerTag sanitize + truncate
		consumerTag := fmt.Sprintf("c-%s-%d-%s", unit.queue, workerID, c.instanceID)
		if len(consumerTag) > maxConsumerTagLen {
			consumerTag = consumerTag[:maxConsumerTagLen]
		}

		consumerMessages, err := ch.Consume(unit.queue, consumerTag, false, false, false, false, nil)
		if err != nil {
			_ = ch.Close()
			errStr := strings.ToLower(err.Error())
			if strings.Contains(errStr, "not found") || strings.Contains(errStr, "404") {
				// 队列未找到，增加计数
				n := atomic.AddInt32(&unit.notFoundCount, 1)
				if n >= defaultMaxNotFound {
					c.logger.Warn(ctx,
						fmt.Sprintf("%d times queue not found: %s, worker exiting", n, err),
						zap.String("event", xmq.EventConsumerChannel),
					)
					return // 达到上限，退出当前 worker
				} else {
					// 等待一段时间再重试
					//time.Sleep(backoff)
					sleepWithContext(ctx, backoff)
					backoff = growBackoff(backoff, maxBackoff)
					continue
				}
			} else {
				// 其他错误，继续做 exponential backoff
				c.logger.Warn(ctx,
					fmt.Sprintf("consumer start consume failed queue=%s err=%v", unit.queue, err),
					zap.String("event", xmq.EventConsumerConsume),
				)
				//time.Sleep(backoff)
				sleepWithContext(ctx, backoff)
				backoff = growBackoff(backoff, maxBackoff)
				continue
			}
		}

		// consume 成功，reset notFoundCount 与 backoff
		atomic.StoreInt32(&unit.notFoundCount, 0)
		backoff = initialBackoff

		// 消费循环
	consumeLoop:
		for {
			select {
			case <-ctx.Done():
				c.logger.Info(ctx,
					fmt.Sprintf("ctx down queue %s consumer tag %s is end...", unit.queue, consumerTag),
					zap.String("event", xmq.EventConsumerConsume),
				)
				_ = ch.Close()
				return
			case d, ok := <-consumerMessages:
				if !ok {
					_ = ch.Close()
					break consumeLoop
				}
				// process with panic capture (panic -> Nack(false,false) to avoid tight-loop)
				func(delivery amqp.Delivery) {
					defer func() {
						if r := recover(); r != nil {
							if ne := delivery.Nack(false, false); ne != nil {
								c.logger.Warn(ctx,
									fmt.Sprintf("d.Nack(false,false) failed after panic queue=%s message_id=%s err=%v", unit.queue, delivery.MessageId, ne),
									zap.String("event", xmq.EventConsumerConsumeACk),
								)
							}
							c.logger.Error(ctx,
								fmt.Sprintf("consumer handler panic recovered queue=%s message_id=%s", unit.queue, delivery.MessageId),
								zap.String("event", xmq.EventConsumerConsume),
								zap.Any("error", r),
							)
						}
					}()
					c.handleDeliveryInProcess(ctx, ch, unit, delivery)
				}(d)
			}
		}

		_ = ch.Close()
		//time.Sleep(1 * time.Second)
		sleepWithContext(ctx, 1*time.Second)
		c.logger.Info(ctx,
			fmt.Sprintf("queue %s consumer tag %s is end...", unit.queue, consumerTag),
			zap.String("event", xmq.EventConsumerConsume),
		)
	}
}

// handleDeliveryInProcess: 在进程内重试，失败后记录错误日志
func (c *Consumer) handleDeliveryInProcess(parentCtx context.Context, ch *amqp.Channel, unit *consumerUnit, d amqp.Delivery) {
	meta := unit.meta
	// 处理异常情况
	if meta.Prefetch <= 0 {
		meta.Prefetch = 1
	}
	if meta.WorkerNum <= 0 {
		meta.WorkerNum = 1
	}
	if meta.RetryLimit <= 0 {
		meta.RetryLimit = 1
	}
	if meta.HandlerTimeout <= 0 {
		meta.HandlerTimeout = 60 * time.Second
	}

	// prepare message
	headers := map[string]interface{}{}
	for k, v := range d.Headers {
		headers[k] = v
	}
	msg := xmq.Message{
		Body:      d.Body,
		Headers:   headers,
		Timestamp: d.Timestamp,
		MessageId: d.MessageId,
	}

	var lastErr error
	for attempt := 0; attempt < meta.RetryLimit; attempt++ {
		// 重试消费
		handlerCtx, cancel := context.WithTimeout(parentCtx, meta.HandlerTimeout)
		startTime := time.Now()
		lastErr = unit.handler(handlerCtx, msg)
		cancel()
		duration := time.Since(startTime)
		if lastErr == nil {
			c.logger.Info(handlerCtx,
				"consumer msg success",
				zap.String("event", xmq.EventConsumerConsume),
				zap.String("queue", unit.queue),
				zap.String("message_id", d.MessageId),
				zap.String("message_body", ""),
				zap.Any("message_header", ""),
				zap.Duration("duration", duration),
				zap.String("status", "success"),
				zap.Int("attempt", attempt+1),
			)
			if ackErr := d.Ack(false); ackErr != nil {
				c.logger.Warn(handlerCtx,
					fmt.Sprintf("ack failed queue=%s message_id=%s err=%v", unit.queue, d.MessageId, ackErr),
					zap.String("event", xmq.EventConsumerConsumeACk),
				)
			}
			return
		} else {
			// 错误也需要记录
			c.logger.Info(handlerCtx,
				"consumer msg fail",
				zap.String("event", xmq.EventConsumerConsume),
				zap.String("queue", unit.queue),
				zap.String("message_id", d.MessageId),
				zap.String("message_body", string(d.Body)),
				zap.Any("message_header", d.Headers),
				zap.Duration("duration", duration),
				zap.String("status", "fail"),
				zap.Int("attempt", attempt+1),
				zap.Error(lastErr),
			)
		}
		// short backoff between in-process attempts
		//time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
		sleepWithContext(handlerCtx, time.Duration(attempt+1)*100*time.Millisecond)
	}

	// 超过重试次数 -> 如果需要，可把未消费消息发送死信队列

	// Ack 原消息，避免无限重试 / redelivery（因为我们已记录失败）
	ackCtx, cancel := context.WithTimeout(parentCtx, 5*time.Second)
	defer cancel()
	if ackErr := d.Ack(false); ackErr != nil {
		c.logger.Warn(ackCtx,
			fmt.Sprintf("ack failed queue=%s message_id=%s err=%v", unit.queue, d.MessageId, ackErr),
			zap.String("event", xmq.EventConsumerConsumeACk),
		)
	}
}

// Register 注册消费
func (c *Consumer) Register(ctx context.Context, queue string, handler xmq.Handler, meta *xmq.ConsumerMeta) error {
	if queue == "" || handler == nil {
		return fmt.Errorf("register: queue/handler must be non-empty")
	}
	if meta == nil {
		meta = defaultMeta()
	}

	u := &consumerUnit{
		queue:   queue,
		handler: handler,
		meta:    meta,
	}
	if _, loaded := c.units.LoadOrStore(queue, u); loaded {
		return fmt.Errorf("register: duplicate handler for %s", queue)
	}
	return nil
}

// Start 启动所有注册的消费者（创建内部 stopCtx）
func (c *Consumer) Start(ctx context.Context) error {
	c.stopCtx, c.stopCancel = context.WithCancel(ctx)

	var err error
	c.units.Range(func(_, v interface{}) bool {
		u := v.(*consumerUnit)
		if e := c.startWorker(c.stopCtx, u); e != nil {
			err = fmt.Errorf("start worker for queue %s failed: %w", u.queue, e)
			return false
		}
		return true
	})
	if err != nil {
		c.logger.Warn(ctx,
			fmt.Sprintf("start worker error: %v", err),
			zap.String("event", xmq.EventConsumerConnection),
		)
		_ = c.Stop(context.Background())
		return err
	}

	<-ctx.Done()
	return nil
}

func growBackoff(curr, max time.Duration) time.Duration {
	curr *= 2
	if curr > max {
		return max
	}
	return curr
}

// Stop 优雅停止，关闭连接管理，并等待 worker goroutine 完全退出 当ctx 被取消时，会等待一个短暂的时间让wg归零后再返回（尽可能清理资源）
func (c *Consumer) Stop(ctx context.Context) error {
	if c == nil {
		return nil
	}
	if !atomic.CompareAndSwapInt32(&c.stopped, 0, 1) {
		return nil
	}

	// cancel internal stop ctx
	if c.stopCancel != nil {
		c.stopCancel()
	}

	// close conn manager
	c.cm.Close(ctx)

	// 等待 wg 或 ctx.Done；当 ctx.Done 触发时，仍然尝试再等待3s
	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		// ctx cancelled; still wait up to 3s for cleanup
		timer := time.NewTimer(3 * time.Second)
		select {
		case <-done:
			timer.Stop()
			return nil
		case <-timer.C:
			return ctx.Err()
		}
	}
}
