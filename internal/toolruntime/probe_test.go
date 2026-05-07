package toolruntime

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClassifyEndpoint(t *testing.T) {
	cases := []struct {
		url     string
		gwPort  int
		want    string
	}{
		{"https://api.anthropic.com", 0, "official"},
		{"https://api.openai.com/v1", 0, "official"},
		{"https://generativelanguage.googleapis.com", 0, "official"},
		{"http://localhost:19090", 19090, "lurus-gateway"},
		{"http://127.0.0.1:8080", 0, "lurus-gateway"},
		{"https://attacker.example/v1", 0, "third-party"},
		{"", 0, "unknown"},
		{"not-a-url", 0, "unknown"},
	}
	for _, c := range cases {
		got := classifyEndpoint(c.url, c.gwPort)
		if got != c.want {
			t.Errorf("classifyEndpoint(%q, %d) = %s; want %s", c.url, c.gwPort, got, c.want)
		}
	}
}

func TestProbeEndpoint_Reachable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()
	client := srv.Client()
	state, _, errMsg := probeEndpoint(t.Context(), client, srv.URL)
	if state != ConnReachable {
		t.Errorf("state=%s, want reachable; err=%s", state, errMsg)
	}
}

func TestProbeEndpoint_5xxIsDegraded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(503)
	}))
	defer srv.Close()
	state, _, _ := probeEndpoint(t.Context(), srv.Client(), srv.URL)
	if state != ConnDegraded {
		t.Errorf("state=%s, want degraded", state)
	}
}

func TestProbeEndpoint_Unreachable(t *testing.T) {
	state, _, _ := probeEndpoint(t.Context(), &http.Client{}, "http://127.0.0.1:1") // port 1 should be closed
	if state != ConnDown {
		t.Errorf("state=%s, want down", state)
	}
}

func TestTomlScalar(t *testing.T) {
	text := `[provider]
type = "anthropic"
api_key = "sk-secret"
model = "claude-sonnet-4-20250514"
base_url = "https://proxy.example/v1" # inline comment
`
	if got := tomlScalar(text, "model"); got != "claude-sonnet-4-20250514" {
		t.Errorf("model=%q", got)
	}
	if got := tomlScalar(text, "base_url"); got != "https://proxy.example/v1" {
		t.Errorf("base_url=%q", got)
	}
	if got := tomlScalar(text, "missing"); got != "" {
		t.Errorf("missing=%q, want empty", got)
	}
}
