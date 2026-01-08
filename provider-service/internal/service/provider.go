package service

import (
	"context"
	"time"

	"github.com/pocketzworld/lurus-switch/provider-service/internal/biz"
	"go.uber.org/zap"
)

// ProviderService is the provider service implementation
type ProviderService struct {
	uc     *biz.ProviderUsecase
	logger *zap.Logger
}

// NewProviderService creates a new provider service
func NewProviderService(uc *biz.ProviderUsecase, logger *zap.Logger) *ProviderService {
	return &ProviderService{
		uc:     uc,
		logger: logger,
	}
}

// GetProviders returns all providers for a platform
func (s *ProviderService) GetProviders(ctx context.Context, platform string, enabledOnly bool) ([]*biz.Provider, error) {
	return s.uc.ListByPlatform(ctx, platform, enabledOnly)
}

// GetProvider returns a specific provider by ID
func (s *ProviderService) GetProvider(ctx context.Context, id int64) (*biz.Provider, error) {
	return s.uc.GetByID(ctx, id)
}

// MatchModel finds providers that support a specific model
func (s *ProviderService) MatchModel(ctx context.Context, platform, model string) ([]*biz.MatchedProvider, error) {
	return s.uc.MatchModel(ctx, platform, model)
}

// CreateProvider creates a new provider
func (s *ProviderService) CreateProvider(ctx context.Context, provider *biz.Provider) (*biz.Provider, error) {
	return s.uc.Create(ctx, provider)
}

// UpdateProvider updates an existing provider
func (s *ProviderService) UpdateProvider(ctx context.Context, provider *biz.Provider) (*biz.Provider, error) {
	return s.uc.Update(ctx, provider)
}

// DeleteProvider deletes a provider
func (s *ProviderService) DeleteProvider(ctx context.Context, id int64) error {
	return s.uc.Delete(ctx, id)
}

// CheckHealth checks provider health
func (s *ProviderService) CheckHealth(ctx context.Context, providerID int64) (*biz.ProviderHealth, error) {
	provider, err := s.uc.GetByID(ctx, providerID)
	if err != nil {
		return nil, err
	}

	// TODO: Implement actual health check (HTTP request to provider API)
	// For now, return a placeholder response
	return &biz.ProviderHealth{
		ProviderID:   provider.ID,
		ProviderName: provider.Name,
		IsHealthy:    provider.Enabled,
		LatencyMs:    0,
		ErrorMessage: "",
		CheckedAt:    time.Now(),
	}, nil
}
