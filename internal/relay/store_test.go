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
