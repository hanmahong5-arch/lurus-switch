package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/pocketzworld/lurus-switch/log-service/internal/biz"
	"github.com/pocketzworld/lurus-switch/log-service/internal/conf"
	"github.com/pocketzworld/lurus-switch/log-service/internal/service"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// HTTPServer is the HTTP server
type HTTPServer struct {
	server *http.Server
	svc    *service.LogService
	logger *zap.Logger
	config *conf.Server
}

// NewHTTPServer creates a new HTTP server
func NewHTTPServer(c *conf.Server, svc *service.LogService, logger *zap.Logger) *HTTPServer {
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

	// Log API
	mux.HandleFunc("/api/v1/logs", s.handleLogs)
	mux.HandleFunc("/api/v1/logs/write", s.handleWriteLog)
	mux.HandleFunc("/api/v1/logs/batch", s.handleWriteBatch)

	// Stats API
	mux.HandleFunc("/api/v1/stats", s.handleStats)
	mux.HandleFunc("/api/v1/stats/hourly", s.handleHourlyStats)
	mux.HandleFunc("/api/v1/stats/daily", s.handleDailyStats)
	mux.HandleFunc("/api/v1/stats/models", s.handleModelUsage)

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
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

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

func (s *HTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "healthy"})
}

func (s *HTTPServer) handleReady(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

// handleLogs handles log queries
func (s *HTTPServer) handleLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx := r.Context()
	q := r.URL.Query()

	filter := &biz.LogFilter{
		UserID:     q.Get("user_id"),
		Platform:   q.Get("platform"),
		Provider:   q.Get("provider"),
		Model:      q.Get("model"),
		TraceID:    q.Get("trace_id"),
		OrderBy:    q.Get("order_by"),
		Descending: q.Get("desc") == "true",
	}

	if limit := q.Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			filter.Limit = l
		}
	}
	if offset := q.Get("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil {
			filter.Offset = o
		}
	}
	if start := q.Get("start_time"); start != "" {
		if t, err := time.Parse(time.RFC3339, start); err == nil {
			filter.StartTime = t
		}
	}
	if end := q.Get("end_time"); end != "" {
		if t, err := time.Parse(time.RFC3339, end); err == nil {
			filter.EndTime = t
		}
	}

	logs, total, err := s.svc.QueryLogs(ctx, filter)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"logs":  logs,
		"total": total,
	})
}

// handleWriteLog handles single log write
func (s *HTTPServer) handleWriteLog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var log biz.RequestLog
	if err := json.NewDecoder(r.Body).Decode(&log); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := s.svc.WriteLog(r.Context(), &log); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// handleWriteBatch handles batch log write
func (s *HTTPServer) handleWriteBatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Logs []*biz.RequestLog `json:"logs"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	written, failed, err := s.svc.WriteLogs(r.Context(), req.Logs)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"written": written,
		"failed":  failed,
	})
}

// handleStats handles aggregated statistics
func (s *HTTPServer) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx := r.Context()
	q := r.URL.Query()

	userID := q.Get("user_id")
	platform := q.Get("platform")
	startTime, endTime := s.parseTimeRange(q)

	stats, err := s.svc.GetStats(ctx, userID, platform, startTime, endTime)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, stats)
}

// handleHourlyStats handles hourly statistics
func (s *HTTPServer) handleHourlyStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx := r.Context()
	q := r.URL.Query()

	userID := q.Get("user_id")
	platform := q.Get("platform")
	startTime, endTime := s.parseTimeRange(q)

	stats, err := s.svc.GetHourlyStats(ctx, userID, platform, startTime, endTime)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{"stats": stats})
}

// handleDailyStats handles daily statistics
func (s *HTTPServer) handleDailyStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx := r.Context()
	q := r.URL.Query()

	userID := q.Get("user_id")
	platform := q.Get("platform")
	startTime, endTime := s.parseTimeRange(q)

	stats, err := s.svc.GetDailyStats(ctx, userID, platform, startTime, endTime)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{"stats": stats})
}

// handleModelUsage handles per-model usage statistics
func (s *HTTPServer) handleModelUsage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx := r.Context()
	q := r.URL.Query()

	userID := q.Get("user_id")
	platform := q.Get("platform")
	startTime, endTime := s.parseTimeRange(q)

	limit := 20
	if l := q.Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	stats, err := s.svc.GetModelUsage(ctx, userID, platform, startTime, endTime, limit)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{"stats": stats})
}

func (s *HTTPServer) parseTimeRange(q map[string][]string) (time.Time, time.Time) {
	var startTime, endTime time.Time

	if start, ok := q["start_time"]; ok && len(start) > 0 {
		if t, err := time.Parse(time.RFC3339, start[0]); err == nil {
			startTime = t
		}
	}
	if end, ok := q["end_time"]; ok && len(end) > 0 {
		if t, err := time.Parse(time.RFC3339, end[0]); err == nil {
			endTime = t
		}
	}

	// Default: last 24 hours
	if startTime.IsZero() {
		startTime = time.Now().Add(-24 * time.Hour)
	}
	if endTime.IsZero() {
		endTime = time.Now()
	}

	return startTime, endTime
}

func (s *HTTPServer) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (s *HTTPServer) writeError(w http.ResponseWriter, status int, message string) {
	s.writeJSON(w, status, map[string]interface{}{
		"error": map[string]interface{}{
			"code":    status,
			"message": message,
		},
	})
}
