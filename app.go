package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"lurus-switch/internal/config"
	"lurus-switch/internal/generator"
	"lurus-switch/internal/packager"
	"lurus-switch/internal/validator"
)

// App struct
type App struct {
	ctx       context.Context
	store     *config.Store
	validator *validator.Validator
}

// NewApp creates a new App application struct
func NewApp() *App {
	store, err := config.NewStore()
	if err != nil {
		// Log error but continue - store will be nil
		fmt.Printf("Warning: failed to initialize config store: %v\n", err)
	}

	return &App{
		store:     store,
		validator: validator.NewValidator(),
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// ============================
// Claude Code Methods
// ============================

// GetDefaultClaudeConfig returns a default Claude configuration
func (a *App) GetDefaultClaudeConfig() *config.ClaudeConfig {
	return config.NewClaudeConfig()
}

// SaveClaudeConfig saves a Claude configuration
func (a *App) SaveClaudeConfig(name string, cfg *config.ClaudeConfig) error {
	if a.store == nil {
		return fmt.Errorf("config store not initialized")
	}
	return a.store.SaveClaudeConfig(name, cfg)
}

// LoadClaudeConfig loads a Claude configuration
func (a *App) LoadClaudeConfig(name string) (*config.ClaudeConfig, error) {
	if a.store == nil {
		return nil, fmt.Errorf("config store not initialized")
	}
	return a.store.LoadClaudeConfig(name)
}

// ListClaudeConfigs lists all saved Claude configurations
func (a *App) ListClaudeConfigs() ([]string, error) {
	if a.store == nil {
		return nil, fmt.Errorf("config store not initialized")
	}
	return a.store.ListConfigs("claude")
}

// DeleteClaudeConfig deletes a Claude configuration
func (a *App) DeleteClaudeConfig(name string) error {
	if a.store == nil {
		return fmt.Errorf("config store not initialized")
	}
	return a.store.DeleteConfig("claude", name)
}

// ValidateClaudeConfig validates a Claude configuration
func (a *App) ValidateClaudeConfig(cfg *config.ClaudeConfig) *validator.ValidationResult {
	return a.validator.ValidateClaudeConfig(cfg)
}

// GenerateClaudeConfig generates Claude configuration files
func (a *App) GenerateClaudeConfig(cfg *config.ClaudeConfig) (string, error) {
	gen := generator.NewClaudeGenerator()
	return gen.GenerateString(cfg)
}

// ExportClaudeConfig exports Claude configuration to a selected directory
func (a *App) ExportClaudeConfig(cfg *config.ClaudeConfig) (string, error) {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Export Directory",
	})
	if err != nil {
		return "", err
	}
	if dir == "" {
		return "", fmt.Errorf("no directory selected")
	}

	gen := generator.NewClaudeGenerator()
	return gen.Generate(cfg, dir)
}

// ============================
// Codex Methods
// ============================

// GetDefaultCodexConfig returns a default Codex configuration
func (a *App) GetDefaultCodexConfig() *config.CodexConfig {
	return config.NewCodexConfig()
}

// SaveCodexConfig saves a Codex configuration
func (a *App) SaveCodexConfig(name string, cfg *config.CodexConfig) error {
	if a.store == nil {
		return fmt.Errorf("config store not initialized")
	}
	return a.store.SaveCodexConfig(name, cfg)
}

// LoadCodexConfig loads a Codex configuration
func (a *App) LoadCodexConfig(name string) (*config.CodexConfig, error) {
	if a.store == nil {
		return nil, fmt.Errorf("config store not initialized")
	}
	return a.store.LoadCodexConfig(name)
}

// ListCodexConfigs lists all saved Codex configurations
func (a *App) ListCodexConfigs() ([]string, error) {
	if a.store == nil {
		return nil, fmt.Errorf("config store not initialized")
	}
	return a.store.ListConfigs("codex")
}

// DeleteCodexConfig deletes a Codex configuration
func (a *App) DeleteCodexConfig(name string) error {
	if a.store == nil {
		return fmt.Errorf("config store not initialized")
	}
	return a.store.DeleteConfig("codex", name)
}

// ValidateCodexConfig validates a Codex configuration
func (a *App) ValidateCodexConfig(cfg *config.CodexConfig) *validator.ValidationResult {
	return a.validator.ValidateCodexConfig(cfg)
}

// GenerateCodexConfig generates Codex configuration files
func (a *App) GenerateCodexConfig(cfg *config.CodexConfig) (string, error) {
	gen := generator.NewCodexGenerator()
	return gen.GenerateString(cfg)
}

// ExportCodexConfig exports Codex configuration to a selected directory
func (a *App) ExportCodexConfig(cfg *config.CodexConfig) (string, error) {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Export Directory",
	})
	if err != nil {
		return "", err
	}
	if dir == "" {
		return "", fmt.Errorf("no directory selected")
	}

	gen := generator.NewCodexGenerator()
	return gen.Generate(cfg, dir)
}

// ============================
// Gemini Methods
// ============================

// GetDefaultGeminiConfig returns a default Gemini configuration
func (a *App) GetDefaultGeminiConfig() *config.GeminiConfig {
	return config.NewGeminiConfig()
}

// SaveGeminiConfig saves a Gemini configuration
func (a *App) SaveGeminiConfig(name string, cfg *config.GeminiConfig) error {
	if a.store == nil {
		return fmt.Errorf("config store not initialized")
	}
	return a.store.SaveGeminiConfig(name, cfg)
}

// LoadGeminiConfig loads a Gemini configuration
func (a *App) LoadGeminiConfig(name string) (*config.GeminiConfig, error) {
	if a.store == nil {
		return nil, fmt.Errorf("config store not initialized")
	}
	return a.store.LoadGeminiConfig(name)
}

// ListGeminiConfigs lists all saved Gemini configurations
func (a *App) ListGeminiConfigs() ([]string, error) {
	if a.store == nil {
		return nil, fmt.Errorf("config store not initialized")
	}
	return a.store.ListConfigs("gemini")
}

// DeleteGeminiConfig deletes a Gemini configuration
func (a *App) DeleteGeminiConfig(name string) error {
	if a.store == nil {
		return fmt.Errorf("config store not initialized")
	}
	return a.store.DeleteConfig("gemini", name)
}

// ValidateGeminiConfig validates a Gemini configuration
func (a *App) ValidateGeminiConfig(cfg *config.GeminiConfig) *validator.ValidationResult {
	return a.validator.ValidateGeminiConfig(cfg)
}

// GenerateGeminiConfig generates Gemini configuration files (Markdown)
func (a *App) GenerateGeminiConfig(cfg *config.GeminiConfig) string {
	gen := generator.NewGeminiGenerator()
	return gen.GenerateMarkdown(cfg)
}

// ExportGeminiConfig exports Gemini configuration to a selected directory
func (a *App) ExportGeminiConfig(cfg *config.GeminiConfig) ([]string, error) {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Export Directory",
	})
	if err != nil {
		return nil, err
	}
	if dir == "" {
		return nil, fmt.Errorf("no directory selected")
	}

	gen := generator.NewGeminiGenerator()
	return gen.GenerateAll(cfg, dir)
}

// ============================
// Packaging Methods
// ============================

// PackageClaudeConfig packages Claude configuration into an executable
func (a *App) PackageClaudeConfig(cfg *config.ClaudeConfig) (string, error) {
	// Select output location
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

	// Create temp directory for config
	tmpDir, err := os.MkdirTemp("", "claude-config-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Generate config files
	gen := generator.NewClaudeGenerator()
	if _, err := gen.Generate(cfg, tmpDir); err != nil {
		return "", fmt.Errorf("failed to generate config: %w", err)
	}

	// Package with Bun
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

// ============================
// Utility Methods
// ============================

// GetConfigDir returns the configuration directory path
func (a *App) GetConfigDir() string {
	if a.store == nil {
		return ""
	}
	return a.store.GetConfigDir()
}

// OpenConfigDir opens the configuration directory in the file explorer
func (a *App) OpenConfigDir() error {
	if a.store == nil {
		return fmt.Errorf("config store not initialized")
	}
	dir := a.store.GetConfigDir()
	return openDirectory(dir)
}

// CheckBunInstalled checks if Bun is installed
func (a *App) CheckBunInstalled() bool {
	return packager.IsBunInstalled()
}

// CheckNodeInstalled checks if Node.js is installed
func (a *App) CheckNodeInstalled() bool {
	return packager.IsNodeInstalled()
}

// openDirectory opens a directory in the system file explorer
func openDirectory(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Platform-specific command to open directory
	var cmd string
	var args []string

	switch goruntime.GOOS {
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

	return exec.Command(cmd, args...).Start()
}
