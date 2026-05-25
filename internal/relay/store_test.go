package relay

import "testing"

// TestStore_UpdateEndpointLatency verifies the latency feedback loop
// added in W3.2: the gateway's fallback observer calls this after every
// successful upstream attempt so Pick()'s ascending-latency sort
// reflects live traffic, not just manual health checks.
func TestStore_UpdateEndpointLatency(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveEndpoint(RelayEndpoint{
		ID: "alpha", URL: "https://alpha.test", LatencyMs: 999,
	}); err != nil {
		t.Fatal(err)
	}

	if err := store.UpdateEndpointLatency("alpha", 42); err != nil {
		t.Fatal(err)
	}

	eps, err := store.ListEndpoints()
	if err != nil {
		t.Fatal(err)
	}
	var got *RelayEndpoint
	for i := range eps {
		if eps[i].ID == "alpha" {
			got = &eps[i]
			break
		}
	}
	if got == nil {
		t.Fatal("alpha not in store after update")
	}
	if got.LatencyMs != 42 {
		t.Errorf("LatencyMs = %d, want 42", got.LatencyMs)
	}
	if !got.Healthy {
		t.Errorf("Healthy should be true after successful update")
	}
	if got.LastChecked == "" {
		t.Errorf("LastChecked should be populated")
	}
}

// TestStore_UpdateEndpointLatency_UnknownIDIsNoop guards against the
// observer firing for built-in endpoints (whose ID won't match any
// user-defined record): the call must silently no-op, not error.
func TestStore_UpdateEndpointLatency_UnknownIDIsNoop(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateEndpointLatency("lurus-api", 42); err != nil {
		t.Errorf("update on builtin should not error: %v", err)
	}
	if err := store.UpdateEndpointLatency("", 42); err != nil {
		t.Errorf("update with empty id should not error: %v", err)
	}
}

// TestStore_MigrateLegacyFallbacks_Idempotent verifies that running the
// W4.2 migration twice has the same effect as running it once: existing
// user endpoints block the second run so we never double-import.
func TestStore_MigrateLegacyFallbacks_Idempotent(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	legacy := []LegacyFallback{
		{Name: "Groq-Free", URL: "https://api.groq.com", Token: "gk-1"},
		{Name: "DeepSeek", URL: "https://api.deepseek.com", Token: "dk-1"},
	}

	n, err := store.MigrateLegacyFallbacks(legacy)
	if err != nil {
		t.Fatalf("first migrate: %v", err)
	}
	if n != 2 {
		t.Fatalf("first migrate added = %d, want 2", n)
	}

	// Second run must skip — any user endpoint blocks repeat migration.
	n2, err := store.MigrateLegacyFallbacks(legacy)
	if err != nil {
		t.Fatalf("second migrate: %v", err)
	}
	if n2 != 0 {
		t.Fatalf("second migrate added = %d, want 0 (idempotent)", n2)
	}

	eps, err := store.ListEndpoints()
	if err != nil {
		t.Fatal(err)
	}
	// 1 builtin (lurus-api) + 2 migrated = 3 total.
	if got := len(eps); got != 3 {
		t.Fatalf("endpoint count = %d, want 3 (1 builtin + 2 migrated)", got)
	}

	gotURLs := map[string]bool{}
	for _, ep := range eps {
		gotURLs[ep.URL] = true
	}
	for _, want := range []string{"https://api.groq.com", "https://api.deepseek.com"} {
		if !gotURLs[want] {
			t.Errorf("migrated endpoint URL %q missing from store", want)
		}
	}
}

// TestStore_MigrateLegacyFallbacks_EmptyAndNoop verifies the two no-op
// branches: passing nil/empty input, and passing a non-empty list while
// the store already has user endpoints from manual edits.
func TestStore_MigrateLegacyFallbacks_EmptyAndNoop(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	n, err := store.MigrateLegacyFallbacks(nil)
	if err != nil || n != 0 {
		t.Fatalf("nil input: got (%d, %v), want (0, nil)", n, err)
	}

	if err := store.SaveEndpoint(RelayEndpoint{ID: "ep1", Name: "manual", URL: "https://x.test"}); err != nil {
		t.Fatal(err)
	}
	n2, err := store.MigrateLegacyFallbacks([]LegacyFallback{
		{Name: "Skip", URL: "https://skip.test"},
	})
	if err != nil || n2 != 0 {
		t.Fatalf("with existing user endpoint: got (%d, %v), want (0, nil)", n2, err)
	}
}
