package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pocketzworld/lurus-common/models"
	"github.com/pocketzworld/lurus-switch/gateway-service/internal/conf"
	"go.uber.org/zap"
)

func TestProviderClient_GetProviders(t *testing.T) {
	// Setup mock server
	providers := []*models.Provider{
		{ID: 1, Name: "Provider1", Platform: "claude", Enabled: true},
		{ID: 2, Name: "Provider2", Platform: "claude", Enabled: true},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/providers" {
			t.Errorf("Expected path /api/v1/providers, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("platform") != "claude" {
			t.Errorf("Expected platform=claude, got %s", r.URL.Query().Get("platform"))
		}

		response := struct {
			Providers []*models.Provider `json:"providers"`
		}{Providers: providers}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client
	config := &conf.Provider{
		Endpoint: server.URL,
		Timeout:  5 * time.Second,
		CacheTTL: 5 * time.Minute,
	}
	logger := zap.NewNop()
	client := NewProviderClient(config, logger)

	// Test GetProviders
	ctx := context.Background()
	result, err := client.GetProviders(ctx, "claude")
	if err != nil {
		t.Fatalf("GetProviders failed: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 providers, got %d", len(result))
	}
	if result[0].Name != "Provider1" {
		t.Errorf("Expected Provider1, got %s", result[0].Name)
	}
}

func TestProviderClient_GetProviders_Cached(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		response := struct {
			Providers []*models.Provider `json:"providers"`
		}{Providers: []*models.Provider{{ID: 1, Name: "Provider1"}}}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &conf.Provider{
		Endpoint: server.URL,
		Timeout:  5 * time.Second,
		CacheTTL: 5 * time.Minute,
	}
	logger := zap.NewNop()
	client := NewProviderClient(config, logger)

	ctx := context.Background()

	// First call - should hit server
	_, err := client.GetProviders(ctx, "claude")
	if err != nil {
		t.Fatalf("First GetProviders failed: %v", err)
	}

	// Second call - should use cache
	_, err = client.GetProviders(ctx, "claude")
	if err != nil {
		t.Fatalf("Second GetProviders failed: %v", err)
	}

	if callCount != 1 {
		t.Errorf("Expected 1 server call (cached), got %d", callCount)
	}
}

func TestProviderClient_MatchModel(t *testing.T) {
	providers := []*models.Provider{
		{
			ID:       1,
			Name:     "Anthropic",
			Platform: "claude",
			APIURL:   "https://api.anthropic.com",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/providers/match" {
			t.Errorf("Expected path /api/v1/providers/match, got %s", r.URL.Path)
		}

		// Return array format like provider-service does
		response := struct {
			Providers []*models.Provider `json:"providers"`
		}{Providers: providers}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	config := &conf.Provider{
		Endpoint: server.URL,
		Timeout:  5 * time.Second,
		CacheTTL: 5 * time.Minute,
	}
	logger := zap.NewNop()
	client := NewProviderClient(config, logger)

	ctx := context.Background()
	result, err := client.MatchModel(ctx, "claude", "claude-3-opus")
	if err != nil {
		t.Fatalf("MatchModel failed: %v", err)
	}

	if result.Name != "Anthropic" {
		t.Errorf("Expected Anthropic, got %s", result.Name)
	}
}

func TestProviderClient_MatchModel_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	config := &conf.Provider{
		Endpoint: server.URL,
		Timeout:  5 * time.Second,
		CacheTTL: 5 * time.Minute,
	}
	logger := zap.NewNop()
	client := NewProviderClient(config, logger)

	ctx := context.Background()
	_, err := client.MatchModel(ctx, "claude", "unknown-model")
	if err == nil {
		t.Error("Expected error for unknown model")
	}
}

func TestProviderCache(t *testing.T) {
	cache := &providerCache{
		data:     make(map[string][]*models.Provider),
		expireAt: make(map[string]time.Time),
		ttl:      100 * time.Millisecond,
	}

	// Test set and get
	providers := []*models.Provider{{ID: 1, Name: "P1"}}
	cache.set("claude", providers)

	result := cache.get("claude")
	if result == nil {
		t.Fatal("Expected cached providers")
	}
	if len(result) != 1 {
		t.Errorf("Expected 1 provider, got %d", len(result))
	}

	// Test cache expiration
	time.Sleep(150 * time.Millisecond)
	result = cache.get("claude")
	if result != nil {
		t.Error("Expected nil after cache expiration")
	}
}

func TestProviderCache_Invalidate(t *testing.T) {
	cache := &providerCache{
		data:     make(map[string][]*models.Provider),
		expireAt: make(map[string]time.Time),
		ttl:      5 * time.Minute,
	}

	providers := []*models.Provider{{ID: 1, Name: "P1"}}
	cache.set("claude", providers)

	// Verify cache exists
	if cache.get("claude") == nil {
		t.Fatal("Expected cached providers before invalidate")
	}

	// Invalidate
	cache.invalidate("claude")

	// Verify cache is gone
	if cache.get("claude") != nil {
		t.Error("Expected nil after invalidate")
	}
}
