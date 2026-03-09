package proxydetect

import (
	"runtime"
	"testing"
)

// clearAllProxyEnv sets all proxy env vars to empty
func clearAllProxyEnv(t *testing.T) {
	t.Helper()
	// On Windows, env vars are case-insensitive, so only set each logical var once
	if runtime.GOOS == "windows" {
		t.Setenv("HTTP_PROXY", "")
		t.Setenv("HTTPS_PROXY", "")
		t.Setenv("ALL_PROXY", "")
	} else {
		t.Setenv("HTTP_PROXY", "")
		t.Setenv("http_proxy", "")
		t.Setenv("HTTPS_PROXY", "")
		t.Setenv("https_proxy", "")
		t.Setenv("ALL_PROXY", "")
		t.Setenv("all_proxy", "")
	}
}

func TestDetectEnvVars_HTTPProxy(t *testing.T) {
	clearAllProxyEnv(t)
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:8080")

	results := detectEnvVars()
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %+v", len(results), results)
	}
	r := results[0]
	if r.Source != "env" {
		t.Errorf("expected source=env, got %s", r.Source)
	}
	if r.Host != "127.0.0.1" || r.Port != 8080 {
		t.Errorf("expected 127.0.0.1:8080, got %s:%d", r.Host, r.Port)
	}
	if r.Type != "http" {
		t.Errorf("expected type=http, got %s", r.Type)
	}
}

func TestDetectEnvVars_Socks5(t *testing.T) {
	clearAllProxyEnv(t)
	t.Setenv("ALL_PROXY", "socks5://localhost:1080")

	results := detectEnvVars()
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %+v", len(results), results)
	}
	r := results[0]
	if r.Type != "socks5" {
		t.Errorf("expected type=socks5, got %s", r.Type)
	}
}

func TestDetectEnvVars_Dedup(t *testing.T) {
	clearAllProxyEnv(t)
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:7890")
	if runtime.GOOS != "windows" {
		// On non-Windows, set lowercase too to test dedup
		t.Setenv("http_proxy", "http://127.0.0.1:7890")
	}

	results := detectEnvVars()
	if len(results) != 1 {
		t.Errorf("expected dedup to 1 result, got %d", len(results))
	}
}

func TestDetectEnvVars_Empty(t *testing.T) {
	clearAllProxyEnv(t)

	results := detectEnvVars()
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty env, got %d", len(results))
	}
}

func TestParseProxyURL_Valid(t *testing.T) {
	tests := []struct {
		raw  string
		host string
		port int
		typ  string
	}{
		{"http://127.0.0.1:7890", "127.0.0.1", 7890, "http"},
		{"socks5://localhost:1080", "localhost", 1080, "socks5"},
		{"127.0.0.1:8080", "127.0.0.1", 8080, "http"},
		{"https://proxy.example.com:443", "proxy.example.com", 443, "http"},
	}

	for _, tt := range tests {
		p, ok := parseProxyURL(tt.raw, "test")
		if !ok {
			t.Errorf("parseProxyURL(%q) returned !ok", tt.raw)
			continue
		}
		if p.Host != tt.host {
			t.Errorf("parseProxyURL(%q).Host = %s, want %s", tt.raw, p.Host, tt.host)
		}
		if p.Port != tt.port {
			t.Errorf("parseProxyURL(%q).Port = %d, want %d", tt.raw, p.Port, tt.port)
		}
		if p.Type != tt.typ {
			t.Errorf("parseProxyURL(%q).Type = %s, want %s", tt.raw, p.Type, tt.typ)
		}
	}
}

func TestParseProxyURL_Invalid(t *testing.T) {
	invalids := []string{"", "   ", "not-a-url", "http://", "http://host-no-port"}
	for _, raw := range invalids {
		_, ok := parseProxyURL(raw, "test")
		if ok {
			t.Errorf("parseProxyURL(%q) should return !ok", raw)
		}
	}
}

func TestDetectAll_Dedup(t *testing.T) {
	clearAllProxyEnv(t)
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:59999")

	results := DetectAll()
	count := 0
	for _, r := range results {
		if r.URL == "http://127.0.0.1:59999" {
			count++
		}
	}
	if count > 1 {
		t.Errorf("expected at most 1 entry for 59999, got %d", count)
	}
}
