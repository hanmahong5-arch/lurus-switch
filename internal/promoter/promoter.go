package promoter

import (
	"context"
	"fmt"

	"lurus-switch/internal/billing"
)

// PromoterInfo holds promoter data assembled from the billing API.
type PromoterInfo struct {
	AffCode        string  `json:"aff_code"`
	ShareLink      string  `json:"share_link"`
	GatewayURL     string  `json:"gateway_url"`
	TotalReferrals int     `json:"total_referrals"`
	TotalEarned    float64 `json:"total_earned"`
	PendingEarned  float64 `json:"pending_earned"`
}

const shareLinkBase = "https://lurus.cn/switch?ref="

// Service provides promoter-related operations.
type Service struct {
	billingClientFn func() (*billing.Client, error)
}

// NewService creates a promoter service. billingClientFn is a lazy factory
// (typically services.ensureBillingClient) that returns a configured billing client.
func NewService(billingClientFn func() (*billing.Client, error)) *Service {
	return &Service{billingClientFn: billingClientFn}
}

// GetInfo retrieves promoter information from the billing API.
// It fetches the user's aff_code and referral statistics.
func (s *Service) GetInfo(ctx context.Context) (*PromoterInfo, error) {
	c, err := s.billingClientFn()
	if err != nil {
		return nil, fmt.Errorf("billing client: %w", err)
	}

	userInfo, err := c.GetUserInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("get user info: %w", err)
	}

	info := &PromoterInfo{
		AffCode:   userInfo.AffCode,
		ShareLink: GenerateShareLink(userInfo.AffCode),
	}

	// Fetch affiliate stats; non-critical — default to zeros on failure.
	if stats, err := c.GetAffiliateStats(ctx); err == nil {
		info.TotalReferrals = stats.TotalReferrals
		info.TotalEarned = stats.TotalEarned
		info.PendingEarned = stats.PendingEarned
	}

	return info, nil
}

// GenerateShareLink builds a promoter share URL from the given aff_code.
func GenerateShareLink(affCode string) string {
	if affCode == "" {
		return ""
	}
	return shareLinkBase + affCode
}
