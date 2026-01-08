package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pocketzworld/lurus-common/observability"
	"github.com/pocketzworld/lurus-switch/provider-service/internal/conf"
	"github.com/pocketzworld/lurus-switch/provider-service/internal/server"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// App is the main application
type App struct {
	httpServer *server.HTTPServer
	logger     *zap.Logger
}

// newApp creates a new App
func newApp(httpServer *server.HTTPServer, logger *zap.Logger) *App {
	return &App{
		httpServer: httpServer,
		logger:     logger,
	}
}

// Run runs the application
func (a *App) Run() error {
	// Start HTTP server in goroutine
	errCh := make(chan error, 1)
	go func() {
		if err := a.httpServer.Start(); err != nil {
			errCh <- err
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case sig := <-quit:
		a.logger.Info("Received shutdown signal", zap.String("signal", sig.String()))
	}

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	a.logger.Info("Shutting down...")
	if err := a.httpServer.Stop(); err != nil {
		a.logger.Error("Failed to stop HTTP server", zap.Error(err))
	}

	_ = ctx // Use context for graceful shutdown if needed
	return nil
}

var (
	configPath string
)

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

	logger.Info("Starting Provider Service",
		zap.String("config", configPath),
	)

	// Load configuration
	cfg, err := conf.Load(configPath)
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
	}

	// Initialize tracing
	tracingCfg := &observability.TracingConfig{
		Enabled:        cfg.Observability.Tracing.Enabled,
		ServiceName:    "provider-service",
		ServiceVersion: "1.0.0",
		Environment:    "development",
		Endpoint:       cfg.Observability.Tracing.Endpoint,
		SampleRate:     cfg.Observability.Tracing.SampleRate,
		Insecure:       true,
	}
	tracer, err := observability.InitTracing(context.Background(), tracingCfg)
	if err != nil {
		logger.Warn("Failed to initialize tracing", zap.Error(err))
	} else if tracingCfg.Enabled {
		defer tracer.Shutdown(context.Background())
		logger.Info("Tracing initialized", zap.String("endpoint", cfg.Observability.Tracing.Endpoint))
	}

	// Initialize application with Wire
	app, cleanup, err := wireApp(cfg, logger)
	if err != nil {
		logger.Fatal("Failed to initialize application", zap.Error(err))
	}
	defer cleanup()

	// Run application
	if err := app.Run(); err != nil {
		logger.Fatal("Application error", zap.Error(err))
	}

	logger.Info("Provider Service stopped")
}
