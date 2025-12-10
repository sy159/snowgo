package rabbitmq_test

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"snowgo/pkg/xlogger"
	"snowgo/pkg/xmq"
	"snowgo/pkg/xmq/rabbitmq"
)

// TestConsumer_ProcessMessage_WhenQueueExists  测试消费固定数量消息
func TestConsumer_ProcessMessage_WhenQueueExists(t *testing.T) {
	exchange := "snow_test.exchange"
	queue := "user.delete.queue"
	expectedCount := 5 // 消费数量

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn := mustDialOrSkip(t)
	defer conn.Close()

	// declare topology
	reg := rabbitmq.NewRegistry(conn).
		Add(rabbitmq.MQDeclare{
			Name: exchange,
			Type: xmq.DirectExchange,
			Queues: []rabbitmq.QueueDeclare{
				{Name: queue, RoutingKeys: []string{"user.delete"}},
			},
		})
	require.NoError(t, reg.RegisterAll(ctx))

	consumerLogger := xlogger.NewLogger("./logs", "rabbitmq-consumer", xlogger.WithFileMaxAgeDays(7))
	cfg := &rabbitmq.ConsumerConnConfig{
		URL:                 rabbitURL(),
		Logger:              consumerLogger,
		ConsumerBackoffBase: 100 * time.Millisecond,
		ReconnectMaxDelay:   2 * time.Second,
	}

	consumer, err := rabbitmq.NewConsumer(ctx, cfg)
	require.NoError(t, err)
	defer func() { _ = consumer.Stop(ctx) }()

	var consumed int32
	done := make(chan struct{})

	// 消费处理函数，倒数第二条消息模拟失败，最后一条消息模拟 panic
	handler := func(ctx context.Context, m xmq.Message) error {
		count := atomic.AddInt32(&consumed, 1)
		t.Logf("consumed message_id=%s body=%s", m.MessageId, string(m.Body))

		switch count {
		case int32(expectedCount - 1):
			return fmt.Errorf("simulated error for penultimate message")
		case int32(expectedCount):
			panic("simulated panic for last message")
		default:
			// 正常消费
		}

		// 消费次数到达 expectedCount，通知 done
		if count >= int32(expectedCount) {
			select {
			case done <- struct{}{}:
			default:
			}
		}
		return nil
	}

	// 注册 handler
	require.NoError(t, consumer.Register(context.Background(), queue, handler, &xmq.ConsumerMeta{
		Prefetch:       4,
		WorkerNum:      2,
		RetryLimit:     2,
		HandlerTimeout: 10 * time.Second,
	}))

	// 启动消费者
	consumerCtx, consumerCancel := context.WithCancel(context.Background())
	defer consumerCancel()
	go func() { _ = consumer.Start(consumerCtx) }()

	// 等待消费完成或超时
	select {
	case <-done:
		t.Logf("all %d messages processed (some may fail/panic)", expectedCount)
	case <-ctx.Done():
		t.Fatalf("timeout waiting for %d messages, consumed=%d", expectedCount, atomic.LoadInt32(&consumed))
	}
}

// TestConsumer_GracefulWhenQueueMissing  队列不存在
func TestConsumer_GracefulWhenQueueMissing(t *testing.T) {
	t.Parallel()

	conn := mustDialOrSkip(t)
	_ = conn.Close()

	consumerLogger := xlogger.NewLogger("./logs", "rabbitmq-consumer", xlogger.WithFileMaxAgeDays(7))
	cfg := &rabbitmq.ConsumerConnConfig{
		URL:                 rabbitURL(),
		Logger:              consumerLogger,
		ConsumerBackoffBase: 100 * time.Millisecond,
		ReconnectMaxDelay:   2 * time.Second,
	}

	consumer, err := rabbitmq.NewConsumer(context.Background(), cfg)
	require.NoError(t, err)
	defer func() {
		_ = consumer.Stop(context.Background())
	}()

	// register a queue name we won't declare
	missingQueue := "snow_test.missing.queue"
	handler := func(ctx context.Context, m xmq.Message) error {
		return nil
	}
	require.NoError(t, consumer.Register(context.Background(), missingQueue, handler, nil))

	// start consumer
	startCtx, startCancel := context.WithCancel(context.Background())
	go func() {
		_ = consumer.Start(startCtx)
	}()
	time.Sleep(500 * time.Millisecond)
	startCancel()
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer stopCancel()
	require.NoError(t, consumer.Stop(stopCtx))
}
