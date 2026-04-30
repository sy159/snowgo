package xauth

import (
	"context"
	"testing"

	e "snowgo/pkg/xerror"
)

func TestGetContext(t *testing.T) {
	t.Run("empty context", func(t *testing.T) {
		ctx := context.Background()
		c := GetContext(ctx)
		if c.TraceId != "" {
			t.Errorf("expected empty TraceId, got %q", c.TraceId)
		}
		if c.IP != "" {
			t.Errorf("expected empty IP, got %q", c.IP)
		}
		if c.UserAgent != "" {
			t.Errorf("expected empty UserAgent, got %q", c.UserAgent)
		}
	})

	t.Run("with values", func(t *testing.T) {
		ctx := context.Background()
		ctx = context.WithValue(ctx, XTraceId, "trace-123")
		ctx = context.WithValue(ctx, XIp, "10.0.0.1")
		ctx = context.WithValue(ctx, XUserAgent, "test-agent")
		c := GetContext(ctx)
		if c.TraceId != "trace-123" {
			t.Errorf("TraceId = %q, want %q", c.TraceId, "trace-123")
		}
		if c.IP != "10.0.0.1" {
			t.Errorf("IP = %q, want %q", c.IP, "10.0.0.1")
		}
		if c.UserAgent != "test-agent" {
			t.Errorf("UserAgent = %q, want %q", c.UserAgent, "test-agent")
		}
	})
}

func TestGetUserContext(t *testing.T) {
	t.Run("missing user ID returns forbidden", func(t *testing.T) {
		ctx := context.Background()
		_, err := GetUserContext(ctx)
		if err == nil {
			t.Fatal("expected error for missing user ID")
		}
		if err.Error() != e.HttpForbidden.GetErrMsg() {
			t.Errorf("expected forbidden error, got %q", err.Error())
		}
	})

	t.Run("zero user ID returns forbidden", func(t *testing.T) {
		ctx := context.Background()
		ctx = context.WithValue(ctx, XUserId, int32(0))
		_, err := GetUserContext(ctx)
		if err == nil {
			t.Fatal("expected error for zero user ID")
		}
	})

	t.Run("negative user ID returns forbidden", func(t *testing.T) {
		ctx := context.Background()
		ctx = context.WithValue(ctx, XUserId, int32(-1))
		_, err := GetUserContext(ctx)
		if err == nil {
			t.Fatal("expected error for negative user ID")
		}
	})

	t.Run("valid user ID", func(t *testing.T) {
		ctx := context.Background()
		ctx = context.WithValue(ctx, XUserId, int32(42))
		ctx = context.WithValue(ctx, XUserName, "alice")
		ctx = context.WithValue(ctx, XSessionId, "sess-1")
		ctx = context.WithValue(ctx, XTraceId, "trace-1")
		uc, err := GetUserContext(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if uc.UserId != 42 {
			t.Errorf("UserId = %d, want 42", uc.UserId)
		}
		if uc.Username != "alice" {
			t.Errorf("Username = %q, want %q", uc.Username, "alice")
		}
		if uc.SessionId != "sess-1" {
			t.Errorf("SessionId = %q, want %q", uc.SessionId, "sess-1")
		}
		if uc.TraceId != "trace-1" {
			t.Errorf("TraceId = %q, want %q", uc.TraceId, "trace-1")
		}
	})

	t.Run("wrong type for user ID returns forbidden", func(t *testing.T) {
		ctx := context.Background()
		ctx = context.WithValue(ctx, XUserId, "not-an-int")
		_, err := GetUserContext(ctx)
		if err == nil {
			t.Fatal("expected error for wrong type user ID")
		}
	})
}
