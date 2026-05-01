package deploy

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// manualProvider is a no-op Provider — the user has already deployed
// newhub and is handing Switch the coordinates. Provision validates the
// inputs and echoes a Result; idempotency is automatic because there is
// no remote state to mutate.
type manualProvider struct{}

func (manualProvider) Kind() Kind { return KindManual }

func (manualProvider) Provision(_ context.Context, in Inputs) (*Result, error) {
	// Boundary validation only — internal fields are trusted to flow back
	// out unchanged.
	hubURL := strings.TrimSpace(in.Manual.HubURL)
	if hubURL == "" {
		return nil, fmt.Errorf("manual deploy: HubURL is required")
	}
	hubURL = strings.TrimRight(hubURL, "/")
	parsed, err := url.Parse(hubURL)
	if err != nil {
		return nil, fmt.Errorf("manual deploy: invalid HubURL %q: %w", hubURL, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("manual deploy: HubURL scheme must be http or https, got %q", parsed.Scheme)
	}
	if parsed.Host == "" {
		return nil, fmt.Errorf("manual deploy: HubURL must include a host")
	}

	token := strings.TrimSpace(in.Manual.AdminToken)
	if token == "" {
		return nil, fmt.Errorf("manual deploy: AdminToken is required")
	}

	return &Result{
		Kind:        KindManual,
		HubURL:      hubURL,
		AdminToken:  token,
		TenantSlug:  strings.TrimSpace(in.Manual.TenantSlug),
		DisplayName: strings.TrimSpace(in.DisplayName),
		Notes:       "",
	}, nil
}
