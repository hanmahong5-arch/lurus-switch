package toolmanifest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Local-override manifest support. Reseller operators edit per-tool entries
// in the Switch admin panel; the result is persisted next to the cache file
// and merged into every Fetch() result so client lookups respect operator
// intent immediately, without round-tripping to api.lurus.cn.
//
// The override file's schema mirrors Manifest so the operator can paste it
// back into the hub-side admin endpoint when one becomes available.

const overrideFilename = "tool_manifest_overrides.json"

// OverridesFile is the on-disk shape: a partial Manifest where the Tools
// map contains only the entries the operator has touched.
type OverridesFile struct {
	UpdatedAt string               `json:"updated_at"`
	Tools     map[string]ToolEntry `json:"tools"`
}

var overrideMu sync.Mutex

// LoadOverrides reads the override file from cacheDir. Missing file → empty
// OverridesFile (no error); corrupt file → error so the caller can surface
// it rather than silently dropping operator edits.
func LoadOverrides(cacheDir string) (*OverridesFile, error) {
	path := filepath.Join(cacheDir, overrideFilename)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &OverridesFile{Tools: map[string]ToolEntry{}}, nil
		}
		return nil, fmt.Errorf("read overrides: %w", err)
	}
	var f OverridesFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("corrupt overrides file: %w", err)
	}
	if f.Tools == nil {
		f.Tools = map[string]ToolEntry{}
	}
	return &f, nil
}

// SaveOverrides serialises the file to disk atomically (write to .tmp + rename).
func SaveOverrides(cacheDir string, f *OverridesFile) error {
	overrideMu.Lock()
	defer overrideMu.Unlock()
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return err
	}
	if f.Tools == nil {
		f.Tools = map[string]ToolEntry{}
	}
	f.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(cacheDir, overrideFilename)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// SetOverride writes a single tool's override entry. Pass entry={} (zero
// value) with all fields empty to effectively "clear" the override, though
// DeleteOverride is the cleaner way to revert.
func SetOverride(cacheDir, name string, entry ToolEntry) error {
	f, err := LoadOverrides(cacheDir)
	if err != nil {
		return err
	}
	f.Tools[name] = entry
	return SaveOverrides(cacheDir, f)
}

// DeleteOverride removes the operator's override for one tool, reverting to
// whatever the upstream/built-in manifest says.
func DeleteOverride(cacheDir, name string) error {
	f, err := LoadOverrides(cacheDir)
	if err != nil {
		return err
	}
	delete(f.Tools, name)
	return SaveOverrides(cacheDir, f)
}

// ResetOverrides clears every operator-set entry.
func ResetOverrides(cacheDir string) error {
	return SaveOverrides(cacheDir, &OverridesFile{Tools: map[string]ToolEntry{}})
}

// Merge applies the overrides on top of base, returning a new manifest.
// Per-tool override entries fully replace the upstream entry (we don't
// shallow-merge fields — the operator's intent for a tool is explicit).
func Merge(base *Manifest, overrides *OverridesFile) *Manifest {
	out := Manifest{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Tools:       map[string]ToolEntry{},
	}
	if base != nil {
		for k, v := range base.Tools {
			out.Tools[k] = v
		}
		if base.GeneratedAt != "" {
			out.GeneratedAt = base.GeneratedAt
		}
	}
	if overrides != nil {
		for k, v := range overrides.Tools {
			out.Tools[k] = v
		}
	}
	return &out
}

// SortedToolNames returns the manifest's tool keys in deterministic order
// — useful for the admin UI so rows don't reshuffle on each save.
func SortedToolNames(m *Manifest) []string {
	if m == nil {
		return nil
	}
	names := make([]string, 0, len(m.Tools))
	for k := range m.Tools {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}
