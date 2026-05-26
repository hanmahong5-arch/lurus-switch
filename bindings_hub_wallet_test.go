package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"lurus-switch/internal/hub/admin"
)

// TestHubGetWalletInfo_PlatformBacked verifies the binding round-trips the
// Hub envelope and surfaces source="platform".
func TestHubGetWalletInfo_PlatformBacked(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/wallet/info" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data": map[string]any{
				"source":         "platform",
				"balance":        250.0,
				"available":      245.0,
				"frozen":         5.0,
				"lifetime_topup": 1000.0,
				"lifetime_spend": 750.0,
				"topup_url":      "https://identity.lurus.cn/wallet/topup",
			},
		})
	}))
	defer srv.Close()

	withFakeHub(t, srv.URL)
	app := &App{}

	info, err := app.HubGetWalletInfo()
	if err != nil {
		t.Fatalf("HubGetWalletInfo: %v", err)
	}
	if info.Source != "platform" {
		t.Errorf("source = %q, want platform", info.Source)
	}
	if info.Balance != 250.0 || info.Frozen != 5.0 {
		t.Errorf("unexpected balances: %+v", info)
	}
}

// TestHubListWalletTransactions_PassesPagination verifies the binding forwards
// page/page_size and unwraps the envelope into a typed page.
func TestHubListWalletTransactions_PassesPagination(t *testing.T) {
	var gotP, gotSize string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/wallet/transactions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		gotP = r.URL.Query().Get("p")
		gotSize = r.URL.Query().Get("page_size")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data": map[string]any{
				"items": []map[string]any{
					{"id": 11, "account_id": 7, "type": "topup", "amount": 50, "balance_after": 50, "description": "topup", "created_at": "2026-05-26T00:00:00Z"},
				},
				"total":     31,
				"page":      3,
				"page_size": 5,
			},
		})
	}))
	defer srv.Close()

	withFakeHub(t, srv.URL)
	app := &App{}

	page, err := app.HubListWalletTransactions(admin.WalletQuery{Page: 3, PageSize: 5})
	if err != nil {
		t.Fatalf("HubListWalletTransactions: %v", err)
	}
	if gotP != "3" || gotSize != "5" {
		t.Errorf("pagination forwarded as (p=%q, page_size=%q), want (3, 5)", gotP, gotSize)
	}
	if page.Total != 31 {
		t.Errorf("total = %d, want 31", page.Total)
	}
	if len(page.Items) != 1 || page.Items[0].ID != 11 {
		t.Errorf("items = %+v", page.Items)
	}
}
