package observability

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TracingConfig represents tracing configuration
type TracingConfig struct {
	// Enabled controls whether tracing is enabled
	Enabled bool `json:"enabled" yaml:"enabled"`

	// ServiceName is the name of the service
	ServiceName string `json:"service_name" yaml:"service_name"`

	// ServiceVersion is the version of the service
	ServiceVersion string `json:"service_version" yaml:"service_version"`

	// Environment is the deployment environment (dev, staging, prod)
	Environment string `json:"environment" yaml:"environment"`

	// Endpoint is the OTLP collector endpoint
	Endpoint string `json:"endpoint" yaml:"endpoint"`

	// SampleRate is the sampling rate (0.0 to 1.0)
	SampleRate float64 `json:"sample_rate" yaml:"sample_rate"`

	// Insecure disables TLS for OTLP exporter
	Insecure bool `json:"insecure" yaml:"insecure"`
}

// DefaultTracingConfig returns default tracing configuration
func DefaultTracingConfig() *TracingConfig {
	return &TracingConfig{
		Enabled:        false,
		ServiceName:    "lurus-switch",
		ServiceVersion: "1.0.0",
		Environment:    "development",
		Endpoint:       "localhost:4317",
		SampleRate:     1.0, // 100% sampling by default
		Insecure:       true,
	}
}

// TracerProvider wraps OpenTelemetry tracer provider
type TracerProvider struct {
	provider *sdktrace.TracerProvider
	config   *TracingConfig
}

// InitTracing initializes OpenTelemetry tracing
func InitTracing(ctx context.Context, cfg *TracingConfig) (*TracerProvider, error) {
	if cfg == nil {
		cfg = DefaultTracingConfig()
	}

	if !cfg.Enabled {
		// Return noop provider
		return &TracerProvider{config: cfg}, nil
	}

	// Create OTLP exporter
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.Endpoint),
	}

	if cfg.Insecure {
		opts = append(opts, otlptracegrpc.WithDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
		opts = append(opts, otlptracegrpc.WithInsecure())
	}

	exporter, err := otlptracegrpc.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Create resource
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			semconv.DeploymentEnvironment(cfg.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create tracer provider
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(cfg.SampleRate)),
	)

	// Set global tracer provider
	otel.SetTracerProvider(provider)

	// Set global propagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return &TracerProvider{
		provider: provider,
		config:   cfg,
	}, nil
}

// Shutdown shuts down the tracer provider
func (tp *TracerProvider) Shutdown(ctx context.Context) error {
	if tp.provider != nil {
		return tp.provider.Shutdown(ctx)
	}
	return nil
}

// Tracer returns a tracer for the given name
func (tp *TracerProvider) Tracer(name string) trace.Tracer {
	if tp.provider != nil {
		return tp.provider.Tracer(name)
	}
	return otel.Tracer(name)
}

// SpanFromContext returns the current span from context
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// StartSpan starts a new span
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return otel.Tracer("").Start(ctx, name, opts...)
}

// AddEvent adds an event to the current span
func AddEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.AddEvent(name, trace.WithAttributes(attrs...))
}

// SetAttributes sets attributes on the current span
func SetAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attrs...)
}

// RecordError records an error on the current span
func RecordError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	span.RecordError(err)
}

// TraceID returns the trace ID from context
func TraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().HasTraceID() {
		return span.SpanContext().TraceID().String()
	}
	return ""
}

// SpanID returns the span ID from context
func SpanID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().HasSpanID() {
		return span.SpanContext().SpanID().String()
	}
	return ""
}
