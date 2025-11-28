package xtrace

import (
	"context"
	"os"
	"snowgo/pkg/xauth"
	"time"

	"snowgo/pkg/xlogger"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

// InitTracer 初始化 OTLP Tracer，失败时退化为 noop，保证不 panic
func InitTracer(serviceName, serviceVersion, env, tempoAddr string) func(context.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if tempoAddr == "" {
		xlogger.Fatalf("tempo endpoint is empty, please set cfg.Application.TempoEndpoint")
	}

	// 创建 exporter
	exp, err := otlptrace.New(ctx,
		otlptracegrpc.NewClient(
			otlptracegrpc.WithEndpoint(tempoAddr),
			otlptracegrpc.WithInsecure(),
		),
	)
	if err != nil {
		xlogger.Errorf("[otlp] create otlp exporter failed, fallback to noop: %v", err)
		otel.SetTextMapPropagator(
			propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}),
		)
		otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
			if err != nil {
				xlogger.Errorf("[otlp] otel internal error (noop): %v", err)
			}
		}))
		tp := sdktrace.NewTracerProvider()
		otel.SetTracerProvider(tp)
		return func(context.Context) error { return nil }
	}

	// 合并资源
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
			semconv.ServiceVersionKey.String(serviceVersion),
			semconv.ProcessPIDKey.Int(os.Getpid()),
			attribute.String("deployment.environment", env),
		),
	)
	if err != nil {
		xlogger.Errorf("[otlp] merge resource failed: %v, fallback to default", err)
		res = resource.Default()
	}

	// TracerProvider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp,
			sdktrace.WithBatchTimeout(2*time.Second),
			sdktrace.WithMaxExportBatchSize(512),
		),
		sdktrace.WithResource(res),
		// 10% 采样（如果链路过多，进行部分采样）
		//sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(0.1))),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}),
	)
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		if err != nil {
			xlogger.Errorf("[otlp] otel internal error: %v", err)
		}
	}))

	xlogger.Infof("[otlp] otel tracer initialized for service=%s version=%s env=%s endpoint=%s",
		serviceName, serviceVersion, env, tempoAddr)

	return func(ctx context.Context) error {
		c, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := tp.Shutdown(c); err != nil {
			xlogger.Errorf("[otlp] otel tracer shutdown error: %v", err)
			return err
		}
		xlogger.Info("[otlp] otel tracer shutdown succeeded")
		return nil
	}
}

// GetTraceID 只读 context，不生成 ID
func GetTraceID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if tid, ok := ctx.Value(xauth.XTraceId).(string); ok {
		return tid
	}
	return ""
}

// NewContextWithTrace 注入 trace_id 到 context
func NewContextWithTrace(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, xauth.XTraceId, traceID)
}
