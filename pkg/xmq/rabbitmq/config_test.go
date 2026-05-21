package rabbitmq

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

// Dummy URL for config-only unit tests (never connects to RabbitMQ).
const testURL = "amqp://test:test@127.0.0.1:5672/testvhost"

// ============================================================
// ProducerConnConfig
// ============================================================

func TestProducerConnConfig(t *testing.T) {
	// === Happy path ===
	t.Run("happy: defaults applied", func(t *testing.T) {
		cfg := NewProducerConnConfig(testURL)
		if cfg.URL != testURL {
			t.Fatalf("URL mismatch: got %s", cfg.URL)
		}
		if cfg.ProducerChannelPoolSize != 8 {
			t.Fatalf("default pool size=8, got %d", cfg.ProducerChannelPoolSize)
		}
		if cfg.PublishGetTimeout != 3*time.Second {
			t.Fatalf("default publish timeout=3s, got %v", cfg.PublishGetTimeout)
		}
		if cfg.ConfirmTimeout != 5*time.Second {
			t.Fatalf("default confirm timeout=5s, got %v", cfg.ConfirmTimeout)
		}
		if cfg.ReconnectInitialDelay != 500*time.Millisecond {
			t.Fatalf("default reconnect delay=500ms, got %v", cfg.ReconnectInitialDelay)
		}
		if cfg.ReconnectMaxDelay != 30*time.Second {
			t.Fatalf("default max reconnect delay=30s, got %v", cfg.ReconnectMaxDelay)
		}
		if cfg.ConsecutiveTimeoutThreshold != 3 {
			t.Fatalf("default consecutive threshold=3, got %d", cfg.ConsecutiveTimeoutThreshold)
		}
		if cfg.Logger == nil {
			t.Fatal("default logger should not be nil (nopLogger)")
		}
	})

	t.Run("happy: options override defaults", func(t *testing.T) {
		cfg := NewProducerConnConfig(testURL,
			WithChannelPoolSize(16),
			WithPublishGetTimeout(10*time.Second),
			WithConfirmTimeout(15*time.Second),
			WithProducerReconnectInitialDelay(1*time.Second),
			WithProducerReconnectMaxDelay(60*time.Second),
			WithConsecTimeoutThreshold(5),
		)
		if cfg.ProducerChannelPoolSize != 16 {
			t.Fatalf("pool size=16, got %d", cfg.ProducerChannelPoolSize)
		}
		if cfg.PublishGetTimeout != 10*time.Second {
			t.Fatalf("publish timeout=10s, got %v", cfg.PublishGetTimeout)
		}
		if cfg.ConfirmTimeout != 15*time.Second {
			t.Fatalf("confirm timeout=15s, got %v", cfg.ConfirmTimeout)
		}
		if cfg.ReconnectInitialDelay != 1*time.Second {
			t.Fatalf("reconnect delay=1s, got %v", cfg.ReconnectInitialDelay)
		}
		if cfg.ReconnectMaxDelay != 60*time.Second {
			t.Fatalf("max reconnect=60s, got %v", cfg.ReconnectMaxDelay)
		}
		if cfg.ConsecutiveTimeoutThreshold != 5 {
			t.Fatalf("threshold=5, got %d", cfg.ConsecutiveTimeoutThreshold)
		}
	})

	// === Boundary values ===
	t.Run("boundary: zero/negative option values ignored", func(t *testing.T) {
		cfg := NewProducerConnConfig(testURL,
			WithChannelPoolSize(0),
			WithChannelPoolSize(-1),
			WithPublishGetTimeout(0),
			WithConfirmTimeout(-5*time.Second),
		)
		if cfg.ProducerChannelPoolSize != 8 {
			t.Fatalf("zero/negative pool size should keep default=8, got %d", cfg.ProducerChannelPoolSize)
		}
		if cfg.PublishGetTimeout != 3*time.Second {
			t.Fatalf("zero publish timeout should keep default=3s, got %v", cfg.PublishGetTimeout)
		}
		if cfg.ConfirmTimeout != 5*time.Second {
			t.Fatalf("negative confirm timeout should keep default=5s, got %v", cfg.ConfirmTimeout)
		}
	})

	// === Expected errors ===
	t.Run("error: empty URL panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic for empty URL")
			}
		}()
		NewProducerConnConfig("")
	})
}

// ============================================================
// ConsumerConnConfig
// ============================================================

func TestConsumerConnConfig(t *testing.T) {
	// === Happy path ===
	t.Run("happy: defaults applied", func(t *testing.T) {
		cfg := NewConsumerConnConfig(testURL)
		if cfg.URL != testURL {
			t.Fatalf("URL mismatch: got %s", cfg.URL)
		}
		if cfg.ConsumerBackoffBase != 1*time.Second {
			t.Fatalf("default backoff base=1s, got %v", cfg.ConsumerBackoffBase)
		}
		if cfg.ReconnectInitialDelay != 500*time.Millisecond {
			t.Fatalf("default reconnect delay=500ms, got %v", cfg.ReconnectInitialDelay)
		}
		if cfg.ReconnectMaxDelay != 30*time.Second {
			t.Fatalf("default max reconnect delay=30s, got %v", cfg.ReconnectMaxDelay)
		}
		if cfg.Logger == nil {
			t.Fatal("default logger should not be nil (nopLogger)")
		}
	})

	t.Run("happy: options override defaults", func(t *testing.T) {
		cfg := NewConsumerConnConfig(testURL,
			WithBackoffBase(2*time.Second),
			WithConsumerReconnectInitialDelay(1*time.Second),
			WithConsumerReconnectMaxDelay(60*time.Second),
		)
		if cfg.ConsumerBackoffBase != 2*time.Second {
			t.Fatalf("backoff=2s, got %v", cfg.ConsumerBackoffBase)
		}
		if cfg.ReconnectInitialDelay != 1*time.Second {
			t.Fatalf("reconnect delay=1s, got %v", cfg.ReconnectInitialDelay)
		}
		if cfg.ReconnectMaxDelay != 60*time.Second {
			t.Fatalf("max reconnect=60s, got %v", cfg.ReconnectMaxDelay)
		}
	})

	// === Boundary values ===
	t.Run("boundary: zero/negative option values ignored", func(t *testing.T) {
		cfg := NewConsumerConnConfig(testURL,
			WithBackoffBase(0),
			WithConsumerReconnectInitialDelay(-1*time.Second),
			WithConsumerReconnectMaxDelay(0),
		)
		if cfg.ConsumerBackoffBase != 1*time.Second {
			t.Fatalf("zero backoff should keep default=1s, got %v", cfg.ConsumerBackoffBase)
		}
		if cfg.ReconnectInitialDelay != 500*time.Millisecond {
			t.Fatalf("negative reconnect should keep default=500ms, got %v", cfg.ReconnectInitialDelay)
		}
		if cfg.ReconnectMaxDelay != 30*time.Second {
			t.Fatalf("zero max reconnect should keep default=30s, got %v", cfg.ReconnectMaxDelay)
		}
	})

	// === Expected errors ===
	t.Run("error: empty URL panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic for empty URL")
			}
		}()
		NewConsumerConnConfig("")
	})
}

// ============================================================
// nopLogger (no-op, but verify no panic)
// ============================================================

func TestNopLogger(t *testing.T) {
	t.Run("happy: all methods are no-op", func(t *testing.T) {
		l := &nopLogger{}
		ctx := context.Background()
		l.Info(ctx, "test info")
		l.Warn(ctx, "test warn")
		l.Error(ctx, "test error")
		// Should not panic
	})
}

// ============================================================
// WithProducerLogger / WithConsumerLogger
// ============================================================

// mockLogger for testing
type mockLogger struct {
	infos  []string
	warns  []string
	errors []string
}

func (m *mockLogger) Info(_ context.Context, msg string, _ ...zap.Field) {
	m.infos = append(m.infos, msg)
}
func (m *mockLogger) Warn(_ context.Context, msg string, _ ...zap.Field) {
	m.warns = append(m.warns, msg)
}
func (m *mockLogger) Error(_ context.Context, msg string, _ ...zap.Field) {
	m.errors = append(m.errors, msg)
}

func TestLoggerOptions(t *testing.T) {
	// === Happy path ===
	t.Run("happy: producer logger set", func(t *testing.T) {
		mockLog := &mockLogger{}
		cfg := NewProducerConnConfig(testURL, WithProducerLogger(mockLog))
		if cfg.Logger == nil {
			t.Fatal("expected logger to be set")
		}
		// Verify it works by calling the logger
		cfg.Logger.Info(context.Background(), "hello")
		if len(mockLog.infos) != 1 || mockLog.infos[0] != "hello" {
			t.Fatal("expected logger to record hello")
		}
	})

	t.Run("happy: consumer logger set", func(t *testing.T) {
		mockLog := &mockLogger{}
		cfg := NewConsumerConnConfig(testURL, WithConsumerLogger(mockLog))
		if cfg.Logger == nil {
			t.Fatal("expected logger to be set")
		}
		cfg.Logger.Warn(context.Background(), "warn-msg")
		if len(mockLog.warns) != 1 || mockLog.warns[0] != "warn-msg" {
			t.Fatal("expected logger to record warn-msg")
		}
	})

	// === Boundary values ===
	t.Run("boundary: nil logger option keeps default", func(t *testing.T) {
		cfg := NewProducerConnConfig(testURL, WithProducerLogger(nil))
		if cfg.Logger == nil {
			t.Fatal("nil logger option should keep nopLogger")
		}
	})

	t.Run("boundary: nil consumer logger option keeps default", func(t *testing.T) {
		cfg := NewConsumerConnConfig(testURL, WithConsumerLogger(nil))
		if cfg.Logger == nil {
			t.Fatal("nil consumer logger option should keep nopLogger")
		}
	})
}
