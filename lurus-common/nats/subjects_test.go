package nats

import (
	"strings"
	"testing"
)

func TestSubjectConstants(t *testing.T) {
	// Test that all subjects start with "lurus."
	subjects := []string{
		SubjectUserCreated,
		SubjectUserUpdated,
		SubjectUserDeleted,
		SubjectUserQuotaChanged,
		SubjectUserGroupChanged,
		SubjectUserDailyQuotaReset,
		SubjectUserDailyQuotaExhausted,
		SubjectSubscriptionCreated,
		SubjectSubscriptionRenewed,
		SubjectSubscriptionCancelled,
		SubjectSubscriptionExpired,
		SubjectPaymentSucceeded,
		SubjectPaymentFailed,
		SubjectUsageRecorded,
		SubjectLLMRequestClaude,
		SubjectLLMRequestCodex,
		SubjectLLMRequestGemini,
		SubjectLLMRequestGeneric,
		SubjectLLMResponse,
		SubjectLogWrite,
		SubjectSyncSession,
		SubjectSyncMessage,
	}

	for _, subject := range subjects {
		if !strings.HasPrefix(subject, "lurus.") {
			t.Errorf("Subject %s should start with 'lurus.'", subject)
		}
	}
}

func TestStreamConstants(t *testing.T) {
	streams := []string{
		StreamLurusEvents,
		StreamLLMEvents,
		StreamLogEvents,
		StreamBillingEvents,
		StreamSyncEvents,
	}

	for _, stream := range streams {
		if stream == "" {
			t.Error("Stream name should not be empty")
		}
		// Stream names should be uppercase with underscores
		if stream != strings.ToUpper(stream) {
			t.Errorf("Stream name %s should be uppercase", stream)
		}
	}
}

func TestConsumerConstants(t *testing.T) {
	consumers := []string{
		ConsumerLogService,
		ConsumerBillingService,
		ConsumerSyncService,
		ConsumerGateway,
	}

	for _, consumer := range consumers {
		if consumer == "" {
			t.Error("Consumer name should not be empty")
		}
	}
}

func TestSubjectPattern(t *testing.T) {
	tests := []struct {
		prefix   string
		expected string
	}{
		{"lurus.user", "lurus.user.>"},
		{"lurus.subscription", "lurus.subscription.>"},
		{"lurus", "lurus.>"},
	}

	for _, tt := range tests {
		result := SubjectPattern(tt.prefix)
		if result != tt.expected {
			t.Errorf("SubjectPattern(%s) = %s, want %s", tt.prefix, result, tt.expected)
		}
	}
}

func TestLLMRequestSubject(t *testing.T) {
	tests := []struct {
		platform string
		expected string
	}{
		{"claude", SubjectLLMRequestClaude},
		{"codex", SubjectLLMRequestCodex},
		{"gemini", SubjectLLMRequestGemini},
		{"unknown", SubjectLLMRequestGeneric},
		{"", SubjectLLMRequestGeneric},
	}

	for _, tt := range tests {
		result := LLMRequestSubject(tt.platform)
		if result != tt.expected {
			t.Errorf("LLMRequestSubject(%s) = %s, want %s", tt.platform, result, tt.expected)
		}
	}
}

func TestSubjectHierarchy(t *testing.T) {
	// Test that subjects follow NATS subject hierarchy conventions
	// Format: namespace.domain.action or namespace.domain.subdomain.action
	
	// User events should be under lurus.user.*
	if !strings.HasPrefix(SubjectUserCreated, "lurus.user.") {
		t.Errorf("User events should be under lurus.user.*")
	}
	
	// Subscription events should be under lurus.subscription.*
	if !strings.HasPrefix(SubjectSubscriptionCreated, "lurus.subscription.") {
		t.Errorf("Subscription events should be under lurus.subscription.*")
	}
	
	// LLM events should be under lurus.llm.*
	if !strings.HasPrefix(SubjectLLMRequestClaude, "lurus.llm.") {
		t.Errorf("LLM events should be under lurus.llm.*")
	}
}

func BenchmarkLLMRequestSubject(b *testing.B) {
	platforms := []string{"claude", "codex", "gemini", "unknown"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		LLMRequestSubject(platforms[i%4])
	}
}
