package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"

	"lurus-switch/internal/mcp"
)

// appDataBaseDir returns the platform-specific base directory for app data.
// Windows: %APPDATA%\lurus-switch
// macOS:   ~/Library/Application Support/lurus-switch
// Linux:   ~/.lurus-switch
func appDataBaseDir() string {
	home, _ := os.UserHomeDir()
	switch goruntime.GOOS {
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(appData, "lurus-switch")
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "lurus-switch")
	default:
		return filepath.Join(home, ".lurus-switch")
	}
}

// openDirectory opens a directory in the system file explorer
func openDirectory(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	var cmd string
	var args []string

	switch goruntime.GOOS {
	case "windows":
		cmd = "explorer"
		args = []string{filepath.FromSlash(dir)}
	case "darwin":
		cmd = "open"
		args = []string{dir}
	default:
		cmd = "xdg-open"
		args = []string{dir}
	}

	return exec.Command(cmd, args...).Start()
}

// applyMCPToTool upserts an MCP server definition into a tool's settings.json
func applyMCPToTool(tool string, server mcp.MCPServer) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	var settingsPath string
	switch tool {
	case "claude":
		settingsPath = filepath.Join(home, ".claude", "settings.json")
	case "gemini":
		settingsPath = filepath.Join(home, ".gemini", "settings.json")
	default:
		return fmt.Errorf("MCP server application not supported for tool: %s", tool)
	}

	return writeJSONSection(settingsPath, "mcpServers."+server.Name, server)
}

// readJSONSection reads a top-level key from a JSON settings file
func readJSONSection(path, key string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]interface{}{}, nil
		}
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	var root map[string]interface{}
	if err := jsonDecodeAny(data, &root); err != nil {
		return nil, err
	}

	if v, ok := root[key]; ok {
		if m, ok := v.(map[string]interface{}); ok {
			return m, nil
		}
	}
	return map[string]interface{}{}, nil
}

// writeJSONSection writes a value to a dot-notation key in a JSON settings file
func writeJSONSection(path, dotKey string, value interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}
		data = []byte("{}")
	}

	var root map[string]interface{}
	if err := jsonDecodeAny(data, &root); err != nil {
		root = make(map[string]interface{})
	}

	keys := splitDotKey(dotKey)
	nestedSetAny(root, keys, value)

	out, err := jsonEncodeIndent(root)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return os.WriteFile(path, out, 0644)
}

// jsonDecodeAny unmarshals JSON bytes into a map
func jsonDecodeAny(data []byte, v *map[string]interface{}) error {
	return json.Unmarshal(data, v)
}

// jsonEncodeIndent marshals a value to indented JSON bytes
func jsonEncodeIndent(v interface{}) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}

func splitDotKey(key string) []string {
	var parts []string
	var buf []byte
	for i := 0; i < len(key); i++ {
		if key[i] == '.' {
			if len(buf) > 0 {
				parts = append(parts, string(buf))
				buf = buf[:0]
			}
		} else {
			buf = append(buf, key[i])
		}
	}
	if len(buf) > 0 {
		parts = append(parts, string(buf))
	}
	return parts
}

func nestedSetAny(m map[string]interface{}, keys []string, value interface{}) {
	if len(keys) == 1 {
		m[keys[0]] = value
		return
	}
	sub, ok := m[keys[0]].(map[string]interface{})
	if !ok {
		sub = make(map[string]interface{})
		m[keys[0]] = sub
	}
	nestedSetAny(sub, keys[1:], value)
}
