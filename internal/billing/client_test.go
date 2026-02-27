package billing

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// helper: wrap data in the standard API envelope
func envelope(t *testing.T, data interface{}) []byte {
	t.Helper()
	raw, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal test data: %v", err)
	}
	resp := apiResponse{Success: true, Data: json.RawMessage(raw)}
	b, _ := json.Marshal(resp)
	return b
}

func envelopeError(message string) []byte {
	resp := apiResponse{Success: false, Message: message}
	b, _ := json.Marshal(resp)
	return b
}

// === NewClient Tests ===

func TestNewClient_TrimTrailingSlash(t *testing.T) {
	c := NewClient("https://api.example.com/", "tenant", "token")
	if c.baseURL != "https://api.example.com" {
		t.Errorf("baseURL = %q, want trailing slash trimmed", c.baseURL)
	}
}

func TestNewClient_MultipleTrailingSlashes(t *testing.T) {
	c := NewClient("https://api.example.com///", "tenant", "token")
	if c.baseURL != "https://api.example.com" {
		t.Errorf("baseURL = %q, want all trailing slashes trimmed", c.baseURL)
	}
}

func TestNewClient_NoTrailingSlash(t *testing.T) {
	c := NewClient("https://api.example.com", "tenant", "token")
	if c.baseURL != "https://api.example.com" {
		t.Errorf("baseURL = %q", c.baseURL)
	}
}

func TestNewClient_EmptyBaseURL(t *testing.T) {
	c := NewClient("", "tenant", "token")
	if c.baseURL != "" {
		t.Errorf("baseURL = %q, want empty", c.baseURL)
	}
}

func TestNewClient_SetsTimeout(t *testing.T) {
	c := NewClient("https://api.example.com", "", "token")
	if c.httpClient.Timeout != defaultTimeout {
		t.Errorf("timeout = %v, want %v", c.httpClient.Timeout, defaultTimeout)
	}
}

// === GetUserInfo Tests ===

func TestGetUserInfo_Success(t *testing.T) {
	info := UserInfo{
		Quota:     100000,
		UsedQuota: 30000,
		Username:  "testuser",
		Group:     "premium",
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v2/user/info" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("missing or wrong auth header: %q", r.Header.Get("Authorization"))
		}
		w.Write(envelope(t, info))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "test-token")
	result, err := c.GetUserInfo(context.Background())
	if err != nil {
		t.Fatalf("GetUserInfo error: %v", err)
	}
	if result.Username != "testuser" {
		t.Errorf("Username = %q, want testuser", result.Username)
	}
	if result.Quota != 100000 {
		t.Errorf("Quota = %d, want 100000", result.Quota)
	}
	if result.Group != "premium" {
		t.Errorf("Group = %q, want premium", result.Group)
	}
}

func TestGetUserInfo_TenantSlugHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slug := r.Header.Get("X-Tenant-Slug")
		if slug != "my-tenant" {
			t.Errorf("X-Tenant-Slug = %q, want my-tenant", slug)
		}
		w.Write(envelope(t, UserInfo{Username: "ok"}))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "my-tenant", "token")
	_, err := c.GetUserInfo(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetUserInfo_NoTenantSlugWhenEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if slug := r.Header.Get("X-Tenant-Slug"); slug != "" {
			t.Errorf("X-Tenant-Slug should be empty when tenant is empty, got %q", slug)
		}
		w.Write(envelope(t, UserInfo{}))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "token")
	_, _ = c.GetUserInfo(context.Background())
}

func TestGetUserInfo_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"success":false,"message":"internal server error"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "token")
	_, err := c.GetUserInfo(context.Background())
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
	if got := err.Error(); got == "" {
		t.Error("error message should not be empty")
	}
}

func TestGetUserInfo_HTTPErrorNoJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte("<html>Bad Gateway</html>"))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "token")
	_, err := c.GetUserInfo(context.Background())
	if err == nil {
		t.Fatal("expected error for HTTP 502")
	}
}

func TestGetUserInfo_APIReturnsFalse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(envelopeError("token expired"))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "token")
	_, err := c.GetUserInfo(context.Background())
	if err == nil {
		t.Fatal("expected error when API success=false")
	}
}

func TestGetUserInfo_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json at all"))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "token")
	_, err := c.GetUserInfo(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid JSON response")
	}
}

func TestGetUserInfo_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.Write(envelope(t, UserInfo{}))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "token")
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := c.GetUserInfo(ctx)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

// === GetQuotaSummary Tests ===

func TestGetQuotaSummary_Success(t *testing.T) {
	info := UserInfo{
		Quota:          100000,
		UsedQuota:      50000,
		RemainingQuota: 50000,
		DailyQuota:     10000,
		DailyUsed:      3000,
		Username:       "testuser",
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(envelope(t, info))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "token")
	summary, err := c.GetQuotaSummary(context.Background())
	if err != nil {
		t.Fatalf("GetQuotaSummary error: %v", err)
	}
	if summary.Quota != 100000 {
		t.Errorf("Quota = %d", summary.Quota)
	}
	if summary.RemainingQuota != 50000 {
		t.Errorf("RemainingQuota = %d", summary.RemainingQuota)
	}
	if summary.Username != "testuser" {
		t.Errorf("Username = %q", summary.Username)
	}
}

// === GetTopUpInfo Tests ===

func TestGetTopUpInfo_Success(t *testing.T) {
	info := TopUpInfo{
		PayMethods:    []map[string]string{{"alipay": "Alipay"}},
		AmountOptions: []int{10, 50, 100},
		MinTopup:      5,
		Discount:      0.95,
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/user/topup/info" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Write(envelope(t, info))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "token")
	result, err := c.GetTopUpInfo(context.Background())
	if err != nil {
		t.Fatalf("GetTopUpInfo error: %v", err)
	}
	if len(result.AmountOptions) != 3 {
		t.Errorf("AmountOptions count = %d", len(result.AmountOptions))
	}
	if result.MinTopup != 5 {
		t.Errorf("MinTopup = %d", result.MinTopup)
	}
}

// === GetPlans Tests ===

func TestGetPlans_Success(t *testing.T) {
	plans := []SubscriptionPlan{
		{Code: "basic", Name: "Basic Plan", Price: 9.99, Currency: "CNY"},
		{Code: "pro", Name: "Pro Plan", Price: 29.99, Currency: "CNY"},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/subscription/plans" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Write(envelope(t, plans))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "token")
	result, err := c.GetPlans(context.Background())
	if err != nil {
		t.Fatalf("GetPlans error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("plan count = %d, want 2", len(result))
	}
	if result[0].Code != "basic" {
		t.Errorf("first plan code = %q", result[0].Code)
	}
	if result[1].Price != 29.99 {
		t.Errorf("second plan price = %f", result[1].Price)
	}
}

func TestGetPlans_EmptyList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(envelope(t, []SubscriptionPlan{}))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "token")
	result, err := c.GetPlans(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty list, got %d", len(result))
	}
}

// === GetSubscriptions Tests ===

func TestGetSubscriptions_Success(t *testing.T) {
	subs := []SubscriptionInfo{
		{ID: 1, PlanCode: "basic", Status: "active", AutoRenew: true},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/subscription/list" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Write(envelope(t, subs))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "token")
	result, err := c.GetSubscriptions(context.Background())
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("count = %d", len(result))
	}
	if result[0].PlanCode != "basic" {
		t.Errorf("plan_code = %q", result[0].PlanCode)
	}
}

// === CreateTopUp Tests ===

func TestCreateTopUp_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v2/user/topup" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q", ct)
		}

		var body struct {
			Amount        int64  `json:"amount"`
			PaymentMethod string `json:"payment_method"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.Amount != 100 {
			t.Errorf("amount = %d", body.Amount)
		}
		if body.PaymentMethod != "alipay" {
			t.Errorf("payment_method = %q", body.PaymentMethod)
		}

		result := PaymentResult{TradeNo: "T123", PaymentURL: "https://pay.example.com/T123"}
		w.Write(envelope(t, result))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "token")
	result, err := c.CreateTopUp(context.Background(), 100, "alipay")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result.TradeNo != "T123" {
		t.Errorf("TradeNo = %q", result.TradeNo)
	}
	if result.PaymentURL != "https://pay.example.com/T123" {
		t.Errorf("PaymentURL = %q", result.PaymentURL)
	}
}

// === Subscribe Tests ===

func TestSubscribe_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/subscription/subscribe" {
			t.Errorf("path = %s", r.URL.Path)
		}
		var body struct {
			PlanCode      string `json:"plan_code"`
			PaymentMethod string `json:"payment_method"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		if body.PlanCode != "pro" {
			t.Errorf("plan_code = %q", body.PlanCode)
		}

		result := PaymentResult{TradeNo: "S456", PaymentURL: "https://pay.example.com/S456"}
		w.Write(envelope(t, result))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "token")
	result, err := c.Subscribe(context.Background(), "pro", "wechat")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result.TradeNo != "S456" {
		t.Errorf("TradeNo = %q", result.TradeNo)
	}
}

// === CancelSubscription Tests ===

func TestCancelSubscription_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/subscription/cancel" {
			t.Errorf("path = %s", r.URL.Path)
		}
		var body struct {
			ID int `json:"id"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		if body.ID != 42 {
			t.Errorf("id = %d", body.ID)
		}
		w.Write([]byte(`{"success":true,"message":"cancelled"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "token")
	err := c.CancelSubscription(context.Background(), 42)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
}

func TestCancelSubscription_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"success":false,"message":"not authorized"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "token")
	err := c.CancelSubscription(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error for 403")
	}
}

// === RedeemCode Tests ===

func TestRedeemCode_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/user/redeem" {
			t.Errorf("path = %s", r.URL.Path)
		}
		var body struct {
			Code string `json:"code"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		if body.Code != "GIFT2024" {
			t.Errorf("code = %q", body.Code)
		}
		result := struct {
			Amount int64 `json:"amount"`
		}{Amount: 5000}
		w.Write(envelope(t, result))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "token")
	amount, err := c.RedeemCode(context.Background(), "GIFT2024")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if amount != 5000 {
		t.Errorf("amount = %d, want 5000", amount)
	}
}

func TestRedeemCode_InvalidCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(envelopeError("invalid redeem code"))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "token")
	_, err := c.RedeemCode(context.Background(), "INVALID")
	if err == nil {
		t.Fatal("expected error for invalid code")
	}
}

// === doRequest Edge Cases ===

func TestDoRequest_EmptyDataField(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"success":true,"message":"ok"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "token")
	// GetSubscriptions should return nil slice when data is empty
	result, err := c.GetSubscriptions(context.Background())
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil for empty data, got %v", result)
	}
}

func TestDoRequest_NetworkError(t *testing.T) {
	c := NewClient("http://127.0.0.1:1", "", "token")
	c.httpClient.Timeout = 100 * time.Millisecond

	_, err := c.GetUserInfo(context.Background())
	if err == nil {
		t.Fatal("expected error for connection refused")
	}
}

// === Type Serialization Tests ===

func TestUserInfo_JSONRoundTrip(t *testing.T) {
	original := UserInfo{
		Quota:       100000,
		UsedQuota:   50000,
		Username:    "test",
		DisplayName: "Test User",
		AffCode:     "AFF123",
		Subscription: &SubscriptionInfo{
			ID:       1,
			PlanCode: "pro",
			Status:   "active",
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded UserInfo
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Username != original.Username {
		t.Errorf("Username = %q", decoded.Username)
	}
	if decoded.Subscription == nil {
		t.Fatal("Subscription should not be nil")
	}
	if decoded.Subscription.PlanCode != "pro" {
		t.Errorf("PlanCode = %q", decoded.Subscription.PlanCode)
	}
}

func TestUserInfo_JSONWithoutSubscription(t *testing.T) {
	original := UserInfo{Username: "noplan"}
	data, _ := json.Marshal(original)

	var decoded UserInfo
	json.Unmarshal(data, &decoded)

	if decoded.Subscription != nil {
		t.Error("Subscription should be nil when omitted")
	}
}

func TestSubscriptionPlan_Features(t *testing.T) {
	plan := SubscriptionPlan{
		Code:     "pro",
		Features: []string{"unlimited", "priority"},
	}
	data, _ := json.Marshal(plan)
	var decoded SubscriptionPlan
	json.Unmarshal(data, &decoded)

	if len(decoded.Features) != 2 {
		t.Errorf("features count = %d", len(decoded.Features))
	}
}

func TestPaymentResult_Fields(t *testing.T) {
	result := PaymentResult{
		TradeNo:    "T001",
		PaymentURL: "https://pay.example.com",
		Message:    "ok",
	}
	data, _ := json.Marshal(result)
	var decoded PaymentResult
	json.Unmarshal(data, &decoded)

	if decoded.TradeNo != "T001" {
		t.Errorf("TradeNo = %q", decoded.TradeNo)
	}
	if decoded.PaymentURL != "https://pay.example.com" {
		t.Errorf("PaymentURL = %q", decoded.PaymentURL)
	}
}

func TestQuotaSummary_Fields(t *testing.T) {
	qs := QuotaSummary{
		Quota:          100,
		UsedQuota:      40,
		RemainingQuota: 60,
		DailyQuota:     10,
		DailyUsed:      3,
		Username:       "u",
	}
	if qs.RemainingQuota != 60 {
		t.Errorf("RemainingQuota = %d", qs.RemainingQuota)
	}
}
