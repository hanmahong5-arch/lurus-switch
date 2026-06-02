package deeplink

import (
	"encoding/json"
	"errors"
	"fmt"

	"lurus-switch/internal/mcp"
	"lurus-switch/internal/promptlib"
)

// MCPSaver is satisfied by *mcp.Store.
type MCPSaver interface {
	SavePreset(p mcp.MCPPreset) error
}

// PromptSaver is satisfied by *promptlib.Store.
type PromptSaver interface {
	SavePrompt(p promptlib.Prompt) error
}

// Apply decodes payload.Data into the appropriate store type and persists it.
// It returns a human-readable summary on success.
//
// Supported types: "mcp", "prompt".
// "skill" and any other type returns a "not yet supported" error.
// "provider" is handled on the frontend (gateway settings) and is not
// routed here — callers should guard against it before calling Apply.
func Apply(payload Payload, mcpStore MCPSaver, promptStore PromptSaver) (string, error) {
	switch payload.Type {
	case "mcp":
		return applyMCP(payload.Data, mcpStore)
	case "prompt":
		return applyPrompt(payload.Data, promptStore)
	default:
		return "", fmt.Errorf("deeplink apply: type %q not yet supported", payload.Type)
	}
}

// applyMCP unmarshals data into an MCPPreset, validates required fields, and
// saves it via the provided store.
func applyMCP(data json.RawMessage, store MCPSaver) (string, error) {
	if store == nil {
		return "", errors.New("deeplink apply: mcp store not initialized")
	}
	var p mcp.MCPPreset
	if err := json.Unmarshal(data, &p); err != nil {
		return "", fmt.Errorf("deeplink apply: invalid mcp payload: %w", err)
	}
	if p.Name == "" {
		return "", errors.New("deeplink apply: mcp preset missing required field: name")
	}
	if p.Server.Type == "" {
		return "", errors.New("deeplink apply: mcp preset missing required field: server.type")
	}
	if err := store.SavePreset(p); err != nil {
		return "", fmt.Errorf("deeplink apply: failed to save mcp preset: %w", err)
	}
	return fmt.Sprintf("MCP preset %q imported", p.Name), nil
}

// applyPrompt unmarshals data into a Prompt, validates required fields, and
// saves it via the provided store.
func applyPrompt(data json.RawMessage, store PromptSaver) (string, error) {
	if store == nil {
		return "", errors.New("deeplink apply: prompt store not initialized")
	}
	var p promptlib.Prompt
	if err := json.Unmarshal(data, &p); err != nil {
		return "", fmt.Errorf("deeplink apply: invalid prompt payload: %w", err)
	}
	if p.Name == "" {
		return "", errors.New("deeplink apply: prompt missing required field: name")
	}
	if p.Content == "" {
		return "", errors.New("deeplink apply: prompt missing required field: content")
	}
	if err := store.SavePrompt(p); err != nil {
		return "", fmt.Errorf("deeplink apply: failed to save prompt: %w", err)
	}
	return fmt.Sprintf("prompt %q imported", p.Name), nil
}
