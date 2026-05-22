package connectivity

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"lurus-switch/internal/netproxy"
)

func httpStatus(code int) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(code)
	}
}

func TestRun_AllOKWhenServersUp(t *testing.T) {
	srv := httptest.NewServer(httpStatus(200))
	defer srv.Close()

	providers := []Provider{
		{ID: "x", Label: "X", URL: srv.URL + "/", Tier: "ai"},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	r := Run(ctx, providers, nil)

	if len(r.Providers) != 1 || !r.Providers[0].DirectOK {
		t.Fatalf("expected direct probe OK, got %+v", r.Providers)
	}
	if len(r.Suggestions) == 0 || r.Suggestions[0].Kind != SuggestAllOK {
		t.Fatalf("expected SuggestAllOK, got %+v", r.Suggestions)
	}
}

func TestRun_BrokenAIRecommendsLurus(t *testing.T) {
	lurus := httptest.NewServer(httpStatus(200))
	defer lurus.Close()

	providers := []Provider{
		{ID: "anthropic", Label: "A", URL: "http://10.255.255.1:9/", Tier: "ai"},
		{ID: "lurus", Label: "L", URL: lurus.URL + "/", Tier: "lurus"},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	r := Run(ctx, providers, nil)

	if r.Providers[0].DirectOK {
		t.Fatal("expected unroutable provider to fail")
	}
	if !r.Providers[1].DirectOK {
		t.Fatal("expected lurus to succeed")
	}
	if !containsKind(r.Suggestions, SuggestUseLurusRelay) {
		t.Fatalf("expected SuggestUseLurusRelay, got %+v", r.Suggestions)
	}
}

func TestRun_BrokenEverythingRecommendsUpstream(t *testing.T) {
	providers := []Provider{
		{ID: "anthropic", Label: "A", URL: "http://10.255.255.1:9/", Tier: "ai"},
		{ID: "lurus", Label: "L", URL: "http://10.255.255.2:9/", Tier: "lurus"},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	r := Run(ctx, providers, nil)

	if !containsKind(r.Suggestions, SuggestUseUpstream) && !containsKind(r.Suggestions, SuggestAutoFillProxy) {
		t.Fatalf("expected upstream-related suggestion, got %+v", r.Suggestions)
	}
}

func TestRun_UpstreamTriedWhenEnabled(t *testing.T) {
	proxy := httptest.NewServer(httpStatus(200))
	defer proxy.Close()

	providers := []Provider{
		{ID: "x", Label: "X", URL: "http://10.255.255.1:9/", Tier: "ai"},
	}
	up := &netproxy.Settings{Enabled: true, URL: proxy.URL}

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	r := Run(ctx, providers, up)

	if r.Providers[0].DirectOK {
		t.Fatal("expected direct to fail (unroutable)")
	}
	if !r.Providers[0].UpstreamTried {
		t.Fatal("expected upstream probe to have been attempted")
	}
}

func TestDetectLocalProxies_FindsOpenLoopbackPort(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:8118")
	if err != nil {
		t.Skipf("port 8118 not available on this machine: %v", err)
	}
	defer ln.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	found := detectLocalProxies(ctx)
	var hit bool
	for _, lp := range found {
		if lp.Port == 8118 {
			hit = true
			if !strings.HasPrefix(lp.URL, "http://") && !strings.HasPrefix(lp.URL, "socks5://") {
				t.Errorf("unexpected URL: %q", lp.URL)
			}
		}
	}
	if !hit {
		t.Fatalf("expected to detect listener on :8118, got %+v", found)
	}
}

func TestBuildSuggestions_AutoFillUsesFirstLocalProxy(t *testing.T) {
	r := Report{
		Providers: []ProviderResult{
			{Provider: Provider{ID: "anthropic", Tier: "ai"}, DirectOK: false},
			{Provider: Provider{ID: "lurus", Tier: "lurus"}, DirectOK: false},
		},
		LocalProxies: []LocalProxy{
			{Host: "127.0.0.1", Port: 18000, URL: "socks5://127.0.0.1:18000", GuessedName: "MasterDnsVPN"},
			{Host: "127.0.0.1", Port: 7890, URL: "http://127.0.0.1:7890", GuessedName: "Clash"},
		},
	}
	suggestions := buildSuggestions(r, nil)
	if !containsKind(suggestions, SuggestAutoFillProxy) {
		t.Fatalf("expected SuggestAutoFillProxy, got %+v", suggestions)
	}
	for _, s := range suggestions {
		if s.Kind == SuggestAutoFillProxy && s.Payload != "socks5://127.0.0.1:18000" {
			t.Errorf("expected MasterDnsVPN payload first, got %q", s.Payload)
		}
	}
}

func TestItoa(t *testing.T) {
	cases := map[int]string{0: "0", 1: "1", 18000: "18000", -42: "-42"}
	for in, want := range cases {
		if got := itoa(in); got != want {
			t.Errorf("itoa(%d) = %q, want %q", in, got, want)
		}
	}
}

func containsKind(list []Suggestion, k SuggestionKind) bool {
	for _, s := range list {
		if s.Kind == k {
			return true
		}
	}
	return false
}
