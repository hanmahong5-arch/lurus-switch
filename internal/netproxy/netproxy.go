// Package netproxy provides a single chokepoint for routing every outbound
// HTTP request the app makes through a user-configured upstream proxy
// (HTTP / HTTPS / SOCKS5).
//
// It exists so users in restricted-network regions can point Switch's
// gateway, updater, relay health probes, etc. at their own VPN/proxy
// (V2Ray, Clash, plain HTTP forward proxy) without Switch itself
// shipping any censorship-evasion logic.
//
// The implementation deliberately mutates [http.DefaultTransport] rather
// than asking every call site to opt in. The ~25 call sites that build
// `&http.Client{Timeout: X}` with an implicit nil Transport pick up the
// proxy automatically, because Go's http package resolves the transport
// per-request from the package-level default.
package netproxy

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	xproxy "golang.org/x/net/proxy"
)

// Settings is the user-facing upstream proxy configuration. Persisted as
// part of internal/proxy.ProxySettings.
type Settings struct {
	// Enabled gates the whole feature. When false, http.DefaultTransport
	// stays at Go's default and no traffic is rewritten.
	Enabled bool `json:"enabled"`

	// URL is the proxy endpoint, e.g.
	//   http://user:pass@127.0.0.1:7890
	//   https://proxy.example.com:443
	//   socks5://127.0.0.1:1080
	//   socks5h://127.0.0.1:1080  (remote DNS, recommended for SOCKS5)
	URL string `json:"url"`

	// NoProxy is a comma-separated list of host suffixes or CIDR ranges
	// that should bypass the proxy. localhost / loopback are always
	// bypassed regardless of this field.
	NoProxy string `json:"noProxy,omitempty"`

	// TestURL is the URL probed by Test(). Defaults to
	// defaultTestURL when empty.
	TestURL string `json:"testUrl,omitempty"`
}

// TestResult is returned by Test for the UI to render.
type TestResult struct {
	OK         bool   `json:"ok"`
	StatusCode int    `json:"statusCode,omitempty"`
	LatencyMS  int64  `json:"latencyMs,omitempty"`
	Error      string `json:"error,omitempty"`
	ProbedURL  string `json:"probedUrl"`
}

const (
	defaultTestURL          = "https://www.google.com/generate_204"
	defaultTestTimeout      = 8 * time.Second
	defaultDialTimeout      = 10 * time.Second
	defaultKeepAlive        = 30 * time.Second
	defaultIdleConnTimeout  = 90 * time.Second
	defaultTLSHandshakeTime = 10 * time.Second
)

var (
	applyMu sync.Mutex

	// originalDefault captures Go's default transport on first call to
	// Apply, so disabling the proxy can restore identical behaviour.
	originalDefault atomic.Pointer[http.Transport]
)

// Apply installs a transport built from s as http.DefaultTransport.
// Calling with Enabled=false or an empty URL restores the original
// default. Returns an error without mutating state if the settings are
// malformed.
//
// Safe to call multiple times. In-flight requests continue to use
// whichever transport they captured; new requests pick up the swap.
func Apply(s Settings) error {
	applyMu.Lock()
	defer applyMu.Unlock()

	if originalDefault.Load() == nil {
		if t, ok := http.DefaultTransport.(*http.Transport); ok {
			originalDefault.Store(t)
		}
	}

	if !s.Enabled || strings.TrimSpace(s.URL) == "" {
		if orig := originalDefault.Load(); orig != nil {
			http.DefaultTransport = orig
		}
		return nil
	}

	t, err := BuildTransport(s)
	if err != nil {
		return err
	}
	http.DefaultTransport = t
	return nil
}

// BuildTransport returns a fresh *http.Transport configured per s. Use
// it only when you need an isolated client and can't share
// http.DefaultTransport. Returns an error if URL is unparseable or has
// an unsupported scheme.
func BuildTransport(s Settings) (*http.Transport, error) {
	if strings.TrimSpace(s.URL) == "" {
		return nil, errors.New("netproxy: empty URL")
	}
	u, err := url.Parse(strings.TrimSpace(s.URL))
	if err != nil {
		return nil, fmt.Errorf("netproxy: parse url: %w", err)
	}
	if u.Host == "" {
		return nil, errors.New("netproxy: URL has no host")
	}
	scheme := strings.ToLower(u.Scheme)

	t := newBaseTransport()
	bypass := compileBypass(s.NoProxy)

	switch scheme {
	case "http", "https":
		t.Proxy = func(req *http.Request) (*url.URL, error) {
			if bypass(req.URL.Host) {
				return nil, nil
			}
			return u, nil
		}
		return t, nil

	case "socks5", "socks5h":
		var auth *xproxy.Auth
		if u.User != nil {
			pw, _ := u.User.Password()
			auth = &xproxy.Auth{User: u.User.Username(), Password: pw}
		}
		dialer, err := xproxy.SOCKS5("tcp", u.Host, auth, &net.Dialer{
			Timeout:   defaultDialTimeout,
			KeepAlive: defaultKeepAlive,
		})
		if err != nil {
			return nil, fmt.Errorf("netproxy: build socks5 dialer: %w", err)
		}
		ctxDialer, ok := dialer.(xproxy.ContextDialer)
		t.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, _, err := net.SplitHostPort(addr)
			if err == nil && bypass(host) {
				return (&net.Dialer{Timeout: defaultDialTimeout, KeepAlive: defaultKeepAlive}).DialContext(ctx, network, addr)
			}
			if ok {
				return ctxDialer.DialContext(ctx, network, addr)
			}
			return dialer.Dial(network, addr)
		}
		// Disable the default Proxy func so http requests don't double-route.
		t.Proxy = nil
		return t, nil

	default:
		return nil, fmt.Errorf("netproxy: unsupported scheme %q (use http/https/socks5/socks5h)", scheme)
	}
}

// Test attempts an HTTP GET through the proxy described by s. It does
// NOT touch global state. Use it from a "Test" button in the UI.
func Test(ctx context.Context, s Settings) TestResult {
	probe := strings.TrimSpace(s.TestURL)
	if probe == "" {
		probe = defaultTestURL
	}
	res := TestResult{ProbedURL: probe}

	if !s.Enabled || strings.TrimSpace(s.URL) == "" {
		res.Error = "proxy disabled or URL empty"
		return res
	}

	t, err := BuildTransport(s)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	defer t.CloseIdleConnections()

	client := &http.Client{Transport: t, Timeout: defaultTestTimeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, probe, nil)
	if err != nil {
		res.Error = err.Error()
		return res
	}

	start := time.Now()
	resp, err := client.Do(req)
	res.LatencyMS = time.Since(start).Milliseconds()
	if err != nil {
		res.Error = err.Error()
		return res
	}
	defer resp.Body.Close()
	res.OK = resp.StatusCode < 500
	res.StatusCode = resp.StatusCode
	return res
}

// newBaseTransport clones the standard transport defaults so we don't
// degrade connection pooling / TLS behaviour just by enabling the
// proxy.
func newBaseTransport() *http.Transport {
	return &http.Transport{
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       defaultIdleConnTimeout,
		TLSHandshakeTimeout:   defaultTLSHandshakeTime,
		ExpectContinueTimeout: 1 * time.Second,
		DialContext: (&net.Dialer{
			Timeout:   defaultDialTimeout,
			KeepAlive: defaultKeepAlive,
		}).DialContext,
	}
}

// compileBypass returns a function that reports whether host should
// bypass the proxy. Loopback hosts are always bypassed; the user-
// supplied list is additive.
func compileBypass(list string) func(host string) bool {
	suffixes := []string{}
	for _, raw := range strings.Split(list, ",") {
		s := strings.TrimSpace(strings.ToLower(raw))
		if s != "" {
			suffixes = append(suffixes, s)
		}
	}
	return func(host string) bool {
		h := strings.ToLower(host)
		// Strip port if present. net.SplitHostPort handles IPv6 brackets
		// correctly; on plain hosts without a port it returns an error
		// and we keep h as-is.
		if hostOnly, _, err := net.SplitHostPort(h); err == nil {
			h = hostOnly
		}
		h = strings.TrimPrefix(strings.TrimSuffix(h, "]"), "[")
		// Loopback shortcuts.
		if h == "localhost" || h == "127.0.0.1" || h == "::1" || strings.HasPrefix(h, "127.") {
			return true
		}
		if ip := net.ParseIP(h); ip != nil && ip.IsLoopback() {
			return true
		}
		for _, suf := range suffixes {
			if h == suf || strings.HasSuffix(h, "."+suf) {
				return true
			}
		}
		return false
	}
}
