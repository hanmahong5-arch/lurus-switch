package main

import (
	"fmt"

	"lurus-switch/internal/audit"
	"lurus-switch/internal/capability"
)

// AuditEntry is the Wails-bound projection of audit.Entry. Wails strips
// the Before/After fields when they're non-trivial Go values, so we
// keep them as `any` here too — the frontend renders them via JSON
// stringify when present.
type AuditEntry = audit.Entry

// AuditFilter mirrors audit.Filter for Wails.
type AuditFilter struct {
	Principal      string `json:"principal"`
	Operation      string `json:"operation"`
	Outcome        string `json:"outcome"`
	OnlyReversible bool   `json:"onlyReversible"`
	OnlyUndone     bool   `json:"onlyUndone"`
	OnlyNotUndone  bool   `json:"onlyNotUndone"`
}

// ListAuditEntries returns up to `limit` recent audit entries, newest first.
func (a *App) ListAuditEntries(limit int, filter AuditFilter) ([]AuditEntry, error) {
	if err := capability.RequireCurrent(capability.CapAuditRead); err != nil {
		return nil, err
	}
	if a.auditJournal == nil {
		return []AuditEntry{}, nil
	}
	return a.auditJournal.List(limit, audit.Filter{
		Principal:      filter.Principal,
		Operation:      filter.Operation,
		Outcome:        filter.Outcome,
		OnlyReversible: filter.OnlyReversible,
		OnlyUndone:     filter.OnlyUndone,
		OnlyNotUndone:  filter.OnlyNotUndone,
	}), nil
}

// GetAuditStats returns aggregate counts for the audit dashboard.
func (a *App) GetAuditStats() (*audit.Stats, error) {
	if err := capability.RequireCurrent(capability.CapAuditRead); err != nil {
		return nil, err
	}
	if a.auditJournal == nil {
		return &audit.Stats{
			ByPrincipal: map[string]int{},
			ByOperation: map[string]int{},
		}, nil
	}
	s := a.auditJournal.Stats()
	return &s, nil
}

// UndoAuditEntry invokes the registered undo handler for a journaled
// mutation. Returns an error if the operation is not reversible, has
// already been undone, or the handler fails (e.g. dependent state has
// since changed in incompatible ways).
func (a *App) UndoAuditEntry(entryID string) error {
	if err := capability.RequireCurrent(capability.CapAuditUndo); err != nil {
		return err
	}
	if a.auditJournal == nil {
		return fmt.Errorf("audit journal not initialized")
	}
	return a.auditJournal.Undo(entryID)
}

// ListAuditCapabilities returns all known capabilities + their human
// descriptions, so the UI can render a key/value reference.
func (a *App) ListAuditCapabilities() map[string]string {
	out := make(map[string]string, len(capability.Description))
	for c, desc := range capability.Description {
		out[string(c)] = desc
	}
	return out
}

// GetCurrentPrincipal exposes the process-wide token identity for the
// audit log header. Useful for "you are operating as: desktop-user" UI.
func (a *App) GetCurrentPrincipal() string {
	return capability.Current().Principal
}
