package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestProvider_JSON(t *testing.T) {
	provider := &Provider{
		ID:        1,
		Name:      "Test Provider",
		APIURL:    "https://api.example.com",
		APIKey:    "test-key",
		Platform:  PlatformClaude,
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		SupportedModels: map[string]bool{
			"claude-3-opus":   true,
			"claude-3-sonnet": true,
		},
		ModelMapping: map[string]string{
			"claude-*": "anthropic/claude-*",
		},
	}

	// Test JSON marshalling
	data, err := json.Marshal(provider)
	if err != nil {
		t.Fatalf("Failed to marshal provider: %v", err)
	}

	// Test JSON unmarshalling
	var decoded Provider
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal provider: %v", err)
	}

	if decoded.ID != provider.ID {
		t.Errorf("ID mismatch: got %d, want %d", decoded.ID, provider.ID)
	}
	if decoded.Name != provider.Name {
		t.Errorf("Name mismatch: got %s, want %s", decoded.Name, provider.Name)
	}
	if decoded.Platform != provider.Platform {
		t.Errorf("Platform mismatch: got %s, want %s", decoded.Platform, provider.Platform)
	}
	if decoded.Enabled != provider.Enabled {
		t.Errorf("Enabled mismatch: got %v, want %v", decoded.Enabled, provider.Enabled)
	}
}

func TestProvider_SupportedModels(t *testing.T) {
	provider := &Provider{
		SupportedModels: map[string]bool{
			"claude-3-opus":   true,
			"claude-3-sonnet": true,
			"claude-3-haiku":  false,
		},
	}

	// Test supported model lookup
	if !provider.SupportedModels["claude-3-opus"] {
		t.Error("Expected claude-3-opus to be supported")
	}
	if !provider.SupportedModels["claude-3-sonnet"] {
		t.Error("Expected claude-3-sonnet to be supported")
	}
	if provider.SupportedModels["claude-3-haiku"] {
		t.Error("Expected claude-3-haiku to not be supported")
	}
	if provider.SupportedModels["nonexistent"] {
		t.Error("Expected nonexistent model to not be supported")
	}
}

func TestProviderHealth(t *testing.T) {
	health := &ProviderHealth{
		ProviderID:   1,
		ProviderName: "Test Provider",
		IsHealthy:    true,
		Latency:      100,
		LastCheck:    time.Now(),
	}

	data, err := json.Marshal(health)
	if err != nil {
		t.Fatalf("Failed to marshal health: %v", err)
	}

	var decoded ProviderHealth
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal health: %v", err)
	}

	if decoded.ProviderID != health.ProviderID {
		t.Errorf("ProviderID mismatch: got %d, want %d", decoded.ProviderID, health.ProviderID)
	}
	if decoded.IsHealthy != health.IsHealthy {
		t.Errorf("IsHealthy mismatch: got %v, want %v", decoded.IsHealthy, health.IsHealthy)
	}
}

func TestProviderStats(t *testing.T) {
	stats := &ProviderStats{
		ProviderID:    1,
		ProviderName:  "Test Provider",
		TotalRequests: 1000,
		SuccessRate:   0.99,
		AvgLatency:    150.5,
		TotalCost:     25.50,
		TotalTokens:   500000,
	}

	data, err := json.Marshal(stats)
	if err != nil {
		t.Fatalf("Failed to marshal stats: %v", err)
	}

	var decoded ProviderStats
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal stats: %v", err)
	}

	if decoded.TotalRequests != stats.TotalRequests {
		t.Errorf("TotalRequests mismatch: got %d, want %d", decoded.TotalRequests, stats.TotalRequests)
	}
	if decoded.SuccessRate != stats.SuccessRate {
		t.Errorf("SuccessRate mismatch: got %f, want %f", decoded.SuccessRate, stats.SuccessRate)
	}
}

func TestPlatformConstants(t *testing.T) {
	if PlatformClaude != "claude" {
		t.Errorf("PlatformClaude should be 'claude', got '%s'", PlatformClaude)
	}
	if PlatformCodex != "codex" {
		t.Errorf("PlatformCodex should be 'codex', got '%s'", PlatformCodex)
	}
	if PlatformGemini != "gemini" {
		t.Errorf("PlatformGemini should be 'gemini', got '%s'", PlatformGemini)
	}
}
