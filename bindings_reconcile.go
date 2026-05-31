package main

import (
	"errors"
	"fmt"
	"strings"

	"lurus-switch/internal/appconfig"
	"lurus-switch/internal/hub/admin"
	"lurus-switch/internal/metering"
)

// ============================
// Usage reconciliation + export bindings (Wave 1 W1.2)
// ============================
//
// Switch records every gateway request locally (internal/metering). The Hub
// independently logs the same traffic under the gateway's user_token. These two
// ledgers drift silently today. ReconcileUsage compares aggregate totals over a
// window and reports the gap; ExportMetering dumps the local records so a
// reseller can hand-join them against their Hub console.
//
// Scope is AGGREGATE-level drift, NOT per-record matching — Switch's per-record
// correlation IDs don't reach the Hub yet (that needs gateway passthrough + Hub
// storage, a later step). The report is honest about this: when the Hub side
// can't be fetched it returns Reconciled=false with a Note, never a false
// all-green.

// reconcileHubClient builds a Hub client authenticated with the SAME token the
// gateway routes upstream through (OIDC session gateway token > manual proxy
// UserToken) — that token's Hub consume logs are exactly what we reconcile
// against. Base URL prefers the Reseller Hub, then the gateway upstream, then
// the locked EndUser Hub.
func (a *App) reconcileHubClient() (*admin.Client, error) {
	token := ""
	if a.authSession != nil && a.authSession.HasGatewayToken() {
		token = a.authSession.GetGatewayToken()
	}

	base := ""
	if a.proxyMgr != nil {
		s := a.proxyMgr.GetSettings()
		if token == "" {
			token = s.UserToken
		}
		base = s.APIEndpoint
	}

	if settings, err := appconfig.LoadAppSettings(); err == nil {
		if settings.Reseller.HubURL != "" {
			base = settings.Reseller.HubURL
		} else if base == "" {
			base = settings.LockedHubURL
		}
	}

	if base == "" {
		return nil, errors.New("Hub URL 未配置")
	}
	if token == "" {
		return nil, errors.New("网关 token 未配置 — 请登录或在代理设置中填写 UserToken")
	}
	return admin.New(admin.Config{BaseURL: base, Token: token})
}

// ReconcileUsage compares Switch's local metering against the Hub's consume-log
// totals for the given period ("today" / "week" / "month"). It always returns a
// report (never nil on a Hub problem) so the UI can show local numbers with a
// "Hub not connected" note. period strings mirror the dashboard bindings so
// both sides use an identical window.
func (a *App) ReconcileUsage(period string) (*metering.ReconcileReport, error) {
	if a.meterStore == nil {
		return nil, errors.New("metering store not initialized")
	}
	from, to := periodToRange(period)
	local := a.meterStore.LocalUsage(from, to)

	c, err := a.reconcileHubClient()
	if err != nil {
		rep := metering.Reconcile(local, metering.HubAgg{}, false)
		rep.Note = "Hub 未连接：" + err.Error()
		return &rep, nil
	}

	agg, err := c.FetchUsage(a.hubCtx(), from.Unix(), to.Unix())
	if err != nil {
		rep := metering.Reconcile(local, metering.HubAgg{}, false)
		// A 404 here almost always means the reconciliation endpoint isn't
		// deployed on this Hub yet — surface that as an actionable next step.
		if admin.IsNotFound(err) {
			rep.Note = "Hub 暂不支持对账（端点未部署）"
		} else {
			rep.Note = "Hub 用量获取失败：" + err.Error()
		}
		return &rep, nil
	}

	hub := metering.HubAgg{
		TokensIn:  agg.TotalPromptTokens,
		TokensOut: agg.TotalCompletionTokens,
		Calls:     agg.RequestCount,
		CostUSD:   agg.CostUSD(),
	}
	rep := metering.Reconcile(local, hub, true)
	return &rep, nil
}

// ExportMetering serializes the local metering records for a period to CSV or
// JSON (keyed by Record.ID) so a reseller can cross-check against their Hub.
func (a *App) ExportMetering(period, format string) (string, error) {
	if a.meterStore == nil {
		return "", errors.New("metering store not initialized")
	}
	from, to := periodToRange(period)
	records := a.meterStore.ExportRange(from, to)

	switch strings.ToLower(strings.TrimSpace(format)) {
	case "csv":
		return metering.MarshalRecordsCSV(records)
	case "json", "":
		b, err := metering.MarshalRecordsJSON(records)
		return string(b), err
	default:
		return "", fmt.Errorf("unsupported export format %q (want csv or json)", format)
	}
}
