package envmgr

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// KeyEntry describes a discovered API key/secret for a tool
type KeyEntry struct {
	Tool        string `json:"tool"`
	Key         string `json:"key"`         // env var or config key name
	MaskedValue string `json:"maskedValue"` // first 4 chars + "****"
	Source      string `json:"source"`      // file path where found
}

// Manager reads API keys from tool config files
type Manager struct{}

// NewManager creates a new environment key manager
func NewManager() *Manager {
	return &Manager{}
}

// ListAllKeys scans config files for all requested tools and returns masked key entries
func (m *Manager) ListAllKeys(tools []string) ([]KeyEntry, error) {
	var entries []KeyEntry
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	for _, tool := range tools {
		switch tool {
		case "claude":
			e := readJSONKey(filepath.Join(home, ".claude", "settings.json"), tool, "env.ANTHROPIC_API_KEY")
			entries = append(entries, e...)

		case "codex":
			e := readTOMLKey(filepath.Join(home, ".codex", "config.toml"), tool, "OPENAI_API_KEY")
			entries = append(entries, e...)

		case "gemini":
			e := readJSONKey(filepath.Join(home, ".gemini", "settings.json"), tool, "apiKey")
			entries = append(entries, e...)

		case "picoclaw":
			e := readJSONModelListKeys(filepath.Join(home, ".picoclaw", "config.json"), tool)
			entries = append(entries, e...)

		case "nullclaw":
			e := readJSONModelListKeys(filepath.Join(home, ".nullclaw", "config.json"), tool)
			entries = append(entries, e...)

		case "zeroclaw":
			e := readZeroClawKey(filepath.Join(home, ".zeroclaw", "config.toml"), tool)
			entries = append(entries, e...)

		case "openclaw":
			e := readJSONKey(filepath.Join(home, ".openclaw", "openclaw.json"), tool, "provider.api_key")
			entries = append(entries, e...)
		}
	}

	return entries, nil
}

// UpdateKey updates an API key in a tool's config file
func (m *Manager) UpdateKey(tool, key, value string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	switch tool {
	case "claude":
		return updateJSONKey(filepath.Join(home, ".claude", "settings.json"), "env.ANTHROPIC_API_KEY", value)
	case "gemini":
		return updateJSONKey(filepath.Join(home, ".gemini", "settings.json"), "apiKey", value)
	default:
		return fmt.Errorf("key update not supported for tool: %s", tool)
	}
}

// maskValue returns the first 4 chars followed by ****
func maskValue(v string) string {
	if v == "" {
		return ""
	}
	if len(v) <= 4 {
		return "****"
	}
	return v[:4] + "****"
}

// readJSONKey reads a JSON config file and extracts a nested key using dot-notation
func readJSONKey(path, tool, dotKey string) []KeyEntry {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var root map[string]interface{}
	if err := json.Unmarshal(data, &root); err != nil {
		return nil
	}

	val := nestedGet(root, strings.Split(dotKey, "."))
	if val == "" {
		return nil
	}
	return []KeyEntry{{Tool: tool, Key: dotKey, MaskedValue: maskValue(val), Source: path}}
}

// readTOMLKey checks the process environment for a tool's API key env var
func readTOMLKey(_, tool, envVar string) []KeyEntry {
	val := os.Getenv(envVar)
	if val == "" {
		return nil
	}
	return []KeyEntry{{Tool: tool, Key: envVar, MaskedValue: maskValue(val), Source: "environment"}}
}

// readJSONModelListKeys extracts api_key from all model_list entries
func readJSONModelListKeys(path, tool string) []KeyEntry {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var cfg struct {
		ModelList []struct {
			Name   string `json:"name"`
			APIKey string `json:"api_key"`
		} `json:"model_list"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil
	}

	var entries []KeyEntry
	for _, m := range cfg.ModelList {
		if m.APIKey == "" {
			continue
		}
		entries = append(entries, KeyEntry{
			Tool:        tool,
			Key:         fmt.Sprintf("model_list[%s].api_key", m.Name),
			MaskedValue: maskValue(m.APIKey),
			Source:      path,
		})
	}
	return entries
}

// nestedGet traverses a nested map using a split key path
func nestedGet(m map[string]interface{}, keys []string) string {
	if len(keys) == 0 {
		return ""
	}
	v, ok := m[keys[0]]
	if !ok {
		return ""
	}
	if len(keys) == 1 {
		if s, ok := v.(string); ok {
			return s
		}
		return ""
	}
	if sub, ok := v.(map[string]interface{}); ok {
		return nestedGet(sub, keys[1:])
	}
	return ""
}

// updateJSONKey updates a dot-notation key in a JSON file
func updateJSONKey(path, dotKey, value string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			data = []byte("{}")
		} else {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}
	}

	var root map[string]interface{}
	if err := json.Unmarshal(data, &root); err != nil {
		root = make(map[string]interface{})
	}

	keys := strings.Split(dotKey, ".")
	nestedSet(root, keys, value)

	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return os.WriteFile(path, out, 0600)
}

// nestedSet sets a value at a dot-notation path in a nested map
func nestedSet(m map[string]interface{}, keys []string, value string) {
	if len(keys) == 1 {
		m[keys[0]] = value
		return
	}
	sub, ok := m[keys[0]].(map[string]interface{})
	if !ok {
		sub = make(map[string]interface{})
		m[keys[0]] = sub
	}
	nestedSet(sub, keys[1:], value)
}

// readZeroClawKey reads the provider.api_key from a ZeroClaw TOML config file.
// ZeroClaw uses TOML; we parse it as a generic map to avoid importing the config package.
func readZeroClawKey(path, tool string) []KeyEntry {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	// Use simple JSON-compatible parsing: unmarshal TOML as map via encoding/json lookalike.
	// Since BurntSushi/toml is available only in the installer package, we do a lightweight
	// text scan: look for api_key = "..." under [provider].
	content := string(data)
	val := extractTOMLSectionKey(content, "provider", "api_key")
	if val == "" {
		return nil
	}
	return []KeyEntry{{Tool: tool, Key: "provider.api_key", MaskedValue: maskValue(val), Source: path}}
}

// extractTOMLSectionKey extracts a string value from a TOML section without external dependencies.
// It handles the common case: [section]\nkey = "value"
func extractTOMLSectionKey(content, section, key string) string {
	lines := strings.Split(content, "\n")
	inSection := false
	sectionHeader := "[" + section + "]"

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == sectionHeader {
			inSection = true
			continue
		}
		// Stop at next section
		if inSection && strings.HasPrefix(trimmed, "[") {
			break
		}
		if inSection && strings.HasPrefix(trimmed, key+" =") {
			parts := strings.SplitN(trimmed, "=", 2)
			if len(parts) == 2 {
				val := strings.TrimSpace(parts[1])
				val = strings.Trim(val, `"'`)
				return val
			}
		}
	}
	return ""
}

