package gateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"lurus-switch/internal/pricing"
)

// TestRecordUsage_OpenAICachedNotDoubleCounted is the core billing-correctness
// guard for the OpenAI-protocol path. OpenAI reports cached_tokens as a SUBSET
// of prompt_tokens, so the old code (which billed prompt_tokens as fresh input)
// would have charged full price for the cached portion. After normalization a
// prompt of 1000 with 800 cached must meter as TokensIn=200 + CacheReadTokens=800,
// and the resulting cost must sit below the full-price number.
func TestRecordUsage_OpenAICachedNotDoubleCounted(t *testing.T) {
	srv, reg, meter, upstream := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":    "ok",
			"model": "claude-sonnet-4-6",
			"usage": map[string]any{
				"prompt_tokens":             1000,
				"completion_tokens":         100,
				"total_tokens":              1100,
				"prompt_tokens_details":     map[string]int{"cached_tokens": 800},
				"completion_tokens_details": map[string]int{"reasoning_tokens": 40},
			},
		})
	})
	defer upstream.Close()

	app, err := reg.Register("Claude Code", "", "")
	if err != nil {
		t.Fatal(err)
	}

	body := `{"model":"claude-sonnet-4-6","messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+app.Token)
	req.Header.Set("Content-Type", "application/json")
	mux := http.NewServeMux()
	srv.registerRoutes(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}

	recs := meter.RecentRecords(10)
	if len(recs) != 1 {
		t.Fatalf("expected 1 metered record, got %d", len(recs))
	}
	r := recs[0]
	if r.TokensIn != 200 {
		t.Errorf("TokensIn = %d, want 200 (prompt 1000 − cached 800)", r.TokensIn)
	}
	if r.CacheReadTokens != 800 {
		t.Errorf("CacheReadTokens = %d, want 800", r.CacheReadTokens)
	}
	if r.CacheCreateTokens != 0 {
		t.Errorf("CacheCreateTokens = %d, want 0 (OpenAI bills no separate cache-write)", r.CacheCreateTokens)
	}
	if r.TokensOut != 100 {
		t.Errorf("TokensOut = %d, want 100", r.TokensOut)
	}
	if r.ReasoningTokens != 40 {
		t.Errorf("ReasoningTokens = %d, want 40 (display only)", r.ReasoningTokens)
	}

	cost := pricing.Cost(r.Model, r.TokensIn, r.TokensOut, r.CacheCreateTokens, r.CacheReadTokens)
	full := pricing.Cost(r.Model, 1000, 100, 0, 0) // the old, wrong, full-price-on-everything figure
	if cost >= full {
		t.Errorf("cached billing %.6f should be cheaper than full-price %.6f", cost, full)
	}
	p := pricing.PriceFor(r.Model)
	want := 200.0/1e6*p.InputPerMTok + 100.0/1e6*p.OutputPerMTok + 800.0/1e6*p.CacheReadPerMTok
	if diff := cost - want; diff > 1e-9 || diff < -1e-9 {
		t.Errorf("cost = %.9f, want %.9f (200×in + 100×out + 800×cacheRead)", cost, want)
	}
}

// TestAnthropicBuffered_OpenAICachedNotDoubleCounted is the same guard for the
// /v1/messages buffered path: the upstream is OpenAI-shaped, so cached_tokens
// must be subtracted out of billed input there too.
func TestAnthropicBuffered_OpenAICachedNotDoubleCounted(t *testing.T) {
	srv, reg, meter, upstream := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":     "chatcmpl",
			"object": "chat.completion",
			"model":  "deepseek-chat",
			"choices": []map[string]any{{
				"index":         0,
				"message":       map[string]any{"role": "assistant", "content": "ok"},
				"finish_reason": "stop",
			}},
			"usage": map[string]any{
				"prompt_tokens":             1000,
				"completion_tokens":         60,
				"total_tokens":              1060,
				"prompt_tokens_details":     map[string]int{"cached_tokens": 900},
				"completion_tokens_details": map[string]int{"reasoning_tokens": 20},
			},
		})
	})
	defer upstream.Close()

	app, err := reg.Register("Claude Code", "", "")
	if err != nil {
		t.Fatal(err)
	}

	body := `{"model":"deepseek-chat","max_tokens":256,"messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest("POST", "/v1/messages", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+app.Token)
	req.Header.Set("Content-Type", "application/json")
	mux := http.NewServeMux()
	srv.registerRoutes(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}

	recs := meter.RecentRecords(10)
	if len(recs) != 1 {
		t.Fatalf("expected 1 metered record, got %d", len(recs))
	}
	r := recs[0]
	if r.TokensIn != 100 {
		t.Errorf("TokensIn = %d, want 100 (1000 − 900 cached)", r.TokensIn)
	}
	if r.CacheReadTokens != 900 {
		t.Errorf("CacheReadTokens = %d, want 900", r.CacheReadTokens)
	}
	if r.TokensOut != 60 {
		t.Errorf("TokensOut = %d, want 60", r.TokensOut)
	}
	if r.ReasoningTokens != 20 {
		t.Errorf("ReasoningTokens = %d, want 20", r.ReasoningTokens)
	}
}
