package observability

import (
	"context"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsConfig represents metrics configuration
type MetricsConfig struct {
	// Enabled controls whether metrics collection is enabled
	Enabled bool `json:"enabled" yaml:"enabled"`

	// Path is the HTTP path for metrics endpoint
	Path string `json:"path" yaml:"path"`

	// Port is the HTTP port for metrics server
	Port int `json:"port" yaml:"port"`

	// Namespace is the metrics namespace prefix
	Namespace string `json:"namespace" yaml:"namespace"`

	// Subsystem is the metrics subsystem prefix
	Subsystem string `json:"subsystem" yaml:"subsystem"`
}

// DefaultMetricsConfig returns default metrics configuration
func DefaultMetricsConfig() *MetricsConfig {
	return &MetricsConfig{
		Enabled:   true,
		Path:      "/metrics",
		Port:      9090,
		Namespace: "lurus",
		Subsystem: "switch",
	}
}

// LLMMetrics holds metrics for LLM operations
type LLMMetrics struct {
	RequestsTotal    *prometheus.CounterVec
	RequestDuration  *prometheus.HistogramVec
	TokensTotal      *prometheus.CounterVec
	CostTotal        *prometheus.CounterVec
	ErrorsTotal      *prometheus.CounterVec
	StreamingActive  *prometheus.GaugeVec
}

// NewLLMMetrics creates new LLM metrics
func NewLLMMetrics(namespace, subsystem string) *LLMMetrics {
	return &LLMMetrics{
		RequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "llm_requests_total",
				Help:      "Total number of LLM requests",
			},
			[]string{"platform", "provider", "model", "status"},
		),
		RequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "llm_request_duration_seconds",
				Help:      "LLM request duration in seconds",
				Buckets:   []float64{.1, .25, .5, 1, 2.5, 5, 10, 30, 60, 120, 300},
			},
			[]string{"platform", "provider", "is_stream"},
		),
		TokensTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "llm_tokens_total",
				Help:      "Total number of tokens processed",
			},
			[]string{"platform", "provider", "direction"}, // direction: input, output
		),
		CostTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "llm_cost_usd_total",
				Help:      "Total cost in USD",
			},
			[]string{"platform", "provider", "model"},
		),
		ErrorsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "llm_errors_total",
				Help:      "Total number of LLM errors",
			},
			[]string{"platform", "provider", "error_type"},
		),
		StreamingActive: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "llm_streaming_active",
				Help:      "Number of active streaming connections",
			},
			[]string{"platform", "provider"},
		),
	}
}

// RecordRequest records an LLM request
func (m *LLMMetrics) RecordRequest(platform, provider, model, status string) {
	m.RequestsTotal.WithLabelValues(platform, provider, model, status).Inc()
}

// RecordDuration records request duration
func (m *LLMMetrics) RecordDuration(platform, provider string, isStream bool, seconds float64) {
	streamLabel := "false"
	if isStream {
		streamLabel = "true"
	}
	m.RequestDuration.WithLabelValues(platform, provider, streamLabel).Observe(seconds)
}

// RecordTokens records token usage
func (m *LLMMetrics) RecordTokens(platform, provider string, input, output int) {
	m.TokensTotal.WithLabelValues(platform, provider, "input").Add(float64(input))
	m.TokensTotal.WithLabelValues(platform, provider, "output").Add(float64(output))
}

// RecordCost records cost
func (m *LLMMetrics) RecordCost(platform, provider, model string, cost float64) {
	m.CostTotal.WithLabelValues(platform, provider, model).Add(cost)
}

// RecordError records an error
func (m *LLMMetrics) RecordError(platform, provider, errorType string) {
	m.ErrorsTotal.WithLabelValues(platform, provider, errorType).Inc()
}

// StreamingStarted increments active streaming connections
func (m *LLMMetrics) StreamingStarted(platform, provider string) {
	m.StreamingActive.WithLabelValues(platform, provider).Inc()
}

// StreamingEnded decrements active streaming connections
func (m *LLMMetrics) StreamingEnded(platform, provider string) {
	m.StreamingActive.WithLabelValues(platform, provider).Dec()
}

// ServiceMetrics holds metrics for service health
type ServiceMetrics struct {
	InfoGauge       *prometheus.GaugeVec
	UpGauge         prometheus.Gauge
	StartTimeGauge  prometheus.Gauge
}

// NewServiceMetrics creates new service metrics
func NewServiceMetrics(namespace, subsystem, serviceName, version string) *ServiceMetrics {
	m := &ServiceMetrics{
		InfoGauge: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "service_info",
				Help:      "Service information",
			},
			[]string{"name", "version"},
		),
		UpGauge: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "service_up",
				Help:      "Service is up",
			},
		),
		StartTimeGauge: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "service_start_time_seconds",
				Help:      "Service start time in Unix seconds",
			},
		),
	}

	m.InfoGauge.WithLabelValues(serviceName, version).Set(1)
	return m
}

// MetricsServer handles metrics HTTP endpoint
type MetricsServer struct {
	config *MetricsConfig
	server *http.Server
}

// NewMetricsServer creates a new metrics server
func NewMetricsServer(cfg *MetricsConfig) *MetricsServer {
	if cfg == nil {
		cfg = DefaultMetricsConfig()
	}

	mux := http.NewServeMux()
	mux.Handle(cfg.Path, promhttp.Handler())

	return &MetricsServer{
		config: cfg,
		server: &http.Server{
			Addr:    formatAddr(cfg.Port),
			Handler: mux,
		},
	}
}

// Start starts the metrics server
func (s *MetricsServer) Start() error {
	if !s.config.Enabled {
		return nil
	}
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the metrics server
func (s *MetricsServer) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func formatAddr(port int) string {
	return ":" + itoa(port)
}

func itoa(i int) string {
	if i < 10 {
		return string(rune('0' + i))
	}
	return itoa(i/10) + string(rune('0'+i%10))
}
