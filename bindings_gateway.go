package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"lurus-switch/internal/gateway"
	"lurus-switch/internal/metering"
)

// ============================
// Gateway Methods (replaces serverctl bindings)
// ============================

// GetGatewayStatus returns the current state of the local API gateway.
func (a *App) GetGatewayStatus() gateway.Status {
	if a.gatewaySrv == nil {
		return gateway.Status{}
	}
	return a.gatewaySrv.Status()
}

// StartGateway starts the local API gateway on localhost.
func (a *App) StartGateway() error {
	if a.gatewaySrv == nil {
		return fmt.Errorf("gateway not initialized")
	}
	// Sync upstream config from proxy settings before starting.
	a.syncGatewayUpstream()
	return a.gatewaySrv.Start(a.ctx)
}

// StopGateway stops the local API gateway.
func (a *App) StopGateway() error {
	if a.gatewaySrv == nil {
		return fmt.Errorf("gateway not initialized")
	}
	return a.gatewaySrv.Stop()
}

// GetGatewayConfig returns the current gateway configuration.
func (a *App) GetGatewayConfig() gateway.Config {
	if a.gatewaySrv == nil {
		return gateway.DefaultConfig()
	}
	return a.gatewaySrv.GetConfig()
}

// SaveGatewayConfig persists a new gateway configuration.
func (a *App) SaveGatewayConfig(cfg gateway.Config) error {
	if a.gatewaySrv == nil {
		return fmt.Errorf("gateway not initialized")
	}
	return a.gatewaySrv.SaveConfig(cfg)
}

// GetGatewayURL returns the base URL of the running gateway, or "" if stopped.
func (a *App) GetGatewayURL() string {
	if a.gatewaySrv == nil {
		return ""
	}
	st := a.gatewaySrv.Status()
	return st.URL
}

// syncGatewayUpstream reads proxy settings and pushes upstream URL/token to the gateway.
func (a *App) syncGatewayUpstream() {
	if a.gatewaySrv == nil || a.proxyMgr == nil {
		return
	}
	settings := a.proxyMgr.GetSettings()
	a.gatewaySrv.UpdateUpstream(settings.APIEndpoint, settings.BuildToolAPIKey())
}

// UpstreamHealthResult holds the outcome of an upstream connectivity test.
type UpstreamHealthResult struct {
	Reachable  bool   `json:"reachable"`
	LatencyMs  int64  `json:"latencyMs"` // round-trip in milliseconds, -1 on failure
	StatusCode int    `json:"statusCode"`
	Endpoint   string `json:"endpoint"`
	Error      string `json:"error,omitempty"`
}

// PingGatewayUpstream tests connectivity to the configured upstream API endpoint.
// Returns latency and reachability so the UI can show upstream health.
func (a *App) PingGatewayUpstream() UpstreamHealthResult {
	if a.proxyMgr == nil {
		return UpstreamHealthResult{Error: "proxy manager not initialized", LatencyMs: -1}
	}

	settings := a.proxyMgr.GetSettings()
	ep := settings.APIEndpoint
	if ep == "" {
		return UpstreamHealthResult{Error: "no upstream configured", LatencyMs: -1}
	}

	// Ping the /v1/models endpoint (lightweight, commonly available).
	target := ep + "/v1/models"

	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return UpstreamHealthResult{Endpoint: ep, Error: err.Error(), LatencyMs: -1}
	}

	// Add auth header if configured.
	apiKey := settings.BuildToolAPIKey()
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	start := time.Now()
	resp, err := client.Do(req)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		return UpstreamHealthResult{
			Endpoint:  ep,
			LatencyMs: -1,
			Error:     err.Error(),
		}
	}
	resp.Body.Close()

	return UpstreamHealthResult{
		Reachable:  resp.StatusCode < 500,
		LatencyMs:  latency,
		StatusCode: resp.StatusCode,
		Endpoint:   ep,
	}
}

// ============================
// Metering Methods
// ============================

// GetTodaySummary returns aggregated usage for today.
func (a *App) GetTodaySummary() metering.DailySummary {
	if a.meterStore == nil {
		return metering.DailySummary{}
	}
	return a.meterStore.TodaySummary()
}

// GetDaySummaries returns daily summaries for the last N days.
func (a *App) GetDaySummaries(days int) []metering.DailySummary {
	if a.meterStore == nil {
		return []metering.DailySummary{}
	}
	if days <= 0 || days > 90 {
		days = 30
	}
	return a.meterStore.DaySummaries(days)
}

// GetAppSummaries returns per-app usage for a date range.
// period: "today", "week", "month"
func (a *App) GetAppSummaries(period string) []metering.AppSummary {
	if a.meterStore == nil {
		return []metering.AppSummary{}
	}
	from, to := periodToRange(period)
	return a.meterStore.AppSummaries(from, to)
}

// GetModelSummaries returns per-model usage for a date range.
func (a *App) GetModelSummaries(period string) []metering.ModelSummary {
	if a.meterStore == nil {
		return []metering.ModelSummary{}
	}
	from, to := periodToRange(period)
	return a.meterStore.ModelSummaries(from, to)
}

// GetRecentActivity returns the N most recent API calls.
func (a *App) GetRecentActivity(n int) []metering.ActivityEntry {
	if a.meterStore == nil {
		return []metering.ActivityEntry{}
	}
	if n <= 0 || n > 100 {
		n = 20
	}
	return a.meterStore.RecentActivity(n)
}

// RequestLogEntry is a detailed view of a single API call for the request log.
type RequestLogEntry struct {
	ID         string `json:"id"`
	Timestamp  string `json:"timestamp"` // ISO 8601
	AppID      string `json:"appId"`
	Model      string `json:"model"`
	TokensIn   int64  `json:"tokensIn"`
	TokensOut  int64  `json:"tokensOut"`
	LatencyMs  int64  `json:"latencyMs"`
	StatusCode int    `json:"statusCode"`
	Cached     bool   `json:"cached"`
	Error      string `json:"error,omitempty"`
}

// GetRequestLog returns detailed recent API calls, optionally filtered by app/model.
// Returns up to `limit` entries (max 200), newest first.
func (a *App) GetRequestLog(limit int, filterApp string, filterModel string) []RequestLogEntry {
	if a.meterStore == nil {
		return []RequestLogEntry{}
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	// Get recent raw records from memory.
	records := a.meterStore.RecentRecords(limit * 2) // over-fetch for filtering

	var out []RequestLogEntry
	for i := len(records) - 1; i >= 0 && len(out) < limit; i-- {
		r := records[i]
		if filterApp != "" && r.AppID != filterApp {
			continue
		}
		if filterModel != "" && r.Model != filterModel {
			continue
		}
		out = append(out, RequestLogEntry{
			ID:         r.ID,
			Timestamp:  r.Timestamp.Format("2006-01-02T15:04:05"),
			AppID:      r.AppID,
			Model:      r.Model,
			TokensIn:   r.TokensIn,
			TokensOut:  r.TokensOut,
			LatencyMs:  r.LatencyMs,
			StatusCode: r.StatusCode,
			Cached:     r.CachedHit,
			Error:      r.ErrorMessage,
		})
	}
	return out
}

// UsageInsight holds combined cost/rate-limit/latency data for a period.
type UsageInsight struct {
	TotalCalls      int64                `json:"totalCalls"`
	TotalTokensIn   int64                `json:"totalTokensIn"`
	TotalTokensOut  int64                `json:"totalTokensOut"`
	CacheHitRate    float64              `json:"cacheHitRate"`    // 0.0 – 1.0
	RateLimitEvents int64                `json:"rateLimitEvents"` // HTTP 429 count
	ErrorEvents     int64                `json:"errorEvents"`     // HTTP 5xx count
	AvgLatencyMs    int64                `json:"avgLatencyMs"`
	TotalCostUSD    float64              `json:"totalCostUSD"`    // estimated total cost
	ModelCosts      []ModelCostBreakdown `json:"modelCosts"`      // per-model breakdown
}

// ModelCostBreakdown holds the estimated cost for one model.
type ModelCostBreakdown struct {
	Model       string  `json:"model"`
	TokensIn    int64   `json:"tokensIn"`
	TokensOut   int64   `json:"tokensOut"`
	InputRatio  float64 `json:"inputRatio"`
	OutputRatio float64 `json:"outputRatio"`
	CostUSD     float64 `json:"costUSD"`
}

// GetUsageInsights returns combined cost/rate-limit/latency insights for a period.
// Cost is estimated by matching model usage against the model catalog pricing ratios.
// Pricing formula: cost = (tokensIn * inputRatio + tokensOut * outputRatio) / 500000 * 2
// (One ratio unit ≈ $2 / 1M tokens is a common newapi convention.)
func (a *App) GetUsageInsights(period string) UsageInsight {
	if a.meterStore == nil {
		return UsageInsight{ModelCosts: []ModelCostBreakdown{}}
	}

	from, to := periodToRange(period)
	raw := a.meterStore.Insights(from, to)

	out := UsageInsight{
		ModelCosts:      []ModelCostBreakdown{},
		TotalCalls:      raw.TotalCalls,
		TotalTokensIn:   raw.TotalTokensIn,
		TotalTokensOut:  raw.TotalTokensOut,
		RateLimitEvents: raw.RateLimitEvents,
		ErrorEvents:     raw.ErrorEvents,
		AvgLatencyMs:    raw.AvgLatencyMs,
	}
	if raw.TotalCalls > 0 {
		out.CacheHitRate = float64(raw.CacheHits) / float64(raw.TotalCalls)
	}

	// Build model pricing lookup from catalog.
	pricingMap := make(map[string][2]float64) // model → [inputRatio, outputRatio]
	if a.catalogMgr != nil {
		cat := a.catalogMgr.GetCatalog()
		for _, m := range cat.Models {
			pricingMap[m.ID] = [2]float64{m.InputRatio, m.OutputRatio}
		}
	}

	// Calculate per-model costs.
	var totalCost float64
	for model, tokIn := range raw.ModelTokensIn {
		tokOut := raw.ModelTokensOut[model]
		ratios, ok := pricingMap[model]
		if !ok {
			// Unknown model: use a conservative default (deepseek-chat level).
			ratios = [2]float64{0.07, 0.14}
		}
		// newapi ratio convention: ratio 1.0 ≈ $2/1M tokens.
		cost := (float64(tokIn)*ratios[0] + float64(tokOut)*ratios[1]) / 500000.0
		totalCost += cost
		out.ModelCosts = append(out.ModelCosts, ModelCostBreakdown{
			Model:       model,
			TokensIn:    tokIn,
			TokensOut:   tokOut,
			InputRatio:  ratios[0],
			OutputRatio: ratios[1],
			CostUSD:     cost,
		})
	}
	out.TotalCostUSD = totalCost

	// Sort by cost descending.
	for i := 0; i < len(out.ModelCosts); i++ {
		for j := i + 1; j < len(out.ModelCosts); j++ {
			if out.ModelCosts[j].CostUSD > out.ModelCosts[i].CostUSD {
				out.ModelCosts[i], out.ModelCosts[j] = out.ModelCosts[j], out.ModelCosts[i]
			}
		}
	}

	return out
}

func periodToRange(period string) (time.Time, time.Time) {
	now := time.Now()
	to := now
	var from time.Time
	switch period {
	case "today":
		from = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	case "week":
		from = now.AddDate(0, 0, -7)
	case "month":
		from = now.AddDate(0, -1, 0)
	default:
		from = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	}
	return from, to
}
