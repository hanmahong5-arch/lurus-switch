package dlp

import (
	"strings"
	"testing"
)

func TestDefaultPatternsCompile(t *testing.T) {
	s := NewScanner()
	if len(s.Patterns()) == 0 {
		t.Fatal("default scanner should have patterns")
	}
}

func TestScan_OpenAIKeyBlocks(t *testing.T) {
	s := NewScanner()
	res := s.Scan("here is my key: sk-abcdefghijklmnopqrstuvwxyz1234567890")
	if !res.Blocked {
		t.Error("expected Blocked=true for OpenAI key")
	}
	if res.HighestPolicy != PolicyBlock {
		t.Errorf("HighestPolicy=%s, want block", res.HighestPolicy)
	}
	if len(res.Hits) == 0 {
		t.Fatal("expected at least one hit")
	}
}

func TestScan_CreditCardRedacts(t *testing.T) {
	s := NewScanner()
	input := "card 4111-1111-1111-1111 expires soon"
	res := s.Scan(input)
	if res.Blocked {
		t.Error("CC pattern should redact, not block")
	}
	if !strings.Contains(res.Redacted, "[REDACTED:pii.credit_card]") {
		t.Errorf("expected redaction marker, got %q", res.Redacted)
	}
}

func TestScan_CleanInput(t *testing.T) {
	s := NewScanner()
	res := s.Scan("hello, please summarize this document about quarterly revenue")
	for _, h := range res.Hits {
		// Email/phone false-positives shouldn't kick in here. If they do, the
		// pattern is too greedy.
		t.Errorf("unexpected hit: %+v", h)
	}
	if res.Blocked {
		t.Error("clean input should not be blocked")
	}
}

func TestScan_AnonymizedSnippet(t *testing.T) {
	s := NewScanner()
	res := s.Scan("contact: alice.smith@example.com please")
	if len(res.Hits) == 0 {
		t.Fatal("expected email hit")
	}
	for _, h := range res.Hits {
		if h.PatternName != "pii.email" {
			continue
		}
		if h.Snippet == "alice.smith@example.com" {
			t.Errorf("snippet should be anonymized, got %q", h.Snippet)
		}
	}
}

func TestSetPolicy_TightensFromWarnToBlock(t *testing.T) {
	s := NewScanner()
	if !s.SetPolicy("pii.email", PolicyBlock) {
		t.Fatal("SetPolicy returned false")
	}
	res := s.Scan("contact: alice.smith@example.com")
	if !res.Blocked {
		t.Error("expected block after policy tightening")
	}
}

func TestAdd_RejectsDuplicateName(t *testing.T) {
	s := NewScanner()
	err := s.Add(Pattern{Name: "pii.email", Regex: `xx`, Policy: PolicyAllow})
	if err == nil {
		t.Error("expected duplicate-name error")
	}
}

func TestAdd_RejectsBadRegex(t *testing.T) {
	s := NewScanner()
	err := s.Add(Pattern{Name: "broken", Regex: `(`, Policy: PolicyAllow})
	if err == nil {
		t.Error("expected regex compile error")
	}
}

func TestRemove(t *testing.T) {
	s := NewScanner()
	if !s.Remove("pii.email") {
		t.Fatal("expected Remove to find pii.email")
	}
	res := s.Scan("contact: x@y.com")
	for _, h := range res.Hits {
		if h.PatternName == "pii.email" {
			t.Error("expected pii.email pattern removed")
		}
	}
}

func TestPatterns_DropsCompiledField(t *testing.T) {
	s := NewScanner()
	for _, p := range s.Patterns() {
		if p.compiled != nil {
			t.Errorf("Patterns() should drop compiled, got %+v", p)
		}
	}
}

func TestScan_RedactionPreservesOrder(t *testing.T) {
	s := NewScanner()
	input := "ssn 123-45-6789 then card 4111111111111111 then more text"
	res := s.Scan(input)
	if !strings.Contains(res.Redacted, "REDACTED:pii.ssn_us") {
		t.Error("missing SSN redaction")
	}
	if !strings.Contains(res.Redacted, "REDACTED:pii.credit_card") {
		t.Error("missing CC redaction")
	}
	// Order: SSN appears before CC in input → must appear in same order in output.
	ssnIdx := strings.Index(res.Redacted, "pii.ssn_us")
	ccIdx := strings.Index(res.Redacted, "pii.credit_card")
	if ssnIdx >= ccIdx {
		t.Error("redaction should preserve match order")
	}
}
