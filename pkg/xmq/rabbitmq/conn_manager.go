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

const (
	// 默认单个 confirmChannel 上允许的最大 pending 数（防止无限增长）
	// 如果需要可将其移到 ProducerConnConfig 并通过配置注入。
	defaultMaxPendingPerChannel = 100000
	// 当尾部清理未命中时的有界反向扫描上限（可调）
	defaultTailScanLimit = 1024
)

// pendingRec 保留等待 confirm 的通道
type pendingRec struct {
	ch   chan amqp.Confirmation
	once sync.Once
}

// pendingNode 是 pendingList 的元素
type pendingNode struct {
	tag uint64
	rec *pendingRec
}

// pendingList 并发安全的有序前缀列表（适合 seq 单调增加的场景）
// 设计理由：在 confirm（尤其是 batch confirm）场景下，可以高效弹出前缀。
// 实现假定 publish 的 seq 是单调递增（由 amqp channel 保证），因此 add 操作可以直接 append。
type pendingList struct {
	mu    sync.Mutex
	list  []pendingNode
	count int32 // 原子计数，避免 confirmRouter 频繁加锁获取大小
}

// add 将节点尾部 append（O(1)）
// 前提：seq 单调递增（GetNextPublishSeqNo 保证）
func (l *pendingList) add(tag uint64, rec *pendingRec) {
	l.mu.Lock()
	l.list = append(l.list, pendingNode{tag: tag, rec: rec})
	atomic.AddInt32(&l.count, 1)
	l.mu.Unlock()
}

// decCount: 原子安全地递减 count，保证不小于 0
func (l *pendingList) decCount(n int32) {
	if n <= 0 {
		return
	}
	for {
		old := atomic.LoadInt32(&l.count)
		if old <= 0 {
			// 已经为0或负，尝试把它置为0（防御性）
			if atomic.CompareAndSwapInt32(&l.count, old, 0) {
				return
			}
			continue
		}
		newv := old - n
		if newv < 0 {
			newv = 0
		}
		if atomic.CompareAndSwapInt32(&l.count, old, newv) {
			return
		}
		// CAS 失败 -> retry
	}
}

// ackLE 唤醒并移除所有 tag <= given tag，返回被唤醒的数量
// ack 参数表示要发送给等待者的 Ack 值（true = ack，false = nack）
func (l *pendingList) ackLE(tag uint64, ack bool) int {
	l.mu.Lock()
	i := 0
	for ; i < len(l.list) && l.list[i].tag <= tag; i++ {
		// non-blocking send: 如果阻塞则直接 close chan（唤醒等待方）
		select {
		case l.list[i].rec.ch <- amqp.Confirmation{DeliveryTag: l.list[i].tag, Ack: ack}:
			l.list[i].rec.once.Do(func() { close(l.list[i].rec.ch) })
		default:
			l.list[i].rec.once.Do(func() { close(l.list[i].rec.ch) })
		}
	}
	if i > 0 {
		// 清零引用帮助 GC，并重切片
		for j := 0; j < i; j++ {
			l.list[j].rec = nil
		}
		l.list = l.list[i:]
		// #nosec G115 -- 值已在安全范围内，int -> int32 转换可控
		l.decCount(int32(i)) //nolint:gosec
	}
	l.mu.Unlock()
	return i
}

// len 获取当前 pending 数量
func (l *pendingList) len() int {
	return int(atomic.LoadInt32(&l.count))
}

// drainAll 在关闭时唤醒所有 pending（按 nack/false）
func (l *pendingList) drainAll() int {
	// 直接调用 ackLE 用一个极大 tag（会在内部减少 count）
	return l.ackLE(^uint64(0), false)
}

// popTailIfMatch: 如果尾部 tag == wantTag，则弹出尾部并关闭其 rec.ch，返回 true
func (l *pendingList) popTailIfMatch(wantTag uint64) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	n := len(l.list)
	if n == 0 {
		return false
	}
	if l.list[n-1].tag == wantTag {
		last := l.list[n-1]
		last.rec.once.Do(func() { close(last.rec.ch) })
		l.list = l.list[:n-1]
		l.decCount(1)
		return true
	}
	return false
}

// removeByTag 在任意位置查找 wantTag 并移除（线性扫描，作为兜底）
func (l *pendingList) removeByTag(wantTag uint64) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	n := len(l.list)
	if n == 0 {
		return false
	}
	for i := 0; i < n; i++ {
		if l.list[i].tag == wantTag {
			found := l.list[i]
			// close via once.Do to avoid double close
			if found.rec != nil {
				found.rec.once.Do(func() { close(found.rec.ch) })
			}
			if i == n-1 {
				l.list = l.list[:n-1]
			} else {
				copy(l.list[i:], l.list[i+1:])
				l.list[len(l.list)-1] = pendingNode{}
				l.list = l.list[:len(l.list)-1]
			}
			l.decCount(1)
			return true
		}
	}
	return false
}

// removeTailNearby: 从尾向前最多 scanLimit 个元素查找 wantTag，找到则移除该元素并返回 true
// 注意：移除中间元素需要移动 slice（cost O(k)），但我们限制了 scanLimit 来控制最坏情况。
// 只有在极罕见场景（回绕/异常）下才会触发此函数。
func (l *pendingList) removeTailNearby(wantTag uint64, scanLimit int) bool {
	if scanLimit <= 0 {
		return false
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	n := len(l.list)
	if n == 0 {
		return false
	}
	scanned := 0
	for i := n - 1; i >= 0 && scanned < scanLimit; i-- {
		if l.list[i].tag == wantTag {
			// 如果是尾部，直接 pop
			if i == n-1 {
				last := l.list[n-1]
				last.rec.once.Do(func() { close(last.rec.ch) })
				l.list = l.list[:n-1]
				l.decCount(1)
				return true
			}
			// 中间元素：移位
			found := l.list[i]
			found.rec.once.Do(func() { close(found.rec.ch) })
			copy(l.list[i:], l.list[i+1:])
			l.list[len(l.list)-1] = pendingNode{}
			l.list = l.list[:len(l.list)-1]
			l.decCount(1)
			return true
		}
		scanned++
	}
	return false
}

type confirmChannel struct {
	ch        *amqp.Channel
	logger    xmq.Logger
	closeCh   chan struct{}
	closeOnce sync.Once

	// mu 用于保护 GetNextPublishSeqNo + PublishWithContext 的原子性（保持 seq 与 publish 原子）
	mu sync.Mutex

	pending *pendingList // 使用有序 pending 列表（尾部 append，前缀弹出）

	closed   int32         // atomic flag
	broken   int32         // atomic flag: channel 被标记为 broken（需要回收）
	consecTO int32         // 连续 confirm 超时计数（用于触发回收）
	lastSeq  uint64        // 仅在 cc.mu 持有时读写
	lastSeqA atomic.Uint64 // 原子副本，供失败路径无锁读取
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
		pending: &pendingList{},
	}
	// 启动 confirmRouter
	go cc.confirmRouter(ctx)
	// 监控 NotifyClose，保证 channel 被 broker 关闭时能正确回收并写日志
	go func() {
		closeC := ch.NotifyClose(make(chan *amqp.Error, 1))
		select {
		case e, ok := <-closeC:
			if !ok {
				// notify channel closed - ensure cleanup
				atomic.StoreInt32(&cc.broken, 1)
				cc.close(ctx)
				cc.logger.Warn(
					ctx,
					"confirm channel notifyClose channel closed",
					zap.String("event", xmq.EventProducerConfirm),
				)
				return
			}
			if e != nil {
				atomic.StoreInt32(&cc.broken, 1)
				cc.close(ctx)
				cc.logger.Warn(
					ctx,
					fmt.Sprintf("confirm channel closed by broker, err is: %s", e),
					zap.String("event", xmq.EventProducerConfirm),
				)
			}
		case <-ctx.Done():
			// 上层取消，安全退出
			cc.close(ctx)
		}
	}()
	return cc, nil
}

// confirmRouter: 读取 NotifyPublish 并路由给 pendingRec
// 兼容 broker 批量确认：当收到某个 deliveryTag 的 confirm 时，会一次性唤醒并删除所有 tag <= deliveryTag 的 pending。
// 写入是非阻塞的；如果写阻塞，则关闭 pending chan 并回收 channel。
func (cc *confirmChannel) confirmRouter(ctx context.Context) {
	confSrc := cc.ch.NotifyPublish(make(chan amqp.Confirmation, 128))
	for {
		select {
		case c, ok := <-confSrc:
			if !ok {
				goto exit
			}
			handled := cc.pending.ackLE(c.DeliveryTag, c.Ack)
			if handled > 0 {
				//cc.logger.Info(
				//	ctx,
				//	fmt.Sprintf("confirm router: batch confirm handled， deliveryTag: %d handledCount: %d ack: %t", c.DeliveryTag, handled, c.Ack),
				//	zap.String("event", xmq.EventProducerConfirm),
				//)
				if c.Ack {
					atomic.StoreInt32(&cc.consecTO, 0)
				}
				continue
			}

			// 若未找到任何 pending（可能是已经超时被清理），记录警告
			cc.logger.Warn(
				ctx,
				fmt.Sprintf("confirm router: no pending found for delivery_tag: %d ack: %t (maybe timedout or reclaimed)", c.DeliveryTag, c.Ack),
				zap.String("event", xmq.EventProducerConfirm),
			)
		case <-cc.closeCh:
			goto exit
		case <-ctx.Done():
			goto exit
		}
	}
exit:
	atomic.StoreInt32(&cc.closed, 1)
	// 关闭并释放所有剩余 pending（标记为失败）
	drained := cc.pending.drainAll()
	if drained > 0 {
		cc.logger.Warn(
			ctx,
			fmt.Sprintf("confirm router: drained: %d remaining pending on channel close", drained),
			zap.String("event", xmq.EventProducerConfirm),
		)
	}
}

// publishWithConfirm: 在 cc.mu 下保证 seq 与 publish 原子性
// - 若 GetNextPublishSeqNo() == 0 则表示 channel 已经 reset，需要回收 -> 返回错误
// - 等待 confirm 的逻辑会在 timeout 后清理 pending 并根据 consecutive threshold 决定是否回收 channel
func (cc *confirmChannel) publishWithConfirm(ctx context.Context, exchange, routingKey string, pub amqp.Publishing, timeout time.Duration, consecThreshold int32) error {
	// quick checks
	if atomic.LoadInt32(&cc.closed) == 1 || atomic.LoadInt32(&cc.broken) == 1 {
		return fmt.Errorf("confirmChannel closed or broken")
	}

	// 若 pending 数量异常大，记录警告（防护；如需更强行为可在这里返回错误/阻塞）
	if cc.pending.len() >= defaultMaxPendingPerChannel {
		cc.logger.Warn(
			ctx,
			fmt.Sprintf("publishWithConfirm: pending size %d exceeds threshold %d", cc.pending.len(), defaultMaxPendingPerChannel),
			zap.String("event", xmq.EventProducerChannel),
		)
		// 这里不直接拒绝 publish，为了兼容你现有逻辑；若需要强回压可返回错误。
	}

	// lock to ensure seq/publish atomicity
	cc.mu.Lock()
	seq := cc.ch.GetNextPublishSeqNo()
	if seq == 0 {
		cc.mu.Unlock()
		atomic.StoreInt32(&cc.broken, 1)
		cc.close(ctx)
		return fmt.Errorf("confirmChannel invalid next seq 0")
	}

	// 若检测到 seq <= lastSeq（并且 lastSeq != 0），说明 channel 重置或 wrap
	if cc.lastSeq != 0 && seq <= cc.lastSeq {
		cc.mu.Unlock()
		atomic.StoreInt32(&cc.broken, 1)
		cc.close(ctx)
		cc.logger.Warn(
			ctx,
			fmt.Sprintf("confirmChannel detected sequence reset/wrap - marking channel broken，seq: %d last_seq: %d", seq, cc.lastSeq),
			zap.String("event", xmq.EventProducerChannel),
		)
		return fmt.Errorf("confirmChannel sequence reset or wrap detected (seq=%d last=%d)", seq, cc.lastSeq)
	}

	// 正常，记录 lastSeq 并 append pending
	cc.lastSeq = seq
	rec := &pendingRec{ch: make(chan amqp.Confirmation, 1)}
	cc.pending.add(seq, rec)
	cc.lastSeqA.Store(seq) // 在 pending 已 append 后写入原子副本

	// 执行发布（在 unlock 之前已把 pending 写入）
	err := cc.ch.PublishWithContext(ctx, exchange, routingKey, false, false, pub)
	cc.mu.Unlock()

	if err != nil {
		// 快速路径：原子副本对比，无锁判断
		if cc.lastSeqA.Load() == seq {
			if cc.pending.popTailIfMatch(seq) {
				atomic.StoreInt32(&cc.broken, 1)
				cc.close(ctx)
				return fmt.Errorf("publish failed: %w", err)
			}
		}
		// 有界反向扫描（极罕见）：尝试在尾部附近扫描有限元素
		if cc.pending.removeTailNearby(seq, defaultTailScanLimit) {
			atomic.StoreInt32(&cc.broken, 1)
			cc.close(ctx)
			return fmt.Errorf("publish failed: %w", err)
		}

		// 最后保险：没有命中 -> 标记 broken 并 close，confirmRouter 的 drainAll 会清理剩余 pending
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
		// 精确清理：优先尝试从尾部精确删除该 seq（常见情况），
		// 如果失败再做有界反向扫描；避免使用 ackLE 吞掉前缀未超时的 pending。
		if !cc.pending.popTailIfMatch(seq) && !cc.pending.removeTailNearby(seq, defaultTailScanLimit) {
			if cc.pending.removeByTag(seq) {
				// removed successfully
			} else {
				cc.logger.Warn(
					ctx,
					fmt.Sprintf("confirm timeout: cleanup missed for seq %d (pending_len=%d)", seq, cc.pending.len()),
					zap.String("event", xmq.EventProducerConfirm),
				)
			}
		}
		// 将该 seq 从 pending 中移除（避免内存泄露）
		// 注意：此处我们不能高效地从任意位置删除（因为是 slice），
		// 但通常超时的 seq 会是最小（按顺序），因此前缀删除将在 confirmRouter 中触发或超时处理。
		// 为安全起见，尝试按前缀删除到 seq（可能会移除更多，属于容忍范围）
		// cc.pending.ackLE(seq, false)

		if consecThreshold > 0 && newCnt >= consecThreshold {
			atomic.StoreInt32(&cc.broken, 1)
			cc.close(ctx)
			cc.logger.Warn(
				ctx,
				fmt.Sprintf("channel consecutive %d confirm timeout reached deliveryTag: %d", newCnt, seq),
				zap.String("event", xmq.EventProducerConfirm),
			)
		} else {
			cc.logger.Warn(
				ctx,
				fmt.Sprintf("confirm timeout (transient), consecutive %d confirm deliveryTag: %d", newCnt, seq),
				zap.String("event", xmq.EventProducerConfirm),
			)
		}
		return xmq.ErrPublishTimeout
	case <-ctx.Done():
		// publish ctx canceled：精确清理该 seq，避免误杀前缀 pending
		if !cc.pending.popTailIfMatch(seq) && !cc.pending.removeTailNearby(seq, defaultTailScanLimit) {
			// 兜底线性删除（忽略返回值）
			_ = cc.pending.removeByTag(seq)
		}
		return ctx.Err()
	case <-cc.closeCh:
		// channel 被关闭：尽量精确移除该 seq（drainAll 会在 confirmRouter 触发）
		if !cc.pending.popTailIfMatch(seq) && !cc.pending.removeTailNearby(seq, defaultTailScanLimit) {
			_ = cc.pending.removeByTag(seq)
		}
		return fmt.Errorf("confirmChannel closed")
	}
}

// close 安全关闭 confirmChannel
func (cc *confirmChannel) close(_ context.Context) {
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

	wg         sync.WaitGroup
	stopCtx    context.Context
	stopCancel context.CancelFunc
}

// newProducerConnManager 生产连接管理
func newProducerConnManager(_ context.Context, cfg *ProducerConnConfig) *producerConnManager {
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
	m.conn.Store((*amqp.Connection)(nil))
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
	m.stopCtx, m.stopCancel = context.WithCancel(context.Background())

	// 创建连接
	conn, err := amqp.Dial(m.cfg.URL)
	if err != nil {
		// prevent leak
		if m.stopCancel != nil {
			m.stopCancel()
		}
		return fmt.Errorf("dial: %w", err)
	}
	// 创建 confirm-channel 池
	m.poolMu.Lock()
	if err := m.refreshPool(m.stopCtx, conn); err != nil {
		m.poolMu.Unlock()
		_ = conn.Close()
		if m.stopCancel != nil {
			m.stopCancel()
		}
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
	go m.reconnectWatcher(m.stopCtx)

	m.logger.Info(
		ctx,
		"producer conn success",
		zap.String("event", xmq.EventProducerConnection),
	)
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

// GetConnection 返回当前底层 *amqp.Connection，若无可用连接返回 ErrNoConnection
// 注意：返回的 *amqp.Connection 是原始对象，调用方负责不要在并发上下文误用其 Channel() 的关闭等操作。
// 推荐调用方使用 conn.Channel() 后立即 defer ch.Close()。
func (m *producerConnManager) GetConnection(ctx context.Context) (*amqp.Connection, error) {
	conn := m.loadConn(ctx)
	if conn == nil || conn.IsClosed() {
		return nil, xmq.ErrNoConnection
	}
	return conn, nil
}

// GetProducerChannel  从池中获取一个可用 channel（跳过 closed/broken）
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
		// 成功放回
		return
	default:
		// 池已满 - 丢弃该 channel 并记录警告（可以改为等待 PutBackTimeout）
		timer := time.NewTimer(500 * time.Millisecond)
		defer timer.Stop()
		select {
		case m.pool <- cc:
			return
		case <-timer.C:
			cc.close(ctx)
			m.logger.Warn(
				ctx,
				fmt.Sprintf("put producer channel: pool full after 500ms, discard channel, pool_cap: %d pool_len: %d", cap(m.pool), len(m.pool)),
				zap.String("event", xmq.EventProducerChannel),
			)
		}
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
	initDelay := m.cfg.ReconnectInitialDelay
	if initDelay <= 0 {
		initDelay = 500 * time.Millisecond
	}
	backoff := initDelay
	maxDelay := m.cfg.ReconnectMaxDelay
	if maxDelay <= 0 {
		maxDelay = 30 * time.Second
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
				m.logger.Warn(
					ctx,
					fmt.Sprintf("producer reconnect fail, err is: %s", err),
					zap.String("event", xmq.EventProducerReconnection),
				)
				time.Sleep(backoff)
				backoff *= 2
				if backoff > maxDelay {
					backoff = maxDelay
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
				if backoff > maxDelay {
					backoff = maxDelay
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
			backoff = initDelay
			m.logger.Info(
				ctx,
				"producer reconnected success",
				zap.String("event", xmq.EventProducerReconnection),
			)
			continue
		}
		// wait for close notification
		closeCh := conn.NotifyClose(make(chan *amqp.Error, 1))
		select {
		case err := <-closeCh:
			if err != nil {
				m.logger.Warn(
					ctx,
					fmt.Sprintf("producer conn closed, err is: %s", err),
					zap.String("event", xmq.EventProducerCloseConnection),
				)
			}
		case <-ctx.Done():
			return
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
	// cancel background goroutines
	if m.stopCancel != nil {
		m.stopCancel()
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
	timeout := time.After(15 * time.Second)
	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()
	for {
		if atomic.LoadInt64(&m.inflight) == 0 {
			break
		}
		select {
		case <-timeout:
			// force return, some inflight didn't finish
			m.logger.Error(
				ctx,
				"producer conn close timeout waiting inflight",
				zap.String("event", xmq.EventProducerCloseConnection),
				zap.String("error", fmt.Sprintf("producer conn close,inflight(%d) didn't finish", m.inflight)),
			)
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
	m.logger.Info(
		ctx,
		"producer conn close success",
		zap.String("event", xmq.EventProducerCloseConnection),
	)
	return nil
}

// loadConn 获取mq client
func (m *producerConnManager) loadConn(_ context.Context) *amqp.Connection {
	v := m.conn.Load()
	if v == nil {
		return nil
	}
	return v.(*amqp.Connection)
}

// storeConn 设置mq client
func (m *producerConnManager) storeConn(_ context.Context, c *amqp.Connection) { m.conn.Store(c) }

type consumerConnManager struct {
	cfg        *ConsumerConnConfig
	conn       atomic.Value // *amqp.Connection
	closed     int32
	logger     xmq.Logger
	stopCtx    context.Context
	stopCancel context.CancelFunc
}

func newConsumerConnManager(_ context.Context, cfg *ConsumerConnConfig) *consumerConnManager {
	if cfg == nil {
		panic("consumer config nil")
	}
	var l xmq.Logger = &nopLogger{}
	if cfg.Logger != nil {
		l = cfg.Logger
	}
	m := &consumerConnManager{
		cfg:    cfg,
		logger: l,
	}
	m.conn.Store((*amqp.Connection)(nil))
	return m
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
	m.stopCtx, m.stopCancel = context.WithCancel(context.Background())
	go m.reconnectWatcher(m.stopCtx)
	m.logger.Info(
		ctx,
		"consumer conn success",
		zap.String("event", xmq.EventConsumerConnection),
	)
	return nil
}

// GetConnection consumer side
func (m *consumerConnManager) GetConnection(ctx context.Context) (*amqp.Connection, error) {
	conn := m.loadConn(ctx)
	if conn == nil || conn.IsClosed() {
		return nil, xmq.ErrNoConnection
	}
	return conn, nil
}

func (m *consumerConnManager) reconnectWatcher(ctx context.Context) {
	initDelay := m.cfg.ReconnectInitialDelay
	if initDelay <= 0 {
		initDelay = 500 * time.Millisecond
	}
	backoff := initDelay
	maxDelay := m.cfg.ReconnectMaxDelay
	if maxDelay <= 0 {
		maxDelay = 30 * time.Second
	}
	for {
		if atomic.LoadInt32(&m.closed) == 1 {
			return
		}
		conn := m.loadConn(ctx)
		if conn == nil || conn.IsClosed() {
			newConn, err := amqp.Dial(m.cfg.URL)
			if err != nil {
				m.logger.Warn(
					ctx,
					fmt.Sprintf("consumer reconnect dial failed, err is: %s", err),
					zap.String("event", xmq.EventConsumerReconnection),
				)
				time.Sleep(backoff)
				backoff *= 2
				if backoff > maxDelay {
					backoff = maxDelay
				}
				continue
			}
			m.storeConn(ctx, newConn)
			backoff = initDelay // reset to computed initDelay
			m.logger.Info(
				ctx,
				"consumer reconnected",
				zap.String("event", xmq.EventConsumerReconnection),
			)
			continue
		}
		// wait for close
		closeCh := conn.NotifyClose(make(chan *amqp.Error, 1))
		select {
		case err := <-closeCh:
			if err != nil {
				m.logger.Warn(
					ctx,
					fmt.Sprintf("consumer connection closed, err is: %s", err),
					zap.String("event", xmq.EventConsumerReconnection),
				)
			}
		case <-ctx.Done():
			// 上层取消，安全退出
			return
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
	// cancel background goroutine if any
	if m.stopCancel != nil {
		m.stopCancel()
	}
	if conn := m.loadConn(ctx); conn != nil {
		_ = conn.Close()
		m.logger.Info(
			ctx,
			"consumer conn close success",
			zap.String("event", xmq.EventConsumerCloseConnection),
		)
		m.storeConn(ctx, nil)
	}
}

func (m *consumerConnManager) loadConn(_ context.Context) *amqp.Connection {
	v := m.conn.Load()
	if v == nil {
		return nil
	}
	return v.(*amqp.Connection)
}

func (m *consumerConnManager) storeConn(_ context.Context, c *amqp.Connection) { m.conn.Store(c) }
