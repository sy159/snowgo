package rabbitmq

import (
	"context"
	"fmt"
	"snowgo/pkg/xmq"
	"sync"
	"sync/atomic"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

////////////////////////////////////////////////////////////////////////////////
// confirmChannel
//
// 负责单个 amqp.Channel 的 Confirm 订阅与 publishWithConfirm。
// 关键点：
//  - pending 每条消息使用缓冲 1 的 channel 用于接收 confirm。
//  - confirmRouter 写 pending 时**绝不阻塞**（非阻塞写 + fallback close）
//  - 在 publishWithConfirm 中，GetNextPublishSeqNo() 若返回 0 => 立即回收 channel 并返回错误。
//  - publishWithConfirm 不做重试（上层 manager 会决定是否 retry/换 channel）。
////////////////////////////////////////////////////////////////////////////////

type pendingRec struct {
	ch   chan amqp.Confirmation
	once sync.Once
}

type confirmChannel struct {
	ch        *amqp.Channel
	logger    xmq.Logger // 使用项目级 Logger（xmq.Logger）
	closeCh   chan struct{}
	closeOnce sync.Once

	// mu 用于保护 GetNextPublishSeqNo + PublishWithContext 的原子性
	mu sync.Mutex

	// pending map: key = deliveryTag (uint64) -> *pendingRec
	pending sync.Map

	closed   int32 // atomic flag
	broken   int32 // atomic flag: channel 被标记为 broken（需要回收）
	consecTO int32 // 连续 confirm 超时计数（用于触发回收）
}

// newConfirmChannel: 创建并启用 confirm 模式（会启动 confirmRouter）
func newConfirmChannel(conn *amqp.Connection, logger xmq.Logger) (*confirmChannel, error) {
	if conn == nil {
		return nil, xmq.ErrNoConnection
	}
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	if err := ch.Confirm(false); err != nil {
		_ = ch.Close()
		return nil, fmt.Errorf("enable confirm: %w", err)
	}
	cc := &confirmChannel{
		ch:      ch,
		logger:  logger,
		closeCh: make(chan struct{}),
	}
	// start router
	go cc.confirmRouter()
	// watch NotifyClose to mark broken
	go func() {
		closeC := ch.NotifyClose(make(chan *amqp.Error, 1))
		if e := <-closeC; e != nil {
			atomic.StoreInt32(&cc.broken, 1)
			cc.close()
			if cc.logger != nil {
				cc.logger.Warn("confirmChannel: channel closed by broker", zap.String("error", e.Error()))
			}
		}
	}()
	return cc, nil
}

// confirmRouter: 读取 NotifyPublish 并路由给 pendingRec
// 非阻塞写：如果写会阻塞（意味着 publisher 已经离开/未等待），会 close pending ch 并回收 channel，避免泄露。
// 使用缓冲 1 的 rec.ch；写仍可能阻塞（如果接收者已退出且缓冲已满），因此使用 select non-blocking。
func (cc *confirmChannel) confirmRouter() {
	confSrc := cc.ch.NotifyPublish(make(chan amqp.Confirmation, 1))
	for {
		select {
		case c, ok := <-confSrc:
			if !ok {
				goto exit
			}
			v, loaded := cc.pending.Load(c.DeliveryTag)
			if !loaded {
				// 没有 pending：记录日志并继续
				if cc.logger != nil {
					cc.logger.Warn("confirmRouter: no pending for deliveryTag", zap.Uint64("tag", c.DeliveryTag))
				}
				continue
			}
			rec := v.(*pendingRec)
			// 非阻塞写：若阻塞则 close rec.ch 以唤醒 publisher，并回收 channel
			select {
			case rec.ch <- c:
				// success: ensure close once and delete
				rec.once.Do(func() { close(rec.ch) })
				cc.pending.Delete(c.DeliveryTag)
			default:
				// can't write without blocking -> wake publisher and recycle channel
				rec.once.Do(func() { close(rec.ch) })
				cc.pending.Delete(c.DeliveryTag)
				atomic.StoreInt32(&cc.broken, 1)
				cc.close()
				if cc.logger != nil {
					cc.logger.Warn("confirmRouter: pending write blocked, marking channel broken", zap.Uint64("tag", c.DeliveryTag))
				}
			}
		case <-cc.closeCh:
			goto exit
		}
	}
exit:
	atomic.StoreInt32(&cc.closed, 1)
	// close all pending
	cc.pending.Range(func(k, v interface{}) bool {
		if r, ok := v.(*pendingRec); ok {
			r.once.Do(func() { close(r.ch) })
			cc.pending.Delete(k)
		}
		return true
	})
}

// publishWithConfirm: 在 cc.mu 下保证 seq 与 publish 原子性
// - 若 GetNextPublishSeqNo() == 0 则表示 channel 已经 reset，需要回收 -> 返回错误
// - 等待 confirm 的逻辑会在 timeout 后清理 pending 并根据 consecutive threshold 决定是否回收 channel
func (cc *confirmChannel) publishWithConfirm(ctx context.Context, exchange, routingKey string, pub amqp.Publishing, timeout time.Duration, consecThreshold int32) error {
	// quick checks
	if atomic.LoadInt32(&cc.closed) == 1 || atomic.LoadInt32(&cc.broken) == 1 {
		return fmt.Errorf("confirmChannel closed or broken")
	}

	// lock to ensure seq/publish atomicity
	cc.mu.Lock()
	seq := cc.ch.GetNextPublishSeqNo()
	if seq == 0 {
		// seq==0: channel internal reset -> recycle and return error
		cc.mu.Unlock()
		atomic.StoreInt32(&cc.broken, 1)
		cc.close()
		return fmt.Errorf("confirmChannel invalid next seq 0")
	}
	rec := &pendingRec{ch: make(chan amqp.Confirmation, 1)}
	cc.pending.Store(seq, rec)

	// perform publish under the lock to avoid race between seq and publish
	err := cc.ch.PublishWithContext(ctx, exchange, routingKey, false, false, pub)
	cc.mu.Unlock()

	if err != nil {
		// publish failed: cleanup pending, mark broken, close channel
		cc.pending.Delete(seq)
		atomic.StoreInt32(&cc.broken, 1)
		cc.close()
		return fmt.Errorf("publish failed: %w", err)
	}

	// wait for confirm / timeout / ctx cancel / channel close
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case c, ok := <-rec.ch:
		if !ok {
			// pending closed - treat as error
			return fmt.Errorf("confirm channel closed")
		}
		if !c.Ack {
			// Nack from broker: treat as nack error and recycle channel
			atomic.StoreInt32(&cc.consecTO, 0)
			atomic.StoreInt32(&cc.broken, 1)
			cc.close()
			return xmq.ErrPublishNack
		}
		// Ack: reset consecutive timeout counter
		atomic.StoreInt32(&cc.consecTO, 0)
		return nil
	case <-timer.C:
		// confirm timeout: increment consecutive counter, possibly mark broken
		newCnt := atomic.AddInt32(&cc.consecTO, 1)
		cc.pending.Delete(seq)
		if consecThreshold > 0 && newCnt >= consecThreshold {
			atomic.StoreInt32(&cc.broken, 1)
			cc.close()
			if cc.logger != nil {
				cc.logger.Warn("confirmChannel consecutive confirm timeout reached", zap.Int32("consecutive", newCnt))
			}
		} else if cc.logger != nil {
			cc.logger.Warn("confirmChannel confirm timeout (transient)", zap.Int32("consecutive", newCnt), zap.Uint64("tag", seq))
		}
		return xmq.ErrPublishTimeout
	case <-ctx.Done():
		cc.pending.Delete(seq)
		return ctx.Err()
	case <-cc.closeCh:
		cc.pending.Delete(seq)
		return fmt.Errorf("confirmChannel closed")
	}
}

func (cc *confirmChannel) close() {
	cc.closeOnce.Do(func() {
		atomic.StoreInt32(&cc.closed, 1)
		select {
		case <-cc.closeCh:
		default:
			close(cc.closeCh)
		}
		if cc.ch != nil {
			_ = cc.ch.Close()
		}
	})
}

////////////////////////////////////////////////////////////////////////////////
// producerConnManager
//
//  - manages one AMQP connection for producers
//  - maintains a fixed-size pool of confirmChannel (channels in confirm mode)
//  - exposes Publish(ctx, exchange, routingKey, pub, timeout) which:
//      1) gets a confirmChannel from pool
//      2) increments inflight
//      3) calls cc.publishWithConfirm(...)
//      4) decrements inflight and returns channel to pool (with careful PutBackTimeout handling)
//  - reconnect watcher: deterministic exponential backoff (no jitter) and rebuild pool on reconnect
////////////////////////////////////////////////////////////////////////////////

type producerConnManager struct {
	cfg *ProducerConfig

	conn atomic.Value // *amqp.Connection

	pool   chan *confirmChannel
	poolMu sync.Mutex // protect refreshPool

	inflight int64 // 当前正在等待 confirm 的消息数量

	closed int32
	logger xmq.Logger

	wg sync.WaitGroup
}

func newProducerConnManager(cfg *ProducerConfig) *producerConnManager {
	if cfg == nil {
		panic("producer config nil")
	}
	// ensure logger non-nil
	var l xmq.Logger = &nopLogger{}
	if cfg.Logger != nil {
		l = cfg.Logger
	}
	m := &producerConnManager{
		cfg:    cfg,
		pool:   make(chan *confirmChannel, cfg.ProducerChannelPoolSize),
		logger: l,
	}
	return m
}

// Start: 建立连接并刷新 pool，启动 reconnect watcher
func (m *producerConnManager) Start() error {
	if atomic.LoadInt32(&m.closed) == 1 {
		return fmt.Errorf("producerConnManager closed")
	}
	// already connected?
	if c := m.loadConn(); c != nil && !c.IsClosed() {
		return nil
	}
	conn, err := amqp.Dial(m.cfg.URL)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	// build pool under lock
	m.poolMu.Lock()
	if err := m.refreshPool(conn); err != nil {
		m.poolMu.Unlock()
		_ = conn.Close()
		return err
	}
	old := m.loadConn()
	m.storeConn(conn)
	m.poolMu.Unlock()

	// delay close old conn if exists
	if old != nil && !old.IsClosed() {
		go func(c *amqp.Connection) {
			time.Sleep(500 * time.Millisecond)
			_ = c.Close()
		}(old)
	}

	// start reconnect watcher
	m.wg.Add(1)
	go m.reconnectWatcher()

	m.logger.Info("producerConnManager started", zap.String("url", m.cfg.URL))
	return nil
}

// refreshPool: 创建新的 confirmChannel 列表并填充 pool（在 poolMu 持有时调用）
func (m *producerConnManager) refreshPool(conn *amqp.Connection) error {
	// drain existing pool (close channels)
	for {
		select {
		case cc := <-m.pool:
			cc.close()
		default:
			goto drained
		}
	}
drained:
	created := make([]*confirmChannel, 0, m.cfg.ProducerChannelPoolSize)
	for i := 0; i < m.cfg.ProducerChannelPoolSize; i++ {
		cc, err := newConfirmChannel(conn, m.logger)
		if err != nil {
			// close already created
			for _, c := range created {
				c.close()
			}
			return err
		}
		created = append(created, cc)
	}
	for _, c := range created {
		m.pool <- c
	}
	return nil
}

// GetProducerChannel: 从池中获取一个可用 channel（跳过 closed/broken）
// 注意：此方法只负责取出 channel；publish/inflight 管理由 Publish 方法完成，但是也提供给外部以兼容场景。
func (m *producerConnManager) GetProducerChannel(ctx context.Context) (*confirmChannel, error) {
	if atomic.LoadInt32(&m.closed) == 1 {
		return nil, xmq.ErrClosed
	}
	if ctx == nil {
		ctx = context.Background()
	}
	timer := time.NewTimer(m.cfg.PublishGetTimeout)
	defer timer.Stop()
	for {
		select {
		case cc := <-m.pool:
			if atomic.LoadInt32(&cc.closed) == 0 && atomic.LoadInt32(&cc.broken) == 0 {
				return cc, nil
			}
			// invalid -> close and continue
			cc.close()
			continue
		case <-timer.C:
			return nil, xmq.ErrGetChannelTimeout
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// PutProducerChannel 放回池。为避免高并发下瞬时池满导致频繁 close/recreate，这里采用等待重试的策略：
func (m *producerConnManager) PutProducerChannel(cc *confirmChannel) {
	if cc == nil {
		return
	}
	// 1. 已关闭 / 已坏：直接 close
	if atomic.LoadInt32(&m.closed) == 1 || atomic.LoadInt32(&cc.closed) == 1 || atomic.LoadInt32(&cc.broken) == 1 {
		cc.close()
		return
	}

	// 2. 非阻塞试一次
	select {
	case m.pool <- cc:
		return
	default:
	}

	// 3. 阻塞等待最多 PutBackTimeout（默认 500 ms ~ 1 s）
	deadline := time.Now().Add(m.cfg.PutBackTimeout)
	for time.Now().Before(deadline) {
		select {
		case m.pool <- cc:
			return
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}

	// 4. 超时：直接丢弃（不 close），让 GC 回收
	if m.logger != nil {
		m.logger.Warn("PutProducerChannel: pool full, discard healthy channel to avoid close-storm",
			zap.Duration("timeout", m.cfg.PutBackTimeout))
	}
	// 不调用 cc.close() —— 通道还活着，只是不归还池
}

// Publish: 对外封装的发布方法（推荐 Producer 使用此方法，而不是直接拿 channel publish）
// 该方法负责：获取 channel、管理 inflight 计数、调用 publishWithConfirm、放回 channel。
// 如果遇到 seq==0 或 channel broken，会重试在 timeout/ctx 限制下重新拿 channel 并重发（尝试次数有限由 ctx/外部控制）。
func (m *producerConnManager) Publish(ctx context.Context, exchange, routingKey string, pub amqp.Publishing) error {
	if atomic.LoadInt32(&m.closed) == 1 {
		return xmq.ErrClosed
	}
	// try-get channel
	cc, err := m.GetProducerChannel(ctx)
	if err != nil {
		return err
	}
	// ensure channel returned or closed
	defer func() {
		// if channel is broken/closed, PutProducerChannel will close it
		m.PutProducerChannel(cc)
	}()

	// increment inflight BEFORE publish so Close can wait正确
	atomic.AddInt64(&m.inflight, 1)
	defer atomic.AddInt64(&m.inflight, -1)

	// call publishWithConfirm (it will return errors for seq==0 / nack / timeout)
	err = cc.publishWithConfirm(ctx, exchange, routingKey, pub, m.cfg.ConfirmTimeout, m.cfg.ConsecutiveTimeoutThreshold)
	return err
}

// reconnectWatcher: deterministic exponential backoff, reconnect and refresh pool
func (m *producerConnManager) reconnectWatcher() {
	defer m.wg.Done()
	backoff := m.cfg.ReconnectInitialDelay
	if backoff <= 0 {
		backoff = 500 * time.Millisecond
	}
	if m.cfg.ReconnectMaxDelay <= 0 {
		m.cfg.ReconnectMaxDelay = 30 * time.Second
	}
	for {
		if atomic.LoadInt32(&m.closed) == 1 {
			return
		}
		conn := m.loadConn()
		// if no conn or closed -> try reconnect
		if conn == nil || conn.IsClosed() {
			newConn, err := amqp.Dial(m.cfg.URL)
			if err != nil {
				if m.logger != nil {
					m.logger.Warn("producer reconnect dial failed", zap.String("error", err.Error()))
				}
				time.Sleep(backoff)
				backoff *= 2
				if backoff > m.cfg.ReconnectMaxDelay {
					backoff = m.cfg.ReconnectMaxDelay
				}
				continue
			}
			// build pool and swap
			m.poolMu.Lock()
			if err := m.refreshPool(newConn); err != nil {
				m.poolMu.Unlock()
				_ = newConn.Close()
				time.Sleep(backoff)
				backoff *= 2
				if backoff > m.cfg.ReconnectMaxDelay {
					backoff = m.cfg.ReconnectMaxDelay
				}
				continue
			}
			old := m.loadConn()
			m.storeConn(newConn)
			m.poolMu.Unlock()
			if old != nil && !old.IsClosed() {
				go func(c *amqp.Connection) {
					time.Sleep(500 * time.Millisecond)
					_ = c.Close()
				}(old)
			}
			// reset backoff
			backoff = m.cfg.ReconnectInitialDelay
			if m.logger != nil {
				m.logger.Info("producer reconnected")
			}
			continue
		}
		// wait for close notification
		closeCh := conn.NotifyClose(make(chan *amqp.Error, 1))
		err := <-closeCh
		if err != nil && m.logger != nil {
			m.logger.Warn("producer connection closed", zap.String("error", err.Error()))
		}
		// nil the conn and loop to reconnect
		m.storeConn(nil)
	}
}

// Close: 优雅关闭，先设置 closed flag，关闭 pool 中所有 channel，等待 reconnect goroutine 退出
// 会等待 inflight 在一定时间内归零（避免永远阻塞，这里用一个合理 timeout）
func (m *producerConnManager) Close() error {
	if !atomic.CompareAndSwapInt32(&m.closed, 0, 1) {
		return nil
	}
	// drain pool
	m.poolMu.Lock()
	for {
		select {
		case cc := <-m.pool:
			cc.close()
		default:
			goto drained
		}
	}
drained:
	m.poolMu.Unlock()
	// close conn
	if conn := m.loadConn(); conn != nil {
		_ = conn.Close()
		m.storeConn(nil)
	}
	// wait for reconnect watcher to stop
	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()
	// wait for inflight -> 0 or timeout
	timeout := time.After(10 * time.Second)
	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()
	for {
		if atomic.LoadInt64(&m.inflight) == 0 {
			break
		}
		select {
		case <-timeout:
			// force return, some inflight didn't finish
			if m.logger != nil {
				m.logger.Warn("producer Close timeout waiting inflight", zap.Int64("inflight", m.inflight))
			}
			goto closeConn
		case <-tick.C:
		}
	}
closeConn:
	// already closed conn above; ensure wg done
	select {
	case <-done:
	case <-time.After(2 * time.Second):
	}
	return nil
}

func (m *producerConnManager) loadConn() *amqp.Connection {
	v := m.conn.Load()
	if v == nil {
		return nil
	}
	return v.(*amqp.Connection)
}
func (m *producerConnManager) storeConn(c *amqp.Connection) { m.conn.Store(c) }

////////////////////////////////////////////////////////////////////////////////
// consumerConnManager
//
//  - 管理 consumer 专用连接（worker 每次创建独立 amqp.Channel）
//  - 提供稳定的 reconnect watcher（确定性 backoff）
////////////////////////////////////////////////////////////////////////////////

type consumerConnManager struct {
	cfg    *ConsumerConfig
	conn   atomic.Value // *amqp.Connection
	closed int32
	logger xmq.Logger
}

func newConsumerConnManager(cfg *ConsumerConfig) *consumerConnManager {
	if cfg == nil {
		panic("consumer config nil")
	}
	var l xmq.Logger = &nopLogger{}
	if cfg.Logger != nil {
		l = cfg.Logger
	}
	return &consumerConnManager{
		cfg:    cfg,
		logger: l,
	}
}

func (m *consumerConnManager) Start() error {
	if atomic.LoadInt32(&m.closed) == 1 {
		return fmt.Errorf("consumerConnManager closed")
	}
	if c := m.loadConn(); c != nil && !c.IsClosed() {
		return nil
	}
	conn, err := amqp.Dial(m.cfg.URL)
	if err != nil {
		return fmt.Errorf("consumer dial: %w", err)
	}
	m.storeConn(conn)
	// start reconnect watcher goroutine
	go m.reconnectWatcher()
	m.logger.Info("consumerConnManager started", zap.String("url", m.cfg.URL))
	return nil
}

func (m *consumerConnManager) reconnectWatcher() {
	backoff := m.cfg.ReconnectInitialDelay
	if backoff <= 0 {
		backoff = 500 * time.Millisecond
	}
	if m.cfg.ReconnectMaxDelay <= 0 {
		m.cfg.ReconnectMaxDelay = 30 * time.Second
	}
	for {
		if atomic.LoadInt32(&m.closed) == 1 {
			return
		}
		conn := m.loadConn()
		if conn == nil || conn.IsClosed() {
			newConn, err := amqp.Dial(m.cfg.URL)
			if err != nil {
				if m.logger != nil {
					m.logger.Warn("consumer reconnect dial failed", zap.String("error", err.Error()))
				}
				time.Sleep(backoff)
				backoff *= 2
				if backoff > m.cfg.ReconnectMaxDelay {
					backoff = m.cfg.ReconnectMaxDelay
				}
				continue
			}
			m.storeConn(newConn)
			backoff = m.cfg.ReconnectInitialDelay
			if m.logger != nil {
				m.logger.Info("consumer reconnected")
			}
			continue
		}
		// wait for close
		closeCh := conn.NotifyClose(make(chan *amqp.Error, 1))
		err := <-closeCh
		if err != nil && m.logger != nil {
			m.logger.Warn("consumer connection closed", zap.String("error", err.Error()))
		}
		m.storeConn(nil)
	}
}

// GetConsumerChannel: 返回一个新的 amqp.Channel（调用者负责 close）
func (m *consumerConnManager) GetConsumerChannel() (*amqp.Channel, error) {
	conn := m.loadConn()
	if conn == nil || conn.IsClosed() {
		return nil, xmq.ErrNoConnection
	}
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	return ch, nil
}

func (m *consumerConnManager) Close() {
	if conn := m.loadConn(); conn != nil {
		_ = conn.Close()
		m.storeConn(nil)
	}
}

func (m *consumerConnManager) loadConn() *amqp.Connection {
	v := m.conn.Load()
	if v == nil {
		return nil
	}
	return v.(*amqp.Connection)
}
func (m *consumerConnManager) storeConn(c *amqp.Connection) { m.conn.Store(c) }
