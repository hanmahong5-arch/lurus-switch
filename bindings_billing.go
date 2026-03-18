package main

import (
	"fmt"
	"net/url"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"lurus-switch/internal/billing"
)

// ============================
// Billing Methods
// ============================

// BillingGetUserInfo retrieves user account and quota information
func (a *App) BillingGetUserInfo() (*billing.UserInfo, error) {
	c, err := a.ensureBillingClient()
	if err != nil {
		return nil, err
	}
	return c.GetUserInfo(a.ctx)
}

// BillingGetQuotaSummary retrieves a lightweight quota summary for the dashboard
func (a *App) BillingGetQuotaSummary() (*billing.QuotaSummary, error) {
	c, err := a.ensureBillingClient()
	if err != nil {
		return nil, err
	}
	return c.GetQuotaSummary(a.ctx)
}

// BillingGetPlans retrieves available subscription plans
func (a *App) BillingGetPlans() ([]billing.SubscriptionPlan, error) {
	c, err := a.ensureBillingClient()
	if err != nil {
		return nil, err
	}
	return c.GetPlans(a.ctx)
}

// BillingGetSubscriptions retrieves the user's current subscriptions
func (a *App) BillingGetSubscriptions() ([]billing.SubscriptionInfo, error) {
	c, err := a.ensureBillingClient()
	if err != nil {
		return nil, err
	}
	return c.GetSubscriptions(a.ctx)
}

// BillingSubscribe creates a subscription request
func (a *App) BillingSubscribe(planCode, paymentMethod string) (*billing.PaymentResult, error) {
	c, err := a.ensureBillingClient()
	if err != nil {
		return nil, err
	}
	return c.Subscribe(a.ctx, planCode, paymentMethod)
}

// BillingCancelSubscription cancels an active subscription
func (a *App) BillingCancelSubscription(id int) error {
	c, err := a.ensureBillingClient()
	if err != nil {
		return err
	}
	return c.CancelSubscription(a.ctx, id)
}

// BillingGetTopUpInfo retrieves available top-up methods and options
func (a *App) BillingGetTopUpInfo() (*billing.TopUpInfo, error) {
	c, err := a.ensureBillingClient()
	if err != nil {
		return nil, err
	}
	return c.GetTopUpInfo(a.ctx)
}

// BillingCreateTopUp creates a top-up payment request
func (a *App) BillingCreateTopUp(amount int64, paymentMethod string) (*billing.PaymentResult, error) {
	c, err := a.ensureBillingClient()
	if err != nil {
		return nil, err
	}
	return c.CreateTopUp(a.ctx, amount, paymentMethod)
}

// BillingRedeemCode redeems a top-up code and returns the credited amount
func (a *App) BillingRedeemCode(code string) (int64, error) {
	c, err := a.ensureBillingClient()
	if err != nil {
		return 0, err
	}
	return c.RedeemCode(a.ctx, code)
}

// BillingGetIdentityOverview retrieves the aggregated identity overview
// (VIP level, Lubell wallet balance, subscription) from lurus-identity
// via the lurus-api proxy endpoint.
func (a *App) BillingGetIdentityOverview(productID string) (*billing.IdentityOverview, error) {
	c, err := a.ensureBillingClient()
	if err != nil {
		return nil, err
	}
	return c.GetIdentityOverview(a.ctx, productID)
}

// FetchCloudPresets fetches configuration presets for the given tool from the Lurus cloud.
// No authentication is required. Returns an empty slice on error (graceful degradation).
func (a *App) FetchCloudPresets(tool string) []billing.ConfigPreset {
	if a.proxyMgr == nil {
		return nil
	}
	settings := a.proxyMgr.GetSettings()
	if settings.APIEndpoint == "" {
		return nil
	}
	c := billing.NewClient(settings.APIEndpoint, settings.TenantSlug, "")
	presets, err := c.FetchPresets(a.ctx, tool)
	if err != nil {
		return nil
	}
	return presets
}

// BillingValidateToken creates a temporary billing client with the given endpoint and token
// and validates the token by fetching the identity overview. Used in the setup wizard
// to verify a Lurus account connection before saving proxy settings.
func (a *App) BillingValidateToken(endpoint, token string) (*billing.IdentityOverview, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("endpoint is required")
	}
	if token == "" {
		return nil, fmt.Errorf("token is required")
	}
	c := billing.NewClient(endpoint, "", token)
	return c.GetIdentityOverview(a.ctx, "")
}

// GetRecommendedConfig returns tool config recommendations based on user's subscription plan.
// free → conservative model, pro/enterprise → optimal model.
func (a *App) GetRecommendedConfig(tool string) map[string]interface{} {
	c, err := a.ensureBillingClient()
	if err != nil {
		return recommendedConfigForPlan(tool, "free")
	}
	ov, err := c.GetIdentityOverview(a.ctx, "")
	if err != nil || ov.Subscription == nil {
		return recommendedConfigForPlan(tool, "free")
	}
	return recommendedConfigForPlan(tool, ov.Subscription.PlanCode)
}

// recommendedConfigForPlan maps a plan code to suggested tool config values.
func recommendedConfigForPlan(tool, planCode string) map[string]interface{} {
	isPro := planCode == "pro" || planCode == "enterprise"
	switch tool {
	case "claude":
		model := "claude-haiku-4-5"
		if isPro {
			model = "claude-sonnet-4-5"
		}
		return map[string]interface{}{"model": model}
	case "codex":
		model := "gpt-4o-mini"
		if isPro {
			model = "gpt-4o"
		}
		return map[string]interface{}{"model": model}
	case "gemini":
		model := "gemini-2.0-flash"
		if isPro {
			model = "gemini-2.5-pro"
		}
		return map[string]interface{}{"model": model}
	}
	return nil
}

// BillingOpenTopup opens the lurus-identity top-up page in the user's default browser.
// The raw topup URL from IdentityOverview is validated and a redirect parameter is appended.
func (a *App) BillingOpenTopup(topupURL string) error {
	parsed, err := url.Parse(topupURL)
	if err != nil {
		return fmt.Errorf("invalid topup URL: %w", err)
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return fmt.Errorf("unsupported URL scheme %q: only http and https are allowed", parsed.Scheme)
	}
	if parsed.Host == "" {
		return fmt.Errorf("topup URL must contain a host")
	}
	q := parsed.Query()
	q.Set("from", "lurus-switch")
	parsed.RawQuery = q.Encode()
	runtime.BrowserOpenURL(a.ctx, parsed.String())
	return nil
}

// BillingOpenPaymentURL opens a payment URL in the user's default browser.
// Only http/https schemes are allowed to prevent arbitrary protocol opening.
func (a *App) BillingOpenPaymentURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid payment URL: %w", err)
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return fmt.Errorf("unsupported URL scheme %q: only http and https are allowed", parsed.Scheme)
	}
	if parsed.Host == "" {
		return fmt.Errorf("payment URL must contain a host")
	}
	runtime.BrowserOpenURL(a.ctx, parsed.String())
	return nil
}
