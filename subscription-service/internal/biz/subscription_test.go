package biz

import (
	"context"
	"errors"
	"testing"
	"time"
)

// MockNewAPIClient is a mock implementation of NewAPIClient for testing
type MockNewAPIClient struct {
	Users                     map[int]*NewAPIUser
	SubscriptionConfigs       map[int]*SubscriptionConfig
	DailyQuotaStatuses        map[int]*DailyQuotaStatus
	UpdateConfigCalled        bool
	GetDailyQuotaCalled       bool
	ResetDailyQuotaCalled     bool
	LastUpdatedConfig         *SubscriptionConfig
	ShouldFailUpdateConfig    bool
	ShouldFailGetDailyQuota   bool
	ShouldFailResetDailyQuota bool
}

func NewMockNewAPIClient() *MockNewAPIClient {
	return &MockNewAPIClient{
		Users:               make(map[int]*NewAPIUser),
		SubscriptionConfigs: make(map[int]*SubscriptionConfig),
		DailyQuotaStatuses:  make(map[int]*DailyQuotaStatus),
	}
}

func (m *MockNewAPIClient) GetUser(ctx context.Context, userID int) (*NewAPIUser, error) {
	if user, ok := m.Users[userID]; ok {
		return user, nil
	}
	return nil, errors.New("user not found")
}

func (m *MockNewAPIClient) UpdateUserQuota(ctx context.Context, userID int, quota int64) error {
	if user, ok := m.Users[userID]; ok {
		user.Quota = quota
		return nil
	}
	return nil // Allow update even if user not in mock
}

func (m *MockNewAPIClient) UpdateUserGroup(ctx context.Context, userID int, group string) error {
	if user, ok := m.Users[userID]; ok {
		user.Group = group
	}
	return nil
}

func (m *MockNewAPIClient) CreateToken(ctx context.Context, userID int, name string, quota int64, expiredTime int64) (*NewAPIToken, error) {
	return &NewAPIToken{
		ID:          1,
		Key:         "sk-test-token",
		RemainQuota: quota,
		ExpiredTime: expiredTime,
	}, nil
}

func (m *MockNewAPIClient) UpdateUserSubscriptionConfig(ctx context.Context, userID int, config *SubscriptionConfig) error {
	m.UpdateConfigCalled = true
	m.LastUpdatedConfig = config

	if m.ShouldFailUpdateConfig {
		return errors.New("failed to update subscription config")
	}

	m.SubscriptionConfigs[userID] = config
	return nil
}

func (m *MockNewAPIClient) GetUserDailyQuotaStatus(ctx context.Context, userID int) (*DailyQuotaStatus, error) {
	m.GetDailyQuotaCalled = true

	if m.ShouldFailGetDailyQuota {
		return nil, errors.New("failed to get daily quota status")
	}

	if status, ok := m.DailyQuotaStatuses[userID]; ok {
		return status, nil
	}

	// Return default status
	return &DailyQuotaStatus{
		UserID:          userID,
		DailyQuota:      1000000,
		DailyUsed:       0,
		DailyRemaining:  1000000,
		LastDailyReset:  time.Now().Unix(),
		NeedsReset:      false,
		CurrentGroup:    "pro",
		BaseGroup:       "pro",
		FallbackGroup:   "free",
		IsUsingFallback: false,
	}, nil
}

func (m *MockNewAPIClient) ResetUserDailyQuota(ctx context.Context, userID int) error {
	m.ResetDailyQuotaCalled = true

	if m.ShouldFailResetDailyQuota {
		return errors.New("failed to reset daily quota")
	}

	if status, ok := m.DailyQuotaStatuses[userID]; ok {
		status.DailyUsed = 0
		status.DailyRemaining = status.DailyQuota
		status.LastDailyReset = time.Now().Unix()
		status.NeedsReset = false
	}

	return nil
}

// MockSubscriptionRepo is a mock implementation of SubscriptionRepo
type MockSubscriptionRepo struct {
	Subscriptions map[int64]*Subscription
	NextID        int64
}

func NewMockSubscriptionRepo() *MockSubscriptionRepo {
	return &MockSubscriptionRepo{
		Subscriptions: make(map[int64]*Subscription),
		NextID:        1,
	}
}

func (r *MockSubscriptionRepo) Create(ctx context.Context, sub *Subscription) error {
	sub.ID = r.NextID
	r.NextID++
	r.Subscriptions[sub.ID] = sub
	return nil
}

func (r *MockSubscriptionRepo) Update(ctx context.Context, sub *Subscription) error {
	r.Subscriptions[sub.ID] = sub
	return nil
}

func (r *MockSubscriptionRepo) GetByID(ctx context.Context, id int64) (*Subscription, error) {
	if sub, ok := r.Subscriptions[id]; ok {
		return sub, nil
	}
	return nil, errors.New("subscription not found")
}

func (r *MockSubscriptionRepo) GetByUserID(ctx context.Context, userID int) (*Subscription, error) {
	for _, sub := range r.Subscriptions {
		if sub.UserID == userID {
			return sub, nil
		}
	}
	return nil, errors.New("subscription not found")
}

func (r *MockSubscriptionRepo) GetActiveByUserID(ctx context.Context, userID int) (*Subscription, error) {
	for _, sub := range r.Subscriptions {
		if sub.UserID == userID && sub.Status == SubscriptionStatusActive && time.Now().Before(sub.ExpiresAt) {
			return sub, nil
		}
	}
	return nil, errors.New("no active subscription found")
}

func (r *MockSubscriptionRepo) ListExpiring(ctx context.Context, before time.Time) ([]*Subscription, error) {
	var result []*Subscription
	now := time.Now()
	for _, sub := range r.Subscriptions {
		if sub.Status == SubscriptionStatusActive && sub.ExpiresAt.Before(before) && sub.ExpiresAt.After(now) {
			result = append(result, sub)
		}
	}
	return result, nil
}

func (r *MockSubscriptionRepo) ListExpired(ctx context.Context) ([]*Subscription, error) {
	var result []*Subscription
	now := time.Now()
	for _, sub := range r.Subscriptions {
		if sub.Status == SubscriptionStatusActive && sub.ExpiresAt.Before(now) {
			result = append(result, sub)
		}
	}
	return result, nil
}

func (r *MockSubscriptionRepo) ListForRenewal(ctx context.Context, before time.Time) ([]*Subscription, error) {
	var result []*Subscription
	now := time.Now()
	for _, sub := range r.Subscriptions {
		if sub.Status == SubscriptionStatusActive && sub.AutoRenew && sub.ExpiresAt.Before(before) && sub.ExpiresAt.After(now) {
			result = append(result, sub)
		}
	}
	return result, nil
}

func (r *MockSubscriptionRepo) ListActive(ctx context.Context) ([]*Subscription, error) {
	var result []*Subscription
	now := time.Now()
	for _, sub := range r.Subscriptions {
		if sub.Status == SubscriptionStatusActive && sub.ExpiresAt.After(now) {
			result = append(result, sub)
		}
	}
	return result, nil
}

func (r *MockSubscriptionRepo) ListWithFilters(ctx context.Context, page, pageSize int, status, planCode string, userID int) ([]*Subscription, int64, error) {
	var result []*Subscription
	for _, sub := range r.Subscriptions {
		result = append(result, sub)
	}
	return result, int64(len(result)), nil
}

func (r *MockSubscriptionRepo) DeductQuota(ctx context.Context, id int64, amount int64) error {
	if sub, ok := r.Subscriptions[id]; ok {
		if sub.CurrentQuota >= amount {
			sub.CurrentQuota -= amount
			sub.UsedQuota += amount
			return nil
		}
		return errors.New("insufficient quota")
	}
	return errors.New("subscription not found")
}

func (r *MockSubscriptionRepo) ResetQuota(ctx context.Context, id int64, quota int64) error {
	if sub, ok := r.Subscriptions[id]; ok {
		sub.CurrentQuota = quota
		sub.UsedQuota = 0
		return nil
	}
	return errors.New("subscription not found")
}

func (r *MockSubscriptionRepo) GetStatsOverview(ctx context.Context) (*StatsOverview, error) {
	return &StatsOverview{
		TotalSubscriptions:  int64(len(r.Subscriptions)),
		ActiveSubscriptions: int64(len(r.Subscriptions)),
		ByPlan:              make(map[string]int64),
		ByStatus:            make(map[string]int64),
	}, nil
}

// MockPlanRepo is a mock implementation of PlanRepo
type MockPlanRepo struct {
	Plans map[string]*Plan
}

func NewMockPlanRepo() *MockPlanRepo {
	repo := &MockPlanRepo{
		Plans: make(map[string]*Plan),
	}
	// Add default plans
	repo.Plans["pro_monthly"] = &Plan{
		ID:            1,
		Code:          "pro_monthly",
		Name:          "Pro Monthly",
		Type:          PlanTypeMonthly,
		Quota:         5000000,
		DailyQuota:    1000000,
		GroupName:     "pro",
		FallbackGroup: "free",
	}
	repo.Plans["free"] = &Plan{
		ID:        2,
		Code:      "free",
		Name:      "Free",
		Type:      PlanTypeMonthly, // Free tier uses monthly billing cycle
		GroupName: "free",
	}
	return repo
}

func (r *MockPlanRepo) Create(ctx context.Context, plan *Plan) error {
	r.Plans[plan.Code] = plan
	return nil
}

func (r *MockPlanRepo) Update(ctx context.Context, plan *Plan) error {
	r.Plans[plan.Code] = plan
	return nil
}

func (r *MockPlanRepo) GetByID(ctx context.Context, id int64) (*Plan, error) {
	for _, plan := range r.Plans {
		if plan.ID == id {
			return plan, nil
		}
	}
	return nil, errors.New("plan not found")
}

func (r *MockPlanRepo) GetByCode(ctx context.Context, code string) (*Plan, error) {
	if plan, ok := r.Plans[code]; ok {
		return plan, nil
	}
	return nil, errors.New("plan not found")
}

func (r *MockPlanRepo) ListActive(ctx context.Context) ([]*Plan, error) {
	var result []*Plan
	for _, plan := range r.Plans {
		result = append(result, plan)
	}
	return result, nil
}

func (r *MockPlanRepo) InitDefaultPlans(ctx context.Context) error {
	return nil
}

// --- Tests ---

func TestSubscribe_SyncsConfigToNewAPI(t *testing.T) {
	ctx := context.Background()

	subRepo := NewMockSubscriptionRepo()
	planRepo := NewMockPlanRepo()
	newAPI := NewMockNewAPIClient()

	uc := NewSubscriptionUsecase(subRepo, planRepo, newAPI)

	// Subscribe user
	sub, err := uc.Subscribe(ctx, 1, "pro_monthly")
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	// Verify subscription created
	if sub == nil {
		t.Fatal("Expected subscription to be created")
	}
	if sub.UserID != 1 {
		t.Errorf("Expected UserID=1, got %d", sub.UserID)
	}
	if sub.Status != SubscriptionStatusActive {
		t.Errorf("Expected status=active, got %s", sub.Status)
	}

	// Verify new-api was called with correct config
	if !newAPI.UpdateConfigCalled {
		t.Error("Expected UpdateUserSubscriptionConfig to be called")
	}

	config := newAPI.LastUpdatedConfig
	if config == nil {
		t.Fatal("Expected config to be set")
	}

	if config.DailyQuota != 1000000 {
		t.Errorf("Expected DailyQuota=1000000, got %d", config.DailyQuota)
	}
	if config.BaseGroup != "pro" {
		t.Errorf("Expected BaseGroup=pro, got %s", config.BaseGroup)
	}
	if config.FallbackGroup != "free" {
		t.Errorf("Expected FallbackGroup=free, got %s", config.FallbackGroup)
	}
}

func TestDeductQuota_OnlyDeductsTotalQuota(t *testing.T) {
	ctx := context.Background()

	subRepo := NewMockSubscriptionRepo()
	planRepo := NewMockPlanRepo()
	newAPI := NewMockNewAPIClient()

	uc := NewSubscriptionUsecase(subRepo, planRepo, newAPI)

	// Create subscription first
	sub, _ := uc.Subscribe(ctx, 1, "pro_monthly")
	initialQuota := sub.CurrentQuota

	// Deduct quota
	err := uc.DeductQuota(ctx, 1, 100000)
	if err != nil {
		t.Fatalf("DeductQuota failed: %v", err)
	}

	// Verify total quota was deducted
	updatedSub, _ := subRepo.GetByID(ctx, sub.ID)
	expectedQuota := initialQuota - 100000
	if updatedSub.CurrentQuota != expectedQuota {
		t.Errorf("Expected CurrentQuota=%d, got %d", expectedQuota, updatedSub.CurrentQuota)
	}
}

func TestCheckQuotaStatus_FetchesFromNewAPI(t *testing.T) {
	ctx := context.Background()

	subRepo := NewMockSubscriptionRepo()
	planRepo := NewMockPlanRepo()
	newAPI := NewMockNewAPIClient()

	// Set up daily quota status in new-api
	newAPI.DailyQuotaStatuses[1] = &DailyQuotaStatus{
		UserID:          1,
		DailyQuota:      1000000,
		DailyUsed:       250000,
		DailyRemaining:  750000,
		CurrentGroup:    "pro",
		BaseGroup:       "pro",
		FallbackGroup:   "free",
		IsUsingFallback: false,
	}

	uc := NewSubscriptionUsecase(subRepo, planRepo, newAPI)

	// Create subscription
	uc.Subscribe(ctx, 1, "pro_monthly")

	// Check quota status
	status, err := uc.CheckQuotaStatus(ctx, 1)
	if err != nil {
		t.Fatalf("CheckQuotaStatus failed: %v", err)
	}

	// Verify new-api was called
	if !newAPI.GetDailyQuotaCalled {
		t.Error("Expected GetUserDailyQuotaStatus to be called")
	}

	// Verify status reflects new-api data
	if status.DailyUsed != 250000 {
		t.Errorf("Expected DailyUsed=250000, got %d", status.DailyUsed)
	}
	if status.DailyRemaining != 750000 {
		t.Errorf("Expected DailyRemaining=750000, got %d", status.DailyRemaining)
	}
	if status.CurrentGroup != "pro" {
		t.Errorf("Expected CurrentGroup=pro, got %s", status.CurrentGroup)
	}
}

func TestCheckQuotaStatus_FallbackWhenNewAPIUnavailable(t *testing.T) {
	ctx := context.Background()

	subRepo := NewMockSubscriptionRepo()
	planRepo := NewMockPlanRepo()
	newAPI := NewMockNewAPIClient()
	newAPI.ShouldFailGetDailyQuota = true

	uc := NewSubscriptionUsecase(subRepo, planRepo, newAPI)

	// Create subscription
	uc.Subscribe(ctx, 1, "pro_monthly")

	// Check quota status - should fall back to subscription data
	status, err := uc.CheckQuotaStatus(ctx, 1)
	if err != nil {
		t.Fatalf("CheckQuotaStatus failed: %v", err)
	}

	// Should still have some data from subscription
	if status.UserID != 1 {
		t.Errorf("Expected UserID=1, got %d", status.UserID)
	}
	if status.PlanCode != "pro_monthly" {
		t.Errorf("Expected PlanCode=pro_monthly, got %s", status.PlanCode)
	}
}

func TestResetDailyQuota_DelegatesToNewAPI(t *testing.T) {
	ctx := context.Background()

	subRepo := NewMockSubscriptionRepo()
	planRepo := NewMockPlanRepo()
	newAPI := NewMockNewAPIClient()

	uc := NewSubscriptionUsecase(subRepo, planRepo, newAPI)

	// Create subscription
	sub, _ := uc.Subscribe(ctx, 1, "pro_monthly")

	// Reset daily quota
	err := uc.ResetSubscriptionDailyQuota(ctx, sub.ID)
	if err != nil {
		t.Fatalf("ResetSubscriptionDailyQuota failed: %v", err)
	}

	// Verify new-api was called
	if !newAPI.ResetDailyQuotaCalled {
		t.Error("Expected ResetUserDailyQuota to be called")
	}
}

func TestProcessDailyReset_IsNoOp(t *testing.T) {
	ctx := context.Background()

	subRepo := NewMockSubscriptionRepo()
	planRepo := NewMockPlanRepo()
	newAPI := NewMockNewAPIClient()

	uc := NewSubscriptionUsecase(subRepo, planRepo, newAPI)

	// ProcessDailyReset should be a no-op
	err := uc.ProcessDailyReset(ctx)
	if err != nil {
		t.Fatalf("ProcessDailyReset failed: %v", err)
	}

	// No new-api calls should be made (handled by new-api cron)
	if newAPI.ResetDailyQuotaCalled {
		t.Error("Expected ResetUserDailyQuota NOT to be called (handled by new-api cron)")
	}
}

func TestProcessExpired_SyncsToNewAPI(t *testing.T) {
	ctx := context.Background()

	subRepo := NewMockSubscriptionRepo()
	planRepo := NewMockPlanRepo()
	newAPI := NewMockNewAPIClient()

	// Create expired subscription directly in repo
	expiredSub := &Subscription{
		ID:        1,
		UserID:    1,
		PlanID:    1,
		Status:    SubscriptionStatusActive,
		ExpiresAt: time.Now().Add(-24 * time.Hour), // Expired yesterday
	}
	subRepo.Subscriptions[1] = expiredSub

	uc := NewSubscriptionUsecase(subRepo, planRepo, newAPI)

	// Process expired
	err := uc.ProcessExpired(ctx)
	if err != nil {
		t.Fatalf("ProcessExpired failed: %v", err)
	}

	// Verify subscription status updated
	updatedSub, _ := subRepo.GetByID(ctx, 1)
	if updatedSub.Status != SubscriptionStatusExpired {
		t.Errorf("Expected status=expired, got %s", updatedSub.Status)
	}

	// Verify new-api was called to reset to free tier
	if !newAPI.UpdateConfigCalled {
		t.Error("Expected UpdateUserSubscriptionConfig to be called")
	}

	config := newAPI.LastUpdatedConfig
	if config.BaseGroup != "free" {
		t.Errorf("Expected BaseGroup=free, got %s", config.BaseGroup)
	}
	if config.DailyQuota != 0 {
		t.Errorf("Expected DailyQuota=0, got %d", config.DailyQuota)
	}
}

func TestNoSubscription_ReturnsFreeStatus(t *testing.T) {
	ctx := context.Background()

	subRepo := NewMockSubscriptionRepo()
	planRepo := NewMockPlanRepo()
	newAPI := NewMockNewAPIClient()

	uc := NewSubscriptionUsecase(subRepo, planRepo, newAPI)

	// Check quota for user without subscription
	status, err := uc.CheckQuotaStatus(ctx, 999)
	if err != nil {
		t.Fatalf("CheckQuotaStatus failed: %v", err)
	}

	// Should return free tier status
	if status.UserID != 999 {
		t.Errorf("Expected UserID=999, got %d", status.UserID)
	}
	if status.CurrentGroup != "free" {
		t.Errorf("Expected CurrentGroup=free, got %s", status.CurrentGroup)
	}
	if status.HasQuota != false {
		t.Error("Expected HasQuota=false for free tier")
	}
	if status.IsFallback != true {
		t.Error("Expected IsFallback=true for free tier")
	}
}

// Benchmark tests
func BenchmarkSubscribe(b *testing.B) {
	ctx := context.Background()

	subRepo := NewMockSubscriptionRepo()
	planRepo := NewMockPlanRepo()
	newAPI := NewMockNewAPIClient()

	uc := NewSubscriptionUsecase(subRepo, planRepo, newAPI)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		uc.Subscribe(ctx, i, "pro_monthly")
	}
}

func BenchmarkCheckQuotaStatus(b *testing.B) {
	ctx := context.Background()

	subRepo := NewMockSubscriptionRepo()
	planRepo := NewMockPlanRepo()
	newAPI := NewMockNewAPIClient()

	uc := NewSubscriptionUsecase(subRepo, planRepo, newAPI)

	// Create a subscription
	uc.Subscribe(ctx, 1, "pro_monthly")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		uc.CheckQuotaStatus(ctx, 1)
	}
}
