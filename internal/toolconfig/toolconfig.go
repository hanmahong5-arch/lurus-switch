package toolconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"lurus-switch/internal/installer"
)

// ToolConfigInfo describes a tool's real config file on disk
type ToolConfigInfo struct {
	Tool     string `json:"tool"`
	Path     string `json:"path"`
	Exists   bool   `json:"exists"`
	Language string `json:"language"` // "json" | "toml" | "markdown"
	Content  string `json:"content"`
}

// configDef maps tool name to its config filename and format
type configDef struct {
	dir      func() string // function returning the config directory
	filename string
	language string
}

func claudeDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude")
}

func codexDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".codex")
}

func geminiDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".gemini")
}

func picoClawDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".picoclaw")
}

func nullClawDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".nullclaw")
}

func zeroClawDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".zeroclaw")
}

func openClawDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".openclaw")
}

var toolDefs = map[string]configDef{
	"claude":   {dir: claudeDir, filename: "settings.json", language: "json"},
	"codex":    {dir: codexDir, filename: "config.toml", language: "toml"},
	"gemini":   {dir: geminiDir, filename: "settings.json", language: "json"},
	"picoclaw": {dir: picoClawDir, filename: "config.json", language: "json"},
	"nullclaw": {dir: nullClawDir, filename: "config.json", language: "json"},
	"zeroclaw": {dir: zeroClawDir, filename: "config.toml", language: "toml"},
	"openclaw": {dir: openClawDir, filename: "openclaw.json", language: "json"},
}

// Default config templates for when no config file exists yet
var defaultTemplates = map[string]string{
	"claude": `{
  "$schema": "https://json.schemastore.org/claude-code-settings.json",
  "env": {
    "ANTHROPIC_API_KEY": "",
    "ANTHROPIC_BASE_URL": ""
  },
  "permissions": {
    "allow": [],
    "deny": []
  }
}
`,
	"codex": `#:schema https://openai.com/codex/config-schema.json

model = ""
approval_policy = "on-failure"
sandbox_mode = "workspace-write"

[model_providers.custom]
name = "Custom Proxy"
base_url = ""
env_key = "OPENAI_API_KEY"
wire_api = "chat"
`,
	"gemini": `{
  "$schema": "https://raw.githubusercontent.com/google-gemini/gemini-cli/main/schemas/settings.schema.json",
  "model": {
    "name": "gemini-2.5-flash"
  },
  "general": {
    "defaultApprovalMode": "default"
  },
  "tools": {
    "sandbox": false
  }
}
`,
	"picoclaw": `{
  "model_list": [
    {
      "name": "default",
      "api_base": "",
      "api_key": "",
      "model_name": "` + installer.DefaultPicoClawModel + `"
    }
  ],
  "agents": {
    "defaults": {
      "model_name": "` + installer.DefaultPicoClawModel + `"
    }
  }
}
`,
	"nullclaw": `{
  "model_list": [
    {
      "name": "code-switch",
      "api_base": "",
      "api_key": "",
      "model_name": "` + installer.DefaultNullClawModel + `"
    }
  ],
  "agents": {
    "defaults": {
      "model_name": "` + installer.DefaultNullClawModel + `"
    }
  }
}
`,
	"zeroclaw": `[provider]
type = "anthropic"
api_key = ""
model = "` + installer.DefaultZeroClawModel + `"
base_url = ""

[gateway]
host = "127.0.0.1"
port = 8765
auth_token = ""

[memory]
backend = "sqlite"
path = ""

[security]
sandbox = false
audit_log = false
`,
	"openclaw": `{
  "gateway": {
    "port": 18789,
    "auth_token": ""
  },
  "provider": {
    "type": "anthropic",
    "api_key": "",
    "model": "` + installer.DefaultOpenClawModel + `"
  },
  "channels": {
    "dm_policy": "all"
  },
  "skills": {
    "enabled": []
  }
}
`,
}

// GetConfigPath returns the full path to a tool's config file
func GetConfigPath(tool string) (string, error) {
	def, ok := toolDefs[tool]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s, expected: claude, codex, gemini, picoclaw, nullclaw, zeroclaw, openclaw", tool)
	}
	return filepath.Join(def.dir(), def.filename), nil
}

// ReadConfig reads a tool's real config file from disk
func ReadConfig(tool string) (*ToolConfigInfo, error) {
	def, ok := toolDefs[tool]
	if !ok {
		return nil, fmt.Errorf("unknown tool: %s", tool)
	}

	configPath := filepath.Join(def.dir(), def.filename)
	info := &ToolConfigInfo{
		Tool:     tool,
		Path:     configPath,
		Language: def.language,
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			info.Exists = false
			info.Content = defaultTemplates[tool]
			return info, nil
		}
		return nil, fmt.Errorf("failed to read config %s: %w", configPath, err)
	}

	info.Exists = true
	info.Content = string(data)
	return info, nil
}

// WriteConfig writes content to a tool's real config file
func WriteConfig(tool, content string) error {
	def, ok := toolDefs[tool]
	if !ok {
		return fmt.Errorf("unknown tool: %s", tool)
	}

	configDir := def.dir()
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory %s: %w", configDir, err)
	}

	configPath := filepath.Join(configDir, def.filename)
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write config %s: %w", configPath, err)
	}

	return nil
}

// GetAllConfigPaths returns the config directory for each tool
func GetAllConfigPaths() map[string]string {
	paths := make(map[string]string)
	for tool, def := range toolDefs {
		paths[tool] = filepath.Join(def.dir(), def.filename)
	}
	return paths
}

// OpenConfigDirectory opens the config directory of a tool in the file explorer
func OpenConfigDirectory(tool string) error {
	def, ok := toolDefs[tool]
	if !ok {
		return fmt.Errorf("unknown tool: %s", tool)
	}

	dir := def.dir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	var cmd string
	var args []string
	switch runtime.GOOS {
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

	return execStart(cmd, args...)
}
