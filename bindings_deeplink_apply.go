package main

import (
	"lurus-switch/internal/deeplink"
)

// ApplyDeepLinkImport decodes a deep-link payload and persists it to the
// appropriate local store:
//   - type "mcp"    → mcp.Store.SavePreset
//   - type "prompt" → promptlib.Store.SavePrompt
//
// Returns a human-readable summary on success. Unknown types (including
// "skill") return a clear "not yet supported" error and never silently
// succeed. "provider" is handled entirely on the frontend (gateway
// settings) and is rejected here if mistakenly forwarded.
func (a *App) ApplyDeepLinkImport(payload deeplink.Payload) (string, error) {
	return deeplink.Apply(payload, a.mcpStr, a.promptStr)
}
