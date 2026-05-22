package gateway

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"lurus-switch/internal/dlp"
)

// TestDLP_BlockPattern_Returns451 confirms a request whose body matches
// a PolicyBlock pattern is rejected before any upstream call.
func TestDLP_BlockPattern_Returns451(t *testing.T) {
	upstreamCalled := false
	srv, reg, _, upstream := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		upstreamCalled = true
		w.WriteHeader(http.StatusOK)
	})
	defer upstream.Close()

	// Wire the default DLP scanner — its built-in patterns block GitHub PATs.
	srv.SetDLPScanner(dlp.NewScanner())

	app, err := reg.Register("Test App", "", "")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	// GitHub PAT is a default PolicyBlock pattern.
	body := `{"model": "gpt-4o", "messages": [{"role": "user", "content": "leaked: ghp_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}]}`
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+app.Token)
	req.Header.Set("Content-Type", "application/json")

	mux := http.NewServeMux()
	srv.registerRoutes(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnavailableForLegalReasons {
		t.Fatalf("expected 451 (DLP blocked), got %d: %s", w.Code, w.Body.String())
	}
	if upstreamCalled {
		t.Error("upstream should NOT be called when DLP blocks the request")
	}

	var errResp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	errInner, _ := errResp["error"].(map[string]any)
	if code, _ := errInner["code"].(string); code != "dlp_blocked" {
		t.Errorf("expected error code dlp_blocked, got %v", errInner)
	}
}

// TestDLP_RedactPattern_RewritesBodyForUpstream confirms PolicyRedact
// patterns swap the body before forwarding so the upstream never sees
// the secret.
func TestDLP_RedactPattern_RewritesBodyForUpstream(t *testing.T) {
	var receivedBody string
	srv, reg, _, upstream := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		receivedBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"id": "ok"})
	})
	defer upstream.Close()

	srv.SetDLPScanner(dlp.NewScanner())

	app, _ := reg.Register("Test App", "", "")

	// JWTs are PolicyRedact in defaults.
	body := `{"model":"gpt-4o","messages":[{"role":"user","content":"token=eyJabcdef0123.eyJabcdef0123.signaturesignaturesig"}]}`
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+app.Token)
	req.Header.Set("Content-Type", "application/json")

	mux := http.NewServeMux()
	srv.registerRoutes(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if strings.Contains(receivedBody, "eyJabcdef0123.eyJabcdef0123.signaturesignaturesig") {
		t.Errorf("upstream received unredacted JWT — DLP redact failed. Body: %s", receivedBody)
	}
	if !strings.Contains(receivedBody, "[REDACTED:secret.jwt]") {
		t.Errorf("expected [REDACTED:secret.jwt] marker in upstream body, got: %s", receivedBody)
	}
}

// TestDLP_NoScanner_PassThrough confirms the gateway works unchanged
// when no DLP scanner is wired.
func TestDLP_NoScanner_PassThrough(t *testing.T) {
	called := false
	srv, reg, _, upstream := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"id": "ok"})
	})
	defer upstream.Close()

	app, _ := reg.Register("Test App", "", "")

	body := `{"model": "gpt-4o", "messages": []}`
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+app.Token)

	mux := http.NewServeMux()
	srv.registerRoutes(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if !called {
		t.Error("upstream should be called when no DLP scanner is configured")
	}
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

// TestDLP_HitRecorded confirms the scanner's hit-ring captures gateway
// hits so the admin UI can poll them.
func TestDLP_HitRecorded(t *testing.T) {
	srv, reg, _, upstream := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"id": "ok"})
	})
	defer upstream.Close()

	scanner := dlp.NewScanner()
	srv.SetDLPScanner(scanner)

	app, _ := reg.Register("Test App", "", "")

	// Email is a default PolicyWarn pattern — won't block, but should
	// land in the hit ring.
	body := `{"model":"gpt-4o","messages":[{"role":"user","content":"reach out to alice@example.com"}]}`
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+app.Token)

	mux := http.NewServeMux()
	srv.registerRoutes(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	hits := scanner.RecentHits(10)
	if len(hits) == 0 {
		t.Fatal("expected at least one hit recorded")
	}
	found := false
	for _, h := range hits {
		if h.Source == "gateway.request" && h.Hit.PatternName == "pii.email" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected gateway.request hit for pii.email, got %+v", hits)
	}
}
