package topology

import (
	"testing"
	"time"
)

func TestCompose_PersonalAllOK(t *testing.T) {
	now := time.Date(2026, 5, 12, 10, 0, 0, 0, time.UTC)
	in := ComposeInput{
		Mode: "personal",
		Tools: []ToolInput{
			{Name: "claude", Installed: true, Version: "1.2.3", Health: "green"},
			{Name: "codex", Installed: false},
		},
		Gateway: GatewayInput{Running: true, Port: 19090, URL: "http://localhost:19090"},
		Proxy:   ProxyInput{Configured: false},
		Hub:     HubInput{URL: "https://hub.lurus.cn", Reachable: true, LatencyMs: 80},
		Providers: []ProviderInput{
			{ID: "anthropic", Label: "Anthropic API", DNSOK: true, DirectOK: true, DirectMs: 120},
			{ID: "openai", Label: "OpenAI API", DNSOK: true, DirectOK: false},
		},
		Auth:         AuthInput{LoggedIn: true, HasGatewayToken: true, UserEmail: "u@x.com"},
		CurrentModel: "claude-sonnet-4-6",
	}
	snap := Compose(in, now)

	if snap.Mode != "personal" {
		t.Errorf("mode = %q, want personal", snap.Mode)
	}
	if !snap.GeneratedAt.Equal(now) {
		t.Errorf("generatedAt = %v, want %v", snap.GeneratedAt, now)
	}

	wantNodes := map[string]NodeStatus{
		"tool:claude":        StatusOK,
		"tool:codex":         StatusNotConfigured,
		"gateway":            StatusOK,
		"proxy":              StatusNotConfigured,
		"auth":               StatusOK,
		"hub":                StatusOK,
		"provider:anthropic": StatusOK,
		"provider:openai":    StatusDown,
	}
	for _, n := range snap.Nodes {
		want, ok := wantNodes[n.ID]
		if !ok {
			continue
		}
		if n.Status != want {
			t.Errorf("node %s status = %q, want %q", n.ID, n.Status, want)
		}
		delete(wantNodes, n.ID)
	}
	if len(wantNodes) > 0 {
		t.Errorf("missing nodes in snapshot: %v", wantNodes)
	}

	// Anthropic highlight on Claude model.
	for _, n := range snap.Nodes {
		if n.ID == "provider:anthropic" && !n.Highlight {
			t.Error("expected anthropic provider highlighted for claude-* model")
		}
	}
}

func TestCompose_EndUserActivationRevoked(t *testing.T) {
	in := ComposeInput{
		Mode:       "enduser",
		Tools:      []ToolInput{{Name: "claude", Installed: true, Health: "green"}},
		Gateway:    GatewayInput{Running: true, Port: 19090},
		Hub:        HubInput{URL: "https://hub.example.com", Reachable: true},
		Activation: ActivationInput{State: "revoked", TenantSlug: "acme"},
	}
	snap := Compose(in, time.Now())

	var auth *Node
	for i := range snap.Nodes {
		if snap.Nodes[i].ID == "auth" {
			auth = &snap.Nodes[i]
		}
	}
	if auth == nil {
		t.Fatal("auth node missing")
	}
	if auth.Status != StatusDown {
		t.Errorf("auth status = %q, want down", auth.Status)
	}
	if auth.Label == "" || auth.Hint == "" {
		t.Error("auth node should carry label + hint for revoked")
	}
}

func TestCompose_GatewayDownPropagates(t *testing.T) {
	in := ComposeInput{
		Mode:    "personal",
		Tools:   []ToolInput{{Name: "claude", Installed: true, Health: "green"}},
		Gateway: GatewayInput{Running: false, Port: 19090},
		Hub:     HubInput{URL: "https://hub.lurus.cn", Reachable: true},
		Auth:    AuthInput{LoggedIn: true, HasGatewayToken: true},
	}
	snap := Compose(in, time.Now())

	// The tool→gateway edge must be down when gateway is stopped.
	foundEdge := false
	for _, e := range snap.Edges {
		if e.From == "tool:claude" && e.To == "gateway" {
			foundEdge = true
			if e.Status != StatusDown {
				t.Errorf("tool→gateway edge status = %q, want down", e.Status)
			}
		}
	}
	if !foundEdge {
		t.Error("tool→gateway edge missing")
	}

	// Headline must point at the broken thing.
	if snap.Summary.Headline == "" {
		t.Error("expected headline to call out broken gateway")
	}
}

func TestCompose_ProxyReachableButGatewayConsumes(t *testing.T) {
	reach := true
	in := ComposeInput{
		Mode:    "personal",
		Gateway: GatewayInput{Running: true, Port: 19090},
		Proxy:   ProxyInput{Configured: true, Enabled: true, URL: "socks5://127.0.0.1:1080", Reachable: &reach, LatencyMs: 12},
		Hub:     HubInput{URL: "https://hub.lurus.cn", Reachable: true},
		Auth:    AuthInput{LoggedIn: true, HasGatewayToken: true},
	}
	snap := Compose(in, time.Now())

	// The hub edge should originate from proxy, not gateway, when proxy is enabled.
	var hubEdge *Edge
	for i := range snap.Edges {
		if snap.Edges[i].To == "hub" {
			hubEdge = &snap.Edges[i]
		}
	}
	if hubEdge == nil {
		t.Fatal("hub edge missing")
	}
	if hubEdge.From != "proxy" {
		t.Errorf("hub edge from = %q, want proxy", hubEdge.From)
	}
}

func TestCompose_ProviderNeedsUpstream(t *testing.T) {
	in := ComposeInput{
		Mode: "personal",
		Providers: []ProviderInput{
			{ID: "anthropic", Label: "Anthropic API", DNSOK: true, DirectOK: false, UpstreamTried: true, UpstreamOK: true, UpstreamMs: 200},
		},
	}
	snap := Compose(in, time.Now())
	for _, n := range snap.Nodes {
		if n.ID != "provider:anthropic" {
			continue
		}
		if n.Status != StatusDegraded {
			t.Errorf("status = %q, want degraded (only reachable via upstream)", n.Status)
		}
		if n.LatencyMs != 200 {
			t.Errorf("latency = %d, want 200 (upstream path)", n.LatencyMs)
		}
	}
}
