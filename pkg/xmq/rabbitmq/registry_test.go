package rabbitmq_test

import (
	"context"
	"snowgo/internal/constant"
	"snowgo/pkg/xmq"
	"snowgo/pkg/xmq/rabbitmq"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry(t *testing.T) {
	ctx := context.Background()

	// 1. 拿到连接（TestMain 已启动 Docker）
	conn, err := amqp.Dial("amqp://snow_dev:zx.123@127.0.0.1:5672/dev")
	require.NoError(t, err)
	defer conn.Close()

	// 2. 构造 Registry 并注册
	reg := rabbitmq.NewRegistry(conn).
		Add(rabbitmq.MQDeclare{
			Name: constant.NormalExchange,
			Type: xmq.DirectExchange,
			Queues: []rabbitmq.QueueDeclare{
				{Name: constant.ExampleNormalQueue, RoutingKeys: []string{constant.ExampleNormalRoutingKey}},
			},
		}).
		Add(rabbitmq.MQDeclare{
			Name: constant.DelayedExchange,
			Type: xmq.DelayedExchange,
			Queues: []rabbitmq.QueueDeclare{
				{Name: constant.ExampleDelayedQueue, RoutingKeys: []string{constant.ExampleDelayedRoutingKey}},
			},
		})

	err = reg.RegisterAll(ctx)
	assert.NoError(t, err)

	// 3. 简单验证：队列存在即可
	ch, err := conn.Channel()
	require.NoError(t, err)
	defer ch.Close()

	// 队列存在性检查
	queues := []string{constant.ExampleNormalQueue, constant.ExampleDelayedQueue}
	for _, q := range queues {
		_, err := ch.QueueDeclarePassive(q, true, false, false, false, nil)
		assert.NoError(t, err)
	}

	// Exchange 被动声明检查
	exchanges := map[string]string{
		constant.NormalExchange:  "direct",
		constant.DelayedExchange: "x-delayed-message",
	}
	for name, kind := range exchanges {
		err := ch.ExchangeDeclarePassive(name, kind, true, false, false, false, nil)
		assert.NoError(t, err)
	}
}
