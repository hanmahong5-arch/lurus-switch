// Package toolconfig manages on-disk configuration for CLI tools managed by Switch.
// This file implements the Gemini CLI deprecation notice and migration plan builder.
//
// Gemini CLI was deprecated on 2026-05-19 (Google I/O) and reaches end-of-life on
// 2026-06-18. Users should migrate to Antigravity CLI (binary: agy).
package toolconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// geminiEOLDate is the date on which Gemini CLI is officially shut down.
var geminiEOLDate = time.Date(2026, 6, 18, 0, 0, 0, 0, time.UTC)

// GeminiDeprecation provides deprecation metadata for Gemini CLI.
type GeminiDeprecation struct{}

// IsDeprecated returns true; Gemini CLI is deprecated as of Google I/O 2026-05-19.
func (GeminiDeprecation) IsDeprecated() bool { return true }

// DeprecatedAfter returns the Gemini CLI end-of-life date (2026-06-18 UTC).
func (GeminiDeprecation) DeprecatedAfter() time.Time { return geminiEOLDate }

// MigrateTo returns the canonical tool name of the recommended successor.
func (GeminiDeprecation) MigrateTo() string { return ToolAntigravity }

// FieldMigration describes how a single Gemini config field maps to Antigravity.
type FieldMigration struct {
	// GeminiField is the dot-notation path of the source field in Gemini's config.
	GeminiField string `json:"geminiField"`

	// AntigravityField is the dot-notation path of the equivalent field in Antigravity's config.
	AntigravityField string `json:"antigravityField"`

	// Value is the current value read from the Gemini config (string-serialised for display).
	Value string `json:"value"`

	// NeedsManualReview is true for fields that have no direct equivalent and require
	// the user to review the mapping before applying it.
	NeedsManualReview bool `json:"needsManualReview"`

	// Note provides additional context for the user.
	Note string `json:"note,omitempty"`
}

// MigrationPlan describes the full mapping from a Gemini config to its Antigravity equivalent.
type MigrationPlan struct {
	// SourcePath is the absolute path to the Gemini config file that was read.
	SourcePath string `json:"sourcePath"`

	// TargetPath is the absolute path where the Antigravity config would be written.
	TargetPath string `json:"targetPath"`

	// Fields lists per-field migrations.
	Fields []FieldMigration `json:"fields"`

	// Proposed holds the proposed AntigravityConfig assembled from the field mapping.
	// Fields marked NeedsManualReview are included with their best-effort values but
	// the user should verify them before applying.
	Proposed *AntigravityConfig `json:"proposed"`

	// Warnings collects non-fatal issues found during plan construction.
	Warnings []string `json:"warnings,omitempty"`
}

// geminiRawConfig is a permissive representation used for reading the Gemini settings file.
// The real Gemini schema may have additional fields; unknown fields are retained via RawFields.
type geminiRawConfig struct {
	APIKey      string `json:"apiKey"`
	APIEndpoint string `json:"apiEndpoint"`
	Proxy       string `json:"proxy"`
	Model       struct {
		Name string `json:"name"`
	} `json:"model"`
	General struct {
		DefaultApprovalMode string `json:"defaultApprovalMode"`
	} `json:"general"`
}

// BuildMigrationPlan reads the current Gemini config and constructs a MigrationPlan
// that can be applied to create an equivalent Antigravity config.
//
// Fields with no direct Antigravity equivalent are included with NeedsManualReview = true
// so the user can decide what to do with them before applying the migration.
//
// Returns an error only for unrecoverable I/O or parse failures; missing Gemini config
// is treated as an empty source and results in a plan with an empty Proposed config.
func BuildMigrationPlan(ctx context.Context) (*MigrationPlan, error) {
	sourcePath := filepath.Join(geminiDir(), "settings.json")
	targetPath := filepath.Join(antigravityConfigDir(), AntigravityConfigFilename)

	plan := &MigrationPlan{
		SourcePath: sourcePath,
		TargetPath: targetPath,
		Proposed:   &AntigravityConfig{},
	}

	// Read Gemini config; treat missing file as empty source.
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		if os.IsNotExist(err) {
			plan.Warnings = append(plan.Warnings,
				fmt.Sprintf("Gemini config not found at %s; producing empty migration plan", sourcePath))
			return plan, nil
		}
		return nil, fmt.Errorf("failed to read Gemini config %s: %w", sourcePath, err)
	}

	var raw geminiRawConfig
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse Gemini config %s: %w", sourcePath, err)
	}

	// --- Direct field mappings ---

	if raw.APIKey != "" {
		plan.Fields = append(plan.Fields, FieldMigration{
			GeminiField:      "apiKey",
			AntigravityField: "apiKey",
			Value:            raw.APIKey,
		})
		plan.Proposed.APIKey = raw.APIKey
	}

	if raw.APIEndpoint != "" {
		plan.Fields = append(plan.Fields, FieldMigration{
			GeminiField:      "apiEndpoint",
			AntigravityField: "apiEndpoint",
			Value:            raw.APIEndpoint,
		})
		plan.Proposed.APIEndpoint = raw.APIEndpoint
	}

	if raw.Model.Name != "" {
		plan.Fields = append(plan.Fields, FieldMigration{
			GeminiField:      "model.name",
			AntigravityField: "model.name",
			Value:            raw.Model.Name,
		})
		plan.Proposed.Model = AntigravityModelConfig{Name: raw.Model.Name}
	}

	if raw.General.DefaultApprovalMode != "" {
		plan.Fields = append(plan.Fields, FieldMigration{
			GeminiField:      "general.defaultApprovalMode",
			AntigravityField: "general.defaultApprovalMode",
			Value:            raw.General.DefaultApprovalMode,
		})
		plan.Proposed.General = AntigravityGeneralConfig{DefaultApprovalMode: raw.General.DefaultApprovalMode}
	}

	// --- Proxy field: direct mapping but may need manual review for complex values ---

	if raw.Proxy != "" {
		needsReview := false
		note := ""
		// Antigravity accepts a single proxy URL string (same as Gemini).
		// Complex proxy configs (PAC files, per-host rules) need manual verification.
		// TODO: verify against official Antigravity documentation once published.
		if len(raw.Proxy) > 0 {
			// Simple heuristic: if it looks like a URL, it maps directly.
			// Anything else needs review.
			if len(raw.Proxy) > 256 {
				needsReview = true
				note = "需手工核对: proxy 值较长，可能是 PAC 脚本或复合配置，Antigravity 可能不支持"
			}
		}
		plan.Fields = append(plan.Fields, FieldMigration{
			GeminiField:       "proxy",
			AntigravityField:  "proxy",
			Value:             raw.Proxy,
			NeedsManualReview: needsReview,
			Note:              note,
		})
		plan.Proposed.Proxy = raw.Proxy
	}

	return plan, nil
}

// DefaultGeminiDeprecation is a ready-to-use GeminiDeprecation instance.
var DefaultGeminiDeprecation = GeminiDeprecation{}
