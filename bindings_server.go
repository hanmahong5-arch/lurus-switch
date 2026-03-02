package main

import (
	"fmt"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"lurus-switch/internal/serverctl"
)

// GetServerStatus returns the current status of the embedded gateway server.
func (a *App) GetServerStatus() serverctl.ServerStatus {
	if a.serverMgr == nil {
		return serverctl.ServerStatus{}
	}
	return a.serverMgr.Status()
}

// StartServer starts the embedded gateway server.
func (a *App) StartServer() error {
	if a.serverMgr == nil {
		return fmt.Errorf("server manager not initialized")
	}
	return a.serverMgr.Start(a.ctx)
}

// StopServer stops the embedded gateway server.
func (a *App) StopServer() error {
	if a.serverMgr == nil {
		return fmt.Errorf("server manager not initialized")
	}
	return a.serverMgr.Stop()
}

// EnsureServerBinary checks for the gateway binary and downloads it if missing.
func (a *App) EnsureServerBinary() error {
	if a.serverMgr == nil {
		return fmt.Errorf("server manager not initialized")
	}
	return a.serverMgr.EnsureBinary(a.ctx, nil)
}

// GetServerURL returns the base URL of the running gateway server, or "" if stopped.
func (a *App) GetServerURL() string {
	if a.serverMgr == nil {
		return ""
	}
	return a.serverMgr.GetURL()
}

// GetServerConfig returns the current gateway server configuration.
func (a *App) GetServerConfig() serverctl.ServerConfig {
	if a.serverMgr == nil {
		return serverctl.ServerConfig{}
	}
	return a.serverMgr.GetConfig()
}

// SaveServerConfig persists a new gateway server configuration.
func (a *App) SaveServerConfig(cfg serverctl.ServerConfig) error {
	if a.serverMgr == nil {
		return fmt.Errorf("server manager not initialized")
	}
	return a.serverMgr.SaveConfig(cfg)
}

// OpenServerAdminPanel opens the gateway admin panel in the default browser.
func (a *App) OpenServerAdminPanel() error {
	url := a.GetServerURL()
	if url == "" {
		return fmt.Errorf("gateway server is not running")
	}
	runtime.BrowserOpenURL(a.ctx, url)
	return nil
}

// GetServerAdminToken returns the stored admin Bearer token for the gateway API.
func (a *App) GetServerAdminToken() string {
	if a.serverMgr == nil {
		return ""
	}
	return a.serverMgr.GetAdminToken()
}
