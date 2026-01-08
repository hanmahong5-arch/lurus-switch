package biz

import (
	"context"
	"errors"
	"testing"
	"time"
)

// MockBillingRepo is a mock implementation of BillingRepo
type MockBillingRepo struct {
	users     map[string]*User
	usages    []*UsageRecord
	quotaUsed map[string]int64
}

func NewMockBillingRepo() *MockBillingRepo {
	return &MockBillingRepo{
		users:     make(map[string]*User),
		usages:    make([]*UsageRecord, 0),
		quotaUsed: make(map[string]int64),
	}
}

func (m *MockBillingRepo) GetUser(ctx context.Context, userID string) (*User, error) {
	user, ok := m.users[userID]
	if !ok {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (m *MockBillingRepo) CreateUser(ctx context.Context, user *User) error {
	m.users[user.ID] = user
	return nil
}

func (m *MockBillingRepo) UpdateUser(ctx context.Context, user *User) error {
	m.users[user.ID] = user
	return nil
}

func (m *MockBillingRepo) RecordUsage(ctx context.Context, usage *UsageRecord) error {
	m.usages = append(m.usages, usage)
	return nil
}

func (m *MockBillingRepo) GetUsageStats(ctx context.Context, userID string, start, end time.Time) (*UsageStats, error) {
	var stats UsageStats
	stats.UserID = userID
	stats.PeriodStart = start
	stats.PeriodEnd = end

	for _, u := range m.usages {
		if u.UserID == userID && u.CreatedAt.After(start) && u.CreatedAt.Before(end) {
			stats.TotalRequests++
			stats.InputTokens += int64(u.InputTokens)
			stats.OutputTokens += int64(u.OutputTokens)
			stats.TotalTokens += int64(u.InputTokens + u.OutputTokens)
			stats.TotalCost += u.TotalCost
		}
	}

	return &stats, nil
}

func (m *MockBillingRepo) GetQuotaUsed(ctx context.Context, userID string) (int64, error) {
	return m.quotaUsed[userID], nil
}

func (m *MockBillingRepo) IncrementQuota(ctx context.Context, userID string, tokens int64) error {
	m.quotaUsed[userID] += tokens
	return nil
}

func (m *MockBillingRepo) ResetQuota(ctx context.Context, userID string) error {
	m.quotaUsed[userID] = 0
	return nil
}

func TestBillingUsecase_CheckBalance_NewUser(t *testing.T) {
	repo := NewMockBillingRepo()
	uc := NewBillingUsecase(repo, 1000000, 100000, 500000, Pricing{InputTokens: 3.0, OutputTokens: 15.0}, nil)

	ctx := context.Background()
	result, err := uc.CheckBalance(ctx, "new-user")
	if err != nil {
		t.Fatalf("CheckBalance failed: %v", err)
	}

	if !result.Allowed {
		t.Error("New user should be allowed")
	}
	if result.QuotaLimit != 1000000 {
		t.Errorf("Expected quota limit 1000000, got %d", result.QuotaLimit)
	}
	if result.QuotaUsed != 0 {
		t.Errorf("Expected quota used 0, got %d", result.QuotaUsed)
	}
}

func TestBillingUsecase_CheckBalance_ExistingUser(t *testing.T) {
	repo := NewMockBillingRepo()
	repo.users["user-1"] = &User{
		ID:           "user-1",
		Plan:         "free",
		Balance:      0,
		QuotaLimit:   500000,
		QuotaUsed:    100000,
		QuotaResetAt: time.Now().Add(24 * time.Hour),
	}

	uc := NewBillingUsecase(repo, 1000000, 100000, 500000, Pricing{InputTokens: 3.0, OutputTokens: 15.0}, nil)

	ctx := context.Background()
	result, err := uc.CheckBalance(ctx, "user-1")
	if err != nil {
		t.Fatalf("CheckBalance failed: %v", err)
	}

	if !result.Allowed {
		t.Error("User with quota remaining should be allowed")
	}
	if result.QuotaRemaining != 400000 {
		t.Errorf("Expected quota remaining 400000, got %d", result.QuotaRemaining)
	}
}

func TestBillingUsecase_CheckBalance_QuotaExceeded(t *testing.T) {
	repo := NewMockBillingRepo()
	repo.users["user-exceeded"] = &User{
		ID:           "user-exceeded",
		Plan:         "free",
		Balance:      0,
		QuotaLimit:   100000,
		QuotaUsed:    150000,
		QuotaResetAt: time.Now().Add(24 * time.Hour),
	}

	uc := NewBillingUsecase(repo, 1000000, 100000, 500000, Pricing{InputTokens: 3.0, OutputTokens: 15.0}, nil)

	ctx := context.Background()
	result, err := uc.CheckBalance(ctx, "user-exceeded")
	if err != nil {
		t.Fatalf("CheckBalance failed: %v", err)
	}

	if result.Allowed {
		t.Error("User with exceeded quota should not be allowed")
	}
	if result.Message != "Quota exceeded" {
		t.Errorf("Expected 'Quota exceeded' message, got '%s'", result.Message)
	}
}

func TestBillingUsecase_CheckBalance_PaidPlanNoBalance(t *testing.T) {
	repo := NewMockBillingRepo()
	repo.users["paid-user"] = &User{
		ID:           "paid-user",
		Plan:         "pro",
		Balance:      0,
		QuotaLimit:   1000000,
		QuotaUsed:    100000,
		QuotaResetAt: time.Now().Add(24 * time.Hour),
	}

	uc := NewBillingUsecase(repo, 1000000, 100000, 500000, Pricing{InputTokens: 3.0, OutputTokens: 15.0}, nil)

	ctx := context.Background()
	result, err := uc.CheckBalance(ctx, "paid-user")
	if err != nil {
		t.Fatalf("CheckBalance failed: %v", err)
	}

	if result.Allowed {
		t.Error("Paid user with no balance should not be allowed")
	}
	if result.Message != "Insufficient balance" {
		t.Errorf("Expected 'Insufficient balance' message, got '%s'", result.Message)
	}
}

func TestBillingUsecase_RecordUsage(t *testing.T) {
	repo := NewMockBillingRepo()
	repo.users["user-1"] = &User{
		ID:         "user-1",
		Plan:       "free",
		QuotaLimit: 1000000,
		QuotaUsed:  0,
	}

	uc := NewBillingUsecase(repo, 1000000, 100000, 500000, Pricing{InputTokens: 3.0, OutputTokens: 15.0}, nil)

	ctx := context.Background()
	usage := &UsageRecord{
		ID:           "usage-1",
		UserID:       "user-1",
		TraceID:      "trace-1",
		Platform:     "claude",
		Model:        "claude-3-opus",
		Provider:     "anthropic",
		InputTokens:  1000,
		OutputTokens: 500,
		CreatedAt:    time.Now(),
	}

	err := uc.RecordUsage(ctx, usage)
	if err != nil {
		t.Fatalf("RecordUsage failed: %v", err)
	}

	// Check that usage was recorded
	if len(repo.usages) != 1 {
		t.Errorf("Expected 1 usage record, got %d", len(repo.usages))
	}

	// Check that quota was incremented
	if repo.quotaUsed["user-1"] != 1500 {
		t.Errorf("Expected quota used 1500, got %d", repo.quotaUsed["user-1"])
	}

	// Check that cost was calculated
	if usage.TotalCost == 0 {
		t.Error("Expected TotalCost to be calculated")
	}
}

func TestBillingUsecase_RecordUsage_WithPreCalculatedCost(t *testing.T) {
	repo := NewMockBillingRepo()
	uc := NewBillingUsecase(repo, 1000000, 100000, 500000, Pricing{InputTokens: 3.0, OutputTokens: 15.0}, nil)

	ctx := context.Background()
	usage := &UsageRecord{
		ID:           "usage-2",
		UserID:       "user-1",
		InputTokens:  1000,
		OutputTokens: 500,
		TotalCost:    0.05, // Pre-calculated
		CreatedAt:    time.Now(),
	}

	err := uc.RecordUsage(ctx, usage)
	if err != nil {
		t.Fatalf("RecordUsage failed: %v", err)
	}

	// Cost should remain as pre-calculated
	if usage.TotalCost != 0.05 {
		t.Errorf("Expected TotalCost 0.05, got %f", usage.TotalCost)
	}
}

func TestBillingUsecase_GetUsageStats(t *testing.T) {
	repo := NewMockBillingRepo()
	now := time.Now()
	repo.usages = []*UsageRecord{
		{UserID: "user-1", InputTokens: 1000, OutputTokens: 500, TotalCost: 0.05, CreatedAt: now.Add(-1 * time.Hour)},
		{UserID: "user-1", InputTokens: 2000, OutputTokens: 1000, TotalCost: 0.10, CreatedAt: now.Add(-2 * time.Hour)},
		{UserID: "user-2", InputTokens: 500, OutputTokens: 250, TotalCost: 0.025, CreatedAt: now.Add(-1 * time.Hour)},
	}

	uc := NewBillingUsecase(repo, 1000000, 100000, 500000, Pricing{InputTokens: 3.0, OutputTokens: 15.0}, nil)

	ctx := context.Background()
	start := now.Add(-24 * time.Hour)
	end := now

	stats, err := uc.GetUsageStats(ctx, "user-1", start, end)
	if err != nil {
		t.Fatalf("GetUsageStats failed: %v", err)
	}

	if stats.TotalRequests != 2 {
		t.Errorf("Expected 2 requests, got %d", stats.TotalRequests)
	}
	if stats.InputTokens != 3000 {
		t.Errorf("Expected 3000 input tokens, got %d", stats.InputTokens)
	}
}

func TestBillingUsecase_UpdateQuota(t *testing.T) {
	repo := NewMockBillingRepo()
	repo.users["user-1"] = &User{
		ID:         "user-1",
		Plan:       "free",
		QuotaLimit: 100000,
		QuotaUsed:  0,
	}

	uc := NewBillingUsecase(repo, 1000000, 100000, 500000, Pricing{InputTokens: 3.0, OutputTokens: 15.0}, nil)

	ctx := context.Background()
	err := uc.UpdateQuota(ctx, "user-1", 500000)
	if err != nil {
		t.Fatalf("UpdateQuota failed: %v", err)
	}

	user, _ := repo.GetUser(ctx, "user-1")
	if user.QuotaLimit != 500000 {
		t.Errorf("Expected quota limit 500000, got %d", user.QuotaLimit)
	}
}

func TestBillingUsecase_AddBalance(t *testing.T) {
	repo := NewMockBillingRepo()
	repo.users["user-1"] = &User{
		ID:      "user-1",
		Plan:    "pro",
		Balance: 10.0,
	}

	uc := NewBillingUsecase(repo, 1000000, 100000, 500000, Pricing{InputTokens: 3.0, OutputTokens: 15.0}, nil)

	ctx := context.Background()
	err := uc.AddBalance(ctx, "user-1", 25.50)
	if err != nil {
		t.Fatalf("AddBalance failed: %v", err)
	}

	user, _ := repo.GetUser(ctx, "user-1")
	if user.Balance != 35.50 {
		t.Errorf("Expected balance 35.50, got %f", user.Balance)
	}
}

func TestBillingUsecase_GetUser_NotFound(t *testing.T) {
	repo := NewMockBillingRepo()
	uc := NewBillingUsecase(repo, 1000000, 100000, 500000, Pricing{InputTokens: 3.0, OutputTokens: 15.0}, nil)

	ctx := context.Background()
	_, err := uc.GetUser(ctx, "nonexistent")
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}
}

func TestPricing_CalculateCost(t *testing.T) {
	// Pricing is $3/1M input, $15/1M output
	uc := NewBillingUsecase(nil, 1000000, 100000, 500000, Pricing{InputTokens: 3.0, OutputTokens: 15.0}, nil)

	// 1M input tokens = $3
	cost := uc.calculateCost(1000000, 0)
	if cost != 3.0 {
		t.Errorf("Expected cost 3.0, got %f", cost)
	}

	// 1M output tokens = $15
	cost = uc.calculateCost(0, 1000000)
	if cost != 15.0 {
		t.Errorf("Expected cost 15.0, got %f", cost)
	}

	// 1000 input + 500 output
	cost = uc.calculateCost(1000, 500)
	expected := (1000.0/1000000*3.0) + (500.0/1000000*15.0) // 0.003 + 0.0075 = 0.0105
	// Use tolerance for floating point comparison
	tolerance := 0.000001
	diff := cost - expected
	if diff < 0 {
		diff = -diff
	}
	if diff > tolerance {
		t.Errorf("Expected cost %f, got %f", expected, cost)
	}
}

func TestBalanceCheckResult(t *testing.T) {
	result := &BalanceCheckResult{
		Allowed:        true,
		Balance:        100.0,
		QuotaLimit:     1000000,
		QuotaUsed:      250000,
		QuotaRemaining: 750000,
	}

	if !result.Allowed {
		t.Error("Expected Allowed to be true")
	}
	if result.QuotaRemaining != 750000 {
		t.Errorf("Expected QuotaRemaining 750000, got %d", result.QuotaRemaining)
	}
}

func TestUsageStats(t *testing.T) {
	stats := &UsageStats{
		UserID:        "user-1",
		TotalRequests: 100,
		TotalTokens:   500000,
		TotalCost:     25.0,
		InputTokens:   300000,
		OutputTokens:  200000,
		PeriodStart:   time.Now().Add(-24 * time.Hour),
		PeriodEnd:     time.Now(),
	}

	if stats.TotalTokens != stats.InputTokens+stats.OutputTokens {
		t.Error("TotalTokens should equal InputTokens + OutputTokens")
	}
}
