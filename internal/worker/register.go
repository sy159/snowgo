package worker

import (
	"context"
	"snowgo/internal/di"
	worker "snowgo/internal/worker/handler"
	"snowgo/pkg/xmq"
	"snowgo/pkg/xmq/rabbitmq"
	"time"
)

func RegisterAll(ctx context.Context, consumer *rabbitmq.Consumer, logger xmq.Logger, container *di.Container) error {

	// 示例
	if err := consumer.Register(
		ctx,
		"user.delete.queue",
		worker.NewExampleHandler(logger).ExampleHandle,
		&xmq.ConsumerMeta{
			Prefetch:       4,
			WorkerNum:      2,
			RetryLimit:     2,
			HandlerTimeout: 10 * time.Second,
		}); err != nil {
		return err
	}

	return nil
}
