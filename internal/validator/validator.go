package validator

import (
	"fmt"
	"net/url"
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

	// Advanced: API endpoint must be a valid http/https URL if set
	if cfg.Advanced.APIEndpoint != "" {
		u, err := url.Parse(cfg.Advanced.APIEndpoint)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			result.addError("advanced.apiEndpoint", "must be a valid http:// or https:// URL")
		}
	}

	// Advanced: timeout must be in a reasonable range if non-zero
	if cfg.Advanced.Timeout != 0 && (cfg.Advanced.Timeout < 5 || cfg.Advanced.Timeout > 3600) {
		result.addError("advanced.timeout", "timeout must be between 5 and 3600 seconds")
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

	// API key format: OpenAI keys start with "sk-"
	if cfg.APIKey != "" && !strings.HasPrefix(cfg.APIKey, "sk-") {
		result.addError("apiKey", "OpenAI API key should start with sk-")
	}

	// History max entries bounds
	if cfg.History.MaxEntries < 0 {
		result.addError("history.maxEntries", "maxEntries must not be negative")
	} else if cfg.History.MaxEntries > 100000 {
		result.addError("history.maxEntries", "maxEntries must not exceed 100000")
	}

	// Provider base URL must be valid when set
	if cfg.Provider.BaseURL != "" {
		u, err := url.Parse(cfg.Provider.BaseURL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			result.addError("provider.baseUrl", "must be a valid http:// or https:// URL")
		}
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

	// API key format hint (if using api_key auth)
	if cfg.Auth.Type == "api_key" && cfg.APIKey != "" {
		if !strings.HasPrefix(cfg.APIKey, "AIza") {
			result.addError("apiKey", "Google API key typically starts with AIza")
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

// ValidatePicoClawConfig validates a PicoClaw configuration
func (v *Validator) ValidatePicoClawConfig(cfg *config.PicoClawConfig) *ValidationResult {
	result := &ValidationResult{Valid: true}

	if cfg == nil {
		result.addError("config", "PicoClaw config must not be nil")
		return result
	}

	// model_list must have at least one entry
	if len(cfg.ModelList) == 0 {
		result.addError("model_list", "model_list must contain at least one model")
	}

	// Validate each model entry
	modelNames := make(map[string]bool)
	for i, m := range cfg.ModelList {
		if m.Name == "" {
			result.addError(fmt.Sprintf("model_list[%d].name", i), "model name is required")
		} else {
			if modelNames[m.Name] {
				result.addError(fmt.Sprintf("model_list[%d].name", i), fmt.Sprintf("duplicate model name: %s", m.Name))
			}
			modelNames[m.Name] = true
		}

		// Validate api_base is a valid URL if provided
		if m.APIBase != "" {
			if _, err := url.ParseRequestURI(m.APIBase); err != nil {
				result.addError(fmt.Sprintf("model_list[%d].api_base", i), fmt.Sprintf("invalid URL: %s", m.APIBase))
			}
		}
	}

	// Validate agents.defaults.model_name exists in model_list
	defaultModel := cfg.Agents.Defaults.ModelName
	if defaultModel != "" && len(cfg.ModelList) > 0 {
		found := false
		for _, m := range cfg.ModelList {
			if m.ModelName == defaultModel {
				found = true
				break
			}
		}
		if !found {
			result.addError("agents.defaults.model_name", fmt.Sprintf("model_name %q not found in any model_list entry", defaultModel))
		}
	}

	return result
}

// ValidateNullClawConfig validates a NullClaw configuration
func (v *Validator) ValidateNullClawConfig(cfg *config.NullClawConfig) *ValidationResult {
	result := &ValidationResult{Valid: true}

	if cfg == nil {
		result.addError("config", "NullClaw config must not be nil")
		return result
	}

	// model_list must have at least one entry
	if len(cfg.ModelList) == 0 {
		result.addError("model_list", "model_list must contain at least one model")
	}

	// Validate each model entry
	modelNames := make(map[string]bool)
	for i, m := range cfg.ModelList {
		if m.Name == "" {
			result.addError(fmt.Sprintf("model_list[%d].name", i), "model name is required")
		} else {
			if modelNames[m.Name] {
				result.addError(fmt.Sprintf("model_list[%d].name", i), fmt.Sprintf("duplicate model name: %s", m.Name))
			}
			modelNames[m.Name] = true
		}

		// Validate api_base is a valid URL if provided
		if m.APIBase != "" {
			if _, err := url.ParseRequestURI(m.APIBase); err != nil {
				result.addError(fmt.Sprintf("model_list[%d].api_base", i), fmt.Sprintf("invalid URL: %s", m.APIBase))
			}
		}
	}

	// Validate agents.defaults.model_name exists in model_list
	defaultModel := cfg.Agents.Defaults.ModelName
	if defaultModel != "" && len(cfg.ModelList) > 0 {
		found := false
		for _, m := range cfg.ModelList {
			if m.ModelName == defaultModel {
				found = true
				break
			}
		}
		if !found {
			result.addError("agents.defaults.model_name", fmt.Sprintf("model_name %q not found in any model_list entry", defaultModel))
		}
	}

	return result
}

// ValidateZeroClawConfig validates a ZeroClaw configuration
func (v *Validator) ValidateZeroClawConfig(cfg *config.ZeroClawConfig) *ValidationResult {
	result := &ValidationResult{Valid: true}

	if cfg == nil {
		result.addError("config", "ZeroClaw config must not be nil")
		return result
	}

	// gateway.port must be in valid range when set
	if cfg.Gateway.Port != 0 && (cfg.Gateway.Port < 1 || cfg.Gateway.Port > 65535) {
		result.addError("gateway.port", fmt.Sprintf("port must be in range 1–65535, got %d", cfg.Gateway.Port))
	}

	// advisory: api_key should be set
	if cfg.Provider.APIKey == "" {
		result.addError("provider.api_key", "api_key is recommended; ensure it is set or provided via environment variable")
	}

	return result
}

// ValidateOpenClawConfig validates an OpenClaw configuration
func (v *Validator) ValidateOpenClawConfig(cfg *config.OpenClawConfig) *ValidationResult {
	result := &ValidationResult{Valid: true}

	if cfg == nil {
		result.addError("config", "OpenClaw config must not be nil")
		return result
	}

	// gateway.port must be in valid range
	if cfg.Gateway.Port < 1 || cfg.Gateway.Port > 65535 {
		result.addError("gateway.port", fmt.Sprintf("port must be in range 1–65535, got %d", cfg.Gateway.Port))
	}

	// provider.type must be a known value when set
	validProviders := []string{"anthropic", "openai", "custom"}
	if cfg.Provider.Type != "" && !contains(validProviders, cfg.Provider.Type) {
		result.addError("provider.type", fmt.Sprintf("invalid provider type %q (expected: anthropic, openai, custom)", cfg.Provider.Type))
	}

	return result
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
