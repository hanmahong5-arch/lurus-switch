package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pocketzworld/lurus-switch/billing-service/internal/biz"
	"github.com/pocketzworld/lurus-switch/billing-service/internal/conf"
	"github.com/pocketzworld/lurus-switch/billing-service/internal/service"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// HTTPServer is the HTTP server
type HTTPServer struct {
	engine  *gin.Engine
	config  *conf.HTTP
	svc     *service.BillingService
	logger  *zap.Logger
}

// NewHTTPServer creates a new HTTP server
func NewHTTPServer(config *conf.HTTP, svc *service.BillingService, logger *zap.Logger) *HTTPServer {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(loggingMiddleware(logger))

	server := &HTTPServer{
		engine: engine,
		config: config,
		svc:    svc,
		logger: logger,
	}

	server.registerRoutes()
	return server
}

// registerRoutes registers all HTTP routes
func (s *HTTPServer) registerRoutes() {
	// Health check
	s.engine.GET("/health", s.healthCheck)
	s.engine.GET("/ready", s.readyCheck)

	// Metrics
	s.engine.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API v1
	v1 := s.engine.Group("/api/v1/billing")
	{
		// Balance and quota check
		v1.GET("/check/:user_id", s.checkBalance)
		v1.GET("/quota/:user_id", s.getQuota)

		// User operations
		v1.GET("/user/:user_id", s.getUser)
		v1.PUT("/user/:user_id/quota", s.updateQuota)
		v1.POST("/user/:user_id/balance", s.addBalance)

		// Usage operations
		v1.POST("/usage", s.recordUsage)
		v1.GET("/stats/:user_id", s.getUsageStats)

		// Sync endpoints for multi-client support
		v1.GET("/sync/:user_id", s.syncStatus)
		v1.GET("/sync/:user_id/stream", s.streamUpdates)
	}
}

// Start starts the HTTP server
func (s *HTTPServer) Start() error {
	s.logger.Info("Starting HTTP server", zap.String("addr", s.config.Addr))
	return s.engine.Run(s.config.Addr)
}

// healthCheck handles health check requests
func (s *HTTPServer) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

// readyCheck handles readiness check requests
func (s *HTTPServer) readyCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

// checkBalance handles balance check requests
func (s *HTTPServer) checkBalance(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	result, err := s.svc.CheckBalance(c.Request.Context(), userID)
	if err != nil {
		s.logger.Error("Failed to check balance", zap.Error(err), zap.String("user_id", userID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// getQuota handles quota query requests
func (s *HTTPServer) getQuota(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	result, err := s.svc.CheckBalance(c.Request.Context(), userID)
	if err != nil {
		s.logger.Error("Failed to get quota", zap.Error(err), zap.String("user_id", userID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":         userID,
		"quota_limit":     result.QuotaLimit,
		"quota_used":      result.QuotaUsed,
		"quota_remaining": result.QuotaRemaining,
	})
}

// getUser handles user query requests
func (s *HTTPServer) getUser(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	user, err := s.svc.GetUser(c.Request.Context(), userID)
	if err != nil {
		if err == biz.ErrUserNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		s.logger.Error("Failed to get user", zap.Error(err), zap.String("user_id", userID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

// updateQuota handles quota update requests
func (s *HTTPServer) updateQuota(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	var req struct {
		Quota int64 `json:"quota" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.svc.UpdateQuota(c.Request.Context(), userID, req.Quota); err != nil {
		if err == biz.ErrUserNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		s.logger.Error("Failed to update quota", zap.Error(err), zap.String("user_id", userID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "quota updated"})
}

// addBalance handles balance addition requests
func (s *HTTPServer) addBalance(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	var req struct {
		Amount float64 `json:"amount" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.svc.AddBalance(c.Request.Context(), userID, req.Amount); err != nil {
		if err == biz.ErrUserNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		s.logger.Error("Failed to add balance", zap.Error(err), zap.String("user_id", userID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "balance added"})
}

// recordUsage handles usage recording requests
func (s *HTTPServer) recordUsage(c *gin.Context) {
	var req struct {
		UserID       string    `json:"user_id" binding:"required"`
		TraceID      string    `json:"trace_id"`
		Platform     string    `json:"platform"`
		Model        string    `json:"model"`
		Provider     string    `json:"provider"`
		InputTokens  int       `json:"input_tokens"`
		OutputTokens int       `json:"output_tokens"`
		TotalCost    float64   `json:"total_cost"`
		Timestamp    time.Time `json:"timestamp"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	usage := &biz.UsageRecord{
		ID:           generateID(),
		UserID:       req.UserID,
		TraceID:      req.TraceID,
		Platform:     req.Platform,
		Model:        req.Model,
		Provider:     req.Provider,
		InputTokens:  req.InputTokens,
		OutputTokens: req.OutputTokens,
		TotalCost:    req.TotalCost,
		CreatedAt:    req.Timestamp,
	}

	if usage.CreatedAt.IsZero() {
		usage.CreatedAt = time.Now()
	}

	if err := s.svc.RecordUsage(c.Request.Context(), usage); err != nil {
		s.logger.Error("Failed to record usage", zap.Error(err), zap.String("user_id", req.UserID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "usage recorded", "id": usage.ID})
}

// getUsageStats handles usage statistics requests
func (s *HTTPServer) getUsageStats(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	// Parse time range
	startStr := c.DefaultQuery("start", "")
	endStr := c.DefaultQuery("end", "")

	var start, end time.Time
	if startStr != "" {
		var err error
		start, err = time.Parse(time.RFC3339, startStr)
		if err != nil {
			// Try parsing as Unix timestamp
			if ts, e := strconv.ParseInt(startStr, 10, 64); e == nil {
				start = time.Unix(ts, 0)
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start time format"})
				return
			}
		}
	} else {
		// Default to start of current month
		now := time.Now()
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	}

	if endStr != "" {
		var err error
		end, err = time.Parse(time.RFC3339, endStr)
		if err != nil {
			if ts, e := strconv.ParseInt(endStr, 10, 64); e == nil {
				end = time.Unix(ts, 0)
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end time format"})
				return
			}
		}
	} else {
		end = time.Now()
	}

	stats, err := s.svc.GetUsageStats(c.Request.Context(), userID, start, end)
	if err != nil {
		s.logger.Error("Failed to get usage stats", zap.Error(err), zap.String("user_id", userID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// syncStatus returns the current sync status for a user
func (s *HTTPServer) syncStatus(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	// Get balance info
	result, err := s.svc.CheckBalance(c.Request.Context(), userID)
	if err != nil {
		s.logger.Error("Failed to get sync status", zap.Error(err), zap.String("user_id", userID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Build sync response
	syncResponse := gin.H{
		"user_id":         userID,
		"quota_limit":     result.QuotaLimit,
		"quota_used":      result.QuotaUsed,
		"quota_remaining": result.QuotaRemaining,
		"balance":         result.Balance,
		"allowed":         result.Allowed,
		"sync_time":       time.Now().UTC().Format(time.RFC3339),
		"ttl":             30, // Suggested refresh interval in seconds
	}

	c.JSON(http.StatusOK, syncResponse)
}

// streamUpdates provides Server-Sent Events for real-time updates
func (s *HTTPServer) streamUpdates(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("X-Accel-Buffering", "no")

	// Create a channel for this client
	clientChan := make(chan string)
	defer close(clientChan)

	// Send initial sync data
	result, err := s.svc.CheckBalance(c.Request.Context(), userID)
	if err == nil {
		initialData := gin.H{
			"type":            "sync",
			"quota_limit":     result.QuotaLimit,
			"quota_used":      result.QuotaUsed,
			"quota_remaining": result.QuotaRemaining,
			"balance":         result.Balance,
			"timestamp":       time.Now().UTC().Format(time.RFC3339),
		}
		data, _ := json.Marshal(initialData)
		c.SSEvent("message", string(data))
		c.Writer.Flush()
	}

	// Send heartbeat every 30 seconds
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Listen for client disconnect
	ctx := c.Request.Context()

	for {
		select {
		case <-ctx.Done():
			s.logger.Debug("SSE client disconnected", zap.String("user_id", userID))
			return
		case <-ticker.C:
			// Send heartbeat with current status
			result, err := s.svc.CheckBalance(c.Request.Context(), userID)
			if err != nil {
				continue
			}
			heartbeat := gin.H{
				"type":            "heartbeat",
				"quota_remaining": result.QuotaRemaining,
				"balance":         result.Balance,
				"timestamp":       time.Now().UTC().Format(time.RFC3339),
			}
			data, _ := json.Marshal(heartbeat)
			c.SSEvent("message", string(data))
			c.Writer.Flush()
		}
	}
}

// loggingMiddleware creates a logging middleware
func loggingMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		logger.Debug("HTTP request",
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.Int("status", status),
			zap.Duration("latency", latency),
			zap.String("client_ip", c.ClientIP()),
		)
	}
}

// generateID generates a unique ID for usage records
func generateID() string {
	return time.Now().Format("20060102150405") + "_" + randomString(8)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}
