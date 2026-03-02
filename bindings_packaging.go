package main

import (
	"fmt"
	"os"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"lurus-switch/internal/config"
	"lurus-switch/internal/generator"
	"lurus-switch/internal/packager"
)

// PackageClaudeConfig packages Claude configuration into an executable
func (a *App) PackageClaudeConfig(cfg *config.ClaudeConfig) (string, error) {
	savePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Save Claude Package",
		DefaultFilename: "claude-custom.exe",
	})
	if err != nil {
		return "", err
	}
	if savePath == "" {
		return "", fmt.Errorf("no save location selected")
	}

	tmpDir, err := os.MkdirTemp("", "claude-config-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	gen := generator.NewClaudeGenerator()
	if _, err := gen.Generate(cfg, tmpDir); err != nil {
		return "", fmt.Errorf("failed to generate config: %w", err)
	}

	pkg, err := packager.NewBunPackager()
	if err != nil {
		return "", fmt.Errorf("Bun packager not available: %w", err)
	}

	if err := pkg.Package(tmpDir, savePath); err != nil {
		return "", fmt.Errorf("failed to package: %w", err)
	}

	return savePath, nil
}

// DownloadCodexBinary downloads the Codex CLI binary
func (a *App) DownloadCodexBinary(version string) (string, error) {
	if version == "" {
		version = "latest"
	}

	pkg, err := packager.NewRustPackager()
	if err != nil {
		return "", err
	}

	return pkg.DownloadCodex(version)
}
