package consumer

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/pocketzworld/lurus-switch/log-service/internal/biz"
	"github.com/pocketzworld/lurus-switch/log-service/internal/conf"
	"go.uber.org/zap"
)

// NATSConsumer consumes log events from NATS
type NATSConsumer struct {
	nc       *nats.Conn
	js       nats.JetStreamContext
	sub      *nats.Subscription
	uc       *biz.LogUsecase
	config   *conf.NATS
	logger   *zap.Logger
	buffer   []*biz.RequestLog
	bufferMu sync.Mutex
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

// NewNATSConsumer creates a new NATS consumer
func NewNATSConsumer(uc *biz.LogUsecase, config *conf.NATS, logger *zap.Logger) (*NATSConsumer, func(), error) {
	if !config.Enabled {
		logger.Info("NATS consumer is disabled")
		return nil, func() {}, nil
	}

	// Connect to NATS
	nc, err := nats.Connect(config.URL,
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
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

	logger.Info("Connected to NATS", zap.String("url", nc.ConnectedUrl()))

	// Initialize JetStream
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
		buffer: make([]*biz.RequestLog, 0, config.BatchSize),
		stopCh: make(chan struct{}),
	}

	cleanup := func() {
		logger.Info("Stopping NATS consumer...")
		close(consumer.stopCh)
		consumer.wg.Wait()
		if consumer.sub != nil {
			consumer.sub.Unsubscribe()
		}
		nc.Close()
	}

	return consumer, cleanup, nil
}

// Start starts consuming messages
func (c *NATSConsumer) Start(ctx context.Context) error {
	if c == nil || c.nc == nil {
		return nil
	}

	// Subscribe to log.write subject
	sub, err := c.js.Subscribe(
		c.config.Subject,
		c.handleMessage,
		nats.Durable(c.config.ConsumerName),
		nats.ManualAck(),
		nats.AckWait(30*time.Second),
		nats.MaxDeliver(3),
	)
	if err != nil {
		// Fallback to regular subscription if JetStream not available
		c.logger.Warn("JetStream subscription failed, falling back to regular subscription", zap.Error(err))
		sub, err = c.nc.Subscribe(c.config.Subject, func(msg *nats.Msg) {
			c.handleMessage(msg)
			msg.Ack()
		})
		if err != nil {
			return err
		}
	}

	c.sub = sub
	c.logger.Info("Subscribed to NATS subject", zap.String("subject", c.config.Subject))

	// Start batch flush goroutine
	c.wg.Add(1)
	go c.batchFlushLoop(ctx)

	return nil
}

// handleMessage handles a single NATS message
func (c *NATSConsumer) handleMessage(msg *nats.Msg) {
	var log biz.RequestLog
	if err := json.Unmarshal(msg.Data, &log); err != nil {
		c.logger.Warn("Failed to unmarshal log message", zap.Error(err))
		msg.Ack() // Ack to avoid redelivery of malformed messages
		return
	}

	c.bufferMu.Lock()
	c.buffer = append(c.buffer, &log)
	shouldFlush := len(c.buffer) >= c.config.BatchSize
	c.bufferMu.Unlock()

	if shouldFlush {
		c.flush(context.Background())
	}

	msg.Ack()
}

// batchFlushLoop periodically flushes the buffer
func (c *NATSConsumer) batchFlushLoop(ctx context.Context) {
	defer c.wg.Done()

	ticker := time.NewTicker(c.config.BatchTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			c.flush(context.Background()) // Final flush
			return
		case <-c.stopCh:
			c.flush(context.Background()) // Final flush
			return
		case <-ticker.C:
			c.flush(ctx)
		}
	}
}

// flush writes buffered logs to database
func (c *NATSConsumer) flush(ctx context.Context) {
	c.bufferMu.Lock()
	if len(c.buffer) == 0 {
		c.bufferMu.Unlock()
		return
	}

	logs := c.buffer
	c.buffer = make([]*biz.RequestLog, 0, c.config.BatchSize)
	c.bufferMu.Unlock()

	written, failed, err := c.uc.WriteLogs(ctx, logs)
	if err != nil {
		c.logger.Error("Failed to flush log batch",
			zap.Error(err),
			zap.Int("written", written),
			zap.Int("failed", failed),
		)
	} else {
		c.logger.Debug("Flushed log batch", zap.Int("count", written))
	}
}
