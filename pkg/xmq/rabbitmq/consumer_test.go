//go:build integration

package rabbitmq_test

import (
	"context"
	"fmt"
	"snowgo/internal/constant"
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
	exchange := constant.NormalExchange
	queue := "snow_test.consumer.basic"
	routingKey := "snow_test.consumer.basic.routing"

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn := mustDialOrSkip(t)
	defer func() { _ = conn.Close() }()

	// declare topology
	reg := rabbitmq.NewRegistry(conn).
		Add(rabbitmq.MQDeclare{
			Name: exchange,
			Type: xmq.DirectExchange,
			Queues: []rabbitmq.QueueDeclare{
				{Name: queue, RoutingKeys: []string{routingKey}},
			},
		})
	require.NoError(t, reg.RegisterAll(ctx))

	// publish a test message
	producerLogger := xlogger.NewLogger(t.TempDir(), "rabbitmq-producer", xlogger.WithFileMaxAgeDays(7))
	producerCfg := rabbitmq.NewProducerConnConfig(
		rabbitURL(),
		rabbitmq.WithProducerLogger(producerLogger),
	)
	producer, err := rabbitmq.NewProducer(ctx, producerCfg)
	require.NoError(t, err)
	defer func() { _ = producer.Close(ctx) }()

	msg := &xmq.Message{Body: []byte("hello-consumer"), MessageId: "test-msg-001"}
	require.NoError(t, producer.Publish(ctx, exchange, routingKey, msg))

	// create consumer
	consumerLogger := xlogger.NewLogger(t.TempDir(), "rabbitmq-consumer", xlogger.WithFileMaxAgeDays(7))
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

	handler := func(ctx context.Context, m xmq.Message) error {
		atomic.AddInt32(&consumed, 1)
		t.Logf("consumed message_id=%s body=%s", m.MessageId, string(m.Body))
		if string(m.Body) == "hello-consumer" {
			select {
			case done <- struct{}{}:
			default:
			}
		}
		return nil
	}

	require.NoError(t, consumer.Register(context.Background(), queue, handler, &xmq.ConsumerMeta{
		Prefetch:       1,
		WorkerNum:      1,
		RetryLimit:     1,
		HandlerTimeout: 5 * time.Second,
	}))

	// start consumer
	consumerCtx, consumerCancel := context.WithCancel(context.Background())
	defer consumerCancel()
	go func() { _ = consumer.Start(consumerCtx) }()

	// wait for consumption or timeout
	select {
	case <-done:
		t.Logf("message consumed, consumed=%d", atomic.LoadInt32(&consumed))
		if atomic.LoadInt32(&consumed) < 1 {
			t.Fatal("expected at least 1 message consumed")
		}
	case <-time.After(10 * time.Second):
		t.Fatalf("timeout waiting for message, consumed=%d", atomic.LoadInt32(&consumed))
	}
}

// TestConsumer_HandlerRetryOnError  测试 handler 失败后重试
func TestConsumer_HandlerRetryOnError(t *testing.T) {
	exchange := constant.NormalExchange
	queue := "snow_test.consumer.retry"
	routingKey := "snow_test.consumer.retry.routing"

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn := mustDialOrSkip(t)
	defer func() { _ = conn.Close() }()

	reg := rabbitmq.NewRegistry(conn).
		Add(rabbitmq.MQDeclare{
			Name: exchange,
			Type: xmq.DirectExchange,
			Queues: []rabbitmq.QueueDeclare{
				{Name: queue, RoutingKeys: []string{routingKey}},
			},
		})
	require.NoError(t, reg.RegisterAll(ctx))

	// publish a test message
	producerLogger := xlogger.NewLogger(t.TempDir(), "rabbitmq-producer", xlogger.WithFileMaxAgeDays(7))
	producer, err := rabbitmq.NewProducer(ctx, rabbitmq.NewProducerConnConfig(
		rabbitURL(),
		rabbitmq.WithProducerLogger(producerLogger),
	))
	require.NoError(t, err)
	defer func() { _ = producer.Close(ctx) }()

	msg := &xmq.Message{Body: []byte("retry-test"), MessageId: "test-msg-retry"}
	require.NoError(t, producer.Publish(ctx, exchange, routingKey, msg))

	// create consumer with handler that fails first attempt, then succeeds
	consumerLogger := xlogger.NewLogger(t.TempDir(), "rabbitmq-consumer", xlogger.WithFileMaxAgeDays(7))
	cfg := &rabbitmq.ConsumerConnConfig{
		URL:                 rabbitURL(),
		Logger:              consumerLogger,
		ConsumerBackoffBase: 100 * time.Millisecond,
		ReconnectMaxDelay:   1 * time.Second,
	}

	consumer, err := rabbitmq.NewConsumer(ctx, cfg)
	require.NoError(t, err)
	defer func() { _ = consumer.Stop(ctx) }()

	var attempts int32
	done := make(chan struct{})

	handler := func(ctx context.Context, m xmq.Message) error {
		n := atomic.AddInt32(&attempts, 1)
		t.Logf("attempt %d: message_id=%s", n, m.MessageId)
		if n == 1 {
			return fmt.Errorf("simulated failure on first attempt")
		}
		select {
		case done <- struct{}{}:
		default:
		}
		return nil
	}

	// RetryLimit=3 to give 3 attempts
	require.NoError(t, consumer.Register(context.Background(), queue, handler, &xmq.ConsumerMeta{
		Prefetch:       1,
		WorkerNum:      1,
		RetryLimit:     3,
		HandlerTimeout: 5 * time.Second,
	}))

	consumerCtx, consumerCancel := context.WithCancel(context.Background())
	defer consumerCancel()
	go func() { _ = consumer.Start(consumerCtx) }()

	select {
	case <-done:
		t.Logf("message consumed after %d attempts", atomic.LoadInt32(&attempts))
		if atomic.LoadInt32(&attempts) < 2 {
			t.Fatalf("expected at least 2 attempts (1 fail + 1 success), got %d", atomic.LoadInt32(&attempts))
		}
	case <-time.After(15 * time.Second):
		t.Fatalf("timeout waiting for retry success, attempts=%d", atomic.LoadInt32(&attempts))
	}
}

// TestConsumer_RegisterValidation  测试 Register 参数校验
func TestConsumer_RegisterValidation(t *testing.T) {
	conn := mustDialOrSkip(t)
	defer func() { _ = conn.Close() }()

	consumerLogger := xlogger.NewLogger(t.TempDir(), "rabbitmq-consumer", xlogger.WithFileMaxAgeDays(7))
	cfg := &rabbitmq.ConsumerConnConfig{
		URL:    rabbitURL(),
		Logger: consumerLogger,
	}

	consumer, err := rabbitmq.NewConsumer(context.Background(), cfg)
	require.NoError(t, err)
	defer func() { _ = consumer.Stop(context.Background()) }()

	t.Run("error: empty queue name", func(t *testing.T) {
		err := consumer.Register(context.Background(), "", func(ctx context.Context, m xmq.Message) error {
			return nil
		}, nil)
		if err == nil {
			t.Fatal("expected error for empty queue name")
		}
	})

	t.Run("error: nil handler", func(t *testing.T) {
		err := consumer.Register(context.Background(), "test-queue", nil, nil)
		if err == nil {
			t.Fatal("expected error for nil handler")
		}
	})
}

// TestConsumer_GracefulWhenQueueMissing  队列不存在
func TestConsumer_GracefulWhenQueueMissing(t *testing.T) {
	t.Parallel()

	conn := mustDialOrSkip(t)
	_ = conn.Close()

	consumerLogger := xlogger.NewLogger(t.TempDir(), "rabbitmq-consumer", xlogger.WithFileMaxAgeDays(7))
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
