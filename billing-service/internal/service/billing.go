package service

import (
	"context"
	"time"

	"github.com/pocketzworld/lurus-switch/billing-service/internal/biz"
	"go.uber.org/zap"
)

// BillingService is the billing service implementation
type BillingService struct {
	uc     *biz.BillingUsecase
	logger *zap.Logger
}

// NewBillingService creates a new billing service
func NewBillingService(uc *biz.BillingUsecase, logger *zap.Logger) *BillingService {
	return &BillingService{
		uc:     uc,
		logger: logger,
	}
}

// CheckBalance checks if user has sufficient balance
func (s *BillingService) CheckBalance(ctx context.Context, userID string) (*biz.BalanceCheckResult, error) {
	return s.uc.CheckBalance(ctx, userID)
}

// RecordUsage records usage
func (s *BillingService) RecordUsage(ctx context.Context, usage *biz.UsageRecord) error {
	return s.uc.RecordUsage(ctx, usage)
}

// GetUser gets user information
func (s *BillingService) GetUser(ctx context.Context, userID string) (*biz.User, error) {
	return s.uc.GetUser(ctx, userID)
}

// GetUsageStats gets usage statistics
func (s *BillingService) GetUsageStats(ctx context.Context, userID string, start, end time.Time) (*biz.UsageStats, error) {
	return s.uc.GetUsageStats(ctx, userID, start, end)
}

// UpdateQuota updates user quota
func (s *BillingService) UpdateQuota(ctx context.Context, userID string, quota int64) error {
	return s.uc.UpdateQuota(ctx, userID, quota)
}

// AddBalance adds balance to user account
func (s *BillingService) AddBalance(ctx context.Context, userID string, amount float64) error {
	return s.uc.AddBalance(ctx, userID, amount)
}
