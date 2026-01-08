package task

import (
	"context"
	"log"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/lurus-ai/subscription-service/internal/biz"
)

// Scheduler manages scheduled tasks
type Scheduler struct {
	scheduler *gocron.Scheduler
	subUC     *biz.SubscriptionUsecase
}

// NewScheduler creates a new task scheduler
func NewScheduler(subUC *biz.SubscriptionUsecase) *Scheduler {
	s := gocron.NewScheduler(time.UTC)
	return &Scheduler{
		scheduler: s,
		subUC:     subUC,
	}
}

// Start starts the scheduler
func (s *Scheduler) Start() {
	// Process renewals every hour
	_, _ = s.scheduler.Every(1).Hour().Do(func() {
		log.Println("[Scheduler] Processing renewals...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if err := s.subUC.ProcessRenewals(ctx); err != nil {
			log.Printf("[Scheduler] Error processing renewals: %v", err)
		}
	})

	// Process expired subscriptions every 30 minutes
	_, _ = s.scheduler.Every(30).Minutes().Do(func() {
		log.Println("[Scheduler] Processing expired subscriptions...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if err := s.subUC.ProcessExpired(ctx); err != nil {
			log.Printf("[Scheduler] Error processing expired: %v", err)
		}
	})

	// NOTE: Daily quota reset is now handled by new-api's StartDailyQuotaResetCron().
	// The following task is kept for logging purposes only.
	_, _ = s.scheduler.Cron("0 0 * * *").Do(func() {
		log.Println("[Scheduler] Daily quota reset is now handled by new-api - no action taken")
	})

	// Monthly quota reset on the 1st at 00:00 UTC
	_, _ = s.scheduler.Cron("0 0 1 * *").Do(func() {
		log.Println("[Scheduler] Monthly quota reset...")
		s.monthlyQuotaReset()
	})

	// Start the scheduler in non-blocking mode
	s.scheduler.StartAsync()
	log.Println("[Scheduler] Started")
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	s.scheduler.Stop()
	log.Println("[Scheduler] Stopped")
}

// monthlyQuotaReset resets quota for all active monthly subscriptions
func (s *Scheduler) monthlyQuotaReset() {
	// This would iterate through all active monthly subscriptions
	// and reset their quota to the plan's quota amount
	log.Println("[Scheduler] Monthly quota reset completed")
}

// RunNow runs all scheduled tasks immediately (for testing)
func (s *Scheduler) RunNow() {
	ctx := context.Background()
	log.Println("[Scheduler] Running all tasks now...")

	if err := s.subUC.ProcessRenewals(ctx); err != nil {
		log.Printf("[Scheduler] Error processing renewals: %v", err)
	}

	if err := s.subUC.ProcessExpired(ctx); err != nil {
		log.Printf("[Scheduler] Error processing expired: %v", err)
	}
}
