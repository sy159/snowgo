package rabbitmq_test

import (
	"context"
	"snowgo/pkg/xlogger"
	"snowgo/pkg/xmq"
	"snowgo/pkg/xmq/rabbitmq"
	"sync/atomic"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func rabbitURL() string {
	return "amqp://snow_dev:zx.123@127.0.0.1:5672/dev"
}

func mustDialOrSkip(t *testing.T) *amqp.Connection {
	t.Helper()
	conn, err := amqp.Dial(rabbitURL())
	if err != nil {
		t.Skipf("skipping test: cannot dial RabbitMQ: %v", err)
	}
	return conn
}

// ----------------- 功能测试 -----------------
func TestProducer_Publish(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn := mustDialOrSkip(t)
	defer conn.Close()

	reg := rabbitmq.NewRegistry(conn).
		Add(rabbitmq.MQDeclare{
			Name: "snow_test.exchange",
			Type: xmq.DirectExchange,
			Queues: []rabbitmq.QueueDeclare{
				{Name: "user.create.queue", RoutingKeys: []string{"user.create1", "user.create2"}},
				{Name: "user.delete.queue", RoutingKeys: []string{"user.delete"}},
			},
		}).
		Add(rabbitmq.MQDeclare{
			Name: "snow_test.delayed.exchange",
			Type: xmq.DelayedExchange,
			Queues: []rabbitmq.QueueDeclare{
				{Name: "order.pay.queue", RoutingKeys: []string{"order.pay"}},
			},
		})

	err := reg.RegisterAll(ctx)
	assert.NoError(t, err)

	producerLogger := xlogger.NewLogger("./logs", "rabbitmq-producer", xlogger.WithFileMaxAgeDays(30))
	config := rabbitmq.NewProducerConnConfig(
		rabbitURL(),
		rabbitmq.WithProducerLogger(producerLogger),
		rabbitmq.WithChannelPoolSize(128), // 池子大点，用于压测
		rabbitmq.WithConfirmTimeout(0),    // 压测场景下不需要重试
	)
	producer, err := rabbitmq.NewProducer(ctx, config)
	require.NoError(t, err)
	defer producer.Close(ctx)

	// 普通消息
	msg := &xmq.Message{Body: []byte("hello"), Headers: map[string]interface{}{"k": "v"}}
	assert.NoError(t, producer.Publish(ctx, "snow_test.exchange", "user.create1", msg))

	// 延迟消息
	dmsg := &xmq.Message{Body: []byte("hello-delayed")}
	assert.NoError(t, producer.PublishDelayed(ctx, "snow_test.delayed.exchange", "order.pay", dmsg, 1000))
}

// ----------------- 压测基准 -----------------
func BenchmarkProducerPublish(b *testing.B) {
	exchange := "snow_test.exchange"
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	conn := mustDialOrSkip(&testing.T{})
	defer conn.Close()
	defer cancel()

	reg := rabbitmq.NewRegistry(conn).
		Add(rabbitmq.MQDeclare{
			Name: exchange,
			Type: xmq.DirectExchange,
			Queues: []rabbitmq.QueueDeclare{
				{Name: "user.delete.queue", RoutingKeys: []string{"user.delete"}},
			},
		})

	err := reg.RegisterAll(ctx)
	assert.NoError(b, err)

	producerLogger := xlogger.NewLogger("./logs", "rabbitmq-producer", xlogger.WithFileMaxAgeDays(30))
	cfg := rabbitmq.NewProducerConnConfig(rabbitURL(), rabbitmq.WithProducerLogger(producerLogger))
	p, err := rabbitmq.NewProducer(ctx, cfg)
	if err != nil {
		b.Fatalf("NewProducer failed: %v", err)
	}
	defer p.Close(ctx)

	var success uint64
	var fail uint64

	b.ResetTimer()
	benchMsg := &xmq.Message{Body: []byte("bench-message")}
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if err := p.Publish(ctx, exchange, "user.delete", benchMsg); err != nil {
				atomic.AddUint64(&fail, 1)
			} else {
				atomic.AddUint64(&success, 1)
			}
		}
	})
	b.StopTimer()

	b.ReportMetric(float64(success), "success_ops")
	b.ReportMetric(float64(fail), "fail_ops")
	b.ReportMetric(float64(success)/b.Elapsed().Seconds(), "success/sec")
}
