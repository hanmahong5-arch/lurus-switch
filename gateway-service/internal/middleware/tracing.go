package middleware

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	tracerName = "gateway-service"
)

// Tracing returns an OpenTelemetry tracing middleware for Hertz
func Tracing() app.HandlerFunc {
	tracer := otel.Tracer(tracerName)
	propagator := otel.GetTextMapPropagator()

	return func(ctx context.Context, c *app.RequestContext) {
		// Extract trace context from incoming request headers
		carrier := &headerCarrier{ctx: c}
		ctx = propagator.Extract(ctx, carrier)

		// Start a new span
		path := string(c.Path())
		method := string(c.Method())
		spanName := method + " " + path

		ctx, span := tracer.Start(ctx, spanName,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				semconv.HTTPMethod(method),
				semconv.HTTPRoute(path),
				semconv.HTTPScheme(string(c.URI().Scheme())),
				semconv.NetHostName(string(c.Host())),
				semconv.UserAgentOriginal(string(c.UserAgent())),
				attribute.String("http.client_ip", c.ClientIP()),
			),
		)
		defer span.End()

		// Store trace ID in request context for logging
		if span.SpanContext().HasTraceID() {
			c.Set("trace_id", span.SpanContext().TraceID().String())
		}

		// Inject trace context into response headers
		propagator.Inject(ctx, carrier)

		// Process request
		c.Next(ctx)

		// Record response status
		statusCode := c.Response.StatusCode()
		span.SetAttributes(semconv.HTTPStatusCode(statusCode))

		if statusCode >= 400 {
			span.SetAttributes(attribute.Bool("error", true))
		}
	}
}

// headerCarrier adapts Hertz RequestContext to OpenTelemetry TextMapCarrier
type headerCarrier struct {
	ctx *app.RequestContext
}

func (c *headerCarrier) Get(key string) string {
	return string(c.ctx.Request.Header.Peek(key))
}

func (c *headerCarrier) Set(key, value string) {
	c.ctx.Response.Header.Set(key, value)
}

func (c *headerCarrier) Keys() []string {
	keys := make([]string, 0)
	c.ctx.Request.Header.VisitAll(func(key, _ []byte) {
		keys = append(keys, string(key))
	})
	return keys
}

// TracingSpanFromContext returns the trace ID and span ID from context
func TracingSpanFromContext(ctx context.Context) (traceID, spanID string) {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().HasTraceID() {
		traceID = span.SpanContext().TraceID().String()
	}
	if span.SpanContext().HasSpanID() {
		spanID = span.SpanContext().SpanID().String()
	}
	return
}

// AddSpanEvent adds an event to the current span
func AddSpanEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.AddEvent(name, trace.WithAttributes(attrs...))
}

// SetSpanAttributes sets attributes on the current span
func SetSpanAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attrs...)
}

// RecordSpanError records an error on the current span
func RecordSpanError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	span.RecordError(err)
}
