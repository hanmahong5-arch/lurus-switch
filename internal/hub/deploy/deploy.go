// Package deploy provisions a lurus-newhub instance on the user's cloud
// infrastructure on behalf of Reseller-mode setup. It exposes a Provider
// interface so the wizard can target Sealos / Aliyun ECS / "manual" with
// the same orchestration code.
//
// The "manual" provider is the Day 1 path — the user has already deployed
// newhub themselves (k3s, docker-compose, raw VM) and just needs Switch
// to record the resulting URL + admin token. The cloud adapters
// (sealos.go, aliyun.go) are TODO stubs until credentials are available
// to test end-to-end; until then they return a clear error pointing the
// user back to the manual path.
package deploy

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// Kind names a provider implementation. Stable across releases since it
// gets persisted in audit records.
type Kind string

const (
	KindManual Kind = "manual"
	KindSealos Kind = "sealos"
	KindAliyun Kind = "aliyun"
)

// AllKinds lists every Kind known to the wizard. Used by the frontend to
// render the picker — order is presentation order.
func AllKinds() []Kind {
	return []Kind{KindManual, KindSealos, KindAliyun}
}

// ParseKind validates and normalizes a raw kind string. Empty or unknown
// values are rejected so callers can fast-fail rather than silently
// fallback.
func ParseKind(raw string) (Kind, error) {
	switch Kind(strings.ToLower(strings.TrimSpace(raw))) {
	case KindManual:
		return KindManual, nil
	case KindSealos:
		return KindSealos, nil
	case KindAliyun:
		return KindAliyun, nil
	}
	return "", fmt.Errorf("unknown deploy kind %q (allowed: manual, sealos, aliyun)", raw)
}

// Inputs is the user-supplied payload for a deploy attempt.
//
// `Manual` carries the fields collected by the manual-entry form. `Sealos`
// and `Aliyun` are placeholders — the cloud adapters will populate their
// own input shapes when implemented.
type Inputs struct {
	Kind        Kind
	DisplayName string // Reseller-facing label, "Acme Corp"

	// Manual provider fields. The wizard sets these directly from form input.
	Manual ManualInputs
}

// ManualInputs is the trivial-path payload — the user already deployed
// newhub somewhere and just hands us the coordinates. Tenant slug is
// optional (V2 multi-tenant routes only).
type ManualInputs struct {
	HubURL     string
	AdminToken string
	TenantSlug string
}

// Result is what a successful Deploy returns. The wizard persists these
// to AppSettings.Reseller and uses them to bootstrap the admin client.
type Result struct {
	Kind        Kind
	HubURL      string
	AdminToken  string
	TenantSlug  string
	DisplayName string

	// Notes carries provider-specific human-readable hints — surfaced in the
	// final wizard step so the user can capture them (e.g. Sealos console
	// link, Aliyun ECS instance ID). Empty for manual.
	Notes string
}

// Provider deploys (or records) a Hub instance for Reseller use.
//
// Provision is the single mutating call. It must be idempotent at the
// Switch boundary: re-running with the same inputs that already produced a
// saved Reseller config should return the existing Result, not error.
type Provider interface {
	Kind() Kind
	Provision(ctx context.Context, in Inputs) (*Result, error)
}

// ErrNotImplemented is returned by stub providers (sealos / aliyun until
// adapters are written) so the frontend can render "Coming soon — please
// use 手动" instead of a generic error.
var ErrNotImplemented = errors.New("deploy provider not yet implemented")

// IsNotImplemented unwraps nested errors that originated as ErrNotImplemented.
func IsNotImplemented(err error) bool {
	return errors.Is(err, ErrNotImplemented)
}

// New picks the provider for the given Kind. Returns an error for unknown
// kinds rather than panicking — keeps boundary validation explicit.
func New(kind Kind) (Provider, error) {
	switch kind {
	case KindManual:
		return manualProvider{}, nil
	case KindSealos:
		return sealosProvider{}, nil
	case KindAliyun:
		return aliyunProvider{}, nil
	}
	return nil, fmt.Errorf("unknown deploy kind: %s", kind)
}
