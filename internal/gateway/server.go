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
	"lurus-switch/internal/metering"
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
	registry *appreg.Registry
	meter    *metering.Store
	fallback *FallbackChain // cascade through backup upstreams on failure

	// Crash recovery callback (optional, set via SetCrashCallback).
	onCrash CrashCallback

	// Runtime counters.
	totalReqs  atomic.Int64
	activeReqs atomic.Int32
}

// NewServer creates a gateway server. Call Start() to begin listening.
func NewServer(appDataDir string, registry *appreg.Registry, meter *metering.Store) *Server {
	cfgPath := filepath.Join(appDataDir, configFileName)
	s := &Server{
		cfgPath:  cfgPath,
		registry: registry,
		meter:    meter,
		fallback: NewFallbackChain(nil),
	}
	s.cfg = s.loadConfig()
	s.fallback.SetEntries(s.cfg.Fallbacks)
	return s
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
