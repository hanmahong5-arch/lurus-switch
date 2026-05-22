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
