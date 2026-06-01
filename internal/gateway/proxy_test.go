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

// TestRecordError_StatusBucketing verifies that recordError books the real
// upstream status so metering buckets it correctly:
//   - 429 → RateLimitEvents (upstream throttled us)
//   - 402 (self-imposed local wall) → NOT in RateLimitEvents, NOT in ErrorEvents
//   - 502 bad gateway → ErrorEvents (5xx)
func TestRecordError_StatusBucketing(t *testing.T) {
	type wantBuckets struct {
		rateLimitEvents int64
		errorEvents     int64
	}
	cases := []struct {
		name       string
		statusCode int
		want       wantBuckets
	}{
		{
			name:       "429 upstream rate-limit → RateLimitEvents",
			statusCode: http.StatusTooManyRequests,
			want:       wantBuckets{rateLimitEvents: 1, errorEvents: 0},
		},
		{
			name:       "402 self-imposed local wall → neither bucket",
			statusCode: http.StatusPaymentRequired,
			want:       wantBuckets{rateLimitEvents: 0, errorEvents: 0},
		},
		{
			name:       "502 bad gateway → ErrorEvents",
			statusCode: http.StatusBadGateway,
			want:       wantBuckets{rateLimitEvents: 0, errorEvents: 1},
		},
		{
			name:       "500 server error → ErrorEvents",
			statusCode: http.StatusInternalServerError,
			want:       wantBuckets{rateLimitEvents: 0, errorEvents: 1},
		},
		{
			name:       "451 DLP block → neither bucket",
			statusCode: http.StatusUnavailableForLegalReasons,
			want:       wantBuckets{rateLimitEvents: 0, errorEvents: 0},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
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

			meta := &RequestMeta{
				AppID:     "test-app",
				StartTime: time.Now(),
			}
			srv.recordError(meta, "test-model", "test error", c.statusCode)

			now := time.Now()
			ins := meter.Insights(now.Add(-time.Minute), now.Add(time.Minute))
			if ins.RateLimitEvents != c.want.rateLimitEvents {
				t.Errorf("RateLimitEvents = %d, want %d", ins.RateLimitEvents, c.want.rateLimitEvents)
			}
			if ins.ErrorEvents != c.want.errorEvents {
				t.Errorf("ErrorEvents = %d, want %d", ins.ErrorEvents, c.want.errorEvents)
			}
		})
	}
}

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

// TestEnsureStreamUsage covers the OpenAI streaming budget-wall fix: only
// streaming requests get stream_options.include_usage injected, and the
// rewrite is idempotent + non-destructive of other fields.
func TestEnsureStreamUsage(t *testing.T) {
	includeUsage := func(t *testing.T, body []byte) bool {
		t.Helper()
		var obj struct {
			StreamOptions *struct {
				IncludeUsage bool `json:"include_usage"`
			} `json:"stream_options"`
		}
		if err := json.Unmarshal(body, &obj); err != nil {
			t.Fatalf("result not valid JSON: %v (%s)", err, body)
		}
		return obj.StreamOptions != nil && obj.StreamOptions.IncludeUsage
	}

	t.Run("streaming request gets include_usage", func(t *testing.T) {
		in := []byte(`{"model":"gpt-4o","stream":true,"messages":[]}`)
		out := ensureStreamUsage(in)
		if !includeUsage(t, out) {
			t.Errorf("include_usage not injected: %s", out)
		}
	})

	t.Run("non-streaming request untouched", func(t *testing.T) {
		in := []byte(`{"model":"gpt-4o","messages":[]}`)
		out := ensureStreamUsage(in)
		if includeUsage(t, out) {
			t.Errorf("include_usage must not be added to non-streaming request: %s", out)
		}
	})

	t.Run("idempotent when already true", func(t *testing.T) {
		in := []byte(`{"model":"gpt-4o","stream":true,"stream_options":{"include_usage":true}}`)
		out := ensureStreamUsage(in)
		if string(out) != string(in) {
			t.Errorf("expected no-op when include_usage already true; got %s", out)
		}
	})

	t.Run("preserves sibling stream_options keys and other fields", func(t *testing.T) {
		in := []byte(`{"model":"gpt-4o","stream":true,"temperature":0.5,"stream_options":{"continuous_usage_stats":true}}`)
		out := ensureStreamUsage(in)
		if !includeUsage(t, out) {
			t.Errorf("include_usage not injected: %s", out)
		}
		var obj struct {
			Temperature   float64 `json:"temperature"`
			StreamOptions struct {
				ContinuousUsageStats bool `json:"continuous_usage_stats"`
			} `json:"stream_options"`
		}
		if err := json.Unmarshal(out, &obj); err != nil {
			t.Fatalf("result not valid JSON: %v", err)
		}
		if obj.Temperature != 0.5 {
			t.Errorf("temperature lost: %s", out)
		}
		if !obj.StreamOptions.ContinuousUsageStats {
			t.Errorf("sibling stream_options key lost: %s", out)
		}
	})

	t.Run("non-JSON body returned as-is", func(t *testing.T) {
		in := []byte(`not json`)
		if out := ensureStreamUsage(in); string(out) != string(in) {
			t.Errorf("non-JSON body must pass through unchanged; got %s", out)
		}
	})
}
