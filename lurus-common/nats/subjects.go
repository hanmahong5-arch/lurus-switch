// Package nats provides NATS-related utilities and subject definitions.
package nats

// Subject constants define the NATS subjects used across Lurus services.
const (
	// User events
	SubjectUserCreated          = "lurus.user.created"
	SubjectUserUpdated          = "lurus.user.updated"
	SubjectUserDeleted          = "lurus.user.deleted"
	SubjectUserQuotaChanged     = "lurus.user.quota.changed"
	SubjectUserGroupChanged     = "lurus.user.group.changed"
	SubjectUserDailyQuotaReset  = "lurus.user.daily_quota.reset"
	SubjectUserDailyQuotaExhausted = "lurus.user.daily_quota.exhausted"
	
	// Subscription events
	SubjectSubscriptionCreated  = "lurus.subscription.created"
	SubjectSubscriptionRenewed  = "lurus.subscription.renewed"
	SubjectSubscriptionCancelled = "lurus.subscription.cancelled"
	SubjectSubscriptionExpired  = "lurus.subscription.expired"
	
	// Billing events
	SubjectPaymentSucceeded     = "lurus.payment.succeeded"
	SubjectPaymentFailed        = "lurus.payment.failed"
	SubjectUsageRecorded        = "lurus.usage.recorded"
	
	// LLM request events (by platform)
	SubjectLLMRequestClaude     = "lurus.llm.request.claude"
	SubjectLLMRequestCodex      = "lurus.llm.request.codex"
	SubjectLLMRequestGemini     = "lurus.llm.request.gemini"
	SubjectLLMRequestGeneric    = "lurus.llm.request.generic"
	SubjectLLMResponse          = "lurus.llm.response"
	
	// Log events
	SubjectLogWrite             = "lurus.log.write"
	
	// Sync events
	SubjectSyncSession          = "lurus.sync.session"
	SubjectSyncMessage          = "lurus.sync.message"
)

// Stream names for JetStream
const (
	StreamLurusEvents   = "LURUS_EVENTS"
	StreamLLMEvents     = "LLM_EVENTS"
	StreamLogEvents     = "LOG_EVENTS"
	StreamBillingEvents = "BILLING_EVENTS"
	StreamSyncEvents    = "SYNC_EVENTS"
)

// Consumer names
const (
	ConsumerLogService    = "log-service"
	ConsumerBillingService = "billing-service"
	ConsumerSyncService   = "sync-service"
	ConsumerGateway       = "gateway-service"
)

// SubjectPattern returns a wildcard pattern for subjects.
func SubjectPattern(prefix string) string {
	return prefix + ".>"
}

// LLMRequestSubject returns the subject for an LLM request by platform.
func LLMRequestSubject(platform string) string {
	switch platform {
	case "claude":
		return SubjectLLMRequestClaude
	case "codex":
		return SubjectLLMRequestCodex
	case "gemini":
		return SubjectLLMRequestGemini
	default:
		return SubjectLLMRequestGeneric
	}
}
