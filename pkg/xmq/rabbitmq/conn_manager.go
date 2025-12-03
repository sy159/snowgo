package rabbitmq

import (
	"context"
	"fmt"
	"math/rand"
	"snowgo/pkg/xmq"
	"sync"
	"sync/atomic"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type pendingRec struct {
	ch   chan amqp.Confirmation
	once sync.Once
}

type confirmChannel struct {
	ch        *amqp.Channel
	mu        sync.Mutex
	closed    int32
	broken    int32
	closeOnce sync.Once
	logger    xmq.Logger
	closeCh   chan struct{}
	pending   sync.Map
	nextSeq   uint64
}

type connectionManager struct {
	cfg            *Config
	conn           atomic.Value // *amqp.Connection
	closed         int32
	pool           chan *confirmChannel
	poolMu         sync.Mutex
	logger         xmq.Logger
	stopCh         chan struct{}
	wg             sync.WaitGroup
	watcherStarted int32
	inflight       int64
}

func newConfirmChannel(conn *amqp.Connection, cfg *Config) (*confirmChannel, error) {
	if conn == nil {
		return nil, xmq.ErrNoConnection
	}
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	if err := ch.Confirm(false); err != nil {
		_ = ch.Close()
		return nil, fmt.Errorf("confirm: %w", err)
	}
	cc := &confirmChannel{
		ch:      ch,
		logger:  cfg.Logger,
		closeCh: make(chan struct{}),
	}
	go cc.confirmRouter()

	// notifyClose watcher
	go func() {
		closeC := ch.NotifyClose(make(chan *amqp.Error, 1))
		if e := <-closeC; e != nil {
			atomic.StoreInt32(&cc.broken, 1)
			cc.close()
			if cc.logger != nil {
				cc.logger.Warn("channel closed by broker", "error", e.Error())
			}
		}
	}()
	return cc, nil
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
					atomic.StoreInt32(&cc.broken, 1)
					cc.pending.Delete(c.DeliveryTag)
					cc.close()
				}
			} else if cc.logger != nil {
				cc.logger.Warn("confirmRouter: no pending rec for tag", "tag", c.DeliveryTag)
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

func (cc *confirmChannel) publishWithConfirm(ctx context.Context, exchange, routingKey string, pub amqp.Publishing, timeout time.Duration) error {
	if atomic.LoadInt32(&cc.closed) == 1 || atomic.LoadInt32(&cc.broken) == 1 {
		return fmt.Errorf("channel closed or broken")
	}

	cc.mu.Lock()
	seq := cc.nextSeq + 1
	if seq == 0 {
		cc.mu.Unlock()
		return fmt.Errorf("seq overflow")
	}
	cc.nextSeq = seq
	rec := &pendingRec{ch: make(chan amqp.Confirmation, 1)}
	cc.pending.Store(seq, rec)

	if err := cc.ch.PublishWithContext(ctx, exchange, routingKey, false, false, pub); err != nil {
		cc.pending.Delete(seq)
		atomic.StoreInt32(&cc.broken, 1)
		cc.mu.Unlock()
		cc.close()
		return fmt.Errorf("publish: %w", err)
	}
	cc.mu.Unlock()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case c, ok := <-rec.ch:
		if !ok {
			return fmt.Errorf("confirm channel closed")
		}
		if !c.Ack {
			atomic.StoreInt32(&cc.broken, 1)
			cc.close()
			return xmq.ErrPublishNack
		}
		return nil
	case <-timer.C:
		if v, ok := cc.pending.Load(seq); ok {
			r := v.(*pendingRec)
			r.once.Do(func() { close(r.ch) })
			cc.pending.Delete(seq)
		}
		atomic.StoreInt32(&cc.broken, 1)
		cc.close()
		return xmq.ErrPublishTimeout
	case <-ctx.Done():
		if v, ok := cc.pending.Load(seq); ok {
			r := v.(*pendingRec)
			r.once.Do(func() { close(r.ch) })
			cc.pending.Delete(seq)
		}
		return ctx.Err()
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

func newConnectionManager(cfg *Config) *connectionManager {
	return &connectionManager{
		cfg:    cfg,
		pool:   make(chan *confirmChannel, cfg.ProducerChannelPoolSize),
		logger: cfg.Logger,
		stopCh: make(chan struct{}),
	}
}

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
	m.log("tcp连接建立", "event.outcome", "success")
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
			m.log("tcp连接断开", "error", err.Error())
		} else {
			m.log("tcp连接关闭(nil)")
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
			m.log("重连成功", "event.outcome", "success")
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

func (m *connectionManager) refreshPool(conn *amqp.Connection) error {
	m.drainPoolLocked()
	var created []*confirmChannel
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
			time.Sleep(time.Millisecond)
			return
		}
	}
}

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

// 公开给消费者使用
func (m *connectionManager) GetConsumerChannel(queue, exchange, routingKey string, qos int) (*amqp.Channel, error) {
	conn := m.loadConn()
	if conn == nil || conn.IsClosed() {
		return nil, xmq.ErrNoConnection
	}
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	if err := ch.Qos(qos, 0, false); err != nil {
		_ = ch.Close()
		return nil, err
	}
	if _, err := ch.QueueDeclare(queue, true, false, false, false, nil); err != nil {
		_ = ch.Close()
		return nil, err
	}
	if exchange != "" {
		if err := ch.QueueBind(queue, routingKey, exchange, false, nil); err != nil {
			_ = ch.Close()
			return nil, err
		}
	}
	return ch, nil
}

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
			if atomic.LoadInt64(&m.inflight) == 0 {
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
		m.cfg.Logger.Info(msg, append([]interface{}{"xmq.role", "producer"}, kvs...)...)
	}
}
func (m *connectionManager) Close(ctx context.Context) error {
	return m.close(ctx)
}
