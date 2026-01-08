package handler

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/pocketzworld/lurus-switch/gateway-service/internal/proxy"
	"go.uber.org/zap"
)

// CodexHandler handles Codex/OpenAI API requests
type CodexHandler struct {
	relay  *proxy.RelayService
	logger *zap.Logger
}

// NewCodexHandler creates a new Codex handler
func NewCodexHandler(relay *proxy.RelayService, logger *zap.Logger) *CodexHandler {
	return &CodexHandler{
		relay:  relay,
		logger: logger,
	}
}

// Responses handles POST /responses (OpenAI Responses API)
func (h *CodexHandler) Responses(ctx context.Context, c *app.RequestContext) {
	h.logger.Debug("Handling Codex responses request",
		zap.String("method", string(c.Method())),
		zap.String("path", string(c.Path())),
	)

	if err := h.relay.ForwardRequest(ctx, c, proxy.PlatformCodex, "/responses"); err != nil {
		h.logger.Error("Codex responses request failed", zap.Error(err))
	}
}

// ChatCompletions handles POST /v1/chat/completions
func (h *CodexHandler) ChatCompletions(ctx context.Context, c *app.RequestContext) {
	h.logger.Debug("Handling chat completions request")

	if err := h.relay.ForwardRequest(ctx, c, proxy.PlatformCodex, "/v1/chat/completions"); err != nil {
		h.logger.Error("Chat completions request failed", zap.Error(err))
	}
}

// ChatCompletionsAlt handles POST /chat/completions (without /v1 prefix)
func (h *CodexHandler) ChatCompletionsAlt(ctx context.Context, c *app.RequestContext) {
	h.logger.Debug("Handling chat completions request (alt)")

	if err := h.relay.ForwardRequest(ctx, c, proxy.PlatformCodex, "/v1/chat/completions"); err != nil {
		h.logger.Error("Chat completions request failed", zap.Error(err))
	}
}

// Completions handles POST /v1/completions (legacy)
func (h *CodexHandler) Completions(ctx context.Context, c *app.RequestContext) {
	h.logger.Debug("Handling completions request")

	if err := h.relay.ForwardRequest(ctx, c, proxy.PlatformCodex, "/v1/completions"); err != nil {
		h.logger.Error("Completions request failed", zap.Error(err))
	}
}

// Embeddings handles POST /v1/embeddings
func (h *CodexHandler) Embeddings(ctx context.Context, c *app.RequestContext) {
	h.logger.Debug("Handling embeddings request")

	if err := h.relay.ForwardRequest(ctx, c, proxy.PlatformCodex, "/v1/embeddings"); err != nil {
		h.logger.Error("Embeddings request failed", zap.Error(err))
	}
}
