package main

import (
	"fmt"
	"os"

	"lurus-switch/internal/capability"
	"lurus-switch/internal/configsync"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	auditOpConfigExport = "config.export"
	auditOpConfigImport = "config.import"
)

// syncDirs resolves the filesystem roots the bundle reads/writes. Tool
// configs live under the user home; everything else under app data.
func syncDirs() (configsync.Dirs, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return configsync.Dirs{}, fmt.Errorf("resolve home dir: %w", err)
	}
	return configsync.Dirs{AppData: appDataBaseDir(), Home: home}, nil
}

// PickExportBundlePath opens a native "save as" dialog for the export zip.
// Returns "" (no error) when the user cancels.
func (a *App) PickExportBundlePath() (string, error) {
	return wailsRuntime.SaveFileDialog(a.ctx, wailsRuntime.SaveDialogOptions{
		Title:           "Export Switch configuration",
		DefaultFilename: "switch-config.zip",
		Filters:         []wailsRuntime.FileFilter{{DisplayName: "Zip bundle (*.zip)", Pattern: "*.zip"}},
	})
}

// PickImportBundlePath opens a native open dialog for an existing bundle.
// Returns "" (no error) when the user cancels.
func (a *App) PickImportBundlePath() (string, error) {
	return wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title:   "Import Switch configuration",
		Filters: []wailsRuntime.FileFilter{{DisplayName: "Zip bundle (*.zip)", Pattern: "*.zip"}},
	})
}

// ExportConfigBundle writes a portable configuration bundle (zip) to
// targetPath. includeKeys=false (default) redacts API keys. Gated by the
// all-capability since it can read every secret; journaled for the trail.
func (a *App) ExportConfigBundle(targetPath string, includeKeys bool) (mf configsync.Manifest, err error) {
	if err = a.requireAndAudit(capability.CapAll, auditOpConfigExport, targetPath, map[string]any{"includeKeys": includeKeys}); err != nil {
		return configsync.Manifest{}, err
	}
	defer func() {
		a.recordOutcome(auditOpConfigExport, targetPath, map[string]any{"includeKeys": includeKeys}, err)
	}()
	dirs, derr := syncDirs()
	if derr != nil {
		return configsync.Manifest{}, derr
	}
	return configsync.ExportToFile(dirs, includeKeys, AppVersion, targetPath)
}

// PreviewImportBundle inspects a bundle without writing anything, returning
// per-component overwrite/create/skip actions for the confirm dialog.
// Read-only — no capability gate.
func (a *App) PreviewImportBundle(sourcePath string) (*configsync.BundlePreview, error) {
	f, err := os.Open(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("open bundle: %w", err)
	}
	defer f.Close()
	dirs, derr := syncDirs()
	if derr != nil {
		return nil, derr
	}
	return configsync.Preview(f, dirs)
}

// ApplyImportBundle writes the accepted components from a bundle to disk,
// backing up any file it overwrites. Gated by the all-capability (it can
// replace tool credentials and every local setting); journaled.
func (a *App) ApplyImportBundle(sourcePath string, accepted map[string]bool) (written []string, err error) {
	if err = a.requireAndAudit(capability.CapAll, auditOpConfigImport, sourcePath, accepted); err != nil {
		return nil, err
	}
	defer func() {
		a.recordOutcome(auditOpConfigImport, sourcePath, map[string]any{"written": written}, err)
	}()
	f, oerr := os.Open(sourcePath)
	if oerr != nil {
		return nil, fmt.Errorf("open bundle: %w", oerr)
	}
	defer f.Close()
	dirs, derr := syncDirs()
	if derr != nil {
		return nil, derr
	}
	return configsync.Apply(f, dirs, accepted)
}
