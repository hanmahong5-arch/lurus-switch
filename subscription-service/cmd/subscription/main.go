package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/lurus-ai/subscription-service/internal/biz"
	"github.com/lurus-ai/subscription-service/internal/client"
	"github.com/lurus-ai/subscription-service/internal/data"
	"github.com/lurus-ai/subscription-service/internal/server"
	"github.com/lurus-ai/subscription-service/internal/task"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	log.Println("Starting subscription-service...")

	// Load configuration
	cfg := loadConfig()

	// Connect to database
	db, err := gorm.Open(postgres.Open(cfg.DatabaseDSN), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Initialize data layer
	dataLayer := data.NewData(db)
	if err := dataLayer.AutoMigrate(); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Initialize repositories
	planRepo := data.NewPlanRepo(dataLayer)
	subRepo := data.NewSubscriptionRepo(dataLayer)

	// Initialize new-api client
	newAPIClient := client.NewNewAPIClient(cfg.NewAPIURL, cfg.NewAPIAdminToken)

	// Initialize use cases
	subUC := biz.NewSubscriptionUsecase(subRepo, planRepo, newAPIClient)

	// Initialize default plans
	if err := planRepo.InitDefaultPlans(context.Background()); err != nil {
		log.Printf("Warning: Failed to init default plans: %v", err)
	}

	// Initialize scheduler
	scheduler := task.NewScheduler(subUC)
	scheduler.Start()

	// Initialize HTTP server
	httpServer := server.NewHTTPServer(subUC, planRepo)

	// Start HTTP server in goroutine
	go func() {
		log.Printf("HTTP server listening on %s", cfg.HTTPAddr)
		if err := httpServer.Run(cfg.HTTPAddr); err != nil {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down...")
	scheduler.Stop()
	log.Println("Goodbye!")
}

type config struct {
	HTTPAddr         string
	DatabaseDSN      string
	NewAPIURL        string
	NewAPIAdminToken string
}

func loadConfig() *config {
	return &config{
		HTTPAddr:         getEnv("HTTP_ADDR", ":18104"),
		DatabaseDSN:      getEnv("DATABASE_DSN", "host=localhost port=5432 user=lurus password=lurus123 dbname=lurus_subscription sslmode=disable"),
		NewAPIURL:        getEnv("NEW_API_URL", "http://localhost:3000"),
		NewAPIAdminToken: getEnv("NEW_API_ADMIN_TOKEN", ""),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
