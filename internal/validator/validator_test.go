package validator

import (
	"testing"

	"lurus-switch/internal/config"
)

// === Validator Constructor Tests ===

func TestNewValidator(t *testing.T) {
	v := NewValidator()
	if v == nil {
		t.Error("NewValidator should return non-nil validator")
	}
}

// === Claude Config Validation Tests ===

func TestValidateClaudeConfig_Valid(t *testing.T) {
	v := NewValidator()
	cfg := config.NewClaudeConfig()

	result := v.ValidateClaudeConfig(cfg)

	if !result.Valid {
		t.Errorf("Expected valid result, got errors: %v", result.Errors)
	}
	if len(result.Errors) != 0 {
		t.Errorf("Expected no errors, got %d", len(result.Errors))
	}
}

func TestValidateClaudeConfig_EmptyModel(t *testing.T) {
	v := NewValidator()
	cfg := &config.ClaudeConfig{}

	result := v.ValidateClaudeConfig(cfg)

	if result.Valid {
		t.Error("Expected invalid result for empty model")
	}
	if !hasError(result, "model", "model is required") {
		t.Error("Expected 'model is required' error")
	}
}

func TestValidateClaudeConfig_UnknownModel(t *testing.T) {
	v := NewValidator()
	cfg := &config.ClaudeConfig{
		Model: "gpt-4", // Not a Claude model
	}

	result := v.ValidateClaudeConfig(cfg)

	if result.Valid {
		t.Error("Expected invalid result for unknown model")
	}
	if !hasErrorField(result, "model") {
		t.Error("Expected model validation error")
	}
}

func TestValidateClaudeConfig_KnownModels(t *testing.T) {
	v := NewValidator()

	knownModels := []string{
		"claude-3-opus-20240229",
		"claude-3-sonnet-20240229",
		"claude-3-haiku-20240307",
		"claude-sonnet-4-20250514",
		"claude-opus-4-20250514",
		"claude-3-5-sonnet-20241022",
		"claude-3-5-haiku-20241022",
	}

	for _, model := range knownModels {
		cfg := config.NewClaudeConfig()
		cfg.Model = model

		result := v.ValidateClaudeConfig(cfg)

		if !result.Valid {
			t.Errorf("Model %s should be valid, got errors: %v", model, result.Errors)
		}
	}
}

func TestValidateClaudeConfig_CustomModelPrefix(t *testing.T) {
	v := NewValidator()
	cfg := config.NewClaudeConfig()
	cfg.Model = "claude-custom-model-v1"

	result := v.ValidateClaudeConfig(cfg)

	if !result.Valid {
		t.Error("Custom claude- prefixed model should be valid")
	}
}

func TestValidateClaudeConfig_InvalidAPIKey_NoPrefix(t *testing.T) {
	v := NewValidator()
	cfg := config.NewClaudeConfig()
	cfg.APIKey = "invalid-api-key"

	result := v.ValidateClaudeConfig(cfg)

	if result.Valid {
		t.Error("Expected invalid result for API key without proper prefix")
	}
	if !hasErrorField(result, "apiKey") {
		t.Error("Expected apiKey validation error")
	}
}

func TestValidateClaudeConfig_ValidAPIKey_SkAnt(t *testing.T) {
	v := NewValidator()
	cfg := config.NewClaudeConfig()
	cfg.APIKey = "sk-ant-api01-abc123xyz789"

	result := v.ValidateClaudeConfig(cfg)

	if !result.Valid {
		t.Errorf("API key with sk-ant- prefix should be valid, got errors: %v", result.Errors)
	}
}

func TestValidateClaudeConfig_ValidAPIKey_Sk(t *testing.T) {
	v := NewValidator()
	cfg := config.NewClaudeConfig()
	cfg.APIKey = "sk-abc123xyz789"

	result := v.ValidateClaudeConfig(cfg)

	if !result.Valid {
		t.Errorf("API key with sk- prefix should be valid, got errors: %v", result.Errors)
	}
}

func TestValidateClaudeConfig_EmptyAPIKey(t *testing.T) {
	v := NewValidator()
	cfg := config.NewClaudeConfig()
	cfg.APIKey = "" // Empty is allowed (optional)

	result := v.ValidateClaudeConfig(cfg)

	if !result.Valid {
		t.Error("Empty API key should be valid (optional field)")
	}
}

func TestValidateClaudeConfig_NegativeMaxTokens(t *testing.T) {
	v := NewValidator()
	cfg := config.NewClaudeConfig()
	cfg.MaxTokens = -1

	result := v.ValidateClaudeConfig(cfg)

	if result.Valid {
		t.Error("Expected invalid result for negative maxTokens")
	}
	if !hasErrorField(result, "maxTokens") {
		t.Error("Expected maxTokens validation error")
	}
}

func TestValidateClaudeConfig_ExcessiveMaxTokens(t *testing.T) {
	v := NewValidator()
	cfg := config.NewClaudeConfig()
	cfg.MaxTokens = 300000 // Exceeds 200000 limit

	result := v.ValidateClaudeConfig(cfg)

	if result.Valid {
		t.Error("Expected invalid result for excessive maxTokens")
	}
	if !hasError(result, "maxTokens", "maxTokens exceeds maximum (200000)") {
		t.Error("Expected 'maxTokens exceeds maximum' error")
	}
}

func TestValidateClaudeConfig_MaxTokensBoundary(t *testing.T) {
	v := NewValidator()

	testCases := []struct {
		name     string
		tokens   int
		expected bool
	}{
		{"zero", 0, true},
		{"positive", 1000, true},
		{"max boundary", 200000, true},
		{"over max", 200001, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.NewClaudeConfig()
			cfg.MaxTokens = tc.tokens

			result := v.ValidateClaudeConfig(cfg)

			if result.Valid != tc.expected {
				t.Errorf("MaxTokens=%d: expected valid=%v, got valid=%v, errors=%v",
					tc.tokens, tc.expected, result.Valid, result.Errors)
			}
		})
	}
}

func TestValidateClaudeConfig_InvalidSandboxType(t *testing.T) {
	v := NewValidator()
	cfg := config.NewClaudeConfig()
	cfg.Sandbox.Enabled = true
	cfg.Sandbox.Type = "invalid"

	result := v.ValidateClaudeConfig(cfg)

	if result.Valid {
		t.Error("Expected invalid result for invalid sandbox type")
	}
	if !hasErrorField(result, "sandbox.type") {
		t.Error("Expected sandbox.type validation error")
	}
}

func TestValidateClaudeConfig_ValidSandboxTypes(t *testing.T) {
	v := NewValidator()

	validTypes := []string{"docker", "wsl", "none"}

	for _, sandboxType := range validTypes {
		t.Run(sandboxType, func(t *testing.T) {
			cfg := config.NewClaudeConfig()
			cfg.Sandbox.Enabled = true
			cfg.Sandbox.Type = sandboxType

			result := v.ValidateClaudeConfig(cfg)

			if !result.Valid {
				t.Errorf("Sandbox type %s should be valid, got errors: %v", sandboxType, result.Errors)
			}
		})
	}
}

func TestValidateClaudeConfig_DisabledSandboxSkipsValidation(t *testing.T) {
	v := NewValidator()
	cfg := config.NewClaudeConfig()
	cfg.Sandbox.Enabled = false
	cfg.Sandbox.Type = "invalid" // Should not matter when disabled

	result := v.ValidateClaudeConfig(cfg)

	if !result.Valid {
		t.Error("Disabled sandbox should skip type validation")
	}
}

// === Codex Config Validation Tests ===

func TestValidateCodexConfig_Valid(t *testing.T) {
	v := NewValidator()
	cfg := config.NewCodexConfig()

	result := v.ValidateCodexConfig(cfg)

	if !result.Valid {
		t.Errorf("Expected valid result, got errors: %v", result.Errors)
	}
}

func TestValidateCodexConfig_EmptyModel(t *testing.T) {
	v := NewValidator()
	cfg := config.NewCodexConfig()
	cfg.Model = ""

	result := v.ValidateCodexConfig(cfg)

	if result.Valid {
		t.Error("Expected invalid result for empty model")
	}
	if !hasError(result, "model", "model is required") {
		t.Error("Expected 'model is required' error")
	}
}

func TestValidateCodexConfig_InvalidApprovalMode(t *testing.T) {
	v := NewValidator()
	cfg := config.NewCodexConfig()
	cfg.ApprovalMode = "invalid"

	result := v.ValidateCodexConfig(cfg)

	if result.Valid {
		t.Error("Expected invalid result for invalid approval mode")
	}
	if !hasErrorField(result, "approvalMode") {
		t.Error("Expected approvalMode validation error")
	}
}

func TestValidateCodexConfig_AllApprovalModes(t *testing.T) {
	v := NewValidator()

	validModes := []string{"suggest", "auto-edit", "full-auto"}

	for _, mode := range validModes {
		t.Run(mode, func(t *testing.T) {
			cfg := config.NewCodexConfig()
			cfg.ApprovalMode = mode

			result := v.ValidateCodexConfig(cfg)

			if !result.Valid {
				t.Errorf("Approval mode %s should be valid, got errors: %v", mode, result.Errors)
			}
		})
	}
}

func TestValidateCodexConfig_InvalidProvider(t *testing.T) {
	v := NewValidator()
	cfg := config.NewCodexConfig()
	cfg.Provider.Type = "invalid"

	result := v.ValidateCodexConfig(cfg)

	if result.Valid {
		t.Error("Expected invalid result for invalid provider type")
	}
	if !hasErrorField(result, "provider.type") {
		t.Error("Expected provider.type validation error")
	}
}

func TestValidateCodexConfig_AllProviders(t *testing.T) {
	v := NewValidator()

	validProviders := []string{"openai", "azure", "openrouter", "custom"}

	for _, provider := range validProviders {
		t.Run(provider, func(t *testing.T) {
			cfg := config.NewCodexConfig()
			cfg.Provider.Type = provider
			// Add required fields for Azure
			if provider == "azure" {
				cfg.Provider.AzureDeployment = "my-deployment"
				cfg.Provider.BaseURL = "https://my-resource.openai.azure.com"
			}

			result := v.ValidateCodexConfig(cfg)

			if !result.Valid {
				t.Errorf("Provider %s should be valid, got errors: %v", provider, result.Errors)
			}
		})
	}
}

func TestValidateCodexConfig_Azure_MissingDeployment(t *testing.T) {
	v := NewValidator()
	cfg := config.NewCodexConfig()
	cfg.Provider.Type = "azure"
	cfg.Provider.BaseURL = "https://my-resource.openai.azure.com"
	cfg.Provider.AzureDeployment = ""

	result := v.ValidateCodexConfig(cfg)

	if result.Valid {
		t.Error("Expected invalid result for Azure provider without deployment")
	}
	if !hasErrorField(result, "provider.azureDeployment") {
		t.Error("Expected provider.azureDeployment validation error")
	}
}

func TestValidateCodexConfig_Azure_MissingBaseURL(t *testing.T) {
	v := NewValidator()
	cfg := config.NewCodexConfig()
	cfg.Provider.Type = "azure"
	cfg.Provider.AzureDeployment = "my-deployment"
	cfg.Provider.BaseURL = ""

	result := v.ValidateCodexConfig(cfg)

	if result.Valid {
		t.Error("Expected invalid result for Azure provider without base URL")
	}
	if !hasErrorField(result, "provider.baseUrl") {
		t.Error("Expected provider.baseUrl validation error")
	}
}

func TestValidateCodexConfig_Azure_Valid(t *testing.T) {
	v := NewValidator()
	cfg := config.NewCodexConfig()
	cfg.Provider.Type = "azure"
	cfg.Provider.AzureDeployment = "my-deployment"
	cfg.Provider.BaseURL = "https://my-resource.openai.azure.com"

	result := v.ValidateCodexConfig(cfg)

	if !result.Valid {
		t.Errorf("Valid Azure config should pass, got errors: %v", result.Errors)
	}
}

func TestValidateCodexConfig_InvalidNetworkAccess(t *testing.T) {
	v := NewValidator()
	cfg := config.NewCodexConfig()
	cfg.Security.NetworkAccess = "invalid"

	result := v.ValidateCodexConfig(cfg)

	if result.Valid {
		t.Error("Expected invalid result for invalid network access")
	}
	if !hasErrorField(result, "security.networkAccess") {
		t.Error("Expected security.networkAccess validation error")
	}
}

func TestValidateCodexConfig_AllNetworkAccess(t *testing.T) {
	v := NewValidator()

	validAccess := []string{"off", "local", "full"}

	for _, access := range validAccess {
		t.Run(access, func(t *testing.T) {
			cfg := config.NewCodexConfig()
			cfg.Security.NetworkAccess = access

			result := v.ValidateCodexConfig(cfg)

			if !result.Valid {
				t.Errorf("Network access %s should be valid, got errors: %v", access, result.Errors)
			}
		})
	}
}

// === Gemini Config Validation Tests ===

func TestValidateGeminiConfig_Valid(t *testing.T) {
	v := NewValidator()
	cfg := config.NewGeminiConfig()

	result := v.ValidateGeminiConfig(cfg)

	if !result.Valid {
		t.Errorf("Expected valid result, got errors: %v", result.Errors)
	}
}

func TestValidateGeminiConfig_EmptyModel(t *testing.T) {
	v := NewValidator()
	cfg := config.NewGeminiConfig()
	cfg.Model = ""

	result := v.ValidateGeminiConfig(cfg)

	if result.Valid {
		t.Error("Expected invalid result for empty model")
	}
	if !hasError(result, "model", "model is required") {
		t.Error("Expected 'model is required' error")
	}
}

func TestValidateGeminiConfig_UnknownModel(t *testing.T) {
	v := NewValidator()
	cfg := config.NewGeminiConfig()
	cfg.Model = "gpt-4"

	result := v.ValidateGeminiConfig(cfg)

	if result.Valid {
		t.Error("Expected invalid result for unknown model")
	}
	if !hasErrorField(result, "model") {
		t.Error("Expected model validation error")
	}
}

func TestValidateGeminiConfig_KnownModels(t *testing.T) {
	v := NewValidator()

	knownModels := []string{
		"gemini-pro",
		"gemini-1.5-pro",
		"gemini-1.5-flash",
		"gemini-2.0-flash",
		"gemini-2.0-pro",
	}

	for _, model := range knownModels {
		t.Run(model, func(t *testing.T) {
			cfg := config.NewGeminiConfig()
			cfg.Model = model

			result := v.ValidateGeminiConfig(cfg)

			if !result.Valid {
				t.Errorf("Model %s should be valid, got errors: %v", model, result.Errors)
			}
		})
	}
}

func TestValidateGeminiConfig_CustomModelPrefix(t *testing.T) {
	v := NewValidator()
	cfg := config.NewGeminiConfig()
	cfg.Model = "gemini-custom-model-v1"

	result := v.ValidateGeminiConfig(cfg)

	if !result.Valid {
		t.Error("Custom gemini- prefixed model should be valid")
	}
}

func TestValidateGeminiConfig_InvalidAuthType(t *testing.T) {
	v := NewValidator()
	cfg := config.NewGeminiConfig()
	cfg.Auth.Type = "invalid"

	result := v.ValidateGeminiConfig(cfg)

	if result.Valid {
		t.Error("Expected invalid result for invalid auth type")
	}
	if !hasErrorField(result, "auth.type") {
		t.Error("Expected auth.type validation error")
	}
}

func TestValidateGeminiConfig_AllAuthTypes(t *testing.T) {
	v := NewValidator()

	validTypes := []string{"api_key", "oauth", "adc"}

	for _, authType := range validTypes {
		t.Run(authType, func(t *testing.T) {
			cfg := config.NewGeminiConfig()
			cfg.Auth.Type = authType

			result := v.ValidateGeminiConfig(cfg)

			if !result.Valid {
				t.Errorf("Auth type %s should be valid, got errors: %v", authType, result.Errors)
			}
		})
	}
}

func TestValidateGeminiConfig_InvalidAPIKey(t *testing.T) {
	v := NewValidator()
	cfg := config.NewGeminiConfig()
	cfg.Auth.Type = "api_key"
	cfg.APIKey = "short" // Too short (not 39 chars)

	result := v.ValidateGeminiConfig(cfg)

	if result.Valid {
		t.Error("Expected invalid result for invalid API key format")
	}
	if !hasErrorField(result, "apiKey") {
		t.Error("Expected apiKey validation error")
	}
}

func TestValidateGeminiConfig_ValidAPIKey_39Chars(t *testing.T) {
	v := NewValidator()
	cfg := config.NewGeminiConfig()
	cfg.Auth.Type = "api_key"
	cfg.APIKey = "AIzaSyAbCdEfGhIjKlMnOpQrStUvWxYz1234567" // Exactly 39 chars

	result := v.ValidateGeminiConfig(cfg)

	if !result.Valid {
		t.Errorf("Valid 39-char API key should pass, got errors: %v", result.Errors)
	}
}

func TestValidateGeminiConfig_EmptyAPIKey(t *testing.T) {
	v := NewValidator()
	cfg := config.NewGeminiConfig()
	cfg.Auth.Type = "api_key"
	cfg.APIKey = "" // Empty is allowed

	result := v.ValidateGeminiConfig(cfg)

	if !result.Valid {
		t.Error("Empty API key should be valid (optional)")
	}
}

func TestValidateGeminiConfig_InvalidTheme(t *testing.T) {
	v := NewValidator()
	cfg := config.NewGeminiConfig()
	cfg.Display.Theme = "invalid"

	result := v.ValidateGeminiConfig(cfg)

	if result.Valid {
		t.Error("Expected invalid result for invalid theme")
	}
	if !hasErrorField(result, "display.theme") {
		t.Error("Expected display.theme validation error")
	}
}

func TestValidateGeminiConfig_AllThemes(t *testing.T) {
	v := NewValidator()

	validThemes := []string{"dark", "light", "auto"}

	for _, theme := range validThemes {
		t.Run(theme, func(t *testing.T) {
			cfg := config.NewGeminiConfig()
			cfg.Display.Theme = theme

			result := v.ValidateGeminiConfig(cfg)

			if !result.Valid {
				t.Errorf("Theme %s should be valid, got errors: %v", theme, result.Errors)
			}
		})
	}
}

func TestValidateGeminiConfig_NegativeMaxFileSize(t *testing.T) {
	v := NewValidator()
	cfg := config.NewGeminiConfig()
	cfg.Behavior.MaxFileSize = -1

	result := v.ValidateGeminiConfig(cfg)

	if result.Valid {
		t.Error("Expected invalid result for negative maxFileSize")
	}
	if !hasErrorField(result, "behavior.maxFileSize") {
		t.Error("Expected behavior.maxFileSize validation error")
	}
}

func TestValidateGeminiConfig_ZeroMaxFileSize(t *testing.T) {
	v := NewValidator()
	cfg := config.NewGeminiConfig()
	cfg.Behavior.MaxFileSize = 0 // Zero is valid

	result := v.ValidateGeminiConfig(cfg)

	if !result.Valid {
		t.Error("Zero maxFileSize should be valid")
	}
}

// === ValidationResult Tests ===

func TestValidationResult_AddError(t *testing.T) {
	result := &ValidationResult{Valid: true}

	result.addError("field1", "error message 1")

	if result.Valid {
		t.Error("Valid should be false after adding error")
	}
	if len(result.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(result.Errors))
	}
	if result.Errors[0].Field != "field1" {
		t.Errorf("Expected field 'field1', got '%s'", result.Errors[0].Field)
	}
	if result.Errors[0].Message != "error message 1" {
		t.Errorf("Expected message 'error message 1', got '%s'", result.Errors[0].Message)
	}
}

func TestValidationResult_MultipleErrors(t *testing.T) {
	result := &ValidationResult{Valid: true}

	result.addError("field1", "error 1")
	result.addError("field2", "error 2")
	result.addError("field3", "error 3")

	if result.Valid {
		t.Error("Valid should be false after adding errors")
	}
	if len(result.Errors) != 3 {
		t.Errorf("Expected 3 errors, got %d", len(result.Errors))
	}
}

// === Helper Function Tests ===

func TestContains(t *testing.T) {
	testCases := []struct {
		slice    []string
		item     string
		expected bool
	}{
		{[]string{"a", "b", "c"}, "a", true},
		{[]string{"a", "b", "c"}, "b", true},
		{[]string{"a", "b", "c"}, "c", true},
		{[]string{"a", "b", "c"}, "d", false},
		{[]string{}, "a", false},
		{nil, "a", false},
	}

	for _, tc := range testCases {
		result := contains(tc.slice, tc.item)
		if result != tc.expected {
			t.Errorf("contains(%v, %s) = %v, expected %v", tc.slice, tc.item, result, tc.expected)
		}
	}
}

// === PicoClaw Config Validation Tests ===

func TestValidatePicoClawConfig_Valid(t *testing.T) {
	v := NewValidator()
	cfg := config.NewPicoClawConfig()

	result := v.ValidatePicoClawConfig(cfg)
	if !result.Valid {
		t.Errorf("expected valid, got errors: %v", result.Errors)
	}
}

func TestValidatePicoClawConfig_Nil(t *testing.T) {
	v := NewValidator()
	result := v.ValidatePicoClawConfig(nil)

	if result.Valid {
		t.Error("nil config should be invalid")
	}
	if !hasErrorField(result, "config") {
		t.Error("expected config error")
	}
}

func TestValidatePicoClawConfig_EmptyModelList(t *testing.T) {
	v := NewValidator()
	cfg := &config.PicoClawConfig{
		ModelList: []config.PicoClawModel{},
	}

	result := v.ValidatePicoClawConfig(cfg)
	if result.Valid {
		t.Error("empty model_list should be invalid")
	}
	if !hasErrorField(result, "model_list") {
		t.Error("expected model_list error")
	}
}

func TestValidatePicoClawConfig_MissingModelName(t *testing.T) {
	v := NewValidator()
	cfg := &config.PicoClawConfig{
		ModelList: []config.PicoClawModel{
			{Name: "", APIBase: "https://api.example.com", ModelName: "test"},
		},
	}

	result := v.ValidatePicoClawConfig(cfg)
	if result.Valid {
		t.Error("missing model name should be invalid")
	}
	if !hasErrorField(result, "model_list[0].name") {
		t.Error("expected model_list[0].name error")
	}
}

func TestValidatePicoClawConfig_DuplicateModelNames(t *testing.T) {
	v := NewValidator()
	cfg := &config.PicoClawConfig{
		ModelList: []config.PicoClawModel{
			{Name: "default", ModelName: "m1"},
			{Name: "default", ModelName: "m2"},
		},
	}

	result := v.ValidatePicoClawConfig(cfg)
	if result.Valid {
		t.Error("duplicate names should be invalid")
	}
	if !hasErrorField(result, "model_list[1].name") {
		t.Error("expected duplicate name error on second entry")
	}
}

func TestValidatePicoClawConfig_InvalidAPIBaseURL(t *testing.T) {
	v := NewValidator()
	cfg := &config.PicoClawConfig{
		ModelList: []config.PicoClawModel{
			{Name: "test", APIBase: "not-a-url", ModelName: "m1"},
		},
	}

	result := v.ValidatePicoClawConfig(cfg)
	if result.Valid {
		t.Error("invalid URL should be invalid")
	}
	if !hasErrorField(result, "model_list[0].api_base") {
		t.Error("expected api_base error")
	}
}

func TestValidatePicoClawConfig_EmptyAPIBaseIsValid(t *testing.T) {
	v := NewValidator()
	cfg := &config.PicoClawConfig{
		ModelList: []config.PicoClawModel{
			{Name: "test", APIBase: "", ModelName: "m1"},
		},
	}

	result := v.ValidatePicoClawConfig(cfg)
	if !result.Valid {
		t.Errorf("empty api_base should be valid, got errors: %v", result.Errors)
	}
}

func TestValidatePicoClawConfig_AgentDefaultNotInList(t *testing.T) {
	v := NewValidator()
	cfg := &config.PicoClawConfig{
		ModelList: []config.PicoClawModel{
			{Name: "model-a", ModelName: "m1"},
			{Name: "model-b", ModelName: "m2"},
		},
		Agents: config.PicoClawAgentSettings{
			Defaults: config.PicoClawAgentDefaults{
				ModelName: "nonexistent-model",
			},
		},
	}

	result := v.ValidatePicoClawConfig(cfg)
	if result.Valid {
		t.Error("agent default referencing non-existent model should be invalid")
	}
	if !hasErrorField(result, "agents.defaults.model_name") {
		t.Error("expected agents.defaults.model_name error")
	}
}

func TestValidatePicoClawConfig_AgentDefaultMatchesByModelName(t *testing.T) {
	// agents.defaults.model_name references the LLM model name (ModelName field)
	v := NewValidator()
	cfg := &config.PicoClawConfig{
		ModelList: []config.PicoClawModel{
			{Name: "my-proxy", APIBase: "https://api.example.com", ModelName: "claude-sonnet-4"},
		},
		Agents: config.PicoClawAgentSettings{
			Defaults: config.PicoClawAgentDefaults{
				ModelName: "claude-sonnet-4", // Matches ModelName field
			},
		},
	}

	result := v.ValidatePicoClawConfig(cfg)
	if !result.Valid {
		t.Errorf("agent default matching ModelName should be valid, got: %v", result.Errors)
	}
}

func TestValidatePicoClawConfig_AgentDefaultMatchesEntryName_NotModelName(t *testing.T) {
	// agents.defaults.model_name does NOT match by entry Name — it uses ModelName
	v := NewValidator()
	cfg := &config.PicoClawConfig{
		ModelList: []config.PicoClawModel{
			{Name: "my-proxy", ModelName: "claude-sonnet-4"},
		},
		Agents: config.PicoClawAgentSettings{
			Defaults: config.PicoClawAgentDefaults{
				ModelName: "my-proxy", // This matches Name but NOT ModelName — should fail
			},
		},
	}

	result := v.ValidatePicoClawConfig(cfg)
	if result.Valid {
		t.Error("agent default matching Name but not ModelName should be invalid")
	}
}

func TestValidatePicoClawConfig_EmptyAgentDefault(t *testing.T) {
	v := NewValidator()
	cfg := &config.PicoClawConfig{
		ModelList: []config.PicoClawModel{
			{Name: "test", ModelName: "m1"},
		},
		Agents: config.PicoClawAgentSettings{
			Defaults: config.PicoClawAgentDefaults{
				ModelName: "", // Empty is allowed (skips check)
			},
		},
	}

	result := v.ValidatePicoClawConfig(cfg)
	if !result.Valid {
		t.Errorf("empty agent default should skip validation, got: %v", result.Errors)
	}
}

func TestValidatePicoClawConfig_MultipleErrors(t *testing.T) {
	v := NewValidator()
	cfg := &config.PicoClawConfig{
		ModelList: []config.PicoClawModel{
			{Name: "", APIBase: "bad-url", ModelName: ""},
			{Name: "", APIBase: "", ModelName: ""},
		},
	}

	result := v.ValidatePicoClawConfig(cfg)
	if result.Valid {
		t.Error("config with multiple issues should be invalid")
	}
	// Should have errors for both missing names and invalid URL
	if len(result.Errors) < 3 {
		t.Errorf("expected at least 3 errors, got %d: %v", len(result.Errors), result.Errors)
	}
}

func TestValidatePicoClawConfig_ValidURL(t *testing.T) {
	v := NewValidator()
	cfg := &config.PicoClawConfig{
		ModelList: []config.PicoClawModel{
			{Name: "test", APIBase: "https://api.example.com/v1", ModelName: "m"},
		},
	}

	result := v.ValidatePicoClawConfig(cfg)
	if !result.Valid {
		t.Errorf("valid URL should pass: %v", result.Errors)
	}
}

// === Test Helpers ===

func hasError(result *ValidationResult, field, message string) bool {
	for _, err := range result.Errors {
		if err.Field == field && err.Message == message {
			return true
		}
	}
	return false
}

func hasErrorField(result *ValidationResult, field string) bool {
	for _, err := range result.Errors {
		if err.Field == field {
			return true
		}
	}
	return false
}

// === ZeroClaw Config Validation Tests ===

func TestValidateZeroClawConfig_Nil(t *testing.T) {
	v := NewValidator()
	result := v.ValidateZeroClawConfig(nil)
	if result.Valid {
		t.Error("expected invalid result for nil config")
	}
}

func TestValidateZeroClawConfig_InvalidPort(t *testing.T) {
	v := NewValidator()
	cfg := config.NewZeroClawConfig()
	cfg.Gateway.Port = 99999

	result := v.ValidateZeroClawConfig(cfg)
	if result.Valid {
		t.Error("expected invalid result for out-of-range port")
	}
	if !hasErrorField(result, "gateway.port") {
		t.Error("expected gateway.port error field")
	}
}

func TestValidateZeroClawConfig_MissingAPIKey_IsAdvisory(t *testing.T) {
	v := NewValidator()
	cfg := config.NewZeroClawConfig()
	cfg.Provider.APIKey = ""

	result := v.ValidateZeroClawConfig(cfg)
	// api_key missing is advisory (not truly invalid)
	_ = result
}

// === OpenClaw Config Validation Tests ===

func TestValidateOpenClawConfig_Nil(t *testing.T) {
	v := NewValidator()
	result := v.ValidateOpenClawConfig(nil)
	if result.Valid {
		t.Error("expected invalid result for nil config")
	}
}

func TestValidateOpenClawConfig_InvalidPort(t *testing.T) {
	v := NewValidator()
	cfg := config.NewOpenClawConfig()
	cfg.Gateway.Port = 0 // invalid

	result := v.ValidateOpenClawConfig(cfg)
	if result.Valid {
		t.Error("expected invalid result for port=0")
	}
	if !hasErrorField(result, "gateway.port") {
		t.Error("expected gateway.port error field")
	}
}

func TestValidateOpenClawConfig_InvalidProviderType(t *testing.T) {
	v := NewValidator()
	cfg := config.NewOpenClawConfig()
	cfg.Provider.Type = "bogus-provider"

	result := v.ValidateOpenClawConfig(cfg)
	if result.Valid {
		t.Error("expected invalid result for unknown provider type")
	}
	if !hasErrorField(result, "provider.type") {
		t.Error("expected provider.type error field")
	}
}

func TestValidateOpenClawConfig_ValidDefaults(t *testing.T) {
	v := NewValidator()
	cfg := config.NewOpenClawConfig()

	result := v.ValidateOpenClawConfig(cfg)
	if !result.Valid {
		t.Errorf("expected valid result for default config, got errors: %v", result.Errors)
	}
}
