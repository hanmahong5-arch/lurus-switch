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
	"lurus-switch/internal/obs"
)

const (
	maxRequestBodySize = 10 << 20 // 10 MB
	upstreamTimeout    = 5 * time.Minute
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

	// Streaming budget-wall fix: OpenAI-protocol streaming responses only
	// emit a usage chunk when the request asks for it via
	// stream_options.include_usage. Without it the sseUsageScanner finds no
	// usage line, books 0 tokens, and the budget guard never increments —
	// i.e. OpenAI streaming silently bypasses the spend wall. (The Anthropic
	// path requests usage during translation, so only Anthropic was covered
	// before.) Inject include_usage for streaming requests so the existing
	// scanner has a usage line to read.
	body = ensureStreamUsage(body)

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
			r.Context(), r.Method, r.URL.Path, r.URL.RawQuery,
			body, outHeaders,
			chain,
		)
	} else {
		resp, servedBy, err = s.fallback.TryUpstream(
			r.Context(), r.Method, r.URL.Path, r.URL.RawQuery,
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
						r.Context(), r.Method, r.URL.Path, r.URL.RawQuery,
						rectifiedBody, outHeaders,
						chain,
					)
				} else {
					retryResp, _, retryErr = s.fallback.TryUpstream(
						r.Context(), r.Method, r.URL.Path, r.URL.RawQuery,
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
	s.recordUsage(meta, model, usage, resp.StatusCode, false)
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

	var scanner sseUsageScanner
	buf := make([]byte, 4096)

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			w.Write(chunk)
			flusher.Flush()
			scanner.feed(chunk)
		}
		if readErr != nil {
			break
		}
	}

	s.recordUsage(meta, model, scanner.finish(), resp.StatusCode, true)
}

// --- metering helpers ---

func (s *Server) recordUsage(meta *RequestMeta, model string, usage UsageFromResponse, statusCode int, streaming bool) {
	if s.meter == nil || meta == nil {
		return
	}
	if usage.Model != "" {
		model = usage.Model
	}

	// Normalize the OpenAI-shape usage into billable streams. OpenAI reports
	// cached_tokens as a SUBSET already inside prompt_tokens, so billing
	// prompt_tokens as fresh input AND cached_tokens as cache-read would
	// double-count the cached portion. Subtract it out: fresh input is
	// (prompt − cached), and cached lands on the discounted cache-read stream.
	// CacheCreate is 0 — OpenAI doesn't bill a separate cache-write. Reasoning
	// is display-only (already inside completion_tokens), never billed twice.
	tokensIn := usage.PromptTokens - usage.CachedTokens
	if tokensIn < 0 {
		tokensIn = 0 // defensive: a malformed upstream can't drive billing negative
	}

	rec := metering.Record{
		AppID:             meta.AppID,
		Model:             model,
		TokensIn:          tokensIn,
		TokensOut:         usage.CompletionTokens,
		CacheReadTokens:   usage.CachedTokens,
		ReasoningTokens:   usage.ReasoningTokens,
		LatencyMs:         time.Since(meta.StartTime).Milliseconds(),
		StatusCode:        statusCode,
		Timestamp:         time.Now(),
		// Enterprise dimensions — empty in Personal/Reseller installs.
		EmployeeID: meta.OwnerEmployeeID,
		CostCenter: meta.CostCenter,
		// Routing — populated when the relay router served this request.
		ServedBy:  meta.ServedBy,
		MatchedBy: meta.MatchedBy,
	}
	s.meter.Record(rec)

	// Mirror the same facts into the optional OTel recorder (no-op unless
	// observability is enabled). Built from rec so the two stay consistent.
	s.observe(obs.RequestObservation{
		Operation:  "chat",
		Model:      model,
		ServedBy:   meta.ServedBy,
		MatchedBy:  meta.MatchedBy,
		TokensIn:   rec.TokensIn,
		TokensOut:  rec.TokensOut,
		StartTime:  meta.StartTime,
		LatencyMs:  rec.LatencyMs,
		StatusCode: statusCode,
		Streaming:  streaming,
	})

	// Feed the budget guard so its session counter stays in sync. The
	// daily counter delegates to the metering store, so no double-counting.
	// The wall counts the full prompt (cached included) so a cache-heavy
	// session still trips the cap on real upstream volume.
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

	s.observe(obs.RequestObservation{
		Operation:  "chat",
		Model:      model,
		ServedBy:   meta.ServedBy,
		MatchedBy:  meta.MatchedBy,
		StartTime:  meta.StartTime,
		LatencyMs:  rec.LatencyMs,
		StatusCode: rec.StatusCode,
		Err:        errMsg,
	})
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

// ensureStreamUsage rewrites an OpenAI-protocol request body so that a
// streaming request (stream:true) carries stream_options.include_usage=true.
// This is what makes the upstream emit the trailing usage chunk that
// sseUsageScanner reads to feed metering + the budget guard.
//
// It is a careful no-op in every case where it shouldn't touch the body:
//   - body isn't a JSON object (e.g. a non-chat passthrough) → returned as-is
//   - the request is not streaming (no "stream":true) → returned as-is
//   - include_usage is already true → returned as-is (idempotent; re-runs are
//     equivalent, and a caller that explicitly set it keeps its value)
//
// Other fields are preserved verbatim by round-tripping through a generic
// map, so we don't drop unknown OpenAI parameters.
func ensureStreamUsage(body []byte) []byte {
	if len(body) == 0 {
		return body
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(body, &obj); err != nil {
		return body // not a JSON object — leave untouched
	}
	// Only act on streaming requests.
	var streaming bool
	if raw, ok := obj["stream"]; ok {
		_ = json.Unmarshal(raw, &streaming)
	}
	if !streaming {
		return body
	}
	// Merge include_usage:true into any existing stream_options, preserving
	// other keys the caller set (e.g. stream_options.continuous_usage_stats).
	opts := map[string]json.RawMessage{}
	if raw, ok := obj["stream_options"]; ok {
		// If stream_options is present but not an object, replace it.
		_ = json.Unmarshal(raw, &opts)
		if opts == nil {
			opts = map[string]json.RawMessage{}
		}
	}
	if cur, ok := opts["include_usage"]; ok {
		var iu bool
		if json.Unmarshal(cur, &iu) == nil && iu {
			return body // already requesting usage — idempotent no-op
		}
	}
	opts["include_usage"] = json.RawMessage("true")
	optsRaw, err := json.Marshal(opts)
	if err != nil {
		return body
	}
	obj["stream_options"] = optsRaw
	out, err := json.Marshal(obj)
	if err != nil {
		return body
	}
	return out
}

func extractUsageFromBody(body []byte) UsageFromResponse {
	var resp struct {
		Model string `json:"model"`
		Usage struct {
			PromptTokens        int64 `json:"prompt_tokens"`
			CompletionTokens    int64 `json:"completion_tokens"`
			TotalTokens         int64 `json:"total_tokens"`
			PromptTokensDetails struct {
				CachedTokens int64 `json:"cached_tokens"`
			} `json:"prompt_tokens_details"`
			CompletionTokensDetails struct {
				ReasoningTokens int64 `json:"reasoning_tokens"`
			} `json:"completion_tokens_details"`
		} `json:"usage"`
	}
	if json.Unmarshal(body, &resp) == nil {
		return UsageFromResponse{
			Model:            resp.Model,
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
			CachedTokens:     resp.Usage.PromptTokensDetails.CachedTokens,
			ReasoningTokens:  resp.Usage.CompletionTokensDetails.ReasoningTokens,
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

// usageNonZero reports whether any usage counter was actually populated.
// Some OpenAI-compatible providers send prompt/completion tokens without a
// total_tokens field, so keying solely on TotalTokens would drop their
// usage — checking any positive field avoids that metering leak.
func usageNonZero(u UsageFromResponse) bool {
	return u.TotalTokens > 0 || u.PromptTokens > 0 || u.CompletionTokens > 0
}

// maxSSELineBuf caps the per-line accumulation buffer. SSE lines always end
// in '\n', so the cap is never hit in practice — it is purely a memory
// backstop against a pathological newline-less upstream stream.
const maxSSELineBuf = 1 << 20 // 1 MB

// sseUsageScanner accumulates raw stream bytes across Read() boundaries so
// usage extraction operates on COMPLETE SSE lines. A naive per-chunk scan
// loses the usage line whenever its bytes straddle a read boundary — the
// common case for streaming chat — which silently books 0 tokens. The
// scanner keeps the last non-zero usage it observes (OpenAI emits usage in
// the penultimate "data: {...}" line when stream_options.include_usage is
// set). It is single-goroutine: the proxy feeds it one chunk at a time.
type sseUsageScanner struct {
	buf  []byte            // unterminated trailing bytes not yet scanned
	last UsageFromResponse // most recent non-zero usage seen
}

// feed appends a chunk and scans every newly completed line for usage.
func (s *sseUsageScanner) feed(chunk []byte) {
	s.buf = append(s.buf, chunk...)
	if idx := bytes.LastIndexByte(s.buf, '\n'); idx >= 0 {
		if u := extractUsageFromSSEChunk(s.buf[:idx+1]); usageNonZero(u) {
			s.last = u
		}
		// Retain only the unterminated remainder after the last newline.
		s.buf = append(s.buf[:0], s.buf[idx+1:]...)
	}
	if len(s.buf) > maxSSELineBuf {
		s.buf = s.buf[:0]
	}
}

// finish scans any trailing line left unterminated when the stream ended and
// returns the final observed usage.
func (s *sseUsageScanner) finish() UsageFromResponse {
	if len(s.buf) > 0 {
		if u := extractUsageFromSSEChunk(s.buf); usageNonZero(u) {
			s.last = u
		}
	}
	return s.last
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
				PromptTokens        int64 `json:"prompt_tokens"`
				CompletionTokens    int64 `json:"completion_tokens"`
				TotalTokens         int64 `json:"total_tokens"`
				PromptTokensDetails struct {
					CachedTokens int64 `json:"cached_tokens"`
				} `json:"prompt_tokens_details"`
				CompletionTokensDetails struct {
					ReasoningTokens int64 `json:"reasoning_tokens"`
				} `json:"completion_tokens_details"`
			} `json:"usage"`
		}
		if json.Unmarshal(data, &msg) == nil && msg.Usage != nil {
			return UsageFromResponse{
				Model:            msg.Model,
				PromptTokens:     msg.Usage.PromptTokens,
				CompletionTokens: msg.Usage.CompletionTokens,
				TotalTokens:      msg.Usage.TotalTokens,
				CachedTokens:     msg.Usage.PromptTokensDetails.CachedTokens,
				ReasoningTokens:  msg.Usage.CompletionTokensDetails.ReasoningTokens,
			}
		}
	}
	return UsageFromResponse{}
}
