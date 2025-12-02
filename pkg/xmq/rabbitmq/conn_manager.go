package rabbitmq

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"snowgo/pkg/xmq"
	"sync"
	"sync/atomic"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// confirmChannel 包装了 amqp.Channel 并监听 confirm
type confirmChannel struct {
	ch        *amqp.Channel
	notifyCh  chan amqp.Confirmation
	mu        sync.Mutex
	closed    int32
	logger    xmq.Logger
	closeOnce sync.Once
}

// 创建一个 confirmChannel 并启动 confirmRouter
func newConfirmChannel(conn *amqp.Connection, cfg *Config) (*confirmChannel, error) {
	if conn == nil {
		return nil, xmq.ErrNoConnection
	}
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	if err := ch.Confirm(false); err != nil {
		ch.Close()
		return nil, fmt.Errorf("confirm: %w", err)
	}
	cc := &confirmChannel{
		ch:       ch,
		notifyCh: make(chan amqp.Confirmation, cfg.NotifyBufSize),
		logger:   cfg.Logger,
	}
	go cc.confirmRouter()
	return cc, nil
}

// confirmRouter 从底层 channel 读取 confirms 并放入 notifyCh
func (cc *confirmChannel) confirmRouter() {
	confSrc := cc.ch.NotifyPublish(make(chan amqp.Confirmation, cap(cc.notifyCh)))
	for c := range confSrc {
		select {
		case cc.notifyCh <- c:
		default:
			if cc.logger != nil {
				cc.logger.Warn("confirm router drop", zap.Any("tag", c.DeliveryTag))
			}
		}
	}
	atomic.StoreInt32(&cc.closed, 1)
	close(cc.notifyCh)
}

// publishWithConfirm 发送消息并等待 broker ack/nack
func (cc *confirmChannel) publishWithConfirm(ctx context.Context, exchange, routingKey string, pub amqp.Publishing, timeout time.Duration) error {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	if atomic.LoadInt32(&cc.closed) == 1 {
		return fmt.Errorf("channel closed")
	}
	if err := cc.ch.PublishWithContext(ctx, exchange, routingKey, false, false, pub); err != nil {
		return fmt.Errorf("publish: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	select {
	case c, ok := <-cc.notifyCh:
		if !ok {
			return fmt.Errorf("notify closed")
		}
		if !c.Ack {
			return xmq.ErrPublishNack
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// 关闭 channel
func (cc *confirmChannel) close() {
	cc.closeOnce.Do(func() {
		atomic.StoreInt32(&cc.closed, 1)
		if cc.ch != nil {
			_ = cc.ch.Close()
		}
	})
}

// ----------------- connectionManager -----------------

type connectionManager struct {
	cfg            *Config
	conn           atomic.Value // *amqp.Connection
	closed         int32
	pool           chan *confirmChannel
	poolMu         sync.Mutex // 保护 pool 刷新
	logger         xmq.Logger
	stopCh         chan struct{}
	wg             sync.WaitGroup
	watcherStarted int32
	inflight       int64 // 真实 inflight 消息计数
}

func newConnectionManager(cfg *Config) *connectionManager {
	return &connectionManager{
		cfg:    cfg,
		pool:   make(chan *confirmChannel, cfg.ProducerChannelPoolSize),
		logger: cfg.Logger,
		stopCh: make(chan struct{}),
	}
}

// start 建立连接并填充 channel pool
func (m *connectionManager) start() error {
	m.poolMu.Lock()
	defer m.poolMu.Unlock()

	if m.loadConn() != nil && !m.loadConn().IsClosed() {
		return nil
	}

	conn, err := amqp.Dial(m.cfg.URL)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	m.storeConn(conn)
	m.logger.Info("connected", zap.String("url", m.cfg.URL))

	if err := m.refreshPool(conn); err != nil {
		m.drainPoolLocked()
		return err
	}

	// 启动单 watcher
	if atomic.CompareAndSwapInt32(&m.watcherStarted, 0, 1) {
		m.wg.Add(1)
		go m.reconnectWatcher()
	}

	return nil
}

// 刷新池：清空旧池 + 创建新池
func (m *connectionManager) refreshPool(conn *amqp.Connection) error {
	// 已加锁保护
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

// drainPoolLocked 必须在 poolMu 锁保护下调用
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

func (m *connectionManager) storeConn(conn *amqp.Connection) { m.conn.Store(conn) }
func (m *connectionManager) loadConn() *amqp.Connection {
	if v := m.conn.Load(); v != nil {
		return v.(*amqp.Connection)
	}
	return nil
}

// reconnectWatcher 单 goroutine watcher
func (m *connectionManager) reconnectWatcher() {
	defer m.wg.Done()
	for {
		conn := m.loadConn()
		if conn == nil {
			if err := m.tryReconnect(); err != nil {
				return
			}
			continue
		}
		closeCh := conn.NotifyClose(make(chan *amqp.Error, 1))
		if err := <-closeCh; err != nil {
			m.logger.Warn("connection closed", zap.String("err", err.Error()))
		} else {
			m.logger.Info("connection closed (nil err)")
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

// tryReconnect 指数回退重连
func (m *connectionManager) tryReconnect() error {
	backoff := m.cfg.ReconnectInitialDelay
	for {
		if m.isClosed() {
			return fmt.Errorf("stopped")
		}
		if err := m.start(); err == nil {
			m.logger.Info("reconnect success")
			return nil
		} else {
			m.logger.Warn("reconnect attempt failed", zap.Error(err), zap.Duration("retry_in", backoff))
		}

		select {
		case <-time.After(backoff):
		case <-m.stopCh:
			return fmt.Errorf("stopped")
		}

		backoff *= 2
		if backoff > m.cfg.ReconnectMaxDelay {
			backoff = m.cfg.ReconnectMaxDelay
		}
	}
}

// getProducerChannel 获取 channel
func (m *connectionManager) getProducerChannel(ctx context.Context) (*confirmChannel, error) {
	if m.isClosed() {
		return nil, xmq.ErrClosed
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Deadline(); !ok && m.cfg.PublishGetTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, m.cfg.PublishGetTimeout)
		defer cancel()
	}

	select {
	case cc := <-m.pool:
		if atomic.LoadInt32(&cc.closed) == 0 {
			return cc, nil
		}
		return m.fallbackChannel()
	case <-ctx.Done():
		return nil, xmq.ErrGetChannelTimeout
	}
}

// fallbackChannel pool 空时新建
func (m *connectionManager) fallbackChannel() (*confirmChannel, error) {
	return newConfirmChannel(m.loadConn(), m.cfg)
}

// putProducerChannel 归还 channel
func (m *connectionManager) putProducerChannel(cc *confirmChannel) {
	if cc == nil || atomic.LoadInt32(&cc.closed) == 1 || m.isClosed() {
		cc.close()
		return
	}
	select {
	case m.pool <- cc:
	default:
		cc.close()
	}
}

// getConsumerChannel 用于声明/获取一个消费者 Channel
func (m *connectionManager) getConsumerChannel(queue, exchange, routingKey string, qos int) (*amqp.Channel, error) {
	conn := m.loadConn()
	if conn == nil {
		return nil, fmt.Errorf("no connection")
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	if err := ch.Qos(qos, 0, false); err != nil {
		ch.Close()
		return nil, err
	}

	// 声明队列
	if _, err := ch.QueueDeclare(
		queue,
		true,  // durable
		false, // autoDelete
		false, // exclusive
		false, // noWait
		nil,   // args
	); err != nil {
		ch.Close()
		return nil, err
	}

	// 如果 exchange 不为空，绑定
	if exchange != "" {
		if err := ch.QueueBind(queue, routingKey, exchange, false, nil); err != nil {
			ch.Close()
			return nil, err
		}
	}

	return ch, nil
}

func (m *connectionManager) isClosed() bool {
	return atomic.LoadInt32(&m.closed) == 1
}

// close 关闭 manager & pool
func (m *connectionManager) close() error {
	if !atomic.CompareAndSwapInt32(&m.closed, 0, 1) {
		return nil
	}
	close(m.stopCh)
	m.poolMu.Lock()
	m.drainPoolLocked()
	m.poolMu.Unlock()
	if conn := m.loadConn(); conn != nil {
		_ = conn.Close()
	}
	m.wg.Wait()
	return nil
}
