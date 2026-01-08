package consumer

import (
	"context"
	"encoding/json"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/pocketzworld/lurus-switch/billing-service/internal/biz"
	"github.com/pocketzworld/lurus-switch/billing-service/internal/conf"
	"go.uber.org/zap"
)

// NATSConsumer consumes usage events from NATS
type NATSConsumer struct {
	nc       *nats.Conn
	js       nats.JetStreamContext
	uc       *biz.BillingUsecase
	config   *conf.NATS
	logger   *zap.Logger
	sub      *nats.Subscription
}

// UsageEvent represents a usage event from Gateway
type UsageEvent struct {
	UserID       string    `json:"user_id"`
	TraceID      string    `json:"trace_id"`
	Platform     string    `json:"platform"`
	Model        string    `json:"model"`
	Provider     string    `json:"provider"`
	InputTokens  int       `json:"input_tokens"`
	OutputTokens int       `json:"output_tokens"`
	TotalCost    float64   `json:"total_cost"`
	Timestamp    time.Time `json:"timestamp"`
}

// NewNATSConsumer creates a new NATS consumer
func NewNATSConsumer(uc *biz.BillingUsecase, config *conf.NATS, logger *zap.Logger) (*NATSConsumer, func(), error) {
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

	consumer := &NATSConsumer{
		nc:     nc,
		js:     js,
		uc:     uc,
		config: config,
		logger: logger,
	}

	cleanup := func() {
		if consumer.sub != nil {
			consumer.sub.Unsubscribe()
		}
		nc.Close()
	}

	return consumer, cleanup, nil
}

// Start starts consuming usage events
func (c *NATSConsumer) Start(ctx context.Context) error {
	// Create durable consumer
	sub, err := c.js.PullSubscribe(
		c.config.Subjects.BillingUsage,
		"billing-service",
		nats.AckExplicit(),
		nats.MaxDeliver(3),
		nats.AckWait(30*time.Second),
	)
	if err != nil {
		// Try regular subscription if JetStream not available
		c.logger.Warn("JetStream not available, using regular subscription", zap.Error(err))
		return c.startRegularSubscription(ctx)
	}

	c.sub = sub
	c.logger.Info("Started NATS consumer",
		zap.String("subject", c.config.Subjects.BillingUsage),
	)

	// Start consumer loop
	go c.consumeLoop(ctx, sub)

	return nil
}

// consumeLoop processes messages from JetStream
func (c *NATSConsumer) consumeLoop(ctx context.Context, sub *nats.Subscription) {
	for {
		select {
		case <-ctx.Done():
			c.logger.Info("NATS consumer stopped")
			return
		default:
			// Fetch messages
			msgs, err := sub.Fetch(10, nats.MaxWait(5*time.Second))
			if err != nil {
				if err != nats.ErrTimeout {
					c.logger.Error("Failed to fetch messages", zap.Error(err))
				}
				continue
			}

			for _, msg := range msgs {
				c.handleMessage(ctx, msg)
			}
		}
	}
}

// startRegularSubscription uses regular NATS subscription
func (c *NATSConsumer) startRegularSubscription(ctx context.Context) error {
	sub, err := c.nc.Subscribe(c.config.Subjects.BillingUsage, func(msg *nats.Msg) {
		c.handleRegularMessage(ctx, msg)
	})
	if err != nil {
		return err
	}

	c.sub = sub
	c.logger.Info("Started regular NATS subscription",
		zap.String("subject", c.config.Subjects.BillingUsage),
	)

	return nil
}

// handleMessage processes a single JetStream message
func (c *NATSConsumer) handleMessage(ctx context.Context, msg *nats.Msg) {
	var event UsageEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		c.logger.Error("Failed to unmarshal usage event", zap.Error(err))
		msg.Nak()
		return
	}

	// Record usage
	usage := &biz.UsageRecord{
		ID:           generateID(),
		UserID:       event.UserID,
		TraceID:      event.TraceID,
		Platform:     event.Platform,
		Model:        event.Model,
		Provider:     event.Provider,
		InputTokens:  event.InputTokens,
		OutputTokens: event.OutputTokens,
		TotalCost:    event.TotalCost,
		CreatedAt:    event.Timestamp,
	}

	if err := c.uc.RecordUsage(ctx, usage); err != nil {
		c.logger.Error("Failed to record usage", zap.Error(err))
		msg.Nak()
		return
	}

	msg.Ack()
	c.logger.Debug("Recorded usage",
		zap.String("user_id", event.UserID),
		zap.String("trace_id", event.TraceID),
		zap.Int("tokens", event.InputTokens+event.OutputTokens),
	)
}

// handleRegularMessage processes a regular NATS message
func (c *NATSConsumer) handleRegularMessage(ctx context.Context, msg *nats.Msg) {
	var event UsageEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		c.logger.Error("Failed to unmarshal usage event", zap.Error(err))
		return
	}

	usage := &biz.UsageRecord{
		ID:           generateID(),
		UserID:       event.UserID,
		TraceID:      event.TraceID,
		Platform:     event.Platform,
		Model:        event.Model,
		Provider:     event.Provider,
		InputTokens:  event.InputTokens,
		OutputTokens: event.OutputTokens,
		TotalCost:    event.TotalCost,
		CreatedAt:    event.Timestamp,
	}

	if err := c.uc.RecordUsage(ctx, usage); err != nil {
		c.logger.Error("Failed to record usage", zap.Error(err))
	}
}

func generateID() string {
	return time.Now().Format("20060102150405") + "_" + randomString(8)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}
