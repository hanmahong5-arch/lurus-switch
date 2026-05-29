package obs

import (
	"context"
	"testing"
	"time"
)

// Noop is the gateway's default Recorder. It must never panic and must
// have no side effects regardless of the observation — that's the whole
// point of keeping observability opt-in with zero cost when disabled.
func TestNoop_RecordRequest_NoPanicNoEffect(t *testing.T) {
	rec := Noop()
	// Fully-populated observation.
	rec.RecordRequest(context.Background(), RequestObservation{
		Operation: "chat", Model: "deepseek-chat", ServedBy: "primary",
		TokensIn: 10, TokensOut: 20, StartTime: time.Unix(1700000000, 0),
		LatencyMs: 42, StatusCode: 200, Streaming: true,
	})
	// Zero-value observation (no start time, no model).
	rec.RecordRequest(context.Background(), RequestObservation{})
	// nil context must not panic either.
	rec.RecordRequest(context.TODO(), RequestObservation{Err: "boom"})
}
