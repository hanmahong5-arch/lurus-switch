package bifrost_test

import (
	"testing"

	"github.com/maximhq/bifrost/core/schemas"

	bfadapter "lurus-switch/internal/proxy/bifrost"
)

// ----- ProviderKey derivation -----

func TestProviderKey_DerivesSwitchPrefixedKey(t *testing.T) {
	got := bfadapter.ProviderKey("lurus-api")
	want := schemas.ModelProvider("switch-lurus-api")
	if got != want {
		t.Errorf("ProviderKey = %q, want %q", got, want)
	}
}

func TestProviderKey_EmptyID_ReturnsSwitchPrefix(t *testing.T) {
	got := bfadapter.ProviderKey("")
	want := schemas.ModelProvider("switch-")
	if got != want {
		t.Errorf("ProviderKey(\"\") = %q, want %q", got, want)
	}
}

// ----- Disabled adapter (opt-in guard) -----

func TestNew_Disabled_ReturnsNil(t *testing.T) {
	a, err := bfadapter.New(nil, bfadapter.Config{Enabled: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a != nil {
		t.Fatal("expected nil adapter when disabled")
	}
}

func TestNew_Disabled_NilAdapter_EnabledReturnsFalse(t *testing.T) {
	var a *bfadapter.Adapter
	if a.Enabled() {
		t.Fatal("nil adapter must report Enabled=false")
	}
}

// ----- Enabled adapter construction -----

func TestNew_Enabled_NoEndpoints_Succeeds(t *testing.T) {
	a, err := bfadapter.New(nil, bfadapter.Config{Enabled: true})
	if err != nil {
		t.Fatalf("expected no error for empty endpoint list, got: %v", err)
	}
	if a == nil {
		t.Fatal("expected non-nil adapter")
	}
	if !a.Enabled() {
		t.Fatal("adapter should report Enabled=true")
	}
	a.Shutdown()
}

func TestNew_Enabled_SingleEndpoint_Succeeds(t *testing.T) {
	eps := []bfadapter.EndpointInfo{
		{ID: "ep-1", URL: "https://api.example.com", APIKey: "sk-test"},
	}
	a, err := bfadapter.New(eps, bfadapter.Config{Enabled: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !a.Enabled() {
		t.Fatal("expected Enabled=true")
	}
	a.Shutdown()
}

func TestNew_Enabled_MultipleEndpoints_Succeeds(t *testing.T) {
	eps := []bfadapter.EndpointInfo{
		{ID: "ep-a", URL: "https://api-a.example.com", APIKey: "key-a"},
		{ID: "ep-b", URL: "https://api-b.example.com", APIKey: "key-b"},
		{ID: "ep-c", URL: "https://api-c.example.com", APIKey: ""},
	}
	a, err := bfadapter.New(eps, bfadapter.Config{Enabled: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	a.Shutdown()
}

// ----- BuildFallbacks -----

func TestBuildFallbacks_EmptyOrdered_ReturnsEmptyPrimary(t *testing.T) {
	a, _ := bfadapter.New(nil, bfadapter.Config{Enabled: true})
	defer a.Shutdown()

	primary, fallbacks := a.BuildFallbacks(nil, "gpt-4")
	if primary != "" {
		t.Errorf("expected empty primary for nil ordered, got %q", primary)
	}
	if len(fallbacks) != 0 {
		t.Errorf("expected 0 fallbacks, got %d", len(fallbacks))
	}
}

func TestBuildFallbacks_SingleEndpoint_NoPeerFallbacks(t *testing.T) {
	a, _ := bfadapter.New(nil, bfadapter.Config{Enabled: true})
	defer a.Shutdown()

	ordered := []bfadapter.EndpointInfo{
		{ID: "only", URL: "https://only.example.com", APIKey: "k"},
	}
	primary, fallbacks := a.BuildFallbacks(ordered, "claude-3-5-haiku-20241022")
	if primary != bfadapter.ProviderKey("only") {
		t.Errorf("primary = %q, want %q", primary, bfadapter.ProviderKey("only"))
	}
	if len(fallbacks) != 0 {
		t.Errorf("expected 0 fallbacks for single endpoint, got %d", len(fallbacks))
	}
}

func TestBuildFallbacks_ThreeEndpoints_ChainPreservesOrder(t *testing.T) {
	a, _ := bfadapter.New(nil, bfadapter.Config{Enabled: true})
	defer a.Shutdown()

	model := "gpt-4o"
	ordered := []bfadapter.EndpointInfo{
		{ID: "primary", URL: "https://p.example.com"},
		{ID: "secondary", URL: "https://s.example.com"},
		{ID: "tertiary", URL: "https://t.example.com"},
	}
	primary, fallbacks := a.BuildFallbacks(ordered, model)

	if primary != bfadapter.ProviderKey("primary") {
		t.Errorf("primary = %q", primary)
	}
	if len(fallbacks) != 2 {
		t.Fatalf("want 2 fallbacks, got %d", len(fallbacks))
	}
	if fallbacks[0].Provider != bfadapter.ProviderKey("secondary") {
		t.Errorf("fallbacks[0].Provider = %q", fallbacks[0].Provider)
	}
	if fallbacks[1].Provider != bfadapter.ProviderKey("tertiary") {
		t.Errorf("fallbacks[1].Provider = %q", fallbacks[1].Provider)
	}
	// Every fallback carries the same model as the primary request.
	for i, fb := range fallbacks {
		if fb.Model != model {
			t.Errorf("fallbacks[%d].Model = %q, want %q", i, fb.Model, model)
		}
	}
}

// ----- Shutdown idempotence -----

func TestShutdown_IdempotentOnNilAdapter(t *testing.T) {
	// Must not panic
	var a *bfadapter.Adapter
	a.Shutdown()
	a.Shutdown()
}

func TestShutdown_DoubleShutdown_NoPanic(t *testing.T) {
	a, _ := bfadapter.New(nil, bfadapter.Config{Enabled: true})
	a.Shutdown()
	a.Shutdown() // second shutdown must not panic
}

// ----- Config defaults -----

func TestConfig_ZeroMaxRetries_DefaultsApplied(t *testing.T) {
	// Construct with zero MaxRetries — should not fail (defaults kick in).
	a, err := bfadapter.New(
		[]bfadapter.EndpointInfo{{ID: "x", URL: "https://x.example.com"}},
		bfadapter.Config{Enabled: true, MaxRetries: 0},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	a.Shutdown()
}

func TestConfig_ZeroTimeout_DefaultsApplied(t *testing.T) {
	a, err := bfadapter.New(
		[]bfadapter.EndpointInfo{{ID: "y", URL: "https://y.example.com"}},
		bfadapter.Config{Enabled: true, RequestTimeoutSec: 0},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	a.Shutdown()
}
