package main

import (
	"fmt"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"lurus-switch/internal/config"
	"lurus-switch/internal/generator"
	"lurus-switch/internal/validator"
)

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
// PicoClaw Methods
// ============================

// GetDefaultPicoClawConfig returns a default PicoClaw configuration
func (a *App) GetDefaultPicoClawConfig() *config.PicoClawConfig {
	return config.NewPicoClawConfig()
}

// SavePicoClawConfig saves a PicoClaw configuration
func (a *App) SavePicoClawConfig(name string, cfg *config.PicoClawConfig) error {
	if a.store == nil {
		return fmt.Errorf("config store not initialized")
	}
	return a.store.SavePicoClawConfig(name, cfg)
}

// LoadPicoClawConfig loads a PicoClaw configuration
func (a *App) LoadPicoClawConfig(name string) (*config.PicoClawConfig, error) {
	if a.store == nil {
		return nil, fmt.Errorf("config store not initialized")
	}
	return a.store.LoadPicoClawConfig(name)
}

// ListPicoClawConfigs lists all saved PicoClaw configurations
func (a *App) ListPicoClawConfigs() ([]string, error) {
	if a.store == nil {
		return nil, fmt.Errorf("config store not initialized")
	}
	return a.store.ListConfigs("picoclaw")
}

// DeletePicoClawConfig deletes a PicoClaw configuration
func (a *App) DeletePicoClawConfig(name string) error {
	if a.store == nil {
		return fmt.Errorf("config store not initialized")
	}
	return a.store.DeleteConfig("picoclaw", name)
}

// ValidatePicoClawConfig validates a PicoClaw configuration
func (a *App) ValidatePicoClawConfig(cfg *config.PicoClawConfig) *validator.ValidationResult {
	return a.validator.ValidatePicoClawConfig(cfg)
}

// GeneratePicoClawConfig generates PicoClaw configuration as a JSON string
func (a *App) GeneratePicoClawConfig(cfg *config.PicoClawConfig) (string, error) {
	gen := generator.NewPicoClawGenerator()
	return gen.GenerateString(cfg)
}

// ExportPicoClawConfig exports PicoClaw configuration to a selected directory
func (a *App) ExportPicoClawConfig(cfg *config.PicoClawConfig) (string, error) {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Export Directory",
	})
	if err != nil {
		return "", err
	}
	if dir == "" {
		return "", fmt.Errorf("no directory selected")
	}

	gen := generator.NewPicoClawGenerator()
	return gen.Generate(cfg, dir)
}

// ============================
// NullClaw Methods
// ============================

// GetDefaultNullClawConfig returns a default NullClaw configuration
func (a *App) GetDefaultNullClawConfig() *config.NullClawConfig {
	return config.NewNullClawConfig()
}

// SaveNullClawConfig saves a NullClaw configuration
func (a *App) SaveNullClawConfig(name string, cfg *config.NullClawConfig) error {
	if a.store == nil {
		return fmt.Errorf("config store not initialized")
	}
	return a.store.SaveNullClawConfig(name, cfg)
}

// LoadNullClawConfig loads a NullClaw configuration
func (a *App) LoadNullClawConfig(name string) (*config.NullClawConfig, error) {
	if a.store == nil {
		return nil, fmt.Errorf("config store not initialized")
	}
	return a.store.LoadNullClawConfig(name)
}

// ListNullClawConfigs lists all saved NullClaw configurations
func (a *App) ListNullClawConfigs() ([]string, error) {
	if a.store == nil {
		return nil, fmt.Errorf("config store not initialized")
	}
	return a.store.ListConfigs("nullclaw")
}

// DeleteNullClawConfig deletes a NullClaw configuration
func (a *App) DeleteNullClawConfig(name string) error {
	if a.store == nil {
		return fmt.Errorf("config store not initialized")
	}
	return a.store.DeleteConfig("nullclaw", name)
}

// ValidateNullClawConfig validates a NullClaw configuration
func (a *App) ValidateNullClawConfig(cfg *config.NullClawConfig) *validator.ValidationResult {
	return a.validator.ValidateNullClawConfig(cfg)
}

// GenerateNullClawConfig generates NullClaw configuration as a JSON string
func (a *App) GenerateNullClawConfig(cfg *config.NullClawConfig) (string, error) {
	gen := generator.NewNullClawGenerator()
	return gen.GenerateString(cfg)
}

// ExportNullClawConfig exports NullClaw configuration to a selected directory
func (a *App) ExportNullClawConfig(cfg *config.NullClawConfig) (string, error) {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Export Directory",
	})
	if err != nil {
		return "", err
	}
	if dir == "" {
		return "", fmt.Errorf("no directory selected")
	}

	gen := generator.NewNullClawGenerator()
	return gen.Generate(cfg, dir)
}

// ============================
// ZeroClaw Methods
// ============================

// GetDefaultZeroClawConfig returns a default ZeroClaw configuration
func (a *App) GetDefaultZeroClawConfig() *config.ZeroClawConfig {
	return config.NewZeroClawConfig()
}

// SaveZeroClawConfig saves a ZeroClaw configuration
func (a *App) SaveZeroClawConfig(name string, cfg *config.ZeroClawConfig) error {
	if a.store == nil {
		return fmt.Errorf("config store not initialized")
	}
	return a.store.SaveZeroClawConfig(name, cfg)
}

// LoadZeroClawConfig loads a ZeroClaw configuration
func (a *App) LoadZeroClawConfig(name string) (*config.ZeroClawConfig, error) {
	if a.store == nil {
		return nil, fmt.Errorf("config store not initialized")
	}
	return a.store.LoadZeroClawConfig(name)
}

// ListZeroClawConfigs lists all saved ZeroClaw configurations
func (a *App) ListZeroClawConfigs() ([]string, error) {
	if a.store == nil {
		return nil, fmt.Errorf("config store not initialized")
	}
	return a.store.ListZeroClawConfigs()
}

// DeleteZeroClawConfig deletes a ZeroClaw configuration
func (a *App) DeleteZeroClawConfig(name string) error {
	if a.store == nil {
		return fmt.Errorf("config store not initialized")
	}
	return a.store.DeleteZeroClawConfig(name)
}

// ValidateZeroClawConfig validates a ZeroClaw configuration
func (a *App) ValidateZeroClawConfig(cfg *config.ZeroClawConfig) *validator.ValidationResult {
	return a.validator.ValidateZeroClawConfig(cfg)
}

// GenerateZeroClawConfig generates ZeroClaw configuration as a TOML string
func (a *App) GenerateZeroClawConfig(cfg *config.ZeroClawConfig) (string, error) {
	gen := generator.NewZeroClawGenerator()
	return gen.GenerateString(cfg)
}

// ExportZeroClawConfig exports ZeroClaw configuration to a selected directory
func (a *App) ExportZeroClawConfig(cfg *config.ZeroClawConfig) (string, error) {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Export Directory",
	})
	if err != nil {
		return "", err
	}
	if dir == "" {
		return "", fmt.Errorf("no directory selected")
	}

	gen := generator.NewZeroClawGenerator()
	return gen.Generate(cfg, dir)
}

// ============================
// OpenClaw Methods
// ============================

// GetDefaultOpenClawConfig returns a default OpenClaw configuration
func (a *App) GetDefaultOpenClawConfig() *config.OpenClawConfig {
	return config.NewOpenClawConfig()
}

// SaveOpenClawConfig saves an OpenClaw configuration
func (a *App) SaveOpenClawConfig(name string, cfg *config.OpenClawConfig) error {
	if a.store == nil {
		return fmt.Errorf("config store not initialized")
	}
	return a.store.SaveOpenClawConfig(name, cfg)
}

// LoadOpenClawConfig loads an OpenClaw configuration
func (a *App) LoadOpenClawConfig(name string) (*config.OpenClawConfig, error) {
	if a.store == nil {
		return nil, fmt.Errorf("config store not initialized")
	}
	return a.store.LoadOpenClawConfig(name)
}

// ListOpenClawConfigs lists all saved OpenClaw configurations
func (a *App) ListOpenClawConfigs() ([]string, error) {
	if a.store == nil {
		return nil, fmt.Errorf("config store not initialized")
	}
	return a.store.ListOpenClawConfigs()
}

// DeleteOpenClawConfig deletes an OpenClaw configuration
func (a *App) DeleteOpenClawConfig(name string) error {
	if a.store == nil {
		return fmt.Errorf("config store not initialized")
	}
	return a.store.DeleteOpenClawConfig(name)
}

// ValidateOpenClawConfig validates an OpenClaw configuration
func (a *App) ValidateOpenClawConfig(cfg *config.OpenClawConfig) *validator.ValidationResult {
	return a.validator.ValidateOpenClawConfig(cfg)
}

// GenerateOpenClawConfig generates OpenClaw configuration as a JSON string
func (a *App) GenerateOpenClawConfig(cfg *config.OpenClawConfig) (string, error) {
	gen := generator.NewOpenClawGenerator()
	return gen.GenerateString(cfg)
}

// ExportOpenClawConfig exports OpenClaw configuration to a selected directory
func (a *App) ExportOpenClawConfig(cfg *config.OpenClawConfig) (string, error) {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Export Directory",
	})
	if err != nil {
		return "", err
	}
	if dir == "" {
		return "", fmt.Errorf("no directory selected")
	}

	gen := generator.NewOpenClawGenerator()
	return gen.Generate(cfg, dir)
}
