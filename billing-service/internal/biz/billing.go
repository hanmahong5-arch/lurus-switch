package biz

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrInsufficientQuota  = errors.New("insufficient quota")
	ErrInsufficientBalance = errors.New("insufficient balance")
)

// User represents a billing user
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	Plan         string    `json:"plan"` // free, pro, enterprise
	Balance      float64   `json:"balance"`
	QuotaLimit   int64     `json:"quota_limit"`
	QuotaUsed    int64     `json:"quota_used"`
	QuotaResetAt time.Time `json:"quota_reset_at"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// UsageRecord represents a usage record
type UsageRecord struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	TraceID      string    `json:"trace_id"`
	Platform     string    `json:"platform"`
	Model        string    `json:"model"`
	Provider     string    `json:"provider"`
	InputTokens  int       `json:"input_tokens"`
	OutputTokens int       `json:"output_tokens"`
	TotalCost    float64   `json:"total_cost"`
	CreatedAt    time.Time `json:"created_at"`
}

// BalanceCheckResult represents the result of a balance check
type BalanceCheckResult struct {
	Allowed       bool    `json:"allowed"`
	Balance       float64 `json:"balance"`
	QuotaLimit    int64   `json:"quota_limit"`
	QuotaUsed     int64   `json:"quota_used"`
	QuotaRemaining int64  `json:"quota_remaining"`
	Message       string  `json:"message,omitempty"`
}

// UsageStats represents usage statistics
type UsageStats struct {
	UserID         string  `json:"user_id"`
	TotalRequests  int64   `json:"total_requests"`
	TotalTokens    int64   `json:"total_tokens"`
	TotalCost      float64 `json:"total_cost"`
	InputTokens    int64   `json:"input_tokens"`
	OutputTokens   int64   `json:"output_tokens"`
	PeriodStart    time.Time `json:"period_start"`
	PeriodEnd      time.Time `json:"period_end"`
}

// BillingRepo is the billing repository interface
type BillingRepo interface {
	// User operations
	GetUser(ctx context.Context, userID string) (*User, error)
	CreateUser(ctx context.Context, user *User) error
	UpdateUser(ctx context.Context, user *User) error

	// Usage operations
	RecordUsage(ctx context.Context, usage *UsageRecord) error
	GetUsageStats(ctx context.Context, userID string, start, end time.Time) (*UsageStats, error)

	// Quota operations
	GetQuotaUsed(ctx context.Context, userID string) (int64, error)
	IncrementQuota(ctx context.Context, userID string, tokens int64) error
	ResetQuota(ctx context.Context, userID string) error
}

// BillingUsecase is the billing business logic
type BillingUsecase struct {
	repo          BillingRepo
	defaultQuota  int64
	freeTierDaily int64
	freeTierMonthly int64
	pricing       Pricing
	logger        *zap.Logger
}

// Pricing holds token pricing
type Pricing struct {
	InputTokens       float64
	OutputTokens      float64
	CacheReadTokens   float64
	CacheCreateTokens float64
}

// NewBillingUsecase creates a new billing usecase
func NewBillingUsecase(
	repo BillingRepo,
	defaultQuota int64,
	freeTierDaily int64,
	freeTierMonthly int64,
	pricing Pricing,
	logger *zap.Logger,
) *BillingUsecase {
	return &BillingUsecase{
		repo:            repo,
		defaultQuota:    defaultQuota,
		freeTierDaily:   freeTierDaily,
		freeTierMonthly: freeTierMonthly,
		pricing:         pricing,
		logger:          logger,
	}
}

// CheckBalance checks if user has sufficient balance/quota
func (uc *BillingUsecase) CheckBalance(ctx context.Context, userID string) (*BalanceCheckResult, error) {
	user, err := uc.repo.GetUser(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			// Create new user with default quota
			user = &User{
				ID:           userID,
				Plan:         "free",
				Balance:      0,
				QuotaLimit:   uc.defaultQuota,
				QuotaUsed:    0,
				QuotaResetAt: uc.getNextResetTime(),
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			}
			if err := uc.repo.CreateUser(ctx, user); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	// Check if quota needs reset
	if time.Now().After(user.QuotaResetAt) {
		user.QuotaUsed = 0
		user.QuotaResetAt = uc.getNextResetTime()
		if err := uc.repo.UpdateUser(ctx, user); err != nil {
			uc.logger.Warn("Failed to reset quota", zap.Error(err))
		}
	}

	remaining := user.QuotaLimit - user.QuotaUsed

	result := &BalanceCheckResult{
		Balance:        user.Balance,
		QuotaLimit:     user.QuotaLimit,
		QuotaUsed:      user.QuotaUsed,
		QuotaRemaining: remaining,
	}

	// Check quota
	if remaining <= 0 {
		result.Allowed = false
		result.Message = "Quota exceeded"
		return result, nil
	}

	// Check balance for paid plans
	if user.Plan != "free" && user.Balance <= 0 {
		result.Allowed = false
		result.Message = "Insufficient balance"
		return result, nil
	}

	result.Allowed = true
	return result, nil
}

// RecordUsage records usage and updates quota
func (uc *BillingUsecase) RecordUsage(ctx context.Context, usage *UsageRecord) error {
	// Calculate cost
	if usage.TotalCost == 0 {
		usage.TotalCost = uc.calculateCost(usage.InputTokens, usage.OutputTokens)
	}

	// Record usage
	if err := uc.repo.RecordUsage(ctx, usage); err != nil {
		return err
	}

	// Update quota
	totalTokens := int64(usage.InputTokens + usage.OutputTokens)
	if err := uc.repo.IncrementQuota(ctx, usage.UserID, totalTokens); err != nil {
		uc.logger.Warn("Failed to increment quota", zap.Error(err))
	}

	return nil
}

// GetUsageStats gets usage statistics for a user
func (uc *BillingUsecase) GetUsageStats(ctx context.Context, userID string, start, end time.Time) (*UsageStats, error) {
	return uc.repo.GetUsageStats(ctx, userID, start, end)
}

// GetUser gets user information
func (uc *BillingUsecase) GetUser(ctx context.Context, userID string) (*User, error) {
	return uc.repo.GetUser(ctx, userID)
}

// UpdateQuota updates user quota
func (uc *BillingUsecase) UpdateQuota(ctx context.Context, userID string, quota int64) error {
	user, err := uc.repo.GetUser(ctx, userID)
	if err != nil {
		return err
	}

	user.QuotaLimit = quota
	user.UpdatedAt = time.Now()
	return uc.repo.UpdateUser(ctx, user)
}

// AddBalance adds balance to user account
func (uc *BillingUsecase) AddBalance(ctx context.Context, userID string, amount float64) error {
	user, err := uc.repo.GetUser(ctx, userID)
	if err != nil {
		return err
	}

	user.Balance += amount
	user.UpdatedAt = time.Now()
	return uc.repo.UpdateUser(ctx, user)
}

// calculateCost calculates the cost for tokens
func (uc *BillingUsecase) calculateCost(inputTokens, outputTokens int) float64 {
	inputCost := float64(inputTokens) / 1000000 * uc.pricing.InputTokens
	outputCost := float64(outputTokens) / 1000000 * uc.pricing.OutputTokens
	return inputCost + outputCost
}

// getNextResetTime calculates the next quota reset time
func (uc *BillingUsecase) getNextResetTime() time.Time {
	now := time.Now()
	// Reset at the start of next month
	return time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())
}
