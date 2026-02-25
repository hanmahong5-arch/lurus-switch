package config

import (
	"encoding/json"
	"strings"
	"testing"
)

// === Constructor Tests ===

func TestNewGeminiConfig(t *testing.T) {
	cfg := NewGeminiConfig()

	if cfg == nil {
		t.Fatal("NewGeminiConfig should return non-nil config")
	}
}

func TestNewGeminiConfig_DefaultValues(t *testing.T) {
	cfg := NewGeminiConfig()

	// Core defaults
	if cfg.Model != "gemini-2.0-flash" {
		t.Errorf("Expected model 'gemini-2.0-flash', got '%s'", cfg.Model)
	}

	// Auth defaults
	if cfg.Auth.Type != "api_key" {
		t.Errorf("Expected auth type 'api_key', got '%s'", cfg.Auth.Type)
	}

	// Behavior defaults
	if cfg.Behavior.Sandbox {
		t.Error("Sandbox should be disabled by default")
	}
	if cfg.Behavior.YoloMode {
		t.Error("YoloMode should be disabled by default")
	}
	if cfg.Behavior.MaxFileSize != 10*1024*1024 {
		t.Errorf("Expected maxFileSize 10MB, got %d", cfg.Behavior.MaxFileSize)
	}

	// Instructions defaults
	if cfg.Instructions.CustomRules == nil {
		t.Error("CustomRules should not be nil")
	}
	if len(cfg.Instructions.CustomRules) != 0 {
		t.Error("CustomRules should be empty by default")
	}

	// Display defaults
	if cfg.Display.Theme != "auto" {
		t.Errorf("Expected theme 'auto', got '%s'", cfg.Display.Theme)
	}
	if !cfg.Display.SyntaxHighlight {
		t.Error("SyntaxHighlight should be enabled by default")
	}
	if !cfg.Display.MarkdownRender {
		t.Error("MarkdownRender should be enabled by default")
	}
}

// === JSON Serialization Tests ===

func TestGeminiConfig_JSONMarshal_AllFields(t *testing.T) {
	cfg := &GeminiConfig{
		Model:     "gemini-2.0-pro",
		APIKey:    "AIzaSyTest123456789012345678901234567",
		ProjectID: "my-project-123",
		Auth: GeminiAuth{
			Type:               "oauth",
			OAuthClientID:      "client-id-123",
			ServiceAccountPath: "/path/to/sa.json",
		},
		Behavior: GeminiBehavior{
			Sandbox:           true,
			AutoApprove:       []string{"file_read", "web_search"},
			YoloMode:          false,
			MaxFileSize:       50 * 1024 * 1024,
			AllowedExtensions: []string{".go", ".ts", ".py"},
		},
		Instructions: GeminiInstructions{
			ProjectDescription: "A test project",
			TechStack:          "Go, React, PostgreSQL",
			CodeStyle:          "Google style guide",
			CustomRules:        []string{"Rule 1", "Rule 2"},
			FileStructure:      "Standard Go layout",
			TestingGuidelines:  "TDD approach",
		},
		Display: GeminiDisplay{
			Theme:           "dark",
			SyntaxHighlight: true,
			MarkdownRender:  true,
		},
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	jsonStr := string(data)
	expectedFields := []string{
		"model", "apiKey", "projectId",
		"auth", "type", "oauthClientId",
		"behavior", "sandbox", "yoloMode",
		"instructions", "projectDescription", "techStack",
		"display", "theme", "syntaxHighlight",
	}

	for _, field := range expectedFields {
		if !strings.Contains(jsonStr, field) {
			t.Errorf("JSON should contain field '%s'", field)
		}
	}
}

func TestGeminiConfig_JSONUnmarshal_PartialData(t *testing.T) {
	jsonData := `{
		"model": "gemini-1.5-pro",
		"auth": {"type": "adc"}
	}`

	var cfg GeminiConfig
	if err := json.Unmarshal([]byte(jsonData), &cfg); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if cfg.Model != "gemini-1.5-pro" {
		t.Errorf("Expected model 'gemini-1.5-pro', got '%s'", cfg.Model)
	}
	if cfg.Auth.Type != "adc" {
		t.Errorf("Expected auth type 'adc', got '%s'", cfg.Auth.Type)
	}
	// Other fields should be zero values
	if cfg.APIKey != "" {
		t.Error("APIKey should be empty")
	}
}

func TestGeminiConfig_JSONRoundTrip(t *testing.T) {
	original := NewGeminiConfig()
	original.APIKey = "AIzaSyTest123456789012345678901234567"
	original.ProjectID = "test-project"
	original.Instructions.CustomRules = []string{"Rule 1", "Rule 2"}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded GeminiConfig
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Model != original.Model {
		t.Error("Model mismatch after round-trip")
	}
	if decoded.APIKey != original.APIKey {
		t.Error("APIKey mismatch after round-trip")
	}
	if decoded.ProjectID != original.ProjectID {
		t.Error("ProjectID mismatch after round-trip")
	}
	if len(decoded.Instructions.CustomRules) != 2 {
		t.Error("CustomRules length mismatch after round-trip")
	}
}

// === GenerateMarkdown Tests ===

func TestGeminiConfig_GenerateMarkdown_Empty(t *testing.T) {
	cfg := &GeminiConfig{}

	md := cfg.GenerateMarkdown()

	if !strings.Contains(md, "# GEMINI.md") {
		t.Error("Markdown should contain header")
	}
	if !strings.Contains(md, "This file provides guidance") {
		t.Error("Markdown should contain description")
	}
}

func TestGeminiConfig_GenerateMarkdown_AllSections(t *testing.T) {
	cfg := &GeminiConfig{
		Instructions: GeminiInstructions{
			ProjectDescription: "My awesome project",
			TechStack:          "Go, TypeScript, PostgreSQL",
			CodeStyle:          "Follow Google style guide",
			CustomRules:        []string{"Always use TDD", "Document public APIs"},
			FileStructure:      "Standard layout with cmd/, internal/, pkg/",
			TestingGuidelines:  "Unit tests required for all packages",
		},
	}

	md := cfg.GenerateMarkdown()

	expectedSections := []string{
		"# GEMINI.md",
		"## Project Description",
		"My awesome project",
		"## Tech Stack",
		"Go, TypeScript, PostgreSQL",
		"## Code Style",
		"Google style guide",
		"## Rules",
		"- Always use TDD",
		"- Document public APIs",
		"## File Structure",
		"Standard layout",
		"## Testing",
		"Unit tests required",
	}

	for _, section := range expectedSections {
		if !strings.Contains(md, section) {
			t.Errorf("Markdown should contain '%s'", section)
		}
	}
}

func TestGeminiConfig_GenerateMarkdown_CustomRulesOnly(t *testing.T) {
	cfg := &GeminiConfig{
		Instructions: GeminiInstructions{
			CustomRules: []string{"Rule 1", "Rule 2", "Rule 3"},
		},
	}

	md := cfg.GenerateMarkdown()

	if !strings.Contains(md, "## Rules") {
		t.Error("Markdown should contain Rules section")
	}
	if !strings.Contains(md, "- Rule 1") {
		t.Error("Markdown should contain Rule 1")
	}
	if !strings.Contains(md, "- Rule 2") {
		t.Error("Markdown should contain Rule 2")
	}
	if !strings.Contains(md, "- Rule 3") {
		t.Error("Markdown should contain Rule 3")
	}
	// Should not contain empty sections
	if strings.Contains(md, "## Project Description") {
		t.Error("Markdown should not contain empty Project Description section")
	}
}

func TestGeminiConfig_GenerateMarkdown_EmptyCustomRules(t *testing.T) {
	cfg := &GeminiConfig{
		Instructions: GeminiInstructions{
			ProjectDescription: "Test project",
			CustomRules:        []string{},
		},
	}

	md := cfg.GenerateMarkdown()

	if strings.Contains(md, "## Rules") {
		t.Error("Markdown should not contain Rules section when empty")
	}
	if !strings.Contains(md, "## Project Description") {
		t.Error("Markdown should contain Project Description section")
	}
}

// === Auth Tests ===

func TestGeminiAuth_APIKey(t *testing.T) {
	auth := GeminiAuth{
		Type: "api_key",
	}

	data, err := json.Marshal(auth)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded GeminiAuth
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Type != "api_key" {
		t.Error("Type mismatch")
	}
}

func TestGeminiAuth_OAuth(t *testing.T) {
	auth := GeminiAuth{
		Type:          "oauth",
		OAuthClientID: "client-123.apps.googleusercontent.com",
	}

	data, err := json.Marshal(auth)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded GeminiAuth
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Type != "oauth" {
		t.Error("Type mismatch")
	}
	if decoded.OAuthClientID != "client-123.apps.googleusercontent.com" {
		t.Error("OAuthClientID mismatch")
	}
}

func TestGeminiAuth_ADC(t *testing.T) {
	auth := GeminiAuth{
		Type:               "adc",
		ServiceAccountPath: "/path/to/service-account.json",
	}

	data, err := json.Marshal(auth)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded GeminiAuth
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Type != "adc" {
		t.Error("Type mismatch")
	}
	if decoded.ServiceAccountPath != "/path/to/service-account.json" {
		t.Error("ServiceAccountPath mismatch")
	}
}

// === Behavior Tests ===

func TestGeminiBehavior_AllFields(t *testing.T) {
	behavior := GeminiBehavior{
		Sandbox:           true,
		AutoApprove:       []string{"file_read", "file_write", "web_search"},
		YoloMode:          true,
		MaxFileSize:       100 * 1024 * 1024,
		AllowedExtensions: []string{".go", ".py", ".js", ".ts"},
	}

	data, err := json.Marshal(behavior)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded GeminiBehavior
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if !decoded.Sandbox {
		t.Error("Sandbox mismatch")
	}
	if !decoded.YoloMode {
		t.Error("YoloMode mismatch")
	}
	if len(decoded.AutoApprove) != 3 {
		t.Error("AutoApprove length mismatch")
	}
	if decoded.MaxFileSize != 100*1024*1024 {
		t.Error("MaxFileSize mismatch")
	}
	if len(decoded.AllowedExtensions) != 4 {
		t.Error("AllowedExtensions length mismatch")
	}
}

func TestGeminiBehavior_YoloMode(t *testing.T) {
	behavior := GeminiBehavior{
		Sandbox:  false,
		YoloMode: true, // Dangerous but valid
	}

	data, err := json.Marshal(behavior)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded GeminiBehavior
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Sandbox {
		t.Error("Sandbox should be false")
	}
	if !decoded.YoloMode {
		t.Error("YoloMode should be true")
	}
}

// === Instructions Tests ===

func TestGeminiInstructions_AllFields(t *testing.T) {
	instructions := GeminiInstructions{
		ProjectDescription: "A microservices platform for AI orchestration",
		TechStack:          "Go 1.22+, gRPC, PostgreSQL 16, Redis 7, Kubernetes",
		CodeStyle:          "Google Go style guide with Uber extensions",
		CustomRules: []string{
			"All public functions must have documentation",
			"Error messages must be actionable",
			"Context must be passed as first parameter",
		},
		FileStructure:     "cmd/, internal/biz/, internal/data/, internal/server/",
		TestingGuidelines: "TDD mandatory, 80% coverage minimum, table-driven tests",
	}

	data, err := json.Marshal(instructions)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded GeminiInstructions
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.ProjectDescription != instructions.ProjectDescription {
		t.Error("ProjectDescription mismatch")
	}
	if decoded.TechStack != instructions.TechStack {
		t.Error("TechStack mismatch")
	}
	if decoded.CodeStyle != instructions.CodeStyle {
		t.Error("CodeStyle mismatch")
	}
	if len(decoded.CustomRules) != 3 {
		t.Error("CustomRules length mismatch")
	}
	if decoded.FileStructure != instructions.FileStructure {
		t.Error("FileStructure mismatch")
	}
	if decoded.TestingGuidelines != instructions.TestingGuidelines {
		t.Error("TestingGuidelines mismatch")
	}
}

func TestGeminiInstructions_EmptyCustomRules(t *testing.T) {
	instructions := GeminiInstructions{
		ProjectDescription: "Test project",
		CustomRules:        []string{},
	}

	data, err := json.Marshal(instructions)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded GeminiInstructions
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Note: In Go, JSON unmarshaling empty array [] results in nil slice
	// This is expected Go behavior, not a bug
	if len(decoded.CustomRules) != 0 {
		t.Error("CustomRules should have zero length after unmarshal")
	}
}

func TestGeminiInstructions_NilCustomRules(t *testing.T) {
	instructions := GeminiInstructions{
		ProjectDescription: "Test project",
		// CustomRules is nil
	}

	data, err := json.Marshal(instructions)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Nil slice should marshal to null, not []
	if strings.Contains(string(data), `"customRules":[]`) {
		t.Error("Nil slice should not marshal to empty array")
	}
}

// === Display Tests ===

func TestGeminiDisplay_AllThemes(t *testing.T) {
	themes := []string{"dark", "light", "auto"}

	for _, theme := range themes {
		t.Run(theme, func(t *testing.T) {
			display := GeminiDisplay{
				Theme:           theme,
				SyntaxHighlight: true,
				MarkdownRender:  true,
			}

			data, err := json.Marshal(display)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			var decoded GeminiDisplay
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			if decoded.Theme != theme {
				t.Errorf("Expected theme '%s', got '%s'", theme, decoded.Theme)
			}
		})
	}
}

func TestGeminiDisplay_AllFields(t *testing.T) {
	display := GeminiDisplay{
		Theme:           "dark",
		SyntaxHighlight: true,
		MarkdownRender:  false,
	}

	data, err := json.Marshal(display)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded GeminiDisplay
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Theme != "dark" {
		t.Error("Theme mismatch")
	}
	if !decoded.SyntaxHighlight {
		t.Error("SyntaxHighlight mismatch")
	}
	if decoded.MarkdownRender {
		t.Error("MarkdownRender should be false")
	}
}

func TestGeminiDisplay_DisabledFeatures(t *testing.T) {
	display := GeminiDisplay{
		Theme:           "light",
		SyntaxHighlight: false,
		MarkdownRender:  false,
	}

	data, err := json.Marshal(display)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded GeminiDisplay
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.SyntaxHighlight {
		t.Error("SyntaxHighlight should be disabled")
	}
	if decoded.MarkdownRender {
		t.Error("MarkdownRender should be disabled")
	}
}
