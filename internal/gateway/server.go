package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"lurus-switch/internal/appreg"
	"lurus-switch/internal/budget"
	"lurus-switch/internal/dlp"
	"lurus-switch/internal/metering"
	"lurus-switch/internal/obs"
	"lurus-switch/internal/relay"
)

const (
	configFileName       = "gateway.json"
	shutdownTimeout      = 5 * time.Second
	maxRestartAttempts   = 5
	initialRestartDelay  = 2 * time.Second
)

// CrashCallback is invoked when the gateway crashes and auto-restarts.
// attempt is the current restart attempt number, err is the crash error.
type CrashCallback func(attempt int, err error)

// Server is the local OpenAI-compatible API gateway.
type Server struct {
	mu        sync.Mutex
	cfg       Config
	cfgPath   string
	server    *http.Server
	startTime time.Time
	running   atomic.Bool
	stopCh    chan struct{} // signals intentional stop to watchdog

	// External dependencies (injected).
	registry   *appreg.Registry
	meter      *metering.Store
	fallback   *FallbackChain // cascade through backup upstreams on failure
	guard      *budget.Guard  // optional spend wall — nil = disabled
	dlpScanner *dlp.Scanner   // optional DLP middleware — nil = disabled

	// dlpAuditFn is called whenever DLP middleware blocks or redacts a
	// request. The injected fn is responsible for writing the audit
	// entry — keeps the gateway free of an audit dependency. The
	// metadata map carries conversation correlation keys (tool /
	// sessionID / messageUUID) when present in the request body.
	dlpAuditFn func(op, target string, payload any, metadata map[string]string)

	// Optional relay router. When wired, the gateway records upstream
	// success / failure into its circuit breaker after each fallback
	// chain attempt. Picking is still owned by FallbackChain in the
	// current request path; the breaker provides observability +
	// future routing inputs without rewriting proxy.go.
	router *relay.Router

	// Crash recovery callback (optional, set via SetCrashCallback).
	onCrash CrashCallback

	// Optional OpenTelemetry recorder. Defaults to obs.Noop() so the
	// proxy path can call it unconditionally; SetObserver swaps in a real
	// OTLP-backed recorder when the user enables observability. The
	// gateway depends only on the obs.Recorder interface — never otel
	// directly — so this stays unit-testable with a fake recorder.
	obs obs.Recorder

	// Runtime counters.
	totalReqs  atomic.Int64
	activeReqs atomic.Int32
}

// NewServer creates a gateway server. Call Start() to begin listening.
//
// Pre-W4.2 (Wave 3): the constructor seeded the FallbackChain from
// cfg.Fallbacks. That path is gone — the relay router owns the active
// chain (see buildChainFromRouter). cfg.Fallbacks is migrated into the
// relay store one-shot at services-init time, then the field stays
// empty for back-compat readers.
func NewServer(appDataDir string, registry *appreg.Registry, meter *metering.Store) *Server {
	cfgPath := filepath.Join(appDataDir, configFileName)
	s := &Server{
		cfgPath:  cfgPath,
		registry: registry,
		meter:    meter,
		fallback: NewFallbackChain(nil),
		obs:      obs.Noop(),
	}
	s.cfg = s.loadConfig()
	return s
}

// GetFallbackChain exposes the cascade chain so callers can attach an
// observer for circuit-breaker bookkeeping. The returned pointer is the
// same instance the server uses; mutations are visible immediately.
func (s *Server) GetFallbackChain() *FallbackChain {
	return s.fallback
}

// SetRelayRouter wires the optional relay router into the gateway. The
// router's circuit breaker is updated on every upstream attempt so the
// tray badge / RelayPage circuit-state column reflect live conditions.
func (s *Server) SetRelayRouter(r *relay.Router) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.router = r
}

// buildChainFromRouter asks the relay router for an ordered chain of
// upstreams for this request. Returns ok=false when the router is
// unwired or has no healthy endpoints, signalling proxy.go to fall
// back to the cfg-driven path (UpstreamURL + persisted Fallbacks).
//
// userToken is the user's Lurus / proxy token, used as the per-entry
// auth when an endpoint has no APIKey of its own — matches the
// bindings_relay.go ApplyAllToolRelays behaviour where ep.APIKey == ""
// means "fall through to the Lurus user token". Keeps a single
// token-swap policy across config-apply and runtime.
func (s *Server) buildChainFromRouter(
	tool, model string,
	estTokens int64,
	hasTools bool,
	userToken string,
) (chain []FallbackEntry, matchedBy string, ok bool) {
	s.mu.Lock()
	router := s.router
	s.mu.Unlock()
	if router == nil || !router.IsActive() {
		return nil, "", false
	}
	res, err := router.Pick(tool, relay.PickHint{
		Model:                model,
		EstimatedInputTokens: estTokens,
		HasTools:             hasTools,
	})
	if err != nil || len(res.Ordered) == 0 {
		return nil, "", false
	}
	out := make([]FallbackEntry, 0, len(res.Ordered))
	for _, ep := range res.Ordered {
		if ep.URL == "" {
			continue
		}
		token := ep.APIKey
		if token == "" {
			token = userToken
		}
		out = append(out, FallbackEntry{
			Name:  endpointDisplayName(ep),
			URL:   NormalizeChannelBaseURL(ep.URL),
			Token: token,
		})
	}
	if len(out) == 0 {
		return nil, "", false
	}
	return out, res.MatchedBy, true
}

// endpointDisplayName falls back to the endpoint ID when Name is
// unset, so the observer / metering always have a stable identifier.
func endpointDisplayName(ep relay.RelayEndpoint) string {
	if ep.Name != "" {
		return ep.Name
	}
	return ep.ID
}

// SetBudgetGuard wires the optional spend-wall into the gateway. Pass
// nil to disable. Safe to call after Start; the next request picks up
// the change.
func (s *Server) SetBudgetGuard(g *budget.Guard) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.guard = g
}

// SetObserver wires the optional OpenTelemetry recorder. Pass nil to reset
// to the no-op recorder. Safe to call after Start; the next request picks
// up the change.
func (s *Server) SetObserver(o obs.Recorder) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if o == nil {
		o = obs.Noop()
	}
	s.obs = o
}

// observe emits one RequestObservation to the configured recorder. Spans
// are retrospective (built from the observation's StartTime + LatencyMs),
// so a background context is correct here — there is no live parent span to
// inherit, and threading r.Context() through every record call site would
// be churn for no signal.
func (s *Server) observe(o obs.RequestObservation) {
	s.mu.Lock()
	rec := s.obs
	s.mu.Unlock()
	if rec == nil {
		return
	}
	rec.RecordRequest(context.Background(), o)
}

// SetCrashCallback registers a callback invoked when the gateway crashes and auto-restarts.
func (s *Server) SetCrashCallback(cb CrashCallback) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onCrash = cb
}

// Start begins listening on localhost:port. Non-blocking — returns once listening.
func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running.Load() {
		return fmt.Errorf("gateway is already running on port %d", s.cfg.Port)
	}

	if s.cfg.UpstreamURL == "" {
		return fmt.Errorf("upstream URL not configured: set it in Switch settings")
	}

	mux := http.NewServeMux()
	s.registerRoutes(mux)

	addr := fmt.Sprintf("127.0.0.1:%d", s.cfg.Port)
	s.server = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Minute,  // LLM requests can be slow
		WriteTimeout: 10 * time.Minute, // streaming responses
		IdleTimeout:  120 * time.Second,
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("port %d is in use by another process — try a different port in Settings, or close the conflicting program: %w", s.cfg.Port, err)
	}

	s.startTime = time.Now()
	s.running.Store(true)
	s.stopCh = make(chan struct{})

	go s.serveWithWatchdog(ctx, ln)

	return nil
}

// serveWithWatchdog runs the HTTP server and auto-restarts on unexpected crashes.
func (s *Server) serveWithWatchdog(ctx context.Context, ln net.Listener) {
	err := s.server.Serve(ln)
	if err == nil || err == http.ErrServerClosed {
		return // graceful shutdown, no restart
	}

	// Unexpected crash — attempt auto-restart with exponential backoff.
	fmt.Fprintf(os.Stderr, "gateway server crashed: %v\n", err)

	delay := initialRestartDelay
	for attempt := 1; attempt <= maxRestartAttempts; attempt++ {
		// Check if intentional stop was requested during the delay.
		select {
		case <-s.stopCh:
			s.running.Store(false)
			return
		case <-ctx.Done():
			s.running.Store(false)
			return
		case <-time.After(delay):
		}

		// Notify via callback.
		s.mu.Lock()
		cb := s.onCrash
		s.mu.Unlock()
		if cb != nil {
			cb(attempt, err)
		}

		// Try to restart.
		s.mu.Lock()
		addr := fmt.Sprintf("127.0.0.1:%d", s.cfg.Port)
		newLn, listenErr := net.Listen("tcp", addr)
		if listenErr != nil {
			s.mu.Unlock()
			fmt.Fprintf(os.Stderr, "gateway restart attempt %d failed: %v\n", attempt, listenErr)
			delay *= 2 // exponential backoff
			continue
		}

		mux := http.NewServeMux()
		s.registerRoutes(mux)
		s.server = &http.Server{
			Addr:         addr,
			Handler:      mux,
			ReadTimeout:  5 * time.Minute,
			WriteTimeout: 10 * time.Minute,
			IdleTimeout:  120 * time.Second,
		}
		s.startTime = time.Now()
		s.mu.Unlock()

		fmt.Fprintf(os.Stderr, "gateway restarted (attempt %d)\n", attempt)

		// Serve again — if this also crashes, the loop continues.
		err = s.server.Serve(newLn)
		if err == nil || err == http.ErrServerClosed {
			return // graceful stop
		}
		fmt.Fprintf(os.Stderr, "gateway crashed again: %v\n", err)
		delay *= 2
	}

	// Exhausted all restart attempts.
	s.running.Store(false)
	fmt.Fprintf(os.Stderr, "gateway failed to restart after %d attempts\n", maxRestartAttempts)
}

// Stop gracefully shuts down the gateway.
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running.Load() {
		return nil // already stopped, idempotent
	}

	// Signal watchdog to stop (don't auto-restart after intentional stop).
	if s.stopCh != nil {
		select {
		case <-s.stopCh:
			// already closed
		default:
			close(s.stopCh)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	var err error
	if s.server != nil {
		err = s.server.Shutdown(ctx)
		s.server = nil
	}
	s.running.Store(false)

	// Flush metering buffer.
	if s.meter != nil {
		s.meter.Flush()
	}

	return err
}

// Status returns the current gateway state.
func (s *Server) Status() Status {
	running := s.running.Load()
	url := ""
	uptime := int64(0)
	if running {
		url = fmt.Sprintf("http://localhost:%d", s.cfg.Port)
		uptime = int64(time.Since(s.startTime).Seconds())
	}
	return Status{
		Running:       running,
		Port:          s.cfg.Port,
		URL:           url,
		Uptime:        uptime,
		TotalRequests: s.totalReqs.Load(),
		ActiveConns:   s.activeReqs.Load(),
	}
}

// GetConfig returns the current gateway config.
func (s *Server) GetConfig() Config {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cfg
}

// SaveConfig persists a new config. Port changes take effect on next restart.
func (s *Server) SaveConfig(cfg Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if cfg.Port == 0 {
		cfg.Port = DefaultConfig().Port
	}
	s.cfg = cfg
	return s.saveConfigLocked()
}

// UpdateUpstream updates the upstream URL and user token without full config save.
// Used when proxy settings change.
func (s *Server) UpdateUpstream(upstreamURL, userToken string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg.UpstreamURL = upstreamURL
	s.cfg.UserToken = userToken
	_ = s.saveConfigLocked()
}

// --- route registration ---

func (s *Server) registerRoutes(mux *http.ServeMux) {
	// OpenAI-compatible API routes (auth required).
	mux.HandleFunc("/v1/chat/completions", s.withAuth(s.handleProxy))
	mux.HandleFunc("/v1/completions", s.withAuth(s.handleProxy))
	mux.HandleFunc("/v1/embeddings", s.withAuth(s.handleProxy))
	mux.HandleFunc("/v1/models", s.withAuth(s.handleProxy))

	// Anthropic Messages API → translate to OpenAI then forward. Lets
	// Claude Code talk to any OpenAI-compatible upstream (DeepSeek,
	// Groq, Ollama, OpenRouter…) while still going through the Switch
	// gateway so Bash-Guard / Budget Wall / Activity Pane all engage.
	mux.HandleFunc("/v1/messages", s.withAuth(s.handleAnthropicMessages))

	mux.HandleFunc("/v1/", s.withAuth(s.handleProxy)) // catch-all for other v1 endpoints

	// Health and control endpoints (no auth).
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/switch/v1/status", s.handleSwitchStatus)
	mux.HandleFunc("/switch/v1/balance", s.handleSwitchBalance)
	mux.HandleFunc("/switch/v1/models", s.withAuth(s.handleProxy)) // alias
}

// handleHealth returns gateway health for monitoring.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	status := s.Status()
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"running": status.Running,
		"uptime":  status.Uptime,
	})
}

// handleSwitchStatus returns gateway status for third-party app integration.
func (s *Server) handleSwitchStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	status := s.Status()
	connCount := 0
	if s.registry != nil {
		connCount = s.registry.ConnectedCount()
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"running":       status.Running,
		"port":          status.Port,
		"totalRequests": status.TotalRequests,
		"connectedApps": connCount,
	})
}

// handleSwitchBalance is a placeholder for balance query.
// In Phase 2 this will query Lurus Cloud via the billing client.
func (s *Server) handleSwitchBalance(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "balance endpoint: integrate with billing client in Phase 2",
	})
}

// --- config persistence ---

func (s *Server) loadConfig() Config {
	data, err := os.ReadFile(s.cfgPath)
	if err != nil {
		return DefaultConfig()
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return DefaultConfig()
	}
	if cfg.Port == 0 {
		cfg.Port = DefaultConfig().Port
	}
	return cfg
}

func (s *Server) saveConfigLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.cfgPath), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.cfgPath, data, 0o600)
}
