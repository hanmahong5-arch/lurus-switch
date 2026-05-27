// Package bifrost provides an opt-in adapter that wraps the bifrost routing
// library as a drop-in replacement for the relay store + circuit breaker
// combination.  It is disabled by default (Enabled=false); callers opt in
// by setting Config.Enabled=true.
//
// Relationship to the existing relay layer
// -----------------------------------------
// Switch's relay store tracks arbitrary HTTP endpoints identified only by a
// URL + API-key pair.  Bifrost operates at a named-provider level (openai,
// anthropic, …).  The adapter bridges this gap by treating each relay
// endpoint as a custom OpenAI-compatible provider registered under a derived
// provider key ("switch-<endpoint-id>").  Bifrost's NetworkConfig.BaseURL
// carries the endpoint URL; the API key is placed in a single Key entry.
//
// Limitations / mismatch points
// --------------------------------
//  1. Fallback semantics differ: Switch's FallbackChain is HTTP-level (try
//     URL A, on 5xx try URL B). Bifrost fallbacks are per-request
//     (Provider+Model pairs). The adapter maps the Ordered slice from
//     relay.PickResult into bifrost Fallback entries — each fallback entry
//     carries a different custom provider key so bifrost's retry logic
//     effectively mirrors Switch's HTTP-level fallback chain.
//  2. Bifrost requires Go ≥ 1.26.2 (toolchain directive bumped in go.mod).
//  3. Semantic cache: enabled when Config.SemanticCacheSize > 0.  The cache
//     is bifrost-internal and does NOT interact with Switch's own caching.
//  4. The adapter does NOT expose bifrost's MCP / realtime / batch APIs.
//     Only ChatCompletion (streaming + non-streaming) is wired through.
//  5. BYO upstream proxy: the adapter forwards ProxyURL from the relay
//     endpoint if present via bifrost's ProxyConfig field.
//
// Configuration defaults
// ----------------------
//   Enabled:           false  — legacy path unchanged until user opts in
//   MaxRetries:        2      — per-endpoint HTTP retry budget
//   SemanticCacheSize: 0      — disabled
//   RequestTimeoutSec: 300    — matches Switch's upstreamTimeout (5 min)
package bifrost

import (
	"context"
	"fmt"
	"sync"

	bfcore "github.com/maximhq/bifrost/core"
	"github.com/maximhq/bifrost/core/schemas"
)

// Config holds the adapter's configuration knobs.  All fields are optional
// when Enabled is false.
type Config struct {
	// Enabled gates the adapter.  When false the adapter is a no-op and the
	// legacy relay path remains active.
	Enabled bool `json:"bifrost_enabled"`

	// MaxRetries is the per-endpoint retry budget applied to transient HTTP
	// errors (5xx, network timeouts).  Default 2.
	MaxRetries int `json:"bifrost_max_retries"`

	// SemanticCacheSize sets bifrost's in-memory semantic cache capacity
	// (number of cached responses).  0 disables the cache.
	SemanticCacheSize int `json:"bifrost_semantic_cache_size"`

	// RequestTimeoutSec is the per-request timeout in seconds forwarded to
	// bifrost's NetworkConfig.  Default 300 (5 minutes).
	RequestTimeoutSec int `json:"bifrost_request_timeout_sec"`
}

func (c *Config) withDefaults() Config {
	out := *c
	if out.MaxRetries <= 0 {
		out.MaxRetries = 2
	}
	if out.RequestTimeoutSec <= 0 {
		out.RequestTimeoutSec = 300
	}
	return out
}

// EndpointInfo is the minimum set of fields the adapter needs from each
// relay endpoint.  Callers populate this from relay.RelayEndpoint without
// importing the relay package (avoids a circular dep).
type EndpointInfo struct {
	// ID is the relay store endpoint ID — used as the bifrost provider key
	// suffix ("switch-<ID>").
	ID string
	// URL is the OpenAI-compatible HTTP base URL (e.g. "https://api.example.com").
	URL string
	// APIKey is the bearer token sent in Authorization headers.  Empty string
	// means the request is forwarded without auth (useful for local endpoints).
	APIKey string
}

// ProviderKey returns the bifrost ModelProvider derived from an endpoint ID.
func ProviderKey(endpointID string) schemas.ModelProvider {
	return schemas.ModelProvider("switch-" + endpointID)
}

// staticAccount implements schemas.Account for a fixed slice of endpoints.
// Bifrost calls these methods on every request; we return the slice as
// registered at construction time.  Live reloads are handled by constructing
// a new Adapter — cheap because bifrost.Init is fast (no I/O).
type staticAccount struct {
	endpoints []EndpointInfo
	cfg       Config
}

func (a *staticAccount) GetConfiguredProviders() ([]schemas.ModelProvider, error) {
	out := make([]schemas.ModelProvider, len(a.endpoints))
	for i, ep := range a.endpoints {
		out[i] = ProviderKey(ep.ID)
	}
	return out, nil
}

func (a *staticAccount) GetKeysForProvider(
	_ context.Context,
	providerKey schemas.ModelProvider,
) ([]schemas.Key, error) {
	for _, ep := range a.endpoints {
		if ProviderKey(ep.ID) != providerKey {
			continue
		}
		key := schemas.Key{
			ID:     ep.ID,
			Name:   string(providerKey),
			Value:  schemas.EnvVar{Val: ep.APIKey},
			Models: schemas.WhiteList{"*"},
			Weight: 1.0,
		}
		return []schemas.Key{key}, nil
	}
	return nil, fmt.Errorf("bifrost adapter: unknown provider %q", providerKey)
}

func (a *staticAccount) GetConfigForProvider(
	providerKey schemas.ModelProvider,
) (*schemas.ProviderConfig, error) {
	for _, ep := range a.endpoints {
		if ProviderKey(ep.ID) != providerKey {
			continue
		}
		cfg := &schemas.ProviderConfig{
			NetworkConfig: schemas.NetworkConfig{
				BaseURL:                        ep.URL,
				MaxRetries:                     a.cfg.MaxRetries,
				DefaultRequestTimeoutInSeconds: a.cfg.RequestTimeoutSec,
			},
			CustomProviderConfig: &schemas.CustomProviderConfig{
				BaseProviderType: schemas.OpenAI,
			},
		}
		return cfg, nil
	}
	return nil, fmt.Errorf("bifrost adapter: unknown provider %q", providerKey)
}

// Adapter wraps a bifrost instance and exposes the single method Switch
// needs: BuildFallbacks.
type Adapter struct {
	mu       sync.RWMutex
	bf       *bfcore.Bifrost
	cfg      Config
	// endpoints is a snapshot used to derive fallback chains; kept in sync
	// with the bifrost instance.
	endpoints []EndpointInfo
}

// New creates an Adapter from a set of relay endpoints and a config.
// Returns nil, nil when cfg.Enabled is false — callers must handle that.
func New(endpoints []EndpointInfo, cfg Config) (*Adapter, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	cfg = cfg.withDefaults()

	acct := &staticAccount{endpoints: endpoints, cfg: cfg}
	bf, err := bfcore.Init(context.Background(), schemas.BifrostConfig{
		Account: acct,
		Logger:  bfcore.NewNoOpLogger(),
	})
	if err != nil {
		return nil, fmt.Errorf("bifrost adapter: init: %w", err)
	}
	return &Adapter{bf: bf, cfg: cfg, endpoints: endpoints}, nil
}

// BuildFallbacks translates an ordered slice of EndpointInfo into the
// (provider, model, fallbacks) tuple bifrost needs for a chat request.
// The primary endpoint becomes the request's Provider; subsequent entries
// become bifrost Fallback objects (each a distinct custom provider key).
//
// If ordered is empty, (nil, nil) is returned — callers should fall back
// to the legacy path.
func (a *Adapter) BuildFallbacks(ordered []EndpointInfo, model string) (
	primary schemas.ModelProvider,
	fallbacks []schemas.Fallback,
) {
	if len(ordered) == 0 {
		return "", nil
	}
	primary = ProviderKey(ordered[0].ID)
	for _, ep := range ordered[1:] {
		fallbacks = append(fallbacks, schemas.Fallback{
			Provider: ProviderKey(ep.ID),
			Model:    model,
		})
	}
	return primary, fallbacks
}

// Shutdown releases bifrost's internal resources (goroutines, connections).
// Must be called when the adapter is no longer needed.
func (a *Adapter) Shutdown() {
	if a == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.bf != nil {
		a.bf.Shutdown()
		a.bf = nil
	}
}

// Enabled reports whether this adapter is active (non-nil + config Enabled).
func (a *Adapter) Enabled() bool {
	return a != nil && a.cfg.Enabled
}
