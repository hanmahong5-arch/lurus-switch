// Package preset provides one-click configuration templates for each supported tool.
// Presets are purely in-memory; applying one fills a config struct that the caller
// then validates and saves via the normal SaveToolConfig path.
package preset

// Preset describes a single named configuration template.
type Preset struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}
