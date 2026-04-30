package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

// FetchModelsTimeout bounds how long /v1/models discovery may take.
const FetchModelsTimeout = 15 * time.Second

type modelsResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

// FetchModels queries an OpenAI-compatible /v1/models endpoint and returns
// the sorted, deduplicated list of model IDs.
//
// baseURL accepts either the provider root (e.g. "https://api.deepseek.com")
// or a URL already pointing at /models — the path is normalized.
// apiKey is sent as Bearer; empty keys are allowed since some self-hosted
// endpoints (Ollama, LM Studio) do not require auth.
func FetchModels(ctx context.Context, baseURL, apiKey string) ([]string, error) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return nil, fmt.Errorf("base URL is required")
	}

	endpoint := modelsEndpoint(baseURL)

	reqCtx, cancel := context.WithTimeout(ctx, FetchModelsTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	if apiKey = strings.TrimSpace(apiKey); apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // cap 1 MiB
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snippet := strings.TrimSpace(string(body))
		if len(snippet) > 200 {
			snippet = snippet[:200] + "…"
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, snippet)
	}

	var parsed modelsResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("parse response (expected OpenAI-compatible JSON): %w", err)
	}

	seen := make(map[string]struct{}, len(parsed.Data))
	ids := make([]string, 0, len(parsed.Data))
	for _, m := range parsed.Data {
		id := strings.TrimSpace(m.ID)
		if id == "" {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}

	if len(ids) == 0 {
		return nil, fmt.Errorf("provider returned no models")
	}

	sort.Strings(ids)
	return ids, nil
}

// modelsEndpoint resolves baseURL to the /v1/models discovery endpoint.
// If baseURL already ends in /models it is used as-is; otherwise /v1/models
// (or /models when the URL already includes a versioned path) is appended.
func modelsEndpoint(baseURL string) string {
	trimmed := strings.TrimRight(baseURL, "/")
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
