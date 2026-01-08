package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// Client wraps NATS connection with enhanced features
type Client struct {
	nc     *nats.Conn
	js     nats.JetStreamContext
	config *Config
	logger *zap.Logger
	mu     sync.RWMutex
	subs   map[string]*nats.Subscription
}

// NewClient creates a new NATS client
func NewClient(cfg *Config, logger *zap.Logger) *Client {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	if logger == nil {
		logger, _ = zap.NewProduction()
	}
	return &Client{
		config: cfg,
		logger: logger,
		subs:   make(map[string]*nats.Subscription),
	}
}

// Connect establishes connection to NATS server
func (c *Client) Connect(ctx context.Context) error {
	if !c.config.Enabled {
		c.logger.Info("NATS is disabled, skipping connection")
		return nil
	}

	opts := c.buildOptions()

	nc, err := nats.Connect(c.config.URL, opts...)
	if err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}

	c.nc = nc
	c.logger.Info("Connected to NATS", zap.String("url", nc.ConnectedUrl()))

	// Initialize JetStream
	js, err := nc.JetStream()
	if err != nil {
		c.logger.Warn("JetStream not available", zap.Error(err))
	} else {
		c.js = js
		c.logger.Info("JetStream enabled")
	}

	return nil
}

// buildOptions constructs NATS connection options
func (c *Client) buildOptions() []nats.Option {
	opts := []nats.Option{
		nats.MaxReconnects(c.config.MaxReconnects),
		nats.ReconnectWait(c.config.ReconnectWait),
		nats.ReconnectBufSize(c.config.ReconnectBufferSize),
		nats.Timeout(c.config.ConnectTimeout),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			c.logger.Warn("NATS disconnected", zap.Error(err))
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			c.logger.Info("NATS reconnected", zap.String("url", nc.ConnectedUrl()))
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			c.logger.Info("NATS connection closed")
		}),
		nats.ErrorHandler(func(nc *nats.Conn, sub *nats.Subscription, err error) {
			c.logger.Error("NATS error",
				zap.String("subject", sub.Subject),
				zap.Error(err))
		}),
	}

	// Authentication
	if c.config.Token != "" {
		opts = append(opts, nats.Token(c.config.Token))
	} else if c.config.Username != "" {
		opts = append(opts, nats.UserInfo(c.config.Username, c.config.Password))
	}

	return opts
}

// Close closes the NATS connection
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for subject, sub := range c.subs {
		if err := sub.Unsubscribe(); err != nil {
			c.logger.Warn("Failed to unsubscribe", zap.String("subject", subject), zap.Error(err))
		}
	}
	c.subs = make(map[string]*nats.Subscription)

	if c.nc != nil {
		c.nc.Close()
		c.nc = nil
	}
	return nil
}

// IsConnected returns true if connected to NATS
func (c *Client) IsConnected() bool {
	return c.nc != nil && c.nc.IsConnected()
}

// IsEnabled returns true if NATS is enabled
func (c *Client) IsEnabled() bool {
	return c.config.Enabled
}

// Conn returns the underlying NATS connection
func (c *Client) Conn() *nats.Conn {
	return c.nc
}

// JetStream returns the JetStream context
func (c *Client) JetStream() nats.JetStreamContext {
	return c.js
}

// Publish publishes a message to a subject
func (c *Client) Publish(subject string, data interface{}) error {
	if !c.IsConnected() {
		return nil // Silent fail when not connected
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	return c.nc.Publish(subject, payload)
}

// PublishRaw publishes raw bytes to a subject
func (c *Client) PublishRaw(subject string, data []byte) error {
	if !c.IsConnected() {
		return nil
	}
	return c.nc.Publish(subject, data)
}

// PublishAsync publishes a message asynchronously using JetStream
func (c *Client) PublishAsync(subject string, data interface{}) (nats.PubAckFuture, error) {
	if !c.IsConnected() || c.js == nil {
		return nil, nil
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	return c.js.PublishAsync(subject, payload)
}

// PublishWithAck publishes and waits for acknowledgment (JetStream)
func (c *Client) PublishWithAck(ctx context.Context, subject string, data interface{}) error {
	if !c.IsConnected() {
		return nil
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	if c.js == nil {
		return c.nc.Publish(subject, payload)
	}

	_, err = c.js.Publish(subject, payload, nats.Context(ctx))
	return err
}

// Request sends a request and waits for a response
func (c *Client) Request(ctx context.Context, subject string, data interface{}) (*nats.Msg, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("not connected")
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	return c.nc.RequestWithContext(ctx, subject, payload)
}

// Subscribe subscribes to a subject
func (c *Client) Subscribe(subject string, handler nats.MsgHandler) error {
	if !c.IsConnected() {
		return fmt.Errorf("not connected")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Unsubscribe existing subscription
	if existing, exists := c.subs[subject]; exists {
		existing.Unsubscribe()
		delete(c.subs, subject)
	}

	sub, err := c.nc.Subscribe(subject, handler)
	if err != nil {
		return err
	}

	c.subs[subject] = sub
	return nil
}

// QueueSubscribe subscribes to a subject with a queue group
func (c *Client) QueueSubscribe(subject, queue string, handler nats.MsgHandler) error {
	if !c.IsConnected() {
		return fmt.Errorf("not connected")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	subKey := subject + ":" + queue
	if existing, exists := c.subs[subKey]; exists {
		existing.Unsubscribe()
		delete(c.subs, subKey)
	}

	sub, err := c.nc.QueueSubscribe(subject, queue, handler)
	if err != nil {
		return err
	}

	c.subs[subKey] = sub
	return nil
}

// Unsubscribe removes a subscription
func (c *Client) Unsubscribe(subject string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if sub, exists := c.subs[subject]; exists {
		if err := sub.Unsubscribe(); err != nil {
			return err
		}
		delete(c.subs, subject)
	}
	return nil
}

// Flush flushes the connection
func (c *Client) Flush() error {
	if !c.IsConnected() {
		return nil
	}
	return c.nc.Flush()
}

// FlushTimeout flushes with timeout
func (c *Client) FlushTimeout(timeout time.Duration) error {
	if !c.IsConnected() {
		return nil
	}
	return c.nc.FlushTimeout(timeout)
}
