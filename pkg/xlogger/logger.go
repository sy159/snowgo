package xlogger

import (
	"context"
	"fmt"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"io"
	"os"
	"path/filepath"
	"snowgo/config"
	"snowgo/pkg/xcolor"
	"snowgo/pkg/xtrace"
	"strings"
	"time"
	//"gopkg.in/natefinch/lumberjack.v2"
)

const (
	logTmFmtWithMS = "2006-01-02 15:04:05.000"
	logTmFmtWithS  = "2006-01-02 15:04:05"
	logTmFmtWithM  = "2006-01-02 15:04"
	logTmFmtWithH  = "2006-01-02 15"
)

var (
	defaultBasePath = "./logs"
	FileWriter      = "file"
	MultiWriter     = "multi"
)

//var Logger *zap.Logger // 也直接zap.L或者zap.S调用获取实例

type prefixWriter struct {
	w      io.Writer
	prefix string
}

// 返回 len(b) 而非底层写入的字节数，遵守 io.Writer 契约（0 <= n <= len(b)）
func (p *prefixWriter) Write(b []byte) (int, error) {
	lines := strings.Split(string(b), "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = p.prefix + line
		}
	}
	_, err := p.w.Write([]byte(strings.Join(lines, "\n")))
	return len(b), err
}

// Init 初始化Logger,设置zap全局logger
func Init(basePath string) {
	if len(basePath) == 0 {
		basePath = defaultBasePath
	}
	cfg := config.Get()
	logEncoder := cfg.Log.LogEncoder
	accountEncoderConf := cfg.Log.AccessEncoder
	logMaxAge := cfg.Log.LogFileMaxAgeDays
	accountMaxAge := cfg.Log.AccessFileMaxAgeDays
	writer := cfg.Log.Output

	// 设置日志输出格式
	encoder := getNormalEncoder()
	if logEncoder == "json" {
		encoder = getJsonEncoder()
	}

	accountEncoder := getAccessNormalEncoder()
	if accountEncoderConf == "json" {
		accountEncoder = getAccessJsonEncoder()
	}

	// 实现两个判断日志等级的interface
	debugLevel := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
		return level == zapcore.DebugLevel
	})
	infoLevel := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
		return level == zapcore.InfoLevel
	})
	// warn警告用于记录访问日志
	accessLevel := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
		return level == zapcore.WarnLevel
	})
	errorLevel := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
		return level >= zapcore.ErrorLevel
	})

	// 根据时间进行分割
	// 统一一个服务名前缀
	stdoutWithPrefix := zapcore.AddSync(&prefixWriter{
		w: os.Stdout,
		prefix: xcolor.GreenFont(fmt.Sprintf("[%s:%s] ", cfg.Application.Server.Name,
			cfg.Application.Server.Version)) + xcolor.YellowFont("[logger] | "),
	})
	debugWriter := stdoutWithPrefix
	infoWriter := stdoutWithPrefix
	accessWriter := stdoutWithPrefix
	errorWriter := stdoutWithPrefix
	// console控制台输出，file输出到文件 multi控制台跟日志文件同时输出
	switch writer {
	case FileWriter:
		debugWriter = getTimeWriter(filepath.Join(basePath, "debug/debug.log"), logMaxAge)
		infoWriter = getTimeWriter(filepath.Join(basePath, "info/info.log"), logMaxAge)
		accessWriter = getTimeWriter(filepath.Join(basePath, "access/access.log"), accountMaxAge)
		errorWriter = getTimeWriter(filepath.Join(basePath, "error/error.log"), logMaxAge)

		// 日志大小分割
		//debugWriter := getSizeWriter("./logs/debug/debug.log")
		//infoWriter := getSizeWriter("./logs/info/info.log")
		//accessWriter := getSizeWriter("./logs/access/access.log")
		//errorWriter := getSizeWriter("./logs/error/error.log")
	case MultiWriter:
		debugWriter = getTimeWriter(filepath.Join(basePath, "debug/debug.log"), logMaxAge)
		infoWriter = getTimeWriter(filepath.Join(basePath, "info/info.log"), logMaxAge)
		accessWriter = getTimeWriter(filepath.Join(basePath, "access/access.log"), accountMaxAge)
		errorWriter = getTimeWriter(filepath.Join(basePath, "xerror/xerror.log"), logMaxAge)

		// 日志大小分割
		//debugWriter := getSizeWriter("./logs/debug/debug.log")
		//infoWriter := getSizeWriter("./logs/info/info.log")
		//accessWriter := getSizeWriter("./logs/access/access.log")
		//errorWriter := getSizeWriter("./logs/xerror/xerror.log")
		debugWriter = zapcore.NewMultiWriteSyncer(stdoutWithPrefix, debugWriter)
		infoWriter = zapcore.NewMultiWriteSyncer(stdoutWithPrefix, infoWriter)
		//accessWriter = zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), accessWriter)  // account其他地方有些
		errorWriter = zapcore.NewMultiWriteSyncer(stdoutWithPrefix, errorWriter)
	}

	// 创建具体的Logger
	core := zapcore.NewTee(
		// zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), zap.NewAtomicLevelAt(zapcore.DebugLevel)),
		zapcore.NewCore(encoder, debugWriter, debugLevel),
		zapcore.NewCore(encoder, infoWriter, infoLevel),
		zapcore.NewCore(accountEncoder, accessWriter, accessLevel),
		zapcore.NewCore(encoder, errorWriter, errorLevel),
	)

	// 构造日志
	Logger := zap.New(
		core,
		zap.AddCaller(),                       // 显示打日志点的文件名和行数
		zap.AddCallerSkip(1),                  // 输出第几层调用者位置，如果不封装日志，修改为0
		zap.AddStacktrace(zapcore.ErrorLevel), // Error及以上级别的日志输出堆栈
		//zap.Development(),                     // 开启开发模式，堆栈跟踪
	)

	// 将自定义的logger替换zap包中全局的logger实例，后续在其他包中只需使用zap.L()调用即可,zap.S()获取的是Sugar()
	zap.ReplaceGlobals(Logger)

}

// 普通编码器配置
func getNormalEncoder() zapcore.Encoder {
	return zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
		TimeKey:    "time",
		LevelKey:   "level",
		NameKey:    "logger",
		CallerKey:  "caller",
		MessageKey: "msg",
		//StacktraceKey: "stacktrace",
		LineEnding: zapcore.DefaultLineEnding,
		//EncodeLevel: 带颜色输出 CapitalColorLevelEncoder
		EncodeLevel: func(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString("[" + level.CapitalString() + "]")
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

// json编码器配置
func getJsonEncoder() zapcore.Encoder {
	return zapcore.NewJSONEncoder(zapcore.EncoderConfig{
		TimeKey:    "time",
		LevelKey:   "level",
		NameKey:    "logger",
		CallerKey:  "caller",
		MessageKey: "msg",
		//StacktraceKey: "stacktrace",
		LineEnding: zapcore.DefaultLineEnding,
		//EncodeLevel: 带颜色输出 CapitalColorLevelEncoder
		EncodeLevel: func(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(level.CapitalString())
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

// 用于记录访问日志(不记录调用地方跟level)
func getAccessNormalEncoder() zapcore.Encoder {
	return zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
		TimeKey:    "time",
		NameKey:    "logger",
		MessageKey: "msg",
		//StacktraceKey: "stacktrace",
		LineEnding: zapcore.DefaultLineEnding,
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

// 用于记录访问日志 json(不记录调用地方跟level)
func getAccessJsonEncoder() zapcore.Encoder {
	return zapcore.NewJSONEncoder(zapcore.EncoderConfig{
		TimeKey:    "time",
		NameKey:    "logger",
		MessageKey: "msg",
		//StacktraceKey: "stacktrace",
		LineEnding: zapcore.DefaultLineEnding,
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

// getSizeWriter 根据日志大小分割写入文件
//func getSizeWriter(filename string) zapcore.WriteSyncer {
//	hook := &lumberjack.Logger{
//		Filename:   filename,
//		MaxSize:    5,  // 每个日志文件保存的最大尺寸 单位：M
//		MaxBackups: 10, // 保留的最大旧日志文件数,默认是保留所有旧的日志文件(MaxAge可能仍会导致它们被删除)
//		//MaxAge:     30,    // 根据文件名中编码的时间戳保留旧日志文件的最大天数和MaxBackups参数配置1个就可以
//		Compress:  true, // 是否压缩
//		LocalTime: true, // LocalTime 确定用于格式化备份文件中的时间戳的时间是否是计算机的本地时间。默认是使用 UTC 时间。
//	}
//	return zapcore.AddSync(hook)
//}

// getTimeWriter 根据日志时间分割写入文件
func getTimeWriter(filename string, maxAgeDay uint32) zapcore.WriteSyncer {
	// 文件最多保留30年
	if maxAgeDay >= 365*30 {
		maxAgeDay = 365 * 30
	}
	hook, err := rotatelogs.New(
		strings.TrimSuffix(filename, ".log")+"-%Y-%m-%d.log",         // 只去除尾部 .log
		rotatelogs.WithLinkName(filename),                            // 生成软链，指向最新日志文件
		rotatelogs.WithMaxAge(time.Duration(maxAgeDay)*24*time.Hour), // 文件最多保留时间
		rotatelogs.WithRotationTime(24*time.Hour),                    // 文件分割间隔
		rotatelogs.WithLocation(time.Local),
	)

	if err != nil {
		// 降级到 stdout，避免 panic 导致服务无法启动
		zap.L().Error("create rotate log writer failed, fallback to stdout",
			zap.String("filename", filename),
			zap.Error(err),
		)
		return zapcore.AddSync(os.Stdout)
	}
	return zapcore.AddSync(hook)
}

// Debug Logger.Debug
func Debug(msg string, fields ...zap.Field) {
	zap.L().Debug(msg, fields...)
}

// Debugf SugaredLogger.Debugf uses fmt.Sprintf to log a templated message.
func Debugf(template string, args ...any) {
	zap.S().Debugf(template, args...)
}

// Info Logger.Info
func Info(msg string, fields ...zap.Field) {
	zap.L().Info(msg, fields...)
}

// Infof SugaredLogger.Infof uses fmt.Sprintf to log a templated message.
func Infof(template string, args ...any) {
	zap.S().Infof(template, args...)
}

// Error Logger.xerror
func Error(msg string, fields ...zap.Field) {
	zap.L().Error(msg, fields...)
}

// Errorf SugaredLogger.Errorf uses fmt.Sprintf to log a templated message.
func Errorf(template string, args ...any) {
	zap.S().Errorf(template, args...)
}

// Panic Logger.Panic 日志输出并调用panic
func Panic(msg string, fields ...zap.Field) {
	zap.L().Panic(msg, fields...)
}

// Panicf SugaredLogger.Panicf uses fmt.Sprintf to log a templated message.
func Panicf(template string, args ...any) {
	zap.S().Panicf(template, args...)
}

// Fatal Logger.Fatal 日志输出并调用exit直接退出
func Fatal(msg string, fields ...zap.Field) {
	zap.L().Fatal(msg, fields...)
}

// Fatalf SugaredLogger.Fatalf uses fmt.Sprintf to log a templated message.
func Fatalf(template string, args ...any) {
	zap.S().Fatalf(template, args...)
}

// Access 写入访问日志
func Access(msg string, fields ...zap.Field) {
	zap.L().Warn(msg, fields...)
}

// getTraceField 从 ctx 读取 trace_id，返回 zap.Field
func getTraceField(ctx context.Context) zap.Field {
	if ctx == nil {
		return zap.Skip()
	}
	// trace信息
	if tid := xtrace.GetTraceID(ctx); tid != "" {
		return zap.String("trace_id", tid)
	}
	return zap.Skip()
}

// getTraceAndMergeFields 将 trace 字段放到前端，然后把 user 提供的 fields 接上
func mergeFieldsWithTrace(ctx context.Context, fields []zap.Field) []zap.Field {
	tf := getTraceField(ctx)
	if tf == zap.Skip() {
		return fields
	}
	out := make([]zap.Field, 0, len(fields)+1)
	out = append(out, tf)
	out = append(out, fields...)
	return out
}

// InfofCtx 使用 fmt 风格的 Info（接受 format + args），内部自动把 trace 加入
func InfofCtx(ctx context.Context, template string, args ...any) {
	zap.L().With(getTraceField(ctx)).Sugar().Infof(template, args...)
}

// ErrorfCtx 使用 fmt 风格的 Error（接受 format + args）
func ErrorfCtx(ctx context.Context, template string, args ...any) {
	zap.L().With(getTraceField(ctx)).Sugar().Errorf(template, args...)
}

// InfoCtx 使用 zap.Field 风格的 Info（适合结构化日志）
func InfoCtx(ctx context.Context, msg string, fields ...zap.Field) {
	all := mergeFieldsWithTrace(ctx, fields)
	zap.L().With(all...).Info(msg)
}

// ErrorCtx 使用 zap.Field 风格的 Error（适合结构化日志）
func ErrorCtx(ctx context.Context, msg string, fields ...zap.Field) {
	all := mergeFieldsWithTrace(ctx, fields)
	zap.L().With(all...).Error(msg)
}
