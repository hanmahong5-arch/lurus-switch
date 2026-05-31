package pricing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// fakeHub returns an httptest server that serves body+status at SwitchPricingPath.
func fakeHub(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != SwitchPricingPath {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

const sampleRateCard = `{
  "success": true,
  "data": {
    "quota_per_unit": 500000,
    "pricing": [
      {"model_name": "gpt-4o", "quota_type": 0, "model_ratio": 1.25, "completion_ratio": 4.0, "model_price": 0},
      {"model_name": "claude-sonnet-4-6", "quota_type": 0, "model_ratio": 1.5, "completion_ratio": 5.0, "model_price": 0},
      {"model_name": "flat-vision-model", "quota_type": 1, "model_ratio": 0, "completion_ratio": 0, "model_price": 0.05}
    ]
  }
}`

func TestFetchRateCard_MapsRatiosToUSD(t *testing.T) {
	srv := fakeHub(t, http.StatusOK, sampleRateCard)

	card, err := FetchRateCard(context.Background(), srv.Client(), srv.URL)
	if err != nil {
		t.Fatalf("FetchRateCard: %v", err)
	}

	// quota_per_unit 500000 → ratio 1 = $2/MTok. gpt-4o ratio 1.25 → $2.50 in.
	gpt, ok := card["gpt-4o"]
	if !ok {
		t.Fatal("gpt-4o missing from card")
	}
	if !approxEq(gpt.InputPerMTok, 2.50) {
		t.Errorf("gpt-4o input = %v, want 2.50", gpt.InputPerMTok)
	}
	// output = ratio × completion_ratio × 2 = 1.25 × 4 × 2 = 10.00.
	if !approxEq(gpt.OutputPerMTok, 10.00) {
		t.Errorf("gpt-4o output = %v, want 10.00", gpt.OutputPerMTok)
	}
	// OpenAI family cache-read 0.50× input.
	if !approxEq(gpt.CacheReadPerMTok, 2.50*0.50) {
		t.Errorf("gpt-4o cache-read = %v, want %v", gpt.CacheReadPerMTok, 2.50*0.50)
	}

	// Claude: ratio 1.5 → $3 in; output 1.5×5×2 = 15; cache-read 0.10×.
	cl, ok := card["claude-sonnet-4-6"]
	if !ok {
		t.Fatal("claude-sonnet-4-6 missing from card")
	}
	if !approxEq(cl.InputPerMTok, 3.00) || !approxEq(cl.OutputPerMTok, 15.00) {
		t.Errorf("claude rates = %+v, want input 3 output 15", cl)
	}
	if !approxEq(cl.CacheReadPerMTok, 3.00*0.10) {
		t.Errorf("claude cache-read = %v, want %v", cl.CacheReadPerMTok, 3.00*0.10)
	}

	// Per-call (quota_type 1) model is skipped — no per-token representation.
	if _, ok := card["flat-vision-model"]; ok {
		t.Error("per-call model should be skipped, but it was mapped")
	}
}

func TestFetchRateCard_AppliedViaOverrideReflectsInPriceFor(t *testing.T) {
	resetOverrides(t)
	srv := fakeHub(t, http.StatusOK, sampleRateCard)

	card, err := FetchRateCard(context.Background(), srv.Client(), srv.URL)
	if err != nil {
		t.Fatalf("FetchRateCard: %v", err)
	}
	Override(card, func() time.Time { return time.Unix(0, 0) })

	// gpt-4o overlay input matches the synced $2.50 (same as static here, but
	// it now flows through the override path — assert the overlay is active).
	if OverrideCount() == 0 {
		t.Fatal("override not applied")
	}
	if p := PriceFor("gpt-4o"); !approxEq(p.InputPerMTok, 2.50) {
		t.Errorf("PriceFor(gpt-4o) input = %v, want 2.50", p.InputPerMTok)
	}
}

func TestFetchRateCard_HubFailureLeavesStaticTableIntact(t *testing.T) {
	resetOverrides(t)

	// 404 → error, no card returned.
	srv404 := fakeHub(t, http.StatusNotFound, `not found`)
	if _, err := FetchRateCard(context.Background(), srv404.Client(), srv404.URL); err == nil {
		t.Error("expected error on HTTP 404")
	}

	// success:false → error.
	srvFail := fakeHub(t, http.StatusOK, `{"success":false,"message":"nope"}`)
	if _, err := FetchRateCard(context.Background(), srvFail.Client(), srvFail.URL); err == nil {
		t.Error("expected error on success:false envelope")
	}

	// empty pricing → "zero usable models" error.
	srvEmpty := fakeHub(t, http.StatusOK, `{"success":true,"data":{"quota_per_unit":500000,"pricing":[]}}`)
	if _, err := FetchRateCard(context.Background(), srvEmpty.Client(), srvEmpty.URL); err == nil {
		t.Error("expected error on empty rate card")
	}

	// Through all failures the static table is unchanged.
	if p := PriceFor("gpt-4o"); !approxEq(p.InputPerMTok, 2.50) {
		t.Errorf("static gpt-4o input drifted to %v after failed syncs", p.InputPerMTok)
	}
	if OverrideCount() != 0 {
		t.Errorf("failed sync should not populate overlay, count = %d", OverrideCount())
	}
}

func TestFetchRateCard_GuardsBadInput(t *testing.T) {
	if _, err := FetchRateCard(context.Background(), http.DefaultClient, ""); err == nil {
		t.Error("empty base URL should error")
	}
	if _, err := FetchRateCard(context.Background(), nil, "https://hub.example"); err == nil {
		t.Error("nil http client should error")
	}
}

func TestMapHubRateCard_DefaultsQuotaPerUnit(t *testing.T) {
	// quota_per_unit <= 0 falls back to 500000 (→ ratio 1 = $2/MTok).
	card := mapHubRateCard([]rateCardItem{
		{ModelName: "gpt-4o", QuotaType: 0, ModelRatio: 1.0, CompletionRatio: 1.0},
	}, 0)
	if p, ok := card["gpt-4o"]; !ok || !approxEq(p.InputPerMTok, 2.0) {
		t.Errorf("default quota-per-unit mapping wrong: %+v ok=%v", card["gpt-4o"], ok)
	}
}

func TestMapHubRateCard_NoCompletionRatioBillsOutputAtInput(t *testing.T) {
	card := mapHubRateCard([]rateCardItem{
		{ModelName: "mystery-model", QuotaType: 0, ModelRatio: 2.0, CompletionRatio: 0},
	}, 500000)
	p := card["mystery-model"]
	// completion_ratio 0 → default 1.0 → output == input == 2.0×2 = 4.0.
	if !approxEq(p.OutputPerMTok, p.InputPerMTok) || !approxEq(p.InputPerMTok, 4.0) {
		t.Errorf("missing completion ratio mapping wrong: %+v", p)
	}
}
