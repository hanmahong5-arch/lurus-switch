package main

import (
	"fmt"
	"time"

	"lurus-switch/internal/gateway"
	"lurus-switch/internal/metering"
)

// ============================
// Gateway Methods (replaces serverctl bindings)
// ============================

// GetGatewayStatus returns the current state of the local API gateway.
func (a *App) GetGatewayStatus() gateway.Status {
	if a.gatewaySrv == nil {
		return gateway.Status{}
	}
	return a.gatewaySrv.Status()
}

// StartGateway starts the local API gateway on localhost.
func (a *App) StartGateway() error {
	if a.gatewaySrv == nil {
		return fmt.Errorf("gateway not initialized")
	}
	// Sync upstream config from proxy settings before starting.
	a.syncGatewayUpstream()
	return a.gatewaySrv.Start(a.ctx)
}

// StopGateway stops the local API gateway.
func (a *App) StopGateway() error {
	if a.gatewaySrv == nil {
		return fmt.Errorf("gateway not initialized")
	}
	return a.gatewaySrv.Stop()
}

// GetGatewayConfig returns the current gateway configuration.
func (a *App) GetGatewayConfig() gateway.Config {
	if a.gatewaySrv == nil {
		return gateway.DefaultConfig()
	}
	return a.gatewaySrv.GetConfig()
}

// SaveGatewayConfig persists a new gateway configuration.
func (a *App) SaveGatewayConfig(cfg gateway.Config) error {
	if a.gatewaySrv == nil {
		return fmt.Errorf("gateway not initialized")
	}
	return a.gatewaySrv.SaveConfig(cfg)
}

// GetGatewayURL returns the base URL of the running gateway, or "" if stopped.
func (a *App) GetGatewayURL() string {
	if a.gatewaySrv == nil {
		return ""
	}
	st := a.gatewaySrv.Status()
	return st.URL
}

// syncGatewayUpstream reads proxy settings and pushes upstream URL/token to the gateway.
func (a *App) syncGatewayUpstream() {
	if a.gatewaySrv == nil || a.proxyMgr == nil {
		return
	}
	settings := a.proxyMgr.GetSettings()
	a.gatewaySrv.UpdateUpstream(settings.APIEndpoint, settings.BuildToolAPIKey())
}

// ============================
// Metering Methods
// ============================

// GetTodaySummary returns aggregated usage for today.
func (a *App) GetTodaySummary() metering.DailySummary {
	if a.meterStore == nil {
		return metering.DailySummary{}
	}
	return a.meterStore.TodaySummary()
}

// GetDaySummaries returns daily summaries for the last N days.
func (a *App) GetDaySummaries(days int) []metering.DailySummary {
	if a.meterStore == nil {
		return nil
	}
	if days <= 0 || days > 90 {
		days = 30
	}
	return a.meterStore.DaySummaries(days)
}

// GetAppSummaries returns per-app usage for a date range.
// period: "today", "week", "month"
func (a *App) GetAppSummaries(period string) []metering.AppSummary {
	if a.meterStore == nil {
		return nil
	}
	from, to := periodToRange(period)
	return a.meterStore.AppSummaries(from, to)
}

// GetModelSummaries returns per-model usage for a date range.
func (a *App) GetModelSummaries(period string) []metering.ModelSummary {
	if a.meterStore == nil {
		return nil
	}
	from, to := periodToRange(period)
	return a.meterStore.ModelSummaries(from, to)
}

// GetRecentActivity returns the N most recent API calls.
func (a *App) GetRecentActivity(n int) []metering.ActivityEntry {
	if a.meterStore == nil {
		return nil
	}
	if n <= 0 || n > 100 {
		n = 20
	}
	return a.meterStore.RecentActivity(n)
}

func periodToRange(period string) (time.Time, time.Time) {
	now := time.Now()
	to := now
	var from time.Time
	switch period {
	case "today":
		from = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	case "week":
		from = now.AddDate(0, 0, -7)
	case "month":
		from = now.AddDate(0, -1, 0)
	default:
		from = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	}
	return from, to
}
