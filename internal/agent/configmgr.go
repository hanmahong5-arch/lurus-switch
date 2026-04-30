package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ConfigManager handles per-agent configuration directories.
// Each agent gets its own isolated config directory under the app data path,
// enabling multiple instances of the same tool with different settings.
type ConfigManager struct {
	baseDir string // e.g. %APPDATA%/lurus-switch/agent-configs
}

// NewConfigManager creates a config manager rooted at dataDir/agent-configs.
func NewConfigManager(dataDir string) (*ConfigManager, error) {
	base := filepath.Join(dataDir, "agent-configs")
	if err := os.MkdirAll(base, 0755); err != nil {
		return nil, fmt.Errorf("create agent-configs dir: %w", err)
	}
	return &ConfigManager{baseDir: base}, nil
}

// AgentDir returns the config directory for a specific agent.
// Creates the directory if it doesn't exist.
func (m *ConfigManager) AgentDir(agentID string) (string, error) {
	dir := filepath.Join(m.baseDir, agentID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create agent config dir: %w", err)
	}
	return dir, nil
}

// WriteJSON writes a JSON config file to an agent's config directory.
func (m *ConfigManager) WriteJSON(agentID, filename string, data any) (string, error) {
	dir, err := m.AgentDir(agentID)
	if err != nil {
		return "", err
	}

	path := filepath.Join(dir, filename)
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(path, b, 0644); err != nil {
		return "", fmt.Errorf("write config file: %w", err)
	}

	return path, nil
}

// WriteRaw writes raw bytes to an agent's config directory.
func (m *ConfigManager) WriteRaw(agentID, filename string, data []byte) (string, error) {
	dir, err := m.AgentDir(agentID)
	if err != nil {
		return "", err
	}

	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", fmt.Errorf("write config file: %w", err)
	}

	return path, nil
}

// ReadJSON reads a JSON config file from an agent's config directory.
func (m *ConfigManager) ReadJSON(agentID, filename string, dest any) error {
	dir := filepath.Join(m.baseDir, agentID)
	path := filepath.Join(dir, filename)

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}

	return json.Unmarshal(data, dest)
}

// Exists checks if an agent has a config directory with the given file.
func (m *ConfigManager) Exists(agentID, filename string) bool {
	path := filepath.Join(m.baseDir, agentID, filename)
	_, err := os.Stat(path)
	return err == nil
}

// Remove deletes an agent's entire config directory.
func (m *ConfigManager) Remove(agentID string) error {
	dir := filepath.Join(m.baseDir, agentID)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil // already gone
	}
	return os.RemoveAll(dir)
}

// ListAgentDirs returns all agent IDs that have config directories.
func (m *ConfigManager) ListAgentDirs() ([]string, error) {
	entries, err := os.ReadDir(m.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var ids []string
	for _, e := range entries {
		if e.IsDir() {
			ids = append(ids, e.Name())
		}
	}
	return ids, nil
}

// ConfigFilename returns the expected config filename for a tool type.
func ConfigFilename(toolType ToolType) string {
	switch toolType {
	case ToolClaude:
		return "settings.json"
	case ToolCodex:
		return "config.toml"
	case ToolGemini:
		return "settings.json"
	case ToolOpenClaw:
		return "openclaw.json"
	case ToolZeroClaw:
		return "config.toml"
	case ToolPicoClaw:
		return "config.json"
	case ToolNullClaw:
		return "config.json"
	default:
		return "config.json"
	}
}
