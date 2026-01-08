package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/pocketzworld/lurus-common/observability"
	"github.com/pocketzworld/lurus-switch/billing-service/internal/biz"
	"github.com/pocketzworld/lurus-switch/billing-service/internal/conf"
	"github.com/pocketzworld/lurus-switch/billing-service/internal/consumer"
	"github.com/pocketzworld/lurus-switch/billing-service/internal/data"
	"github.com/pocketzworld/lurus-switch/billing-service/internal/server"
	"github.com/pocketzworld/lurus-switch/billing-service/internal/service"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v3"
)

var configPath string

func init() {
	flag.StringVar(&configPath, "conf", "configs/config.yaml", "config path")
}

func main() {
	flag.Parse()

	// Initialize logger
	logger := initLogger()
	defer logger.Sync()

	// Load configuration
	config, err := loadConfig(configPath)
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}

	// Initialize tracing
	tracingCfg := &observability.TracingConfig{
		Enabled:        config.Observability.Tracing.Enabled,
		ServiceName:    "billing-service",
		ServiceVersion: "1.0.0",
		Environment:    "development",
		Endpoint:       config.Observability.Tracing.Endpoint,
		SampleRate:     config.Observability.Tracing.SampleRate,
		Insecure:       true,
	}
	tracer, err := observability.InitTracing(context.Background(), tracingCfg)
	if err != nil {
		logger.Warn("Failed to initialize tracing", zap.Error(err))
	} else if tracingCfg.Enabled {
		defer tracer.Shutdown(context.Background())
		logger.Info("Tracing initialized", zap.String("endpoint", config.Observability.Tracing.Endpoint))
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize data layer
	dataLayer, dataCleanup, err := data.NewData(&config.Database, &config.Redis, logger)
	if err != nil {
		logger.Fatal("Failed to initialize data layer", zap.Error(err))
	}
	defer dataCleanup()

	// Initialize repository
	repo := data.NewBillingRepo(dataLayer, logger)

	// Initialize pricing from config
	pricing := biz.Pricing{
		InputTokens:  config.Billing.Pricing.InputTokens,
		OutputTokens: config.Billing.Pricing.OutputTokens,
	}

	// Initialize business logic
	usecase := biz.NewBillingUsecase(
		repo,
		config.Billing.DefaultQuota,
		config.Billing.FreeTier.DailyLimit,
		config.Billing.FreeTier.MonthlyLimit,
		pricing,
		logger,
	)

	// Initialize service
	svc := service.NewBillingService(usecase, logger)

	// Initialize NATS consumer
	natsConsumer, natsCleanup, err := consumer.NewNATSConsumer(usecase, &config.NATS, logger)
	if err != nil {
		logger.Fatal("Failed to initialize NATS consumer", zap.Error(err))
	}
	defer natsCleanup()

	// Start NATS consumer
	if err := natsConsumer.Start(ctx); err != nil {
		logger.Fatal("Failed to start NATS consumer", zap.Error(err))
	}

	// Initialize HTTP server
	httpServer := server.NewHTTPServer(&config.Server.HTTP, svc, logger)

	// Start HTTP server in goroutine
	go func() {
		if err := httpServer.Start(); err != nil {
			logger.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	logger.Info("Billing service started",
		zap.String("http_addr", config.Server.HTTP.Addr),
		zap.String("nats_url", config.NATS.URL),
	)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down billing service...")
	cancel()
}

func loadConfig(path string) (*conf.Bootstrap, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Expand environment variables
	data = []byte(os.ExpandEnv(string(data)))

	var config conf.Bootstrap
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func initLogger() *zap.Logger {
	config := zap.NewProductionConfig()
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	// Check for debug mode
	if os.Getenv("DEBUG") == "true" {
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	}

	logger, err := config.Build()
	if err != nil {
		panic(err)
	}

	return logger
}
