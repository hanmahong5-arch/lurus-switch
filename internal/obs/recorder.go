// Package obs is Switch's optional OpenTelemetry GenAI observability
// layer. The gateway records one RequestObservation per proxied request
// through the narrow Recorder interface; the default Noop() implementation
// has zero cost, so observability stays off unless the user configures an
// OTLP endpoint.
//
// Design: the Recorder interface + Noop live in this file and import no
// OpenTelemetry packages, so the gateway depends only on this seam. The
// gateway never imports otel directly, stays unit-testable against a fake
// Recorder, and all SDK wiring is isolated in otel.go (built behind the
// same package so a disabled install still links but pays nothing at run
// time via Noop).
package obs

import (
	"context"
	"time"
)

// Recorder receives one observation per completed gateway request. The
// gateway calls RecordRequest from the same place it records metering, so
// model / tokens / latency are already resolved — there is no live span
// threaded through the auth→handler→record path.
type Recorder interface {
	RecordRequest(ctx context.Context, o RequestObservation)
}

// RequestObservation is the flat set of facts the gateway knows about a
// completed request. It mirrors the metering.Record dimensions plus timing.
// Kept flat because the OTel attribute mapping references each field by
// name (see otelRecorder.RecordRequest).
type RequestObservation struct {
	Operation  string // "chat" (OpenAI path) | "messages" (Anthropic path)
	Model      string
	ServedBy   string // upstream that served the request → server.address
	MatchedBy  string // relay rule that selected the upstream (may be empty)
	TokensIn   int64
	TokensOut  int64
	StartTime  time.Time
	LatencyMs  int64
	StatusCode int
	Streaming  bool
	Err        string // non-empty marks the span as errored
}

// Noop returns a Recorder that does nothing — the gateway's default, so
// observability is opt-in with zero runtime cost when disabled.
func Noop() Recorder { return noopRecorder{} }

type noopRecorder struct{}

func (noopRecorder) RecordRequest(context.Context, RequestObservation) {}
