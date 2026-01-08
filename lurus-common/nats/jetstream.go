package nats

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// JetStreamManager manages JetStream streams and consumers
type JetStreamManager struct {
	js     nats.JetStreamContext
	logger *zap.Logger
}

// NewJetStreamManager creates a new JetStream manager
func NewJetStreamManager(js nats.JetStreamContext, logger *zap.Logger) *JetStreamManager {
	if logger == nil {
		logger, _ = zap.NewProduction()
	}
	return &JetStreamManager{
		js:     js,
		logger: logger,
	}
}

// EnsureStream creates or updates a stream
func (m *JetStreamManager) EnsureStream(ctx context.Context, cfg StreamConfig) (*nats.StreamInfo, error) {
	if m.js == nil {
		return nil, fmt.Errorf("JetStream not available")
	}

	// Check if stream exists
	info, err := m.js.StreamInfo(cfg.Name, nats.Context(ctx))
	if err == nil {
		m.logger.Debug("Stream already exists", zap.String("name", cfg.Name))
		return info, nil
	}

	// Create stream
	jsCfg := &nats.StreamConfig{
		Name:        cfg.Name,
		Subjects:    cfg.Subjects,
		MaxAge:      cfg.MaxAge,
		MaxBytes:    cfg.MaxBytes,
		MaxMsgs:     cfg.MaxMsgs,
		Replicas:    cfg.Replicas,
		Description: cfg.Description,
	}

	// Set retention policy
	switch cfg.Retention {
	case "interest":
		jsCfg.Retention = nats.InterestPolicy
	case "workqueue":
		jsCfg.Retention = nats.WorkQueuePolicy
	default:
		jsCfg.Retention = nats.LimitsPolicy
	}

	// Set storage type
	switch cfg.Storage {
	case "memory":
		jsCfg.Storage = nats.MemoryStorage
	default:
		jsCfg.Storage = nats.FileStorage
	}

	info, err = m.js.AddStream(jsCfg, nats.Context(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to create stream %s: %w", cfg.Name, err)
	}

	m.logger.Info("Stream created",
		zap.String("name", cfg.Name),
		zap.Strings("subjects", cfg.Subjects))
	return info, nil
}

// EnsureStreams creates multiple streams
func (m *JetStreamManager) EnsureStreams(ctx context.Context, configs []StreamConfig) error {
	for _, cfg := range configs {
		if _, err := m.EnsureStream(ctx, cfg); err != nil {
			return err
		}
	}
	return nil
}

// DeleteStream deletes a stream
func (m *JetStreamManager) DeleteStream(ctx context.Context, name string) error {
	if m.js == nil {
		return fmt.Errorf("JetStream not available")
	}
	return m.js.DeleteStream(name, nats.Context(ctx))
}

// StreamInfo returns information about a stream
func (m *JetStreamManager) StreamInfo(ctx context.Context, name string) (*nats.StreamInfo, error) {
	if m.js == nil {
		return nil, fmt.Errorf("JetStream not available")
	}
	return m.js.StreamInfo(name, nats.Context(ctx))
}

// ConsumerConfig represents a JetStream consumer configuration
type ConsumerConfig struct {
	Stream        string        `json:"stream"`
	Name          string        `json:"name"`
	Durable       string        `json:"durable"`
	FilterSubject string        `json:"filter_subject"`
	AckWait       time.Duration `json:"ack_wait"`
	MaxDeliver    int           `json:"max_deliver"`
	MaxAckPending int           `json:"max_ack_pending"`
	DeliverPolicy string        `json:"deliver_policy"` // all, last, new, by_start_sequence, by_start_time
}

// EnsureConsumer creates or updates a consumer
func (m *JetStreamManager) EnsureConsumer(ctx context.Context, cfg ConsumerConfig) (*nats.ConsumerInfo, error) {
	if m.js == nil {
		return nil, fmt.Errorf("JetStream not available")
	}

	// Check if consumer exists
	info, err := m.js.ConsumerInfo(cfg.Stream, cfg.Name, nats.Context(ctx))
	if err == nil {
		return info, nil
	}

	// Create consumer
	jsCfg := &nats.ConsumerConfig{
		Name:          cfg.Name,
		Durable:       cfg.Durable,
		FilterSubject: cfg.FilterSubject,
		AckWait:       cfg.AckWait,
		MaxDeliver:    cfg.MaxDeliver,
		MaxAckPending: cfg.MaxAckPending,
		AckPolicy:     nats.AckExplicitPolicy,
	}

	// Set delivery policy
	switch cfg.DeliverPolicy {
	case "last":
		jsCfg.DeliverPolicy = nats.DeliverLastPolicy
	case "new":
		jsCfg.DeliverPolicy = nats.DeliverNewPolicy
	default:
		jsCfg.DeliverPolicy = nats.DeliverAllPolicy
	}

	info, err = m.js.AddConsumer(cfg.Stream, jsCfg, nats.Context(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer %s: %w", cfg.Name, err)
	}

	m.logger.Info("Consumer created",
		zap.String("stream", cfg.Stream),
		zap.String("name", cfg.Name))
	return info, nil
}

// PullSubscribe creates a pull subscription
func (m *JetStreamManager) PullSubscribe(subject, durable string, opts ...nats.SubOpt) (*nats.Subscription, error) {
	if m.js == nil {
		return nil, fmt.Errorf("JetStream not available")
	}
	return m.js.PullSubscribe(subject, durable, opts...)
}

// Subscribe creates a push subscription
func (m *JetStreamManager) Subscribe(subject string, handler nats.MsgHandler, opts ...nats.SubOpt) (*nats.Subscription, error) {
	if m.js == nil {
		return nil, fmt.Errorf("JetStream not available")
	}
	return m.js.Subscribe(subject, handler, opts...)
}

// QueueSubscribe creates a queue subscription
func (m *JetStreamManager) QueueSubscribe(subject, queue string, handler nats.MsgHandler, opts ...nats.SubOpt) (*nats.Subscription, error) {
	if m.js == nil {
		return nil, fmt.Errorf("JetStream not available")
	}
	return m.js.QueueSubscribe(subject, queue, handler, opts...)
}

// Publish publishes a message to JetStream
func (m *JetStreamManager) Publish(subject string, data []byte, opts ...nats.PubOpt) (*nats.PubAck, error) {
	if m.js == nil {
		return nil, fmt.Errorf("JetStream not available")
	}
	return m.js.Publish(subject, data, opts...)
}

// PublishAsync publishes a message asynchronously
func (m *JetStreamManager) PublishAsync(subject string, data []byte, opts ...nats.PubOpt) (nats.PubAckFuture, error) {
	if m.js == nil {
		return nil, fmt.Errorf("JetStream not available")
	}
	return m.js.PublishAsync(subject, data, opts...)
}
