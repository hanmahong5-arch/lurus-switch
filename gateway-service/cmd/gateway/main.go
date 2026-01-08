package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/pocketzworld/lurus-common/observability"
	"github.com/pocketzworld/lurus-switch/gateway-service/internal/client"
	"github.com/pocketzworld/lurus-switch/gateway-service/internal/conf"
	"github.com/pocketzworld/lurus-switch/gateway-service/internal/handler"
	"github.com/pocketzworld/lurus-switch/gateway-service/internal/middleware"
	"github.com/pocketzworld/lurus-switch/gateway-service/internal/proxy"
	"github.com/pocketzworld/lurus-switch/gateway-service/pkg/nats"
)

var configPath string

func init() {
	flag.StringVar(&configPath, "conf", "configs/config.yaml", "config path")
}

func main() {
	flag.Parse()

	// Initialize logger
	logConfig := zap.NewProductionConfig()
	logConfig.EncoderConfig.TimeKey = "time"
	logConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	logger, err := logConfig.Build()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	logger.Info("Starting Gateway Service", zap.String("config", configPath))

	// Load configuration
	cfg, err := conf.Load(configPath)
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
	}

	// Initialize tracing
	tracingCfg := &observability.TracingConfig{
		Enabled:        cfg.Tracing.Enabled,
		ServiceName:    "gateway-service",
		ServiceVersion: "1.0.0",
		Environment:    cfg.Server.Mode,
		Endpoint:       cfg.Tracing.Endpoint,
		SampleRate:     cfg.Tracing.SampleRate,
		Insecure:       true,
	}
	tracer, err := observability.InitTracing(context.Background(), tracingCfg)
	if err != nil {
		logger.Warn("Failed to initialize tracing", zap.Error(err))
	} else if tracingCfg.Enabled {
		defer tracer.Shutdown(context.Background())
		logger.Info("Tracing initialized", zap.String("endpoint", cfg.Tracing.Endpoint))
	}

	// Initialize clients
	providerClient := client.NewProviderClient(&cfg.Provider, logger)
	billingClient := client.NewBillingClient(&cfg.Billing, logger)

	// Initialize NATS publisher
	var natsPublisher *nats.Publisher
	var natsCleanup func()
	if cfg.Features.AsyncLogging && cfg.NATS.URL != "" {
		natsPublisher, natsCleanup, err = nats.NewPublisher(&cfg.NATS, logger)
		if err != nil {
			logger.Warn("Failed to connect to NATS, continuing without async logging", zap.Error(err))
		} else {
			defer natsCleanup()
			logger.Info("Connected to NATS", zap.String("url", cfg.NATS.URL))
		}
	}

	// Initialize relay service
	relay := proxy.NewRelayService(cfg, providerClient, billingClient, natsPublisher, logger)

	// Initialize handlers
	claudeHandler := handler.NewClaudeHandler(relay, logger)
	codexHandler := handler.NewCodexHandler(relay, logger)
	geminiHandler := handler.NewGeminiHandler(relay, logger)

	// Create Hertz server
	h := server.Default(
		server.WithHostPorts(cfg.Server.HTTP.Addr),
		server.WithReadTimeout(cfg.Server.HTTP.ReadTimeout),
		server.WithWriteTimeout(cfg.Server.HTTP.WriteTimeout),
		server.WithIdleTimeout(cfg.Server.HTTP.IdleTimeout),
		server.WithMaxRequestBodySize(100*1024*1024), // 100MB max body
	)

	// Set log level
	hlog.SetLevel(hlog.LevelInfo)

	// Global middleware
	h.Use(middleware.CORS())
	h.Use(middleware.Tracing())
	h.Use(middleware.Logger(logger))
	h.Use(middleware.Metrics())

	// Health endpoints
	h.GET("/health", func(ctx context.Context, c *app.RequestContext) {
		c.JSON(http.StatusOK, map[string]string{"status": "healthy"})
	})
	h.GET("/ready", func(ctx context.Context, c *app.RequestContext) {
		c.JSON(http.StatusOK, map[string]string{"status": "ready"})
	})

	// Metrics endpoint (using standard http handler adapter)
	h.GET("/metrics", func(ctx context.Context, c *app.RequestContext) {
		req, _ := http.NewRequest("GET", "/metrics", nil)
		promhttp.Handler().ServeHTTP(
			&responseWriterAdapter{c},
			req,
		)
	})

	// Claude API routes
	h.POST("/v1/messages", claudeHandler.Messages)
	h.POST("/v1/messages/count_tokens", claudeHandler.CountTokens)
	h.POST("/v1/messages/batches", claudeHandler.Batches)

	// Codex/OpenAI API routes
	h.POST("/responses", codexHandler.Responses)
	h.POST("/v1/chat/completions", codexHandler.ChatCompletions)
	h.POST("/chat/completions", codexHandler.ChatCompletionsAlt)
	h.POST("/v1/completions", codexHandler.Completions)
	h.POST("/v1/embeddings", codexHandler.Embeddings)

	// Gemini API routes - use wildcard for model:action pattern
	h.GET("/v1beta/models", geminiHandler.Models)
	h.POST("/v1beta/models/*modelAction", geminiHandler.HandleModelAction)
	h.Any("/v1beta/*path", geminiHandler.GenericEndpoint)

	// Start server
	go func() {
		logger.Info("Starting HTTP server", zap.String("addr", cfg.Server.HTTP.Addr))
		h.Spin()
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	logger.Info("Received shutdown signal", zap.String("signal", sig.String()))

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := h.Shutdown(ctx); err != nil {
		logger.Error("Server shutdown error", zap.Error(err))
	}

	logger.Info("Gateway Service stopped")
}

// responseWriterAdapter adapts Hertz RequestContext to http.ResponseWriter
type responseWriterAdapter struct {
	c *app.RequestContext
}

func (w *responseWriterAdapter) Header() http.Header {
	headers := make(http.Header)
	w.c.Response.Header.VisitAll(func(key, value []byte) {
		headers.Add(string(key), string(value))
	})
	return headers
}

func (w *responseWriterAdapter) Write(data []byte) (int, error) {
	w.c.Response.AppendBody(data)
	return len(data), nil
}

func (w *responseWriterAdapter) WriteHeader(statusCode int) {
	w.c.Response.SetStatusCode(statusCode)
}
