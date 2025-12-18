package worker

import (
	"context"
	"snowgo/internal/constant"
	worker "snowgo/internal/worker/handler"
	"snowgo/pkg/xmq"
	"snowgo/pkg/xmq/rabbitmq"
	"time"
)

type Deps struct {
	Logger xmq.Logger
}

func RegisterAll(ctx context.Context, consumer *rabbitmq.Consumer, deps *Deps) error {

	// 示例消费
	if err := consumer.Register(
		ctx,
		constant.ExampleNormalQueue,
		worker.NewExampleHandler(deps.Logger).ExampleHandle,
		&xmq.ConsumerMeta{
			Prefetch:       4,                // Qos prefetch count
			WorkerNum:      2,                // 并发 worker 数
			RetryLimit:     2,                // 同步重试次数（包括第一次尝试）
			HandlerTimeout: 10 * time.Second, // handler 超时时间
		}); err != nil {
		return err
	}
	if err := consumer.Register(
		ctx,
		constant.ExampleDelayedQueue,
		worker.NewExampleHandler(deps.Logger).ExampleHandle,
		&xmq.ConsumerMeta{
			Prefetch:       4,                // Qos prefetch count
			WorkerNum:      2,                // 并发 worker 数
			RetryLimit:     2,                // 同步重试次数（包括第一次尝试）
			HandlerTimeout: 10 * time.Second, // handler 超时时间
		}); err != nil {
		return err
	}

	return nil
}
