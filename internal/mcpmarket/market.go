// Package mcpmarket provides access to the MCP server registry and handles
// cross-tool MCP configuration installation.
//
// Key capabilities:
//   - List servers from the embedded builtin manifest (always available)
//   - Fetch and cache server listings from the public registry
//   - Install an MCP server into one or more CLI tool configs simultaneously
//   - Save an installed server as a reusable preset via internal/mcp
package mcpmarket

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	_ "embed"

	"lurus-switch/internal/mcp"
)

//go:embed builtin_servers.json
var builtinServersJSON []byte

const (
	// registryBase is the public registry REST API.
	registryBase = "https://registry.smithery.ai"

	// registryTimeout is the per-request deadline for registry calls.
	registryTimeout = 15 * time.Second

	// registryDefaultLimit is the default page size for ListFromRegistry.
	registryDefaultLimit = 40
)

// Market provides server discovery and cross-tool installation.
type Market struct {
	// httpClient is used for all registry requests.  Callers may replace it
	// in tests to avoid network calls.  When nil, http.DefaultClient is used
	// so that BYO proxy configured via internal/netproxy is picked up
	// automatically (netproxy mutates http.DefaultTransport).
	httpClient *http.Client
}

// NewMarket creates a Market with the default HTTP client.
func NewMarket() *Market {
	return &Market{
		httpClient: &http.Client{Timeout: registryTimeout},
	}
}

// loadBuiltin parses the embedded builtin_servers.json.
func (m *Market) loadBuiltin() ([]MarketServer, error) {
	var servers []MarketServer
	if err := json.Unmarshal(builtinServersJSON, &servers); err != nil {
		return nil, fmt.Errorf("mcpmarket: parse builtin manifest: %w", err)
	}
	return servers, nil
}

// ListServers returns servers from the local cache merged with builtins.
// Registry entries override builtins with the same QualifiedName.
// The call never blocks on a network request; use RefreshFromRegistry to
// populate the cache.
func (m *Market) ListServers() ([]MarketServer, error) {
	builtin, err := m.loadBuiltin()
	if err != nil {
		return nil, err
	}

	cached, _, _, _ := loadCache()

	// Merge: registry entries override builtins by QualifiedName when non-empty.
	merged := make(map[string]MarketServer, len(builtin)+len(cached))
	for _, s := range builtin {
		key := s.QualifiedName
		if key == "" {
			key = s.ID
		}
		merged[key] = s
	}
	for _, s := range cached {
		key := s.QualifiedName
		if key == "" {
			key = s.ID
		}
		merged[key] = s
	}

	out := make([]MarketServer, 0, len(merged))
	for _, s := range merged {
		out = append(out, s)
	}
	return out, nil
}

// GetServer looks up a server by ID or QualifiedName.
// It searches the merged list; registry entries take precedence over builtins.
func (m *Market) GetServer(ctx context.Context, id string) (*MarketServer, error) {
	servers, err := m.ListServers()
	if err != nil {
		return nil, err
	}
	for i := range servers {
		if servers[i].ID == id || servers[i].QualifiedName == id {
			return &servers[i], nil
		}
	}
	return nil, fmt.Errorf("mcpmarket: server not found: %s", id)
}

// RefreshFromRegistry fetches the first page of servers (up to registryDefaultLimit)
// from the public registry, merges with the existing cache, and persists the result.
// Network errors are silently swallowed so builtins remain usable offline.
// The query string filters by display name or description; pass "" for all.
func (m *Market) RefreshFromRegistry(ctx context.Context, query string) error {
	// Skip refresh if cache is fresh.
	_, fetchedAt, _, _ := loadCache()
	if !fetchedAt.IsZero() && time.Since(fetchedAt) < cacheMaxAge {
		return nil
	}

	url := fmt.Sprintf("%s/servers?pageSize=%d", registryBase, registryDefaultLimit)
	if query != "" {
		url += "&q=" + urlQueryEscape(query)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("mcpmarket: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	client := m.httpClient
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Do(req)
	if err != nil {
		// Network unreachable — not fatal.
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("mcpmarket: read registry response: %w", err)
	}

	var apiResp registryListResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return fmt.Errorf("mcpmarket: parse registry response: %w", err)
	}

	etag := resp.Header.Get("ETag")
	servers := convertRegistryServers(apiResp.Servers)
	return saveCache(servers, etag)
}

// convertRegistryServers maps registry API server objects to MarketServer.
func convertRegistryServers(raw []registryServer) []MarketServer {
	out := make([]MarketServer, 0, len(raw))
	for _, r := range raw {
		ms := MarketServer{
			ID:            r.ID,
			QualifiedName: r.QualifiedName,
			Name:          r.DisplayName,
			Description:   r.Description,
			Stars:         r.UseCount,
			Verified:      r.Verified,
			Homepage:      r.Homepage,
			Builtin:       false,
			FetchedAt:     time.Now(),
		}
		if r.QualifiedName != "" {
			ms.InstallCommand = fmt.Sprintf("npx -y %s", r.QualifiedName)
		}
		if r.Namespace != "" {
			ms.Author = r.Namespace
		}
		out = append(out, ms)
	}
	return out
}

// InstallToTools writes the MCP server entry into each of the requested target
// tool config files.  userConfig provides the user-supplied values for keys
// defined in server.ConfigSchema (env vars, connection strings, etc.).
// Each tool is attempted independently; a failure for one does not abort the others.
func (m *Market) InstallToTools(
	ctx context.Context,
	server MarketServer,
	userConfig map[string]string,
	targetTools []TargetTool,
) (*InstallReport, error) {
	if len(targetTools) == 0 {
		return nil, fmt.Errorf("mcpmarket.InstallToTools: at least one target tool required")
	}

	report := &InstallReport{Statuses: make([]ToolInstallStatus, 0, len(targetTools))}

	// Build the MCPServer value from the market entry and user-supplied config.
	mcpServer := buildMCPServer(server, userConfig)

	for _, tool := range targetTools {
		configPath, writeErr := installToSingleTool(tool, mcpServer)
		status := ToolInstallStatus{Tool: tool}
		if writeErr != nil {
			status.OK = false
			status.Error = writeErr.Error()
		} else {
			status.OK = true
			status.Path = configPath
		}
		report.Statuses = append(report.Statuses, status)
	}
	return report, nil
}

// buildMCPServer constructs an mcp.MCPServer from a market entry and user config.
func buildMCPServer(server MarketServer, userConfig map[string]string) mcp.MCPServer {
	ms := mcp.MCPServer{
		Name:    serverSlug(server),
		Type:    "stdio",
		Command: "npx",
		Args:    []string{"-y", server.QualifiedName},
	}
	if ms.Args[1] == "" {
		// Builtin fallback: use ID as the npm package name convention.
		ms.Args[1] = server.ID
	}
	if len(userConfig) > 0 {
		ms.Env = make(map[string]string, len(userConfig))
		for k, v := range userConfig {
			ms.Env[k] = v
		}
	}
	return ms
}

// serverSlug returns a filesystem-safe slug for use as the MCP server name.
func serverSlug(s MarketServer) string {
	if s.QualifiedName != "" {
		// Strip leading @ and replace / with - for a clean key.
		slug := strings.TrimPrefix(s.QualifiedName, "@")
		slug = strings.ReplaceAll(slug, "/", "-")
		return slug
	}
	return s.ID
}

// installToSingleTool writes the MCP server entry into the given tool's
// settings file and returns the path that was written.
func installToSingleTool(tool TargetTool, server mcp.MCPServer) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("mcpmarket: get home dir: %w", err)
	}

	settingsPath, err := toolMCPConfigPath(tool, home)
	if err != nil {
		return "", err
	}

	if err := patchMCPConfig(settingsPath, server); err != nil {
		return "", err
	}
	return settingsPath, nil
}

// toolMCPConfigPath returns the path of the JSON settings file where an MCP
// server block should be written for the given tool.
//
// Claude Code: ~/.claude/settings.json  (mcpServers top-level key)
// Cursor:      ~/.cursor/mcp.json       (mcpServers top-level key)
// Gemini CLI:  ~/.gemini/settings.json  (mcpServers top-level key)
// Antigravity: platform-specific config dir / config.json
func toolMCPConfigPath(tool TargetTool, home string) (string, error) {
	switch tool {
	case ToolClaudeCode:
		return filepath.Join(home, ".claude", "settings.json"), nil
	case ToolCursor:
		return filepath.Join(home, ".cursor", "mcp.json"), nil
	case ToolGemini:
		return filepath.Join(home, ".gemini", "settings.json"), nil
	case ToolAntigravity:
		// Antigravity uses %LOCALAPPDATA%\Antigravity\config.json on Windows.
		// Fall back to home-relative path on other platforms.
		switch {
		case isWindows():
			localAppData := os.Getenv("LOCALAPPDATA")
			if localAppData == "" {
				localAppData = filepath.Join(home, "AppData", "Local")
			}
			return filepath.Join(localAppData, "Antigravity", "config.json"), nil
		default:
			return filepath.Join(home, ".config", "antigravity", "config.json"), nil
		}
	default:
		return "", fmt.Errorf("mcpmarket: unsupported target tool: %s", tool)
	}
}

// patchMCPConfig reads the existing JSON config file (or creates it), then
// upserts the MCP server under the "mcpServers" top-level key.
func patchMCPConfig(path string, server mcp.MCPServer) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("mcpmarket: read %s: %w", path, err)
		}
		data = []byte("{}")
	}

	var root map[string]any
	if jsonErr := json.Unmarshal(data, &root); jsonErr != nil {
		root = make(map[string]any)
	}

	// Ensure mcpServers map exists.
	mcpServers, _ := root["mcpServers"].(map[string]any)
	if mcpServers == nil {
		mcpServers = make(map[string]any)
	}

	// Build the server entry.
	entry := map[string]any{
		"command": server.Command,
		"args":    server.Args,
		"type":    server.Type,
	}
	if len(server.Env) > 0 {
		entry["env"] = server.Env
	}
	mcpServers[server.Name] = entry
	root["mcpServers"] = mcpServers

	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return fmt.Errorf("mcpmarket: marshal %s: %w", path, err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("mcpmarket: mkdir %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, out, 0644); err != nil {
		return fmt.Errorf("mcpmarket: write %s: %w", path, err)
	}
	return nil
}

// SaveAsPreset saves a market server plus user config as a reusable mcp.MCPPreset
// using the provided mcp.Store.
func (m *Market) SaveAsPreset(store *mcp.Store, server MarketServer, userConfig map[string]string) (*mcp.MCPPreset, error) {
	if store == nil {
		return nil, fmt.Errorf("mcpmarket.SaveAsPreset: store must not be nil")
	}
	mcpServer := buildMCPServer(server, userConfig)
	preset := mcp.MCPPreset{
		Name:        server.Name,
		Description: server.Description,
		Server:      mcpServer,
		Tags:        []string{server.Category, "market"},
	}
	if err := store.SavePreset(preset); err != nil {
		return nil, fmt.Errorf("mcpmarket.SaveAsPreset: %w", err)
	}
	return &preset, nil
}

// urlQueryEscape percent-encodes a string for use as a URL query value.
// This avoids importing net/url in a way that could conflict with stdlib usage.
func urlQueryEscape(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z', r >= '0' && r <= '9',
			r == '-', r == '_', r == '.', r == '~':
			b.WriteRune(r)
		case r == ' ':
			b.WriteByte('+')
		default:
			encoded := fmt.Sprintf("%%%02X", r)
			b.WriteString(encoded)
		}
	}
	return b.String()
}
