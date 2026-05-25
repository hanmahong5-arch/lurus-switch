package main

import (
	"fmt"
	goruntime "runtime"
)

// ============================
// System Info Bindings
// ============================
//
// Host-level metadata + config-directory utilities. The Wails UI uses
// these to render the About/Diagnostics screens and to deep-link the
// user into their on-disk config when they want to hand-edit.

// SystemInfo contains runtime information about the host system
type SystemInfo struct {
	AppVersion string `json:"appVersion"`
	GOOS       string `json:"goos"`
	GOARCH     string `json:"goarch"`
}

// GetSystemInfo returns basic system information
func (a *App) GetSystemInfo() *SystemInfo {
	return &SystemInfo{
		AppVersion: AppVersion,
		GOOS:       goruntime.GOOS,
		GOARCH:     goruntime.GOARCH,
	}
}

// GetConfigDir returns the configuration directory path
func (a *App) GetConfigDir() string {
	if a.store == nil {
		return ""
	}
	return a.store.GetConfigDir()
}

// OpenConfigDir opens the configuration directory in the file explorer
func (a *App) OpenConfigDir() error {
	if a.store == nil {
		return fmt.Errorf("config store not initialized")
	}
	dir := a.store.GetConfigDir()
	return openDirectory(dir)
}
