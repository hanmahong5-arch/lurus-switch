package main

import (
	"fmt"
	"os"

	"lurus-switch/internal/sysenv"
)

// GetSystemEnvironment returns a composite snapshot of PATH entries,
// autostart status, and git configuration.
func (a *App) GetSystemEnvironment() (*sysenv.SystemEnvironment, error) {
	pathEntries, err := sysenv.GetUserPath()
	if err != nil {
		return nil, fmt.Errorf("failed to read PATH: %w", err)
	}

	autostart, err := sysenv.IsAutostartEnabled()
	if err != nil {
		return nil, fmt.Errorf("failed to check autostart: %w", err)
	}

	gitInfo, err := sysenv.DetectGit()
	if err != nil {
		return nil, fmt.Errorf("failed to detect git: %w", err)
	}

	return &sysenv.SystemEnvironment{
		PathEntries: pathEntries,
		Autostart:   autostart,
		Git:         gitInfo,
	}, nil
}

// AddToPath appends a directory to the user's PATH.
func (a *App) AddToPath(dir string) error {
	if dir == "" {
		return fmt.Errorf("directory path must not be empty")
	}
	return sysenv.AddToUserPath(dir)
}

// RemoveFromPath removes a directory from the user's PATH.
func (a *App) RemoveFromPath(dir string) error {
	if dir == "" {
		return fmt.Errorf("directory path must not be empty")
	}
	return sysenv.RemoveFromUserPath(dir)
}

// SetEnvironmentVariable sets a user environment variable.
func (a *App) SetEnvironmentVariable(key, value string) error {
	if key == "" {
		return fmt.Errorf("variable name must not be empty")
	}
	return sysenv.SetEnvVar(key, value)
}

// DeleteEnvironmentVariable removes a user environment variable.
func (a *App) DeleteEnvironmentVariable(key string) error {
	if key == "" {
		return fmt.Errorf("variable name must not be empty")
	}
	return sysenv.DeleteEnvVar(key)
}

// EnableAutostart configures the application to start automatically on login.
func (a *App) EnableAutostart(args string) error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to determine executable path: %w", err)
	}
	return sysenv.EnableAutostart(exePath, args)
}

// DisableAutostart removes the autostart configuration.
func (a *App) DisableAutostart() error {
	return sysenv.DisableAutostart()
}

// GetRollbackHistory returns all rollback entries, newest first.
func (a *App) GetRollbackHistory() ([]sysenv.RollbackEntry, error) {
	return sysenv.ListRollbacks()
}

// ApplyRollback reverses a previous system environment change.
func (a *App) ApplyRollback(id string) error {
	if id == "" {
		return fmt.Errorf("rollback ID must not be empty")
	}
	return sysenv.ApplyRollback(id)
}
