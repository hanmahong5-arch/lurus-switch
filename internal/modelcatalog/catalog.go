package modelcatalog

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Model represents a single LLM model available through the gateway.
type Model struct {
	ID          string   `json:"id"`
	DisplayName string   `json:"displayName"`
	Provider    string   `json:"provider"`
	InputRatio  float64  `json:"inputRatio"`
	OutputRatio float64  `json:"outputRatio"`
	Tags        []string `json:"tags"`
	Recommended bool     `json:"recommended"`
}

// Catalog holds a list of available models and metadata.
type Catalog struct {
	Models    []Model   `json:"models"`
	FetchedAt time.Time `json:"fetchedAt"`
}

// Manager handles fetching, caching, and querying the model catalog.
type Manager struct {
	mu       sync.RWMutex
	cacheDir string
	catalog  *Catalog
}

const (
	cacheFile = "model-catalog.json"
	cacheTTL  = 1 * time.Hour
	fetchTimeout = 10 * time.Second
)

// NewManager creates a model catalog manager with the given cache directory.
func NewManager(appDataDir string) *Manager {
	return &Manager{cacheDir: appDataDir}
}

// Fetch retrieves the model catalog from the gateway API, with disk cache and offline fallback.
func (m *Manager) Fetch(ctx context.Context, apiBase, apiKey string) (*Catalog, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Return memory cache if fresh
	if m.catalog != nil && time.Since(m.catalog.FetchedAt) < cacheTTL {
		return m.catalog, nil
	}

	// Try disk cache first
	if cached, err := m.loadCache(); err == nil && time.Since(cached.FetchedAt) < cacheTTL {
		m.catalog = cached
		return cached, nil
	}

	// Fetch from API
	if apiBase != "" {
		if cat, err := m.fetchFromAPI(ctx, apiBase, apiKey); err == nil {
			m.catalog = cat
			_ = m.saveCache(cat)
			return cat, nil
		}
	}

	// Fallback to stale cache
	if cached, err := m.loadCache(); err == nil {
		m.catalog = cached
		return cached, nil
	}

	// Final fallback: builtin defaults
	cat := DefaultCatalog()
	m.catalog = cat
	return cat, nil
}

// GetCatalog returns the current catalog (from memory, cache, or defaults).
func (m *Manager) GetCatalog() *Catalog {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.catalog != nil {
		return m.catalog
	}
	return DefaultCatalog()
}

// fetchFromAPI calls GET /api/pricing on the gateway to get model data.
func (m *Manager) fetchFromAPI(ctx context.Context, apiBase, apiKey string) (*Catalog, error) {
	reqCtx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()

	url := apiBase + "/api/pricing"
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build pricing request: %w", err)
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch pricing: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("pricing API returned %d", resp.StatusCode)
	}

	// Parse the pricing API response — array of model objects
	var rawModels []struct {
		ModelName     string  `json:"model_name"`
		ModelRatio    float64 `json:"model_ratio"`
		ModelRatio2   float64 `json:"model_ratio_2"`   // output ratio
		CompletionRatio float64 `json:"completion_ratio"` // alternative output ratio
		GroupRatio    float64 `json:"group_ratio"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rawModels); err != nil {
		return nil, fmt.Errorf("parse pricing response: %w", err)
	}

	models := make([]Model, 0, len(rawModels))
	for _, rm := range rawModels {
		outputRatio := rm.ModelRatio2
		if outputRatio == 0 {
			outputRatio = rm.CompletionRatio
		}
		m := Model{
			ID:          rm.ModelName,
			DisplayName: rm.ModelName,
			Provider:    inferProvider(rm.ModelName),
			InputRatio:  rm.ModelRatio,
			OutputRatio: outputRatio,
			Tags:        inferTags(rm.ModelName),
			Recommended: isRecommended(rm.ModelName),
		}
		models = append(models, m)
	}

	cat := &Catalog{
		Models:    models,
		FetchedAt: time.Now(),
	}
	return cat, nil
}

func (m *Manager) loadCache() (*Catalog, error) {
	path := filepath.Join(m.cacheDir, cacheFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cat Catalog
	if err := json.Unmarshal(data, &cat); err != nil {
		return nil, err
	}
	return &cat, nil
}

func (m *Manager) saveCache(cat *Catalog) error {
	if err := os.MkdirAll(m.cacheDir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cat, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(m.cacheDir, cacheFile), data, 0o600)
}

// DefaultCatalog returns a hardcoded catalog of popular domestic + international models.
func DefaultCatalog() *Catalog {
	return &Catalog{
		FetchedAt: time.Now(),
		Models: []Model{
			{ID: "deepseek-chat", DisplayName: "DeepSeek V3", Provider: "DeepSeek", InputRatio: 0.07, OutputRatio: 0.14, Tags: []string{"domestic", "fast", "cheap", "coding"}, Recommended: true},
			{ID: "deepseek-reasoner", DisplayName: "DeepSeek R1", Provider: "DeepSeek", InputRatio: 0.14, OutputRatio: 0.28, Tags: []string{"domestic", "reasoning"}, Recommended: true},
			{ID: "qwen-plus", DisplayName: "Qwen Plus", Provider: "Alibaba", InputRatio: 0.14, OutputRatio: 0.28, Tags: []string{"domestic", "fast", "cheap"}, Recommended: false},
			{ID: "qwen-max", DisplayName: "Qwen Max", Provider: "Alibaba", InputRatio: 0.56, OutputRatio: 1.12, Tags: []string{"domestic", "reasoning"}, Recommended: false},
			{ID: "glm-4-plus", DisplayName: "GLM-4 Plus", Provider: "Zhipu", InputRatio: 0.28, OutputRatio: 0.56, Tags: []string{"domestic", "fast"}, Recommended: false},
			{ID: "claude-sonnet-4-20250514", DisplayName: "Claude Sonnet 4", Provider: "Anthropic", InputRatio: 1.5, OutputRatio: 7.5, Tags: []string{"international", "quality", "coding"}, Recommended: true},
			{ID: "gpt-4o", DisplayName: "GPT-4o", Provider: "OpenAI", InputRatio: 1.25, OutputRatio: 5.0, Tags: []string{"international", "fast"}, Recommended: false},
			{ID: "gpt-4o-mini", DisplayName: "GPT-4o Mini", Provider: "OpenAI", InputRatio: 0.075, OutputRatio: 0.3, Tags: []string{"international", "fast", "cheap"}, Recommended: false},
		},
	}
}

// inferProvider guesses the provider from a model ID.
func inferProvider(modelID string) string {
	switch {
	case contains(modelID, "deepseek"):
		return "DeepSeek"
	case contains(modelID, "qwen"):
		return "Alibaba"
	case contains(modelID, "glm"):
		return "Zhipu"
	case contains(modelID, "claude"):
		return "Anthropic"
	case contains(modelID, "gpt"):
		return "OpenAI"
	case contains(modelID, "gemini"):
		return "Google"
	case contains(modelID, "moonshot"), contains(modelID, "kimi"):
		return "Moonshot"
	case contains(modelID, "yi-"):
		return "01.AI"
	default:
		return "Other"
	}
}

// inferTags generates tags from a model ID.
func inferTags(modelID string) []string {
	tags := []string{}
	switch {
	case contains(modelID, "deepseek"), contains(modelID, "qwen"), contains(modelID, "glm"),
		contains(modelID, "moonshot"), contains(modelID, "kimi"), contains(modelID, "yi-"):
		tags = append(tags, "domestic")
	default:
		tags = append(tags, "international")
	}
	return tags
}

func isRecommended(modelID string) bool {
	switch modelID {
	case "deepseek-chat", "deepseek-reasoner", "claude-sonnet-4-20250514":
		return true
	default:
		return false
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsLower(s, substr))
}

func containsLower(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
