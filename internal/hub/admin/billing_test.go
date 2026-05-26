package admin

import (
	"context"
	"net/http"
	"testing"
)

func TestGetWalletInfo_Success(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/wallet/info" {
			t.Errorf("path = %s, want /api/wallet/info", r.URL.Path)
		}
		envRespond(w, map[string]any{
			"source":         "platform",
			"balance":        100.5,
			"frozen":         5.0,
			"available":      95.5,
			"lifetime_topup": 500.0,
			"lifetime_spend": 399.5,
			"topup_url":      "https://identity.lurus.cn/wallet/topup",
		})
	})

	info, err := c.GetWalletInfo(context.Background())
	if err != nil {
		t.Fatalf("GetWalletInfo: %v", err)
	}
	if info.Source != "platform" {
		t.Errorf("source = %q, want platform", info.Source)
	}
	if info.Balance != 100.5 || info.Available != 95.5 {
		t.Errorf("unexpected balances: %+v", info)
	}
	if info.TopupURL == "" {
		t.Error("topup_url should be populated when platform-backed")
	}
}

func TestGetWalletInfo_InternalFallback(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		envRespond(w, map[string]any{
			"source":         "internal",
			"balance":        12.0,
			"available":      12.0,
			"lifetime_spend": 3.0,
		})
	})

	info, err := c.GetWalletInfo(context.Background())
	if err != nil {
		t.Fatalf("GetWalletInfo: %v", err)
	}
	if info.Source != "internal" {
		t.Errorf("source = %q, want internal", info.Source)
	}
	if info.TopupURL != "" {
		t.Errorf("topup_url should be empty for internal fallback, got %q", info.TopupURL)
	}
}

func TestListWalletTransactions_PaginationAndShape(t *testing.T) {
	var gotP, gotPageSize string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/wallet/transactions" {
			t.Errorf("path = %s, want /api/wallet/transactions", r.URL.Path)
		}
		gotP = r.URL.Query().Get("p")
		gotPageSize = r.URL.Query().Get("page_size")
		envRespond(w, map[string]any{
			"items": []map[string]any{
				{"id": 1, "account_id": 99, "type": "topup", "amount": 100, "balance_after": 100, "description": "first topup", "created_at": "2026-05-26T00:00:00Z"},
				{"id": 2, "account_id": 99, "type": "debit", "amount": 12.5, "balance_after": 87.5, "product_id": "lurus-api", "description": "API spend", "created_at": "2026-05-26T01:00:00Z"},
			},
			"total":     17,
			"page":      2,
			"page_size": 10,
		})
	})

	page, err := c.ListWalletTransactions(context.Background(), WalletQuery{Page: 2, PageSize: 10})
	if err != nil {
		t.Fatalf("ListWalletTransactions: %v", err)
	}
	if gotP != "2" || gotPageSize != "10" {
		t.Errorf("pagination query = (p=%q, page_size=%q), want (2, 10)", gotP, gotPageSize)
	}
	if page.Total != 17 || page.Page != 2 || page.PageSize != 10 {
		t.Errorf("paging meta = %+v", page)
	}
	if len(page.Items) != 2 {
		t.Fatalf("items len = %d, want 2", len(page.Items))
	}
	if page.Items[1].Type != "debit" || page.Items[1].ProductID != "lurus-api" {
		t.Errorf("items[1] = %+v", page.Items[1])
	}
}

func TestListWalletTransactions_DefaultsWhenZero(t *testing.T) {
	var rawQuery string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		rawQuery = r.URL.RawQuery
		envRespond(w, map[string]any{"items": []any{}, "total": 0})
	})

	if _, err := c.ListWalletTransactions(context.Background(), WalletQuery{}); err != nil {
		t.Fatalf("ListWalletTransactions: %v", err)
	}
	if rawQuery != "" {
		t.Errorf("zero-value query should omit p/page_size, got %q", rawQuery)
	}
}
