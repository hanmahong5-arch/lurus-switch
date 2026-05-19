package configapply

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Planner converts an intent + opaque params into a concrete ChangePlan. Each
// mutating call site (SaveToolConfig, DeepLinkImport, ConnectAllTools, ...)
// registers one Planner with the Registry so the binding layer stays generic.
type Planner interface {
	Plan(params map[string]any) (*ChangePlan, error)
	Intent() string
	Describe(params map[string]any) string
}

// Registry holds intent → Planner mappings. Threadsafe.
type Registry struct {
	mu       sync.RWMutex
	planners map[string]Planner
}

func NewRegistry() *Registry {
	return &Registry{planners: map[string]Planner{}}
}

func (r *Registry) Register(p Planner) error {
	if p == nil {
		return fmt.Errorf("nil planner")
	}
	intent := p.Intent()
	if intent == "" {
		return fmt.Errorf("planner has empty intent")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.planners[intent]; exists {
		return fmt.Errorf("intent already registered: %s", intent)
	}
	r.planners[intent] = p
	return nil
}

func (r *Registry) Plan(intent string, params map[string]any) (*ChangePlan, error) {
	r.mu.RLock()
	p, ok := r.planners[intent]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("no planner for intent: %s", intent)
	}
	plan, err := p.Plan(params)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, fmt.Errorf("planner returned nil plan for intent: %s", intent)
	}
	if plan.ID == "" {
		plan.ID = uuid.NewString()
	}
	if plan.CreatedAt == "" {
		plan.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if plan.Intent == "" {
		plan.Intent = intent
	}
	if plan.Description == "" {
		plan.Description = p.Describe(params)
	}
	for i := range plan.Changes {
		ch := &plan.Changes[i]
		if ch.Mode == 0 {
			ch.Mode = 0644
		}
		if ch.DiffSummary == "" {
			ch.DiffSummary = DiffSummary(ch.Before, ch.After)
		}
		if ch.UnifiedDiff == "" && ch.Kind != KindDelete {
			ch.UnifiedDiff = UnifiedDiff(ch.Before, ch.After, 3)
		}
	}
	return plan, nil
}

func (r *Registry) Intents() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.planners))
	for k := range r.planners {
		out = append(out, k)
	}
	return out
}

// SaveSingleFilePlanner is the simplest reusable planner: write/overwrite one
// file with new content. Used by SaveToolConfig, DeepLinkImport, and any caller
// whose mutation is a single-file rewrite.
type SaveSingleFilePlanner struct {
	IntentName  string
	DescribeFn  func(params map[string]any) string
	SideEffects func(params map[string]any) []string
}

func (p SaveSingleFilePlanner) Intent() string { return p.IntentName }
func (p SaveSingleFilePlanner) Describe(params map[string]any) string {
	if p.DescribeFn != nil {
		return p.DescribeFn(params)
	}
	return fmt.Sprintf("%s: %v", p.IntentName, params["path"])
}

func (p SaveSingleFilePlanner) Plan(params map[string]any) (*ChangePlan, error) {
	path, _ := params["path"].(string)
	after, _ := params["content"].(string)
	if path == "" {
		return nil, fmt.Errorf("missing 'path' param")
	}

	before, err := ReadFileOrEmpty(path)
	if err != nil {
		return nil, fmt.Errorf("read before: %w", err)
	}

	kind := KindUpdate
	if before == "" {
		kind = KindCreate
	}
	if after == "" && before != "" {
		kind = KindDelete
	}

	var sideEffects []string
	if p.SideEffects != nil {
		sideEffects = p.SideEffects(params)
	}

	return &ChangePlan{
		Changes: []FileChange{{
			Path:   path,
			Kind:   kind,
			Before: before,
			After:  after,
			Mode:   0644,
		}},
		SideEffects: sideEffects,
	}, nil
}
