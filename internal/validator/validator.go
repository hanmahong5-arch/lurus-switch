package validator

import (
	"fmt"
	"regexp"
	"strings"

	"lurus-switch/internal/config"
)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationResult contains the result of validation
type ValidationResult struct {
	Valid  bool              `json:"valid"`
	Errors []ValidationError `json:"errors"`
}

// Validator provides configuration validation
type Validator struct{}

// NewValidator creates a new validator
func NewValidator() *Validator {
	return &Validator{}
}

// ValidateClaudeConfig validates a Claude configuration
func (v *Validator) ValidateClaudeConfig(cfg *config.ClaudeConfig) *ValidationResult {
	result := &ValidationResult{Valid: true}

	// Model validation
	if cfg.Model == "" {
		result.addError("model", "model is required")
	} else if !v.isValidClaudeModel(cfg.Model) {
		result.addError("model", fmt.Sprintf("unknown model: %s", cfg.Model))
	}

	// API key validation (if provided)
	if cfg.APIKey != "" {
		if !strings.HasPrefix(cfg.APIKey, "sk-ant-") && !strings.HasPrefix(cfg.APIKey, "sk-") {
			result.addError("apiKey", "invalid API key format")
		}
	}

	// Max tokens validation
	if cfg.MaxTokens < 0 {
		result.addError("maxTokens", "maxTokens must be positive")
	} else if cfg.MaxTokens > 200000 {
		result.addError("maxTokens", "maxTokens exceeds maximum (200000)")
	}

	// Sandbox validation
	if cfg.Sandbox.Enabled {
		validTypes := []string{"docker", "wsl", "none"}
		if !contains(validTypes, cfg.Sandbox.Type) {
			result.addError("sandbox.type", fmt.Sprintf("invalid sandbox type: %s", cfg.Sandbox.Type))
		}
	}

	return result
}

// ValidateCodexConfig validates a Codex configuration
func (v *Validator) ValidateCodexConfig(cfg *config.CodexConfig) *ValidationResult {
	result := &ValidationResult{Valid: true}

	// Model validation
	if cfg.Model == "" {
		result.addError("model", "model is required")
	}

	// Approval mode validation
	validModes := []string{"suggest", "auto-edit", "full-auto"}
	if !contains(validModes, cfg.ApprovalMode) {
		result.addError("approvalMode", fmt.Sprintf("invalid approval mode: %s", cfg.ApprovalMode))
	}

	// Provider validation
	validProviders := []string{"openai", "azure", "openrouter", "custom"}
	if !contains(validProviders, cfg.Provider.Type) {
		result.addError("provider.type", fmt.Sprintf("invalid provider type: %s", cfg.Provider.Type))
	}

	// Azure validation
	if cfg.Provider.Type == "azure" {
		if cfg.Provider.AzureDeployment == "" {
			result.addError("provider.azureDeployment", "azure deployment is required for Azure provider")
		}
		if cfg.Provider.BaseURL == "" {
			result.addError("provider.baseUrl", "base URL is required for Azure provider")
		}
	}

	// Network access validation
	validNetworkAccess := []string{"off", "local", "full"}
	if !contains(validNetworkAccess, cfg.Security.NetworkAccess) {
		result.addError("security.networkAccess", fmt.Sprintf("invalid network access: %s", cfg.Security.NetworkAccess))
	}

	return result
}

// ValidateGeminiConfig validates a Gemini configuration
func (v *Validator) ValidateGeminiConfig(cfg *config.GeminiConfig) *ValidationResult {
	result := &ValidationResult{Valid: true}

	// Model validation
	if cfg.Model == "" {
		result.addError("model", "model is required")
	} else if !v.isValidGeminiModel(cfg.Model) {
		result.addError("model", fmt.Sprintf("unknown model: %s", cfg.Model))
	}

	// Auth type validation
	validAuthTypes := []string{"api_key", "oauth", "adc"}
	if !contains(validAuthTypes, cfg.Auth.Type) {
		result.addError("auth.type", fmt.Sprintf("invalid auth type: %s", cfg.Auth.Type))
	}

	// API key validation (if using api_key auth)
	if cfg.Auth.Type == "api_key" && cfg.APIKey != "" {
		if !regexp.MustCompile(`^[A-Za-z0-9_-]{39}$`).MatchString(cfg.APIKey) {
			// Google API keys are typically 39 characters
			result.addError("apiKey", "invalid API key format")
		}
	}

	// Display theme validation
	validThemes := []string{"dark", "light", "auto"}
	if !contains(validThemes, cfg.Display.Theme) {
		result.addError("display.theme", fmt.Sprintf("invalid theme: %s", cfg.Display.Theme))
	}

	// Max file size validation
	if cfg.Behavior.MaxFileSize < 0 {
		result.addError("behavior.maxFileSize", "maxFileSize must be positive")
	}

	return result
}

// isValidClaudeModel checks if the model is a known Claude model
func (v *Validator) isValidClaudeModel(model string) bool {
	knownModels := []string{
		"claude-3-opus",
		"claude-3-sonnet",
		"claude-3-haiku",
		"claude-sonnet-4",
		"claude-opus-4",
		"claude-3-5-sonnet",
		"claude-3-5-haiku",
	}

	for _, known := range knownModels {
		if strings.Contains(model, known) {
			return true
		}
	}

	// Allow custom model names
	return strings.HasPrefix(model, "claude-")
}

// isValidGeminiModel checks if the model is a known Gemini model
func (v *Validator) isValidGeminiModel(model string) bool {
	knownModels := []string{
		"gemini-pro",
		"gemini-1.5-pro",
		"gemini-1.5-flash",
		"gemini-2.0-flash",
		"gemini-2.0-pro",
	}

	for _, known := range knownModels {
		if strings.Contains(model, known) {
			return true
		}
	}

	// Allow custom model names
	return strings.HasPrefix(model, "gemini-")
}

// addError adds an error to the validation result
func (r *ValidationResult) addError(field, message string) {
	r.Valid = false
	r.Errors = append(r.Errors, ValidationError{
		Field:   field,
		Message: message,
	})
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
