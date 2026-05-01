package deploy

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestParseKind(t *testing.T) {
	t.Run("accepts canonical kinds", func(t *testing.T) {
		for _, raw := range []string{"manual", "sealos", "aliyun", "MANUAL", "  Sealos  "} {
			if _, err := ParseKind(raw); err != nil {
				t.Errorf("ParseKind(%q) unexpected error: %v", raw, err)
			}
		}
	})
	t.Run("rejects empty and unknown", func(t *testing.T) {
		for _, raw := range []string{"", "k8s", "vercel"} {
			if _, err := ParseKind(raw); err == nil {
				t.Errorf("ParseKind(%q) expected error, got nil", raw)
			}
		}
	})
}

func TestManual_Provision_TrimsAndValidates(t *testing.T) {
	p, err := New(KindManual)
	if err != nil {
		t.Fatal(err)
	}
	res, err := p.Provision(context.Background(), Inputs{
		Kind:        KindManual,
		DisplayName: "  Acme Corp  ",
		Manual: ManualInputs{
			HubURL:     "https://hub.acme.example/",
			AdminToken: "  tok-abc  ",
			TenantSlug: "  acme  ",
		},
	})
	if err != nil {
		t.Fatalf("Provision: %v", err)
	}
	if res.HubURL != "https://hub.acme.example" {
		t.Errorf("HubURL not trimmed: %q", res.HubURL)
	}
	if res.AdminToken != "tok-abc" {
		t.Errorf("AdminToken not trimmed: %q", res.AdminToken)
	}
	if res.TenantSlug != "acme" {
		t.Errorf("TenantSlug not trimmed: %q", res.TenantSlug)
	}
	if res.DisplayName != "Acme Corp" {
		t.Errorf("DisplayName not trimmed: %q", res.DisplayName)
	}
	if res.Kind != KindManual {
		t.Errorf("Kind = %q, want manual", res.Kind)
	}
}

func TestManual_Provision_RejectsInvalidInputs(t *testing.T) {
	p, _ := New(KindManual)

	cases := map[string]ManualInputs{
		"empty url":      {HubURL: "", AdminToken: "tok"},
		"bad scheme":     {HubURL: "ftp://hub.example", AdminToken: "tok"},
		"no host":        {HubURL: "https://", AdminToken: "tok"},
		"empty token":    {HubURL: "https://hub.example", AdminToken: ""},
		"whitespace tok": {HubURL: "https://hub.example", AdminToken: "   "},
	}
	for name, m := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := p.Provision(context.Background(), Inputs{Kind: KindManual, Manual: m}); err == nil {
				t.Error("expected validation error, got nil")
			}
		})
	}
}

func TestStub_ReturnsNotImplemented(t *testing.T) {
	for _, k := range []Kind{KindSealos, KindAliyun} {
		t.Run(string(k), func(t *testing.T) {
			p, err := New(k)
			if err != nil {
				t.Fatal(err)
			}
			_, err = p.Provision(context.Background(), Inputs{Kind: k})
			if !IsNotImplemented(err) {
				t.Errorf("expected ErrNotImplemented, got %v", err)
			}
			if !strings.Contains(err.Error(), "手动接入") {
				t.Errorf("error message should hint at manual fallback: %q", err.Error())
			}
		})
	}
}

func TestNew_RejectsUnknownKind(t *testing.T) {
	if _, err := New(Kind("nope")); err == nil {
		t.Error("New(nope) expected error, got nil")
	}
}

func TestErrNotImplemented_IsNotImplemented(t *testing.T) {
	wrapped := errors.New("upstream: " + ErrNotImplemented.Error())
	if IsNotImplemented(wrapped) {
		// errors.Is requires explicit wrapping with %w, not stringification —
		// guard against accidental string-only wrappers.
		t.Error("plain string-wrapped error must NOT match IsNotImplemented")
	}
}
