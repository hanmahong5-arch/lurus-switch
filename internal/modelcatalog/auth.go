package modelcatalog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// AuthVerdict classifies the outcome of a model-authenticity probe.
// The probe asks the upstream "you claim to offer model X — when I
// actually request it, what model do you say served the response?"
//
// IMPORTANT — what this detects vs. doesn't detect:
//   - DETECTS: the upstream returning a different model id than what
//     we requested (silent downgrade / mis-mapped channel).
//   - DOES NOT DETECT: a model that returns the right id but is
//     actually fronting a cheaper / smaller / fine-tuned variant.
//     That would require behavioural fingerprinting (probe questions
//     whose answers diverge across model families), which is out of
//     scope for this declaration-layer check.
//
// UI copy must surface this distinction so users don't read "match" as
// "guaranteed authentic" — see ModelAuthenticityPanel.
type AuthVerdict string

const (
	VerdictMatch        AuthVerdict = "match"        // requested == reported
	VerdictMismatch     AuthVerdict = "mismatch"     // requested != reported
	VerdictInconclusive AuthVerdict = "inconclusive" // response missing model field
	VerdictAuth         AuthVerdict = "auth"         // 401/403 — can't verify
	VerdictUnreachable  AuthVerdict = "unreachable" // DNS / connection
	VerdictTimeout      AuthVerdict = "timeout"     // probe exceeded budget
	VerdictError        AuthVerdict = "error"       // other non-2xx / parse fail
)

// ModelAuthResult is one (endpoint, model) probe outcome. Surfaced
// verbatim through the Wails binding so the matrix component can
// render a verdict badge and tooltip without further translation.
type ModelAuthResult struct {
	ProviderID     string      `json:"providerId"`
	ProviderName   string      `json:"providerName"`
	RequestedModel string      `json:"requestedModel"`
	ReportedModel  string      `json:"reportedModel"`
	Verdict        AuthVerdict `json:"verdict"`
	LatencyMs      int64       `json:"latencyMs"`
	Note           string      `json:"note,omitempty"`
	TestedAt       time.Time   `json:"testedAt"`
}

// authProbePrompt is the fixed canned prompt sent for every probe.
// Kept tiny and deterministic so the cost stays predictable and the
// upstream can't escape the budget by streaming. Output is bounded by
// max_tokens=1 below.
const authProbePrompt = "ping"

// authProbeMaxTokens caps the upstream response. The completion text
// is irrelevant — we only need the response envelope's `model` field.
const authProbeMaxTokens = 1

// authProbeBudget is the per-probe deadline. Long enough for one
// round-trip to a cold endpoint, short enough that a stuck upstream
// doesn't wedge the whole sweep.
const authProbeBudget = 10 * time.Second

// ProbeAuthenticity probes each (endpoint, model) pair by issuing a
// minimal chat completion and comparing the requested model id
// against the model id reported in the response envelope. Probes run
// sequentially per endpoint (we don't want to fan out 50 paid calls
// in parallel by accident); endpoints themselves are independent and
// can be probed in parallel by the caller.
//
// Caller MUST budget tokens — this method makes one real chat call
// per (endpoint × model) pair. Default behaviour from
// RunModelAuthCheck is "user-triggered only, every model exactly
// once" to keep cost obvious.
func ProbeAuthenticity(ctx context.Context, ep ProviderEndpoint, models []string) []ModelAuthResult {
	out := make([]ModelAuthResult, 0, len(models))
	for _, model := range models {
		model = strings.TrimSpace(model)
		if model == "" {
			continue
		}
		out = append(out, probeOne(ctx, ep, model))
	}
	return out
}

func probeOne(ctx context.Context, ep ProviderEndpoint, model string) ModelAuthResult {
	res := ModelAuthResult{
		ProviderID:     ep.ID,
		ProviderName:   ep.Name,
		RequestedModel: model,
		TestedAt:       time.Now(),
	}
	if strings.TrimSpace(ep.BaseURL) == "" {
		res.Verdict = VerdictError
		res.Note = "base URL is empty"
		return res
	}

	reqCtx, cancel := context.WithTimeout(ctx, authProbeBudget)
	defer cancel()

	endpoint := chatCompletionsEndpoint(ep.BaseURL)
	payload := map[string]any{
		"model":      model,
		"max_tokens": authProbeMaxTokens,
		"messages": []map[string]string{
			{"role": "user", "content": authProbePrompt},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		res.Verdict = VerdictError
		res.Note = err.Error()
		return res
	}

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		res.Verdict = VerdictError
		res.Note = err.Error()
		return res
	}
	if key := strings.TrimSpace(ep.APIKey); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	start := time.Now()
	resp, err := http.DefaultClient.Do(req)
	res.LatencyMs = time.Since(start).Milliseconds()
	if err != nil {
		if reqCtx.Err() == context.DeadlineExceeded {
			res.Verdict = VerdictTimeout
		} else {
			res.Verdict = VerdictUnreachable
		}
		res.Note = err.Error()
		return res
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		res.Verdict = VerdictAuth
		res.Note = fmt.Sprintf("HTTP %d", resp.StatusCode)
		return res
	case resp.StatusCode < 200 || resp.StatusCode >= 300:
		res.Verdict = VerdictError
		res.Note = fmt.Sprintf("HTTP %d", resp.StatusCode)
		return res
	}

	var parsed struct {
		Model string `json:"model"`
	}
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&parsed); err != nil {
		res.Verdict = VerdictInconclusive
		res.Note = "response not JSON: " + err.Error()
		return res
	}
	res.ReportedModel = parsed.Model
	if strings.TrimSpace(parsed.Model) == "" {
		res.Verdict = VerdictInconclusive
		res.Note = "response missing `model` field"
		return res
	}
	if !modelMatches(model, parsed.Model) {
		res.Verdict = VerdictMismatch
		return res
	}
	res.Verdict = VerdictMatch
	return res
}

// modelMatches treats prefix relationships as a match — many upstreams
// canonicalise "claude-sonnet-4-6" into "claude-sonnet-4-6-20250601"
// when echoing the model id back, which would otherwise flag as a
// mismatch even though it's the same family.
func modelMatches(requested, reported string) bool {
	a := strings.ToLower(strings.TrimSpace(requested))
	b := strings.ToLower(strings.TrimSpace(reported))
	if a == "" || b == "" {
		return false
	}
	if a == b {
		return true
	}
	return strings.HasPrefix(b, a) || strings.HasPrefix(a, b)
}

// chatCompletionsEndpoint resolves baseURL to its chat completions
// path. Mirrors modelsEndpoint's heuristic — accept a base URL with
// or without a /v1 suffix.
func chatCompletionsEndpoint(baseURL string) string {
	trimmed := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	lower := strings.ToLower(trimmed)
	switch {
	case strings.HasSuffix(lower, "/chat/completions"):
		return trimmed
	case strings.Contains(lower, "/v1") || strings.Contains(lower, "/v2") ||
		strings.Contains(lower, "/v3") || strings.Contains(lower, "/v4"):
		return trimmed + "/chat/completions"
	default:
		return trimmed + "/v1/chat/completions"
	}
}
