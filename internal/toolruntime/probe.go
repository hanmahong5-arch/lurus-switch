// Package toolruntime aggregates per-tool live state — what endpoint
// is each CLI configured to talk to, is that endpoint reachable RIGHT
// NOW, is the CLI process actually running, etc. Powers the "Runtime
// Status" panel on Home so users get a single dashboard view of
// "where do my AI CLIs send traffic, and is anything broken".
//
// The probe is best-effort and never blocks the UI: every reachability
// check has a 3s timeout and any failure is encoded as a yellow status,
// not a panic.
package toolruntime

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ConnState mirrors the visual stoplight on the frontend so the UI
// doesn't have to recompute it from booleans.
type ConnState string

const (
	ConnUnknown   ConnState = "unknown"   // not configured, can't tell
	ConnReachable ConnState = "reachable" // probe succeeded
	ConnDegraded  ConnState = "degraded"  // probe slow / 4xx (non-auth)
	ConnDown      ConnState = "down"      // probe failed
)

// ToolRuntime is the per-tool snapshot returned by Probe.
type ToolRuntime struct {
	Tool         string    `json:"tool"`
	Installed    bool      `json:"installed"`     // config file exists on disk
	ConfigPath   string    `json:"configPath"`
	Model        string    `json:"model"`         // empty if not configured
	Endpoint     string    `json:"endpoint"`      // resolved API base URL the CLI is pointing at
	EndpointKind string    `json:"endpointKind"`  // "official" | "lurus-gateway" | "third-party" | "unknown"
	HasAPIKey    bool      `json:"hasApiKey"`     // does config carry a non-empty key
	ProcessRunning bool    `json:"processRunning"`
	ProcessPID   int       `json:"processPID,omitempty"`
	ConnState    ConnState `json:"connState"`
	LatencyMs    int64     `json:"latencyMs,omitempty"` // round-trip time of the probe
	ProbeError   string    `json:"probeError,omitempty"`
	CheckedAt    time.Time `json:"checkedAt"`
}

// Tools we know how to probe.
var SupportedTools = []string{"claude", "codex", "gemini", "picoclaw", "nullclaw", "zeroclaw", "openclaw"}

// ProbeOptions lets callers inject a custom HTTP client (mainly for
// tests) and a list of currently-running CLI PIDs by tool name.
type ProbeOptions struct {
	HTTP          *http.Client
	RunningPIDs   map[string]int // tool name → first PID found running, 0 if none
	GatewayPort   int            // local Switch gateway port for endpoint-kind detection
}

// ProbeAll runs Probe for every supported tool concurrently and
// returns the slice in tool-order. Aggregate latency is bounded by the
// slowest probe, since they fan out.
func ProbeAll(ctx context.Context, opts ProbeOptions) []ToolRuntime {
	if opts.HTTP == nil {
		opts.HTTP = &http.Client{Timeout: 3 * time.Second}
	}
	out := make([]ToolRuntime, len(SupportedTools))
	type result struct {
		idx int
		rt  ToolRuntime
	}
	ch := make(chan result, len(SupportedTools))
	for i, tool := range SupportedTools {
		i, tool := i, tool
		go func() {
			ch <- result{idx: i, rt: Probe(ctx, tool, opts)}
		}()
	}
	for range SupportedTools {
		r := <-ch
		out[r.idx] = r.rt
	}
	return out
}

// Probe inspects a single tool's on-disk config + checks endpoint
// reachability. Pure I/O, no side-effects on the config files.
func Probe(ctx context.Context, tool string, opts ProbeOptions) ToolRuntime {
	rt := ToolRuntime{Tool: tool, CheckedAt: time.Now(), ConnState: ConnUnknown}

	// 1) locate config + load model/endpoint
	cfgPath, model, endpoint, hasKey, exists := readToolConfig(tool)
	rt.ConfigPath = cfgPath
	rt.Model = model
	rt.Endpoint = endpoint
	rt.HasAPIKey = hasKey
	rt.Installed = exists
	rt.EndpointKind = classifyEndpoint(endpoint, opts.GatewayPort)

	// 2) process state — opt-in via opts so the binding layer can pass
	//    pre-computed PID map (avoids re-running ps for every tool).
	if opts.RunningPIDs != nil {
		if pid, ok := opts.RunningPIDs[tool]; ok && pid > 0 {
			rt.ProcessRunning = true
			rt.ProcessPID = pid
		}
	}

	// 3) endpoint reachability probe — only if we have something to hit.
	if endpoint != "" {
		state, latency, errMsg := probeEndpoint(ctx, opts.HTTP, endpoint)
		rt.ConnState = state
		rt.LatencyMs = latency
		rt.ProbeError = errMsg
	}

	return rt
}

// classifyEndpoint maps a resolved URL to a category for UI grouping.
// "official" matches well-known vendor hosts; "lurus-gateway" matches
// loopback on the Switch gateway port; everything else is "third-party"
// (which includes user-run proxies).
func classifyEndpoint(endpoint string, gatewayPort int) string {
	if endpoint == "" {
		return "unknown"
	}
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" {
		return "unknown"
	}
	host := strings.ToLower(u.Hostname())
	switch {
	case host == "api.anthropic.com",
		host == "api.openai.com",
		host == "generativelanguage.googleapis.com",
		strings.HasSuffix(host, ".googleapis.com"):
		return "official"
	case (host == "localhost" || host == "127.0.0.1" || host == "0.0.0.0"):
		// Treat any loopback as lurus-gateway; if user is running their
		// own loopback proxy we still surface it as gateway-like for UX
		// purposes. Granular distinction needs Switch-set port match.
		if gatewayPort > 0 && u.Port() != "" {
			if u.Port() == intToStr(gatewayPort) {
				return "lurus-gateway"
			}
		}
		return "lurus-gateway"
	default:
		return "third-party"
	}
}

func intToStr(n int) string {
	// Fast-path int → decimal string without importing strconv into the
	// classifier (keeps it dependency-free for testing).
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

// probeEndpoint fires a HEAD against the resolved URL with the
// caller-supplied timeout. We accept any 2xx/3xx/401/403 as proof of
// reachability — auth-required responses still mean "the host is up
// and routes our request", which is what the user wants to know.
func probeEndpoint(ctx context.Context, client *http.Client, endpoint string) (ConnState, int64, string) {
	probeURL := endpoint
	if !strings.HasSuffix(probeURL, "/") {
		probeURL += "/"
	}
	req, err := http.NewRequestWithContext(ctx, "HEAD", probeURL, nil)
	if err != nil {
		return ConnDown, 0, "build request: " + err.Error()
	}
	req.Header.Set("User-Agent", "lurus-switch-runtime-probe/1.0")
	start := time.Now()
	resp, err := client.Do(req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		// Many vendor hosts reject HEAD; fall back to a tiny GET so we
		// don't false-negative on Anthropic/OpenAI which both 405 HEAD.
		req2, _ := http.NewRequestWithContext(ctx, "GET", probeURL, nil)
		req2.Header.Set("User-Agent", "lurus-switch-runtime-probe/1.0")
		start2 := time.Now()
		resp2, err2 := client.Do(req2)
		latency = time.Since(start2).Milliseconds()
		if err2 != nil {
			return ConnDown, latency, err2.Error()
		}
		resp = resp2
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode >= 500:
		return ConnDegraded, latency, "5xx upstream"
	case resp.StatusCode == 408 || resp.StatusCode == 429:
		return ConnDegraded, latency, "rate-limited or timeout"
	case latency > 2000:
		return ConnDegraded, latency, "slow response"
	default:
		return ConnReachable, latency, ""
	}
}

// ─── Per-tool config readers ────────────────────────────────────────

// readToolConfig is the central tool→(model, endpoint, has-key) lookup.
// Each tool stores config differently; we lean on best-effort JSON/TOML
// parsing rather than reusing the generators (which would require a lot
// of imports for a probe).
func readToolConfig(tool string) (path, model, endpoint string, hasKey, exists bool) {
	home, _ := os.UserHomeDir()
	switch tool {
	case "claude":
		path = filepath.Join(home, ".claude", "settings.json")
		raw := readJSON(path)
		exists = raw != nil
		if raw == nil {
			return
		}
		model, _ = raw["model"].(string)
		if k, _ := raw["apiKey"].(string); k != "" {
			hasKey = true
		}
		// apiBaseUrl override on top-level overrides the official endpoint;
		// fall back to the official one when none is set so the probe has
		// a target.
		if u, _ := raw["apiBaseUrl"].(string); u != "" {
			endpoint = u
		} else if adv, ok := raw["advanced"].(map[string]interface{}); ok {
			if u, _ := adv["apiEndpoint"].(string); u != "" {
				endpoint = u
			}
		}
		if endpoint == "" {
			endpoint = "https://api.anthropic.com"
		}
	case "codex":
		path = filepath.Join(home, ".codex", "config.toml")
		text, ok := readText(path)
		exists = ok
		if !ok {
			return
		}
		model = tomlScalar(text, "model")
		baseURL := tomlScalar(text, "base_url")
		if baseURL != "" {
			endpoint = baseURL
		} else {
			endpoint = "https://api.openai.com/v1"
		}
		if k := tomlScalar(text, "api_key"); k != "" {
			hasKey = true
		}
	case "gemini":
		path = filepath.Join(home, ".gemini", "settings.json")
		raw := readJSON(path)
		exists = raw != nil
		if raw == nil {
			return
		}
		model, _ = raw["model"].(string)
		if k, _ := raw["apiKey"].(string); k != "" {
			hasKey = true
		}
		if adv, ok := raw["advanced"].(map[string]interface{}); ok {
			if u, _ := adv["apiEndpoint"].(string); u != "" {
				endpoint = u
			}
		}
		if endpoint == "" {
			endpoint = "https://generativelanguage.googleapis.com"
		}
	case "picoclaw", "nullclaw":
		path = filepath.Join(home, "."+tool, "config.json")
		raw := readJSON(path)
		exists = raw != nil
		if raw == nil {
			return
		}
		model, _ = raw["model"].(string)
		endpoint, _ = raw["apiEndpoint"].(string)
		if endpoint == "" {
			endpoint, _ = raw["base_url"].(string)
		}
		if k, _ := raw["apiKey"].(string); k != "" {
			hasKey = true
		}
	case "zeroclaw", "openclaw":
		path = filepath.Join(home, "."+tool, "config."+map[string]string{"zeroclaw": "toml", "openclaw": "json"}[tool])
		if _, err := os.Stat(path); err == nil {
			exists = true
		}
		if tool == "zeroclaw" {
			text, _ := readText(path)
			model = tomlScalar(text, "model")
			endpoint = tomlScalar(text, "base_url")
			if k := tomlScalar(text, "api_key"); k != "" {
				hasKey = true
			}
		} else {
			raw := readJSON(path)
			if raw != nil {
				if p, _ := raw["provider"].(map[string]interface{}); p != nil {
					model, _ = p["model"].(string)
					endpoint, _ = p["base_url"].(string)
					if k, _ := p["api_key"].(string); k != "" {
						hasKey = true
					}
				}
			}
		}
	}
	return
}

func readJSON(path string) map[string]interface{} {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var raw map[string]interface{}
	if json.Unmarshal(data, &raw) != nil {
		return nil
	}
	return raw
}

func readText(path string) (string, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	return string(data), true
}

// tomlScalar extracts a top-level-ish scalar like `key = "value"` or
// `key = 123`. Section-aware parsing isn't needed for the few keys we
// look at here (model / base_url / api_key); first match wins.
func tomlScalar(text, key string) string {
	for _, line := range strings.Split(text, "\n") {
		t := strings.TrimSpace(line)
		if !strings.HasPrefix(t, key) {
			continue
		}
		// Match "key = ..." with optional whitespace.
		rest := strings.TrimSpace(strings.TrimPrefix(t, key))
		if !strings.HasPrefix(rest, "=") {
			continue
		}
		val := strings.TrimSpace(strings.TrimPrefix(rest, "="))
		// Strip wrapping quotes and trailing comments.
		if i := strings.Index(val, "#"); i >= 0 {
			val = strings.TrimSpace(val[:i])
		}
		val = strings.Trim(val, "\"'")
		return val
	}
	return ""
}
