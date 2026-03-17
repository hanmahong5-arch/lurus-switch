package relay

import (
	"context"
	"net/http"
	"sync"
	"time"
)

const healthTimeout = 5 * time.Second

// HealthCheckTimeout is the context timeout for CheckHealth calls.
const HealthCheckTimeout = healthTimeout

// CheckHealth pings all endpoints concurrently and returns updated copies with
// LatencyMs, Healthy, and LastChecked populated.
func CheckHealth(ctx context.Context, endpoints []RelayEndpoint) []RelayEndpoint {
	results := make([]RelayEndpoint, len(endpoints))
	copy(results, endpoints)

	var wg sync.WaitGroup
	for i := range results {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ep := &results[idx]
			ep.LatencyMs, ep.Healthy = ping(ctx, ep.URL)
			ep.LastChecked = time.Now().Format(time.RFC3339)
		}(i)
	}
	wg.Wait()
	return results
}

// ping attempts a HEAD/GET request to the endpoint URL and returns (latencyMs, ok).
func ping(ctx context.Context, rawURL string) (int64, bool) {
	if rawURL == "" {
		return -1, false
	}
	client := &http.Client{
		Timeout: healthTimeout,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return -1, false
	}

	start := time.Now()
	resp, err := client.Do(req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return -1, false
	}
	resp.Body.Close()
	return latency, resp.StatusCode < 500
}
