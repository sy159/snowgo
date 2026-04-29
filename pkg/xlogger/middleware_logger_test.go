package xlogger

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// newBufferedLogger creates a MiddlewareLogger that writes JSON logs to a bytes.Buffer instead of files.
func newBufferedLogger(t *testing.T) (*MiddlewareLogger, *bytes.Buffer) {
	t.Helper()
	buf := &bytes.Buffer{}
	encoder := zapcore.NewJSONEncoder(zapcore.EncoderConfig{
		TimeKey:    "log_timestamp",
		MessageKey: "log_msg",
		LevelKey:   "log_level",
		EncodeLevel: func(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(level.CapitalString())
		},
		EncodeTime: func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
		},
	})
	core := zapcore.NewCore(encoder, zapcore.AddSync(buf), zapcore.InfoLevel)
	zl := zap.New(core)
	return &MiddlewareLogger{logger: zl}, buf
}

func TestMiddlewareLogger_Info(t *testing.T) {
	logger, buf := newBufferedLogger(t)
	ctx := context.Background()

	logger.Info(ctx, "hello test", zap.String("key", "value"))
	logger.Sync()

	output := buf.String()
	if output == "" {
		t.Fatal("expected log output, got empty string")
	}
	if !strings.Contains(output, "hello test") {
		t.Fatalf("expected message 'hello test' in output: %s", output)
	}
	if !strings.Contains(output, "INFO") {
		t.Fatalf("expected INFO level in output: %s", output)
	}
	if !strings.Contains(output, `"key":"value"`) {
		t.Fatalf("expected field key=value in output: %s", output)
	}

	// Verify JSON structure
	var entry map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &entry); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if entry["log_msg"] != "hello test" {
		t.Fatalf("log_msg mismatch: got %v", entry["log_msg"])
	}
	if entry["log_level"] != "INFO" {
		t.Fatalf("log_level mismatch: got %v", entry["log_level"])
	}
}

func TestMiddlewareLogger_Error(t *testing.T) {
	logger, buf := newBufferedLogger(t)
	ctx := context.Background()

	logger.Error(ctx, "something failed", zap.Int("code", 500))
	logger.Sync()

	output := buf.String()
	if !strings.Contains(output, "something failed") {
		t.Fatalf("expected error message in output: %s", output)
	}
	if !strings.Contains(output, "ERROR") {
		t.Fatalf("expected ERROR level in output: %s", output)
	}
}

func TestMiddlewareLogger_Warn(t *testing.T) {
	logger, buf := newBufferedLogger(t)
	ctx := context.Background()

	logger.Warn(ctx, "warning msg", zap.String("topic", "test-topic"))
	logger.Sync()

	output := buf.String()
	if !strings.Contains(output, "warning msg") {
		t.Fatalf("expected warn message in output: %s", output)
	}
	if !strings.Contains(output, "WARN") {
		t.Fatalf("expected WARN level in output: %s", output)
	}
}

func TestMiddlewareLogger_MultipleEntries(t *testing.T) {
	logger, buf := newBufferedLogger(t)
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		logger.Info(ctx, "producer test", zap.String("topic", "test-topic"), zap.Int("msgId", i))
	}
	logger.Sync()

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 10 {
		t.Fatalf("expected 10 log lines, got %d", len(lines))
	}

	for _, line := range lines {
		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("line is not valid JSON: %v", err)
		}
		if entry["log_msg"] != "producer test" {
			t.Fatalf("log_msg mismatch: got %v", entry["log_msg"])
		}
	}
}

func TestMiddlewareLogger_ConsoleOption(t *testing.T) {
	t.Run("with console output", func(t *testing.T) {
		logger := NewLogger(t.TempDir(), "test-console", WithConsoleOutput(true), WithFileMaxAgeDays(3))
		ctx := context.Background()
		logger.Info(ctx, "console test", zap.Int("id", 1))
		logger.Sync()
	})

	t.Run("without console output", func(t *testing.T) {
		logger := NewLogger(t.TempDir(), "test-file", WithConsoleOutput(false), WithFileMaxAgeDays(3))
		ctx := context.Background()
		logger.Info(ctx, "file test", zap.Int("id", 2))
		logger.Sync()
	})
}
