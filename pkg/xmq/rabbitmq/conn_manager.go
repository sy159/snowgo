package rabbitmq

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"snowgo/pkg/xmq"
)

/* ---------- 内部数据结构 ---------- */

// pendingRec 单条 publish 的 confirm 等待记录
type pendingRec struct {
	ch   chan amqp.Confirmation
	once sync.Once
}

// confirmChannel 封装一个 amqp.Channel，提供「同步 confirm」能力
type confirmChannel struct {
	ch        *amqp.Channel
	mu        sync.Mutex
	closed    int32
	broken    int32
	closeOnce sync.Once
	logger    xmq.Logger
	closeCh   chan struct{}
	pending   sync.Map
	// 连续超时次数
	consecutiveTimeout int32
}

// connectionManager 管理单条 TCP 连接与 Channel 池（仅 Producer 用）
type connectionManager struct {
	cfg    *Config
	conn   atomic.Value // *amqp.Connection
	closed int32
	pool   chan *confirmChannel
	poolMu sync.Mutex
	logger xmq.Logger
	stopCh chan struct{}
	wg     sync.WaitGroup

	// 优雅关闭
	inflight int64

	// 保证 reconnectWatcher 只启动一次
	watcherStarted int32
	// 连续超时阈值
	consecutiveTimeoutThreshold int
}

/* ---------- 新建 Channel（confirm 模式） ---------- */

func newConfirmChannel(conn *amqp.Connection, cfg *Config) (*confirmChannel, error) {
	if conn == nil {
		return nil, xmq.ErrNoConnection
	}
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("create channel: %w", err)
	}
	if err = ch.Confirm(false); err != nil {
		_ = ch.Close()
		return nil, fmt.Errorf("confirm: %w", err)
	}
	cc := &confirmChannel{
		ch:      ch,
		logger:  cfg.Logger,
		closeCh: make(chan struct{}),
	}
	go cc.confirmRouter()
	go cc.watchBrokerClose()
	return cc, nil
}

func (cc *confirmChannel) watchBrokerClose() {
	closeC := cc.ch.NotifyClose(make(chan *amqp.Error, 1))
	if e := <-closeC; e != nil {
		atomic.StoreInt32(&cc.broken, 1)
		cc.close()
		cc.logWarn("channel closed by broker", "error", e.Error())
	}
}

func (cc *confirmChannel) confirmRouter() {
	confSrc := cc.ch.NotifyPublish(make(chan amqp.Confirmation, 1))
	for {
		select {
		case c, ok := <-confSrc:
			if !ok {
				goto exit
			}
			if v, ok := cc.pending.Load(c.DeliveryTag); ok {
				rec := v.(*pendingRec)
				select {
				case rec.ch <- c:
					rec.once.Do(func() { close(rec.ch) })
					cc.pending.Delete(c.DeliveryTag)
				default:
					cc.pending.Delete(c.DeliveryTag)
				}
			} else {
				cc.logWarn("confirmRouter: no pending rec for tag", "tag", c.DeliveryTag)
			}
		case <-cc.closeCh:
			goto exit
		}
	}
exit:
	atomic.StoreInt32(&cc.closed, 1)
	cc.pending.Range(func(k, v interface{}) bool {
		if rec, ok := v.(*pendingRec); ok {
			rec.once.Do(func() { close(rec.ch) })
			cc.pending.Delete(k)
		}
		return true
	})
}

/* ---------- 同步 Publish ---------- */

func (cc *confirmChannel) publishWithConfirm(ctx context.Context, exchange, routingKey string, pub amqp.Publishing, timeout time.Duration, mgr *connectionManager) error {
	if atomic.LoadInt32(&cc.closed) == 1 || atomic.LoadInt32(&cc.broken) == 1 {
		return fmt.Errorf("channel closed or broken")
	}
	seq := cc.ch.GetNextPublishSeqNo()
	rec := &pendingRec{ch: make(chan amqp.Confirmation)}
	cc.pending.Store(seq, rec)

	if err := cc.ch.PublishWithContext(ctx, exchange, routingKey, false, false, pub); err != nil {
		cc.pending.Delete(seq)
		atomic.StoreInt32(&cc.broken, 1)
		return fmt.Errorf("publish: %w", err)
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case c, ok := <-rec.ch:
		if !ok {
			return fmt.Errorf("confirm channel closed")
		}
		if !c.Ack {
			atomic.StoreInt32(&cc.broken, 1)
			return xmq.ErrPublishNack
		}
		atomic.StoreInt32(&cc.consecutiveTimeout, 0)
		return nil
	case <-timer.C:
		cc.pending.Delete(seq)
		if cc.recordTimeout(mgr) >= mgr.consecutiveTimeoutThreshold {
			atomic.StoreInt32(&cc.broken, 1)
			cc.close()
		}
		return xmq.ErrPublishTimeout
	case <-ctx.Done():
		cc.pending.Delete(seq)
		return ctx.Err()
	}
}

func (cc *confirmChannel) recordTimeout(mgr *connectionManager) int {
	cnt := int(atomic.LoadInt32(&cc.consecutiveTimeout))
	if cnt >= mgr.consecutiveTimeoutThreshold {
		atomic.StoreInt32(&cc.broken, 1)
		cc.close()
	}
	return cnt
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

func (cc *confirmChannel) logWarn(msg string, kvs ...interface{}) {
	if cc.logger != nil {
		cc.logger.Warn(msg, kvs...)
	}
}

/* ---------- connectionManager 构造 ---------- */

func newConnectionManager(cfg *Config) *connectionManager {
	return &connectionManager{
		cfg:                         cfg,
		pool:                        make(chan *confirmChannel, cfg.ProducerChannelPoolSize),
		logger:                      cfg.Logger,
		stopCh:                      make(chan struct{}),
		consecutiveTimeoutThreshold: 3,
	}
}

/* ---------- TCP 连接保活 ---------- */

func (m *connectionManager) start() error {
	m.poolMu.Lock()
	defer m.poolMu.Unlock()
	if c := m.loadConn(); c != nil && !c.IsClosed() {
		return nil
	}
	conn, err := amqp.Dial(m.cfg.URL)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	m.storeConn(conn)
	m.log("tcp connection established", "event.outcome", "success")
	if err := m.refreshPool(conn); err != nil {
		m.drainPoolLocked()
		return err
	}
	if atomic.CompareAndSwapInt32(&m.watcherStarted, 0, 1) {
		m.wg.Add(1)
		go m.reconnectWatcher()
	}
	return nil
}

func (m *connectionManager) reconnectWatcher() {
	defer m.wg.Done()
	for {
		conn := m.loadConn()
		if conn == nil || conn.IsClosed() {
			if err := m.tryReconnect(); err != nil {
				return
			}
			continue
		}
		closeCh := conn.NotifyClose(make(chan *amqp.Error, 1))
		if err := <-closeCh; err != nil {
			m.log("tcp connection lost", "error", err.Error())
		} else {
			m.log("tcp connection closed (nil)")
		}
		m.storeConn(nil)
		m.poolMu.Lock()
		m.drainPoolLocked()
		m.poolMu.Unlock()
		select {
		case <-m.stopCh:
			return
		default:
		}
		if err := m.tryReconnect(); err != nil {
			return
		}
	}
}

func (m *connectionManager) tryReconnect() error {
	backoff := m.cfg.ReconnectInitialDelay
	for {
		if m.isClosed() {
			return fmt.Errorf("stopped")
		}
		if err := m.start(); err == nil {
			m.log("reconnect success", "event.outcome", "success")
			return nil
		}
		backoff = jitter(backoff)
		if backoff > m.cfg.ReconnectMaxDelay {
			backoff = m.cfg.ReconnectMaxDelay
		}
		time.Sleep(backoff)
	}
}

func jitter(d time.Duration) time.Duration {
	return time.Duration(float64(d) * (0.8 + 0.4*rand.Float64()))
}

/* ---------- Channel 池 ---------- */

func (m *connectionManager) refreshPool(conn *amqp.Connection) error {
	m.drainPoolLocked()
	created := make([]*confirmChannel, 0, m.cfg.ProducerChannelPoolSize)
	for i := 0; i < m.cfg.ProducerChannelPoolSize; i++ {
		cc, err := newConfirmChannel(conn, m.cfg)
		if err != nil {
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

func (m *connectionManager) drainPoolLocked() {
	for {
		select {
		case cc := <-m.pool:
			cc.close()
		default:
			return
		}
	}
}

/* ---------- Producer API ---------- */

func (m *connectionManager) GetProducerChannel(ctx context.Context) (*confirmChannel, error) {
	if m.isClosed() {
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
			cc.close()
			continue
		case <-timer.C:
			return nil, xmq.ErrGetChannelTimeout
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func (m *connectionManager) PutProducerChannel(cc *confirmChannel) {
	if cc == nil {
		return
	}
	if m.isClosed() || atomic.LoadInt32(&cc.closed) == 1 || atomic.LoadInt32(&cc.broken) == 1 {
		cc.close()
		return
	}
	select {
	case m.pool <- cc:
	default:
		cc.close()
	}
}

/* ---------- Consumer API（仅新建 Channel，无池化） ---------- */

var consumerChannelCount int64 // 进程级计数器

func (m *connectionManager) GetConsumerChannel(queue, exchange, routingKey string, qos int) (*amqp.Channel, error) {
	conn := m.loadConn()
	if conn == nil || conn.IsClosed() {
		return nil, xmq.ErrNoConnection
	}
	if atomic.LoadInt64(&consumerChannelCount) >= int64(m.cfg.MaxChannelsPerConn) {
		return nil, xmq.ErrChannelExhausted
	}
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	atomic.AddInt64(&consumerChannelCount, 1)
	// 关闭时计数减一
	go func() {
		<-ch.NotifyClose(make(chan *amqp.Error, 1))
		atomic.AddInt64(&consumerChannelCount, -1)
	}()

	if err = ch.Qos(qos, 0, false); err != nil {
		_ = ch.Close()
		return nil, err
	}
	if _, err = ch.QueueDeclare(queue, true, false, false, false, nil); err != nil {
		_ = ch.Close()
		return nil, err
	}
	if exchange != "" {
		if err = ch.QueueBind(queue, routingKey, exchange, false, nil); err != nil {
			_ = ch.Close()
			return nil, err
		}
	}
	return ch, nil
}

/* ---------- 优雅关闭 ---------- */

func (m *connectionManager) close(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&m.closed, 0, 1) {
		return nil
	}
	close(m.stopCh)
	m.poolMu.Lock()
	m.drainPoolLocked()
	m.poolMu.Unlock()

	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			if atomic.LoadInt64(&m.inflight) == 0 && atomic.LoadInt32(&m.closed) == 1 {
				goto closeConn
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
closeConn:
	if conn := m.loadConn(); conn != nil {
		_ = conn.Close()
	}
	m.wg.Wait()
	return nil
}

/* ---------- 工具 ---------- */

func (m *connectionManager) loadConn() *amqp.Connection {
	if v := m.conn.Load(); v != nil {
		return v.(*amqp.Connection)
	}
	return nil
}
func (m *connectionManager) storeConn(conn *amqp.Connection) { m.conn.Store(conn) }
func (m *connectionManager) isClosed() bool                  { return atomic.LoadInt32(&m.closed) == 1 }
func (m *connectionManager) log(msg string, kvs ...interface{}) {
	if m.cfg.Logger != nil {
		m.cfg.Logger.Info(msg, append([]interface{}{"xmq.role", string(m.cfg.Role)}, kvs...)...)
	}
}
func (m *connectionManager) Close(ctx context.Context) error { return m.close(ctx) }
