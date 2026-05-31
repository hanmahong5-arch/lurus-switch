package gateway

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"lurus-switch/internal/relay"
)

// TestShouldFallback is the table that pins which upstream outcomes roll
// over to the next chain entry. 401/403/402 are the load-bearing cases for
// resellers: an upstream that rejects the key (ban / forbidden / no credit)
// must fail over, not be handed back to the caller.
func TestShouldFallback(t *testing.T) {
	cases := []struct {
		name   string
		status int  // ignored when err != nil or resp is nil
		nilRes bool // resp == nil
		err    error
		want   bool
	}{
		{name: "200 OK", status: http.StatusOK, want: false},
		{name: "400 client error", status: http.StatusBadRequest, want: false},
		{name: "404 not found", status: http.StatusNotFound, want: false},
		{name: "422 unprocessable", status: http.StatusUnprocessableEntity, want: false},
		{name: "401 unauthorized", status: http.StatusUnauthorized, want: true},
		{name: "402 payment required", status: http.StatusPaymentRequired, want: true},
		{name: "403 forbidden", status: http.StatusForbidden, want: true},
		{name: "429 rate limited", status: http.StatusTooManyRequests, want: true},
		{name: "500 server error", status: http.StatusInternalServerError, want: true},
		{name: "503 unavailable", status: http.StatusServiceUnavailable, want: true},
		{name: "nil response", nilRes: true, want: true},
		{name: "transport error", err: errors.New("connection refused"), want: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var resp *http.Response
			if !c.nilRes && c.err == nil {
				resp = &http.Response{StatusCode: c.status}
			}
			if got := shouldFallback(resp, c.err); got != c.want {
				t.Fatalf("shouldFallback(status=%d, err=%v) = %v, want %v", c.status, c.err, got, c.want)
			}
		})
	}
}

// TestFallback_Upstream401TripsBreaker proves the end-to-end consequence of
// the shouldFallback change: a dead (401) endpoint reports ok=false to the
// observer on every attempt, so the relay circuit breaker — wired exactly
// the way services.go wires it — opens after the failure threshold and stops
// allowing traffic to the banned key.
func TestFallback_Upstream401TripsBreaker(t *testing.T) {
	dead := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"key banned"}`))
	}))
	defer dead.Close()

	// Frozen clock so the breaker stays open (NextProbe is in the future).
	frozen := func() time.Time { return time.Unix(0, 0) }
	breaker := relay.NewCircuitBreakerForTest(3, time.Minute, frozen)

	fc := NewFallbackChain(nil)
	fc.SetObserver(func(name string, ok bool, errMsg string, latencyMs int64) {
		if ok {
			breaker.RecordSuccess(name)
		} else {
			breaker.RecordFailure(name, errMsg)
		}
	})

	chain := []FallbackEntry{{Name: "dead", URL: dead.URL, Token: "k"}}
	for i := 0; i < 3; i++ {
		_, _, err := fc.TryUpstreamChain(
			context.Background(), "POST", "/v1/chat/completions", "",
			[]byte(`{}`), http.Header{}, chain,
		)
		if err == nil {
			t.Fatalf("attempt %d: expected error after the 401-only chain is exhausted", i)
		}
	}

	if breaker.Allow("dead") {
		t.Fatalf("breaker should be open after 3 consecutive upstream 401s")
	}
}
