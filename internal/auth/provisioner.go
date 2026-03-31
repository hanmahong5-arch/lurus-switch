package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	provisionTimeout = 30 * time.Second
)

// ProvisionRequest is the payload for the lurus-api internal provisioning endpoint.
type ProvisionRequest struct {
	ZitadelSub         string `json:"zitadel_sub"`
	Email              string `json:"email"`
	DisplayName        string `json:"display_name"`
	CreateInitialToken bool   `json:"create_initial_token"`
	InitialTokenName   string `json:"initial_token_name"`
}

// ProvisionResponse is the response from the provisioning endpoint.
type ProvisionResponse struct {
	UserID   int    `json:"user_id"`
	TokenKey string `json:"token_key,omitempty"`
	Status   string `json:"status"` // "created" or "existing"
}

// Provision calls the lurus-api internal provisioning endpoint to create or retrieve
// a gateway user and token. This is idempotent: repeated calls return the existing token.
func Provision(ctx context.Context, apiGatewayURL, internalKey, zitadelSub, email, displayName string) (*ProvisionResponse, error) {
	if apiGatewayURL == "" {
		return nil, fmt.Errorf("gateway URL is empty")
	}
	if internalKey == "" {
		return nil, fmt.Errorf("internal API key is empty")
	}
	if zitadelSub == "" {
		return nil, fmt.Errorf("zitadel_sub is required for provisioning")
	}

	ctx, cancel := context.WithTimeout(ctx, provisionTimeout)
	defer cancel()

	reqBody := ProvisionRequest{
		ZitadelSub:         zitadelSub,
		Email:              email,
		DisplayName:        displayName,
		CreateInitialToken: true,
		InitialTokenName:   "lurus-switch",
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal provision request: %w", err)
	}

	url := apiGatewayURL + "/internal/user/provision"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create provision request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", internalKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("provision API request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read provision response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		preview := string(respBody)
		if len(preview) > 500 {
			preview = preview[:500]
		}
		return nil, fmt.Errorf("provision failed (HTTP %d): %s", resp.StatusCode, preview)
	}

	var result ProvisionResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse provision response: %w", err)
	}

	return &result, nil
}
