package pricing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// SwitchPricingPath is the public, cacheable Hub endpoint that mirrors the
// per-tenant pricing catalogue without auth. Switch syncs its rate card from
// here so the cost dashboard reflects the operator's live Hub ratios instead
// of the hard-coded fallback table.
const SwitchPricingPath = "/api/v2/switch/pricing"

// defaultQuotaPerUnit is newapi's standard quota↔USD scale: 500000 quota = $1
// (i.e. $0.002 / 1K tokens). The Hub echoes its own quota_per_unit in the
// response so we don't hard-couple to this, but it's the safe fallback when
// the field is absent or non-positive.
const defaultQuotaPerUnit = 500000.0

// maxRateCardModels bounds how many models we ingest from one sync so a
// pathological Hub response can't blow up memory. Real catalogues are a few
// hundred entries.
const maxRateCardModels = 5000

// rateCardResponse is the envelope returned by SwitchPricingPath. Only the
// fields needed to reconstruct a per-token Price are modeled; everything else
// (vendors, group_ratio, endpoint types) is ignored here.
type rateCardResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		Pricing      []rateCardItem `json:"pricing"`
		QuotaPerUnit float64        `json:"quota_per_unit"`
	} `json:"data"`
}

// rateCardItem is one model's pricing row. ModelRatio is the per-token input
// multiplier; CompletionRatio scales output off ModelRatio. QuotaType==1 (or
// a positive ModelPrice) means per-call flat pricing, which has no per-token
// representation — those rows are skipped.
type rateCardItem struct {
	ModelName       string  `json:"model_name"`
	QuotaType       int     `json:"quota_type"`
	ModelRatio      float64 `json:"model_ratio"`
	CompletionRatio float64 `json:"completion_ratio"`
	ModelPrice      float64 `json:"model_price"`
}

// FetchRateCard GETs the Hub's public Switch pricing catalogue and maps it into
// per-model Prices keyed by model id (used directly as a lookup prefix). The
// caller supplies the *http.Client so the request honours the app's BYO
// upstream proxy (a timeout-only client still routes through the patched
// default transport). The returned map is suitable to pass to Override.
//
// A transport error, non-200 status, success:false envelope, or a card that
// maps to zero usable models all surface as an error so the caller keeps the
// existing overlay (or the static table) rather than wiping pricing.
func FetchRateCard(ctx context.Context, httpClient *http.Client, hubBaseURL string) (map[string]Price, error) {
	base := strings.TrimRight(strings.TrimSpace(hubBaseURL), "/")
	if base == "" {
		return nil, errors.New("pricing: hub base URL is required")
	}
	if httpClient == nil {
		return nil, errors.New("pricing: http client is required")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+SwitchPricingPath, nil)
	if err != nil {
		return nil, fmt.Errorf("pricing: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("pricing: fetch rate card: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("pricing: hub returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20)) // 8MiB ceiling
	if err != nil {
		return nil, fmt.Errorf("pricing: read rate card: %w", err)
	}

	var parsed rateCardResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("pricing: decode rate card: %w", err)
	}
	if !parsed.Success {
		msg := parsed.Message
		if msg == "" {
			msg = "hub reported success:false"
		}
		return nil, fmt.Errorf("pricing: %s", msg)
	}

	card := mapHubRateCard(parsed.Data.Pricing, parsed.Data.QuotaPerUnit)
	if len(card) == 0 {
		return nil, errors.New("pricing: rate card mapped to zero usable models")
	}
	return card, nil
}

// mapHubRateCard converts the Hub's ratio-based pricing rows into per-token
// USD Prices. Pure and deterministic so it can be tested without HTTP.
//
// Derivation (newapi convention): a ratio of 1 corresponds to
// 1e6 / quotaPerUnit USD per 1M tokens. Input = modelRatio × that; output =
// modelRatio × completionRatio × that. Cache streams reuse the model family's
// multipliers (cacheMults) since the public catalogue doesn't carry cache
// ratios.
func mapHubRateCard(items []rateCardItem, quotaPerUnit float64) map[string]Price {
	if quotaPerUnit <= 0 {
		quotaPerUnit = defaultQuotaPerUnit
	}
	usdPerMTokForRatio1 := 1_000_000.0 / quotaPerUnit

	out := make(map[string]Price)
	for _, it := range items {
		if len(out) >= maxRateCardModels {
			break
		}
		name := strings.ToLower(strings.TrimSpace(it.ModelName))
		if name == "" {
			continue
		}
		// Per-call flat pricing has no per-token form — leave those to the
		// static table / fallback rather than inventing a token rate.
		if it.QuotaType == 1 || it.ModelPrice > 0 {
			continue
		}
		if it.ModelRatio <= 0 {
			continue
		}
		completionRatio := it.CompletionRatio
		if completionRatio <= 0 {
			completionRatio = 1.0 // no separate output ratio published → bill output at input rate
		}

		input := it.ModelRatio * usdPerMTokForRatio1
		output := it.ModelRatio * completionRatio * usdPerMTokForRatio1
		createMult, readMult := cacheMults(name)
		out[name] = rateWithCache(input, output, createMult, readMult)
	}
	return out
}
