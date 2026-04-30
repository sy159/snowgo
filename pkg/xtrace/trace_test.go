package xtrace

import (
	"context"
	"testing"

	"snowgo/pkg/xauth"
)

func TestGetTraceID(t *testing.T) {
	t.Run("empty context", func(t *testing.T) {
		if got := GetTraceID(context.Background()); got != "" {
			t.Errorf("GetTraceID(empty) = %q, want empty", got)
		}
	})

	t.Run("with trace ID", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), xauth.XTraceId, "abc-123")
		if got := GetTraceID(ctx); got != "abc-123" {
			t.Errorf("GetTraceID = %q, want %q", got, "abc-123")
		}
	})

	t.Run("wrong type value", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), xauth.XTraceId, 12345)
		if got := GetTraceID(ctx); got != "" {
			t.Errorf("GetTraceID(wrong type) = %q, want empty", got)
		}
	})
}

func TestNewContextWithTrace(t *testing.T) {
	ctx := NewContextWithTrace(context.Background(), "trace-xyz")
	got := GetTraceID(ctx)
	if got != "trace-xyz" {
		t.Errorf("GetTraceID after NewContextWithTrace = %q, want %q", got, "trace-xyz")
	}
}
