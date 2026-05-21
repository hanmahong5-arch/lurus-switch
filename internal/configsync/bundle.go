// Package configsync exports and imports a Switch configuration bundle so a
// user can move their setup between machines. The bundle is a zip with a
// manifest.json plus one entry (file or directory tree) per component.
//
// The package is named configsync rather than "sync" to avoid shadowing the
// stdlib sync package at call sites.
//
// Secret handling: by default API keys are redacted out of the bundle
// (includeKeys=false). The denylist is intentionally conservative — see
// redact.go — and reviewers must walk the secrets checklist when adding a
// new component, since a missed key would be exported in plaintext.
package configsync

import "time"

// nowFunc is the clock used for ExportedAt / backup timestamps. Overridable
// in tests so bundle metadata is deterministic.
var nowFunc = time.Now

// SchemaVersion is the bundle format version. Bumped on breaking changes;
// an importer refuses a bundle whose SchemaVersion it doesn't understand.
const SchemaVersion = 1

const manifestEntry = "manifest.json"

// Component keys. Keep in sync with componentSpecs in exporter.go.
const (
	CompAppSettings     = "app-settings"
	CompCustomProviders = "custom-providers"
	CompSnapshots       = "snapshots"
	CompToolConfigs     = "tool-configs"
	CompMCPPresets      = "mcp-presets"
	CompPrompts         = "prompts"
)

// AllComponents is the canonical export order.
var AllComponents = []string{
	CompAppSettings,
	CompCustomProviders,
	CompToolConfigs,
	CompMCPPresets,
	CompPrompts,
	CompSnapshots,
}

// Manifest is the bundle's self-description, stored as manifest.json.
type Manifest struct {
	SchemaVersion int       `json:"schemaVersion"`
	ExportedAt    time.Time `json:"exportedAt"`
	AppVersion    string    `json:"appVersion"`
	IncludesKeys  bool      `json:"includesKeys"`
	Components    []string  `json:"components"` // components actually present
}

// Dirs holds the filesystem roots the exporter reads from and the importer
// writes to. Injecting them (instead of calling os.UserHomeDir directly)
// keeps Export/Import unit-testable against temp dirs.
type Dirs struct {
	AppData string // e.g. %APPDATA%\lurus-switch
	Home    string // user home — tool configs live under ~/.claude etc.
}

// BundlePreview is the result of inspecting a bundle without applying it.
type BundlePreview struct {
	Manifest   Manifest           `json:"manifest"`
	Components []ComponentPreview `json:"components"`
	Err        string             `json:"err,omitempty"` // non-fatal warning
}

// ComponentPreview describes what importing one component would do.
type ComponentPreview struct {
	Key      string `json:"key"`
	InBundle bool   `json:"inBundle"`
	Action   string `json:"action"` // "overwrite" | "create" | "skip"
	Detail   string `json:"detail,omitempty"`
}
