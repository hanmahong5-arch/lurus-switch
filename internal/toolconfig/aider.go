// Package toolconfig manages on-disk configuration for CLI tools managed by Switch.
// This file handles the Aider pairing tool (https://github.com/paul-gauthier/aider),
// a Python-based terminal coding assistant installed via pip.
//
// Configuration lives in ~/.aider.conf.yml (YAML). Aider supports per-provider API keys
// as top-level YAML fields (anthropic-api-key, openai-api-key) plus base URL overrides.
// Keys for other providers are injected via the .env mechanism (GEMINI_API_KEY, etc.).
package toolconfig

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	// ToolAider is the canonical tool name used throughout Switch.
	ToolAider = "aider"

	// aiderConfigFilename is the Aider YAML config file name.
	aiderConfigFilename = ".aider.conf.yml"

	// aiderEnvFilename is the optional .env file for additional API keys.
	aiderEnvFilename = ".aider.env"
)

// CredSet holds provider API keys to inject into Aider's configuration.
// Each field is optional; an empty string means "do not touch this key in config".
//
// Switch callers should populate this from their own credential store
// (e.g. relay.RelayEndpoint.APIKey or envmgr.Manager) and pass it to
// InjectCredentials. This type does NOT read from internal/auth itself
// so the injection path stays decoupled from session state.
type CredSet struct {
	// AnthropicKey is written to YAML field: anthropic-api-key
	AnthropicKey string
	// OpenAIKey is written to YAML field: openai-api-key
	OpenAIKey string
	// OpenAIBaseURL optionally overrides the OpenAI-compatible endpoint.
	// Written to YAML field: openai-api-base
	OpenAIBaseURL string
	// GeminiKey is written to .env as GEMINI_API_KEY (Aider YAML only supports
	// OpenAI and Anthropic keys; all others go to the .env file).
	GeminiKey string
	// DeepSeekKey is written to .env as DEEPSEEK_API_KEY.
	DeepSeekKey string
}

// AiderDetectResult holds the outcome of probing for an aider installation.
type AiderDetectResult struct {
	Installed bool
	Path      string
	Version   string
}

// DetectAider checks whether the aider binary is available on this machine.
// It searches PATH first, then the Python user-scripts directories common
// on Windows (per-user pip install without admin rights).
func DetectAider() (*AiderDetectResult, error) {
	r := &AiderDetectResult{}

	path, err := findAiderBinary()
	if err != nil {
		// Not found — return a clean "not installed" result, not an error.
		return r, nil
	}

	r.Path = path
	r.Installed = true

	// Best-effort version probe; non-zero exit is not fatal.
	out, err := exec.Command(path, "--version").Output()
	if err == nil {
		r.Version = parseAiderVersion(strings.TrimSpace(string(out)))
	}
	return r, nil
}

// AiderConfigPath returns the absolute path to ~/.aider.conf.yml.
func AiderConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("aider: resolve home dir: %w", err)
	}
	return filepath.Join(home, aiderConfigFilename), nil
}

// ReadAiderConfig reads ~/.aider.conf.yml into a generic YAML map.
// If the file does not exist an empty map is returned without error — this
// matches the "first run" case where no config has been created yet.
func ReadAiderConfig() (map[string]any, error) {
	path, err := AiderConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]any), nil
		}
		return nil, fmt.Errorf("aider: read config %s: %w", path, err)
	}

	var cfg map[string]any
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("aider: parse config %s: %w", path, err)
	}
	if cfg == nil {
		cfg = make(map[string]any)
	}
	return cfg, nil
}

// WriteAiderConfig serialises cfg to ~/.aider.conf.yml.
// The directory (~/) is guaranteed to exist; the file is written 0600.
func WriteAiderConfig(cfg map[string]any) error {
	if cfg == nil {
		return fmt.Errorf("aider: cfg must not be nil")
	}

	path, err := AiderConfigPath()
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("aider: marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("aider: write config %s: %w", path, err)
	}
	return nil
}

// InjectCredentials merges the non-empty fields of creds into ~/.aider.conf.yml
// without disturbing any other keys the user may have set. It then writes
// .env-style keys for providers that Aider does not support in YAML.
//
// The function is idempotent: re-running with the same credentials produces
// the same file state.
//
// Callers should obtain creds by reading from Switch's own credential store
// (e.g. relay endpoints, envmgr, or app settings) rather than constructing
// them from raw user input.
func InjectCredentials(creds CredSet) error {
	cfg, err := ReadAiderConfig()
	if err != nil {
		return fmt.Errorf("aider: load existing config before inject: %w", err)
	}

	changed := false

	// YAML-native keys — supported directly in .aider.conf.yml
	if creds.AnthropicKey != "" {
		cfg["anthropic-api-key"] = creds.AnthropicKey
		changed = true
	}
	if creds.OpenAIKey != "" {
		cfg["openai-api-key"] = creds.OpenAIKey
		changed = true
	}
	if creds.OpenAIBaseURL != "" {
		cfg["openai-api-base"] = creds.OpenAIBaseURL
		changed = true
	}

	if changed {
		if err := WriteAiderConfig(cfg); err != nil {
			return err
		}
	}

	// .env-style keys — providers not natively supported in YAML config
	return injectAiderEnvKeys(creds)
}

// injectAiderEnvKeys writes provider API keys that Aider requires via
// environment variables into ~/.aider.env. Existing lines for unrelated
// variables are preserved; only the specific keys in creds are updated.
func injectAiderEnvKeys(creds CredSet) error {
	// Map of env var name → value to inject. Empty values are skipped.
	toInject := map[string]string{
		"GEMINI_API_KEY":   creds.GeminiKey,
		"DEEPSEEK_API_KEY": creds.DeepSeekKey,
	}

	// Filter out empty entries — don't touch keys the caller didn't set.
	active := make(map[string]string)
	for k, v := range toInject {
		if v != "" {
			active[k] = v
		}
	}
	if len(active) == 0 {
		return nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("aider: resolve home dir for .env: %w", err)
	}
	envPath := filepath.Join(home, aiderEnvFilename)

	// Load existing .env lines.
	existing := make(map[string]string)
	order := []string{}

	if raw, err := os.ReadFile(envPath); err == nil {
		for _, line := range strings.Split(string(raw), "\n") {
			line = strings.TrimRight(line, "\r")
			if line == "" || strings.HasPrefix(line, "#") {
				order = append(order, line)
				continue
			}
			idx := strings.IndexByte(line, '=')
			if idx < 0 {
				order = append(order, line)
				continue
			}
			k := line[:idx]
			v := line[idx+1:]
			// Strip surrounding quotes if present.
			v = strings.Trim(v, `"'`)
			existing[k] = v
			order = append(order, k) // preserve insertion order
		}
	}

	// Merge active keys into existing, tracking which we've seen.
	for k, v := range active {
		existing[k] = v
		// Add to order only if not already present.
		found := false
		for _, o := range order {
			if o == k {
				found = true
				break
			}
		}
		if !found {
			order = append(order, k)
		}
	}

	// Rebuild the file.
	var sb strings.Builder
	for _, key := range order {
		if key == "" || strings.HasPrefix(key, "#") {
			sb.WriteString(key)
			sb.WriteByte('\n')
			continue
		}
		val, ok := existing[key]
		if !ok {
			continue
		}
		sb.WriteString(key)
		sb.WriteByte('=')
		sb.WriteString(val)
		sb.WriteByte('\n')
	}

	if err := os.WriteFile(envPath, []byte(sb.String()), 0o600); err != nil {
		return fmt.Errorf("aider: write .env %s: %w", envPath, err)
	}
	return nil
}

// --- internal helpers -------------------------------------------------------

// findAiderBinary locates the aider executable.
// Search order: PATH → Python user-scripts directories (Windows-specific
// per-user pip install paths).
func findAiderBinary() (string, error) {
	names := aiderBinaryNames()
	for _, name := range names {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}

	// Windows: pip installs user scripts to %APPDATA%\Python\Python3X\Scripts\
	if runtime.GOOS == "windows" {
		for _, candidate := range windowsAiderCandidates() {
			if _, err := os.Stat(candidate); err == nil {
				return candidate, nil
			}
		}
	}

	return "", fmt.Errorf("aider binary not found in PATH or known install locations")
}

// aiderBinaryNames returns the platform-appropriate binary names to probe.
func aiderBinaryNames() []string {
	if runtime.GOOS == "windows" {
		return []string{"aider.exe", "aider"}
	}
	return []string{"aider"}
}

// windowsAiderCandidates returns common per-user pip script paths on Windows.
// These cover Python 3.9 through 3.14 since aider requires Python >= 3.9.
func windowsAiderCandidates() []string {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		home, _ := os.UserHomeDir()
		appData = filepath.Join(home, "AppData", "Roaming")
	}

	var paths []string
	for minor := 9; minor <= 14; minor++ {
		tag := fmt.Sprintf("Python3%d", minor)
		paths = append(paths,
			filepath.Join(appData, "Python", tag, "Scripts", "aider.exe"),
		)
	}

	// Scoop and chocolatey install locations
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData != "" {
		paths = append(paths,
			filepath.Join(localAppData, "Programs", "Python", "Scripts", "aider.exe"),
		)
	}

	return paths
}

// parseAiderVersion extracts the version string from `aider --version` output.
// Aider prints either "aider X.Y.Z" or just "X.Y.Z".
func parseAiderVersion(output string) string {
	// Strip leading "aider " prefix if present.
	s := strings.TrimPrefix(output, "aider ")
	// Take the first space-delimited token in case there's extra detail.
	if idx := strings.IndexAny(s, " \t\n"); idx >= 0 {
		s = s[:idx]
	}
	if s == "" {
		return "unknown"
	}
	return s
}
