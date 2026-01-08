package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
)

// Provider represents an LLM provider entity
type Provider struct {
	ID              int64             `json:"id"`
	Name            string            `json:"name"`
	APIURL          string            `json:"api_url"`
	APIKey          string            `json:"api_key"`
	Platform        string            `json:"platform"`
	Enabled         bool              `json:"enabled"`
	Level           int               `json:"level"`
	Site            string            `json:"site"`
	Icon            string            `json:"icon"`
	Tint            string            `json:"tint"`
	Accent          string            `json:"accent"`
	SupportedModels map[string]bool   `json:"supported_models"`
	ModelMapping    map[string]string `json:"model_mapping"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

// ProviderHealth represents provider health check result
type ProviderHealth struct {
	ProviderID   int64     `json:"provider_id"`
	ProviderName string    `json:"provider_name"`
	IsHealthy    bool      `json:"is_healthy"`
	LatencyMs    int64     `json:"latency_ms"`
	ErrorMessage string    `json:"error_message"`
	CheckedAt    time.Time `json:"checked_at"`
}

// MatchedProvider represents a provider matched for a model request
type MatchedProvider struct {
	Provider    *Provider `json:"provider"`
	MappedModel string    `json:"mapped_model"`
	Priority    int       `json:"priority"`
}

// ProviderRepo is the provider repository interface
type ProviderRepo interface {
	Create(ctx context.Context, provider *Provider) (*Provider, error)
	Update(ctx context.Context, provider *Provider) (*Provider, error)
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*Provider, error)
	ListByPlatform(ctx context.Context, platform string, enabledOnly bool) ([]*Provider, error)
	ListAll(ctx context.Context, enabledOnly bool) ([]*Provider, error)
}

// ProviderCache is the provider cache interface
type ProviderCache interface {
	Get(ctx context.Context, key string) (*Provider, error)
	Set(ctx context.Context, key string, provider *Provider, ttl time.Duration) error
	GetList(ctx context.Context, key string) ([]*Provider, error)
	SetList(ctx context.Context, key string, providers []*Provider, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	DeleteByPattern(ctx context.Context, pattern string) error
}

// ProviderUsecase is the provider business logic
type ProviderUsecase struct {
	repo   ProviderRepo
	cache  ProviderCache
	logger *zap.Logger
}

// NewProviderUsecase creates a new provider usecase
func NewProviderUsecase(repo ProviderRepo, cache ProviderCache, logger *zap.Logger) *ProviderUsecase {
	return &ProviderUsecase{
		repo:   repo,
		cache:  cache,
		logger: logger,
	}
}

// Create creates a new provider
func (uc *ProviderUsecase) Create(ctx context.Context, provider *Provider) (*Provider, error) {
	// Validate configuration
	if errs := uc.ValidateConfiguration(provider); len(errs) > 0 {
		return nil, fmt.Errorf("validation failed: %s", strings.Join(errs, "; "))
	}

	// Set defaults
	if provider.Level == 0 {
		provider.Level = 1
	}
	provider.CreatedAt = time.Now()
	provider.UpdatedAt = time.Now()

	// Create in database
	created, err := uc.repo.Create(ctx, provider)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	// Invalidate cache
	uc.invalidateCache(ctx, provider.Platform)

	uc.logger.Info("Provider created",
		zap.Int64("id", created.ID),
		zap.String("name", created.Name),
		zap.String("platform", created.Platform))

	return created, nil
}

// Update updates an existing provider
func (uc *ProviderUsecase) Update(ctx context.Context, provider *Provider) (*Provider, error) {
	// Get existing provider
	existing, err := uc.repo.GetByID(ctx, provider.ID)
	if err != nil {
		return nil, fmt.Errorf("provider not found: %w", err)
	}

	// Name is immutable
	if existing.Name != provider.Name {
		return nil, fmt.Errorf("provider name cannot be changed")
	}

	// Validate configuration
	if errs := uc.ValidateConfiguration(provider); len(errs) > 0 {
		return nil, fmt.Errorf("validation failed: %s", strings.Join(errs, "; "))
	}

	provider.UpdatedAt = time.Now()

	// Update in database
	updated, err := uc.repo.Update(ctx, provider)
	if err != nil {
		return nil, fmt.Errorf("failed to update provider: %w", err)
	}

	// Invalidate cache
	uc.invalidateCache(ctx, provider.Platform)

	uc.logger.Info("Provider updated",
		zap.Int64("id", updated.ID),
		zap.String("name", updated.Name))

	return updated, nil
}

// Delete deletes a provider
func (uc *ProviderUsecase) Delete(ctx context.Context, id int64) error {
	// Get existing provider for cache invalidation
	existing, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("provider not found: %w", err)
	}

	if err := uc.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete provider: %w", err)
	}

	// Invalidate cache
	uc.invalidateCache(ctx, existing.Platform)

	uc.logger.Info("Provider deleted", zap.Int64("id", id))
	return nil
}

// GetByID gets a provider by ID
func (uc *ProviderUsecase) GetByID(ctx context.Context, id int64) (*Provider, error) {
	// Try cache first
	cacheKey := fmt.Sprintf("provider:id:%d", id)
	if cached, err := uc.cache.Get(ctx, cacheKey); err == nil && cached != nil {
		return cached, nil
	}

	// Get from database
	provider, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Cache for 5 minutes
	uc.cache.Set(ctx, cacheKey, provider, 5*time.Minute)

	return provider, nil
}

// ListByPlatform lists providers by platform
func (uc *ProviderUsecase) ListByPlatform(ctx context.Context, platform string, enabledOnly bool) ([]*Provider, error) {
	// Try cache first
	cacheKey := fmt.Sprintf("providers:platform:%s:enabled:%v", platform, enabledOnly)
	if cached, err := uc.cache.GetList(ctx, cacheKey); err == nil && len(cached) > 0 {
		return cached, nil
	}

	// Get from database
	providers, err := uc.repo.ListByPlatform(ctx, platform, enabledOnly)
	if err != nil {
		return nil, err
	}

	// Cache for 5 minutes
	if len(providers) > 0 {
		uc.cache.SetList(ctx, cacheKey, providers, 5*time.Minute)
	}

	return providers, nil
}

// MatchModel finds providers that support a specific model
func (uc *ProviderUsecase) MatchModel(ctx context.Context, platform, model string) ([]*MatchedProvider, error) {
	// Get all enabled providers for the platform
	providers, err := uc.ListByPlatform(ctx, platform, true)
	if err != nil {
		return nil, err
	}

	var matched []*MatchedProvider
	for _, p := range providers {
		if uc.IsModelSupported(p, model) {
			mappedModel := uc.GetEffectiveModel(p, model)
			matched = append(matched, &MatchedProvider{
				Provider:    p,
				MappedModel: mappedModel,
				Priority:    p.Level,
			})
		}
	}

	// Sort by priority (lower level = higher priority)
	for i := 0; i < len(matched); i++ {
		for j := i + 1; j < len(matched); j++ {
			if matched[i].Priority > matched[j].Priority {
				matched[i], matched[j] = matched[j], matched[i]
			}
		}
	}

	return matched, nil
}

// IsModelSupported checks if a provider supports a specific model
func (uc *ProviderUsecase) IsModelSupported(p *Provider, modelName string) bool {
	// Backward compatibility: if no whitelist and no mapping, assume all models supported
	if (p.SupportedModels == nil || len(p.SupportedModels) == 0) &&
		(p.ModelMapping == nil || len(p.ModelMapping) == 0) {
		return true
	}

	// Check supported models (exact match)
	if p.SupportedModels != nil && p.SupportedModels[modelName] {
		return true
	}

	// Check supported models (wildcard match)
	if p.SupportedModels != nil {
		for supportedModel := range p.SupportedModels {
			if matchWildcard(supportedModel, modelName) {
				return true
			}
		}
	}

	// Check model mapping (exact match)
	if p.ModelMapping != nil {
		if _, exists := p.ModelMapping[modelName]; exists {
			return true
		}

		// Check model mapping (wildcard match)
		for pattern := range p.ModelMapping {
			if matchWildcard(pattern, modelName) {
				return true
			}
		}
	}

	return false
}

// GetEffectiveModel returns the actual model name after mapping
func (uc *ProviderUsecase) GetEffectiveModel(p *Provider, requestedModel string) string {
	if p.ModelMapping == nil || len(p.ModelMapping) == 0 {
		return requestedModel
	}

	// Exact mapping first
	if mappedModel, exists := p.ModelMapping[requestedModel]; exists {
		return mappedModel
	}

	// Wildcard mapping
	for pattern, replacement := range p.ModelMapping {
		if matchWildcard(pattern, requestedModel) {
			return applyWildcardMapping(pattern, replacement, requestedModel)
		}
	}

	return requestedModel
}

// ValidateConfiguration validates provider configuration
func (uc *ProviderUsecase) ValidateConfiguration(p *Provider) []string {
	var errors []string

	// Rule 1: ModelMapping values must be in SupportedModels
	if p.ModelMapping != nil && p.SupportedModels != nil {
		for externalModel, internalModel := range p.ModelMapping {
			if strings.Contains(internalModel, "*") {
				continue // Skip wildcard mappings
			}

			supported := false
			if p.SupportedModels[internalModel] {
				supported = true
			} else {
				for supportedPattern := range p.SupportedModels {
					if matchWildcard(supportedPattern, internalModel) {
						supported = true
						break
					}
				}
			}

			if !supported {
				errors = append(errors, fmt.Sprintf(
					"invalid model mapping: '%s' -> '%s', target model '%s' not in supportedModels",
					externalModel, internalModel, internalModel,
				))
			}
		}
	}

	// Rule 2: Warning if ModelMapping without SupportedModels
	if p.ModelMapping != nil && len(p.ModelMapping) > 0 &&
		(p.SupportedModels == nil || len(p.SupportedModels) == 0) {
		errors = append(errors,
			"warning: modelMapping configured without supportedModels, target models cannot be validated",
		)
	}

	return errors
}

// invalidateCache invalidates cache for a platform
func (uc *ProviderUsecase) invalidateCache(ctx context.Context, platform string) {
	patterns := []string{
		fmt.Sprintf("providers:platform:%s:*", platform),
		"providers:platform::*", // All platforms cache
	}
	for _, pattern := range patterns {
		uc.cache.DeleteByPattern(ctx, pattern)
	}
}

// matchWildcard performs wildcard pattern matching
func matchWildcard(pattern, text string) bool {
	if !strings.Contains(pattern, "*") {
		return pattern == text
	}

	parts := strings.Split(pattern, "*")
	if len(parts) == 2 {
		prefix, suffix := parts[0], parts[1]
		return strings.HasPrefix(text, prefix) && strings.HasSuffix(text, suffix)
	}

	return false
}

// applyWildcardMapping applies wildcard mapping
func applyWildcardMapping(pattern, replacement, input string) string {
	if !strings.Contains(pattern, "*") || !strings.Contains(replacement, "*") {
		return replacement
	}

	parts := strings.Split(pattern, "*")
	if len(parts) != 2 {
		return replacement
	}

	prefix, suffix := parts[0], parts[1]
	if !strings.HasPrefix(input, prefix) || !strings.HasSuffix(input, suffix) {
		return replacement
	}

	wildcardPart := input[len(prefix) : len(input)-len(suffix)]
	return strings.Replace(replacement, "*", wildcardPart, 1)
}

// SerializeSupportedModels serializes supported models to JSON
func SerializeSupportedModels(models map[string]bool) string {
	if models == nil {
		return "{}"
	}
	data, _ := json.Marshal(models)
	return string(data)
}

// DeserializeSupportedModels deserializes supported models from JSON
func DeserializeSupportedModels(data string) map[string]bool {
	var models map[string]bool
	json.Unmarshal([]byte(data), &models)
	return models
}

// SerializeModelMapping serializes model mapping to JSON
func SerializeModelMapping(mapping map[string]string) string {
	if mapping == nil {
		return "{}"
	}
	data, _ := json.Marshal(mapping)
	return string(data)
}

// DeserializeModelMapping deserializes model mapping from JSON
func DeserializeModelMapping(data string) map[string]string {
	var mapping map[string]string
	json.Unmarshal([]byte(data), &mapping)
	return mapping
}
