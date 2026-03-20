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
	configFileName   = "gateway.json"
	shutdownTimeout  = 5 * time.Second
)

// Server is the local OpenAI-compatible API gateway.
type Server struct {
	mu        sync.Mutex
	cfg       Config
	cfgPath   string
	server    *http.Server
	startTime time.Time
	running   atomic.Bool

	// External dependencies (injected).
	registry *appreg.Registry
	meter    *metering.Store

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
	}
	s.cfg = s.loadConfig()
	return s
}

// Start begins listening on localhost:port. Non-blocking — returns once listening.
func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running.Load() {
		return nil // already running
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
		return fmt.Errorf("listen on %s: %w", addr, err)
	}

	s.startTime = time.Now()
	s.running.Store(true)

	go func() {
		if err := s.server.Serve(ln); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "gateway server error: %v\n", err)
			s.running.Store(false)
		}
	}()

	return nil
}

// Stop gracefully shuts down the gateway.
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running.Load() {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	err := s.server.Shutdown(ctx)
	s.running.Store(false)
	s.server = nil

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
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"running": status.Running,
		"uptime":  status.Uptime,
	})
}

// handleSwitchStatus returns gateway status for third-party app integration.
func (s *Server) handleSwitchStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	status := s.Status()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"running":       status.Running,
		"port":          status.Port,
		"totalRequests": status.TotalRequests,
		"connectedApps": s.registry.ConnectedCount(),
	})
}

// handleSwitchBalance is a placeholder for balance query.
// In Phase 2 this will query Lurus Cloud via the billing client.
func (s *Server) handleSwitchBalance(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
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
