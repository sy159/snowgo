package worker

import (
	"context"
	"snowgo/pkg/xmq"
	"time"
)

type ExampleHandler struct {
	Logger xmq.Logger
}

func NewExampleHandler(logger xmq.Logger) *ExampleHandler {
	return &ExampleHandler{
		Logger: logger,
	}
}

func (h *ExampleHandler) ExampleHandle(ctx context.Context, msg xmq.Message) error {
	time.Sleep(time.Second)
	h.Logger.Info(ctx, "example handle success")
	return nil
}
