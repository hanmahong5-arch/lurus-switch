package appreg

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	registryFileName = "app-registry.json"
	tokenPrefix      = "sk-switch-"
	tokenRandomBytes = 16 // 32 hex chars
)

// Registry manages registered apps and their tokens.
type Registry struct {
	mu       sync.RWMutex
	filePath string
	apps     map[string]*App   // keyed by app ID
	byToken  map[string]string // token → app ID (lookup index)
}

// NewRegistry creates a Registry that persists to appDataDir/app-registry.json.
func NewRegistry(appDataDir string) (*Registry, error) {
	fp := filepath.Join(appDataDir, registryFileName)
	r := &Registry{
		filePath: fp,
		apps:     make(map[string]*App),
		byToken:  make(map[string]string),
	}
	if err := r.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("load app registry: %w", err)
	}
	r.ensureBuiltins()
	return r, nil
}

// List returns all registered apps.
func (r *Registry) List() []*App {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*App, 0, len(r.apps))
	for _, a := range r.apps {
		cp := *a
		out = append(out, &cp)
	}
	return out
}

// Get returns an app by ID, or nil if not found.
func (r *Registry) Get(id string) *App {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.apps[id]
	if !ok {
		return nil
	}
	cp := *a
	return &cp
}

// LookupByToken returns the app ID for a given token, or "" if invalid.
func (r *Registry) LookupByToken(token string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.byToken[token]
}

// TouchLastSeen updates the LastSeenAt timestamp for an app.
func (r *Registry) TouchLastSeen(appID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if a, ok := r.apps[appID]; ok {
		a.LastSeenAt = time.Now()
	}
	// best-effort save, don't block on error
	_ = r.saveLocked()
}

// Register creates a new user-defined app and returns it.
func (r *Registry) Register(name, icon, description string) (*App, error) {
	if name == "" {
		return nil, fmt.Errorf("app name is required")
	}

	token, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	id := generateAppID()
	app := &App{
		ID:          id,
		Name:        name,
		Kind:        KindUser,
		Tier:        TierManual,
		Token:       token,
		Icon:        icon,
		Description: description,
		CreatedAt:   time.Now(),
		Connected:   true,
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.apps[id] = app
	r.byToken[token] = id
	if err := r.saveLocked(); err != nil {
		delete(r.apps, id)
		delete(r.byToken, token)
		return nil, fmt.Errorf("save registry: %w", err)
	}
	cp := *app
	return &cp, nil
}

// Delete removes a user-registered app. Builtin apps cannot be deleted.
func (r *Registry) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	app, ok := r.apps[id]
	if !ok {
		return fmt.Errorf("app %q not found", id)
	}
	if app.Kind == KindBuiltin {
		return fmt.Errorf("cannot delete builtin app %q", id)
	}

	delete(r.byToken, app.Token)
	delete(r.apps, id)
	return r.saveLocked()
}

// ResetToken generates a new token for an app, invalidating the old one.
func (r *Registry) ResetToken(id string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	app, ok := r.apps[id]
	if !ok {
		return "", fmt.Errorf("app %q not found", id)
	}

	newToken, err := generateToken()
	if err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}

	delete(r.byToken, app.Token)
	app.Token = newToken
	r.byToken[newToken] = id

	if err := r.saveLocked(); err != nil {
		return "", fmt.Errorf("save registry: %w", err)
	}
	return newToken, nil
}

// SetConnected marks an app as connected or disconnected.
func (r *Registry) SetConnected(id string, connected bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	app, ok := r.apps[id]
	if !ok {
		return fmt.Errorf("app %q not found", id)
	}
	app.Connected = connected
	return r.saveLocked()
}

// ConnectedCount returns the number of apps marked as connected.
func (r *Registry) ConnectedCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	n := 0
	for _, a := range r.apps {
		if a.Connected {
			n++
		}
	}
	return n
}

// --- persistence ---

func (r *Registry) load() error {
	data, err := os.ReadFile(r.filePath)
	if err != nil {
		return err
	}
	var apps []*App
	if err := json.Unmarshal(data, &apps); err != nil {
		return fmt.Errorf("unmarshal registry: %w", err)
	}
	for _, a := range apps {
		r.apps[a.ID] = a
		if a.Token != "" {
			r.byToken[a.Token] = a.ID
		}
	}
	return nil
}

func (r *Registry) saveLocked() error {
	apps := make([]*App, 0, len(r.apps))
	for _, a := range r.apps {
		apps = append(apps, a)
	}
	data, err := json.MarshalIndent(apps, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(r.filePath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(r.filePath, data, 0o600)
}

// --- builtin tools ---

func (r *Registry) ensureBuiltins() {
	for _, bt := range builtinTools() {
		if _, exists := r.apps[bt.ID]; exists {
			continue
		}
		token, err := generateToken()
		if err != nil {
			continue
		}
		app := &App{
			ID:          bt.ID,
			Name:        bt.Name,
			Kind:        KindBuiltin,
			Tier:        bt.Tier,
			Token:       token,
			Icon:        bt.Icon,
			Description: bt.Description,
			CreatedAt:   time.Now(),
			Connected:   false,
		}
		r.apps[app.ID] = app
		r.byToken[token] = app.ID
	}
	_ = r.saveLocked()
}

func builtinTools() []BuiltinTool {
	return []BuiltinTool{
		{ID: "claude", Name: "Claude Code", Tier: TierAuto, Icon: "claude", Description: "Anthropic AI coding agent"},
		{ID: "codex", Name: "Codex CLI", Tier: TierAuto, Icon: "codex", Description: "OpenAI coding agent"},
		{ID: "gemini", Name: "Gemini CLI", Tier: TierAuto, Icon: "gemini", Description: "Google AI coding agent"},
		{ID: "aider", Name: "Aider", Tier: TierAuto, Icon: "aider", Description: "AI pair programming in your terminal"},
		{ID: "picoclaw", Name: "PicoClaw", Tier: TierAuto, Icon: "picoclaw", Description: "Lightweight AI coding agent"},
		{ID: "nullclaw", Name: "NullClaw", Tier: TierAuto, Icon: "nullclaw", Description: "Minimalist AI coding agent"},
		{ID: "zeroclaw", Name: "ZeroClaw", Tier: TierAuto, Icon: "zeroclaw", Description: "Zero-config AI coding agent"},
		{ID: "openclaw", Name: "OpenClaw", Tier: TierAuto, Icon: "openclaw", Description: "Open-source AI coding agent"},
		{ID: "cursor", Name: "Cursor", Tier: TierGuided, Icon: "cursor", Description: "AI-first code editor"},
		{ID: "windsurf", Name: "Windsurf", Tier: TierGuided, Icon: "windsurf", Description: "Codeium AI IDE"},
		{ID: "continue", Name: "Continue", Tier: TierGuided, Icon: "continue", Description: "Open-source AI code assistant"},
		{ID: "cline", Name: "Cline", Tier: TierGuided, Icon: "cline", Description: "Autonomous AI coding agent for VS Code"},
		{ID: "trae", Name: "Trae", Tier: TierGuided, Icon: "trae", Description: "ByteDance AI IDE"},
		{ID: "zed-ai", Name: "Zed AI", Tier: TierGuided, Icon: "zed", Description: "Zed editor AI assistant"},
	}
}

// --- helpers ---

func generateToken() (string, error) {
	b := make([]byte, tokenRandomBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return tokenPrefix + hex.EncodeToString(b), nil
}

func generateAppID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return "app-" + hex.EncodeToString(b)
}
