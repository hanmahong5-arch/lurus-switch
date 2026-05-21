package configsync

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const maxBundleBytes = 64 << 20 // 64 MiB cap — bundles are config, not media

// readBundle slurps a zip from r (bounded) and returns its reader + manifest.
func readBundle(r io.Reader) (*zip.Reader, Manifest, []byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxBundleBytes))
	if err != nil {
		return nil, Manifest{}, nil, fmt.Errorf("read bundle: %w", err)
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, Manifest{}, nil, fmt.Errorf("open bundle (not a valid zip?): %w", err)
	}
	mf, err := readManifest(zr)
	if err != nil {
		return nil, Manifest{}, nil, err
	}
	if mf.SchemaVersion != SchemaVersion {
		return nil, Manifest{}, nil, fmt.Errorf("unsupported bundle schema v%d (this build understands v%d)", mf.SchemaVersion, SchemaVersion)
	}
	return zr, mf, data, nil
}

func readManifest(zr *zip.Reader) (Manifest, error) {
	for _, f := range zr.File {
		if f.Name == manifestEntry {
			rc, err := f.Open()
			if err != nil {
				return Manifest{}, fmt.Errorf("open manifest: %w", err)
			}
			defer rc.Close()
			raw, err := io.ReadAll(io.LimitReader(rc, 1<<20))
			if err != nil {
				return Manifest{}, fmt.Errorf("read manifest: %w", err)
			}
			var mf Manifest
			if err := json.Unmarshal(raw, &mf); err != nil {
				return Manifest{}, fmt.Errorf("parse manifest: %w", err)
			}
			return mf, nil
		}
	}
	return Manifest{}, fmt.Errorf("bundle has no manifest.json — not a Switch config bundle")
}

// Preview inspects a bundle without writing anything. For each component in
// the bundle it reports whether applying would overwrite an existing target
// or create a fresh one.
func Preview(r io.Reader, d Dirs) (*BundlePreview, error) {
	_, mf, _, err := readBundle(r)
	if err != nil {
		return nil, err
	}
	out := &BundlePreview{Manifest: mf}
	inBundle := make(map[string]bool, len(mf.Components))
	for _, c := range mf.Components {
		inBundle[c] = true
	}
	for _, key := range AllComponents {
		cp := ComponentPreview{Key: key, InBundle: inBundle[key]}
		if !inBundle[key] {
			cp.Action = "skip"
		} else if componentExistsOnDisk(key, d) {
			cp.Action = "overwrite"
			cp.Detail = "existing config will be backed up then replaced"
		} else {
			cp.Action = "create"
		}
		out.Components = append(out.Components, cp)
	}
	return out, nil
}

// componentExistsOnDisk reports whether any target file for the component is
// already present (so Apply would overwrite rather than create).
func componentExistsOnDisk(key string, d Dirs) bool {
	for _, it := range componentTargets(key, d) {
		if _, err := os.Stat(it); err == nil {
			return true
		}
	}
	return false
}

// componentTargets lists the absolute destination paths a component touches.
// For directory components it returns the directory itself.
func componentTargets(key string, d Dirs) []string {
	switch key {
	case CompAppSettings:
		return []string{filepath.Join(d.AppData, "app-settings.json")}
	case CompCustomProviders:
		return []string{filepath.Join(d.AppData, "custom-providers.json")}
	case CompSnapshots:
		return []string{filepath.Join(d.AppData, "snapshots")}
	case CompMCPPresets:
		return []string{filepath.Join(d.AppData, "mcp-presets")}
	case CompPrompts:
		return []string{filepath.Join(d.AppData, "prompts")}
	case CompToolConfigs:
		out := make([]string, 0, len(toolConfigSpecs))
		for _, spec := range toolConfigSpecs {
			out = append(out, filepath.Join(d.Home, spec.rel))
		}
		return out
	default:
		return nil
	}
}

// zipPathTarget maps an entry's in-bundle path to its absolute destination,
// and the component it belongs to. Returns ok=false for unknown paths (e.g.
// manifest.json or a path that escapes the known layout).
func zipPathTarget(zipPath string, d Dirs) (component string, dest string, ok bool) {
	clean := filepath.ToSlash(zipPath)
	if strings.Contains(clean, "..") {
		return "", "", false // zip-slip guard
	}
	switch {
	case clean == "app-settings.json":
		return CompAppSettings, filepath.Join(d.AppData, "app-settings.json"), true
	case clean == "custom-providers.json":
		return CompCustomProviders, filepath.Join(d.AppData, "custom-providers.json"), true
	case strings.HasPrefix(clean, "snapshots/"):
		return CompSnapshots, filepath.Join(d.AppData, "snapshots", strings.TrimPrefix(clean, "snapshots/")), true
	case strings.HasPrefix(clean, "mcp-presets/"):
		return CompMCPPresets, filepath.Join(d.AppData, "mcp-presets", strings.TrimPrefix(clean, "mcp-presets/")), true
	case strings.HasPrefix(clean, "prompts/"):
		return CompPrompts, filepath.Join(d.AppData, "prompts", strings.TrimPrefix(clean, "prompts/")), true
	case strings.HasPrefix(clean, "tool-configs/"):
		name := strings.TrimPrefix(clean, "tool-configs/")
		for _, spec := range toolConfigSpecs {
			if spec.zipName == name {
				return CompToolConfigs, filepath.Join(d.Home, spec.rel), true
			}
		}
		return "", "", false
	default:
		return "", "", false
	}
}

// Apply writes the accepted components from the bundle to disk. Before
// overwriting any existing file it is copied to "<file>.before-import-<ts>"
// so the user can roll back. accepted maps component key -> include.
//
// Returns the list of components actually written.
func Apply(r io.Reader, d Dirs, accepted map[string]bool) ([]string, error) {
	zr, mf, _, err := readBundle(r)
	if err != nil {
		return nil, err
	}
	inBundle := make(map[string]bool, len(mf.Components))
	for _, c := range mf.Components {
		inBundle[c] = true
	}

	ts := nowFunc().Format("20060102-150405")
	written := make(map[string]bool)

	for _, f := range zr.File {
		if f.Name == manifestEntry || f.FileInfo().IsDir() {
			continue
		}
		comp, dest, ok := zipPathTarget(f.Name, d)
		if !ok {
			continue
		}
		if !inBundle[comp] || !accepted[comp] {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("open %s: %w", f.Name, err)
		}
		data, err := io.ReadAll(io.LimitReader(rc, maxBundleBytes))
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", f.Name, err)
		}

		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return nil, fmt.Errorf("create dir for %s: %w", dest, err)
		}
		if err := backupIfExists(dest, ts); err != nil {
			return nil, err
		}
		if err := os.WriteFile(dest, data, 0o600); err != nil {
			return nil, fmt.Errorf("write %s: %w", dest, err)
		}
		written[comp] = true
	}

	if err := pruneBackups(d, ts); err != nil {
		// Non-fatal: cleanup failure shouldn't fail the import.
		_ = err
	}

	out := make([]string, 0, len(written))
	for _, key := range AllComponents {
		if written[key] {
			out = append(out, key)
		}
	}
	return out, nil
}

// backupIfExists copies dest to "<dest>.before-import-<ts>" when dest exists.
func backupIfExists(dest, ts string) error {
	data, err := os.ReadFile(dest)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read existing %s for backup: %w", dest, err)
	}
	backup := fmt.Sprintf("%s.before-import-%s", dest, ts)
	if err := os.WriteFile(backup, data, 0o600); err != nil {
		return fmt.Errorf("write backup %s: %w", backup, err)
	}
	return nil
}

const maxBackupsPerFile = 5

// pruneBackups keeps only the most recent maxBackupsPerFile ".before-import-*"
// files per original, so repeated imports don't accumulate forever. Best
// effort — the caller ignores the error.
func pruneBackups(d Dirs, _ string) error {
	dirs := []string{d.AppData, filepath.Join(d.AppData, "snapshots"), filepath.Join(d.AppData, "mcp-presets"), filepath.Join(d.AppData, "prompts")}
	for _, spec := range toolConfigSpecs {
		dirs = append(dirs, filepath.Dir(filepath.Join(d.Home, spec.rel)))
	}
	for _, dir := range dirs {
		pruneBackupsInDir(dir)
	}
	return nil
}

func pruneBackupsInDir(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	// Group backups by their original name (everything before ".before-import-").
	groups := make(map[string][]string)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if i := strings.Index(name, ".before-import-"); i >= 0 {
			orig := name[:i]
			groups[orig] = append(groups[orig], name)
		}
	}
	for _, names := range groups {
		if len(names) <= maxBackupsPerFile {
			continue
		}
		// Timestamp suffix sorts lexicographically == chronologically.
		sortStrings(names)
		for _, old := range names[:len(names)-maxBackupsPerFile] {
			_ = os.Remove(filepath.Join(dir, old))
		}
	}
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}
