package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
)

// WalletInfo mirrors Hub's GET /api/wallet/info response. Source is "platform"
// when the data came from lurus-platform, "internal" when Hub fell back to
// local quota (no platform account linked) — Switch's Wallet page shows a
// banner explaining the limitation in the "internal" case.
type WalletInfo struct {
	Source         string  `json:"source"`
	Balance        float64 `json:"balance"`
	Frozen         float64 `json:"frozen"`
	Available      float64 `json:"available"`
	LifetimeTopup  float64 `json:"lifetime_topup"`
	LifetimeSpend  float64 `json:"lifetime_spend"`
	ActivePreAuths int64   `json:"active_preauths"`
	PendingOrders  int64   `json:"pending_orders"`
	TopupURL       string  `json:"topup_url,omitempty"`
}

// WalletTransaction is one row in the Wallet page's transactions table. All
// fields ride through unchanged from platform; ReferenceID may be empty for
// system-generated entries (e.g. monthly grants).
type WalletTransaction struct {
	ID            int64           `json:"id"`
	AccountID     int64           `json:"account_id"`
	Type          string          `json:"type"`
	Amount        float64         `json:"amount"`
	BalanceAfter  float64         `json:"balance_after"`
	ProductID     string          `json:"product_id"`
	ReferenceType string          `json:"reference_type"`
	ReferenceID   string          `json:"reference_id"`
	Description   string          `json:"description"`
	Metadata      json.RawMessage `json:"metadata,omitempty"`
	CreatedAt     string          `json:"created_at"`
}

// WalletQuery carries the Wallet page's filter inputs. Wails serializes this
// to TS as `admin.WalletQuery`. Zero values mean "no filter".
type WalletQuery struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

// WalletTransactionPage envelopes a single page of transactions.
type WalletTransactionPage struct {
	Items    []WalletTransaction `json:"items"`
	Total    int64               `json:"total"`
	Page     int                 `json:"page"`
	PageSize int                 `json:"page_size"`
}

// GetWalletInfo fetches the Hub's wallet snapshot (balance + lifetime + held).
func (c *Client) GetWalletInfo(ctx context.Context) (*WalletInfo, error) {
	var out WalletInfo
	if err := c.do(ctx, http.MethodGet, "/api/wallet/info", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListWalletTransactions fetches paginated wallet transactions.
func (c *Client) ListWalletTransactions(ctx context.Context, q WalletQuery) (*WalletTransactionPage, error) {
	v := url.Values{}
	if q.Page > 0 {
		v.Set("p", strconv.Itoa(q.Page))
	}
	if q.PageSize > 0 {
		v.Set("page_size", strconv.Itoa(q.PageSize))
	}
	var out WalletTransactionPage
	if err := c.do(ctx, http.MethodGet, "/api/wallet/transactions", v, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
