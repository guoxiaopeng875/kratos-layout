package tracing

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	kafkago "github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/go-kratos/kratos-layout/internal/conf"
)

const (
	defaultExportTimeout = 10 * time.Second
	shutdownTimeout      = 5 * time.Second
)

// Tracer is the package-level OTel tracer, initialized by Setup.
// It is safe to call before Setup — the zero-value trace.Tracer is a no-op.
var Tracer trace.Tracer = noop.NewTracerProvider().Tracer("")

// Setup installs a global OpenTelemetry TracerProvider and TextMapPropagator.
// If cfg is nil or no endpoint is configured, tracing is left disabled and a
// no-op cleanup is returned.
func Setup(ctx context.Context, cfg *conf.Tracing, serviceName string, logger log.Logger) (func(context.Context), error) {
	if cfg == nil || !hasEndpoint(cfg) {
		return func(context.Context) {}, nil
	}

	exporter, err := newExporter(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("tracing: create OTLP exporter: %w", err)
	}

	reportedName := serviceName
	if override := cfg.GetServiceName(); override != "" {
		reportedName = override
	}
	res, err := resource.New(ctx,
		resource.WithSchemaURL(semconv.SchemaURL),
		resource.WithAttributes(semconv.ServiceNameKey.String(reportedName)),
	)
	if err != nil {
		return nil, fmt.Errorf("tracing: build resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sampler(cfg)),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	Tracer = tp.Tracer(serviceName)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	helper := log.NewHelper(log.With(logger, "module", "tracing"))
	cleanup := func(ctx context.Context) {
		shutdownCtx, cancel := context.WithTimeout(ctx, shutdownTimeout)
		defer cancel()
		if err := tp.Shutdown(shutdownCtx); err != nil {
			helper.Warnf("tracer provider shutdown: %v", err)
		}
	}
	return cleanup, nil
}

func hasEndpoint(cfg *conf.Tracing) bool {
	switch cfg.Endpoint.(type) {
	case *conf.Tracing_HttpEndpoint:
		return cfg.GetHttpEndpoint() != ""
	case *conf.Tracing_HttpEndpointUrl:
		return cfg.GetHttpEndpointUrl() != ""
	}
	return false
}

func newExporter(ctx context.Context, cfg *conf.Tracing) (*otlptrace.Exporter, error) {
	timeout := defaultExportTimeout
	if cfg.Timeout != nil && cfg.Timeout.AsDuration() > 0 {
		timeout = cfg.Timeout.AsDuration()
	}

	opts := []otlptracehttp.Option{otlptracehttp.WithTimeout(timeout)}
	switch cfg.Endpoint.(type) {
	case *conf.Tracing_HttpEndpoint:
		opts = append(opts, otlptracehttp.WithEndpoint(cfg.GetHttpEndpoint()))
	case *conf.Tracing_HttpEndpointUrl:
		opts = append(opts, otlptracehttp.WithEndpointURL(cfg.GetHttpEndpointUrl()))
	}
	if cfg.Insecure != nil && *cfg.Insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	return otlptrace.New(ctx, otlptracehttp.NewClient(opts...))
}

func sampler(cfg *conf.Tracing) sdktrace.Sampler {
	if cfg.SampleRatio == nil {
		return sdktrace.AlwaysSample()
	}
	return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(float64(*cfg.SampleRatio)))
}

// KafkaHeaderCarrier adapts *[]kafkago.Header to propagation.TextMapCarrier
// so that OTel propagators can inject trace context into Kafka message headers.
type KafkaHeaderCarrier struct {
	headers *[]kafkago.Header
}

func NewKafkaHeaderCarrier(headers *[]kafkago.Header) KafkaHeaderCarrier {
	return KafkaHeaderCarrier{headers: headers}
}

func (c KafkaHeaderCarrier) Get(key string) string {
	for _, h := range *c.headers {
		if h.Key == key {
			return string(h.Value)
		}
	}
	return ""
}

func (c KafkaHeaderCarrier) Set(key, value string) {
	for i, h := range *c.headers {
		if h.Key == key {
			(*c.headers)[i].Value = []byte(value)
			return
		}
	}
	*c.headers = append(*c.headers, kafkago.Header{Key: key, Value: []byte(value)})
}

func (c KafkaHeaderCarrier) Keys() []string {
	keys := make([]string, len(*c.headers))
	for i, h := range *c.headers {
		keys[i] = h.Key
	}
	return keys
}
