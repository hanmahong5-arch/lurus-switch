package publisher

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/pocketzworld/lurus-switch/billing-service/internal/middleware"
	"go.uber.org/zap"
)

// EventType defines the type of billing event
type EventType string

const (
	EventQuotaUpdated   EventType = "quota.updated"
	EventBalanceChanged EventType = "balance.changed"
	EventUsageRecorded  EventType = "usage.recorded"
	EventQuotaLow       EventType = "quota.low"
	EventQuotaExhausted EventType = "quota.exhausted"
)

// BillingEvent represents a billing event for NATS
type BillingEvent struct {
	Type      EventType   `json:"type"`
	UserID    string      `json:"user_id"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// QuotaUpdateData contains quota update details
type QuotaUpdateData struct {
	QuotaLimit     int64   `json:"quota_limit"`
	QuotaUsed      int64   `json:"quota_used"`
	QuotaRemaining int64   `json:"quota_remaining"`
	PercentUsed    float64 `json:"percent_used"`
}

// BalanceChangeData contains balance change details
type BalanceChangeData struct {
	Balance     float64 `json:"balance"`
	Change      float64 `json:"change"`
	Reason      string  `json:"reason"`
	ReferenceID string  `json:"reference_id,omitempty"`
}

// UsageRecordedData contains usage record details
type UsageRecordedData struct {
	Platform     string  `json:"platform"`
	Model        string  `json:"model"`
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	Cost         float64 `json:"cost"`
	TraceID      string  `json:"trace_id,omitempty"`
}

// Publisher handles NATS event publishing for billing
type Publisher struct {
	conn   *nats.Conn
	logger *zap.Logger
}

// NewPublisher creates a new NATS publisher
func NewPublisher(natsURL string, logger *zap.Logger) (*Publisher, error) {
	opts := []nats.Option{
		nats.Name("billing-service-publisher"),
		nats.ReconnectWait(2 * time.Second),
		nats.MaxReconnects(-1),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				logger.Warn("NATS disconnected", zap.Error(err))
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger.Info("NATS reconnected", zap.String("url", nc.ConnectedUrl()))
		}),
	}

	conn, err := nats.Connect(natsURL, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	logger.Info("Connected to NATS for publishing", zap.String("url", natsURL))

	return &Publisher{
		conn:   conn,
		logger: logger,
	}, nil
}

// Close closes the NATS connection
func (p *Publisher) Close() {
	if p.conn != nil {
		p.conn.Close()
	}
}

// PublishQuotaUpdate publishes a quota update event
func (p *Publisher) PublishQuotaUpdate(userID string, quotaLimit, quotaUsed int64) error {
	quotaRemaining := quotaLimit - quotaUsed
	percentUsed := float64(quotaUsed) / float64(quotaLimit) * 100

	event := BillingEvent{
		Type:      EventQuotaUpdated,
		UserID:    userID,
		Timestamp: time.Now(),
		Data: QuotaUpdateData{
			QuotaLimit:     quotaLimit,
			QuotaUsed:      quotaUsed,
			QuotaRemaining: quotaRemaining,
			PercentUsed:    percentUsed,
		},
	}

	if err := p.publish(userID, event); err != nil {
		return err
	}

	// Check for low quota warning (80% used)
	if percentUsed >= 80 && percentUsed < 100 {
		p.PublishQuotaLow(userID, quotaRemaining, percentUsed)
	}

	// Check for quota exhausted (100% used)
	if percentUsed >= 100 {
		p.PublishQuotaExhausted(userID)
	}

	middleware.RecordNATSEvent(string(EventQuotaUpdated))
	return nil
}

// PublishBalanceChange publishes a balance change event
func (p *Publisher) PublishBalanceChange(userID string, balance, change float64, reason, referenceID string) error {
	event := BillingEvent{
		Type:      EventBalanceChanged,
		UserID:    userID,
		Timestamp: time.Now(),
		Data: BalanceChangeData{
			Balance:     balance,
			Change:      change,
			Reason:      reason,
			ReferenceID: referenceID,
		},
	}

	if err := p.publish(userID, event); err != nil {
		return err
	}

	middleware.RecordNATSEvent(string(EventBalanceChanged))
	return nil
}

// PublishUsageRecorded publishes a usage recorded event
func (p *Publisher) PublishUsageRecorded(userID, platform, model string, inputTokens, outputTokens int, cost float64, traceID string) error {
	event := BillingEvent{
		Type:      EventUsageRecorded,
		UserID:    userID,
		Timestamp: time.Now(),
		Data: UsageRecordedData{
			Platform:     platform,
			Model:        model,
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
			Cost:         cost,
			TraceID:      traceID,
		},
	}

	if err := p.publish(userID, event); err != nil {
		return err
	}

	middleware.RecordNATSEvent(string(EventUsageRecorded))
	return nil
}

// PublishQuotaLow publishes a low quota warning event
func (p *Publisher) PublishQuotaLow(userID string, remaining int64, percentUsed float64) error {
	event := BillingEvent{
		Type:      EventQuotaLow,
		UserID:    userID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"remaining":    remaining,
			"percent_used": percentUsed,
			"message":      fmt.Sprintf("Quota is %.1f%% used, %d tokens remaining", percentUsed, remaining),
		},
	}

	if err := p.publish(userID, event); err != nil {
		return err
	}

	middleware.RecordNATSEvent(string(EventQuotaLow))
	middleware.RecordLowBalanceAlert()
	return nil
}

// PublishQuotaExhausted publishes a quota exhausted event
func (p *Publisher) PublishQuotaExhausted(userID string) error {
	event := BillingEvent{
		Type:      EventQuotaExhausted,
		UserID:    userID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"message": "Quota exhausted. Please upgrade or wait for reset.",
		},
	}

	if err := p.publish(userID, event); err != nil {
		return err
	}

	middleware.RecordNATSEvent(string(EventQuotaExhausted))
	return nil
}

// publish sends an event to the user's billing channel
func (p *Publisher) publish(userID string, event BillingEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		p.logger.Error("Failed to marshal billing event",
			zap.Error(err),
			zap.String("user_id", userID),
			zap.String("event_type", string(event.Type)),
		)
		return err
	}

	// Publish to user-specific subject
	subject := fmt.Sprintf("billing.%s", userID)
	if err := p.conn.Publish(subject, data); err != nil {
		p.logger.Error("Failed to publish billing event",
			zap.Error(err),
			zap.String("subject", subject),
			zap.String("event_type", string(event.Type)),
		)
		return err
	}

	p.logger.Debug("Published billing event",
		zap.String("subject", subject),
		zap.String("event_type", string(event.Type)),
		zap.String("user_id", userID),
	)

	return nil
}
