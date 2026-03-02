package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"lurus-switch/internal/analytics"
	"lurus-switch/internal/process"
)

// ============================
// Admin & Health Methods (Phase I.1)
// ============================

// PingLurusAPI checks if the configured API server is reachable
func (a *App) PingLurusAPI() (bool, error) {
	if a.proxyMgr == nil {
		return false, fmt.Errorf("proxy manager not initialized")
	}
	s := a.proxyMgr.GetSettings()
	if s.APIEndpoint == "" {
		return false, fmt.Errorf("API endpoint not configured")
	}

	target := s.APIEndpoint + "/api/v1/health"
	ctx, cancel := context.WithTimeout(a.ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return false, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, nil
	}
	resp.Body.Close()
	return resp.StatusCode < 500, nil
}

// OpenAdminPanel opens the configured API's admin panel in the system browser
func (a *App) OpenAdminPanel() error {
	if a.proxyMgr == nil {
		return fmt.Errorf("proxy manager not initialized")
	}
	s := a.proxyMgr.GetSettings()
	if s.APIEndpoint == "" {
		return fmt.Errorf("API endpoint not configured")
	}
	adminURL := s.APIEndpoint + "/admin"
	runtime.BrowserOpenURL(a.ctx, adminURL)
	return nil
}

// ============================
// Analytics Methods (Phase I.2)
// ============================

// GetUsageReport returns a usage summary for the past N days
func (a *App) GetUsageReport(days int) (*analytics.UsageReport, error) {
	if a.tracker == nil {
		return &analytics.UsageReport{
			ToolActions: make(map[string]map[string]int),
			DailyActive: make(map[string]int),
			ConfigCounts: make(map[string]int),
		}, nil
	}
	return a.tracker.GetReport(days)
}

// ============================
// Process Monitor Methods (Phase E)
// ============================

// ListCLIProcesses returns running CLI tool processes
func (a *App) ListCLIProcesses() ([]process.ProcessInfo, error) {
	return a.processMon.ListCLIProcesses(a.ctx)
}

// KillCLIProcess terminates a process by PID
func (a *App) KillCLIProcess(pid int) error {
	return a.processMon.KillProcess(a.ctx, pid)
}

// LaunchTool starts a CLI tool in a managed session and returns the session ID
func (a *App) LaunchTool(tool string, args []string) (string, error) {
	return a.processMon.LaunchTool(a.ctx, tool, args)
}

// GetToolOutput returns recent output lines from a managed session
func (a *App) GetToolOutput(sessionID string, maxLines int) ([]string, error) {
	return a.processMon.GetOutput(sessionID, maxLines)
}

// StopToolSession terminates a managed session
func (a *App) StopToolSession(sessionID string) error {
	return a.processMon.StopSession(sessionID)
}
