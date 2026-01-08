package handler

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/pocketzworld/lurus-switch/gateway-service/internal/proxy"
	"go.uber.org/zap"
)

// ClaudeHandler handles Claude API requests
type ClaudeHandler struct {
	relay  *proxy.RelayService
	logger *zap.Logger
}

// NewClaudeHandler creates a new Claude handler
func NewClaudeHandler(relay *proxy.RelayService, logger *zap.Logger) *ClaudeHandler {
	return &ClaudeHandler{
		relay:  relay,
		logger: logger,
	}
}

// Messages handles POST /v1/messages (Claude API)
func (h *ClaudeHandler) Messages(ctx context.Context, c *app.RequestContext) {
	h.logger.Debug("Handling Claude messages request",
		zap.String("method", string(c.Method())),
		zap.String("path", string(c.Path())),
	)

	if err := h.relay.ForwardRequest(ctx, c, proxy.PlatformClaude, "/v1/messages"); err != nil {
		h.logger.Error("Claude request failed", zap.Error(err))
	}
}

// CountTokens handles POST /v1/messages/count_tokens
func (h *ClaudeHandler) CountTokens(ctx context.Context, c *app.RequestContext) {
	h.logger.Debug("Handling Claude count tokens request")

	if err := h.relay.ForwardRequest(ctx, c, proxy.PlatformClaude, "/v1/messages/count_tokens"); err != nil {
		h.logger.Error("Claude count tokens request failed", zap.Error(err))
	}
}

// Batches handles POST /v1/messages/batches
func (h *ClaudeHandler) Batches(ctx context.Context, c *app.RequestContext) {
	h.logger.Debug("Handling Claude batches request")

	if err := h.relay.ForwardRequest(ctx, c, proxy.PlatformClaude, "/v1/messages/batches"); err != nil {
		h.logger.Error("Claude batches request failed", zap.Error(err))
	}
}
