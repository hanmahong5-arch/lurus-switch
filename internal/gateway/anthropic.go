package gateway

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"lurus-switch/internal/budget"
	"lurus-switch/internal/metering"
	"lurus-switch/internal/obs"
	"lurus-switch/internal/translator"
)

// handleAnthropicMessages bridges Claude Code (Anthropic Messages API
// /v1/messages) to whatever OpenAI-compatible upstream the gateway is
// configured to talk to. Translation is performed in two phases:
//
//	  request:  Anthropic JSON → OpenAI JSON  → upstream
//	  response: upstream OpenAI → Anthropic   → client
//
// Both Bash-Guard and Budget Wall integrations live higher up the
// stack so they continue to work — Budget Wall checks before we
// forward, Bash-Guard runs as a CLI-side hook regardless.
func (s *Server) handleAnthropicMessages(w http.ResponseWriter, r *http.Request) {
	s.activeReqs.Add(1)
	defer s.activeReqs.Add(-1)
	s.totalReqs.Add(1)

	meta := getMeta(r)

	rawBody, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBodySize))
	if err != nil {
		writeAnthropicError(w, http.StatusBadRequest, "invalid_request_error",
			"failed to read request body")
		return
	}
	r.Body.Close()

	// DLP middleware — scan the raw Anthropic body before translation.
	rawBody, dlpBlocked, dlpReason := s.applyDLPRequest(rawBody, r.URL.Path)
	if dlpBlocked {
		writeAnthropicError(w, http.StatusUnavailableForLegalReasons, "permission_error", dlpReason)
		return
	}

	var req translator.AnthropicRequest
	if err := json.Unmarshal(rawBody, &req); err != nil {
		writeAnthropicError(w, http.StatusBadRequest, "invalid_request_error",
			"malformed Anthropic request body: "+err.Error())
		return
	}
	model := req.Model
	if meta != nil {
		meta.Model = model
	}

	// Budget Wall — bail out before paying any tokens upstream.
	s.mu.Lock()
	guard := s.guard
	upstreamURL := s.cfg.UpstreamURL
	userToken := s.cfg.UserToken
	s.mu.Unlock()
	if guard != nil {
		if v := guard.Check(); !v.Allowed {
			writeAnthropicError(w, http.StatusTooManyRequests, "rate_limit_error",
				fmt.Sprintf("Lurus Switch budget wall: %s. Raise the limit or click 'reset session' in the Budget panel.", v.Reason))
			s.recordError(meta, model, v.Reason)
			return
		}
	}
	if upstreamURL == "" {
		writeAnthropicError(w, http.StatusServiceUnavailable, "api_error",
			"Gateway upstream not configured. Open Lurus Switch settings to set your API endpoint.")
		return
	}
	if userToken == "" {
		writeAnthropicError(w, http.StatusPaymentRequired, "authentication_error",
			"No upstream API key configured. Open Account → Connection in Lurus Switch.")
		return
	}

	// Translate Anthropic → OpenAI.
	openAIReq, err := translator.RequestToOpenAI(&req)
	if err != nil {
		writeAnthropicError(w, http.StatusBadRequest, "invalid_request_error", err.Error())
		return
	}
	openAIBody, err := json.Marshal(openAIReq)
	if err != nil {
		writeAnthropicError(w, http.StatusInternalServerError, "api_error",
			"translator marshal failed: "+err.Error())
		return
	}

	// Forward via existing fallback chain — same retry / fallback chain
	// the OpenAI-protocol path uses. Prefer the relay router's ordered
	// chain when wired so /v1/messages traffic obeys the same routing
	// rules as the OpenAI-protocol path.
	normalizedURL := NormalizeChannelBaseURL(upstreamURL)
	outHeaders := make(http.Header)
	copyRequestHeaders2(outHeaders, r)
	outHeaders.Set("Content-Type", "application/json")

	chain, matchedBy, routerOK := s.buildChainFromRouter(
		toolFromRequest(r),
		model,
		estimateTokens(openAIBody),
		bodyHasTools(openAIBody),
		userToken,
	)

	var resp *http.Response
	var servedBy string
	if routerOK {
		if meta != nil {
			meta.MatchedBy = matchedBy
		}
		resp, servedBy, err = s.fallback.TryUpstreamChain(
			"POST", "/v1/chat/completions", "",
			openAIBody, outHeaders,
			chain,
		)
	} else {
		resp, servedBy, err = s.fallback.TryUpstream(
			"POST", "/v1/chat/completions", "",
			openAIBody, outHeaders,
			normalizedURL, userToken,
		)
	}
	if err != nil {
		writeAnthropicError(w, http.StatusBadGateway, "api_error",
			fmt.Sprintf("upstream error: %v", err))
		s.recordError(meta, model, err.Error())
		return
	}
	defer resp.Body.Close()
	if meta != nil {
		meta.ServedBy = servedBy
	}

	// Forward 4xx/5xx from upstream back as Anthropic errors so Claude
	// Code sees an error envelope it understands.
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
		writeAnthropicError(w, resp.StatusCode, anthropicErrTypeForStatus(resp.StatusCode),
			fmt.Sprintf("upstream %d: %s", resp.StatusCode, string(body)))
		s.recordError(meta, model, string(body))
		return
	}

	isStreaming := req.Stream &&
		strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream")
	if isStreaming {
		s.streamAnthropic(w, resp, meta, model, &req, guard)
	} else {
		s.bufferedAnthropic(w, resp, meta, model, &req, guard)
	}
}

func (s *Server) bufferedAnthropic(
	w http.ResponseWriter, resp *http.Response,
	meta *RequestMeta, model string, req *translator.AnthropicRequest,
	guard *budget.Guard,
) {
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxRequestBodySize))
	if err != nil {
		writeAnthropicError(w, http.StatusBadGateway, "api_error",
			"read upstream body: "+err.Error())
		return
	}
	var openAIResp translator.OpenAIResponse
	if err := json.Unmarshal(body, &openAIResp); err != nil {
		writeAnthropicError(w, http.StatusBadGateway, "api_error",
			"decode upstream OpenAI response: "+err.Error())
		return
	}
	anthResp := translator.ResponseToAnthropic(&openAIResp, model)
	out, _ := json.Marshal(anthResp)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(out)

	// Account.
	s.recordAnthropicUsage(meta, model, openAIResp.Usage, resp.StatusCode, false)
	if guard != nil {
		guard.RecordUsage(int64(openAIResp.Usage.PromptTokens), int64(openAIResp.Usage.CompletionTokens))
	}
	_ = req
}

func (s *Server) streamAnthropic(
	w http.ResponseWriter, resp *http.Response,
	meta *RequestMeta, model string, req *translator.AnthropicRequest,
	guard *budget.Guard,
) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		// Fallback: degrade to buffered.
		s.bufferedAnthropic(w, resp, meta, model, req, guard)
		return
	}

	// Anthropic SSE shape — let downstream HTTP infra know.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	tr := translator.NewStreamTranslator(
		"msg_"+strings.ReplaceAll(meta.AppID+"-"+time.Now().Format("150405"), " ", ""),
		model, 0,
	)
	if err := tr.Run(resp.Body, w, flusher.Flush); err != nil {
		// Stream connection broken — best we can do is log via metering.
		s.recordError(meta, model, "anthropic stream: "+err.Error())
		return
	}
	_ = req

	// Account streaming usage. The translator captured the upstream's final
	// usage chunk (we request include_usage during translation), so its
	// Usage() now holds the real token counts. Without this the gateway's
	// primary client — Claude Code on /v1/messages with stream:true — would
	// go entirely unmetered and silently bypass the budget wall.
	inTok, outTok := tr.Usage()
	s.recordAnthropicUsage(meta, model, translator.OpenAIUsage{
		PromptTokens:     inTok,
		CompletionTokens: outTok,
		TotalTokens:      inTok + outTok,
	}, http.StatusOK, true)
	if guard != nil {
		guard.RecordUsage(int64(inTok), int64(outTok))
	}
}

// recordAnthropicUsage mirrors recordUsage in proxy.go but takes the
// pre-translated OpenAI usage struct (since the upstream is OpenAI-
// shaped even for the Anthropic-input path).
func (s *Server) recordAnthropicUsage(meta *RequestMeta, model string, u translator.OpenAIUsage, statusCode int, streaming bool) {
	if s.meter == nil || meta == nil {
		return
	}
	rec := metering.Record{
		AppID:      meta.AppID,
		Model:      model,
		TokensIn:   int64(u.PromptTokens),
		TokensOut:  int64(u.CompletionTokens),
		LatencyMs:  time.Since(meta.StartTime).Milliseconds(),
		StatusCode: statusCode,
		Timestamp:  time.Now(),
		// Routing attribution — same dimensions the OpenAI-protocol path
		// records, so dashboards bucket Claude Code traffic by upstream too.
		ServedBy:  meta.ServedBy,
		MatchedBy: meta.MatchedBy,
	}
	s.meter.Record(rec)

	// Mirror into the optional OTel recorder. Operation "messages" marks the
	// Anthropic-input path so dashboards can split it from the OpenAI path.
	s.observe(obs.RequestObservation{
		Operation:  "messages",
		Model:      model,
		ServedBy:   meta.ServedBy,
		MatchedBy:  meta.MatchedBy,
		TokensIn:   int64(u.PromptTokens),
		TokensOut:  int64(u.CompletionTokens),
		StartTime:  meta.StartTime,
		LatencyMs:  rec.LatencyMs,
		StatusCode: statusCode,
		Streaming:  streaming,
	})
}

// writeAnthropicError emits an error envelope in the shape Claude Code
// parses with its `[kind=…]` suffix expectation. We follow Anthropic's
// shape: `{type:"error", error:{type:"invalid_request_error", message:"…"}}`.
func writeAnthropicError(w http.ResponseWriter, status int, errType, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	body := translator.AnthropicError{
		Type: "error",
		Error: translator.AnthropicErrorBody{
			Type:    errType,
			Message: msg,
		},
	}
	out, _ := json.Marshal(body)
	w.Write(out)
}

// anthropicErrTypeForStatus picks the right Anthropic error type for a
// proxied upstream HTTP status. The taxonomy is documented at
// https://docs.anthropic.com/en/api/errors.
func anthropicErrTypeForStatus(status int) string {
	switch {
	case status == 401, status == 403:
		return "authentication_error"
	case status == 404:
		return "not_found_error"
	case status == 429:
		return "rate_limit_error"
	case status >= 500:
		return "api_error"
	default:
		return "invalid_request_error"
	}
}
