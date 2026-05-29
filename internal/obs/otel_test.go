package obs

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// newTestRecorder wires the recorder against in-memory SDK providers so the
// whole gen_ai mapping is exercised without an OTLP exporter or network.
func newTestRecorder(t *testing.T) (*otelRecorder, *tracetest.SpanRecorder, *sdkmetric.ManualReader) {
	t.Helper()
	spanRec := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRec))
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	rec, err := newRecorder(tp, mp)
	if err != nil {
		t.Fatalf("newRecorder: %v", err)
	}
	return rec, spanRec, reader
}

func TestOtelRecorder_Span_GenAIAttributesAndTiming(t *testing.T) {
	rec, spanRec, _ := newTestRecorder(t)

	start := time.Unix(1700000000, 0)
	rec.RecordRequest(context.Background(), RequestObservation{
		Operation: "chat", Model: "deepseek-chat", ServedBy: "primary",
		MatchedBy: "long-context", TokensIn: 100, TokensOut: 250,
		StartTime: start, LatencyMs: 1500, StatusCode: 200, Streaming: true,
	})

	spans := spanRec.Ended()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	sp := spans[0]
	if sp.Name() != "gen_ai.chat" {
		t.Errorf("span name = %q, want gen_ai.chat", sp.Name())
	}
	// Span timing must be retrospective: end == start + latency.
	if d := sp.EndTime().Sub(sp.StartTime()); d != 1500*time.Millisecond {
		t.Errorf("span duration = %v, want 1.5s", d)
	}

	attrs := attrMap(sp.Attributes())
	if got := attrs["gen_ai.request.model"].AsString(); got != "deepseek-chat" {
		t.Errorf("gen_ai.request.model = %q", got)
	}
	if got := attrs["gen_ai.response.model"].AsString(); got != "deepseek-chat" {
		t.Errorf("gen_ai.response.model = %q", got)
	}
	if got := attrs["gen_ai.usage.input_tokens"].AsInt64(); got != 100 {
		t.Errorf("input_tokens = %d", got)
	}
	if got := attrs["gen_ai.usage.output_tokens"].AsInt64(); got != 250 {
		t.Errorf("output_tokens = %d", got)
	}
	if got := attrs["server.address"].AsString(); got != "primary" {
		t.Errorf("server.address = %q", got)
	}
	if got := attrs["gen_ai.operation.name"].AsString(); got != "chat" {
		t.Errorf("operation.name = %q", got)
	}
}

func TestOtelRecorder_Span_ErrorStatus(t *testing.T) {
	rec, spanRec, _ := newTestRecorder(t)
	rec.RecordRequest(context.Background(), RequestObservation{
		Operation: "messages", Model: "claude-x", StartTime: time.Unix(1700000000, 0),
		LatencyMs: 10, StatusCode: 502, Err: "all upstreams failed",
	})
	spans := spanRec.Ended()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	if spans[0].Status().Code != codes.Error {
		t.Errorf("span status = %v, want Error", spans[0].Status().Code)
	}
}

func TestOtelRecorder_Metrics_TokenUsageSplitByType(t *testing.T) {
	rec, _, reader := newTestRecorder(t)
	rec.RecordRequest(context.Background(), RequestObservation{
		Operation: "chat", Model: "deepseek-chat", ServedBy: "primary",
		TokensIn: 100, TokensOut: 250, StartTime: time.Unix(1700000000, 0),
		LatencyMs: 1500, StatusCode: 200,
	})

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("Collect: %v", err)
	}

	hist, ok := findHistogram(rm, "gen_ai.client.token.usage")
	if !ok {
		t.Fatalf("gen_ai.client.token.usage histogram not found")
	}
	var inSum, outSum int64
	for _, dp := range hist.DataPoints {
		typ, _ := dp.Attributes.Value("gen_ai.token.type")
		switch typ.AsString() {
		case "input":
			inSum = dp.Sum
		case "output":
			outSum = dp.Sum
		}
	}
	if inSum != 100 {
		t.Errorf("input token sum = %d, want 100", inSum)
	}
	if outSum != 250 {
		t.Errorf("output token sum = %d, want 250", outSum)
	}

	// A request count of 1 must also be booked.
	if cnt, ok := findCounter(rm, "gen_ai.client.requests"); !ok || cnt != 1 {
		t.Errorf("gen_ai.client.requests = %d (found=%v), want 1", cnt, ok)
	}
}

// --- helpers ---

func attrMap(kvs []attribute.KeyValue) map[attribute.Key]attribute.Value {
	m := make(map[attribute.Key]attribute.Value, len(kvs))
	for _, kv := range kvs {
		m[kv.Key] = kv.Value
	}
	return m
}

func findHistogram(rm metricdata.ResourceMetrics, name string) (metricdata.Histogram[int64], bool) {
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != name {
				continue
			}
			if h, ok := m.Data.(metricdata.Histogram[int64]); ok {
				return h, true
			}
		}
	}
	return metricdata.Histogram[int64]{}, false
}

func findCounter(rm metricdata.ResourceMetrics, name string) (int64, bool) {
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != name {
				continue
			}
			if s, ok := m.Data.(metricdata.Sum[int64]); ok {
				var total int64
				for _, dp := range s.DataPoints {
					total += dp.Value
				}
				return total, true
			}
		}
	}
	return 0, false
}
