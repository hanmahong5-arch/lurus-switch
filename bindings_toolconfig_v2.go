package main

import (
	"context"
	"time"

	"lurus-switch/internal/toolconfig"
)

// ============================
// Toolconfig V2 Bindings
// ============================
// Exposes Gemini deprecation workflow and Aider credential injection to the
// frontend. Antigravity and opencode are handled generically via the existing
// ReadToolConfig / SaveToolConfig / AutoConfigureToolForGateway bindings because
// they are registered in toolDefs; this file only adds the specialty endpoints
// that have no generic equivalent.

// GeminiDeprecationStatus is the frontend-friendly representation of Gemini CLI
// deprecation state. All fields are JSON-serialisable primitives so the Wails
// runtime does not need to handle time.Time marshalling on the JS side.
type GeminiDeprecationStatus struct {
	// IsDeprecated is always true; included for forward-compatibility.
	IsDeprecated bool `json:"isDeprecated"`

	// EOLDate is the ISO-8601 date string (YYYY-MM-DD) of the end-of-life date.
	EOLDate string `json:"eolDate"`

	// DaysRemaining is the number of calendar days until EOL from today.
	// Negative means EOL has already passed.
	DaysRemaining int `json:"daysRemaining"`

	// SuccessorTool is the canonical name of the recommended replacement.
	SuccessorTool string `json:"successorTool"`
}

// GetGeminiDeprecationStatus returns current deprecation metadata for Gemini CLI.
func (a *App) GetGeminiDeprecationStatus() (*GeminiDeprecationStatus, error) {
	d := toolconfig.DefaultGeminiDeprecation
	eol := d.DeprecatedAfter()
	now := time.Now().UTC().Truncate(24 * time.Hour)
	days := int(eol.Sub(now).Hours() / 24)
	return &GeminiDeprecationStatus{
		IsDeprecated:  d.IsDeprecated(),
		EOLDate:       eol.Format("2006-01-02"),
		DaysRemaining: days,
		SuccessorTool: d.MigrateTo(),
	}, nil
}

// BuildGeminiMigrationPlan reads the current Gemini config and constructs a
// MigrationPlan describing how each field maps to the successor config format.
func (a *App) BuildGeminiMigrationPlan() (*toolconfig.MigrationPlan, error) {
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()
	return toolconfig.BuildMigrationPlan(ctx)
}

// ApplyGeminiMigration builds a migration plan and writes the resulting config
// to the successor tool's config directory in a single step.
// Returns a ToolConfigResult so the frontend can check .success and display .message.
func (a *App) ApplyGeminiMigration() *ToolConfigResult {
	ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
	defer cancel()

	plan, err := toolconfig.BuildMigrationPlan(ctx)
	if err != nil {
		return &ToolConfigResult{Tool: "gemini", Success: false, Message: "migration plan failed: " + err.Error()}
	}
	if plan.Proposed == nil {
		return &ToolConfigResult{Tool: "gemini", Success: false, Message: "migration plan returned nil config"}
	}

	if err := toolconfig.WriteAntigravityConfig(plan.Proposed); err != nil {
		return &ToolConfigResult{Tool: "gemini", Success: false, Message: "write antigravity config failed: " + err.Error()}
	}

	msg := "Migration applied — antigravity config written to " + plan.TargetPath
	if len(plan.Warnings) > 0 {
		msg += "; warnings: " + plan.Warnings[0]
	}
	return &ToolConfigResult{Tool: "gemini", Success: true, Message: msg}
}

// ── Aider ────────────────────────────────────────────────────────────────────

// AiderDetectResult mirrors toolconfig.AiderDetectResult for Wails binding.
type AiderDetectResult struct {
	Installed bool   `json:"installed"`
	Path      string `json:"path"`
	Version   string `json:"version"`
}

// DetectAider checks whether the aider binary is present on the system.
func (a *App) DetectAider() (*AiderDetectResult, error) {
	r, err := toolconfig.DetectAider()
	if err != nil {
		return nil, err
	}
	return &AiderDetectResult{
		Installed: r.Installed,
		Path:      r.Path,
		Version:   r.Version,
	}, nil
}

// InjectAiderCredentials merges the active relay credentials into
// ~/.aider.conf.yml (and ~/.aider.env for providers that go via env).
// The CredSet is assembled here from the Switch relay store so the frontend
// never has to pass raw API keys.
func (a *App) InjectAiderCredentials() *ToolConfigResult {
	if a.relayStore == nil {
		return &ToolConfigResult{Tool: toolconfig.ToolAider, Success: false, Message: "relay store not initialized"}
	}

	endpoints, err := a.relayStore.ListEndpoints()
	if err != nil {
		return &ToolConfigResult{Tool: toolconfig.ToolAider, Success: false, Message: "failed to read relay endpoints: " + err.Error()}
	}

	// Build CredSet from available relay endpoints.
	// Priority: use the first endpoint key as both anthropic and openai-compatible
	// key since the Switch gateway presents an OpenAI-compatible interface.
	var creds toolconfig.CredSet
	gwURL := a.gatewayBaseURL()

	for _, ep := range endpoints {
		if ep.APIKey == "" {
			continue
		}
		if creds.OpenAIKey == "" {
			creds.OpenAIKey = ep.APIKey
			creds.OpenAIBaseURL = ep.URL
		}
		if creds.AnthropicKey == "" {
			creds.AnthropicKey = ep.APIKey
		}
	}

	// Fallback: inject the gateway URL as the OpenAI-compatible base so that
	// aider can route through Switch even without an explicit relay.
	if gwURL != "" && creds.OpenAIBaseURL == "" {
		creds.OpenAIBaseURL = gwURL + "/v1"
	}

	// A base URL with no API key cannot authenticate aider. Fail loudly here
	// rather than writing a half-configured file and reporting success — the
	// user would otherwise see a "success" toast then be prompted for a key
	// the first time they launch aider.
	if creds.OpenAIKey == "" && creds.AnthropicKey == "" {
		return &ToolConfigResult{
			Tool:    toolconfig.ToolAider,
			Success: false,
			Message: "no relay credentials available — add an endpoint with an API key in Gateway → Relay first",
		}
	}

	if err := toolconfig.InjectCredentials(creds); err != nil {
		return &ToolConfigResult{Tool: toolconfig.ToolAider, Success: false, Message: "credential injection failed: " + err.Error()}
	}

	return &ToolConfigResult{Tool: toolconfig.ToolAider, Success: true, Message: "Aider credentials injected successfully"}
}
