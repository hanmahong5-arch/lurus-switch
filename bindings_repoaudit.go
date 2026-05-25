package main

import (
	"fmt"

	"lurus-switch/internal/repoaudit"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// ============================
// Repo Trust Audit Bindings
// ============================
//
// Scans an arbitrary project directory for AI-CLI config overrides
// (CLAUDE.md, .codex/, etc.) that could indicate prompt-injection or
// credential-exfiltration. The UI lets the user review findings and
// quarantine files before launching any CLI inside the repo.

// AuditRepo scans a project directory for AI-CLI config overrides that
// could indicate prompt-injection or credential-exfiltration vectors.
// Powers the "Repo Trust Audit" UI; the user reviews findings before
// launching any CLI inside the repo.
func (a *App) AuditRepo(path string) (*repoaudit.AuditReport, error) {
	if path == "" {
		return nil, fmt.Errorf("path is required")
	}
	return repoaudit.Audit(path)
}

// PickRepoAndAudit opens the native directory picker and immediately
// runs the audit on the chosen directory. Returns nil (no error) if the
// user cancels — the UI distinguishes "no result" from "error" by the
// presence of the report.
func (a *App) PickRepoAndAudit() (*repoaudit.AuditReport, error) {
	dir, err := wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Pick a project directory to audit",
	})
	if err != nil {
		return nil, err
	}
	if dir == "" {
		return nil, nil
	}
	return repoaudit.Audit(dir)
}

// QuarantineFile renames a file flagged by AuditRepo so the AI CLI no
// longer reads it. Returns the new path so the UI can show the user
// where the file went and how to restore it later.
func (a *App) QuarantineFile(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path is required")
	}
	return repoaudit.Quarantine(path)
}
