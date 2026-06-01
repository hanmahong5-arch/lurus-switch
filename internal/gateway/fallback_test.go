package gateway

import (
	"context"
	"errors"
	"fmt"
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

// TestTryUpstreamChain_LastStatusSurfaced verifies that when all chain entries
// fail, the returned error wraps an *UpstreamExhaustedError whose LastStatus
// carries the final HTTP status code. A transport-only failure (no HTTP
// response ever arrived) must set LastStatus==0.
func TestTryUpstreamChain_LastStatusSurfaced(t *testing.T) {
	cases := []struct {
		name           string
		upstreamStatus int  // 0 = simulate transport failure (close immediately)
		wantLastStatus int
	}{
		{name: "429 rate-limit", upstreamStatus: http.StatusTooManyRequests, wantLastStatus: 429},
		{name: "401 unauthorized", upstreamStatus: http.StatusUnauthorized, wantLastStatus: 401},
		{name: "402 payment required", upstreamStatus: http.StatusPaymentRequired, wantLastStatus: 402},
		{name: "500 server error", upstreamStatus: http.StatusInternalServerError, wantLastStatus: 500},
		{name: "transport failure (no response)", upstreamStatus: 0, wantLastStatus: 0},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var ts *httptest.Server
			if c.upstreamStatus == 0 {
				// Immediately close the connection to simulate a transport error.
				ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Hijack and close without writing a response.
					hj, ok := w.(http.Hijacker)
					if !ok {
						t.Error("ResponseWriter does not implement Hijacker")
						return
					}
					conn, _, _ := hj.Hijack()
					conn.Close()
				}))
			} else {
				status := c.upstreamStatus
				ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(status)
					fmt.Fprintf(w, `{"error":"upstream %d"}`, status)
				}))
			}
			defer ts.Close()

			fc := NewFallbackChain(nil)
			chain := []FallbackEntry{{Name: "only", URL: ts.URL, Token: "k"}}

			_, _, err := fc.TryUpstreamChain(
				context.Background(), "POST", "/v1/chat/completions", "",
				[]byte(`{}`), http.Header{}, chain,
			)
			if err == nil {
				t.Fatal("expected an error from an exhausted chain")
			}

			var ue *UpstreamExhaustedError
			if !errors.As(err, &ue) {
				t.Fatalf("expected *UpstreamExhaustedError, got %T: %v", err, err)
			}
			if ue.LastStatus != c.wantLastStatus {
				t.Fatalf("LastStatus = %d, want %d", ue.LastStatus, c.wantLastStatus)
			}
		})
	}
}
