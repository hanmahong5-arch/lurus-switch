package gateway

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"lurus-switch/internal/obs"
)

// captureRecorder is a fake obs.Recorder that stores every observation so a
// test can assert the gateway feeds the observability seam with the right
// model / tokens / servedBy / streaming flag — the gateway-side contract,
// independent of any otel SDK.
type captureRecorder struct {
	mu   sync.Mutex
	seen []obs.RequestObservation
}

func (c *captureRecorder) RecordRequest(_ context.Context, o obs.RequestObservation) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.seen = append(c.seen, o)
}

func (c *captureRecorder) all() []obs.RequestObservation {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]obs.RequestObservation(nil), c.seen...)
}

func serveOnce(t *testing.T, srv *Server, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	srv.registerRoutes(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

func TestObserve_OpenAIBuffered(t *testing.T) {
	srv, reg, _, upstream := setupTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "chatcmpl-x", "model": "deepseek-chat",
			"usage":   map[string]int{"prompt_tokens": 11, "completion_tokens": 22, "total_tokens": 33},
			"choices": []map[string]any{{"message": map[string]string{"content": "hi"}}},
		})
	})
	defer upstream.Close()

	cap := &captureRecorder{}
	srv.SetObserver(cap)

	app, err := reg.Register("t", "", "")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("POST", "/v1/chat/completions",
		strings.NewReader(`{"model":"deepseek-chat","messages":[{"role":"user","content":"Hi"}]}`))
	req.Header.Set("Authorization", "Bearer "+app.Token)
	req.Header.Set("Content-Type", "application/json")

	if w := serveOnce(t, srv, req); w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}

	seen := cap.all()
	if len(seen) != 1 {
		t.Fatalf("expected 1 observation, got %d", len(seen))
	}
	o := seen[0]
	if o.Operation != "chat" {
		t.Errorf("operation = %q, want chat", o.Operation)
	}
	if o.Model != "deepseek-chat" {
		t.Errorf("model = %q", o.Model)
	}
	if o.TokensIn != 11 || o.TokensOut != 22 {
		t.Errorf("tokens = in:%d out:%d, want 11/22", o.TokensIn, o.TokensOut)
	}
	if o.Streaming {
		t.Error("buffered request must not be marked streaming")
	}
	if o.StatusCode != http.StatusOK {
		t.Errorf("status = %d", o.StatusCode)
	}
}

func TestObserve_OpenAIStreaming(t *testing.T) {
	srv, reg, _, upstream := setupTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"hi\"}}]}\n\n")
		io.WriteString(w, "data: {\"choices\":[],\"usage\":{\"prompt_tokens\":7,\"completion_tokens\":5,\"total_tokens\":12}}\n\n")
		io.WriteString(w, "data: [DONE]\n\n")
	})
	defer upstream.Close()

	cap := &captureRecorder{}
	srv.SetObserver(cap)

	app, err := reg.Register("t", "", "")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("POST", "/v1/chat/completions",
		strings.NewReader(`{"model":"deepseek-chat","stream":true,"messages":[{"role":"user","content":"Hi"}]}`))
	req.Header.Set("Authorization", "Bearer "+app.Token)
	req.Header.Set("Content-Type", "application/json")

	if w := serveOnce(t, srv, req); w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}

	seen := cap.all()
	if len(seen) != 1 {
		t.Fatalf("expected 1 observation, got %d", len(seen))
	}
	o := seen[0]
	if !o.Streaming {
		t.Error("streaming request must be marked streaming")
	}
	if o.TokensIn != 7 || o.TokensOut != 5 {
		t.Errorf("tokens = in:%d out:%d, want 7/5", o.TokensIn, o.TokensOut)
	}
}

func TestObserve_AnthropicBufferedUsesMessagesOperation(t *testing.T) {
	srv, reg, _, upstream := setupTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "chatcmpl-x", "object": "chat.completion", "model": "deepseek-chat",
			"choices": []map[string]any{{
				"index": 0, "finish_reason": "stop",
				"message": map[string]any{"role": "assistant", "content": "Hello!"},
			}},
			"usage": map[string]int{"prompt_tokens": 5, "completion_tokens": 4, "total_tokens": 9},
		})
	})
	defer upstream.Close()

	cap := &captureRecorder{}
	srv.SetObserver(cap)

	app, err := reg.Register("Claude Code", "", "")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("POST", "/v1/messages",
		strings.NewReader(`{"model":"deepseek-chat","max_tokens":1024,"messages":[{"role":"user","content":"Hi"}]}`))
	req.Header.Set("Authorization", "Bearer "+app.Token)
	req.Header.Set("Content-Type", "application/json")

	if w := serveOnce(t, srv, req); w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}

	seen := cap.all()
	if len(seen) != 1 {
		t.Fatalf("expected 1 observation, got %d", len(seen))
	}
	o := seen[0]
	if o.Operation != "messages" {
		t.Errorf("operation = %q, want messages", o.Operation)
	}
	if o.TokensIn != 5 || o.TokensOut != 4 {
		t.Errorf("tokens = in:%d out:%d, want 5/4", o.TokensIn, o.TokensOut)
	}
}

// Default server (no SetObserver) must use the Noop recorder — i.e. the
// proxy path runs end-to-end without a recorder wired, no panic.
func TestObserve_DefaultIsNoop(t *testing.T) {
	srv, reg, _, upstream := setupTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "chatcmpl-x", "model": "deepseek-chat",
			"usage":   map[string]int{"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2},
			"choices": []map[string]any{{"message": map[string]string{"content": "hi"}}},
		})
	})
	defer upstream.Close()

	app, err := reg.Register("t", "", "")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("POST", "/v1/chat/completions",
		strings.NewReader(`{"model":"deepseek-chat","messages":[{"role":"user","content":"Hi"}]}`))
	req.Header.Set("Authorization", "Bearer "+app.Token)
	req.Header.Set("Content-Type", "application/json")

	if w := serveOnce(t, srv, req); w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}
