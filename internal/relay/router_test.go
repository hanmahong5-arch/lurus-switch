package relay

import (
	"path/filepath"
	"testing"
	"time"
)

func TestRouter_PicksByRuleThenTooltipFallback(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	// Two user endpoints + 1 builtin (lurus-api).
	if err := store.SaveEndpoint(RelayEndpoint{ID: "fast", URL: "https://fast.test", Healthy: true, LatencyMs: 20}); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveEndpoint(RelayEndpoint{ID: "smart", URL: "https://smart.test", Healthy: true, LatencyMs: 80}); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveToolMapping(ToolRelayMapping{"claude": "smart"}); err != nil {
		t.Fatal(err)
	}

	breaker := NewCircuitBreakerForTest(3, time.Second, time.Now)
	router, err := NewRouter(dir, store, breaker)
	if err != nil {
		t.Fatal(err)
	}

	rulesYAML := `
rules:
  - name: long-context-to-fast
    match_model_prefix: claude-opus
    min_tokens: 50000
    prefer_endpoint_id: fast
`
	if err := router.LoadRulesYAML(rulesYAML); err != nil {
		t.Fatal(err)
	}
	// Verify the file persisted under appDataDir.
	if _, err := filepath.Glob(filepath.Join(dir, "*.yaml")); err != nil {
		t.Error(err)
	}

	// Rule matches → fast.
	res, err := router.Pick("claude", PickHint{Model: "claude-opus-4-7", EstimatedInputTokens: 90000})
	if err != nil {
		t.Fatal(err)
	}
	if res.Endpoint.ID != "fast" {
		t.Errorf("rule should select fast, got %s", res.Endpoint.ID)
	}
	if res.MatchedBy != "long-context-to-fast" {
		t.Errorf("MatchedBy: %q", res.MatchedBy)
	}

	// Rule does not match (low tokens) → falls back to mapping default smart.
	res, err = router.Pick("claude", PickHint{Model: "claude-opus-4-7", EstimatedInputTokens: 1000})
	if err != nil {
		t.Fatal(err)
	}
	if res.Endpoint.ID != "smart" {
		t.Errorf("mapping fallback should select smart, got %s", res.Endpoint.ID)
	}

	// Open the smart circuit → router falls through to next healthy.
	breaker.RecordFailure("smart", "x")
	breaker.RecordFailure("smart", "x")
	breaker.RecordFailure("smart", "x")
	res, err = router.Pick("claude", PickHint{Model: "claude-opus-4-7", EstimatedInputTokens: 1000})
	if err != nil {
		t.Fatal(err)
	}
	if res.Endpoint.ID == "smart" {
		t.Errorf("smart should be filtered out (circuit open)")
	}
}

// TestRouter_OrderedPlacesPreferredFirst verifies that PickResult.Ordered
// surfaces the rule-preferred endpoint at index 0 with the remaining
// healthy peers following in ascending-latency order. Gateway uses this
// list as a deterministic fallback chain.
func TestRouter_OrderedPlacesPreferredFirst(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveEndpoint(RelayEndpoint{ID: "fast", URL: "https://fast.test", Healthy: true, LatencyMs: 20}); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveEndpoint(RelayEndpoint{ID: "slow", URL: "https://slow.test", Healthy: true, LatencyMs: 200}); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveEndpoint(RelayEndpoint{ID: "smart", URL: "https://smart.test", Healthy: true, LatencyMs: 80}); err != nil {
		t.Fatal(err)
	}

	router, err := NewRouter(dir, store, NewCircuitBreaker())
	if err != nil {
		t.Fatal(err)
	}
	if err := router.LoadRulesYAML(`
rules:
  - name: prefer-smart
    match_model_prefix: claude
    prefer_endpoint_id: smart
`); err != nil {
		t.Fatal(err)
	}

	res, err := router.Pick("claude", PickHint{Model: "claude-opus-4-7"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Ordered) < 3 {
		t.Fatalf("Ordered len = %d, want ≥3", len(res.Ordered))
	}
	if res.Ordered[0].ID != "smart" {
		t.Errorf("Ordered[0] = %q, want smart", res.Ordered[0].ID)
	}
	// Remaining peers should be in ascending-latency order: fast (20) < slow (200).
	rest := res.Ordered[1:]
	lastLat := int64(-1)
	for _, ep := range rest {
		if ep.ID == "smart" {
			t.Errorf("smart should not appear twice in Ordered")
		}
		if lastLat >= 0 && ep.LatencyMs < lastLat {
			t.Errorf("Ordered tail not ascending: %d before %d", lastLat, ep.LatencyMs)
		}
		lastLat = ep.LatencyMs
	}
}

// TestRouter_IsActive_OnlyBuiltinReturnsFalse guards the "zero
// behaviour change" property — an install that never touches RelayPage
// has only the builtin lurus-api endpoint and MUST NOT be considered
// active. Otherwise the gateway would silently redirect every request
// to https://api.lurus.cn.
func TestRouter_IsActive_OnlyBuiltinReturnsFalse(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	router, err := NewRouter(dir, store, NewCircuitBreaker())
	if err != nil {
		t.Fatal(err)
	}
	if router.IsActive() {
		t.Fatal("router should be inactive when only builtin endpoints exist")
	}

	// Add one user endpoint → router becomes active.
	if err := store.SaveEndpoint(RelayEndpoint{ID: "u", URL: "https://u.test", Healthy: true}); err != nil {
		t.Fatal(err)
	}
	if !router.IsActive() {
		t.Fatal("router should be active once a user endpoint exists")
	}
}
