package modelcatalog

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

// Status classifies a provider-endpoint probe outcome. Deliberately coarse:
// we test ENDPOINT health (is /v1/models reachable + does it list models),
// NOT whether each individual model can complete a chat. The UI copy must
// say so — a "ok" here means "endpoint up and listing models", not "every
// model works".
type Status string

const (
	StatusOK          Status = "ok"          // 2xx + parseable model list
	StatusAuth        Status = "auth"        // 401/403 — key wrong or missing
	StatusUnreachable Status = "unreachable" // DNS / connection / TLS failure
	StatusTimeout     Status = "timeout"     // exceeded per-probe deadline
	StatusError       Status = "error"       // other non-2xx / parse failure
)

// ProviderEndpoint is the minimal shape the tester needs to probe a provider.
type ProviderEndpoint struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	BaseURL       string   `json:"baseUrl"`
	APIKey        string   `json:"apiKey"`
	DefaultModels []string `json:"defaultModels"`
}

// TestResult is the outcome of probing one provider endpoint. Models holds
// the model IDs the endpoint actually advertised, so the UI can cross them
// against DefaultModels to render a per-model "listed / not listed" matrix —
// again, "listed", not "verified chattable".
type TestResult struct {
	ProviderID   string    `json:"providerId"`
	ProviderName string    `json:"providerName"`
	Status       Status    `json:"status"`
	LatencyMs    int64     `json:"latencyMs"`
	Models       []string  `json:"models"`
	Error        string    `json:"error,omitempty"`
	TestedAt     time.Time `json:"testedAt"`
}

const (
	defaultWorkers     = 10
	defaultProbeBudget = 5 * time.Second
)

// Tester runs bounded-concurrency endpoint probes.
type Tester struct {
	Workers int
	Timeout time.Duration
}

// NewTester returns a Tester with sane defaults (10 workers, 5s per probe).
func NewTester() *Tester {
	return &Tester{Workers: defaultWorkers, Timeout: defaultProbeBudget}
}

// RunHealthCheck probes every endpoint, emitting one TestResult per provider
// on the returned channel as each completes (so the UI can stream progress).
// The channel is closed when all probes finish. Concurrency is capped by a
// semaphore at t.Workers — reuses the bounded-fan-out pattern from
// relay.CheckHealth, sized for "5 providers × a few models" workloads.
func (t *Tester) RunHealthCheck(ctx context.Context, endpoints []ProviderEndpoint) <-chan TestResult {
	out := make(chan TestResult)
	workers := t.Workers
	if workers <= 0 {
		workers = defaultWorkers
	}
	budget := t.Timeout
	if budget <= 0 {
		budget = defaultProbeBudget
	}

	go func() {
		defer close(out)
		sem := make(chan struct{}, workers)
		done := make(chan TestResult, len(endpoints))
		for _, ep := range endpoints {
			sem <- struct{}{}
			go func(e ProviderEndpoint) {
				defer func() { <-sem }()
				done <- probe(ctx, e, budget)
			}(ep)
		}
		for range endpoints {
			select {
			case r := <-done:
				out <- r
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

// probe performs a single endpoint health check.
func probe(ctx context.Context, ep ProviderEndpoint, budget time.Duration) TestResult {
	res := TestResult{
		ProviderID:   ep.ID,
		ProviderName: ep.Name,
		TestedAt:     time.Now(),
	}
	if strings.TrimSpace(ep.BaseURL) == "" {
		res.Status = StatusError
		res.Error = "base URL is empty"
		return res
	}

	reqCtx, cancel := context.WithTimeout(ctx, budget)
	defer cancel()

	endpoint := modelsEndpoint(ep.BaseURL)
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, endpoint, nil)
	if err != nil {
		res.Status = StatusError
		res.Error = err.Error()
		return res
	}
	if key := strings.TrimSpace(ep.APIKey); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	req.Header.Set("Accept", "application/json")

	start := time.Now()
	resp, err := http.DefaultClient.Do(req)
	res.LatencyMs = time.Since(start).Milliseconds()
	if err != nil {
		if reqCtx.Err() == context.DeadlineExceeded {
			res.Status = StatusTimeout
		} else {
			res.Status = StatusUnreachable
		}
		res.Error = err.Error()
		return res
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		res.Status = StatusAuth
		res.Error = "HTTP " + resp.Status
		return res
	case resp.StatusCode < 200 || resp.StatusCode >= 300:
		res.Status = StatusError
		res.Error = "HTTP " + resp.Status
		return res
	}

	res.Models = parseModelIDs(body)
	res.Status = StatusOK
	return res
}

// modelsEndpoint resolves baseURL to its /v1/models discovery endpoint,
// mirroring provider.modelsEndpoint (kept local to avoid exporting it and to
// keep this an independent probe path).
func modelsEndpoint(baseURL string) string {
	trimmed := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	lower := strings.ToLower(trimmed)
	switch {
	case strings.HasSuffix(lower, "/models"):
		return trimmed
	case strings.Contains(lower, "/v1") || strings.Contains(lower, "/v2") ||
		strings.Contains(lower, "/v3") || strings.Contains(lower, "/v4"):
		return trimmed + "/models"
	default:
		return trimmed + "/v1/models"
	}
}

func parseModelIDs(body []byte) []string {
	var parsed struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil
	}
	ids := make([]string, 0, len(parsed.Data))
	for _, m := range parsed.Data {
		if id := strings.TrimSpace(m.ID); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}
