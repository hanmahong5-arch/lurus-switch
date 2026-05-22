package main

import (
	"fmt"
	"sort"

	"lurus-switch/internal/toolmanifest"
)

// ============================
// Tool Manifest Admin Bindings
// ============================
//
// Surface for the Reseller "工具上架" admin tab. Reseller operators edit
// per-tool entries (status / version / platform URLs + SHA256) which are
// persisted as a local-overrides file. The Switch installer respects the
// merged manifest immediately — no GitHub round-trip, no waiting for an
// api.lurus.cn manifest update.

// ToolManifestRow is the per-tool admin view: merged effective entry plus
// a flag indicating whether the operator has overridden the upstream.
type ToolManifestRow struct {
	Name          string                              `json:"name"`
	Type          string                              `json:"type"`
	NpmPackage    string                              `json:"npmPackage,omitempty"`
	LatestVersion string                              `json:"latestVersion"`
	Status        string                              `json:"status"`
	Platforms     map[string]toolmanifest.PlatformAsset `json:"platforms"`
	Overridden    bool                                `json:"overridden"`
}

// ToolManifestAdminView bundles everything the admin tab needs to render
// without juggling two separate bindings.
type ToolManifestAdminView struct {
	Rows           []ToolManifestRow `json:"rows"`
	UpstreamSource string            `json:"upstreamSource"` // "live" | "cache" | "builtin"
	UpdatedAt      string            `json:"updatedAt"`
}

// GetToolManifestAdminView returns the merged manifest annotated with which
// entries the operator has overridden. Tools are returned in stable order.
func (a *App) GetToolManifestAdminView() (*ToolManifestAdminView, error) {
	base := a.loadManifest()
	overrides, err := toolmanifest.LoadOverrides(appDataBaseDir())
	if err != nil {
		return nil, fmt.Errorf("load overrides: %w", err)
	}
	merged := toolmanifest.Merge(base, overrides)

	rows := make([]ToolManifestRow, 0, len(merged.Tools))
	for name, entry := range merged.Tools {
		_, isOverride := overrides.Tools[name]
		platforms := entry.Platforms
		if platforms == nil {
			platforms = map[string]toolmanifest.PlatformAsset{}
		}
		rows = append(rows, ToolManifestRow{
			Name:          name,
			Type:          entry.Type,
			NpmPackage:    entry.NpmPackage,
			LatestVersion: entry.LatestVersion,
			Status:        entry.Status,
			Platforms:     platforms,
			Overridden:    isOverride,
		})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Name < rows[j].Name })

	return &ToolManifestAdminView{
		Rows:      rows,
		UpdatedAt: overrides.UpdatedAt,
	}, nil
}

// SaveToolManifestEntry persists one tool's override entry. Empty platforms
// map / empty status all valid — the operator may want to mark a tool
// "coming-soon" with no URLs yet, or a "stable" tool whose download URL
// hasn't been uploaded.
func (a *App) SaveToolManifestEntry(name string, entry toolmanifest.ToolEntry) error {
	if name == "" {
		return fmt.Errorf("tool name required")
	}
	if err := toolmanifest.SetOverride(appDataBaseDir(), name, entry); err != nil {
		return err
	}
	// Refresh the in-memory manifest so subsequent Install/Topology calls
	// see the edit without restart. Best-effort — refreshManifest re-reads
	// HTTP + cache + overrides.
	go safeGo("refresh-manifest-after-override", func() { a.refreshManifest() })
	return nil
}

// DeleteToolManifestEntry removes the operator's override for one tool,
// reverting to whatever the upstream/built-in manifest says.
func (a *App) DeleteToolManifestEntry(name string) error {
	if name == "" {
		return fmt.Errorf("tool name required")
	}
	if err := toolmanifest.DeleteOverride(appDataBaseDir(), name); err != nil {
		return err
	}
	go safeGo("refresh-manifest-after-override", func() { a.refreshManifest() })
	return nil
}

// ResetToolManifestOverrides clears every operator-set override, restoring
// the upstream manifest entirely.
func (a *App) ResetToolManifestOverrides() error {
	if err := toolmanifest.ResetOverrides(appDataBaseDir()); err != nil {
		return err
	}
	go safeGo("refresh-manifest-after-override", func() { a.refreshManifest() })
	return nil
}
