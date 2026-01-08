package biz

import (
	"context"
	"time"
)

// SubscriptionStatus defines subscription status
type SubscriptionStatus string

const (
	SubscriptionStatusActive    SubscriptionStatus = "active"
	SubscriptionStatusExpired   SubscriptionStatus = "expired"
	SubscriptionStatusCancelled SubscriptionStatus = "cancelled"
	SubscriptionStatusPending   SubscriptionStatus = "pending"
)

// Subscription represents a user's subscription
// NOTE: Daily quota state (TodayUsed, LastDailyResetAt, CurrentGroup) is now managed by new-api.
// subscription-service only manages subscription lifecycle and syncs config to new-api.
type Subscription struct {
	ID             int64              `json:"id" gorm:"primaryKey"`
	UserID         int                `json:"user_id" gorm:"index;not null"`          // new-api user_id
	PlanID         int64              `json:"plan_id" gorm:"index;not null"`
	Plan           *Plan              `json:"plan,omitempty" gorm:"foreignKey:PlanID"`
	Status         SubscriptionStatus `json:"status" gorm:"size:20;default:'pending'"`
	StartedAt      time.Time          `json:"started_at"`
	ExpiresAt      time.Time          `json:"expires_at" gorm:"index"`
	AutoRenew      bool               `json:"auto_renew" gorm:"default:true"`
	CurrentQuota   int64              `json:"current_quota"`                          // Remaining quota in current period
	UsedQuota      int64              `json:"used_quota" gorm:"default:0"`            // Used quota in current period
	LastResetAt    time.Time          `json:"last_reset_at"`                          // Last quota reset time

	// Daily quota config (from plan) - state is managed by new-api
	DailyQuota     int64     `json:"daily_quota"`                                     // Daily limit from plan

	CancelledAt    *time.Time         `json:"cancelled_at,omitempty"`
	CancelReason   string             `json:"cancel_reason,omitempty" gorm:"size:255"`
	ExternalID     string             `json:"external_id,omitempty" gorm:"size:100"`  // External payment system ID
	CreatedAt      time.Time          `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      time.Time          `json:"updated_at" gorm:"autoUpdateTime"`
}

func (Subscription) TableName() string {
	return "subscriptions"
}

// IsActive returns whether the subscription is currently active
func (s *Subscription) IsActive() bool {
	return s.Status == SubscriptionStatusActive && time.Now().Before(s.ExpiresAt)
}

// HasQuota returns whether the subscription has remaining quota
func (s *Subscription) HasQuota() bool {
	return s.CurrentQuota > 0
}

// HasDailyQuotaConfig returns whether the subscription has daily quota limit configured
func (s *Subscription) HasDailyQuotaConfig() bool {
	return s.DailyQuota > 0
}

// DaysUntilExpiry returns the number of days until expiry
func (s *Subscription) DaysUntilExpiry() int {
	if time.Now().After(s.ExpiresAt) {
		return 0
	}
	return int(time.Until(s.ExpiresAt).Hours() / 24)
}

// SubscriptionRepo defines the subscription repository interface
// NOTE: Daily quota state operations (DeductDailyQuota, ResetDailyQuota, UpdateCurrentGroup)
// have been removed as daily quota state is now managed by new-api.
type SubscriptionRepo interface {
	Create(ctx context.Context, sub *Subscription) error
	Update(ctx context.Context, sub *Subscription) error
	GetByID(ctx context.Context, id int64) (*Subscription, error)
	GetByUserID(ctx context.Context, userID int) (*Subscription, error)
	GetActiveByUserID(ctx context.Context, userID int) (*Subscription, error)
	ListExpiring(ctx context.Context, before time.Time) ([]*Subscription, error)
	ListExpired(ctx context.Context) ([]*Subscription, error)
	ListForRenewal(ctx context.Context, before time.Time) ([]*Subscription, error)
	ListActive(ctx context.Context) ([]*Subscription, error)
	ListWithFilters(ctx context.Context, page, pageSize int, status, planCode string, userID int) ([]*Subscription, int64, error)
	DeductQuota(ctx context.Context, id int64, amount int64) error
	ResetQuota(ctx context.Context, id int64, quota int64) error
	// Stats
	GetStatsOverview(ctx context.Context) (*StatsOverview, error)
}

// PlanRepo defines the plan repository interface
type PlanRepo interface {
	Create(ctx context.Context, plan *Plan) error
	Update(ctx context.Context, plan *Plan) error
	GetByID(ctx context.Context, id int64) (*Plan, error)
	GetByCode(ctx context.Context, code string) (*Plan, error)
	ListActive(ctx context.Context) ([]*Plan, error)
	InitDefaultPlans(ctx context.Context) error
}

// SubscriptionUsecase defines the subscription business logic
type SubscriptionUsecase struct {
	subRepo  SubscriptionRepo
	planRepo PlanRepo
	newAPI   NewAPIClient
}

// NewAPIClient defines the interface for new-api integration
// NOTE: subscription-service now uses UpdateUserSubscriptionConfig for unified quota/group management.
type NewAPIClient interface {
	GetUser(ctx context.Context, userID int) (*NewAPIUser, error)
	UpdateUserQuota(ctx context.Context, userID int, quota int64) error
	UpdateUserGroup(ctx context.Context, userID int, group string) error
	CreateToken(ctx context.Context, userID int, name string, quota int64, expiredTime int64) (*NewAPIToken, error)
	// New unified subscription config API
	UpdateUserSubscriptionConfig(ctx context.Context, userID int, config *SubscriptionConfig) error
	GetUserDailyQuotaStatus(ctx context.Context, userID int) (*DailyQuotaStatus, error)
	ResetUserDailyQuota(ctx context.Context, userID int) error
}

// SubscriptionConfig represents the config to sync to new-api
type SubscriptionConfig struct {
	DailyQuota    int64  `json:"daily_quota"`
	BaseGroup     string `json:"base_group"`
	FallbackGroup string `json:"fallback_group"`
	Quota         int64  `json:"quota,omitempty"`
}

// DailyQuotaStatus represents the daily quota status from new-api
type DailyQuotaStatus struct {
	UserID          int    `json:"user_id"`
	DailyQuota      int64  `json:"daily_quota"`
	DailyUsed       int64  `json:"daily_used"`
	DailyRemaining  int64  `json:"daily_remaining"`
	LastDailyReset  int64  `json:"last_daily_reset"`
	NeedsReset      bool   `json:"needs_reset"`
	CurrentGroup    string `json:"current_group"`
	BaseGroup       string `json:"base_group"`
	FallbackGroup   string `json:"fallback_group"`
	IsUsingFallback bool   `json:"is_using_fallback"`
}

// NewAPIUser represents a user from new-api
type NewAPIUser struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Quota     int64  `json:"quota"`
	UsedQuota int64  `json:"used_quota"`
	Group     string `json:"group"`
	Status    int    `json:"status"`
}

// NewAPIToken represents a token from new-api
type NewAPIToken struct {
	ID          int    `json:"id"`
	Key         string `json:"key"`
	RemainQuota int64  `json:"remain_quota"`
	ExpiredTime int64  `json:"expired_time"`
}

// NewSubscriptionUsecase creates a new subscription usecase
func NewSubscriptionUsecase(subRepo SubscriptionRepo, planRepo PlanRepo, newAPI NewAPIClient) *SubscriptionUsecase {
	return &SubscriptionUsecase{
		subRepo:  subRepo,
		planRepo: planRepo,
		newAPI:   newAPI,
	}
}

// Subscribe creates a new subscription for a user
func (uc *SubscriptionUsecase) Subscribe(ctx context.Context, userID int, planCode string) (*Subscription, error) {
	// Get plan
	plan, err := uc.planRepo.GetByCode(ctx, planCode)
	if err != nil {
		return nil, err
	}

	// Calculate expiry based on plan type
	now := time.Now()
	var expiresAt time.Time
	switch plan.Type {
	case PlanTypeMonthly:
		expiresAt = now.AddDate(0, 1, 0)
	case PlanTypeYearly:
		expiresAt = now.AddDate(1, 0, 0)
	default:
		expiresAt = now.AddDate(0, 1, 0)
	}

	// Create subscription (no daily quota state - managed by new-api)
	sub := &Subscription{
		UserID:       userID,
		PlanID:       plan.ID,
		Status:       SubscriptionStatusActive,
		StartedAt:    now,
		ExpiresAt:    expiresAt,
		AutoRenew:    true,
		CurrentQuota: plan.Quota,
		LastResetAt:  now,
		DailyQuota:   plan.DailyQuota,
	}

	if err := uc.subRepo.Create(ctx, sub); err != nil {
		return nil, err
	}

	// Sync subscription config to new-api (unified API)
	config := &SubscriptionConfig{
		DailyQuota:    plan.DailyQuota,
		BaseGroup:     plan.GroupName,
		FallbackGroup: plan.FallbackGroup,
		Quota:         plan.Quota,
	}
	if err := uc.newAPI.UpdateUserSubscriptionConfig(ctx, userID, config); err != nil {
		// Log error but don't fail - can be retried
		// TODO: Add retry mechanism
	}

	sub.Plan = plan
	return sub, nil
}

// GetUserSubscription returns the active subscription for a user
func (uc *SubscriptionUsecase) GetUserSubscription(ctx context.Context, userID int) (*Subscription, error) {
	return uc.subRepo.GetActiveByUserID(ctx, userID)
}

// Cancel cancels a subscription
func (uc *SubscriptionUsecase) Cancel(ctx context.Context, subID int64, reason string) error {
	sub, err := uc.subRepo.GetByID(ctx, subID)
	if err != nil {
		return err
	}

	now := time.Now()
	sub.Status = SubscriptionStatusCancelled
	sub.AutoRenew = false
	sub.CancelledAt = &now
	sub.CancelReason = reason

	return uc.subRepo.Update(ctx, sub)
}

// DeductQuota deducts quota from a subscription (total quota only)
// NOTE: Daily quota deduction is now handled by new-api during PostConsumeQuota.
// This method only updates the subscription's total quota for billing tracking.
func (uc *SubscriptionUsecase) DeductQuota(ctx context.Context, userID int, amount int64) error {
	sub, err := uc.subRepo.GetActiveByUserID(ctx, userID)
	if err != nil {
		return err
	}

	// Only deduct from total quota (daily quota is managed by new-api)
	return uc.subRepo.DeductQuota(ctx, sub.ID, amount)
}

// CheckQuotaStatus returns the current group and quota status for a user
// NOTE: Daily quota status is now fetched from new-api (single source of truth).
func (uc *SubscriptionUsecase) CheckQuotaStatus(ctx context.Context, userID int) (*QuotaStatus, error) {
	sub, err := uc.subRepo.GetActiveByUserID(ctx, userID)
	if err != nil {
		// No subscription - return free tier status
		return &QuotaStatus{
			UserID:       userID,
			HasQuota:     false,
			CurrentGroup: "free",
			IsFallback:   true,
		}, nil
	}

	// Load plan if not loaded
	if sub.Plan == nil {
		sub.Plan, _ = uc.planRepo.GetByID(ctx, sub.PlanID)
	}

	// Get daily quota status from new-api (single source of truth)
	dailyStatus, err := uc.newAPI.GetUserDailyQuotaStatus(ctx, userID)
	if err != nil {
		// Fallback to subscription-only data if new-api unavailable
		status := &QuotaStatus{
			UserID:     userID,
			HasQuota:   sub.HasQuota(),
			DailyQuota: sub.DailyQuota,
			TotalQuota: sub.CurrentQuota,
			TotalUsed:  sub.UsedQuota,
			ExpiresAt:  sub.ExpiresAt,
		}
		if sub.Plan != nil {
			status.PlanCode = sub.Plan.Code
			status.PlanName = sub.Plan.Name
			status.CurrentGroup = sub.Plan.GroupName
			status.FallbackGroup = sub.Plan.FallbackGroup
		}
		return status, nil
	}

	// Build status from new-api daily quota info + subscription info
	hasQuota := true
	if dailyStatus.DailyQuota > 0 {
		hasQuota = dailyStatus.DailyRemaining > 0
	}

	status := &QuotaStatus{
		UserID:           userID,
		HasQuota:         hasQuota,
		CurrentGroup:     dailyStatus.CurrentGroup,
		DailyQuota:       dailyStatus.DailyQuota,
		DailyUsed:        dailyStatus.DailyUsed,
		DailyRemaining:   dailyStatus.DailyRemaining,
		TotalQuota:       sub.CurrentQuota,
		TotalUsed:        sub.UsedQuota,
		ExpiresAt:        sub.ExpiresAt,
		IsFallback:       dailyStatus.IsUsingFallback,
		LastDailyResetAt: time.Unix(dailyStatus.LastDailyReset, 0),
	}

	if sub.Plan != nil {
		status.PlanCode = sub.Plan.Code
		status.PlanName = sub.Plan.Name
		status.FallbackGroup = sub.Plan.FallbackGroup
	}

	return status, nil
}

// QuotaStatus represents the current quota status for a user
type QuotaStatus struct {
	UserID           int       `json:"user_id"`
	PlanCode         string    `json:"plan_code"`
	PlanName         string    `json:"plan_name"`
	HasQuota         bool      `json:"has_quota"`
	CurrentGroup     string    `json:"current_group"`
	FallbackGroup    string    `json:"fallback_group,omitempty"`
	DailyQuota       int64     `json:"daily_quota"`
	DailyUsed        int64     `json:"daily_used"`
	DailyRemaining   int64     `json:"daily_remaining"`
	TotalQuota       int64     `json:"total_quota"`
	TotalUsed        int64     `json:"total_used"`
	ExpiresAt        time.Time `json:"expires_at"`
	IsFallback       bool      `json:"is_fallback"`
	LastDailyResetAt time.Time `json:"last_daily_reset_at"`
}

// resetDailyQuota resets the daily quota for a user via new-api
// NOTE: This is now a thin wrapper around new-api's ResetUserDailyQuota.
func (uc *SubscriptionUsecase) resetDailyQuota(ctx context.Context, sub *Subscription) error {
	return uc.newAPI.ResetUserDailyQuota(ctx, sub.UserID)
}

// ProcessDailyReset is now a no-op as daily quota reset is handled by new-api's cron job.
// This method is kept for backward compatibility but does nothing.
func (uc *SubscriptionUsecase) ProcessDailyReset(ctx context.Context) error {
	// NOTE: Daily quota reset is now handled by new-api's StartDailyQuotaResetCron().
	// This method is kept for backward compatibility.
	return nil
}

// ProcessRenewals processes subscription renewals
func (uc *SubscriptionUsecase) ProcessRenewals(ctx context.Context) error {
	// Get subscriptions expiring in the next 24 hours with auto_renew enabled
	before := time.Now().Add(24 * time.Hour)
	subs, err := uc.subRepo.ListForRenewal(ctx, before)
	if err != nil {
		return err
	}

	for _, sub := range subs {
		if err := uc.renewSubscription(ctx, sub); err != nil {
			// Log error and continue
			continue
		}
	}

	return nil
}

// renewSubscription renews a single subscription
func (uc *SubscriptionUsecase) renewSubscription(ctx context.Context, sub *Subscription) error {
	plan, err := uc.planRepo.GetByID(ctx, sub.PlanID)
	if err != nil {
		return err
	}

	// Calculate new expiry
	now := time.Now()
	var newExpiry time.Time
	switch plan.Type {
	case PlanTypeMonthly:
		newExpiry = sub.ExpiresAt.AddDate(0, 1, 0)
	case PlanTypeYearly:
		newExpiry = sub.ExpiresAt.AddDate(1, 0, 0)
	}

	// Update subscription (no daily quota state - managed by new-api)
	sub.ExpiresAt = newExpiry
	sub.CurrentQuota = plan.Quota
	sub.UsedQuota = 0
	sub.LastResetAt = now
	sub.DailyQuota = plan.DailyQuota

	if err := uc.subRepo.Update(ctx, sub); err != nil {
		return err
	}

	// Sync subscription config to new-api (unified API)
	config := &SubscriptionConfig{
		DailyQuota:    plan.DailyQuota,
		BaseGroup:     plan.GroupName,
		FallbackGroup: plan.FallbackGroup,
		Quota:         plan.Quota,
	}
	return uc.newAPI.UpdateUserSubscriptionConfig(ctx, sub.UserID, config)
}

// ProcessExpired marks expired subscriptions
func (uc *SubscriptionUsecase) ProcessExpired(ctx context.Context) error {
	subs, err := uc.subRepo.ListExpired(ctx)
	if err != nil {
		return err
	}

	for _, sub := range subs {
		sub.Status = SubscriptionStatusExpired
		if err := uc.subRepo.Update(ctx, sub); err != nil {
			continue
		}

		// Reset user to free tier via unified API
		config := &SubscriptionConfig{
			DailyQuota:    0,    // No daily quota for free tier
			BaseGroup:     "free",
			FallbackGroup: "",
			Quota:         0,
		}
		_ = uc.newAPI.UpdateUserSubscriptionConfig(ctx, sub.UserID, config)
	}

	return nil
}

// ListSubscriptions returns paginated subscriptions with optional filters
func (uc *SubscriptionUsecase) ListSubscriptions(ctx context.Context, page, pageSize int, status, planCode string) ([]*Subscription, int64, error) {
	return uc.subRepo.ListWithFilters(ctx, page, pageSize, status, planCode, 0)
}

// AdminListSubscriptions returns paginated subscriptions for admin with user filter
func (uc *SubscriptionUsecase) AdminListSubscriptions(ctx context.Context, page, pageSize int, status, planCode string, userID int) ([]*Subscription, int64, error) {
	return uc.subRepo.ListWithFilters(ctx, page, pageSize, status, planCode, userID)
}

// GetSubscriptionByID returns a subscription by its ID
func (uc *SubscriptionUsecase) GetSubscriptionByID(ctx context.Context, id int64) (*Subscription, error) {
	return uc.subRepo.GetByID(ctx, id)
}

// ResetSubscriptionDailyQuota manually resets daily quota for a subscription
func (uc *SubscriptionUsecase) ResetSubscriptionDailyQuota(ctx context.Context, id int64) error {
	sub, err := uc.subRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	return uc.resetDailyQuota(ctx, sub)
}

// StatsOverview represents subscription statistics
type StatsOverview struct {
	TotalSubscriptions   int64            `json:"total_subscriptions"`
	ActiveSubscriptions  int64            `json:"active_subscriptions"`
	ExpiredSubscriptions int64            `json:"expired_subscriptions"`
	TotalRevenue         int64            `json:"total_revenue"`
	ByPlan               map[string]int64 `json:"by_plan"`
	ByStatus             map[string]int64 `json:"by_status"`
}

// GetStatsOverview returns subscription statistics
func (uc *SubscriptionUsecase) GetStatsOverview(ctx context.Context) (*StatsOverview, error) {
	return uc.subRepo.GetStatsOverview(ctx)
}
