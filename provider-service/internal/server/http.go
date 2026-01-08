package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/pocketzworld/lurus-switch/provider-service/internal/biz"
	"github.com/pocketzworld/lurus-switch/provider-service/internal/conf"
	"github.com/pocketzworld/lurus-switch/provider-service/internal/service"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// HTTPServer is the HTTP server
type HTTPServer struct {
	server  *http.Server
	svc     *service.ProviderService
	logger  *zap.Logger
	config  *conf.Server
}

// NewHTTPServer creates a new HTTP server
func NewHTTPServer(c *conf.Server, svc *service.ProviderService, logger *zap.Logger) *HTTPServer {
	s := &HTTPServer{
		svc:    svc,
		logger: logger,
		config: c,
	}

	mux := http.NewServeMux()

	// Health and metrics
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/ready", s.handleReady)
	mux.Handle("/metrics", promhttp.Handler())

	// Provider API
	mux.HandleFunc("/api/v1/providers", s.handleProviders)
	mux.HandleFunc("/api/v1/providers/", s.handleProvider)
	mux.HandleFunc("/api/v1/providers/match", s.handleMatchModel)

	s.server = &http.Server{
		Addr:         c.HTTP.Addr,
		Handler:      s.withMiddleware(mux),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: c.HTTP.Timeout,
		IdleTimeout:  120 * time.Second,
	}

	return s
}

// Start starts the HTTP server
func (s *HTTPServer) Start() error {
	s.logger.Info("Starting HTTP server", zap.String("addr", s.config.HTTP.Addr))
	return s.server.ListenAndServe()
}

// Stop stops the HTTP server
func (s *HTTPServer) Stop() error {
	return s.server.Close()
}

// withMiddleware wraps the handler with middleware
func (s *HTTPServer) withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Logging
		defer func() {
			s.logger.Debug("HTTP request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Duration("duration", time.Since(start)),
			)
		}()

		next.ServeHTTP(w, r)
	})
}

// handleHealth handles health check
func (s *HTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "healthy"})
}

// handleReady handles readiness check
func (s *HTTPServer) handleReady(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

// handleProviders handles provider list and create
func (s *HTTPServer) handleProviders(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	switch r.Method {
	case http.MethodGet:
		platform := r.URL.Query().Get("platform")
		enabledOnly := r.URL.Query().Get("enabled_only") == "true"

		providers, err := s.svc.GetProviders(ctx, platform, enabledOnly)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		s.writeJSON(w, http.StatusOK, map[string]interface{}{
			"providers": providers,
		})

	case http.MethodPost:
		var provider biz.Provider
		if err := json.NewDecoder(r.Body).Decode(&provider); err != nil {
			s.writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		created, err := s.svc.CreateProvider(ctx, &provider)
		if err != nil {
			s.writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		s.writeJSON(w, http.StatusCreated, map[string]interface{}{
			"provider": created,
		})

	default:
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleProvider handles single provider operations
func (s *HTTPServer) handleProvider(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract ID from path
	path := r.URL.Path[len("/api/v1/providers/"):]
	if path == "" {
		s.writeError(w, http.StatusBadRequest, "provider ID required")
		return
	}

	// Check for sub-routes
	if path == "health" {
		s.handleProviderHealth(w, r)
		return
	}

	id, err := strconv.ParseInt(path, 10, 64)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid provider ID")
		return
	}

	switch r.Method {
	case http.MethodGet:
		provider, err := s.svc.GetProvider(ctx, id)
		if err != nil {
			s.writeError(w, http.StatusNotFound, "provider not found")
			return
		}

		s.writeJSON(w, http.StatusOK, map[string]interface{}{
			"provider": provider,
		})

	case http.MethodPut:
		var provider biz.Provider
		if err := json.NewDecoder(r.Body).Decode(&provider); err != nil {
			s.writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		provider.ID = id

		updated, err := s.svc.UpdateProvider(ctx, &provider)
		if err != nil {
			s.writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		s.writeJSON(w, http.StatusOK, map[string]interface{}{
			"provider": updated,
		})

	case http.MethodDelete:
		if err := s.svc.DeleteProvider(ctx, id); err != nil {
			s.writeError(w, http.StatusNotFound, "provider not found")
			return
		}

		s.writeJSON(w, http.StatusOK, map[string]bool{"success": true})

	default:
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleMatchModel handles model matching
func (s *HTTPServer) handleMatchModel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx := r.Context()
	platform := r.URL.Query().Get("platform")
	model := r.URL.Query().Get("model")

	if platform == "" || model == "" {
		s.writeError(w, http.StatusBadRequest, "platform and model are required")
		return
	}

	matched, err := s.svc.MatchModel(ctx, platform, model)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"providers": matched,
	})
}

// handleProviderHealth handles provider health check
func (s *HTTPServer) handleProviderHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx := r.Context()
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		s.writeError(w, http.StatusBadRequest, "provider ID required")
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid provider ID")
		return
	}

	health, err := s.svc.CheckHealth(ctx, id)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "provider not found")
		return
	}

	s.writeJSON(w, http.StatusOK, health)
}

// writeJSON writes a JSON response
func (s *HTTPServer) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes an error response
func (s *HTTPServer) writeError(w http.ResponseWriter, status int, message string) {
	s.writeJSON(w, status, map[string]interface{}{
		"error": map[string]interface{}{
			"code":    status,
			"message": message,
		},
	})
}
