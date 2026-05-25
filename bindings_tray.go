package main

import (
	"context"
	"time"

	"lurus-switch/internal/tray"
)

// ============================
// Tray Subsystem Callbacks
// ============================
//
// Provider functions wired into tray.Manager at startup so the menu can
// pull live quota + gateway status without owning a billing/gateway
// dependency itself. Not Wails bindings — they're called by the tray
// goroutine, but they live with the rest of the tray surface so app.go
// stays focused on lifecycle.

// trayQuotaSnapshot is the tray's quota-usage provider. Returns UsedPercent = -1
// when the billing client is unavailable so the tray can render an "unknown" tier.
func (a *App) trayQuotaSnapshot() tray.QuotaSnapshot {
	client, err := a.ensureBillingClient()
	if err != nil || client == nil {
		return tray.QuotaSnapshot{UsedPercent: -1}
	}
	ctx, cancel := context.WithTimeout(a.ctx, 5*time.Second)
	defer cancel()
	info, err := client.GetUserInfo(ctx)
	if err != nil || info == nil || info.Quota == 0 {
		return tray.QuotaSnapshot{UsedPercent: -1}
	}
	pct := float64(info.UsedQuota) / float64(info.Quota) * 100
	return tray.QuotaSnapshot{UsedPercent: pct}
}

// trayGatewayStatus is the tray's gateway-status provider.
func (a *App) trayGatewayStatus() tray.GatewayStatus {
	if a.gatewaySrv == nil {
		return tray.GatewayStatus{}
	}
	s := a.gatewaySrv.Status()
	return tray.GatewayStatus{Running: s.Running, Port: s.Port}
}
