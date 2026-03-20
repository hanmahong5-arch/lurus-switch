package main

import (
	"fmt"

	"lurus-switch/internal/appreg"
)

// ============================
// App Registry Methods
// ============================

// GetRegisteredApps returns all registered apps (builtin + user).
func (a *App) GetRegisteredApps() []*appreg.App {
	if a.appRegistry == nil {
		return nil
	}
	return a.appRegistry.List()
}

// GetRegisteredApp returns a single app by ID.
func (a *App) GetRegisteredApp(id string) *appreg.App {
	if a.appRegistry == nil {
		return nil
	}
	return a.appRegistry.Get(id)
}

// RegisterApp creates a new user-defined app and returns it with its token.
func (a *App) RegisterApp(name, icon, description string) (*appreg.App, error) {
	if a.appRegistry == nil {
		return nil, fmt.Errorf("app registry not initialized")
	}
	return a.appRegistry.Register(name, icon, description)
}

// DeleteApp removes a user-registered app.
func (a *App) DeleteApp(id string) error {
	if a.appRegistry == nil {
		return fmt.Errorf("app registry not initialized")
	}
	return a.appRegistry.Delete(id)
}

// ResetAppToken generates a new token for an app, invalidating the old one.
func (a *App) ResetAppToken(id string) (string, error) {
	if a.appRegistry == nil {
		return "", fmt.Errorf("app registry not initialized")
	}
	return a.appRegistry.ResetToken(id)
}

// SetAppConnected marks an app as connected or disconnected.
func (a *App) SetAppConnected(id string, connected bool) error {
	if a.appRegistry == nil {
		return fmt.Errorf("app registry not initialized")
	}
	return a.appRegistry.SetConnected(id, connected)
}

// GetConnectedAppCount returns the number of apps currently connected to the gateway.
func (a *App) GetConnectedAppCount() int {
	if a.appRegistry == nil {
		return 0
	}
	return a.appRegistry.ConnectedCount()
}
