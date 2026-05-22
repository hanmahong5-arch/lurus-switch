package main

import (
	"context"
	"time"

	"lurus-switch/internal/connectivity"
	"lurus-switch/internal/netproxy"
)

// ============================
// Connectivity Doctor Bindings
// ============================

const connectivityDoctorTimeout = 12 * time.Second

// RunConnectivityDiagnostic probes the canonical AI providers + Lurus hub
// + infra services in parallel, both directly and (if the user has one
// configured) through their upstream proxy. The returned Report carries
// the per-provider state matrix plus actionable suggestions the UI can
// render as one-click remedies.
func (a *App) RunConnectivityDiagnostic() connectivity.Report {
	ctx, cancel := context.WithTimeout(a.ctx, connectivityDoctorTimeout)
	defer cancel()

	var upstream *netproxy.Settings
	if a.proxyMgr != nil {
		if up := a.proxyMgr.GetSettings().UpstreamProxy; up != nil {
			cp := *up
			upstream = &cp
		}
	}
	return connectivity.Run(ctx, connectivity.DefaultProviders(), upstream)
}

// DetectLocalProxies returns just the local-proxy scan portion of the
// doctor — used by the "Auto-detect" button in the Upstream Proxy panel
// so it can fast-path without running the full diagnostic.
func (a *App) DetectLocalProxies() []connectivity.LocalProxy {
	ctx, cancel := context.WithTimeout(a.ctx, 3*time.Second)
	defer cancel()
	return connectivity.Run(ctx, []connectivity.Provider{}, nil).LocalProxies
}
