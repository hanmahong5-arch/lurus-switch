package gateway

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"lurus-switch/internal/appreg"
	"lurus-switch/internal/metering"
)

func setupTestServer(t *testing.T, upstreamHandler http.HandlerFunc) (*Server, *appreg.Registry, *metering.Store, *httptest.Server) {
	t.Helper()

	upstream := httptest.NewServer(upstreamHandler)

	dir := t.TempDir()
	reg, err := appreg.NewRegistry(dir)
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	meter, err := metering.NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	srv := NewServer(dir, reg, meter)
	srv.cfg.UpstreamURL = upstream.URL
	srv.cfg.UserToken = "test-upstream-token"

	return srv, reg, meter, upstream
}

func TestProxy_NonStreaming(t *testing.T) {
	// Mock upstream that returns a standard OpenAI response.
	srv, reg, meter, upstream := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Verify the auth header was swapped.
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-upstream-token" {
			t.Errorf("expected upstream token, got %q", auth)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    "chatcmpl-test",
			"model": "claude-sonnet-4-6",
			"usage": map[string]int{
				"prompt_tokens":     100,
				"completion_tokens": 200,
				"total_tokens":      300,
			},
			"choices": []map[string]interface{}{
				{"message": map[string]string{"content": "Hello!"}},
			},
		})
	})
	defer upstream.Close()

	// Register a test app and get its token.
	app, err := reg.Register("Test App", "", "")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	// Build test request.
	body := `{"model": "claude-sonnet-4-6", "messages": [{"role": "user", "content": "Hi"}]}`
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+app.Token)
	req.Header.Set("Content-Type", "application/json")

	// Set up handler with auth.
	mux := http.NewServeMux()
	srv.registerRoutes(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(respBody))
	}

	// Verify metering recorded the call.
	time.Sleep(10 * time.Millisecond) // metering is async-ish
	summary := meter.TodaySummary()
	if summary.TotalCalls < 1 {
		t.Error("expected at least 1 metered call")
	}
}

func TestProxy_AuthRequired(t *testing.T) {
	srv, _, _, upstream := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("upstream should not be called for unauthorized requests")
	})
	defer upstream.Close()

	// Request without auth.
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader("{}"))
	mux := http.NewServeMux()
	srv.registerRoutes(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestProxy_InvalidToken(t *testing.T) {
	srv, _, _, upstream := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("upstream should not be called for invalid token")
	})
	defer upstream.Close()

	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader("{}"))
	req.Header.Set("Authorization", "Bearer sk-switch-invalid-token")
	mux := http.NewServeMux()
	srv.registerRoutes(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestProxy_HealthEndpoint(t *testing.T) {
	dir := t.TempDir()
	reg, _ := appreg.NewRegistry(dir)
	meter, _ := metering.NewStore(dir)
	srv := NewServer(dir, reg, meter)

	mux := http.NewServeMux()
	srv.registerRoutes(mux)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode health response: %v", err)
	}
	if result["status"] != "ok" {
		t.Fatalf("expected status ok, got %v", result["status"])
	}
}

func TestExtractModelFromBody(t *testing.T) {
	body := []byte(`{"model": "gpt-4o", "messages": []}`)
	model := extractModelFromBody(body)
	if model != "gpt-4o" {
		t.Fatalf("expected gpt-4o, got %q", model)
	}
}

func TestExtractUsageFromBody(t *testing.T) {
	body := []byte(`{"model":"test","usage":{"prompt_tokens":10,"completion_tokens":20,"total_tokens":30}}`)
	usage := extractUsageFromBody(body)
	if usage.PromptTokens != 10 || usage.CompletionTokens != 20 || usage.TotalTokens != 30 {
		t.Fatalf("unexpected usage: %+v", usage)
	}
}

func TestExtractUsageFromSSEChunk(t *testing.T) {
	chunk := []byte(`data: {"model":"test","usage":{"prompt_tokens":5,"completion_tokens":15,"total_tokens":20}}` + "\n\n")
	usage := extractUsageFromSSEChunk(chunk)
	if usage.TotalTokens != 20 {
		t.Fatalf("expected 20 total tokens, got %d", usage.TotalTokens)
	}

	// Chunk without usage should return zero.
	chunk2 := []byte(`data: {"choices":[{"delta":{"content":"Hi"}}]}` + "\n\n")
	usage2 := extractUsageFromSSEChunk(chunk2)
	if usage2.TotalTokens != 0 {
		t.Fatalf("expected 0 total tokens, got %d", usage2.TotalTokens)
	}

	// [DONE] chunk should return zero.
	chunk3 := []byte("data: [DONE]\n\n")
	usage3 := extractUsageFromSSEChunk(chunk3)
	if usage3.TotalTokens != 0 {
		t.Fatalf("expected 0 total tokens for DONE, got %d", usage3.TotalTokens)
	}
}

// feedAll drives a scanner with the given byte fragments and returns the
// final usage — modelling proxyStreaming's chunked Read() loop.
func feedAll(fragments ...[]byte) UsageFromResponse {
	var s sseUsageScanner
	for _, f := range fragments {
		s.feed(f)
	}
	return s.finish()
}

func TestSSEUsageScanner_WholeLine(t *testing.T) {
	full := []byte(`data: {"model":"m","usage":{"prompt_tokens":5,"completion_tokens":15,"total_tokens":20}}` + "\n\n")
	u := feedAll(full)
	if u.TotalTokens != 20 || u.PromptTokens != 5 || u.CompletionTokens != 15 {
		t.Fatalf("unexpected usage: %+v", u)
	}
}

func TestSSEUsageScanner_SplitAcrossBoundary(t *testing.T) {
	// The bug: a 4 KB Read() splits the usage line. Each half is unparseable
	// on its own; only line-buffered accumulation recovers it.
	full := `data: {"choices":[{"delta":{"content":"hi"}}]}` + "\n\n" +
		`data: {"model":"m","usage":{"prompt_tokens":7,"completion_tokens":11,"total_tokens":18}}` + "\n\n"
	for split := 1; split < len(full); split++ {
		u := feedAll([]byte(full[:split]), []byte(full[split:]))
		if u.TotalTokens != 18 {
			t.Fatalf("split at %d: expected 18 total tokens, got %d (%+v)", split, u.TotalTokens, u)
		}
	}
}

func TestSSEUsageScanner_ByteByByte(t *testing.T) {
	full := []byte(`data: {"usage":{"prompt_tokens":3,"completion_tokens":4,"total_tokens":7}}` + "\n\n")
	var s sseUsageScanner
	for i := 0; i < len(full); i++ {
		s.feed(full[i : i+1])
	}
	if u := s.finish(); u.TotalTokens != 7 {
		t.Fatalf("byte-by-byte: expected 7 total tokens, got %d", u.TotalTokens)
	}
}

func TestSSEUsageScanner_NoTotalTokens(t *testing.T) {
	// Some OpenAI-compatible providers omit total_tokens; usage must still be
	// recorded from prompt/completion (the usageNonZero fix).
	full := []byte(`data: {"usage":{"prompt_tokens":12,"completion_tokens":8}}` + "\n\n")
	u := feedAll(full)
	if u.PromptTokens != 12 || u.CompletionTokens != 8 {
		t.Fatalf("expected prompt=12 completion=8, got %+v", u)
	}
}

func TestSSEUsageScanner_UnterminatedTail(t *testing.T) {
	// Stream ends without a trailing newline — finish() must still scan it.
	full := []byte(`data: {"usage":{"prompt_tokens":1,"completion_tokens":2,"total_tokens":3}}`)
	u := feedAll(full)
	if u.TotalTokens != 3 {
		t.Fatalf("expected 3 total tokens from unterminated tail, got %d", u.TotalTokens)
	}
}

func TestSSEUsageScanner_NoUsage(t *testing.T) {
	full := []byte("data: {\"choices\":[{\"delta\":{\"content\":\"hi\"}}]}\n\ndata: [DONE]\n\n")
	if u := feedAll(full); usageNonZero(u) {
		t.Fatalf("expected zero usage, got %+v", u)
	}
}
