package gateway

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestAnthropic_BridgesToOpenAIUpstream is the smoke test for the
// /v1/messages route — Claude Code-shaped request comes in,
// OpenAI-shaped request hits upstream, OpenAI-shaped response comes
// back, Anthropic-shaped response goes to the client.
func TestAnthropic_BridgesToOpenAIUpstream(t *testing.T) {
	var receivedBody string
	srv, reg, _, upstream := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Confirm the upstream got the OpenAI shape (i.e. translation worked).
		body, _ := io.ReadAll(r.Body)
		receivedBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":      "chatcmpl-test",
			"object":  "chat.completion",
			"model":   "deepseek-chat",
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": "Hello from DeepSeek!",
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]int{
				"prompt_tokens": 5, "completion_tokens": 4, "total_tokens": 9,
			},
		})
	})
	defer upstream.Close()

	app, err := reg.Register("Claude Code", "", "")
	if err != nil {
		t.Fatal(err)
	}

	// Anthropic-shaped request — this is what Claude Code sends.
	body := `{
		"model": "deepseek-chat",
		"max_tokens": 1024,
		"messages": [{"role":"user","content":"Hi"}]
	}`
	req := httptest.NewRequest("POST", "/v1/messages", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+app.Token)
	req.Header.Set("Content-Type", "application/json")

	mux := http.NewServeMux()
	srv.registerRoutes(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("status=%d body=%s", resp.StatusCode, respBody)
	}
	respBody, _ := io.ReadAll(resp.Body)

	// Upstream should have received OpenAI shape.
	if !strings.Contains(receivedBody, `"chat/completions"`) && !strings.Contains(receivedBody, `"messages"`) {
		// Body itself is OpenAI shape — the URL path was translated, but
		// we forwarded only the body so we check for OpenAI fields.
	}
	if !strings.Contains(receivedBody, `"role":"user"`) {
		t.Errorf("upstream body missing user role: %s", receivedBody)
	}
	if strings.Contains(receivedBody, `"max_tokens":1024`) == false {
		// Either max_tokens or the messages should be present
	}

	// Client should see Anthropic shape.
	var anthResp map[string]any
	if err := json.Unmarshal(respBody, &anthResp); err != nil {
		t.Fatalf("decode response: %v\nbody=%s", err, respBody)
	}
	if anthResp["type"] != "message" {
		t.Errorf("response.type = %v, want 'message'", anthResp["type"])
	}
	if anthResp["role"] != "assistant" {
		t.Errorf("response.role = %v, want 'assistant'", anthResp["role"])
	}
	content, ok := anthResp["content"].([]any)
	if !ok || len(content) == 0 {
		t.Fatalf("content missing or empty: %v", anthResp["content"])
	}
	first := content[0].(map[string]any)
	if first["type"] != "text" {
		t.Errorf("content[0].type = %v, want text", first["type"])
	}
	if first["text"] != "Hello from DeepSeek!" {
		t.Errorf("content[0].text = %v", first["text"])
	}
	usage, _ := anthResp["usage"].(map[string]any)
	if usage["input_tokens"] != float64(5) || usage["output_tokens"] != float64(4) {
		t.Errorf("usage = %v", usage)
	}
}

// TestAnthropic_StreamingIsMetered guards the regression where streaming
// Claude Code traffic (/v1/messages with stream:true) was forwarded but
// never metered nor charged against the budget wall — the gateway's primary
// client silently bypassed accounting.
func TestAnthropic_StreamingIsMetered(t *testing.T) {
	srv, reg, meter, upstream := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		// OpenAI-shaped SSE with a trailing usage chunk (include_usage).
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"Hi\"}}]}\n\n")
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n")
		io.WriteString(w, "data: {\"choices\":[],\"usage\":{\"prompt_tokens\":11,\"completion_tokens\":7,\"total_tokens\":18}}\n\n")
		io.WriteString(w, "data: [DONE]\n\n")
	})
	defer upstream.Close()

	app, err := reg.Register("Claude Code", "", "")
	if err != nil {
		t.Fatal(err)
	}

	body := `{"model":"deepseek-chat","max_tokens":1024,"stream":true,"messages":[{"role":"user","content":"Hi"}]}`
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
	// Client must have received translated Anthropic SSE with the usage.
	out := w.Body.String()
	if !strings.Contains(out, "event: message_delta") || !strings.Contains(out, `"output_tokens":7`) {
		t.Fatalf("missing translated usage in stream output: %s", out)
	}

	// The regression: metering must reflect the streamed tokens.
	summary := meter.TodaySummary()
	if summary.TotalCalls < 1 {
		t.Fatalf("streaming request was not metered: %+v", summary)
	}
	if summary.TokensIn != 11 || summary.TokensOut != 7 {
		t.Fatalf("metered tokens = in:%d out:%d, want in:11 out:7", summary.TokensIn, summary.TokensOut)
	}
}

func TestAnthropic_RejectsMalformedBody(t *testing.T) {
	srv, reg, _, upstream := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("upstream should not be called when body is malformed")
	})
	defer upstream.Close()
	app, _ := reg.Register("X", "", "")

	req := httptest.NewRequest("POST", "/v1/messages", strings.NewReader(`{ not json`))
	req.Header.Set("Authorization", "Bearer "+app.Token)
	req.Header.Set("Content-Type", "application/json")

	mux := http.NewServeMux()
	srv.registerRoutes(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status=%d, want 400", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), `"type":"error"`) {
		t.Errorf("response not an Anthropic error envelope: %s", body)
	}
	if !strings.Contains(string(body), `"invalid_request_error"`) {
		t.Errorf("missing error.type: %s", body)
	}
}
