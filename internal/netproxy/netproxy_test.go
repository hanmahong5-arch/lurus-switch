package netproxy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestBuildTransport_Disabled(t *testing.T) {
	_, err := BuildTransport(Settings{Enabled: false})
	if err == nil {
		t.Fatal("expected error for empty URL")
	}
}

func TestBuildTransport_BadScheme(t *testing.T) {
	_, err := BuildTransport(Settings{Enabled: true, URL: "ftp://example.com:21"})
	if err == nil || !strings.Contains(err.Error(), "unsupported scheme") {
		t.Fatalf("expected unsupported scheme error, got %v", err)
	}
}

func TestBuildTransport_BadURL(t *testing.T) {
	_, err := BuildTransport(Settings{Enabled: true, URL: "://nohost"})
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestBuildTransport_HTTP_ProxyFuncSet(t *testing.T) {
	tr, err := BuildTransport(Settings{Enabled: true, URL: "http://127.0.0.1:7890"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tr.Proxy == nil {
		t.Fatal("expected Proxy func to be set for http scheme")
	}
	req, _ := http.NewRequest("GET", "https://api.anthropic.com/v1/messages", nil)
	u, err := tr.Proxy(req)
	if err != nil {
		t.Fatalf("proxy func returned error: %v", err)
	}
	if u == nil || u.Host != "127.0.0.1:7890" {
		t.Fatalf("expected proxy host 127.0.0.1:7890, got %v", u)
	}
}

func TestBuildTransport_SOCKS5_DialContextSet(t *testing.T) {
	tr, err := BuildTransport(Settings{Enabled: true, URL: "socks5://127.0.0.1:1080"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tr.DialContext == nil {
		t.Fatal("expected DialContext to be set for socks5")
	}
	if tr.Proxy != nil {
		t.Fatal("expected Proxy func to be nil for socks5 (double-routing risk)")
	}
}

func TestCompileBypass_LoopbackAlways(t *testing.T) {
	b := compileBypass("")
	for _, h := range []string{"localhost", "127.0.0.1", "127.0.0.5", "::1"} {
		if !b(h) {
			t.Errorf("loopback %q should bypass", h)
		}
	}
	if b("api.anthropic.com") {
		t.Error("non-loopback should not bypass with empty list")
	}
}

func TestCompileBypass_SuffixMatch(t *testing.T) {
	b := compileBypass("lurus.cn, anthropic.com")
	cases := map[string]bool{
		"hub.lurus.cn":         true,
		"api.lurus.cn":         true,
		"lurus.cn":             true,
		"anthropic.com":        true,
		"api.anthropic.com":    true,
		"notlurus.cn":          false,
		"google.com":           false,
		"localhost":            true,
		"127.0.0.1":            true,
	}
	for h, want := range cases {
		if got := b(h); got != want {
			t.Errorf("bypass(%q) = %v, want %v", h, got, want)
		}
	}
}

func TestCompileBypass_StripsPort(t *testing.T) {
	b := compileBypass("anthropic.com")
	if !b("api.anthropic.com:443") {
		t.Error("expected host:port form to match suffix")
	}
}

func TestApply_DisableRestoresDefault(t *testing.T) {
	originalDefault.Store(nil)
	prev := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = prev })

	if err := Apply(Settings{Enabled: true, URL: "http://127.0.0.1:7890"}); err != nil {
		t.Fatalf("apply enabled: %v", err)
	}
	if http.DefaultTransport == prev {
		t.Fatal("expected default transport to change on enable")
	}
	if err := Apply(Settings{Enabled: false}); err != nil {
		t.Fatalf("apply disabled: %v", err)
	}
	if http.DefaultTransport != prev {
		t.Fatal("expected default transport restored on disable")
	}
}

func TestApply_BadConfigDoesNotMutate(t *testing.T) {
	originalDefault.Store(nil)
	prev := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = prev })

	if err := Apply(Settings{Enabled: true, URL: "ftp://nope"}); err == nil {
		t.Fatal("expected error on bad scheme")
	}
	if http.DefaultTransport != prev {
		t.Fatal("default transport must be unchanged on error")
	}
}

// Integration: real HTTP proxy via httptest. Verifies the request
// actually flows through the configured proxy.
func TestTest_GoesThroughProxy(t *testing.T) {
	var hit bool
	proxySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		w.WriteHeader(204)
	}))
	defer proxySrv.Close()

	// Use proxySrv as both the proxy and the upstream — for http URLs
	// the transport sends the absolute URL to the proxy, which our
	// httptest server handles as a normal request.
	target, _ := url.Parse(proxySrv.URL + "/generate_204")
	res := Test(context.Background(), Settings{
		Enabled: true,
		URL:     proxySrv.URL,
		TestURL: target.String(),
	})
	if !hit {
		t.Fatal("proxy server was not contacted")
	}
	if !res.OK {
		t.Fatalf("expected OK, got %+v", res)
	}
	if res.StatusCode != 204 {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
}

func TestTest_DisabledReturnsError(t *testing.T) {
	res := Test(context.Background(), Settings{Enabled: false})
	if res.OK || res.Error == "" {
		t.Fatalf("expected disabled-state error, got %+v", res)
	}
}
