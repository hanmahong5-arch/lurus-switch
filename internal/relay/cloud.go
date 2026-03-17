package relay

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const cloudTimeout = 10 * time.Second

// CloudFetchTimeout is the context timeout for FetchCloudRelays calls.
const CloudFetchTimeout = cloudTimeout

// MigratedLegacyRelayID is the sentinel ID used when proxy settings are
// auto-migrated into the relay store on first run.
const MigratedLegacyRelayID = "migrated-legacy"

// FetchCloudRelays fetches the recommended relay endpoint list from the Lurus API.
// apiBase should be the configured API endpoint (e.g. "https://newapi.lurus.cn").
func FetchCloudRelays(ctx context.Context, apiBase string) ([]RelayEndpoint, error) {
	if apiBase == "" {
		return nil, fmt.Errorf("apiBase is required")
	}

	base := strings.TrimRight(apiBase, "/")
	url := base + "/api/v2/relays/recommended"

	client := &http.Client{Timeout: cloudTimeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch cloud relays: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cloud relay API returned HTTP %d", resp.StatusCode)
	}

	var endpoints []RelayEndpoint
	if err := json.NewDecoder(resp.Body).Decode(&endpoints); err != nil {
		return nil, fmt.Errorf("parse cloud relay response: %w", err)
	}

	return endpoints, nil
}
