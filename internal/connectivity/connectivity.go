// Package connectivity provides a "Doctor"-style probe that answers the
// question every CN user opens Switch wanting answered:
//
//   "Why can't I reach Claude / OpenAI / Gemini, and what's the
//    cheapest fix?"
//
// It returns a state matrix per provider (Direct vs Through-Upstream-Proxy
// reachability) plus heuristic suggestions (system-proxy env vars, local
// SOCKS5 listeners on common ports). The UI consumes the matrix to render
// remedies — this package itself takes no action, modifies no state.
package connectivity

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"lurus-switch/internal/netproxy"
)

const (
	probeTimeout    = 5 * time.Second
	dialTimeout     = 2 * time.Second
	dnsLookupTimeout = 3 * time.Second
)

// Provider is one row in the doctor's table.
type Provider struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	URL   string `json:"url"`
	// Tier is purely advisory for UI ordering. "ai" providers shown first,
	// "infra" (GitHub / npm) below.
	Tier string `json:"tier"`
}

// ProviderResult captures the outcome of probing one provider.
type ProviderResult struct {
	Provider     Provider `json:"provider"`
	DNSOK        bool     `json:"dnsOK"`
	DNSError     string   `json:"dnsError,omitempty"`
	DirectOK     bool     `json:"directOK"`
	DirectMS     int64    `json:"directMs,omitempty"`
	DirectError  string   `json:"directError,omitempty"`
	UpstreamOK   bool     `json:"upstreamOK,omitempty"`
	UpstreamMS   int64    `json:"upstreamMs,omitempty"`
	UpstreamError string  `json:"upstreamError,omitempty"`
	UpstreamTried bool    `json:"upstreamTried"`
}

// LocalProxy is a detected SOCKS5 / HTTP proxy listening on a loopback port.
type LocalProxy struct {
	Host         string `json:"host"`
	Port         int    `json:"port"`
	URL          string `json:"url"`
	GuessedName  string `json:"guessedName,omitempty"`
}

// SystemProxy reflects environment variables / OS settings.
type SystemProxy struct {
	HTTPProxy  string `json:"httpProxy,omitempty"`
	HTTPSProxy string `json:"httpsProxy,omitempty"`
	AllProxy   string `json:"allProxy,omitempty"`
	NoProxy    string `json:"noProxy,omitempty"`
}

// Report bundles everything the UI needs to render the doctor view.
type Report struct {
	GeneratedAt time.Time         `json:"generatedAt"`
	Providers   []ProviderResult  `json:"providers"`
	LocalProxies []LocalProxy     `json:"localProxies"`
	SystemProxy SystemProxy       `json:"systemProxy"`
	// Suggestions is a short ordered list of one-line remedies the UI can
	// surface as actionable buttons.
	Suggestions []Suggestion `json:"suggestions"`
}

// SuggestionKind is a tag the UI can dispatch on.
type SuggestionKind string

const (
	SuggestUseUpstream     SuggestionKind = "use-upstream"
	SuggestAutoFillProxy   SuggestionKind = "auto-fill-proxy"
	SuggestUseLurusRelay   SuggestionKind = "use-lurus-relay"
	SuggestSwitchModel     SuggestionKind = "switch-model"
	SuggestAllOK           SuggestionKind = "all-ok"
)

// Suggestion is a UI-actionable remedy.
type Suggestion struct {
	Kind   SuggestionKind `json:"kind"`
	Title  string         `json:"title"`
	Detail string         `json:"detail"`
	// Payload carries action-specific data, e.g. a proxy URL to apply.
	Payload string `json:"payload,omitempty"`
}

// DefaultProviders returns the standard set probed. UI-visible order.
func DefaultProviders() []Provider {
	return []Provider{
		{ID: "anthropic", Label: "Anthropic API", URL: "https://api.anthropic.com/", Tier: "ai"},
		{ID: "openai", Label: "OpenAI API", URL: "https://api.openai.com/", Tier: "ai"},
		{ID: "gemini", Label: "Google Gemini", URL: "https://generativelanguage.googleapis.com/", Tier: "ai"},
		{ID: "lurus", Label: "Lurus Hub", URL: "https://hub.lurus.cn/", Tier: "lurus"},
		{ID: "github", Label: "GitHub", URL: "https://api.github.com/", Tier: "infra"},
		{ID: "npm", Label: "npm Registry", URL: "https://registry.npmjs.org/", Tier: "infra"},
	}
}

// commonProxyPorts is the ordered list of loopback ports we probe for an
// already-running local proxy. Ordering matters — first hit wins for
// auto-detect. MasterDnsVPN is first because plugins/dnstunnel ships with
// it as the default.
var commonProxyPorts = []struct {
	Port int
	Name string
}{
	{18000, "MasterDnsVPN"},
	{7890, "Clash"},
	{7891, "Clash (HTTP)"},
	{1080, "V2Ray / Shadowsocks (SOCKS5)"},
	{1087, "V2Ray (HTTP)"},
	{8080, "HTTP proxy"},
	{8118, "Privoxy"},
	{10808, "V2RayN (SOCKS5)"},
	{10809, "V2RayN (HTTP)"},
}

// Run executes the full doctor probe. Pass the user's current upstream
// settings so the report includes a "via upstream" column. Pass a nil
// upstream to skip that column.
func Run(ctx context.Context, providers []Provider, upstream *netproxy.Settings) Report {
	if providers == nil {
		providers = DefaultProviders()
	}
	r := Report{
		GeneratedAt:  time.Now(),
		Providers:    make([]ProviderResult, len(providers)),
		LocalProxies: detectLocalProxies(ctx),
		SystemProxy:  detectSystemProxy(),
	}

	// Probe providers in parallel — each probe is bounded by probeTimeout
	// and we deliberately don't cancel the whole batch on one failure.
	var wg sync.WaitGroup
	wg.Add(len(providers))
	for i := range providers {
		go func(i int) {
			defer wg.Done()
			r.Providers[i] = probeProvider(ctx, providers[i], upstream)
		}(i)
	}
	wg.Wait()

	r.Suggestions = buildSuggestions(r, upstream)
	return r
}

func probeProvider(ctx context.Context, p Provider, upstream *netproxy.Settings) ProviderResult {
	out := ProviderResult{Provider: p}

	u, err := url.Parse(p.URL)
	if err != nil {
		out.DirectError = err.Error()
		return out
	}

	// DNS first — separate signal: "DNS works but TCP fails" vs "DNS fails
	// entirely" point at different remedies.
	dnsCtx, cancel := context.WithTimeout(ctx, dnsLookupTimeout)
	defer cancel()
	if _, err := (&net.Resolver{}).LookupHost(dnsCtx, u.Hostname()); err != nil {
		out.DNSError = err.Error()
	} else {
		out.DNSOK = true
	}

	// Direct probe (uses http.DefaultTransport, which currently reflects
	// the user's saved upstream proxy if enabled — so for the "direct"
	// column we build a fresh transport with no proxy).
	direct := &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{Timeout: dialTimeout}).DialContext,
			TLSHandshakeTimeout: dialTimeout,
		},
		Timeout: probeTimeout,
	}
	out.DirectOK, out.DirectMS, out.DirectError = headProbe(ctx, direct, p.URL)

	// Upstream probe — only if the user has a non-empty config to test.
	if upstream != nil && upstream.Enabled && strings.TrimSpace(upstream.URL) != "" {
		out.UpstreamTried = true
		tr, err := netproxy.BuildTransport(*upstream)
		if err != nil {
			out.UpstreamError = err.Error()
		} else {
			defer tr.CloseIdleConnections()
			via := &http.Client{Transport: tr, Timeout: probeTimeout}
			out.UpstreamOK, out.UpstreamMS, out.UpstreamError = headProbe(ctx, via, p.URL)
		}
	}
	return out
}

func headProbe(ctx context.Context, client *http.Client, target string) (ok bool, ms int64, errStr string) {
	reqCtx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodHead, target, nil)
	if err != nil {
		return false, 0, err.Error()
	}
	req.Header.Set("User-Agent", "lurus-switch-doctor/1.0")
	start := time.Now()
	resp, err := client.Do(req)
	ms = time.Since(start).Milliseconds()
	if err != nil {
		return false, ms, err.Error()
	}
	defer resp.Body.Close()
	// 4xx is fine — the endpoint answered. We only care about reachability.
	ok = resp.StatusCode < 500
	if !ok {
		errStr = "HTTP " + resp.Status
	}
	return
}

func detectLocalProxies(ctx context.Context) []LocalProxy {
	out := make([]LocalProxy, 0, 2)
	for _, c := range commonProxyPorts {
		addr := net.JoinHostPort("127.0.0.1", itoa(c.Port))
		dialCtx, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
		conn, err := (&net.Dialer{}).DialContext(dialCtx, "tcp", addr)
		cancel()
		if err != nil {
			continue
		}
		_ = conn.Close()
		// Guess scheme: ports we recognise as SOCKS5 we tag as socks5; the
		// rest we expose as http and let the user override.
		scheme := "http"
		if c.Port == 1080 || c.Port == 10808 || c.Port == 18000 || c.Port == 7890 {
			scheme = "socks5"
		}
		out = append(out, LocalProxy{
			Host:        "127.0.0.1",
			Port:        c.Port,
			URL:         scheme + "://127.0.0.1:" + itoa(c.Port),
			GuessedName: c.Name,
		})
	}
	return out
}

func detectSystemProxy() SystemProxy {
	return SystemProxy{
		HTTPProxy:  firstNonEmpty(getEnv("HTTP_PROXY"), getEnv("http_proxy")),
		HTTPSProxy: firstNonEmpty(getEnv("HTTPS_PROXY"), getEnv("https_proxy")),
		AllProxy:   firstNonEmpty(getEnv("ALL_PROXY"), getEnv("all_proxy")),
		NoProxy:    firstNonEmpty(getEnv("NO_PROXY"), getEnv("no_proxy")),
	}
}

func buildSuggestions(r Report, upstream *netproxy.Settings) []Suggestion {
	var out []Suggestion

	// Group results.
	var aiBroken, aiDirect, lurusBroken int
	var anthropicBroken bool
	for _, p := range r.Providers {
		switch p.Provider.Tier {
		case "ai":
			if !p.DirectOK {
				aiBroken++
				if p.Provider.ID == "anthropic" {
					anthropicBroken = true
				}
			} else {
				aiDirect++
			}
		case "lurus":
			if !p.DirectOK {
				lurusBroken++
			}
		}
	}

	upstreamEnabled := upstream != nil && upstream.Enabled && strings.TrimSpace(upstream.URL) != ""

	// Case 1: nothing's broken. Tell them.
	if aiBroken == 0 && lurusBroken == 0 {
		out = append(out, Suggestion{
			Kind:   SuggestAllOK,
			Title:  "All providers reachable directly",
			Detail: "Your network can talk to Anthropic, OpenAI, Gemini, and Lurus Hub without any proxy. You don't need to change anything.",
		})
		return out
	}

	// Case 2: AI providers are broken but Lurus hub works → push them to
	// use Lurus relay as the path of least resistance (the product's own
	// moat). Cheapest user action.
	if aiBroken > 0 && lurusBroken == 0 {
		out = append(out, Suggestion{
			Kind:   SuggestUseLurusRelay,
			Title:  "Use Lurus Hub as your AI relay",
			Detail: "Direct access to AI providers is blocked, but Lurus Hub is reachable. Switch can route Claude/OpenAI/Gemini through Lurus's gateway — this is the supported path and includes billing/SLA.",
		})
	}

	// Case 3: AI broken, Lurus also broken (or already failing through
	// upstream) → recommend the BYO upstream proxy. If we detected a
	// running local proxy, offer to auto-fill it.
	if (aiBroken > 0 && lurusBroken > 0) || anthropicBroken {
		if !upstreamEnabled && len(r.LocalProxies) > 0 {
			pick := r.LocalProxies[0]
			out = append(out, Suggestion{
				Kind:    SuggestAutoFillProxy,
				Title:   "Detected a local proxy at " + pick.URL,
				Detail:  "It looks like " + pick.GuessedName + " is already running on this machine. Enable Switch's upstream proxy and point it here in one click.",
				Payload: pick.URL,
			})
		} else if !upstreamEnabled {
			out = append(out, Suggestion{
				Kind:   SuggestUseUpstream,
				Title:  "Configure an upstream proxy",
				Detail: "Run V2Ray / Clash / Shadowsocks locally (or the plugins/dnstunnel client), then enable Switch's upstream proxy and point it at the local SOCKS5 port.",
			})
		}
	}

	// Case 4: upstream IS enabled but isn't helping → tell them.
	if upstreamEnabled {
		var fixedByUpstream, brokenViaUpstream int
		for _, p := range r.Providers {
			if !p.UpstreamTried {
				continue
			}
			if !p.DirectOK && p.UpstreamOK {
				fixedByUpstream++
			}
			if !p.UpstreamOK {
				brokenViaUpstream++
			}
		}
		if fixedByUpstream > 0 {
			out = append(out, Suggestion{
				Kind:   SuggestAllOK,
				Title:  "Upstream proxy is working",
				Detail: "The upstream proxy fixes reachability for " + itoa(fixedByUpstream) + " provider(s) that fail directly. Keep it enabled.",
			})
		} else if brokenViaUpstream > 0 {
			out = append(out, Suggestion{
				Kind:   SuggestUseUpstream,
				Title:  "Upstream proxy is configured but not helping",
				Detail: "The proxy URL you set doesn't reach the AI providers either. Check the proxy is running, the URL is correct, and any auth credentials are valid.",
			})
		}
	}

	// Case 5: even Lurus broken → mention model switch as fallback.
	if lurusBroken > 0 && !upstreamEnabled {
		out = append(out, Suggestion{
			Kind:   SuggestSwitchModel,
			Title:  "Try a China-friendly model",
			Detail: "Lurus Hub is unreachable. As a temporary fallback, Switch can route to providers with mainland access (DeepSeek, Qwen, GLM) via newapi.lurus.cn.",
		})
	}

	return out
}

// --- helpers --- //

func getEnv(k string) string {
	return strings.TrimSpace(envLookup(k))
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func itoa(n int) string {
	// Local tiny itoa to keep dependency footprint flat — strconv would be
	// fine too, kept as-is for symmetry with the test helpers.
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
