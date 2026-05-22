package main

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"lurus-switch/internal/appconfig"
	"lurus-switch/internal/connectivity"
	"lurus-switch/internal/netproxy"
	"lurus-switch/internal/redemption"
	"lurus-switch/internal/toolhealth"
	"lurus-switch/internal/topology"
)

// ============================
// Topology Snapshot Binding
// ============================
//
// GetTopologySnapshot returns a single mode-aware snapshot of every runtime
// entity Switch coordinates — CLI tools, the local gateway, the upstream
// proxy, the Lurus Hub, the upstream providers, the OIDC / activation
// token. The frontend renders this as a clickable architecture diagram and
// dispatches inline repair actions on red nodes.
//
// All probes run concurrently behind a single context deadline so the home
// page can poll this safely on a 10s cadence. Probe failures degrade the
// snapshot (the affected node turns red/yellow) rather than failing the
// whole call — every branch state must be visible.

const topologySnapshotTimeout = 12 * time.Second

// GetTopologySnapshot composes the full topology view.
func (a *App) GetTopologySnapshot() topology.Snapshot {
	ctx, cancel := context.WithTimeout(a.ctx, topologySnapshotTimeout)
	defer cancel()

	mode := a.resolveMode()
	in := topology.ComposeInput{Mode: mode}

	// Probes that share no state run in parallel; collect into the input
	// struct under a mutex (cheap — Compose runs after wg.Wait).
	var mu sync.Mutex
	var wg sync.WaitGroup

	// 1. Tool detection + health.
	wg.Add(1)
	go safeGo("topology-tools", func() {
		defer wg.Done()
		tools := a.collectTools(ctx)
		mu.Lock()
		in.Tools = tools
		mu.Unlock()
	})

	// 2. Gateway status.
	wg.Add(1)
	go safeGo("topology-gateway", func() {
		defer wg.Done()
		gw := topology.GatewayInput{}
		if a.gatewaySrv != nil {
			st := a.gatewaySrv.Status()
			cfg := a.gatewaySrv.GetConfig()
			gw = topology.GatewayInput{
				Running:       st.Running,
				Port:          st.Port,
				URL:           st.URL,
				Uptime:        st.Uptime,
				TotalRequests: st.TotalRequests,
				UpstreamURL:   cfg.UpstreamURL,
			}
			if gw.Port == 0 {
				gw.Port = cfg.Port
			}
		}
		mu.Lock()
		in.Gateway = gw
		mu.Unlock()
	})

	// 3. Upstream proxy snapshot + (optional) live test.
	wg.Add(1)
	go safeGo("topology-proxy", func() {
		defer wg.Done()
		p := topology.ProxyInput{}
		if a.proxyMgr != nil {
			settings := a.proxyMgr.GetSettings()
			if up := settings.UpstreamProxy; up != nil && strings.TrimSpace(up.URL) != "" {
				p.Configured = true
				p.URL = up.URL
				p.Enabled = up.Enabled
				if up.Enabled {
					res := netproxy.Test(ctx, *up)
					ok := res.OK
					p.Reachable = &ok
					p.LatencyMs = res.LatencyMS
					p.Error = res.Error
				}
			}
		}
		mu.Lock()
		in.Proxy = p
		mu.Unlock()
	})

	// 4. Hub reachability — distinguish Personal (hub.lurus.cn) from
	//    Reseller (their own) from EndUser (locked).
	wg.Add(1)
	go safeGo("topology-hub", func() {
		defer wg.Done()
		hubURL, hubUpstream := a.resolveHubURL(mode)
		hub := topology.HubInput{URL: hubURL}
		if hubURL != "" {
			ok, latency, errMsg := probeHTTPS(ctx, hubURL, hubUpstream)
			hub.Reachable = ok
			hub.LatencyMs = latency
			hub.Error = errMsg
		}
		mu.Lock()
		in.Hub = hub
		mu.Unlock()
	})

	// 5. Provider reachability via the existing connectivity doctor.
	wg.Add(1)
	go safeGo("topology-providers", func() {
		defer wg.Done()
		var upstream *netproxy.Settings
		if a.proxyMgr != nil {
			if up := a.proxyMgr.GetSettings().UpstreamProxy; up != nil {
				cp := *up
				upstream = &cp
			}
		}
		// Only AI providers — github/npm tier is covered by the tool
		// install probe and would clutter the topology canvas.
		report := connectivity.Run(ctx, aiProviders(), upstream)
		out := make([]topology.ProviderInput, 0, len(report.Providers))
		for _, p := range report.Providers {
			out = append(out, topology.ProviderInput{
				ID:            p.Provider.ID,
				Label:         p.Provider.Label,
				DNSOK:         p.DNSOK,
				DirectOK:      p.DirectOK,
				DirectMs:      p.DirectMS,
				UpstreamOK:    p.UpstreamOK,
				UpstreamMs:    p.UpstreamMS,
				UpstreamTried: p.UpstreamTried,
				Error:         p.DirectError,
			})
		}
		mu.Lock()
		in.Providers = out
		mu.Unlock()
	})

	// 6. Auth / activation — depends on mode.
	wg.Add(1)
	go safeGo("topology-auth", func() {
		defer wg.Done()
		if mode == "enduser" {
			if a.redemptionStore != nil {
				st := a.redemptionStore.Status(time.Now())
				act := topology.ActivationInput{
					State:      string(st.State),
					HubURL:     st.HubURL,
					TenantSlug: st.TenantSlug,
					ExpiresAt:  st.ExpiresAt,
					LastBeat:   st.LastHeartbeat,
				}
				mu.Lock()
				in.Activation = act
				mu.Unlock()
			} else {
				mu.Lock()
				in.Activation = topology.ActivationInput{State: string(redemption.StateUnactivated)}
				mu.Unlock()
			}
			return
		}
		// Personal / Reseller / Enterprise: OIDC.
		if a.authSession != nil {
			st := a.authSession.GetAuthState()
			authIn := topology.AuthInput{
				LoggedIn:        st.IsLoggedIn,
				HasGatewayToken: st.HasGatewayToken,
				ExpiresAt:       st.ExpiresAt,
			}
			if st.User != nil {
				authIn.UserEmail = st.User.Email
			}
			mu.Lock()
			in.Auth = authIn
			mu.Unlock()
		}
	})

	// 7. Current model — pulls from proxy settings (the active gateway model).
	wg.Add(1)
	go safeGo("topology-currentmodel", func() {
		defer wg.Done()
		if a.proxyMgr == nil {
			return
		}
		s := a.proxyMgr.GetSettings()
		mu.Lock()
		in.CurrentModel = s.Model
		mu.Unlock()
	})

	wg.Wait()

	snap := topology.Compose(in, time.Now())
	topology.SortNodesForRender(snap.Nodes)
	return snap
}

// collectTools merges DetectAll + CheckAll outputs into the topology
// input shape. Defensive against either probe failing.
func (a *App) collectTools(ctx context.Context) []topology.ToolInput {
	out := []topology.ToolInput{}
	if a.instMgr == nil {
		return out
	}
	statuses, err := a.instMgr.DetectAll(ctx)
	if err != nil || statuses == nil {
		return out
	}
	healthMap := toolhealth.CheckAll()
	mf := a.loadManifest()

	// Stable order — same as TOOL_ORDER on the frontend.
	order := []string{"claude", "codex", "gemini", "picoclaw", "nullclaw", "zeroclaw", "openclaw"}
	for _, name := range order {
		st, ok := statuses[name]
		if !ok {
			continue
		}
		ti := topology.ToolInput{
			Name:       name,
			Installed:  st.Installed,
			Version:    st.Version,
			Path:       st.Path,
			Update:     st.UpdateAvailable,
			ComingSoon: mf.IsComingSoon(name),
		}
		if h, ok := healthMap[name]; ok && h != nil {
			ti.Health = string(h.Status)
		}
		out = append(out, ti)
	}
	return out
}

// resolveMode reads the persisted AppMode, defaulting to "personal" when
// nothing is on disk so the topology view still renders during first run.
func (a *App) resolveMode() string {
	s, err := appconfig.LoadAppSettings()
	if err != nil || s.AppMode == "" {
		return "personal"
	}
	return s.AppMode
}

// resolveHubURL picks the right Hub coordinate per mode. Returns the URL
// and the optional upstream proxy snapshot to use when probing it.
func (a *App) resolveHubURL(mode string) (string, *netproxy.Settings) {
	var upstream *netproxy.Settings
	if a.proxyMgr != nil {
		if up := a.proxyMgr.GetSettings().UpstreamProxy; up != nil && up.Enabled {
			cp := *up
			upstream = &cp
		}
	}
	switch mode {
	case "enduser":
		if s, err := appconfig.LoadAppSettings(); err == nil {
			if locked := strings.TrimSpace(s.LockedHubURL); locked != "" {
				return locked, upstream
			}
			if hub := strings.TrimSpace(s.Reseller.HubURL); hub != "" {
				return hub, upstream
			}
		}
		return "", upstream
	case "reseller":
		if s, err := appconfig.LoadAppSettings(); err == nil {
			if hub := strings.TrimSpace(s.Reseller.HubURL); hub != "" {
				return hub, upstream
			}
		}
		// Reseller without a hub configured yet → fall back to Lurus
		// so the canvas shows the canonical target.
		return "https://hub.lurus.cn/", upstream
	default:
		// Personal / Enterprise / unset: Lurus operated hub.
		if a.proxyMgr != nil {
			if ep := strings.TrimSpace(a.proxyMgr.GetSettings().APIEndpoint); ep != "" {
				return ep, upstream
			}
		}
		return "https://hub.lurus.cn/", upstream
	}
}

// probeHTTPS does a HEAD request through the optional upstream proxy and
// returns reachability, latency, and a trimmed error string. Mirrors the
// connectivity doctor's headProbe but is local to this binding because we
// pass a different transport when an upstream proxy is enabled.
func probeHTTPS(ctx context.Context, target string, upstream *netproxy.Settings) (bool, int64, string) {
	if _, err := url.Parse(target); err != nil {
		return false, 0, err.Error()
	}
	var tr *http.Transport
	if upstream != nil && upstream.Enabled && strings.TrimSpace(upstream.URL) != "" {
		t, err := netproxy.BuildTransport(*upstream)
		if err != nil {
			return false, 0, err.Error()
		}
		tr = t
	} else {
		tr = &http.Transport{
			DialContext:         (&net.Dialer{Timeout: 3 * time.Second}).DialContext,
			TLSHandshakeTimeout: 3 * time.Second,
		}
	}
	defer tr.CloseIdleConnections()
	cli := &http.Client{Transport: tr, Timeout: 5 * time.Second}

	reqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodHead, target, nil)
	if err != nil {
		return false, 0, err.Error()
	}
	req.Header.Set("User-Agent", "lurus-switch-topology/1.0")
	start := time.Now()
	resp, err := cli.Do(req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return false, latency, err.Error()
	}
	resp.Body.Close()
	if resp.StatusCode >= 500 {
		return false, latency, "HTTP " + resp.Status
	}
	return true, latency, ""
}

// aiProviders returns the AI subset of the connectivity doctor's provider
// list — we want the topology canvas to focus on what serves model traffic.
// GitHub / npm reachability is surfaced via the tool install nodes
// (those probes update the installer manifest). Lurus Hub is intentionally
// excluded here: it's already rendered as its own Hub node so listing it
// twice on the canvas would mislead the user.
func aiProviders() []connectivity.Provider {
	all := connectivity.DefaultProviders()
	out := make([]connectivity.Provider, 0, 5)
	for _, p := range all {
		if p.Tier == "ai" {
			out = append(out, p)
		}
	}
	// Add DeepSeek explicitly — it's a key China-friendly fallback and is
	// not in the doctor's default list.
	out = append(out, connectivity.Provider{
		ID:    "deepseek",
		Label: "DeepSeek",
		URL:   "https://api.deepseek.com/",
		Tier:  "ai",
	})
	return out
}
