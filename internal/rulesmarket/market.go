package rulesmarket

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
)

//go:embed builtin_manifest.json
var builtinManifestJSON []byte

// cacheMaxAge is how long the remote template cache is considered fresh.
const cacheMaxAge = 24 * time.Hour

// Market provides access to rule templates from the builtin list and optional
// remote manifests.
type Market struct {
	// httpClient is used for remote fetches.  Replace in tests.
	httpClient *http.Client
}

// NewMarket creates a new Market with default settings.
func NewMarket() *Market {
	return &Market{
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// loadBuiltin parses the embedded manifest and returns the builtin templates.
func (m *Market) loadBuiltin() ([]RuleTemplate, error) {
	var templates []RuleTemplate
	if err := json.Unmarshal(builtinManifestJSON, &templates); err != nil {
		return nil, fmt.Errorf("rulesmarket: parse builtin manifest: %w", err)
	}
	return templates, nil
}

// ListTemplates returns the merged set of builtin and (if available) cached
// remote templates.  It never blocks on a network call; use RefreshFromRemote
// to update the cache.
func (m *Market) ListTemplates() ([]RuleTemplate, error) {
	builtin, err := m.loadBuiltin()
	if err != nil {
		return nil, err
	}

	cached, _, _ := loadCache()

	// Merge: remote templates override builtin by ID.
	merged := make(map[string]RuleTemplate, len(builtin)+len(cached))
	for _, t := range builtin {
		merged[t.ID] = t
	}
	for _, t := range cached {
		merged[t.ID] = t
	}

	out := make([]RuleTemplate, 0, len(merged))
	for _, t := range merged {
		out = append(out, t)
	}
	return out, nil
}

// RefreshFromRemote fetches templates from the provided manifest URL and
// merges them into the local cache.  If url is empty the call is a no-op.
// The function uses ETag to skip re-downloading unchanged content.
func (m *Market) RefreshFromRemote(ctx context.Context, url string) error {
	if url == "" {
		return nil
	}

	// Check whether cache is fresh enough to skip the fetch.
	cached, fetchedAt, _ := loadCache()
	if !fetchedAt.IsZero() && time.Since(fetchedAt) < cacheMaxAge && len(cached) > 0 {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("rulesmarket: build request: %w", err)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		// Network unavailable — not fatal; builtins still work.
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil // Non-200 treated as unavailable, not an error
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("rulesmarket: read response: %w", err)
	}

	var remote []RuleTemplate
	if err := json.Unmarshal(body, &remote); err != nil {
		return fmt.Errorf("rulesmarket: parse remote manifest: %w", err)
	}

	return saveCache(remote)
}

// WriteRuleToProject writes the given template (converted to targetFormat) into
// projectDir.  If the target file already exists, the content is appended with
// a section separator rather than overwriting.  Set overwrite=true to replace
// the file entirely.
func (m *Market) WriteRuleToProject(
	ctx context.Context,
	projectDir string,
	template RuleTemplate,
	targetFormat Format,
	overwrite bool,
) (*WriteResult, error) {
	if projectDir == "" {
		return nil, fmt.Errorf("rulesmarket.WriteRuleToProject: projectDir must not be empty")
	}
	if template.Content == "" {
		return nil, fmt.Errorf("rulesmarket.WriteRuleToProject: template content must not be empty")
	}

	converted, err := Convert(template.Content, template.Format, targetFormat)
	if err != nil {
		return nil, fmt.Errorf("rulesmarket.WriteRuleToProject: convert: %w", err)
	}

	fileName := TargetFileName(targetFormat)
	targetPath := filepath.Join(projectDir, fileName)

	existing, readErr := os.ReadFile(targetPath)
	if readErr != nil && !os.IsNotExist(readErr) {
		return nil, fmt.Errorf("rulesmarket.WriteRuleToProject: read existing file: %w", readErr)
	}

	fileExists := readErr == nil

	if fileExists && !overwrite {
		// Append with a clear section separator
		if strings.Contains(string(existing), converted) {
			// Already present verbatim — skip to stay idempotent
			return &WriteResult{Path: targetPath, Skipped: true}, nil
		}
		separator := "\n\n---\n<!-- rules-market: " + template.ID + " -->\n\n"
		newContent := string(existing) + separator + converted + "\n"
		if err := os.WriteFile(targetPath, []byte(newContent), 0644); err != nil {
			return nil, fmt.Errorf("rulesmarket.WriteRuleToProject: append: %w", err)
		}
		return &WriteResult{Path: targetPath, Appended: true}, nil
	}

	// Overwrite or new file
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return nil, fmt.Errorf("rulesmarket.WriteRuleToProject: create dir: %w", err)
	}
	if err := os.WriteFile(targetPath, []byte(converted+"\n"), 0644); err != nil {
		return nil, fmt.Errorf("rulesmarket.WriteRuleToProject: write: %w", err)
	}
	return &WriteResult{Path: targetPath}, nil
}
