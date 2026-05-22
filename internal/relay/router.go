package relay

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

const routerRulesFile = "relay-rules.yaml"

// PickHint is the per-request context the router uses when applying
// rules. All fields are optional — empty values match the "no
// predicate" rule branch.
type PickHint struct {
	Model              string `json:"model,omitempty"`
	EstimatedInputTokens int64 `json:"estimatedInputTokens,omitempty"`
	HasTools           bool   `json:"hasTools,omitempty"`
}

// Rule represents one entry in relay-rules.yaml. Predicates AND together;
// the first rule whose predicates all match decides the EndpointID.
type Rule struct {
	Name              string `yaml:"name" json:"name"`
	MatchModelPrefix  string `yaml:"match_model_prefix,omitempty" json:"matchModelPrefix,omitempty"`
	MinTokens         int64  `yaml:"min_tokens,omitempty" json:"minTokens,omitempty"`
	PreferEndpointID  string `yaml:"prefer_endpoint_id" json:"preferEndpointID"`
}

// Rules is the deserialised on-disk YAML. Wrapped so we can carry extra
// metadata (loaded path, last error) for the UI.
type Rules struct {
	Rules []Rule `yaml:"rules" json:"rules"`
}

// Router owns the loaded rules + the circuit breaker. Pick is the only
// API the gateway needs.
type Router struct {
	mu        sync.RWMutex
	rules     Rules
	rulesPath string
	breaker   *CircuitBreaker
	store     *Store
}

// NewRouter loads (or creates) the rules YAML beside the rest of
// Switch's data and wires a fresh circuit breaker. The breaker is
// reused for the lifetime of the process — endpoint state survives
// rule reloads.
func NewRouter(appDataDir string, store *Store, breaker *CircuitBreaker) (*Router, error) {
	r := &Router{
		rulesPath: filepath.Join(appDataDir, routerRulesFile),
		breaker:   breaker,
		store:     store,
	}
	if err := r.loadRules(); err != nil {
		// Missing file is fine — empty rule set means "always pick the
		// tool→relay mapping default" which is the current behaviour.
		if !os.IsNotExist(err) {
			return r, fmt.Errorf("relay router: load rules: %w", err)
		}
	}
	return r, nil
}

// LoadRulesYAML replaces the in-memory rules from a YAML string and
// persists to disk. Validates strict to surface schema typos.
func (r *Router) LoadRulesYAML(s string) error {
	var parsed Rules
	dec := yaml.NewDecoder(strings.NewReader(s))
	dec.KnownFields(true)
	if err := dec.Decode(&parsed); err != nil {
		return fmt.Errorf("relay router: parse rules: %w", err)
	}
	r.mu.Lock()
	r.rules = parsed
	r.mu.Unlock()
	if err := os.WriteFile(r.rulesPath, []byte(s), 0o600); err != nil {
		return fmt.Errorf("relay router: persist rules: %w", err)
	}
	return nil
}

// RulesYAML returns the persisted YAML so the UI editor stays in sync
// with disk. Returns empty string when the file is missing.
func (r *Router) RulesYAML() string {
	data, err := os.ReadFile(r.rulesPath)
	if err != nil {
		return ""
	}
	return string(data)
}

func (r *Router) loadRules() error {
	data, err := os.ReadFile(r.rulesPath)
	if err != nil {
		return err
	}
	var parsed Rules
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		return err
	}
	r.rules = parsed
	return nil
}

// PickResult is the router's verdict for one request.
type PickResult struct {
	Endpoint  RelayEndpoint
	MatchedBy string // rule name; "" when only the tool→mapping default applied
	Healthy   []RelayEndpoint
}

// Pick decides which RelayEndpoint to route a request to. Strategy:
//   1. Apply user rules in order; the first match yields preferEndpointID.
//   2. Fall back to the tool→endpoint mapping the user set in Settings.
//   3. Filter out endpoints whose circuit is open.
//   4. Sort the remaining healthy peers by ascending latency and pick first.
//
// Returns an error only when there's literally no healthy endpoint.
func (r *Router) Pick(tool string, hint PickHint) (PickResult, error) {
	if r == nil || r.store == nil {
		return PickResult{}, fmt.Errorf("router not initialised")
	}
	endpoints, err := r.store.ListEndpoints()
	if err != nil {
		return PickResult{}, err
	}

	r.mu.RLock()
	rules := r.rules.Rules
	r.mu.RUnlock()

	preferred := ""
	matchedBy := ""
	for _, rule := range rules {
		if rule.MatchModelPrefix != "" && !strings.HasPrefix(hint.Model, rule.MatchModelPrefix) {
			continue
		}
		if rule.MinTokens > 0 && hint.EstimatedInputTokens < rule.MinTokens {
			continue
		}
		preferred = rule.PreferEndpointID
		matchedBy = rule.Name
		break
	}

	// Tool→mapping fallback.
	if preferred == "" {
		mapping, _ := r.store.GetToolMapping()
		if id, ok := mapping[tool]; ok {
			preferred = id
		}
	}

	healthy := r.healthyEndpoints(endpoints)
	if len(healthy) == 0 {
		return PickResult{Healthy: healthy}, fmt.Errorf("router: no healthy endpoints available")
	}

	// Prefer the explicitly-chosen endpoint when it's healthy.
	if preferred != "" {
		for _, ep := range healthy {
			if ep.ID == preferred {
				return PickResult{Endpoint: ep, MatchedBy: matchedBy, Healthy: healthy}, nil
			}
		}
	}
	// Otherwise return the lowest-latency healthy one.
	sort.SliceStable(healthy, func(i, j int) bool {
		return healthy[i].LatencyMs < healthy[j].LatencyMs
	})
	return PickResult{Endpoint: healthy[0], Healthy: healthy}, nil
}

func (r *Router) healthyEndpoints(endpoints []RelayEndpoint) []RelayEndpoint {
	if r.breaker == nil {
		return endpoints
	}
	out := make([]RelayEndpoint, 0, len(endpoints))
	for _, ep := range endpoints {
		if !r.breaker.Allow(ep.ID) {
			continue
		}
		out = append(out, ep)
	}
	return out
}

// Breaker exposes the embedded circuit breaker so the gateway can record
// per-request outcomes after the upstream call returns.
func (r *Router) Breaker() *CircuitBreaker {
	return r.breaker
}
