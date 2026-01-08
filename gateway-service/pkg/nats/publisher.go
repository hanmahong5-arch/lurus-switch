package nats

import (
	"encoding/json"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/pocketzworld/lurus-switch/gateway-service/internal/conf"
	"go.uber.org/zap"
)

// Publisher publishes events to NATS
type Publisher struct {
	nc       *nats.Conn
	js       nats.JetStreamContext
	config   *conf.NATS
	subjects *conf.NATSSubject
	logger   *zap.Logger
}

// NewPublisher creates a new NATS publisher
func NewPublisher(config *conf.NATS, logger *zap.Logger) (*Publisher, func(), error) {
	nc, err := nats.Connect(config.URL,
		nats.MaxReconnects(-1),
		nats.ReconnectWait(time.Second),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			logger.Warn("NATS disconnected", zap.Error(err))
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger.Info("NATS reconnected", zap.String("url", nc.ConnectedUrl()))
		}),
	)
	if err != nil {
		return nil, nil, err
	}

	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, nil, err
	}

	publisher := &Publisher{
		nc:       nc,
		js:       js,
		config:   config,
		subjects: &config.Subjects,
		logger:   logger,
	}

	cleanup := func() {
		nc.Close()
	}

	return publisher, cleanup, nil
}

// LogEvent represents a log event
type LogEvent struct {
	ID              string    `json:"id"`
	TraceID         string    `json:"trace_id"`
	RequestID       string    `json:"request_id"`
	UserID          string    `json:"user_id"`
	Platform        string    `json:"platform"`
	Model           string    `json:"model"`
	Provider        string    `json:"provider"`
	ProviderModel   string    `json:"provider_model"`
	IsStream        bool      `json:"is_stream"`
	HTTPCode        int       `json:"http_code"`
	DurationSec     float64   `json:"duration_sec"`
	FinishReason    string    `json:"finish_reason"`
	InputTokens     int       `json:"input_tokens"`
	OutputTokens    int       `json:"output_tokens"`
	CacheReadTokens int       `json:"cache_read_tokens"`
	TotalCost       float64   `json:"total_cost"`
	ErrorType       string    `json:"error_type"`
	ErrorMessage    string    `json:"error_message"`
	CreatedAt       time.Time `json:"created_at"`
}

// PublishLogEvent publishes a log event
func (p *Publisher) PublishLogEvent(event interface{}) error {
	data, err := json.Marshal(event)
	if err != nil {
		p.logger.Error("Failed to marshal log event", zap.Error(err))
		return err
	}

	_, err = p.js.Publish(p.subjects.LogWrite, data)
	if err != nil {
		p.logger.Error("Failed to publish log event", zap.Error(err))
		return err
	}

	p.logger.Debug("Published log event", zap.String("subject", p.subjects.LogWrite))
	return nil
}

// UsageEvent represents a billing usage event
type UsageEvent struct {
	UserID       string    `json:"user_id"`
	Platform     string    `json:"platform"`
	Model        string    `json:"model"`
	Provider     string    `json:"provider"`
	InputTokens  int       `json:"input_tokens"`
	OutputTokens int       `json:"output_tokens"`
	TotalCost    float64   `json:"total_cost"`
	TraceID      string    `json:"trace_id"`
	Timestamp    time.Time `json:"timestamp"`
}

// PublishUsageEvent publishes a billing usage event
func (p *Publisher) PublishUsageEvent(event *UsageEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		p.logger.Error("Failed to marshal usage event", zap.Error(err))
		return err
	}

	_, err = p.js.Publish(p.subjects.BillingUsage, data)
	if err != nil {
		p.logger.Error("Failed to publish usage event", zap.Error(err))
		return err
	}

	p.logger.Debug("Published usage event",
		zap.String("subject", p.subjects.BillingUsage),
		zap.String("user_id", event.UserID),
	)
	return nil
}

// IsConnected returns true if connected to NATS
func (p *Publisher) IsConnected() bool {
	return p.nc.IsConnected()
}
