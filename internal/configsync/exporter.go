package configsync

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// item is one file in the bundle. zipPath is its path inside the archive;
// srcPath is its absolute location on disk. redact marks files whose JSON/
// TOML secrets are blanked when includeKeys is false.
type item struct {
	zipPath string
	srcPath string
	redact  bool
	isTOML  bool
}

// componentItems resolves the on-disk files for a component, given roots.
// Directory components (snapshots/prompts/mcp-presets) are expanded into
// their member files. Missing files yield an empty slice — exporting a
// component the user never used is simply a no-op for that component.
func componentItems(key string, d Dirs) []item {
	switch key {
	case CompAppSettings:
		return []item{{
			zipPath: "app-settings.json",
			srcPath: filepath.Join(d.AppData, "app-settings.json"),
			redact:  true,
		}}
	case CompCustomProviders:
		return []item{{
			zipPath: "custom-providers.json",
			srcPath: filepath.Join(d.AppData, "custom-providers.json"),
			redact:  true,
		}}
	case CompSnapshots:
		return dirItems("snapshots", filepath.Join(d.AppData, "snapshots"), false, false)
	case CompMCPPresets:
		return dirItems("mcp-presets", filepath.Join(d.AppData, "mcp-presets"), false, false)
	case CompPrompts:
		return dirItems("prompts", filepath.Join(d.AppData, "prompts"), false, false)
	case CompToolConfigs:
		return toolConfigItems(d.Home)
	default:
		return nil
	}
}

// toolConfigSpecs maps each supported CLI to its on-disk config and the zip
// path it lands under. JSON configs are redactable per-key; codex is TOML.
var toolConfigSpecs = []struct {
	rel     string // relative to home
	zipName string
	isTOML  bool
}{
	{filepath.Join(".claude", "settings.json"), "claude-settings.json", false},
	{filepath.Join(".codex", "config.toml"), "codex-config.toml", true},
	{filepath.Join(".gemini", "settings.json"), "gemini-settings.json", false},
	{filepath.Join(".picoclaw", "config.json"), "picoclaw-config.json", false},
	{filepath.Join(".nullclaw", "config.json"), "nullclaw-config.json", false},
}

func toolConfigItems(home string) []item {
	out := make([]item, 0, len(toolConfigSpecs))
	for _, spec := range toolConfigSpecs {
		out = append(out, item{
			zipPath: "tool-configs/" + spec.zipName,
			srcPath: filepath.Join(home, spec.rel),
			redact:  true,
			isTOML:  spec.isTOML,
		})
	}
	return out
}

func dirItems(zipPrefix, srcDir string, redact, isTOML bool) []item {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return nil
	}
	var out []item
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		out = append(out, item{
			zipPath: zipPrefix + "/" + e.Name(),
			srcPath: filepath.Join(srcDir, e.Name()),
			redact:  redact,
			isTOML:  isTOML,
		})
	}
	return out
}

// Export writes a configuration bundle to w. Only components that actually
// have files on disk are included; the manifest lists exactly those. When
// includeKeys is false, JSON/TOML secrets are redacted in-flight.
func Export(d Dirs, includeKeys bool, appVersion string, w io.Writer) (Manifest, error) {
	zw := zip.NewWriter(w)
	present := make([]string, 0, len(AllComponents))

	for _, key := range AllComponents {
		items := componentItems(key, d)
		wroteAny := false
		for _, it := range items {
			data, err := os.ReadFile(it.srcPath)
			if err != nil {
				continue // missing file — skip silently
			}
			if !includeKeys && it.redact {
				if it.isTOML {
					data = redactTOMLLines(data)
				} else {
					data = redactJSON(data)
				}
			}
			fw, err := zw.Create(it.zipPath)
			if err != nil {
				_ = zw.Close()
				return Manifest{}, fmt.Errorf("zip create %s: %w", it.zipPath, err)
			}
			if _, err := fw.Write(data); err != nil {
				_ = zw.Close()
				return Manifest{}, fmt.Errorf("zip write %s: %w", it.zipPath, err)
			}
			wroteAny = true
		}
		if wroteAny {
			present = append(present, key)
		}
	}

	mf := Manifest{
		SchemaVersion: SchemaVersion,
		ExportedAt:    nowFunc(),
		AppVersion:    appVersion,
		IncludesKeys:  includeKeys,
		Components:    present,
	}
	mfData, err := json.MarshalIndent(mf, "", "  ")
	if err != nil {
		_ = zw.Close()
		return Manifest{}, fmt.Errorf("marshal manifest: %w", err)
	}
	fw, err := zw.Create(manifestEntry)
	if err != nil {
		_ = zw.Close()
		return Manifest{}, fmt.Errorf("zip create manifest: %w", err)
	}
	if _, err := fw.Write(mfData); err != nil {
		_ = zw.Close()
		return Manifest{}, fmt.Errorf("write manifest: %w", err)
	}

	if err := zw.Close(); err != nil {
		return Manifest{}, fmt.Errorf("close zip: %w", err)
	}
	return mf, nil
}

// ExportToFile is a convenience wrapper that writes the bundle to a path.
func ExportToFile(d Dirs, includeKeys bool, appVersion, targetPath string) (Manifest, error) {
	if !strings.HasSuffix(strings.ToLower(targetPath), ".zip") {
		targetPath += ".zip"
	}
	f, err := os.Create(targetPath)
	if err != nil {
		return Manifest{}, fmt.Errorf("create bundle file: %w", err)
	}
	defer f.Close()
	return Export(d, includeKeys, appVersion, f)
}
