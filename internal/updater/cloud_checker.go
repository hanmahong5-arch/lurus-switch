package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const cloudCheckerTimeout = 10 * time.Second

// CloudVersionChecker fetches tool version information from the Lurus cloud endpoint.
// This avoids individual npm/GitHub requests by aggregating all tool versions server-side.
type CloudVersionChecker struct {
	endpoint   string
	httpClient *http.Client
}

// cloudToolVersionsResponse is the expected API response shape from
// GET <endpoint>/api/v2/switch/tools/versions
type cloudToolVersionsResponse struct {
	Success bool              `json:"success"`
	Data    map[string]string `json:"data"` // tool name → latest version string
}

// NewCloudVersionChecker creates a new checker targeting the given API endpoint.
func NewCloudVersionChecker(endpoint string) *CloudVersionChecker {
	return &CloudVersionChecker{
		endpoint:   strings.TrimRight(endpoint, "/"),
		httpClient: &http.Client{Timeout: cloudCheckerTimeout},
	}
}

// FetchAllVersions calls GET <endpoint>/api/v2/switch/tools/versions and returns
// a map of tool name → latest version string. Returns an error if the request fails
// or the endpoint is not configured.
func (c *CloudVersionChecker) FetchAllVersions(ctx context.Context) (map[string]string, error) {
	if c.endpoint == "" {
		return nil, fmt.Errorf("cloud endpoint not configured")
	}

	url := c.endpoint + "/api/v2/switch/tools/versions"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cloud version request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MB limit
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cloud endpoint returned HTTP %d", resp.StatusCode)
	}

	var result cloudToolVersionsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if !result.Success || result.Data == nil {
		return nil, fmt.Errorf("cloud endpoint returned unsuccessful response")
	}

	return result.Data, nil
}
