package worker

import (
	"context"
	"fmt"
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
	h.Logger.Info(ctx, fmt.Sprintf("example handle success, msg is: %s", msg.Body))
	return nil
}
