package capability

import (
	"context"
	"errors"
	"testing"
)

func TestToken_Has(t *testing.T) {
	tok := NewToken("test", CapPricingRead, CapChannelRead)
	if !tok.Has(CapPricingRead) {
		t.Error("expected CapPricingRead granted")
	}
	if tok.Has(CapPricingWrite) {
		t.Error("expected CapPricingWrite NOT granted")
	}
}

func TestToken_AllGrantsEverything(t *testing.T) {
	tok := AllToken("admin")
	for _, c := range AllCaps() {
		if !tok.Has(c) {
			t.Errorf("AllToken should grant %s", c)
		}
	}
}

func TestRequire_GrantedSilent(t *testing.T) {
	ctx := WithToken(context.Background(), NewToken("test", CapAuditRead))
	if err := Require(ctx, CapAuditRead); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestRequire_DeniedWithError(t *testing.T) {
	ctx := WithToken(context.Background(), NewToken("test", CapAuditRead))
	err := Require(ctx, CapAuditUndo)
	if err == nil {
		t.Fatal("expected error")
	}
	var capErr *Error
	if !errors.As(err, &capErr) {
		t.Fatalf("expected *Error, got %T", err)
	}
	if capErr.Required != CapAuditUndo {
		t.Errorf("expected Required=%s, got %s", CapAuditUndo, capErr.Required)
	}
}

func TestRequire_NoTokenAlwaysDenies(t *testing.T) {
	// A context with no token at all must default-deny — this is the
	// safe behavior so a developer who forgets to set the token can't
	// accidentally allow everything.
	ctx := context.Background()
	if err := Require(ctx, CapAuditRead); err == nil {
		t.Error("no-token context should deny")
	}
}

func TestSetCurrent_Switchover(t *testing.T) {
	original := Current()
	defer SetCurrent(original)

	SetCurrent(NewToken("agent:sales", CapNotifyUser))
	if err := RequireCurrent(CapNotifyUser); err != nil {
		t.Error("agent:sales should have CapNotifyUser")
	}
	if err := RequireCurrent(CapChannelWrite); err == nil {
		t.Error("agent:sales should NOT have CapChannelWrite")
	}
}
