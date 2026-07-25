package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/Wei-Shaw/sub2api/internal/util/responseheaders"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"go.uber.org/zap"
)

// ForwardAsChatCompletions accepts a Chat Completions request body, converts it
// to OpenAI Responses API format, forwards to the OpenAI upstream, and converts
// the response back to Chat Completions format. All account types (OAuth and API
// Key) go through the Responses API conversion path since the upstream only
// exposes the /v1/responses endpoint.
func (s *OpenAIGatewayService) ForwardAsChatCompletions(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	body []byte,
	promptCacheKey string,
	defaultMappedModel string,
) (*OpenAIForwardResult, error) {
	return s.ForwardAsChatCompletionsContext(ctx, gatewayctx.FromGin(c), account, body, promptCacheKey, defaultMappedModel)
}

func (s *OpenAIGatewayService) ForwardAsChatCompletionsContext(
	ctx context.Context,
	c gatewayctx.GatewayContext,
	account *Account,
	body []byte,
	promptCacheKey string,
	defaultMappedModel string,
) (*OpenAIForwardResult, error) {
	startTime := time.Now()

	// 1. Parse Chat Completions request
	var chatReq apicompat.ChatCompletionsRequest
	if err := json.Unmarshal(body, &chatReq); err != nil {
		return nil, fmt.Errorf("parse chat completions request: %w", err)
	}
	originalModel := chatReq.Model
	clientStream := chatReq.Stream
	includeUsage := chatReq.StreamOptions != nil && chatReq.StreamOptions.IncludeUsage

	if account.IsOpenAIChatWebMode() && !clientStream {
		if handled, result, err := s.tryForwardOpenAIChatWebImageChatCompletionsContext(ctx, c, account, body, originalModel, startTime); handled {
			return result, err
		}
	}

	// 2. Resolve model mapping early so compat prompt_cache_key injection can
	// derive a stable seed from the final upstream model family.
	mappedModel, modelMappingMatched := resolveOpenAIForwardModelWithMatch(account, originalModel, defaultMappedModel)
	if shouldNormalizeChatCompatModel(mappedModel, modelMappingMatched, defaultMappedModel) {
		mappedModel = normalizeOpenAIModelForUpstream(account, mappedModel)
	} else {
		mappedModel = strings.TrimSpace(mappedModel)
	}
	SetOpsUpstreamModelContext(c, mappedModel)

	if shouldUseOpenAICompatibleChatCompletionsPassthroughContext(c, account) {
		passthroughBody := body
		passthroughModel := resolveOpenAICompatibleChatCompletionsPassthroughModel(account, originalModel)
		if passthroughModel != "" && passthroughModel != originalModel {
			patchedBody, patchErr := sjson.SetBytes(body, "model", passthroughModel)
			if patchErr != nil {
				return nil, fmt.Errorf("set chat completions passthrough model: %w", patchErr)
			}
			passthroughBody = patchedBody
		}
		SetOpsUpstreamModelContext(c, passthroughModel)
		reasoningEffort := extractOpenAIReasoningEffortFromBody(passthroughBody, originalModel)
		return s.forwardOpenAIPassthroughContext(ctx, c, account, passthroughBody, originalModel, reasoningEffort, clientStream, startTime)
	}

	promptCacheKey = strings.TrimSpace(promptCacheKey)
	compatPromptCacheInjected := false
	if promptCacheKey == "" && shouldApplyOpenAICodexOAuthTransform(account) && shouldAutoInjectPromptCacheKeyForCompat(mappedModel) {
		promptCacheKey = deriveCompatPromptCacheKey(&chatReq, mappedModel)
		compatPromptCacheInjected = promptCacheKey != ""
	}

	// 3. Convert to Responses and forward
	// ChatCompletionsToResponses always sets Stream=true (upstream always streams).
	responsesReq, err := apicompat.ChatCompletionsToResponses(&chatReq)
	if err != nil {
		return nil, fmt.Errorf("convert chat completions to responses: %w", err)
	}
	responsesReq.Model = mappedModel
	normalizeResponsesRequestServiceTier(responsesReq)

	logFields := []zap.Field{
		zap.Int64("account_id", account.ID),
		zap.String("original_model", originalModel),
		zap.String("mapped_model", mappedModel),
		zap.Bool("stream", clientStream),
	}
	if compatPromptCacheInjected {
		logFields = append(logFields,
			zap.Bool("compat_prompt_cache_key_injected", true),
			zap.String("compat_prompt_cache_key_sha256", hashSensitiveValueForLog(promptCacheKey)),
		)
	}
	logger.L().Debug("openai chat_completions: model mapping applied", logFields...)

	// 4. Marshal Responses request body, then apply OAuth codex transform
	responsesBody, err := json.Marshal(responsesReq)
	if err != nil {
		return nil, fmt.Errorf("marshal responses request: %w", err)
	}

	if shouldApplyOpenAICodexOAuthTransform(account) {
		var reqBody map[string]any
		if err := json.Unmarshal(responsesBody, &reqBody); err != nil {
			return nil, fmt.Errorf("unmarshal for codex transform: %w", err)
		}
		isJSONObjectFormat := strings.EqualFold(strings.TrimSpace(gjson.GetBytes(responsesBody, "text.format.type").String()), "json_object")
		codexResult := applyCodexOAuthTransformWithOptions(reqBody, codexOAuthTransformOptions{
			SkipDefaultInstructions:             true,
			OmitPromotedSystemMessagesFromInput: !isJSONObjectFormat,
		})
		ensureCodexOAuthInstructionsField(reqBody)
		if codexResult.PromptCacheKey != "" {
			promptCacheKey = codexResult.PromptCacheKey
		} else if promptCacheKey != "" {
			reqBody["prompt_cache_key"] = promptCacheKey
		}
		responsesBody, err = json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("remarshal after codex transform: %w", err)
		}
	}
	if account.Type == AccountTypeAPIKey {
		if trimmedKey := strings.TrimSpace(promptCacheKey); trimmedKey != "" {
			var reqBody map[string]any
			if err := json.Unmarshal(responsesBody, &reqBody); err != nil {
				return nil, fmt.Errorf("unmarshal for prompt cache key injection: %w", err)
			}
			if existing, ok := reqBody["prompt_cache_key"].(string); !ok || strings.TrimSpace(existing) == "" {
				reqBody["prompt_cache_key"] = trimmedKey
				responsesBody, err = json.Marshal(reqBody)
				if err != nil {
					return nil, fmt.Errorf("remarshal after prompt cache key injection: %w", err)
				}
			}
		}
	}

	if account.IsOpenAIChatWebMode() {
		resp, prepared, token, err := s.beginOpenAIChatWebConversationRequest(ctx, c, account, responsesBody)
		if err != nil {
			return nil, err
		}
		defer func() { _ = resp.Body.Close() }()
		defer s.pingOpenAIChatWebSentinel(context.Background(), account, token, prepared)

		if resp.StatusCode >= 400 {
			return s.handleChatCompletionsErrorResponseContext(resp, c, account)
		}

		var result *OpenAIForwardResult
		var handleErr error
		if clientStream {
			result, handleErr = s.handleChatStreamingResponseContext(resp, c, account, originalModel, mappedModel, includeUsage, startTime)
		} else {
			result, handleErr = s.handleChatBufferedStreamingResponseContext(resp, c, account, originalModel, mappedModel, startTime)
		}
		if handleErr == nil && result != nil {
			if responsesReq.ServiceTier != "" {
				st := responsesReq.ServiceTier
				result.ServiceTier = &st
			}
			if responsesReq.Reasoning != nil && responsesReq.Reasoning.Effort != "" {
				re := responsesReq.Reasoning.Effort
				result.ReasoningEffort = &re
			}
		}
		if handleErr != nil && hasOpsCyberPolicyContext(c) {
			return nil, handleErr
		}
		return result, handleErr
	}

	// 5. Get access token. Agent Identity signs each built request instead.
	token, err := s.getOpenAIRequestAccessToken(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("get access token: %w", err)
	}

	proxyURL := ""
	if account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}

	// 6-7. Build and send, recovering a stale Agent Identity task at most once.
	agentTaskRecoveryAttempted := false
	var resp *http.Response
	for {
		upstreamCtx, releaseUpstreamCtx := detachUpstreamContext(ctx)
		upstreamReq, buildErr := s.buildUpstreamRequestContext(upstreamCtx, c, account, responsesBody, token, true, promptCacheKey, false)
		releaseUpstreamCtx()
		if buildErr != nil {
			return nil, fmt.Errorf("build upstream request: %w", buildErr)
		}
		if promptCacheKey != "" {
			apiKeyID := getAPIKeyIDFromGatewayContext(c)
			upstreamReq.Header.Set("session_id", generateSessionUUID(isolateOpenAISessionID(apiKeyID, promptCacheKey)))
		}

		var cancelQuickFail context.CancelFunc
		if clientStream {
			upstreamReq = s.applyOpenAITransportOverride(upstreamReq, responsesBody, true)
		} else {
			upstreamReq, cancelQuickFail = withProxyQuickFailRequest(upstreamReq, proxyURL)
		}
		resp, err = s.httpUpstream.Do(upstreamReq, proxyURL, account.ID, account.Concurrency)
		if err != nil {
			if cancelQuickFail != nil {
				cancelQuickFail()
			}
			break
		}
		if cancelQuickFail != nil {
			resp = attachProxyQuickFailCancel(resp, cancelQuickFail)
		}
		if resp.StatusCode < 400 || agentTaskRecoveryAttempted || !s.isAgentIdentityAccount(ctx, account) {
			break
		}
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		_ = resp.Body.Close()
		resp.Body = io.NopCloser(bytes.NewReader(respBody))
		if !isAgentIdentityTaskInvalidHTTPResponse(resp.StatusCode, respBody) {
			break
		}
		agentTaskRecoveryAttempted = true
		expectedTaskID := account.GetCredential("task_id")
		if recoveryErr := s.recoverAgentIdentityTask(ctx, account, expectedTaskID); recoveryErr != nil {
			return nil, fmt.Errorf("agent identity task recovery failed: %w", recoveryErr)
		}
	}
	if err != nil {
		safeErr := sanitizeUpstreamErrorMessage(err.Error())
		setOpsUpstreamErrorContext(c, 0, safeErr, "")
		appendOpsUpstreamErrorContext(c, OpsUpstreamErrorEvent{
			Platform:           account.Platform,
			AccountID:          account.ID,
			AccountName:        account.Name,
			UpstreamStatusCode: 0,
			Kind:               "request_error",
			Message:            safeErr,
		})
		return nil, newProxyRequestFailoverError(account, proxyURL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	// 8. Handle error response with failover
	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		_ = resp.Body.Close()
		resp.Body = io.NopCloser(bytes.NewReader(respBody))

		upstreamMsg := strings.TrimSpace(extractUpstreamErrorMessage(respBody))
		upstreamMsg = sanitizeUpstreamErrorMessage(upstreamMsg)
		if s.shouldFailoverOpenAIUpstreamResponse(resp.StatusCode, upstreamMsg, respBody) {
			upstreamDetail := ""
			if s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
				maxBytes := s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes
				if maxBytes <= 0 {
					maxBytes = 2048
				}
				upstreamDetail = truncateString(string(respBody), maxBytes)
			}
			appendOpsUpstreamErrorContext(c, OpsUpstreamErrorEvent{
				Platform:           account.Platform,
				AccountID:          account.ID,
				AccountName:        account.Name,
				UpstreamStatusCode: resp.StatusCode,
				UpstreamRequestID:  resp.Header.Get("x-request-id"),
				Kind:               "failover",
				Message:            upstreamMsg,
				Detail:             upstreamDetail,
			})
			if s.rateLimitService != nil {
				s.rateLimitService.HandleUpstreamError(ctx, account, resp.StatusCode, resp.Header, respBody)
			}
			s.queueOpenAIRuntimeStateSync(account.ID)
			return nil, buildOpenAIUpstreamFailoverError(account, resp.StatusCode, upstreamMsg, respBody)
		}
		return s.handleChatCompletionsErrorResponseContext(resp, c, account)
	}

	// 9. Handle normal response
	var result *OpenAIForwardResult
	var handleErr error
	if clientStream {
		result, handleErr = s.handleChatStreamingResponseContext(resp, c, account, originalModel, mappedModel, includeUsage, startTime)
	} else {
		result, handleErr = s.handleChatBufferedStreamingResponseContext(resp, c, account, originalModel, mappedModel, startTime)
	}

	// Propagate ServiceTier and ReasoningEffort to result for billing
	if handleErr == nil && result != nil {
		if responsesReq.ServiceTier != "" {
			st := responsesReq.ServiceTier
			result.ServiceTier = &st
		}
		if responsesReq.Reasoning != nil && responsesReq.Reasoning.Effort != "" {
			re := responsesReq.Reasoning.Effort
			result.ReasoningEffort = &re
		}
	}
	if handleErr != nil && hasOpsCyberPolicyContext(c) {
		return nil, handleErr
	}

	// Extract and save Codex usage snapshot from response headers (for OAuth accounts)
	if handleErr == nil && shouldApplyOpenAICodexOAuthTransform(account) {
		if snapshot := ParseCodexRateLimitHeaders(resp.Header); snapshot != nil {
			s.updateCodexUsageSnapshot(ctx, account.ID, snapshot)
		}
	}

	return result, handleErr
}

func shouldNormalizeChatCompatModel(model string, mappingMatched bool, defaultMappedModel string) bool {
	if mappingMatched || strings.TrimSpace(defaultMappedModel) != "" {
		return true
	}
	trimmed := strings.TrimSpace(model)
	if trimmed == "" || isOpenAIImageGenerationModel(trimmed) {
		return true
	}
	segment := strings.ToLower(lastOpenAIModelSegment(trimmed))
	if getNormalizedCodexModel(segment) != "" {
		return true
	}
	return strings.Contains(segment, "gpt-5") ||
		strings.Contains(segment, "gpt 5") ||
		strings.Contains(segment, "codex")
}

// handleChatCompletionsErrorResponse reads an upstream error and returns it in
// OpenAI Chat Completions error format.
func (s *OpenAIGatewayService) handleChatCompletionsErrorResponse(
	resp *http.Response,
	c *gin.Context,
	account *Account,
	requestedModel ...string,
) (*OpenAIForwardResult, error) {
	return s.handleCompatErrorResponse(resp, c, account, writeChatCompletionsError, requestedModel...)
}

func (s *OpenAIGatewayService) handleChatCompletionsErrorResponseContext(
	resp *http.Response,
	c gatewayctx.GatewayContext,
	account *Account,
) (*OpenAIForwardResult, error) {
	return s.handleCompatErrorResponseContext(resp, c, account, writeChatCompletionsErrorContext)
}

// handleChatBufferedStreamingResponse reads all Responses SSE events from the
// upstream, finds the terminal event, converts to a Chat Completions JSON
// response, and writes it to the client.
func (s *OpenAIGatewayService) handleChatBufferedStreamingResponse(
	resp *http.Response,
	c *gin.Context,
	originalModel string,
	mappedModel string,
	startTime time.Time,
) (*OpenAIForwardResult, error) {
	return s.handleChatBufferedStreamingResponseContext(resp, gatewayctx.FromGin(c), nil, originalModel, mappedModel, startTime)
}

func (s *OpenAIGatewayService) handleChatBufferedStreamingResponseContext(
	resp *http.Response,
	c gatewayctx.GatewayContext,
	account *Account,
	originalModel string,
	mappedModel string,
	startTime time.Time,
) (*OpenAIForwardResult, error) {
	requestID := resp.Header.Get("x-request-id")
	scanner := s.newUpstreamSSEScanner(resp.Body)

	var finalResponse *apicompat.ResponsesResponse
	var usage OpenAIUsage
	partial := newBufferedResponsesAccumulator(originalModel, requestID)
	var parser openAICompatSSEFrameParser
	processPayload := func(payload string) (bool, error) {
		if strings.TrimSpace(payload) == "" || strings.TrimSpace(payload) == "[DONE]" {
			return false, nil
		}
		var event apicompat.ResponsesStreamEvent
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			logger.L().Warn("openai chat_completions buffered: failed to parse event",
				zap.Error(err),
				zap.String("request_id", requestID),
			)
			return false, nil
		}
		partial.applyEvent(&event)
		if event.Usage != nil {
			usage = copyOpenAIUsageFromResponsesUsage(event.Usage)
		}
		if event.Response != nil && event.Response.Usage != nil {
			usage = copyOpenAIUsageFromResponsesUsage(event.Response.Usage)
		}
		if hit, code, message := detectOpenAICyberPolicy([]byte(payload)); hit {
			markOpsCyberPolicyContext(c, CyberPolicyMark{
				Code:           code,
				Message:        message,
				Body:           truncateString(payload, 4096),
				UpstreamStatus: http.StatusOK,
				UpstreamInTok:  usage.InputTokens,
				UpstreamOutTok: usage.OutputTokens,
			})
			if message == "" {
				message = "Request blocked by upstream cyber-security policy"
			}
			writeChatCompletionsErrorContext(c, http.StatusBadGateway, "invalid_request_error", message)
			return true, fmt.Errorf("upstream cyber policy blocked request")
		}
		if strings.TrimSpace(event.Type) == "response.failed" || strings.TrimSpace(event.Type) == "error" {
			message := extractOpenAISSEErrorMessage([]byte(payload))
			if message == "" {
				message = "OpenAI upstream response failed"
			}
			writeChatCompletionsErrorContext(c, http.StatusBadGateway, "invalid_request_error", message)
			return true, fmt.Errorf("upstream response failed: %s", message)
		}
		if openAIStreamEventTypeIsTerminal(strings.TrimSpace(event.Type)) && event.Response != nil {
			finalResponse = event.Response
			if finalResponse.Usage == nil && event.Usage != nil {
				finalResponse.Usage = event.Usage
			}
			return true, nil
		}
		return false, nil
	}
	var terminalErr error
	for scanner.Scan() {
		line := scanner.Text()
		if isOpenAICompatDoneSentinelLine(line) {
			break
		}
		frame, ok := parser.AddLine(line)
		if !ok {
			continue
		}
		payload := openAICompatPayloadWithEventType(frame.Data, frame.EventType)
		done, err := processPayload(payload)
		if err != nil {
			terminalErr = err
		}
		if done {
			break
		}
	}
	if finalResponse == nil && terminalErr == nil {
		if frame, ok := parser.Finish(); ok {
			payload := openAICompatPayloadWithEventType(frame.Data, frame.EventType)
			_, terminalErr = processPayload(payload)
		}
	}

	if err := scanner.Err(); err != nil {
		if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			logger.L().Warn("openai chat_completions buffered: read error",
				zap.Error(err),
				zap.String("request_id", requestID),
			)
		}
	}
	if terminalErr != nil {
		return nil, terminalErr
	}

	if finalResponse == nil {
		if partial.hasUsefulOutput() {
			logger.L().Info("openai chat_completions buffered: upstream ended without terminal event, returning partial response",
				zap.String("request_id", requestID),
			)
			finalResponse = partial.responseSnapshot()
		} else {
			writeChatCompletionsErrorContext(c, http.StatusBadGateway, "api_error", "Upstream stream ended without a terminal response event")
			return nil, fmt.Errorf("upstream stream ended without terminal event")
		}
	}

	chatResp := apicompat.ResponsesToChatCompletions(finalResponse, originalModel)

	if s.responseHeaderFilter != nil {
		responseheaders.WriteFilteredHeaders(c.Header(), resp.Header, s.responseHeaderFilter)
	}
	c.SetHeader("Content-Type", "application/json; charset=utf-8")
	c.Header().Del("Content-Length")
	c.Header().Del("Transfer-Encoding")
	c.WriteJSON(http.StatusOK, chatResp)

	return &OpenAIForwardResult{
		RequestID:     requestID,
		Usage:         usage,
		Model:         originalModel,
		BillingModel:  mappedModel,
		UpstreamModel: mappedModel,
		Stream:        false,
		Duration:      time.Since(startTime),
	}, nil
}

// handleChatStreamingResponse reads Responses SSE events from upstream,
// converts each to Chat Completions SSE chunks, and writes them to the client.
func (s *OpenAIGatewayService) handleChatStreamingResponse(
	resp *http.Response,
	c *gin.Context,
	originalModel string,
	mappedModel string,
	includeUsage bool,
	startTime time.Time,
) (*OpenAIForwardResult, error) {
	return s.handleChatStreamingResponseContext(resp, gatewayctx.FromGin(c), nil, originalModel, mappedModel, includeUsage, startTime)
}

func (s *OpenAIGatewayService) handleChatStreamingResponseContext(
	resp *http.Response,
	c gatewayctx.GatewayContext,
	account *Account,
	originalModel string,
	mappedModel string,
	includeUsage bool,
	startTime time.Time,
) (*OpenAIForwardResult, error) {
	requestID := resp.Header.Get("x-request-id")

	streamHeadersWritten := false
	writeStreamHeaders := func() {
		if streamHeadersWritten {
			return
		}
		if s.responseHeaderFilter != nil {
			responseheaders.WriteFilteredHeaders(c.Header(), resp.Header, s.responseHeaderFilter)
		}
		c.SetHeader("Content-Type", "text/event-stream")
		c.SetHeader("Cache-Control", "no-cache")
		c.SetHeader("Connection", "keep-alive")
		c.SetHeader("X-Accel-Buffering", "no")
		streamHeadersWritten = true
	}

	state := apicompat.NewResponsesEventToChatState()
	state.Model = originalModel
	// Usage is required for downstream relay billing even when the client did
	// not explicitly request stream_options.include_usage.
	state.IncludeUsage = true

	var usage OpenAIUsage
	var firstTokenMs *int
	firstChunk := true
	clientDisconnected := false
	clientOutputStarted := false
	terminalSeen := false
	var terminalErr error
	pendingSSE := make([]string, 0, 2)
	flushController := s.newOpenAIHTTPStreamFlushController()
	flushClient := func() bool {
		if clientDisconnected {
			return false
		}
		writeStreamHeaders()
		if err := c.Flush(); err != nil {
			clientDisconnected = true
			return false
		}
		flushController.markFlushed()
		return true
	}

	scanner := bufio.NewScanner(resp.Body)
	maxLineSize := defaultMaxLineSize
	if s.cfg != nil && s.cfg.Gateway.MaxLineSize > 0 {
		maxLineSize = s.cfg.Gateway.MaxLineSize
	}
	scanner.Buffer(make([]byte, 0, 64*1024), maxLineSize)

	resultWithUsage := func() *OpenAIForwardResult {
		return &OpenAIForwardResult{
			RequestID:        requestID,
			Usage:            usage,
			Model:            originalModel,
			BillingModel:     mappedModel,
			UpstreamModel:    mappedModel,
			Stream:           true,
			Duration:         time.Since(startTime),
			FirstTokenMs:     firstTokenMs,
			ClientDisconnect: clientDisconnected,
		}
	}

	processDataLine := func(payload string) bool {
		isFirstTokenEvent := firstChunk
		if firstChunk {
			firstChunk = false
			ms := int(time.Since(startTime).Milliseconds())
			firstTokenMs = &ms
		}

		var event apicompat.ResponsesStreamEvent
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			logger.L().Warn("openai chat_completions stream: failed to parse event",
				zap.Error(err),
				zap.String("request_id", requestID),
			)
			return false
		}

		if event.Usage != nil {
			usage = copyOpenAIUsageFromResponsesUsage(event.Usage)
		}
		if event.Response != nil && event.Response.Usage != nil {
			usage = copyOpenAIUsageFromResponsesUsage(event.Response.Usage)
		}
		isTerminal := openAIStreamEventTypeIsTerminal(strings.TrimSpace(event.Type)) || strings.TrimSpace(event.Type) == "error"
		if isTerminal {
			terminalSeen = true
		}
		if hit, code, message := detectOpenAICyberPolicy([]byte(payload)); hit {
			markOpsCyberPolicyContext(c, CyberPolicyMark{
				Code:           code,
				Message:        message,
				Body:           truncateString(payload, 4096),
				UpstreamStatus: http.StatusOK,
				UpstreamInTok:  usage.InputTokens,
				UpstreamOutTok: usage.OutputTokens,
			})
			if message == "" {
				message = "Request blocked by upstream cyber-security policy"
			}
			if !clientDisconnected {
				writeStreamHeaders()
				_, _ = c.WriteBytes(0, []byte(buildChatStreamErrorSSE("cyber_policy", message)))
				_, _ = c.WriteBytes(0, []byte("data: [DONE]\n\n"))
				_ = flushClient()
			}
			terminalErr = fmt.Errorf("upstream cyber policy blocked request")
			return true
		}
		if strings.TrimSpace(event.Type) == "response.failed" || strings.TrimSpace(event.Type) == "error" {
			message := extractOpenAISSEErrorMessage([]byte(payload))
			if message == "" {
				message = "OpenAI upstream response failed"
			}
			if !clientDisconnected {
				if !clientOutputStarted {
					pendingSSE = pendingSSE[:0]
					writeChatCompletionsErrorContext(c, http.StatusBadGateway, "invalid_request_error", message)
				} else {
					_, _ = c.WriteBytes(0, []byte(buildChatStreamErrorSSE("upstream_error", message)))
					_, _ = c.WriteBytes(0, []byte("data: [DONE]\n\n"))
					_ = flushClient()
				}
			}
			terminalErr = fmt.Errorf("upstream response failed: %s", message)
			return true
		}

		chunks := apicompat.ResponsesEventToChatChunks(&event, state)
		encodedChunks := make([]string, 0, len(chunks))
		for _, chunk := range chunks {
			sse, err := apicompat.ChatChunkToSSE(chunk)
			if err != nil {
				logger.L().Warn("openai chat_completions stream: failed to marshal chunk",
					zap.Error(err),
					zap.String("request_id", requestID),
				)
				continue
			}
			encodedChunks = append(encodedChunks, sse)
		}
		if strings.TrimSpace(event.Type) == "response.created" {
			pendingSSE = append(pendingSSE, encodedChunks...)
			return isTerminal
		}
		if len(encodedChunks) > 0 && !clientDisconnected {
			encodedChunks = append(pendingSSE, encodedChunks...)
			pendingSSE = pendingSSE[:0]
		}
		for _, sse := range encodedChunks {
			if clientDisconnected {
				break
			}
			writeStreamHeaders()
			if _, err := c.WriteBytes(0, []byte(sse)); err != nil {
				logger.L().Info("openai chat_completions stream: client disconnected",
					zap.String("request_id", requestID),
				)
				clientDisconnected = true
				break
			}
			clientOutputStarted = true
		}
		if len(encodedChunks) > 0 && !clientDisconnected && flushController.shouldFlush(isFirstTokenEvent || isTerminal) {
			_ = flushClient()
		}
		return isTerminal
	}

	finalizeStream := func() (*OpenAIForwardResult, error) {
		if terminalErr != nil {
			return resultWithUsage(), terminalErr
		}
		if finalChunks := apicompat.FinalizeResponsesChatStream(state); len(finalChunks) > 0 && !clientDisconnected {
			for _, chunk := range finalChunks {
				sse, err := apicompat.ChatChunkToSSE(chunk)
				if err != nil {
					continue
				}
				writeStreamHeaders()
				if _, err := c.WriteBytes(0, []byte(sse)); err != nil {
					clientDisconnected = true
					break
				}
			}
		}
		// Send [DONE] sentinel
		if !clientDisconnected {
			writeStreamHeaders()
			_, _ = c.WriteBytes(0, []byte("data: [DONE]\n\n"))
			_ = flushClient()
		}
		return resultWithUsage(), nil
	}
	missingTerminalErr := func() (*OpenAIForwardResult, error) {
		result := resultWithUsage()
		message := "OpenAI chat completions stream ended before a terminal event"
		if !clientDisconnected {
			if clientOutputStarted {
				_, _ = c.WriteBytes(0, []byte(buildChatStreamErrorSSE("api_error", message)))
			}
			_, _ = c.WriteBytes(0, []byte("data: [DONE]\n\n"))
			_ = flushClient()
		}
		return result, fmt.Errorf("stream usage incomplete: missing terminal event")
	}

	handleScanErr := func(err error) {
		if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			logger.L().Warn("openai chat_completions stream: read error",
				zap.Error(err),
				zap.String("request_id", requestID),
			)
		}
	}

	// Determine keepalive interval
	keepaliveInterval := time.Duration(0)
	if s.cfg != nil && s.cfg.Gateway.StreamKeepaliveInterval > 0 {
		keepaliveInterval = time.Duration(s.cfg.Gateway.StreamKeepaliveInterval) * time.Second
	}

	// No keepalive: fast synchronous path
	if keepaliveInterval <= 0 {
		var parser openAICompatSSEFrameParser
		for scanner.Scan() {
			line := scanner.Text()
			if isOpenAICompatDoneSentinelLine(line) {
				if terminalSeen {
					return finalizeStream()
				}
				return missingTerminalErr()
			}
			frame, ok := parser.AddLine(line)
			if !ok {
				continue
			}
			if processDataLine(openAICompatPayloadWithEventType(frame.Data, frame.EventType)) {
				return finalizeStream()
			}
		}
		if frame, ok := parser.Finish(); ok {
			if strings.TrimSpace(frame.Data) == "[DONE]" {
				if terminalSeen {
					return finalizeStream()
				}
				return missingTerminalErr()
			}
			if processDataLine(openAICompatPayloadWithEventType(frame.Data, frame.EventType)) {
				return finalizeStream()
			}
		}
		if scanErr := scanner.Err(); scanErr != nil {
			handleScanErr(scanErr)
			if !terminalSeen {
				return resultWithUsage(), newOpenAIUpstreamStreamReadError(scanErr)
			}
		}
		if terminalSeen {
			return finalizeStream()
		}
		return missingTerminalErr()
	}

	// With keepalive: goroutine + channel + select
	type scanEvent struct {
		line string
		err  error
	}
	events := make(chan scanEvent, 16)
	done := make(chan struct{})
	sendEvent := func(ev scanEvent) bool {
		select {
		case events <- ev:
			return true
		case <-done:
			return false
		}
	}
	go func() {
		defer close(events)
		for scanner.Scan() {
			if !sendEvent(scanEvent{line: scanner.Text()}) {
				return
			}
		}
		if err := scanner.Err(); err != nil {
			_ = sendEvent(scanEvent{err: err})
		}
	}()
	defer close(done)

	keepaliveTicker := time.NewTicker(keepaliveInterval)
	defer keepaliveTicker.Stop()
	lastDataAt := time.Now()
	var parser openAICompatSSEFrameParser

	for {
		select {
		case ev, ok := <-events:
			if !ok {
				if frame, frameOK := parser.Finish(); frameOK {
					if strings.TrimSpace(frame.Data) == "[DONE]" {
						if terminalSeen {
							return finalizeStream()
						}
						return missingTerminalErr()
					}
					if processDataLine(openAICompatPayloadWithEventType(frame.Data, frame.EventType)) {
						return finalizeStream()
					}
				}
				if terminalSeen {
					return finalizeStream()
				}
				return missingTerminalErr()
			}
			if ev.err != nil {
				handleScanErr(ev.err)
				if terminalSeen {
					return finalizeStream()
				}
				return resultWithUsage(), newOpenAIUpstreamStreamReadError(ev.err)
			}
			lastDataAt = time.Now()
			line := ev.line
			if isOpenAICompatDoneSentinelLine(line) {
				if terminalSeen {
					return finalizeStream()
				}
				return missingTerminalErr()
			}
			frame, ok := parser.AddLine(line)
			if !ok {
				continue
			}
			if processDataLine(openAICompatPayloadWithEventType(frame.Data, frame.EventType)) {
				return finalizeStream()
			}

		case <-keepaliveTicker.C:
			if time.Since(lastDataAt) < keepaliveInterval {
				continue
			}
			// Send SSE comment as keepalive
			if clientDisconnected {
				continue
			}
			writeStreamHeaders()
			if _, err := c.WriteBytes(0, []byte(":\n\n")); err != nil {
				logger.L().Info("openai chat_completions stream: client disconnected during keepalive",
					zap.String("request_id", requestID),
				)
				clientDisconnected = true
				continue
			}
			_ = flushClient()
		}
	}
}

// writeChatCompletionsError writes an error response in OpenAI Chat Completions format.
func writeChatCompletionsError(c *gin.Context, statusCode int, errType, message string) {
	writeChatCompletionsErrorContext(gatewayctx.FromGin(c), statusCode, errType, message)
}

func writeChatCompletionsErrorContext(c gatewayctx.GatewayContext, statusCode int, errType, message string) {
	if c == nil {
		return
	}
	c.WriteJSON(statusCode, gin.H{
		"error": gin.H{
			"type":    errType,
			"message": message,
		},
	})
}
