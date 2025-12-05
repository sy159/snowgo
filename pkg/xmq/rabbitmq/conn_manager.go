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

// newConfirmChannel 创建并启用 confirm 模式（会启动 confirmRouter），用于生产使用
func newConfirmChannel(ctx context.Context, conn *amqp.Connection, logger xmq.Logger) (*confirmChannel, error) {
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
	go cc.confirmRouter(ctx)
	// watch NotifyClose to mark broken
	go func() {
		closeC := ch.NotifyClose(make(chan *amqp.Error, 1))
		if e := <-closeC; e != nil {
			atomic.StoreInt32(&cc.broken, 1)
			cc.close(ctx)
			cc.logger.Warn("confirmChannel: channel closed by broker", zap.String("error", e.Error()))
		}
	}()
	return cc, nil
}

// confirmRouter: 读取 NotifyPublish 并路由给 pendingRec
// 非阻塞写：如果写会阻塞（意味着 publisher 已经离开/未等待），会 close pending ch 并回收 channel，避免泄露。
// 使用缓冲 1 的 rec.ch；写仍可能阻塞（如果接收者已退出且缓冲已满），因此使用 select non-blocking。
func (cc *confirmChannel) confirmRouter(ctx context.Context) {
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
				cc.logger.Warn("confirmRouter: no pending for deliveryTag", zap.Uint64("tag", c.DeliveryTag))
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
				cc.close(ctx)
				cc.logger.Warn("confirmRouter: pending write blocked, marking channel broken", zap.Uint64("tag", c.DeliveryTag))
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
		cc.close(ctx)
		return fmt.Errorf("confirmChannel invalid next seq 0")
	}
	rec := &pendingRec{ch: make(chan amqp.Confirmation, 1)}
	cc.pending.Store(seq, rec)

	// 加锁定执行发布操作，以避免 seq 和 publish 之间的竞争
	err := cc.ch.PublishWithContext(ctx, exchange, routingKey, false, false, pub)
	cc.mu.Unlock()

	if err != nil {
		// 发布失败，标记为损坏，关闭通道
		cc.pending.Delete(seq)
		atomic.StoreInt32(&cc.broken, 1)
		cc.close(ctx)
		return fmt.Errorf("publish failed: %w", err)
	}

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
			cc.close(ctx)
			return xmq.ErrPublishNack
		}
		// Ack: reset consecutive timeout counter
		atomic.StoreInt32(&cc.consecTO, 0)
		return nil
	case <-timer.C:
		// 确认是否超时：连续计数器加一，可能标记为失败
		newCnt := atomic.AddInt32(&cc.consecTO, 1)
		cc.pending.Delete(seq)
		if consecThreshold > 0 && newCnt >= consecThreshold {
			atomic.StoreInt32(&cc.broken, 1)
			cc.close(ctx)
			cc.logger.Warn("confirmChannel consecutive confirm timeout reached", zap.Int32("consecutive", newCnt))
		} else {
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

func (cc *confirmChannel) close(ctx context.Context) {
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

type producerConnManager struct {
	cfg *ProducerConnConfig

	conn atomic.Value // *amqp.Connection

	pool   chan *confirmChannel
	poolMu sync.Mutex // protect refreshPool

	inflight int64 // 当前正在等待 confirm 的消息数量

	closed int32
	logger xmq.Logger

	wg sync.WaitGroup
}

// newProducerConnManager 生产连接管理
func newProducerConnManager(ctx context.Context, cfg *ProducerConnConfig) *producerConnManager {
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

// Start  建立连接并刷新 pool，启动 reconnect watcher
func (m *producerConnManager) Start(ctx context.Context) error {
	if atomic.LoadInt32(&m.closed) == 1 {
		return xmq.ErrProducerConnManagerClosed
	}
	// already connected
	if c := m.loadConn(ctx); c != nil && !c.IsClosed() {
		return nil
	}
	// 创建连接
	conn, err := amqp.Dial(m.cfg.URL)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	// 创建 confirm-channel 池
	m.poolMu.Lock()
	if err := m.refreshPool(ctx, conn); err != nil {
		m.poolMu.Unlock()
		_ = conn.Close()
		return err
	}
	old := m.loadConn(ctx)
	m.storeConn(ctx, conn)
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
	go m.reconnectWatcher(ctx)

	m.logger.Info("producerConnManager started", zap.String("url", m.cfg.URL))
	return nil
}

// refreshPool: 创建新的 confirmChannel 列表并填充 pool（在 poolMu 持有时调用）
func (m *producerConnManager) refreshPool(ctx context.Context, conn *amqp.Connection) error {
	// drain existing pool (close channels)
	for {
		select {
		case cc := <-m.pool:
			cc.close(ctx)
		default:
			goto drained
		}
	}
drained:
	created := make([]*confirmChannel, 0, m.cfg.ProducerChannelPoolSize)
	for i := 0; i < m.cfg.ProducerChannelPoolSize; i++ {
		cc, err := newConfirmChannel(ctx, conn, m.logger)
		if err != nil {
			// close already created
			for _, c := range created {
				c.close(ctx)
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

// GetProducerChannel  从池中获取一个可用 channel（跳过 closed/broken）
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
			cc.close(ctx)
			continue
		case <-timer.C:
			return nil, xmq.ErrGetChannelTimeout
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// PutProducerChannel 放回池。为避免高并发下瞬时池满导致频繁
func (m *producerConnManager) PutProducerChannel(ctx context.Context, cc *confirmChannel) {
	if cc == nil {
		return
	}

	// 已关闭 / 已坏：直接 close
	if atomic.LoadInt32(&m.closed) == 1 || atomic.LoadInt32(&cc.closed) == 1 || atomic.LoadInt32(&cc.broken) == 1 {
		cc.close(ctx)
		return
	}

	// 尝试1次非阻塞放入
	select {
	case m.pool <- cc:
		return
	default:
		m.logger.Warn("PutProducerChannel: pool full, discard channel",
			zap.Int("pool_size", cap(m.pool)),
			zap.Int("pool_len", len(m.pool)))
	}
}

// Publish 对外封装的发布方法（推荐 Producer 使用此方法，而不是直接拿 channel publish）
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
		// 使用完的通道返回池子，如果通道有问题，会直接释放
		m.PutProducerChannel(ctx, cc)
	}()

	// 在publish以前 incr inflight保证 close正确
	atomic.AddInt64(&m.inflight, 1)
	defer atomic.AddInt64(&m.inflight, -1)

	// call publishWithConfirm (it will return errors for seq==0 / nack / timeout)
	err = cc.publishWithConfirm(ctx, exchange, routingKey, pub, m.cfg.ConfirmTimeout, m.cfg.ConsecutiveTimeoutThreshold)
	return err
}

// reconnectWatcher: deterministic exponential backoff, reconnect and refresh pool
func (m *producerConnManager) reconnectWatcher(ctx context.Context) {
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
		conn := m.loadConn(ctx)
		// if no conn or closed -> try reconnect
		if conn == nil || conn.IsClosed() {
			newConn, err := amqp.Dial(m.cfg.URL)
			if err != nil {
				m.logger.Warn("producer reconnect dial failed", zap.String("error", err.Error()))
				time.Sleep(backoff)
				backoff *= 2
				if backoff > m.cfg.ReconnectMaxDelay {
					backoff = m.cfg.ReconnectMaxDelay
				}
				continue
			}
			// build pool and swap
			m.poolMu.Lock()
			if err := m.refreshPool(ctx, newConn); err != nil {
				m.poolMu.Unlock()
				_ = newConn.Close()
				time.Sleep(backoff)
				backoff *= 2
				if backoff > m.cfg.ReconnectMaxDelay {
					backoff = m.cfg.ReconnectMaxDelay
				}
				continue
			}
			old := m.loadConn(ctx)
			m.storeConn(ctx, newConn)
			m.poolMu.Unlock()
			if old != nil && !old.IsClosed() {
				go func(c *amqp.Connection) {
					time.Sleep(500 * time.Millisecond)
					_ = c.Close()
				}(old)
			}
			// reset backoff
			backoff = m.cfg.ReconnectInitialDelay
			m.logger.Info("producer reconnected")
			continue
		}
		// wait for close notification
		closeCh := conn.NotifyClose(make(chan *amqp.Error, 1))
		err := <-closeCh
		if err != nil {
			m.logger.Warn("producer connection closed", zap.String("error", err.Error()))
		}
		// nil the conn and loop to reconnect
		m.storeConn(ctx, nil)
	}
}

// Close 优雅关闭，先设置 closed flag，关闭 pool 中所有 channel，等待 reconnect goroutine 退出
// 会等待 inflight 在一定时间内归零（避免永远阻塞，这里用一个合理 timeout）
func (m *producerConnManager) Close(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&m.closed, 0, 1) {
		return nil
	}
	// drain pool
	m.poolMu.Lock()
	for {
		select {
		case cc := <-m.pool:
			cc.close(ctx)
		default:
			goto drained
		}
	}
drained:
	m.poolMu.Unlock()
	// close conn
	if conn := m.loadConn(ctx); conn != nil {
		_ = conn.Close()
		m.storeConn(ctx, nil)
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
			m.logger.Warn("producer Close timeout waiting inflight", zap.Int64("inflight", m.inflight))
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

// loadConn 获取mq client
func (m *producerConnManager) loadConn(ctx context.Context) *amqp.Connection {
	v := m.conn.Load()
	if v == nil {
		return nil
	}
	return v.(*amqp.Connection)
}

// storeConn 设置mq client
func (m *producerConnManager) storeConn(ctx context.Context, c *amqp.Connection) { m.conn.Store(c) }

type consumerConnManager struct {
	cfg    *ConsumerConnConfig
	conn   atomic.Value // *amqp.Connection
	closed int32
	logger xmq.Logger
}

func newConsumerConnManager(ctx context.Context, cfg *ConsumerConnConfig) *consumerConnManager {
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

func (m *consumerConnManager) Start(ctx context.Context) error {
	if atomic.LoadInt32(&m.closed) == 1 {
		return fmt.Errorf("consumerConnManager closed")
	}
	if c := m.loadConn(ctx); c != nil && !c.IsClosed() {
		return nil
	}
	conn, err := amqp.Dial(m.cfg.URL)
	if err != nil {
		return fmt.Errorf("consumer dial: %w", err)
	}
	m.storeConn(ctx, conn)
	// start reconnect watcher goroutine
	go m.reconnectWatcher(ctx)
	m.logger.Info("consumerConnManager started", zap.String("url", m.cfg.URL))
	return nil
}

func (m *consumerConnManager) reconnectWatcher(ctx context.Context) {
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
		conn := m.loadConn(ctx)
		if conn == nil || conn.IsClosed() {
			newConn, err := amqp.Dial(m.cfg.URL)
			if err != nil {
				m.logger.Warn("consumer reconnect dial failed", zap.String("error", err.Error()))
				time.Sleep(backoff)
				backoff *= 2
				if backoff > m.cfg.ReconnectMaxDelay {
					backoff = m.cfg.ReconnectMaxDelay
				}
				continue
			}
			m.storeConn(ctx, newConn)
			backoff = m.cfg.ReconnectInitialDelay
			m.logger.Info("consumer reconnected")
			continue
		}
		// wait for close
		closeCh := conn.NotifyClose(make(chan *amqp.Error, 1))
		err := <-closeCh
		if err != nil {
			m.logger.Warn("consumer connection closed", zap.String("error", err.Error()))
		}
		m.storeConn(ctx, nil)
	}
}

// GetConsumerChannel 返回一个新的 amqp.Channel（调用者负责 close）
func (m *consumerConnManager) GetConsumerChannel(ctx context.Context) (*amqp.Channel, error) {
	conn := m.loadConn(ctx)
	if conn == nil || conn.IsClosed() {
		return nil, xmq.ErrNoConnection
	}
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	return ch, nil
}

func (m *consumerConnManager) Close(ctx context.Context) {
	if conn := m.loadConn(ctx); conn != nil {
		_ = conn.Close()
		m.storeConn(ctx, nil)
	}
}

func (m *consumerConnManager) loadConn(ctx context.Context) *amqp.Connection {
	v := m.conn.Load()
	if v == nil {
		return nil
	}
	return v.(*amqp.Connection)
}

func (m *consumerConnManager) storeConn(ctx context.Context, c *amqp.Connection) { m.conn.Store(c) }
