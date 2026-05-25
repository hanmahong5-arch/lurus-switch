package gateway

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"lurus-switch/internal/metering"
)

const (
	maxRequestBodySize  = 10 << 20 // 10 MB
	upstreamTimeout     = 5 * time.Minute
)

// handleProxy forwards the request to the upstream LLM provider,
// swapping the per-app token for the user's Lurus Cloud token.
// Supports both streaming (SSE) and non-streaming responses.
func (s *Server) handleProxy(w http.ResponseWriter, r *http.Request) {
	s.activeReqs.Add(1)
	defer s.activeReqs.Add(-1)
	s.totalReqs.Add(1)

	meta := getMeta(r)

	// Read request body (needed to parse model name for metering).
	body, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBodySize))
	if err != nil {
		writeOpenAIError(w, http.StatusBadRequest, "invalid_request", "failed to read request body")
		return
	}
	r.Body.Close()

	// DLP middleware — scan the raw body before any further processing.
	// Block policy returns 451 immediately; redact policy swaps the body
	// so downstream forwarding (and metering) sees the masked version.
	body, dlpBlocked, dlpReason := s.applyDLPRequest(body, r.URL.Path)
	if dlpBlocked {
		writeOpenAIError(w, http.StatusUnavailableForLegalReasons, "dlp_blocked", dlpReason)
		s.recordError(meta, "", dlpReason)
		return
	}

	// Extract model from request body for metering.
	model := extractModelFromBody(body)
	if meta != nil {
		meta.Model = model
	}

	// Check upstream is configured.
	s.mu.Lock()
	upstreamURL := s.cfg.UpstreamURL
	userToken := s.cfg.UserToken
	s.mu.Unlock()

	if upstreamURL == "" {
		writeOpenAIError(w, http.StatusServiceUnavailable, "gateway_not_configured",
			"Gateway upstream not configured. Open Lurus Switch settings to set your API endpoint.")
		return
	}
	if userToken == "" {
		writeOpenAIError(w, http.StatusPaymentRequired, "no_balance",
			"No Lurus account connected. Open Lurus Switch to log in and add balance.")
		return
	}

	// Active Budget Wall — bail out *before* paying upstream tokens
	// when the user-configured spend cap is already reached.
	s.mu.Lock()
	guard := s.guard
	s.mu.Unlock()
	if guard != nil {
		if v := guard.Check(); !v.Allowed {
			writeOpenAIError(w, http.StatusTooManyRequests, "spend_cap_reached",
				fmt.Sprintf("Lurus Switch budget wall: %s. Raise the limit or click 'reset session' in the Budget panel.", v.Reason))
			s.recordError(meta, model, v.Reason)
			return
		}
	}

	// Normalize base URL: strip /v1 to prevent path duplication.
	normalizedURL := NormalizeChannelBaseURL(upstreamURL)

	// Collect request headers for upstream (swap auth token).
	outHeaders := make(http.Header)
	copyRequestHeaders2(outHeaders, r)

	// Build the upstream chain. If the relay router has healthy
	// endpoints + a matching rule (or tool→mapping), use that as the
	// authoritative chain. Otherwise fall back to the cfg-driven path
	// (UpstreamURL + persisted FallbackChain entries) for zero
	// behaviour change in unconfigured installs.
	chain, matchedBy, routerOK := s.buildChainFromRouter(
		toolFromRequest(r),
		model,
		estimateTokens(body),
		bodyHasTools(body),
		userToken,
	)

	var resp *http.Response
	var servedBy string
	if routerOK {
		if meta != nil {
			meta.MatchedBy = matchedBy
		}
		resp, servedBy, err = s.fallback.TryUpstreamChain(
			r.Method, r.URL.Path, r.URL.RawQuery,
			body, outHeaders,
			chain,
		)
	} else {
		resp, servedBy, err = s.fallback.TryUpstream(
			r.Method, r.URL.Path, r.URL.RawQuery,
			body, outHeaders,
			normalizedURL, userToken,
		)
	}
	if err != nil {
		writeOpenAIError(w, http.StatusBadGateway, "upstream_error",
			fmt.Sprintf("all upstreams failed: %v", err))
		s.recordError(meta, model, err.Error())
		return
	}
	defer resp.Body.Close()

	if meta != nil {
		meta.ServedBy = servedBy
	}

	// Thinking Budget Rectifier: if upstream returns a budget constraint error,
	// auto-fix the request and retry once (inspired by CC-Switch).
	if resp.StatusCode == http.StatusBadRequest {
		errBody, readErr := io.ReadAll(io.LimitReader(resp.Body, 4096))
		resp.Body.Close()
		if readErr == nil && ShouldRectifyThinkingBudget(string(errBody)) {
			rectifiedBody, result := RectifyThinkingBudget(body)
			if result.Applied {
				var retryResp *http.Response
				var retryErr error
				// Retry via the same chain we used the first time.
				if routerOK {
					retryResp, _, retryErr = s.fallback.TryUpstreamChain(
						r.Method, r.URL.Path, r.URL.RawQuery,
						rectifiedBody, outHeaders,
						chain,
					)
				} else {
					retryResp, _, retryErr = s.fallback.TryUpstream(
						r.Method, r.URL.Path, r.URL.RawQuery,
						rectifiedBody, outHeaders,
						normalizedURL, userToken,
					)
				}
				if retryErr == nil {
					defer retryResp.Body.Close()
					resp = retryResp
					body = rectifiedBody
					// Fall through to normal response handling below
					goto handleResponse
				}
			}
		}
		// Budget rectifier didn't apply or retry failed — return original error
		copyResponseHeaders(w, resp)
		w.WriteHeader(resp.StatusCode)
		w.Write(errBody)
		s.recordError(meta, model, string(errBody))
		return
	}

handleResponse:
	// Determine if this is a streaming response.
	isStreaming := strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream")

	if isStreaming {
		s.proxyStreaming(w, resp, meta, model)
	} else {
		s.proxyBuffered(w, resp, meta, model)
	}
}

// proxyBuffered handles non-streaming responses: read full body, extract usage, forward.
func (s *Server) proxyBuffered(w http.ResponseWriter, resp *http.Response, meta *RequestMeta, model string) {
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxRequestBodySize))
	if err != nil {
		writeOpenAIError(w, http.StatusBadGateway, "upstream_read_error",
			"failed to read upstream response")
		return
	}

	// Copy response headers.
	copyResponseHeaders(w, resp)
	w.WriteHeader(resp.StatusCode)
	w.Write(respBody)

	// Extract usage for metering.
	usage := extractUsageFromBody(respBody)
	s.recordUsage(meta, model, usage, resp.StatusCode)
}

// proxyStreaming pipes SSE chunks from upstream to client in real-time.
func (s *Server) proxyStreaming(w http.ResponseWriter, resp *http.Response, meta *RequestMeta, model string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		// Fallback to buffered if flushing is not supported.
		s.proxyBuffered(w, resp, meta, model)
		return
	}

	// Copy response headers.
	copyResponseHeaders(w, resp)
	w.WriteHeader(resp.StatusCode)

	var totalUsage UsageFromResponse
	buf := make([]byte, 4096)

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			w.Write(chunk)
			flusher.Flush()

			// Try to extract usage from the final SSE data chunk.
			// OpenAI sends usage in the last "data: {...}" line when stream_options.include_usage is set.
			if u := extractUsageFromSSEChunk(chunk); u.TotalTokens > 0 {
				totalUsage = u
			}
		}
		if readErr != nil {
			break
		}
	}

	s.recordUsage(meta, model, totalUsage, resp.StatusCode)
}

// --- metering helpers ---

func (s *Server) recordUsage(meta *RequestMeta, model string, usage UsageFromResponse, statusCode int) {
	if s.meter == nil || meta == nil {
		return
	}
	if usage.Model != "" {
		model = usage.Model
	}
	rec := metering.Record{
		AppID:      meta.AppID,
		Model:      model,
		TokensIn:   usage.PromptTokens,
		TokensOut:  usage.CompletionTokens,
		LatencyMs:  time.Since(meta.StartTime).Milliseconds(),
		StatusCode: statusCode,
		Timestamp:  time.Now(),
		// Enterprise dimensions — empty in Personal/Reseller installs.
		EmployeeID: meta.OwnerEmployeeID,
		CostCenter: meta.CostCenter,
		// Routing — populated when the relay router served this request.
		ServedBy:  meta.ServedBy,
		MatchedBy: meta.MatchedBy,
	}
	s.meter.Record(rec)

	// Feed the budget guard so its session counter stays in sync. The
	// daily counter delegates to the metering store, so no double-counting.
	s.mu.Lock()
	guard := s.guard
	s.mu.Unlock()
	if guard != nil {
		guard.RecordUsage(usage.PromptTokens, usage.CompletionTokens)
	}
}

func (s *Server) recordError(meta *RequestMeta, model, errMsg string) {
	if s.meter == nil || meta == nil {
		return
	}
	rec := metering.Record{
		AppID:        meta.AppID,
		Model:        model,
		LatencyMs:    time.Since(meta.StartTime).Milliseconds(),
		StatusCode:   502,
		ErrorMessage: errMsg,
		Timestamp:    time.Now(),
		ServedBy:     meta.ServedBy,
		MatchedBy:    meta.MatchedBy,
	}
	s.meter.Record(rec)
}

// --- request/response helpers ---

var proxiedHeaders = []string{
	"Content-Type", "Accept", "User-Agent",
	"X-Request-ID", "X-Stainless-Arch", "X-Stainless-Lang",
	"X-Stainless-OS", "X-Stainless-Package-Version",
	"X-Stainless-Runtime", "X-Stainless-Runtime-Version",
}

func copyRequestHeaders(dst, src *http.Request) {
	for _, key := range proxiedHeaders {
		if v := src.Header.Get(key); v != "" {
			dst.Header.Set(key, v)
		}
	}
	dst.Header.Set("Content-Length", src.Header.Get("Content-Length"))
}

// copyRequestHeaders2 copies proxied headers from an http.Request into an http.Header map.
// Used by FallbackChain which needs a standalone header set (not tied to a single request).
func copyRequestHeaders2(dst http.Header, src *http.Request) {
	for _, key := range proxiedHeaders {
		if v := src.Header.Get(key); v != "" {
			dst.Set(key, v)
		}
	}
	if cl := src.Header.Get("Content-Length"); cl != "" {
		dst.Set("Content-Length", cl)
	}
}

func copyResponseHeaders(w http.ResponseWriter, resp *http.Response) {
	for _, key := range []string{
		"Content-Type", "X-Request-ID", "X-RateLimit-Limit-Requests",
		"X-RateLimit-Limit-Tokens", "X-RateLimit-Remaining-Requests",
		"X-RateLimit-Remaining-Tokens", "X-RateLimit-Reset-Requests",
		"X-RateLimit-Reset-Tokens", "OpenAI-Processing-Ms",
	} {
		if v := resp.Header.Get(key); v != "" {
			w.Header().Set(key, v)
		}
	}
}

// --- JSON extraction helpers ---

func extractModelFromBody(body []byte) string {
	var req struct {
		Model string `json:"model"`
	}
	if json.Unmarshal(body, &req) == nil {
		return req.Model
	}
	return ""
}

func extractUsageFromBody(body []byte) UsageFromResponse {
	var resp struct {
		Model string `json:"model"`
		Usage struct {
			PromptTokens     int64 `json:"prompt_tokens"`
			CompletionTokens int64 `json:"completion_tokens"`
			TotalTokens      int64 `json:"total_tokens"`
		} `json:"usage"`
	}
	if json.Unmarshal(body, &resp) == nil {
		return UsageFromResponse{
			Model:            resp.Model,
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		}
	}
	return UsageFromResponse{}
}

// estimateTokens is a cheap byte-length heuristic used to feed the
// router's PickHint.EstimatedInputTokens predicate. Rules of thumb:
// English ≈ 4 chars/tok, code ≈ 3.5 chars/tok — using /4 keeps the
// estimate conservative (slightly under-counts) which is fine for
// triggering "if >= 50k tokens, prefer long-context endpoint" rules.
// Exact tokenisation is the upstream's job, not the router's.
func estimateTokens(body []byte) int64 {
	if len(body) == 0 {
		return 0
	}
	return int64(len(body)) / 4
}

// bodyHasTools reports whether the request body declares a non-empty
// "tools" array (OpenAI tools / Anthropic tool-use shape). Routers can
// use this to steer tool-calling traffic to endpoints that support it.
func bodyHasTools(body []byte) bool {
	var probe struct {
		Tools []json.RawMessage `json:"tools"`
	}
	if err := json.Unmarshal(body, &probe); err != nil {
		return false
	}
	return len(probe.Tools) > 0
}

// toolFromRequest sniffs the User-Agent to guess which CLI sent the
// request (claude / codex / gemini / picoclaw / …). Returns "" when
// the User-Agent is missing or unrecognised — Router.Pick handles ""
// safely (no tool→mapping match, rule predicates still apply).
func toolFromRequest(r *http.Request) string {
	ua := strings.ToLower(r.Header.Get("User-Agent"))
	if ua == "" {
		return ""
	}
	switch {
	case strings.Contains(ua, "claude"):
		return "claude"
	case strings.Contains(ua, "codex"):
		return "codex"
	case strings.Contains(ua, "gemini"):
		return "gemini"
	case strings.Contains(ua, "picoclaw"):
		return "picoclaw"
	case strings.Contains(ua, "nullclaw"):
		return "nullclaw"
	case strings.Contains(ua, "openclaw"):
		return "openclaw"
	}
	return ""
}

// extractUsageFromSSEChunk attempts to parse usage from an SSE data line.
// SSE format: "data: {json}\n\n"
func extractUsageFromSSEChunk(chunk []byte) UsageFromResponse {
	lines := bytes.Split(chunk, []byte("\n"))
	for _, line := range lines {
		line = bytes.TrimSpace(line)
		if !bytes.HasPrefix(line, []byte("data: ")) {
			continue
		}
		data := line[6:] // strip "data: "
		if bytes.Equal(data, []byte("[DONE]")) {
			continue
		}
		var msg struct {
			Model string `json:"model"`
			Usage *struct {
				PromptTokens     int64 `json:"prompt_tokens"`
				CompletionTokens int64 `json:"completion_tokens"`
				TotalTokens      int64 `json:"total_tokens"`
			} `json:"usage"`
		}
		if json.Unmarshal(data, &msg) == nil && msg.Usage != nil {
			return UsageFromResponse{
				Model:            msg.Model,
				PromptTokens:     msg.Usage.PromptTokens,
				CompletionTokens: msg.Usage.CompletionTokens,
				TotalTokens:      msg.Usage.TotalTokens,
			}
		}
	}
	return UsageFromResponse{}
}
