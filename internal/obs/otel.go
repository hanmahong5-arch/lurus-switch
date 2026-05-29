package obs

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// instrumentationName is the tracer / meter scope name. Stable so back-end
// dashboards can group all gateway spans under one instrumentation scope.
const instrumentationName = "lurus-switch/gateway"

// Config controls the OTLP/HTTP exporter wiring. All fields originate in
// appconfig.ObservabilityConfig.
type Config struct {
	ServiceName    string            // resource service.name; "" → "lurus-switch"
	ServiceVersion string            // resource service.version
	Endpoint       string            // OTLP/HTTP endpoint: "host:4318" or "http(s)://host:4318"
	Headers        map[string]string // optional OTLP headers (e.g. auth)
}

// New builds an OTLP/HTTP-backed Recorder plus a shutdown func that flushes
// and tears down the providers. The exporters connect lazily on first
// export, so New does not block on (or fail because of) an unreachable
// collector — a misconfigured endpoint surfaces as background export errors,
// never a startup failure.
func New(cfg Config) (Recorder, func(context.Context) error, error) {
	ctx := context.Background()

	traceExp, err := otlptracehttp.New(ctx, traceHTTPOpts(cfg)...)
	if err != nil {
		return Noop(), nil, fmt.Errorf("otlp trace exporter: %w", err)
	}
	metricExp, err := otlpmetrichttp.New(ctx, metricHTTPOpts(cfg)...)
	if err != nil {
		_ = traceExp.Shutdown(ctx)
		return Noop(), nil, fmt.Errorf("otlp metric exporter: %w", err)
	}

	res := newResource(cfg)
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExp),
		sdktrace.WithResource(res),
	)
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExp)),
		sdkmetric.WithResource(res),
	)

	rec, err := newRecorder(tp, mp)
	if err != nil {
		_ = tp.Shutdown(ctx)
		_ = mp.Shutdown(ctx)
		return Noop(), nil, fmt.Errorf("build recorder instruments: %w", err)
	}

	shutdown := func(ctx context.Context) error {
		return errors.Join(tp.Shutdown(ctx), mp.Shutdown(ctx))
	}
	return rec, shutdown, nil
}

// newResource builds the OTel resource. NewSchemaless avoids coupling the
// build to a specific semconv schema-URL version; the attribute keys
// (service.name / service.version) are the stable semconv names regardless.
func newResource(cfg Config) *resource.Resource {
	name := cfg.ServiceName
	if name == "" {
		name = "lurus-switch"
	}
	return resource.NewSchemaless(
		attribute.String("service.name", name),
		attribute.String("service.version", cfg.ServiceVersion),
	)
}

// traceHTTPOpts / metricHTTPOpts map Config onto the exporter options. A
// bare "host:port" endpoint is treated as a plaintext collector
// (WithInsecure) — the common local-collector case; an explicit scheme is
// honored via WithEndpointURL.
func traceHTTPOpts(cfg Config) []otlptracehttp.Option {
	opts := []otlptracehttp.Option{}
	if ep := strings.TrimSpace(cfg.Endpoint); ep != "" {
		if strings.Contains(ep, "://") {
			opts = append(opts, otlptracehttp.WithEndpointURL(ep))
		} else {
			opts = append(opts, otlptracehttp.WithEndpoint(ep), otlptracehttp.WithInsecure())
		}
	}
	if len(cfg.Headers) > 0 {
		opts = append(opts, otlptracehttp.WithHeaders(cfg.Headers))
	}
	return opts
}

func metricHTTPOpts(cfg Config) []otlpmetrichttp.Option {
	opts := []otlpmetrichttp.Option{}
	if ep := strings.TrimSpace(cfg.Endpoint); ep != "" {
		if strings.Contains(ep, "://") {
			opts = append(opts, otlpmetrichttp.WithEndpointURL(ep))
		} else {
			opts = append(opts, otlpmetrichttp.WithEndpoint(ep), otlpmetrichttp.WithInsecure())
		}
	}
	if len(cfg.Headers) > 0 {
		opts = append(opts, otlpmetrichttp.WithHeaders(cfg.Headers))
	}
	return opts
}

// otelRecorder maps a RequestObservation onto OTel GenAI-semconv spans and
// metrics. Spans are recorded retrospectively (start + end timestamps set
// from the observation) because the gateway calls in after the request has
// already completed and metering is recorded — no live span is threaded
// through the request path.
type otelRecorder struct {
	tracer       trace.Tracer
	tokenUsage   metric.Int64Histogram   // gen_ai.client.token.usage, attr type=input/output
	requestDur   metric.Float64Histogram // gen_ai.client.operation.duration, ms
	requestCount metric.Int64Counter     // gen_ai.client.requests, attr status code
}

// newRecorder builds the recorder from API-level providers. Split out from
// New so tests can inject in-memory SDK providers (tracetest span recorder +
// manual metric reader) without an OTLP exporter / network.
func newRecorder(tp trace.TracerProvider, mp metric.MeterProvider) (*otelRecorder, error) {
	m := mp.Meter(instrumentationName)
	tokenUsage, err := m.Int64Histogram(
		"gen_ai.client.token.usage",
		metric.WithUnit("{token}"),
		metric.WithDescription("Number of input and output tokens used per request"),
	)
	if err != nil {
		return nil, err
	}
	requestDur, err := m.Float64Histogram(
		"gen_ai.client.operation.duration",
		metric.WithUnit("ms"),
		metric.WithDescription("Gateway request latency"),
	)
	if err != nil {
		return nil, err
	}
	requestCount, err := m.Int64Counter(
		"gen_ai.client.requests",
		metric.WithDescription("Count of gateway requests by response status"),
	)
	if err != nil {
		return nil, err
	}
	return &otelRecorder{
		tracer:       tp.Tracer(instrumentationName),
		tokenUsage:   tokenUsage,
		requestDur:   requestDur,
		requestCount: requestCount,
	}, nil
}

func (r *otelRecorder) RecordRequest(ctx context.Context, o RequestObservation) {
	op := o.Operation
	if op == "" {
		op = "chat"
	}

	// Retrospective span: start at the request's recorded StartTime and end
	// it immediately at start+latency, so the back-end shows the true wall
	// span without us holding a live span across the request path.
	start := o.StartTime
	if start.IsZero() {
		// No recorded start (shouldn't happen) — fall back to a zero-width
		// span ending now rather than fabricating a duration.
		_, span := r.tracer.Start(ctx, "gen_ai."+op,
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithAttributes(r.spanAttributes(o)...),
		)
		r.finishSpan(span, o)
		return
	}
	end := start.Add(time.Duration(o.LatencyMs) * time.Millisecond)
	_, span := r.tracer.Start(ctx, "gen_ai."+op,
		trace.WithTimestamp(start),
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(r.spanAttributes(o)...),
	)
	if o.Err != "" {
		span.SetStatus(codes.Error, o.Err)
	}
	span.End(trace.WithTimestamp(end))

	r.recordMetrics(ctx, o)
}

func (r *otelRecorder) finishSpan(span trace.Span, o RequestObservation) {
	if o.Err != "" {
		span.SetStatus(codes.Error, o.Err)
	}
	span.End()
	r.recordMetrics(context.Background(), o)
}

// spanAttributes maps the observation onto OTel GenAI semantic-convention
// attribute keys (gen_ai.*) plus server.address. Keys are spelled out as
// string literals rather than pulled from a semconv package so the build
// doesn't pin a semconv version (the gen_ai constants have churned between
// releases) — the literals ARE the stable convention names.
func (r *otelRecorder) spanAttributes(o RequestObservation) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		attribute.String("gen_ai.operation.name", o.Operation),
		attribute.String("gen_ai.request.model", o.Model),
		attribute.String("gen_ai.response.model", o.Model),
		attribute.Int64("gen_ai.usage.input_tokens", o.TokensIn),
		attribute.Int64("gen_ai.usage.output_tokens", o.TokensOut),
		attribute.Bool("gen_ai.streaming", o.Streaming),
		attribute.Int("http.response.status_code", o.StatusCode),
	}
	if o.ServedBy != "" {
		attrs = append(attrs, attribute.String("server.address", o.ServedBy))
	}
	if o.MatchedBy != "" {
		attrs = append(attrs, attribute.String("lurus.relay.matched_by", o.MatchedBy))
	}
	return attrs
}

// baseMetricAttributes are the dimensions shared by every metric data point.
func (r *otelRecorder) baseMetricAttributes(o RequestObservation) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		attribute.String("gen_ai.operation.name", o.Operation),
		attribute.String("gen_ai.request.model", o.Model),
	}
	if o.ServedBy != "" {
		attrs = append(attrs, attribute.String("server.address", o.ServedBy))
	}
	return attrs
}

func (r *otelRecorder) recordMetrics(ctx context.Context, o RequestObservation) {
	base := r.baseMetricAttributes(o)
	// withExtra allocates a fresh slice per call so the multiple appends
	// below never share base's backing array. metric.WithAttributes copies
	// before sorting today, so reusing base would be safe — but a fresh
	// slice removes the dependency on that copy and the aliasing footgun.
	withExtra := func(extra ...attribute.KeyValue) metric.MeasurementOption {
		out := make([]attribute.KeyValue, 0, len(base)+len(extra))
		out = append(out, base...)
		out = append(out, extra...)
		return metric.WithAttributes(out...)
	}
	if o.TokensIn > 0 {
		r.tokenUsage.Record(ctx, o.TokensIn, withExtra(attribute.String("gen_ai.token.type", "input")))
	}
	if o.TokensOut > 0 {
		r.tokenUsage.Record(ctx, o.TokensOut, withExtra(attribute.String("gen_ai.token.type", "output")))
	}
	r.requestDur.Record(ctx, float64(o.LatencyMs), withExtra())
	r.requestCount.Add(ctx, 1, withExtra(attribute.Int("http.response.status_code", o.StatusCode)))
}
