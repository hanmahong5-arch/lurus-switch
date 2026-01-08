package middleware

import (
	"context"
	"strconv"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP metrics
	requestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gateway_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	requestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gateway_request_duration_seconds",
			Help:    "HTTP request latencies in seconds",
			Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1, 2, 5, 10, 30, 60, 120, 300},
		},
		[]string{"method", "path"},
	)

	requestSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gateway_request_size_bytes",
			Help:    "HTTP request sizes in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"method", "path"},
	)

	responseSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gateway_response_size_bytes",
			Help:    "HTTP response sizes in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"method", "path"},
	)

	// LLM business metrics
	LLMRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gateway_llm_requests_total",
			Help: "Total number of LLM API requests by platform, provider, and model",
		},
		[]string{"platform", "provider", "model", "status"},
	)

	LLMRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gateway_llm_request_duration_seconds",
			Help:    "LLM request latencies in seconds",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60, 120, 300, 600},
		},
		[]string{"platform", "provider", "is_stream"},
	)

	TokensTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gateway_tokens_total",
			Help: "Total number of tokens processed",
		},
		[]string{"platform", "provider", "direction"}, // direction: input/output
	)

	CostTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gateway_cost_usd_total",
			Help: "Total cost in USD",
		},
		[]string{"platform", "provider", "model"},
	)

	ProviderErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gateway_provider_errors_total",
			Help: "Total number of provider errors by type",
		},
		[]string{"platform", "provider", "error_type"},
	)

	ActiveRequests = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gateway_active_requests",
			Help: "Number of active requests currently being processed",
		},
		[]string{"platform"},
	)

	ProviderFailovers = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gateway_provider_failovers_total",
			Help: "Total number of provider failovers",
		},
		[]string{"platform", "from_provider", "to_provider"},
	)

	CacheHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gateway_cache_hits_total",
			Help: "Total number of cache hits",
		},
		[]string{"cache_type"}, // provider_config, etc.
	)

	CacheMisses = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gateway_cache_misses_total",
			Help: "Total number of cache misses",
		},
		[]string{"cache_type"},
	)
)

// Metrics returns a Prometheus metrics middleware
func Metrics() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		start := time.Now()
		path := normalizePath(string(c.Path()))
		method := string(c.Method())

		// Track request size
		reqSize := float64(len(c.Request.Body()))
		requestSize.WithLabelValues(method, path).Observe(reqSize)

		// Process request
		c.Next(ctx)

		// Track metrics after request
		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Response.StatusCode())
		respSize := float64(len(c.Response.Body()))

		requestsTotal.WithLabelValues(method, path, status).Inc()
		requestDuration.WithLabelValues(method, path).Observe(duration)
		responseSize.WithLabelValues(method, path).Observe(respSize)
	}
}

// RecordLLMRequest records metrics for an LLM request
func RecordLLMRequest(platform, provider, model, status string, duration float64, isStream bool) {
	LLMRequestsTotal.WithLabelValues(platform, provider, model, status).Inc()
	streamStr := "false"
	if isStream {
		streamStr = "true"
	}
	LLMRequestDuration.WithLabelValues(platform, provider, streamStr).Observe(duration)
}

// RecordTokens records token usage metrics
func RecordTokens(platform, provider string, inputTokens, outputTokens int) {
	if inputTokens > 0 {
		TokensTotal.WithLabelValues(platform, provider, "input").Add(float64(inputTokens))
	}
	if outputTokens > 0 {
		TokensTotal.WithLabelValues(platform, provider, "output").Add(float64(outputTokens))
	}
}

// RecordCost records cost metrics
func RecordCost(platform, provider, model string, cost float64) {
	if cost > 0 {
		CostTotal.WithLabelValues(platform, provider, model).Add(cost)
	}
}

// RecordProviderError records provider error metrics
func RecordProviderError(platform, provider, errorType string) {
	ProviderErrors.WithLabelValues(platform, provider, errorType).Inc()
}

// RecordFailover records provider failover metrics
func RecordFailover(platform, fromProvider, toProvider string) {
	ProviderFailovers.WithLabelValues(platform, fromProvider, toProvider).Inc()
}

// IncrementActiveRequests increments the active request counter
func IncrementActiveRequests(platform string) {
	ActiveRequests.WithLabelValues(platform).Inc()
}

// DecrementActiveRequests decrements the active request counter
func DecrementActiveRequests(platform string) {
	ActiveRequests.WithLabelValues(platform).Dec()
}

// RecordCacheHit records a cache hit
func RecordCacheHit(cacheType string) {
	CacheHits.WithLabelValues(cacheType).Inc()
}

// RecordCacheMiss records a cache miss
func RecordCacheMiss(cacheType string) {
	CacheMisses.WithLabelValues(cacheType).Inc()
}

// normalizePath normalizes the path for metrics
func normalizePath(path string) string {
	// Keep main endpoints, normalize parameters
	switch {
	case path == "/v1/messages":
		return "/v1/messages"
	case path == "/responses":
		return "/responses"
	case path == "/v1/chat/completions":
		return "/v1/chat/completions"
	case path == "/chat/completions":
		return "/chat/completions"
	case path == "/health" || path == "/ready":
		return path
	case path == "/metrics":
		return "/metrics"
	default:
		// For Gemini paths with model names, normalize
		if len(path) > 10 && path[:10] == "/v1beta/mo" {
			return "/v1beta/models/:model"
		}
		return path
	}
}
