package gateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// okUsageUpstream returns a handler that replies with a fixed OpenAI usage so
// the dedup tests can assert on exact metered totals.
func okUsageUpstream() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":    "ok",
			"model": "claude-sonnet-4-6",
			"usage": map[string]int{"prompt_tokens": 100, "completion_tokens": 50, "total_tokens": 150},
		})
	}
}

// TestProxy_IdempotencyKeyDedup proves that two requests carrying the same
// client Idempotency-Key bill exactly once — the SDK-retry case the dedup
// guard exists for.
func TestProxy_IdempotencyKeyDedup(t *testing.T) {
	srv, reg, meter, upstream := setupTestServer(t, okUsageUpstream())
	defer upstream.Close()

	app, err := reg.Register("Claude Code", "", "")
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	srv.registerRoutes(mux)

	send := func() {
		body := `{"model":"claude-sonnet-4-6","messages":[{"role":"user","content":"hi"}]}`
		req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+app.Token)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", "fixed-key-123")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
	}
	send()
	send()

	sum := meter.TodaySummary()
	if sum.TotalCalls != 1 {
		t.Fatalf("TotalCalls = %d, want 1 (same Idempotency-Key deduped)", sum.TotalCalls)
	}
	if sum.TokensIn != 100 {
		t.Fatalf("TokensIn = %d, want 100 (cached/retry not double-counted)", sum.TokensIn)
	}
}

// TestProxy_NoIdempotencyKeyRecordsEachRequest proves the dedup is scoped to a
// stable correlation key: without one, each request gets a unique generated id
// and bills independently (no blanket request collapsing).
func TestProxy_NoIdempotencyKeyRecordsEachRequest(t *testing.T) {
	srv, reg, meter, upstream := setupTestServer(t, okUsageUpstream())
	defer upstream.Close()

	app, err := reg.Register("Claude Code", "", "")
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	srv.registerRoutes(mux)

	send := func() {
		body := `{"model":"claude-sonnet-4-6","messages":[{"role":"user","content":"hi"}]}`
		req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+app.Token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
	}
	send()
	send()

	sum := meter.TodaySummary()
	if sum.TotalCalls != 2 {
		t.Fatalf("TotalCalls = %d, want 2 (distinct generated ids bill separately)", sum.TotalCalls)
	}
}
