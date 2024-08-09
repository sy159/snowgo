package logger

import (
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"time"
)

type MiddlewareLogger struct {
	logger *zap.Logger
}

var defaultLog = logOptions{
	enableConsoleOutput: true,
	fileMaxAgeDays:      7,
}

type LogOptions func(*logOptions)

type logOptions struct {
	enableConsoleOutput bool
	fileMaxAgeDays      uint
}

// WithConsoleOutput 选择是否启用控制台输出
func WithConsoleOutput(enabled bool) LogOptions {
	return func(l *logOptions) {
		l.enableConsoleOutput = enabled
	}
}

// WithFileMaxAgeDays 设置日志保存的天数
func WithFileMaxAgeDays(days uint) LogOptions {
	return func(l *logOptions) {
		if days > 2 {
			l.fileMaxAgeDays = days
		}
	}
}

// NewLogger 创建JSON格式的日志记录器
func NewLogger(name string, opts ...LogOptions) *MiddlewareLogger {

	logOpts := defaultLog
	for _, opt := range opts {
		opt(&logOpts)
	}
	if logOpts.fileMaxAgeDays < 2 {
		logOpts.fileMaxAgeDays = 2
	}

	encoder := getMiddlewareJsonEncoder()

	infoLevel := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
		return level == zapcore.InfoLevel
	})

	infoWriter := getTimeWriter(fmt.Sprintf("./logs/%s/%s.log", name, name), logOpts.fileMaxAgeDays)

	if logOpts.enableConsoleOutput {
		infoWriter = zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), infoWriter)
	}

	core := zapcore.NewTee(
		zapcore.NewCore(encoder, infoWriter, infoLevel),
	)

	zapLogger := zap.New(
		core,
		zap.AddCaller(),
		zap.AddCallerSkip(1),
	)

	return &MiddlewareLogger{logger: zapLogger}
}

// json编码器配置
func getMiddlewareJsonEncoder() zapcore.Encoder {
	return zapcore.NewJSONEncoder(zapcore.EncoderConfig{
		TimeKey:    "log_timestamp",
		NameKey:    "logger",
		CallerKey:  "log_caller",
		MessageKey: "log_msg",
		LineEnding: zapcore.DefaultLineEnding,
		EncodeLevel: func(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
		},
		// 时间格式 zapcore.ISO8601TimeEncoder
		EncodeTime: func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(t.Format(logTmFmtWithMS))
		},

		// 时间序列化， 经过的时间(毫秒)；  zapcore.SecondsDurationEncoder s
		EncodeDuration: func(d time.Duration, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendInt64(int64(d) / int64(time.Millisecond))
		},
		EncodeCaller: zapcore.ShortCallerEncoder, // 短路径编码器(相对路径+行号)
		EncodeName:   zapcore.FullNameEncoder,
	})
}

// Log 记录信息日志
func (l *MiddlewareLogger) Log(msg string, fields ...zap.Field) {
	l.logger.Info(msg, fields...)
}

// Sync 确保日志写入文件或控制台
func (l *MiddlewareLogger) Sync() {
	_ = l.logger.Sync()
}
