package biz

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"
)

// MockProviderRepo is a mock implementation of ProviderRepo
type MockProviderRepo struct {
	providers map[int64]*Provider
	nextID    int64
}

func NewMockProviderRepo() *MockProviderRepo {
	return &MockProviderRepo{
		providers: make(map[int64]*Provider),
		nextID:    1,
	}
}

func (m *MockProviderRepo) Create(ctx context.Context, provider *Provider) (*Provider, error) {
	provider.ID = m.nextID
	m.nextID++
	// Store a copy to prevent mutation
	copy := *provider
	m.providers[provider.ID] = &copy
	return provider, nil
}

func (m *MockProviderRepo) Update(ctx context.Context, provider *Provider) (*Provider, error) {
	if _, ok := m.providers[provider.ID]; !ok {
		return nil, errors.New("provider not found")
	}
	m.providers[provider.ID] = provider
	return provider, nil
}

func (m *MockProviderRepo) Delete(ctx context.Context, id int64) error {
	if _, ok := m.providers[id]; !ok {
		return errors.New("provider not found")
	}
	delete(m.providers, id)
	return nil
}

func (m *MockProviderRepo) GetByID(ctx context.Context, id int64) (*Provider, error) {
	p, ok := m.providers[id]
	if !ok {
		return nil, errors.New("provider not found")
	}
	return p, nil
}

func (m *MockProviderRepo) ListByPlatform(ctx context.Context, platform string, enabledOnly bool) ([]*Provider, error) {
	var result []*Provider
	for _, p := range m.providers {
		if p.Platform == platform && (!enabledOnly || p.Enabled) {
			result = append(result, p)
		}
	}
	return result, nil
}

func (m *MockProviderRepo) ListAll(ctx context.Context, enabledOnly bool) ([]*Provider, error) {
	var result []*Provider
	for _, p := range m.providers {
		if !enabledOnly || p.Enabled {
			result = append(result, p)
		}
	}
	return result, nil
}

// MockProviderCache is a mock implementation of ProviderCache
type MockProviderCache struct {
	single map[string]*Provider
	list   map[string][]*Provider
}

func NewMockProviderCache() *MockProviderCache {
	return &MockProviderCache{
		single: make(map[string]*Provider),
		list:   make(map[string][]*Provider),
	}
}

func (m *MockProviderCache) Get(ctx context.Context, key string) (*Provider, error) {
	p, ok := m.single[key]
	if !ok {
		return nil, errors.New("not found")
	}
	return p, nil
}

func (m *MockProviderCache) Set(ctx context.Context, key string, provider *Provider, ttl time.Duration) error {
	m.single[key] = provider
	return nil
}

func (m *MockProviderCache) GetList(ctx context.Context, key string) ([]*Provider, error) {
	list, ok := m.list[key]
	if !ok {
		return nil, errors.New("not found")
	}
	return list, nil
}

func (m *MockProviderCache) SetList(ctx context.Context, key string, providers []*Provider, ttl time.Duration) error {
	m.list[key] = providers
	return nil
}

func (m *MockProviderCache) Delete(ctx context.Context, key string) error {
	delete(m.single, key)
	delete(m.list, key)
	return nil
}

func (m *MockProviderCache) DeleteByPattern(ctx context.Context, pattern string) error {
	// Simple pattern matching for tests
	for k := range m.single {
		delete(m.single, k)
	}
	for k := range m.list {
		delete(m.list, k)
	}
	return nil
}

func TestProviderUsecase_Create(t *testing.T) {
	repo := NewMockProviderRepo()
	cache := NewMockProviderCache()
	logger := zap.NewNop()
	uc := NewProviderUsecase(repo, cache, logger)

	ctx := context.Background()
	provider := &Provider{
		Name:     "Test Provider",
		APIURL:   "https://api.example.com",
		APIKey:   "test-key",
		Platform: "claude",
		Enabled:  true,
	}

	created, err := uc.Create(ctx, provider)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if created.ID != 1 {
		t.Errorf("Expected ID 1, got %d", created.ID)
	}
	if created.Level != 1 {
		t.Errorf("Expected default level 1, got %d", created.Level)
	}
	if created.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}
}

func TestProviderUsecase_Update(t *testing.T) {
	repo := NewMockProviderRepo()
	cache := NewMockProviderCache()
	logger := zap.NewNop()
	uc := NewProviderUsecase(repo, cache, logger)

	ctx := context.Background()
	provider := &Provider{
		Name:     "Test Provider",
		APIURL:   "https://api.example.com",
		Platform: "claude",
		Enabled:  true,
	}

	created, _ := uc.Create(ctx, provider)

	// Update the provider
	created.APIURL = "https://api2.example.com"
	created.Enabled = false

	updated, err := uc.Update(ctx, created)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if updated.APIURL != "https://api2.example.com" {
		t.Errorf("Expected updated API URL")
	}
	if updated.Enabled {
		t.Error("Expected Enabled to be false")
	}
}

func TestProviderUsecase_Update_NameImmutable(t *testing.T) {
	repo := NewMockProviderRepo()
	cache := NewMockProviderCache()
	logger := zap.NewNop()
	uc := NewProviderUsecase(repo, cache, logger)

	ctx := context.Background()
	provider := &Provider{
		Name:     "Test Provider",
		APIURL:   "https://api.example.com",
		Platform: "claude",
		Enabled:  true,
	}

	created, _ := uc.Create(ctx, provider)
	created.Name = "New Name"

	_, err := uc.Update(ctx, created)
	if err == nil {
		t.Error("Expected error when changing name")
	}
}

func TestProviderUsecase_Delete(t *testing.T) {
	repo := NewMockProviderRepo()
	cache := NewMockProviderCache()
	logger := zap.NewNop()
	uc := NewProviderUsecase(repo, cache, logger)

	ctx := context.Background()
	provider := &Provider{
		Name:     "Test Provider",
		APIURL:   "https://api.example.com",
		Platform: "claude",
		Enabled:  true,
	}

	created, _ := uc.Create(ctx, provider)

	err := uc.Delete(ctx, created.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deletion
	_, err = uc.GetByID(ctx, created.ID)
	if err == nil {
		t.Error("Expected error when getting deleted provider")
	}
}

func TestProviderUsecase_ListByPlatform(t *testing.T) {
	repo := NewMockProviderRepo()
	cache := NewMockProviderCache()
	logger := zap.NewNop()
	uc := NewProviderUsecase(repo, cache, logger)

	ctx := context.Background()

	// Create providers for different platforms
	uc.Create(ctx, &Provider{Name: "Claude1", Platform: "claude", Enabled: true})
	uc.Create(ctx, &Provider{Name: "Claude2", Platform: "claude", Enabled: true})
	uc.Create(ctx, &Provider{Name: "Codex1", Platform: "codex", Enabled: true})
	uc.Create(ctx, &Provider{Name: "Claude3", Platform: "claude", Enabled: false})

	// List Claude providers (enabled only)
	providers, err := uc.ListByPlatform(ctx, "claude", true)
	if err != nil {
		t.Fatalf("ListByPlatform failed: %v", err)
	}

	if len(providers) != 2 {
		t.Errorf("Expected 2 enabled Claude providers, got %d", len(providers))
	}

	// List all Claude providers
	providers, err = uc.ListByPlatform(ctx, "claude", false)
	if err != nil {
		t.Fatalf("ListByPlatform failed: %v", err)
	}

	if len(providers) != 3 {
		t.Errorf("Expected 3 Claude providers, got %d", len(providers))
	}
}

func TestProviderUsecase_IsModelSupported(t *testing.T) {
	uc := NewProviderUsecase(nil, nil, nil)

	tests := []struct {
		name     string
		provider *Provider
		model    string
		expected bool
	}{
		{
			name: "No whitelist - all supported",
			provider: &Provider{
				SupportedModels: nil,
				ModelMapping:    nil,
			},
			model:    "any-model",
			expected: true,
		},
		{
			name: "Exact match in SupportedModels",
			provider: &Provider{
				SupportedModels: map[string]bool{
					"claude-3-opus":   true,
					"claude-3-sonnet": true,
				},
			},
			model:    "claude-3-opus",
			expected: true,
		},
		{
			name: "No match in SupportedModels",
			provider: &Provider{
				SupportedModels: map[string]bool{
					"claude-3-opus": true,
				},
			},
			model:    "claude-3-sonnet",
			expected: false,
		},
		{
			name: "Wildcard match in SupportedModels",
			provider: &Provider{
				SupportedModels: map[string]bool{
					"claude-*": true,
				},
			},
			model:    "claude-3-opus",
			expected: true,
		},
		{
			name: "Exact match in ModelMapping",
			provider: &Provider{
				ModelMapping: map[string]string{
					"gpt-4": "openai/gpt-4",
				},
			},
			model:    "gpt-4",
			expected: true,
		},
		{
			name: "Wildcard match in ModelMapping",
			provider: &Provider{
				ModelMapping: map[string]string{
					"claude-*": "anthropic/claude-*",
				},
			},
			model:    "claude-3-opus",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := uc.IsModelSupported(tt.provider, tt.model)
			if result != tt.expected {
				t.Errorf("IsModelSupported(%s) = %v, want %v", tt.model, result, tt.expected)
			}
		})
	}
}

func TestProviderUsecase_GetEffectiveModel(t *testing.T) {
	uc := NewProviderUsecase(nil, nil, nil)

	tests := []struct {
		name     string
		provider *Provider
		model    string
		expected string
	}{
		{
			name:     "No mapping",
			provider: &Provider{ModelMapping: nil},
			model:    "claude-3-opus",
			expected: "claude-3-opus",
		},
		{
			name: "Exact mapping",
			provider: &Provider{
				ModelMapping: map[string]string{
					"gpt-4": "openai/gpt-4",
				},
			},
			model:    "gpt-4",
			expected: "openai/gpt-4",
		},
		{
			name: "Wildcard mapping",
			provider: &Provider{
				ModelMapping: map[string]string{
					"claude-*": "anthropic/claude-*",
				},
			},
			model:    "claude-3-opus",
			expected: "anthropic/claude-3-opus",
		},
		{
			name: "No match - return original",
			provider: &Provider{
				ModelMapping: map[string]string{
					"gpt-*": "openai/gpt-*",
				},
			},
			model:    "claude-3-opus",
			expected: "claude-3-opus",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := uc.GetEffectiveModel(tt.provider, tt.model)
			if result != tt.expected {
				t.Errorf("GetEffectiveModel(%s) = %s, want %s", tt.model, result, tt.expected)
			}
		})
	}
}

func TestProviderUsecase_MatchModel(t *testing.T) {
	repo := NewMockProviderRepo()
	cache := NewMockProviderCache()
	logger := zap.NewNop()
	uc := NewProviderUsecase(repo, cache, logger)

	ctx := context.Background()

	// Create providers
	uc.Create(ctx, &Provider{
		Name:     "Provider1",
		Platform: "claude",
		Enabled:  true,
		Level:    2,
		SupportedModels: map[string]bool{
			"claude-3-opus": true,
		},
	})
	uc.Create(ctx, &Provider{
		Name:     "Provider2",
		Platform: "claude",
		Enabled:  true,
		Level:    1,
		SupportedModels: map[string]bool{
			"claude-*": true,
		},
	})

	matched, err := uc.MatchModel(ctx, "claude", "claude-3-opus")
	if err != nil {
		t.Fatalf("MatchModel failed: %v", err)
	}

	if len(matched) != 2 {
		t.Errorf("Expected 2 matched providers, got %d", len(matched))
	}

	// Should be sorted by priority (level 1 first)
	if matched[0].Priority != 1 {
		t.Errorf("Expected first match to have priority 1, got %d", matched[0].Priority)
	}
}

func TestProviderUsecase_ValidateConfiguration(t *testing.T) {
	uc := NewProviderUsecase(nil, nil, nil)

	tests := []struct {
		name      string
		provider  *Provider
		hasErrors bool
	}{
		{
			name: "Valid configuration",
			provider: &Provider{
				SupportedModels: map[string]bool{"claude-3-opus": true},
				ModelMapping: map[string]string{
					"opus": "claude-3-opus",
				},
			},
			hasErrors: false,
		},
		{
			name: "Invalid mapping - target not in supported",
			provider: &Provider{
				SupportedModels: map[string]bool{"claude-3-opus": true},
				ModelMapping: map[string]string{
					"sonnet": "claude-3-sonnet", // Not in supported models
				},
			},
			hasErrors: true,
		},
		{
			name: "Warning - mapping without supported models",
			provider: &Provider{
				SupportedModels: nil,
				ModelMapping: map[string]string{
					"opus": "claude-3-opus",
				},
			},
			hasErrors: true, // Warning is treated as error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := uc.ValidateConfiguration(tt.provider)
			hasErrors := len(errs) > 0
			if hasErrors != tt.hasErrors {
				t.Errorf("ValidateConfiguration hasErrors = %v, want %v, errors: %v", hasErrors, tt.hasErrors, errs)
			}
		})
	}
}

func TestMatchWildcard(t *testing.T) {
	tests := []struct {
		pattern  string
		text     string
		expected bool
	}{
		{"claude-*", "claude-3-opus", true},
		{"claude-*", "gpt-4", false},
		{"*-opus", "claude-3-opus", true},
		{"claude-*-opus", "claude-3-opus", true},
		{"claude", "claude", true},
		{"claude", "claude-3", false},
	}

	for _, tt := range tests {
		result := matchWildcard(tt.pattern, tt.text)
		if result != tt.expected {
			t.Errorf("matchWildcard(%s, %s) = %v, want %v", tt.pattern, tt.text, result, tt.expected)
		}
	}
}

func TestApplyWildcardMapping(t *testing.T) {
	tests := []struct {
		pattern     string
		replacement string
		input       string
		expected    string
	}{
		{"claude-*", "anthropic/claude-*", "claude-3-opus", "anthropic/claude-3-opus"},
		{"gpt-*", "openai/gpt-*", "gpt-4-turbo", "openai/gpt-4-turbo"},
		{"exact", "mapped", "exact", "mapped"},
	}

	for _, tt := range tests {
		result := applyWildcardMapping(tt.pattern, tt.replacement, tt.input)
		if result != tt.expected {
			t.Errorf("applyWildcardMapping(%s, %s, %s) = %s, want %s",
				tt.pattern, tt.replacement, tt.input, result, tt.expected)
		}
	}
}

func TestSerializeDeserializeSupportedModels(t *testing.T) {
	models := map[string]bool{
		"claude-3-opus":   true,
		"claude-3-sonnet": true,
	}

	serialized := SerializeSupportedModels(models)
	deserialized := DeserializeSupportedModels(serialized)

	if len(deserialized) != len(models) {
		t.Errorf("Expected %d models, got %d", len(models), len(deserialized))
	}

	for k, v := range models {
		if deserialized[k] != v {
			t.Errorf("Model %s: expected %v, got %v", k, v, deserialized[k])
		}
	}
}

func TestSerializeDeserializeModelMapping(t *testing.T) {
	mapping := map[string]string{
		"opus":   "claude-3-opus",
		"sonnet": "claude-3-sonnet",
	}

	serialized := SerializeModelMapping(mapping)
	deserialized := DeserializeModelMapping(serialized)

	if len(deserialized) != len(mapping) {
		t.Errorf("Expected %d mappings, got %d", len(mapping), len(deserialized))
	}

	for k, v := range mapping {
		if deserialized[k] != v {
			t.Errorf("Mapping %s: expected %s, got %s", k, v, deserialized[k])
		}
	}
}
