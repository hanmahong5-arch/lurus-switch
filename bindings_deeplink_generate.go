package main

import (
	"encoding/json"
	"fmt"

	"lurus-switch/internal/deeplink"
)

// GenerateImportLink builds a switch:// deep-link URL from a type string and a
// JSON object string.  The returned URL can be shared with other Switch users;
// when opened it is decoded by Parse and emitted as a "deeplink:import" event.
//
// linkType must be one of: "provider", "mcp", "prompt", "skill".
// dataJSON must be a JSON object string, e.g. `{"model":"gpt-4"}`.
func (a *App) GenerateImportLink(linkType string, dataJSON string) (string, error) {
	if linkType == "" {
		return "", fmt.Errorf("deeplink: linkType is required")
	}
	if dataJSON == "" {
		return "", fmt.Errorf("deeplink: dataJSON is required")
	}

	// Validate that dataJSON is well-formed JSON before constructing a Payload.
	if !json.Valid([]byte(dataJSON)) {
		return "", fmt.Errorf("deeplink: dataJSON is not valid JSON")
	}

	p := &deeplink.Payload{
		Type: linkType,
		Data: json.RawMessage(dataJSON),
	}

	return deeplink.Generate(p)
}
