package middleware

import (
	"context"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"go.uber.org/zap"
)

// Logger returns a logging middleware
func Logger(logger *zap.Logger) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		start := time.Now()
		path := string(c.Path())
		method := string(c.Method())

		// Process request
		c.Next(ctx)

		// Log after request
		latency := time.Since(start)
		statusCode := c.Response.StatusCode()

		logger.Info("HTTP request",
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("status", statusCode),
			zap.Duration("latency", latency),
			zap.String("client_ip", c.ClientIP()),
			zap.Int("body_size", len(c.Response.Body())),
		)
	}
}
