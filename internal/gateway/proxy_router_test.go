package gateway

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"lurus-switch/internal/appreg"
	"lurus-switch/internal/metering"
	"lurus-switch/internal/relay"
)

// TestProxy_RouterDrivenChainPicksByRule wires a relay router with a
// `match_model_prefix` rule and verifies that a matching request lands
// on the rule's target endpoint, not the cfg.UpstreamURL.
func TestProxy_RouterDrivenChainPicksByRule(t *testing.T) {
	cfgUp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("cfg upstream should not be hit when router has a healthy endpoint")
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer cfgUp.Close()

	hitA := int32(0)
	endpointA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hitA, 1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":    "ok",
			"model": "claude-sonnet-4-6",
			"usage": map[string]int{"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2},
		})
	}))
	defer endpointA.Close()

	dir := t.TempDir()
	reg, _ := appreg.NewRegistry(dir)
	meter, _ := metering.NewStore(dir)

	store, err := relay.NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveEndpoint(relay.RelayEndpoint{
		ID: "alpha", Name: "alpha", URL: endpointA.URL, APIKey: "alpha-key", Healthy: true, LatencyMs: 10,
	}); err != nil {
		t.Fatal(err)
	}
	router, err := relay.NewRouter(dir, store, relay.NewCircuitBreaker())
	if err != nil {
		t.Fatal(err)
	}
	if err := router.LoadRulesYAML(`
rules:
  - name: claude-to-alpha
    match_model_prefix: claude
    prefer_endpoint_id: alpha
`); err != nil {
		t.Fatal(err)
	}

	srv := NewServer(dir, reg, meter)
	srv.cfg.UpstreamURL = cfgUp.URL
	srv.cfg.UserToken = "user-token"
	srv.SetRelayRouter(router)

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

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("status=%d body=%s", resp.StatusCode, respBody)
	}
	if got := atomic.LoadInt32(&hitA); got != 1 {
		t.Fatalf("endpoint alpha hit count = %d, want 1", got)
	}
}

// TestProxy_EmptyRouterFallsBackToCfg verifies the legacy cfg-driven
// path stays untouched when the router has no healthy endpoints —
// guarantees zero behaviour change for installs that never configure
// the relay store.
func TestProxy_EmptyRouterFallsBackToCfg(t *testing.T) {
	hitCfg := int32(0)
	srv, reg, meter, upstream := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hitCfg, 1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":    "ok",
			"model": "x",
			"usage": map[string]int{"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2},
		})
	})
	defer upstream.Close()
	_ = meter

	// Wire a router with NO endpoints — Pick will fail, gateway must
	// fall back to cfg.UpstreamURL.
	dir := t.TempDir()
	store, err := relay.NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	router, err := relay.NewRouter(dir, store, relay.NewCircuitBreaker())
	if err != nil {
		t.Fatal(err)
	}
	srv.SetRelayRouter(router)

	app, err := reg.Register("X", "", "")
	if err != nil {
		t.Fatal(err)
	}

	body := `{"model":"x","messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+app.Token)
	req.Header.Set("Content-Type", "application/json")

	mux := http.NewServeMux()
	srv.registerRoutes(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("status=%d", w.Result().StatusCode)
	}
	if got := atomic.LoadInt32(&hitCfg); got != 1 {
		t.Fatalf("cfg upstream hit count = %d, want 1", got)
	}
}

// TestProxy_RouterChainCascadesOnFailure builds a 2-endpoint chain
// where the preferred endpoint returns 500; the gateway must
// transparently cascade to the second endpoint.
func TestProxy_RouterChainCascadesOnFailure(t *testing.T) {
	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"down"}`))
	}))
	defer failing.Close()

	hitBackup := int32(0)
	backup := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hitBackup, 1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":    "ok",
			"model": "x",
			"usage": map[string]int{"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2},
		})
	}))
	defer backup.Close()

	dir := t.TempDir()
	reg, _ := appreg.NewRegistry(dir)
	meter, _ := metering.NewStore(dir)

	store, err := relay.NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveEndpoint(relay.RelayEndpoint{
		ID: "primary", Name: "primary", URL: failing.URL, APIKey: "k1", Healthy: true, LatencyMs: 10,
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveEndpoint(relay.RelayEndpoint{
		ID: "backup", Name: "backup", URL: backup.URL, APIKey: "k2", Healthy: true, LatencyMs: 20,
	}); err != nil {
		t.Fatal(err)
	}
	router, err := relay.NewRouter(dir, store, relay.NewCircuitBreaker())
	if err != nil {
		t.Fatal(err)
	}
	if err := router.LoadRulesYAML(`
rules:
  - name: pin-primary
    match_model_prefix: x
    prefer_endpoint_id: primary
`); err != nil {
		t.Fatal(err)
	}

	srv := NewServer(dir, reg, meter)
	srv.cfg.UpstreamURL = "http://ignored.invalid"
	srv.cfg.UserToken = "user-token"
	srv.SetRelayRouter(router)
	app, _ := reg.Register("X", "", "")

	body := `{"model":"x","messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+app.Token)
	req.Header.Set("Content-Type", "application/json")
	mux := http.NewServeMux()
	srv.registerRoutes(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(w.Result().Body)
		t.Fatalf("status=%d body=%s", w.Result().StatusCode, respBody)
	}
	if got := atomic.LoadInt32(&hitBackup); got != 1 {
		t.Fatalf("backup endpoint hit count = %d, want 1", got)
	}
}

// TestProxy_RouterChainCascadesOn401 mirrors the 500 cascade test for the
// upstream-ban case: the preferred endpoint returns 401 (key revoked /
// banned), and the gateway must roll over to the backup instead of handing
// the 401 back to the caller. This is the reseller-critical path the
// shouldFallback 401/403/402 change unlocks.
func TestProxy_RouterChainCascadesOn401(t *testing.T) {
	banned := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"key banned"}`))
	}))
	defer banned.Close()

	hitBackup := int32(0)
	backup := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hitBackup, 1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":    "ok",
			"model": "x",
			"usage": map[string]int{"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2},
		})
	}))
	defer backup.Close()

	dir := t.TempDir()
	reg, _ := appreg.NewRegistry(dir)
	meter, _ := metering.NewStore(dir)

	store, err := relay.NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveEndpoint(relay.RelayEndpoint{
		ID: "primary", Name: "primary", URL: banned.URL, APIKey: "k1", Healthy: true, LatencyMs: 10,
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveEndpoint(relay.RelayEndpoint{
		ID: "backup", Name: "backup", URL: backup.URL, APIKey: "k2", Healthy: true, LatencyMs: 20,
	}); err != nil {
		t.Fatal(err)
	}
	router, err := relay.NewRouter(dir, store, relay.NewCircuitBreaker())
	if err != nil {
		t.Fatal(err)
	}
	if err := router.LoadRulesYAML(`
rules:
  - name: pin-primary
    match_model_prefix: x
    prefer_endpoint_id: primary
`); err != nil {
		t.Fatal(err)
	}

	srv := NewServer(dir, reg, meter)
	srv.cfg.UpstreamURL = "http://ignored.invalid"
	srv.cfg.UserToken = "user-token"
	srv.SetRelayRouter(router)
	app, _ := reg.Register("X", "", "")

	body := `{"model":"x","messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+app.Token)
	req.Header.Set("Content-Type", "application/json")
	mux := http.NewServeMux()
	srv.registerRoutes(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(w.Result().Body)
		t.Fatalf("status=%d body=%s", w.Result().StatusCode, respBody)
	}
	if got := atomic.LoadInt32(&hitBackup); got != 1 {
		t.Fatalf("backup endpoint hit count = %d, want 1 (401 primary should cascade)", got)
	}
}
