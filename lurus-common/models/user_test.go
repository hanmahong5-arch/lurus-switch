package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestUser_JSON(t *testing.T) {
	user := &User{
		ID:         "user-123",
		Username:   "testuser",
		Email:      "test@example.com",
		Plan:       PlanPro,
		QuotaTotal: 1000000,
		QuotaUsed:  50000,
		IsAdmin:    false,
		IsDisabled: false,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	data, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("Failed to marshal user: %v", err)
	}

	var decoded User
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal user: %v", err)
	}

	if decoded.ID != user.ID {
		t.Errorf("ID mismatch: got %s, want %s", decoded.ID, user.ID)
	}
	if decoded.Username != user.Username {
		t.Errorf("Username mismatch: got %s, want %s", decoded.Username, user.Username)
	}
	if decoded.Plan != user.Plan {
		t.Errorf("Plan mismatch: got %s, want %s", decoded.Plan, user.Plan)
	}
	if decoded.QuotaTotal != user.QuotaTotal {
		t.Errorf("QuotaTotal mismatch: got %f, want %f", decoded.QuotaTotal, user.QuotaTotal)
	}
}

func TestUserQuota(t *testing.T) {
	quota := &UserQuota{
		UserID:      "user-123",
		QuotaTotal:  1000000,
		QuotaUsed:   250000,
		QuotaRemain: 750000,
		DailyLimit:  100000,
		DailyUsed:   25000,
		ResetAt:     time.Now().Add(24 * time.Hour),
		LastUpdated: time.Now(),
	}

	data, err := json.Marshal(quota)
	if err != nil {
		t.Fatalf("Failed to marshal quota: %v", err)
	}

	var decoded UserQuota
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal quota: %v", err)
	}

	if decoded.QuotaRemain != quota.QuotaRemain {
		t.Errorf("QuotaRemain mismatch: got %f, want %f", decoded.QuotaRemain, quota.QuotaRemain)
	}
}

func TestPlanConstants(t *testing.T) {
	if PlanFree != "free" {
		t.Errorf("PlanFree should be 'free', got '%s'", PlanFree)
	}
	if PlanBasic != "basic" {
		t.Errorf("PlanBasic should be 'basic', got '%s'", PlanBasic)
	}
	if PlanPro != "pro" {
		t.Errorf("PlanPro should be 'pro', got '%s'", PlanPro)
	}
	if PlanEnterprise != "enterprise" {
		t.Errorf("PlanEnterprise should be 'enterprise', got '%s'", PlanEnterprise)
	}
}

func TestDevice_JSON(t *testing.T) {
	device := &Device{
		ID:            "device-1",
		UserID:        "user-123",
		DeviceID:      "abc-123-def",
		DeviceName:    "My Laptop",
		DeviceType:    DeviceTypeDesktop,
		ClientVersion: "1.0.0",
		LastSeenAt:    time.Now(),
		LastIP:        "192.168.1.1",
		CreatedAt:     time.Now(),
	}

	data, err := json.Marshal(device)
	if err != nil {
		t.Fatalf("Failed to marshal device: %v", err)
	}

	var decoded Device
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal device: %v", err)
	}

	if decoded.DeviceID != device.DeviceID {
		t.Errorf("DeviceID mismatch: got %s, want %s", decoded.DeviceID, device.DeviceID)
	}
	if decoded.DeviceType != device.DeviceType {
		t.Errorf("DeviceType mismatch: got %s, want %s", decoded.DeviceType, device.DeviceType)
	}
}

func TestDeviceTypeConstants(t *testing.T) {
	if DeviceTypeDesktop != "desktop" {
		t.Errorf("DeviceTypeDesktop should be 'desktop', got '%s'", DeviceTypeDesktop)
	}
	if DeviceTypeMobile != "mobile" {
		t.Errorf("DeviceTypeMobile should be 'mobile', got '%s'", DeviceTypeMobile)
	}
	if DeviceTypeCLI != "cli" {
		t.Errorf("DeviceTypeCLI should be 'cli', got '%s'", DeviceTypeCLI)
	}
	if DeviceTypeWeb != "web" {
		t.Errorf("DeviceTypeWeb should be 'web', got '%s'", DeviceTypeWeb)
	}
}

func TestPresenceStatus(t *testing.T) {
	if PresenceOnline != "online" {
		t.Errorf("PresenceOnline should be 'online', got '%s'", PresenceOnline)
	}
	if PresenceOffline != "offline" {
		t.Errorf("PresenceOffline should be 'offline', got '%s'", PresenceOffline)
	}
	if PresenceAway != "away" {
		t.Errorf("PresenceAway should be 'away', got '%s'", PresenceAway)
	}
}

func TestPresence_JSON(t *testing.T) {
	presence := &Presence{
		UserID:        "user-123",
		DeviceID:      "device-1",
		DeviceType:    DeviceTypeDesktop,
		Status:        PresenceOnline,
		ClientVersion: "1.0.0",
		LastSeenAt:    time.Now(),
	}

	data, err := json.Marshal(presence)
	if err != nil {
		t.Fatalf("Failed to marshal presence: %v", err)
	}

	var decoded Presence
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal presence: %v", err)
	}

	if decoded.Status != presence.Status {
		t.Errorf("Status mismatch: got %s, want %s", decoded.Status, presence.Status)
	}
}
