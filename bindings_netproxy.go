package main

import (
	"context"
	"time"

	"lurus-switch/internal/netproxy"
)

// ============================
// Upstream Proxy Bindings
// ============================

// netproxyTestTimeout caps the live-test probe so a dead proxy can't
// hang the UI button.
const netproxyTestTimeout = 12 * time.Second

// GetUpstreamProxy returns the currently-saved upstream proxy settings,
// or zero-value Settings (Enabled=false) if the user hasn't configured
// one yet. The structure is returned by value so the Wails type
// generator can emit a concrete TS type.
func (a *App) GetUpstreamProxy() netproxy.Settings {
	if a.proxyMgr == nil {
		return netproxy.Settings{}
	}
	if up := a.proxyMgr.GetSettings().UpstreamProxy; up != nil {
		return *up
	}
	return netproxy.Settings{}
}

// TestUpstreamProxy issues a single probe through the supplied settings
// (does NOT persist) and reports latency + status, or a descriptive
// error. Caller drives this from a "Test" button — keep synchronous.
func (a *App) TestUpstreamProxy(s netproxy.Settings) netproxy.TestResult {
	ctx, cancel := context.WithTimeout(a.ctx, netproxyTestTimeout)
	defer cancel()
	return netproxy.Test(ctx, s)
}
