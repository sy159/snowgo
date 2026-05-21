package xlogger

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"snowgo/pkg/xtrace"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestPrefixWriter_Write(t *testing.T) {
	t.Run("happy: single line with prefix", func(t *testing.T) {
		buf := &bytes.Buffer{}
		pw := &prefixWriter{w: buf, prefix: "[PREFIX] "}
		n, err := pw.Write([]byte("hello"))
		if err != nil {
			t.Fatalf("Write error: %v", err)
		}
		if n != 5 {
			t.Fatalf("Write returned n=%d, want 5", n)
		}
		if buf.String() != "[PREFIX] hello" {
			t.Fatalf("got %q, want %q", buf.String(), "[PREFIX] hello")
		}
	})

	t.Run("boundary: multi-line with prefix", func(t *testing.T) {
		buf := &bytes.Buffer{}
		pw := &prefixWriter{w: buf, prefix: "> "}
		n, err := pw.Write([]byte("line1\nline2\nline3"))
		if err != nil {
			t.Fatalf("Write error: %v", err)
		}
		if n != 17 {
			t.Fatalf("Write returned n=%d, want 17", n)
		}
		lines := strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n")
		if len(lines) != 3 {
			t.Fatalf("expected 3 lines, got %d", len(lines))
		}
		for i, line := range lines {
			if !strings.HasPrefix(line, "> ") {
				t.Fatalf("line %d missing prefix: %q", i, line)
			}
		}
	})

	t.Run("boundary: empty input", func(t *testing.T) {
		buf := &bytes.Buffer{}
		pw := &prefixWriter{w: buf, prefix: "P:"}
		n, err := pw.Write([]byte(""))
		if err != nil {
			t.Fatalf("Write error: %v", err)
		}
		if n != 0 {
			t.Fatalf("Write returned n=%d, want 0", n)
		}
	})

	t.Run("boundary: trailing newline", func(t *testing.T) {
		buf := &bytes.Buffer{}
		pw := &prefixWriter{w: buf, prefix: "[LOG] "}
		n, err := pw.Write([]byte("hello\n"))
		if err != nil {
			t.Fatalf("Write error: %v", err)
		}
		if n != 6 {
			t.Fatalf("Write returned n=%d, want 6", n)
		}
		if !strings.Contains(buf.String(), "[LOG] hello") {
			t.Fatalf("missing prefix in output: %q", buf.String())
		}
	})

	t.Run("boundary: empty prefix", func(t *testing.T) {
		buf := &bytes.Buffer{}
		pw := &prefixWriter{w: buf, prefix: ""}
		_, err := pw.Write([]byte("hello"))
		if err != nil {
			t.Fatalf("Write error: %v", err)
		}
		if buf.String() != "hello" {
			t.Fatalf("got %q, want %q", buf.String(), "hello")
		}
	})
}

func TestGetNormalEncoder(t *testing.T) {
	enc := getNormalEncoder()
	if enc == nil {
		t.Fatal("getNormalEncoder returned nil")
	}
	buf := &bytes.Buffer{}
	core := zapcore.NewCore(enc, zapcore.AddSync(buf), zapcore.DebugLevel)
	logger := zap.New(core)
	logger.Info("test message", zap.String("key", "val"))
	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Fatalf("expected 'test message' in output: %q", output)
	}
	if !strings.Contains(output, "INFO") {
		t.Fatalf("expected 'INFO' level in output: %q", output)
	}
}

func TestGetJsonEncoder(t *testing.T) {
	enc := getJsonEncoder()
	if enc == nil {
		t.Fatal("getJsonEncoder returned nil")
	}
	buf := &bytes.Buffer{}
	core := zapcore.NewCore(enc, zapcore.AddSync(buf), zapcore.DebugLevel)
	logger := zap.New(core)
	logger.Info("json test", zap.Int("code", 200))
	output := buf.String()
	if !strings.Contains(output, "json test") {
		t.Fatalf("expected 'json test' in output: %q", output)
	}
	if !strings.Contains(output, `"code":200`) {
		t.Fatalf("expected code=200 in output: %q", output)
	}
}

func TestGetAccessNormalEncoder(t *testing.T) {
	enc := getAccessNormalEncoder()
	if enc == nil {
		t.Fatal("getAccessNormalEncoder returned nil")
	}
	buf := &bytes.Buffer{}
	core := zapcore.NewCore(enc, zapcore.AddSync(buf), zapcore.WarnLevel)
	logger := zap.New(core)
	logger.Warn("access log entry")
	output := buf.String()
	if !strings.Contains(output, "access log entry") {
		t.Fatalf("expected 'access log entry' in output: %q", output)
	}
}

func TestGetAccessJsonEncoder(t *testing.T) {
	enc := getAccessJsonEncoder()
	if enc == nil {
		t.Fatal("getAccessJsonEncoder returned nil")
	}
	buf := &bytes.Buffer{}
	core := zapcore.NewCore(enc, zapcore.AddSync(buf), zapcore.WarnLevel)
	logger := zap.New(core)
	logger.Warn("access json entry")
	output := buf.String()
	if !strings.Contains(output, "access json entry") {
		t.Fatalf("expected 'access json entry' in output: %q", output)
	}
}

func TestGetTraceField_Happy(t *testing.T) {
	t.Run("happy: context with trace ID", func(t *testing.T) {
		ctx := xtrace.NewContextWithTrace(context.Background(), "my-trace-id")
		f := getTraceField(ctx)
		if f == zap.Skip() {
			t.Fatal("expected non-skip field for context with trace ID")
		}
	})

	t.Run("boundary: context with empty trace ID", func(t *testing.T) {
		ctx := xtrace.NewContextWithTrace(context.Background(), "")
		f := getTraceField(ctx)
		if f != zap.Skip() {
			t.Fatal("expected Skip for empty trace ID")
		}
	})
}

func TestMergeFieldsWithTrace_Happy(t *testing.T) {
	t.Run("happy: with trace ID and extra fields", func(t *testing.T) {
		ctx := xtrace.NewContextWithTrace(context.Background(), "trace-456")
		extra := []zap.Field{zap.String("user", "alice"), zap.Int("count", 5)}
		result := mergeFieldsWithTrace(ctx, extra)
		if len(result) != 3 {
			t.Fatalf("expected 3 fields (trace + 2 extra), got %d", len(result))
		}
	})
}

func TestLoggerFunctions_NoPanic(t *testing.T) {
	// These functions call the global zap logger (noop if uninitialized).
	// They should not panic even without Init().
	t.Run("basic log functions", func(t *testing.T) {
		Debug("test debug", zap.String("k", "v"))
		Debugf("test debugf %d", 42)
		Info("test info", zap.String("k", "v"))
		Infof("test infof %d", 42)
		Error("test error", zap.String("k", "v"))
		Errorf("test errorf %d", 42)
		Access("test access", zap.String("k", "v"))
	})

	t.Run("context log functions", func(t *testing.T) {
		ctx := xtrace.NewContextWithTrace(context.Background(), "trace-789")
		InfofCtx(ctx, "test infofctx %d", 42)
		ErrorfCtx(ctx, "test errorfctx %d", 42)
		InfoCtx(ctx, "test infoctx", zap.String("k", "v"))
		ErrorCtx(ctx, "test errorctx", zap.String("k", "v"))
	})

	t.Run("context log functions with nil ctx", func(t *testing.T) {
		InfofCtx(context.Background(), "test bg %d", 42)
		InfoCtx(context.Background(), "test bg", zap.String("k", "v"))
	})
}

func TestGetTimeWriter_Boundary(t *testing.T) {
	t.Run("boundary: maxAge >= 30 years is capped", func(t *testing.T) {
		ws := getTimeWriter(t.TempDir()+"/test.log", 365*30+1)
		if ws == nil {
			t.Fatal("getTimeWriter should not return nil")
		}
	})
}
