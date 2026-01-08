package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/pocketzworld/lurus-common/models"
	"github.com/pocketzworld/lurus-switch/gateway-service/internal/conf"
	"go.uber.org/zap"
)

// ProviderClient is the client for Provider Service
type ProviderClient struct {
	config     *conf.Provider
	httpClient *http.Client
	logger     *zap.Logger
	cache      *providerCache
}

// providerCache caches provider configurations
type providerCache struct {
	mu       sync.RWMutex
	data     map[string][]*models.Provider // platform -> providers
	expireAt map[string]time.Time
	ttl      time.Duration
}

// NewProviderClient creates a new provider client
func NewProviderClient(config *conf.Provider, logger *zap.Logger) *ProviderClient {
	return &ProviderClient{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		logger: logger,
		cache: &providerCache{
			data:     make(map[string][]*models.Provider),
			expireAt: make(map[string]time.Time),
			ttl:      config.CacheTTL,
		},
	}
}

// GetProviders gets providers for a platform
func (c *ProviderClient) GetProviders(ctx context.Context, platform string) ([]*models.Provider, error) {
	// Check cache first
	if providers := c.cache.get(platform); providers != nil {
		return providers, nil
	}

	// Fetch from Provider Service
	url := fmt.Sprintf("%s/api/v1/providers?platform=%s", c.config.Endpoint, platform)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch providers: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("provider service returned %d", resp.StatusCode)
	}

	var result struct {
		Providers []*models.Provider `json:"providers"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Cache the result
	c.cache.set(platform, result.Providers)

	return result.Providers, nil
}

// MatchModel finds a provider that supports the model
func (c *ProviderClient) MatchModel(ctx context.Context, platform, model string) (*models.Provider, error) {
	url := fmt.Sprintf("%s/api/v1/providers/match?platform=%s&model=%s", c.config.Endpoint, platform, model)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to match model: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("no provider found for model: %s", model)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("provider service returned %d", resp.StatusCode)
	}

	// Provider service returns {"providers": [...]}
	var result struct {
		Providers []*models.Provider `json:"providers"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Providers) == 0 {
		return nil, fmt.Errorf("no provider found for model: %s", model)
	}

	return result.Providers[0], nil
}

// InvalidateCache invalidates the cache for a platform
func (c *ProviderClient) InvalidateCache(platform string) {
	c.cache.invalidate(platform)
}

// Cache methods

func (c *providerCache) get(platform string) []*models.Provider {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if expireAt, ok := c.expireAt[platform]; ok {
		if time.Now().Before(expireAt) {
			return c.data[platform]
		}
	}
	return nil
}

func (c *providerCache) set(platform string, providers []*models.Provider) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[platform] = providers
	c.expireAt[platform] = time.Now().Add(c.ttl)
}

func (c *providerCache) invalidate(platform string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.data, platform)
	delete(c.expireAt, platform)
}
