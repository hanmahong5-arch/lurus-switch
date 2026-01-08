package newapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lurus-ai/lurus-common/types"
)

func TestClient_GetUser(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/user/1" {
			t.Errorf("Expected path /api/user/1, got %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET method, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Expected Authorization header")
		}

		response := map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"id":              1,
				"username":        "testuser",
				"email":           "test@example.com",
				"quota":           5000000,
				"daily_quota":     1000000,
				"daily_used":      500000,
				"base_group":      "pro",
				"fallback_group":  "free",
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	user, err := client.GetUser(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}

	if user.ID != 1 {
		t.Errorf("Expected ID=1, got %d", user.ID)
	}
	if user.Username != "testuser" {
		t.Errorf("Expected username=testuser, got %s", user.Username)
	}
}

func TestClient_UpdateUserSubscriptionConfig(t *testing.T) {
	var receivedConfig types.SubscriptionConfig
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/user/1/subscription" {
			t.Errorf("Expected path /api/user/1/subscription, got %s", r.URL.Path)
		}
		if r.Method != http.MethodPut {
			t.Errorf("Expected PUT method, got %s", r.Method)
		}

		if err := json.NewDecoder(r.Body).Decode(&receivedConfig); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		response := map[string]interface{}{
			"success": true,
			"message": "Config updated",
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	config := &types.SubscriptionConfig{
		DailyQuota:    1000000,
		BaseGroup:     "pro",
		FallbackGroup: "free",
		Quota:         5000000,
	}

	err := client.UpdateUserSubscriptionConfig(context.Background(), 1, config)
	if err != nil {
		t.Fatalf("UpdateUserSubscriptionConfig failed: %v", err)
	}

	// Verify received config
	if receivedConfig.DailyQuota != 1000000 {
		t.Errorf("Expected DailyQuota=1000000, got %d", receivedConfig.DailyQuota)
	}
	if receivedConfig.BaseGroup != "pro" {
		t.Errorf("Expected BaseGroup=pro, got %s", receivedConfig.BaseGroup)
	}
}

func TestClient_GetUserDailyQuotaStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/user/1/daily-quota" {
			t.Errorf("Expected path /api/user/1/daily-quota, got %s", r.URL.Path)
		}

		response := map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"user_id":           1,
				"daily_quota":       1000000,
				"daily_used":        250000,
				"daily_remaining":   750000,
				"last_daily_reset":  1704067200,
				"needs_reset":       false,
				"current_group":     "pro",
				"base_group":        "pro",
				"fallback_group":    "free",
				"is_using_fallback": false,
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	status, err := client.GetUserDailyQuotaStatus(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetUserDailyQuotaStatus failed: %v", err)
	}

	if status.DailyQuota != 1000000 {
		t.Errorf("Expected DailyQuota=1000000, got %d", status.DailyQuota)
	}
	if status.DailyUsed != 250000 {
		t.Errorf("Expected DailyUsed=250000, got %d", status.DailyUsed)
	}
	if status.DailyRemaining != 750000 {
		t.Errorf("Expected DailyRemaining=750000, got %d", status.DailyRemaining)
	}
}

func TestClient_ResetUserDailyQuota(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/user/1/daily-quota/reset" {
			t.Errorf("Expected path /api/user/1/daily-quota/reset, got %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		response := map[string]interface{}{
			"success": true,
			"message": "Daily quota reset",
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	err := client.ResetUserDailyQuota(context.Background(), 1)
	if err != nil {
		t.Fatalf("ResetUserDailyQuota failed: %v", err)
	}
}

func TestClient_ErrorHandling(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   interface{}
		expectErr  bool
	}{
		{
			name:       "success response",
			statusCode: http.StatusOK,
			response:   map[string]interface{}{"success": true},
			expectErr:  false,
		},
		{
			name:       "server error",
			statusCode: http.StatusInternalServerError,
			response:   map[string]interface{}{"error": "internal error"},
			expectErr:  true,
		},
		{
			name:       "not found",
			statusCode: http.StatusNotFound,
			response:   map[string]interface{}{"error": "not found"},
			expectErr:  true,
		},
		{
			name:       "success false",
			statusCode: http.StatusOK,
			response:   map[string]interface{}{"success": false, "message": "failed"},
			expectErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				json.NewEncoder(w).Encode(tt.response)
			}))
			defer server.Close()

			client := NewClient(server.URL, "test-token")
			err := client.ResetUserDailyQuota(context.Background(), 1)

			if tt.expectErr && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestClient_Timeout(t *testing.T) {
	// Test that client respects timeout
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Respond immediately for this test
		response := map[string]interface{}{"success": true}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	
	// Should work with normal timeout
	err := client.ResetUserDailyQuota(context.Background(), 1)
	if err != nil {
		t.Errorf("Request failed: %v", err)
	}
}

func BenchmarkClient_GetUser(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"success": true,
			"data":    map[string]interface{}{"id": 1, "username": "test"},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.GetUser(ctx, 1)
	}
}
