package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"lurus-switch/internal/billing"
	"lurus-switch/internal/config"
	"lurus-switch/internal/generator"
	"lurus-switch/internal/installer"
	"lurus-switch/internal/packager"
	"lurus-switch/internal/proxy"
	"lurus-switch/internal/toolconfig"
	"lurus-switch/internal/updater"
	"lurus-switch/internal/validator"
)

// AppVersion is the current version of Lurus Switch, set at build time via -ldflags
var AppVersion = "0.1.0"

// App struct
type App struct {
	ctx           context.Context
	store         *config.Store
	validator     *validator.Validator
	instMgr       *installer.Manager
	proxyMgr      *proxy.ProxyManager
	selfUpdater   *updater.SelfUpdater
	npmChecker    *updater.NpmChecker
	billingMu     sync.Mutex
	billingClient *billing.Client
}

// NewApp creates a new App application struct
func NewApp() *App {
	store, err := config.NewStore()
	if err != nil {
		// Log error but continue - store will be nil
		fmt.Printf("Warning: failed to initialize config store: %v\n", err)
	}

	proxyMgr, _ := proxy.NewProxyManager()

	return &App{
		store:       store,
		validator:   validator.NewValidator(),
		instMgr:     installer.NewManager(),
		proxyMgr:    proxyMgr,
		selfUpdater: updater.NewSelfUpdater(AppVersion),
		npmChecker:  updater.NewNpmChecker(),
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

// ============================
// Tool Installation Methods
// ============================

// DetectAllTools checks installation status of all CLI tools
func (a *App) DetectAllTools() (map[string]*installer.ToolStatus, error) {
	return a.instMgr.DetectAll(a.ctx)
}

// InstallTool installs a specific CLI tool by name (claude/codex/gemini)
func (a *App) InstallTool(name string) (*installer.InstallResult, error) {
	return a.instMgr.InstallTool(a.ctx, name)
}

// InstallAllTools installs all CLI tools sequentially
func (a *App) InstallAllTools() []installer.InstallResult {
	return a.instMgr.InstallAll(a.ctx)
}

// UpdateTool updates a specific CLI tool to the latest version
func (a *App) UpdateTool(name string) (*installer.InstallResult, error) {
	return a.instMgr.UpdateTool(a.ctx, name)
}

// UpdateAllTools updates all CLI tools to the latest versions
func (a *App) UpdateAllTools() []installer.InstallResult {
	return a.instMgr.UpdateAll(a.ctx)
}

// ============================
// Update Check Methods
// ============================

// CheckAllUpdates checks for updates on all installed CLI tools
func (a *App) CheckAllUpdates() map[string]*updater.UpdateInfo {
	statuses, _ := a.instMgr.DetectAll(a.ctx)
	toolVersions := make(map[string]string)
	for name, status := range statuses {
		if status.Installed {
			toolVersions[name] = status.Version
		}
	}
	return a.npmChecker.CheckAllTools(toolVersions)
}

// CheckSelfUpdate checks if a newer version of Switch is available
func (a *App) CheckSelfUpdate() (*updater.UpdateInfo, error) {
	return a.selfUpdater.CheckUpdate()
}

// ApplySelfUpdate downloads and applies the latest Switch update
func (a *App) ApplySelfUpdate() error {
	return a.selfUpdater.ApplyUpdate()
}

// GetAppVersion returns the current Switch version string
func (a *App) GetAppVersion() string {
	return AppVersion
}

// ============================
// Proxy / NewAPI Methods
// ============================

// GetProxySettings returns the saved NewAPI proxy settings
func (a *App) GetProxySettings() *proxy.ProxySettings {
	if a.proxyMgr == nil {
		return &proxy.ProxySettings{}
	}
	return a.proxyMgr.GetSettings()
}

// SaveProxySettings saves NewAPI proxy settings to disk and updates billing client
func (a *App) SaveProxySettings(s *proxy.ProxySettings) error {
	if a.proxyMgr == nil {
		return fmt.Errorf("proxy manager not initialized")
	}
	if err := a.proxyMgr.SaveSettings(s); err != nil {
		return err
	}

	// Update billing client when token/tenant change
	a.billingMu.Lock()
	if s.UserToken != "" && s.APIEndpoint != "" {
		a.billingClient = billing.NewClient(s.APIEndpoint, s.TenantSlug, s.UserToken)
	} else {
		a.billingClient = nil
	}
	a.billingMu.Unlock()
	return nil
}

// ConfigureAllProxy applies the saved proxy settings to all installed tools
func (a *App) ConfigureAllProxy() map[string]string {
	if a.proxyMgr == nil {
		return map[string]string{"error": "proxy manager not initialized"}
	}
	settings := a.proxyMgr.GetSettings()
	errs := a.instMgr.ConfigureAllProxy(a.ctx, settings.APIEndpoint, settings.APIKey)
	result := make(map[string]string)
	for name, err := range errs {
		result[name] = err.Error()
	}
	return result
}

// ============================
// Billing Methods
// ============================

// ensureBillingClient lazily initializes the billing client from saved proxy settings.
// Returns a snapshot of the client safe to use without holding the lock.
func (a *App) ensureBillingClient() (*billing.Client, error) {
	a.billingMu.Lock()
	defer a.billingMu.Unlock()

	if a.billingClient != nil {
		return a.billingClient, nil
	}
	if a.proxyMgr == nil {
		return nil, fmt.Errorf("proxy manager not initialized")
	}
	s := a.proxyMgr.GetSettings()
	if s.UserToken == "" {
		return nil, fmt.Errorf("user token not configured: paste your token in Proxy Settings")
	}
	if s.APIEndpoint == "" {
		return nil, fmt.Errorf("API endpoint not configured")
	}
	a.billingClient = billing.NewClient(s.APIEndpoint, s.TenantSlug, s.UserToken)
	return a.billingClient, nil
}

// BillingGetUserInfo retrieves user account and quota information
func (a *App) BillingGetUserInfo() (*billing.UserInfo, error) {
	c, err := a.ensureBillingClient()
	if err != nil {
		return nil, err
	}
	return c.GetUserInfo(a.ctx)
}

// BillingGetQuotaSummary retrieves a lightweight quota summary for the dashboard
func (a *App) BillingGetQuotaSummary() (*billing.QuotaSummary, error) {
	c, err := a.ensureBillingClient()
	if err != nil {
		return nil, err
	}
	return c.GetQuotaSummary(a.ctx)
}

// BillingGetPlans retrieves available subscription plans
func (a *App) BillingGetPlans() ([]billing.SubscriptionPlan, error) {
	c, err := a.ensureBillingClient()
	if err != nil {
		return nil, err
	}
	return c.GetPlans(a.ctx)
}

// BillingGetSubscriptions retrieves the user's current subscriptions
func (a *App) BillingGetSubscriptions() ([]billing.SubscriptionInfo, error) {
	c, err := a.ensureBillingClient()
	if err != nil {
		return nil, err
	}
	return c.GetSubscriptions(a.ctx)
}

// BillingSubscribe creates a subscription request
func (a *App) BillingSubscribe(planCode, paymentMethod string) (*billing.PaymentResult, error) {
	c, err := a.ensureBillingClient()
	if err != nil {
		return nil, err
	}
	return c.Subscribe(a.ctx, planCode, paymentMethod)
}

// BillingCancelSubscription cancels an active subscription
func (a *App) BillingCancelSubscription(id int) error {
	c, err := a.ensureBillingClient()
	if err != nil {
		return err
	}
	return c.CancelSubscription(a.ctx, id)
}

// BillingGetTopUpInfo retrieves available top-up methods and options
func (a *App) BillingGetTopUpInfo() (*billing.TopUpInfo, error) {
	c, err := a.ensureBillingClient()
	if err != nil {
		return nil, err
	}
	return c.GetTopUpInfo(a.ctx)
}

// BillingCreateTopUp creates a top-up payment request
func (a *App) BillingCreateTopUp(amount int64, paymentMethod string) (*billing.PaymentResult, error) {
	c, err := a.ensureBillingClient()
	if err != nil {
		return nil, err
	}
	return c.CreateTopUp(a.ctx, amount, paymentMethod)
}

// BillingRedeemCode redeems a top-up code and returns the credited amount
func (a *App) BillingRedeemCode(code string) (int64, error) {
	c, err := a.ensureBillingClient()
	if err != nil {
		return 0, err
	}
	return c.RedeemCode(a.ctx, code)
}

// BillingOpenPaymentURL opens a payment URL in the user's default browser.
// Only http/https schemes are allowed to prevent arbitrary protocol opening.
func (a *App) BillingOpenPaymentURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid payment URL: %w", err)
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return fmt.Errorf("unsupported URL scheme %q: only http and https are allowed", parsed.Scheme)
	}
	if parsed.Host == "" {
		return fmt.Errorf("payment URL must contain a host")
	}
	runtime.BrowserOpenURL(a.ctx, parsed.String())
	return nil
}

// ============================
// Bun Runtime Methods
// ============================

// InstallBun installs the Bun runtime and returns its path
func (a *App) InstallBun() (string, error) {
	return a.instMgr.GetRuntime().InstallBun(a.ctx)
}

// ============================
// Tool Config File Methods
// ============================

// ReadToolConfig reads a tool's real config file from disk (claude/codex/gemini)
func (a *App) ReadToolConfig(tool string) (*toolconfig.ToolConfigInfo, error) {
	return toolconfig.ReadConfig(tool)
}

// SaveToolConfig writes content to a tool's real config file
func (a *App) SaveToolConfig(tool, content string) error {
	return toolconfig.WriteConfig(tool, content)
}

// GetToolConfigPath returns the full path to a tool's config file
func (a *App) GetToolConfigPath(tool string) (string, error) {
	return toolconfig.GetConfigPath(tool)
}

// OpenToolConfigDir opens the config directory of a tool in the file explorer
func (a *App) OpenToolConfigDir(tool string) error {
	return toolconfig.OpenConfigDirectory(tool)
}

// GetAllToolConfigPaths returns the config file paths for all tools
func (a *App) GetAllToolConfigPaths() map[string]string {
	return toolconfig.GetAllConfigPaths()
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
