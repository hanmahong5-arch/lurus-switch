package handler

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/pocketzworld/lurus-switch/gateway-service/internal/proxy"
	"go.uber.org/zap"
)

// GeminiHandler handles Gemini API requests
type GeminiHandler struct {
	relay  *proxy.RelayService
	logger *zap.Logger
}

// NewGeminiHandler creates a new Gemini handler
func NewGeminiHandler(relay *proxy.RelayService, logger *zap.Logger) *GeminiHandler {
	return &GeminiHandler{
		relay:  relay,
		logger: logger,
	}
}

// GenerateContent handles POST /v1beta/models/{model}:generateContent
func (h *GeminiHandler) GenerateContent(ctx context.Context, c *app.RequestContext) {
	model := c.Param("model")
	h.logger.Debug("Handling Gemini generateContent request",
		zap.String("model", model),
	)

	endpoint := fmt.Sprintf("/v1beta/models/%s:generateContent", model)
	if err := h.relay.ForwardRequest(ctx, c, proxy.PlatformGemini, endpoint); err != nil {
		h.logger.Error("Gemini generateContent request failed", zap.Error(err))
	}
}

// StreamGenerateContent handles POST /v1beta/models/{model}:streamGenerateContent
func (h *GeminiHandler) StreamGenerateContent(ctx context.Context, c *app.RequestContext) {
	model := c.Param("model")
	h.logger.Debug("Handling Gemini streamGenerateContent request",
		zap.String("model", model),
	)

	endpoint := fmt.Sprintf("/v1beta/models/%s:streamGenerateContent", model)
	if err := h.relay.ForwardRequest(ctx, c, proxy.PlatformGemini, endpoint); err != nil {
		h.logger.Error("Gemini streamGenerateContent request failed", zap.Error(err))
	}
}

// CountTokens handles POST /v1beta/models/{model}:countTokens
func (h *GeminiHandler) CountTokens(ctx context.Context, c *app.RequestContext) {
	model := c.Param("model")
	h.logger.Debug("Handling Gemini countTokens request",
		zap.String("model", model),
	)

	endpoint := fmt.Sprintf("/v1beta/models/%s:countTokens", model)
	if err := h.relay.ForwardRequest(ctx, c, proxy.PlatformGemini, endpoint); err != nil {
		h.logger.Error("Gemini countTokens request failed", zap.Error(err))
	}
}

// Models handles GET /v1beta/models
func (h *GeminiHandler) Models(ctx context.Context, c *app.RequestContext) {
	h.logger.Debug("Handling Gemini models list request")

	if err := h.relay.ForwardRequest(ctx, c, proxy.PlatformGemini, "/v1beta/models"); err != nil {
		h.logger.Error("Gemini models list request failed", zap.Error(err))
	}
}

// HandleModelAction handles POST /v1beta/models/{model}:{action}
// The wildcard captures the full "model:action" pattern
func (h *GeminiHandler) HandleModelAction(ctx context.Context, c *app.RequestContext) {
	modelAction := c.Param("modelAction")
	// Remove leading slash if present
	modelAction = strings.TrimPrefix(modelAction, "/")

	h.logger.Debug("Handling Gemini model action request",
		zap.String("modelAction", modelAction),
	)

	endpoint := fmt.Sprintf("/v1beta/models/%s", modelAction)
	if err := h.relay.ForwardRequest(ctx, c, proxy.PlatformGemini, endpoint); err != nil {
		h.logger.Error("Gemini model action request failed", zap.Error(err))
	}
}

// GenericEndpoint handles any /v1beta/* endpoint dynamically
func (h *GeminiHandler) GenericEndpoint(ctx context.Context, c *app.RequestContext) {
	path := string(c.Path())
	h.logger.Debug("Handling Gemini generic endpoint",
		zap.String("path", path),
	)

	// Extract the endpoint from path
	endpoint := path
	if strings.HasPrefix(endpoint, "/gemini") {
		endpoint = strings.TrimPrefix(endpoint, "/gemini")
	}

	if err := h.relay.ForwardRequest(ctx, c, proxy.PlatformGemini, endpoint); err != nil {
		h.logger.Error("Gemini generic endpoint request failed", zap.Error(err))
	}
}
