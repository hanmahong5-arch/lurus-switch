package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP metrics
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "billing_http_requests_total",
			Help: "Total HTTP requests to billing service",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "billing_http_request_duration_seconds",
			Help:    "HTTP request latency in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
		},
		[]string{"method", "path"},
	)

	// Billing business metrics
	balanceChecksTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "billing_balance_checks_total",
			Help: "Total balance check operations",
		},
		[]string{"result"}, // "allowed", "denied", "error"
	)

	usageRecordsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "billing_usage_records_total",
			Help: "Total usage records processed",
		},
		[]string{"platform", "status"},
	)

	tokensProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "billing_tokens_processed_total",
			Help: "Total tokens processed by billing",
		},
		[]string{"platform", "direction"}, // direction: input, output
	)

	costRecorded = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "billing_cost_usd_total",
			Help: "Total cost recorded in USD",
		},
		[]string{"platform", "model"},
	)

	quotaUpdatesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "billing_quota_updates_total",
			Help: "Total quota update operations",
		},
		[]string{"operation"}, // "increase", "decrease", "reset"
	)

	activeUsersGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "billing_active_users",
			Help: "Number of active users with balance",
		},
	)

	lowBalanceAlerts = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "billing_low_balance_alerts_total",
			Help: "Total low balance alerts triggered",
		},
	)

	natsEventsPublished = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "billing_nats_events_published_total",
			Help: "Total NATS events published",
		},
		[]string{"event_type"},
	)
)

// MetricsMiddleware returns a Gin middleware for recording HTTP metrics
func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := normalizePath(c.FullPath())

		c.Next()

		status := strconv.Itoa(c.Writer.Status())
		method := c.Request.Method
		duration := time.Since(start).Seconds()

		httpRequestsTotal.WithLabelValues(method, path, status).Inc()
		httpRequestDuration.WithLabelValues(method, path).Observe(duration)
	}
}

// normalizePath normalizes the path for metrics
func normalizePath(path string) string {
	if path == "" {
		return "unknown"
	}
	return path
}

// RecordBalanceCheck records a balance check result
func RecordBalanceCheck(result string) {
	balanceChecksTotal.WithLabelValues(result).Inc()
}

// RecordUsage records usage metrics
func RecordUsage(platform string, inputTokens, outputTokens int, cost float64, model string) {
	usageRecordsTotal.WithLabelValues(platform, "success").Inc()
	tokensProcessed.WithLabelValues(platform, "input").Add(float64(inputTokens))
	tokensProcessed.WithLabelValues(platform, "output").Add(float64(outputTokens))
	costRecorded.WithLabelValues(platform, model).Add(cost)
}

// RecordUsageError records a usage recording error
func RecordUsageError(platform string) {
	usageRecordsTotal.WithLabelValues(platform, "error").Inc()
}

// RecordQuotaUpdate records a quota update operation
func RecordQuotaUpdate(operation string) {
	quotaUpdatesTotal.WithLabelValues(operation).Inc()
}

// SetActiveUsers sets the active users gauge
func SetActiveUsers(count int) {
	activeUsersGauge.Set(float64(count))
}

// RecordLowBalanceAlert records a low balance alert
func RecordLowBalanceAlert() {
	lowBalanceAlerts.Inc()
}

// RecordNATSEvent records a NATS event publication
func RecordNATSEvent(eventType string) {
	natsEventsPublished.WithLabelValues(eventType).Inc()
}
