package types

import (
	"encoding/json"
	"testing"
	"time"
)

func TestUserQuotaStatus_JSONMarshaling(t *testing.T) {
	status := UserQuotaStatus{
		UserID:          1,
		Username:        "testuser",
		Email:           "test@example.com",
		Quota:           5000000,
		UsedQuota:       1000000,
		Group:           "pro",
		DailyQuota:      1000000,
		DailyUsed:       250000,
		DailyRemaining:  750000,
		LastDailyReset:  time.Now().Unix(),
		NeedsReset:      false,
		BaseGroup:       "pro",
		FallbackGroup:   "free",
		IsUsingFallback: false,
		Status:          1,
	}

	// Marshal
	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Unmarshal
	var decoded UserQuotaStatus
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify
	if decoded.UserID != status.UserID {
		t.Errorf("UserID mismatch: got %d, want %d", decoded.UserID, status.UserID)
	}
	if decoded.DailyQuota != status.DailyQuota {
		t.Errorf("DailyQuota mismatch: got %d, want %d", decoded.DailyQuota, status.DailyQuota)
	}
	if decoded.BaseGroup != status.BaseGroup {
		t.Errorf("BaseGroup mismatch: got %s, want %s", decoded.BaseGroup, status.BaseGroup)
	}
}

func TestSubscriptionConfig_Validation(t *testing.T) {
	tests := []struct {
		name   string
		config SubscriptionConfig
		valid  bool
	}{
		{
			name: "valid full config",
			config: SubscriptionConfig{
				DailyQuota:    1000000,
				BaseGroup:     "pro",
				FallbackGroup: "free",
				Quota:         5000000,
			},
			valid: true,
		},
		{
			name: "valid unlimited config",
			config: SubscriptionConfig{
				DailyQuota: 0,
				BaseGroup:  "unlimited",
			},
			valid: true,
		},
		{
			name: "valid free tier",
			config: SubscriptionConfig{
				BaseGroup: "free",
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON round-trip
			data, err := json.Marshal(tt.config)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			var decoded SubscriptionConfig
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			if decoded.DailyQuota != tt.config.DailyQuota {
				t.Errorf("DailyQuota mismatch")
			}
			if decoded.BaseGroup != tt.config.BaseGroup {
				t.Errorf("BaseGroup mismatch")
			}
		})
	}
}

func TestDailyQuotaInfo_Calculations(t *testing.T) {
	tests := []struct {
		name       string
		info       DailyQuotaInfo
		hasQuota   bool
		isFallback bool
	}{
		{
			name: "has quota remaining",
			info: DailyQuotaInfo{
				DailyQuota:      1000000,
				DailyUsed:       500000,
				DailyRemaining:  500000,
				CurrentGroup:    "pro",
				BaseGroup:       "pro",
				FallbackGroup:   "free",
				IsUsingFallback: false,
			},
			hasQuota:   true,
			isFallback: false,
		},
		{
			name: "quota exhausted using fallback",
			info: DailyQuotaInfo{
				DailyQuota:      1000000,
				DailyUsed:       1000000,
				DailyRemaining:  0,
				CurrentGroup:    "free",
				BaseGroup:       "pro",
				FallbackGroup:   "free",
				IsUsingFallback: true,
			},
			hasQuota:   false,
			isFallback: true,
		},
		{
			name: "unlimited quota",
			info: DailyQuotaInfo{
				DailyQuota:     0,
				DailyUsed:      5000000,
				DailyRemaining: -1,
				CurrentGroup:   "unlimited",
				BaseGroup:      "unlimited",
			},
			hasQuota:   true,
			isFallback: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasQuota := tt.info.DailyQuota <= 0 || tt.info.DailyRemaining > 0
			if hasQuota != tt.hasQuota {
				t.Errorf("HasQuota = %v, want %v", hasQuota, tt.hasQuota)
			}

			if tt.info.IsUsingFallback != tt.isFallback {
				t.Errorf("IsUsingFallback = %v, want %v", tt.info.IsUsingFallback, tt.isFallback)
			}
		})
	}
}

func TestUser_FieldsPresent(t *testing.T) {
	user := User{
		ID:             1,
		Username:       "testuser",
		Email:          "test@example.com",
		Status:         1,
		Role:           1,
		Group:          "pro",
		Quota:          5000000,
		UsedQuota:      1000000,
		DailyQuota:     1000000,
		DailyUsed:      500000,
		LastDailyReset: time.Now().Unix(),
		BaseGroup:      "pro",
		FallbackGroup:  "free",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Verify new fields exist
	if user.DailyQuota != 1000000 {
		t.Errorf("DailyQuota = %d, want 1000000", user.DailyQuota)
	}
	if user.DailyUsed != 500000 {
		t.Errorf("DailyUsed = %d, want 500000", user.DailyUsed)
	}
	if user.BaseGroup != "pro" {
		t.Errorf("BaseGroup = %s, want pro", user.BaseGroup)
	}
	if user.FallbackGroup != "free" {
		t.Errorf("FallbackGroup = %s, want free", user.FallbackGroup)
	}
}

func BenchmarkUserQuotaStatus_Marshal(b *testing.B) {
	status := UserQuotaStatus{
		UserID:         1,
		Quota:          5000000,
		DailyQuota:     1000000,
		DailyUsed:      500000,
		DailyRemaining: 500000,
		Group:          "pro",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		json.Marshal(status)
	}
}

func BenchmarkSubscriptionConfig_Unmarshal(b *testing.B) {
	data := []byte(`{"daily_quota":1000000,"base_group":"pro","fallback_group":"free","quota":5000000}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var config SubscriptionConfig
		json.Unmarshal(data, &config)
	}
}
