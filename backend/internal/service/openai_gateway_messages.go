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
	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/Wei-Shaw/sub2api/internal/util/responseheaders"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ForwardAsAnthropic accepts an Anthropic Messages request body, converts it
// to OpenAI Responses API format, forwards to the OpenAI upstream, and converts
// the response back to Anthropic Messages format. This enables Claude Code
// clients to access OpenAI models through the standard /v1/messages endpoint.
func (s *OpenAIGatewayService) ForwardAsAnthropic(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	body []byte,
	promptCacheKey string,
	defaultMappedModel string,
) (*OpenAIForwardResult, error) {
	return s.ForwardAsAnthropicContext(ctx, gatewayctx.FromGin(c), account, body, promptCacheKey, defaultMappedModel)
}

func (s *OpenAIGatewayService) ForwardAsAnthropicContext(
	ctx context.Context,
	c gatewayctx.GatewayContext,
	account *Account,
	body []byte,
	promptCacheKey string,
	defaultMappedModel string,
) (*OpenAIForwardResult, error) {
	startTime := time.Now()

	// 1. Parse Anthropic request
	var anthropicReq apicompat.AnthropicRequest
	if err := json.Unmarshal(body, &anthropicReq); err != nil {
		return nil, fmt.Errorf("parse anthropic request: %w", err)
	}
	anthropicDigestReq := cloneAnthropicRequestForDigest(&anthropicReq)
	originalModel := anthropicReq.Model
	applyOpenAICompatModelNormalization(&anthropicReq)
	normalizedModel := anthropicReq.Model
	clientStream := anthropicReq.Stream // client's original stream preference

	billingModel := resolveOpenAIForwardModel(account, normalizedModel, defaultMappedModel)
	upstreamModel := normalizeOpenAIModelForUpstream(account, billingModel)
	promptCacheKey = strings.TrimSpace(promptCacheKey)
	apiKeyID := getAPIKeyIDFromGatewayContext(c)
	anthropicDigestChain := ""
	anthropicMatchedDigestChain := ""
	compatPromptCacheInjected := false
	if promptCacheKey == "" && account.Platform == PlatformGrok {
		if sessionSeed := extractClaudeCodeSessionIDContext(c, body); sessionSeed != "" {
			promptCacheKey = sessionSeed
			compatPromptCacheInjected = true
		} else if sessionSeed := promptCacheKeyFromAnthropicMetadataSession(&anthropicReq); sessionSeed != "" {
			promptCacheKey = sessionSeed
			compatPromptCacheInjected = true
		}
	}
	if promptCacheKey == "" && shouldAutoInjectPromptCacheKeyForCompat(upstreamModel) {
		promptCacheKey = promptCacheKeyFromAnthropicMetadataSession(&anthropicReq)
		if promptCacheKey == "" {
			promptCacheKey = deriveAnthropicCacheControlPromptCacheKey(&anthropicReq)
		}
		if promptCacheKey == "" {
			anthropicDigestChain = buildOpenAICompatAnthropicDigestChain(anthropicDigestReq)
			if reusedKey, matchedChain := s.findOpenAICompatAnthropicDigestPromptCacheKey(account, apiKeyID, anthropicDigestChain); reusedKey != "" {
				promptCacheKey = reusedKey
				anthropicMatchedDigestChain = matchedChain
			} else {
				promptCacheKey = promptCacheKeyFromAnthropicDigest(anthropicDigestChain)
			}
		}
		compatPromptCacheInjected = promptCacheKey != ""
	}
	compatReplayTrimmed := false
	compatReplayGuardEnabled := shouldAutoInjectPromptCacheKeyForCompat(upstreamModel)
	compatContinuationEnabled := openAICompatContinuationEnabled(account, upstreamModel)
	previousResponseID := ""
	if compatContinuationEnabled {
		previousResponseID = s.getOpenAICompatSessionResponseIDContext(ctx, c, account, promptCacheKey)
	}
	compatContinuationDisabled := compatContinuationEnabled &&
		s.isOpenAICompatSessionContinuationDisabledContext(ctx, c, account, promptCacheKey)
	compatTurnState := ""
	if compatReplayGuardEnabled && account.Type != AccountTypeOAuth && previousResponseID == "" && !compatContinuationDisabled {
		compatReplayTrimmed = applyAnthropicCompatFullReplayGuard(&anthropicReq)
	}

	// 2. Convert Anthropic → Responses
	responsesReq, err := apicompat.AnthropicToResponses(&anthropicReq)
	if err != nil {
		return nil, fmt.Errorf("convert anthropic to responses: %w", err)
	}

	// Upstream always uses streaming (upstream may not support sync mode).
	// The client's original preference determines the response format.
	responsesReq.Stream = true
	isStream := true

	// 2b. Handle BetaFastMode → service_tier: "priority"
	if containsBetaToken(c.HeaderValue("anthropic-beta"), claude.BetaFastMode) {
		responsesReq.ServiceTier = "priority"
	}

	responsesReq.Model = upstreamModel
	if previousResponseID != "" {
		responsesReq.PreviousResponseID = previousResponseID
		trimAnthropicCompatResponsesInputToLatestTurn(responsesReq)
	}
	if compatReplayGuardEnabled && account.Type != AccountTypeOAuth {
		appendOpenAICompatClaudeCodeTodoGuard(responsesReq)
	}
	SetOpsUpstreamModelContext(c, upstreamModel)

	logFields := []zap.Field{
		zap.Int64("account_id", account.ID),
		zap.String("original_model", originalModel),
		zap.String("normalized_model", normalizedModel),
		zap.String("billing_model", billingModel),
		zap.String("upstream_model", upstreamModel),
		zap.Bool("stream", isStream),
	}
	if compatPromptCacheInjected {
		logFields = append(logFields, zap.Bool("compat_prompt_cache_key_injected", true))
	}
	if compatReplayTrimmed {
		logFields = append(logFields, zap.Bool("compat_full_replay_trimmed", true))
	}
	if previousResponseID != "" {
		logFields = append(logFields, zap.Bool("compat_previous_response_id_attached", true))
	}
	logger.L().Debug("openai messages: model mapping applied", logFields...)

	// 4. Marshal Responses request body, then apply OAuth codex transform
	responsesBody, err := json.Marshal(responsesReq)
	if err != nil {
		return nil, fmt.Errorf("marshal responses request: %w", err)
	}

	if account.Type == AccountTypeOAuth && account.Platform != PlatformGrok {
		var reqBody map[string]any
		if err := json.Unmarshal(responsesBody, &reqBody); err != nil {
			return nil, fmt.Errorf("unmarshal for codex transform: %w", err)
		}
		codexResult := applyCodexOAuthTransformWithOptions(reqBody, codexOAuthTransformOptions{
			SkipDefaultInstructions: true,
			PreserveToolCallIDs:     true,
		})
		forcedTemplateText := ""
		if s.cfg != nil {
			forcedTemplateText = s.cfg.Gateway.ForcedCodexInstructionsTemplate
		}
		templateUpstreamModel := upstreamModel
		if codexResult.NormalizedModel != "" {
			templateUpstreamModel = codexResult.NormalizedModel
		}
		existingInstructions, _ := reqBody["instructions"].(string)
		if strings.TrimSpace(existingInstructions) == "" {
			existingInstructions = extractPromptLikeInstructionsFromInput(reqBody)
		}
		if _, err := applyForcedCodexInstructionsTemplate(reqBody, forcedTemplateText, forcedCodexInstructionsTemplateData{
			ExistingInstructions: strings.TrimSpace(existingInstructions),
			OriginalModel:        originalModel,
			NormalizedModel:      normalizedModel,
			BillingModel:         billingModel,
			UpstreamModel:        templateUpstreamModel,
		}); err != nil {
			return nil, err
		}
		ensureCodexOAuthInstructionsField(reqBody)
		if shouldAutoInjectPromptCacheKeyForCompat(upstreamModel) {
			appendOpenAICompatClaudeCodeTodoGuardToRequestBody(reqBody)
		}
		if codexResult.NormalizedModel != "" {
			upstreamModel = codexResult.NormalizedModel
			SetOpsUpstreamModelContext(c, upstreamModel)
		}
		if codexResult.PromptCacheKey != "" {
			promptCacheKey = codexResult.PromptCacheKey
		}
		delete(reqBody, "prompt_cache_key")
		if shouldAutoInjectPromptCacheKeyForCompat(upstreamModel) {
			compatTurnState = s.getOpenAICompatSessionTurnStateContext(ctx, c, account, promptCacheKey)
		}
		// OAuth codex transform forces stream=true upstream, so always use
		// the streaming response handler regardless of what the client asked.
		isStream = true
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

	responsesBody, err = s.applyOpenAIFastPolicyToBody(ctx, account, upstreamModel, responsesBody)
	if err != nil {
		return nil, err
	}

	if account.IsOpenAIChatWebMode() {
		resp, prepared, token, err := s.beginOpenAIChatWebConversationRequest(ctx, c, account, responsesBody)
		if err != nil {
			return nil, err
		}
		defer func() { _ = resp.Body.Close() }()
		defer s.pingOpenAIChatWebSentinel(context.Background(), account, token, prepared)

		if resp.StatusCode >= 400 {
			return s.handleAnthropicErrorResponseContext(resp, c, account)
		}

		var result *OpenAIForwardResult
		var handleErr error
		if clientStream {
			result, handleErr = s.handleAnthropicStreamingResponseContext(resp, c, account, originalModel, billingModel, startTime)
		} else {
			result, handleErr = s.handleAnthropicBufferedStreamingResponseContext(resp, c, originalModel, billingModel, startTime)
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
		upstreamReq, buildErr := s.buildUpstreamRequestContext(upstreamCtx, c, account, responsesBody, token, isStream, promptCacheKey, false)
		releaseUpstreamCtx()
		if buildErr != nil {
			return nil, fmt.Errorf("build upstream request: %w", buildErr)
		}
		if promptCacheKey != "" {
			apiKeyID := getAPIKeyIDFromGatewayContext(c)
			upstreamReq.Header.Set("session_id", generateSessionUUID(isolateOpenAISessionID(apiKeyID, promptCacheKey)))
		}
		if account.Type == AccountTypeOAuth && account.Platform != PlatformGrok {
			ensureCodexIdentityHeaders(upstreamReq.Header)
			enforceCodexIdentityHeaders(upstreamReq.Header)
		}
		if account.Type == AccountTypeOAuth && promptCacheKey != "" &&
			strings.TrimSpace(c.HeaderValue("conversation_id")) == "" {
			upstreamReq.Header.Del("conversation_id")
		}
		if compatTurnState != "" && upstreamReq.Header.Get("x-codex-turn-state") == "" {
			upstreamReq.Header.Set("x-codex-turn-state", compatTurnState)
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
		if previousResponseID != "" && (isOpenAICompatPreviousResponseNotFound(resp.StatusCode, upstreamMsg, respBody) || isOpenAICompatPreviousResponseUnsupported(resp.StatusCode, upstreamMsg, respBody)) {
			if isOpenAICompatPreviousResponseUnsupported(resp.StatusCode, upstreamMsg, respBody) {
				s.disableOpenAICompatSessionContinuationContext(ctx, c, account, promptCacheKey)
			} else {
				s.deleteOpenAICompatSessionResponseIDContext(ctx, c, account, promptCacheKey)
			}
			return s.ForwardAsAnthropicContext(ctx, c, account, body, promptCacheKey, defaultMappedModel)
		}
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
		// Non-failover error: return Anthropic-formatted error to client
		return s.handleAnthropicErrorResponseContext(resp, c, account)
	}
	if account.Type == AccountTypeOAuth && promptCacheKey != "" {
		if turnState := strings.TrimSpace(resp.Header.Get("x-codex-turn-state")); turnState != "" {
			s.bindOpenAICompatSessionTurnStateContext(ctx, c, account, promptCacheKey, turnState)
		}
	}

	// 9. Handle normal response
	// Upstream is always streaming; choose response format based on client preference.
	var result *OpenAIForwardResult
	var handleErr error
	if clientStream {
		result, handleErr = s.handleAnthropicStreamingResponseContext(resp, c, account, originalModel, billingModel, startTime)
	} else {
		// Client wants JSON: buffer the streaming response and assemble a JSON reply.
		result, handleErr = s.handleAnthropicBufferedStreamingResponseContext(resp, c, originalModel, billingModel, startTime)
	}

	// Propagate ServiceTier and ReasoningEffort to result for billing
	if handleErr == nil && result != nil {
		if compatContinuationEnabled && promptCacheKey != "" && result.ResponseID != "" {
			s.bindOpenAICompatSessionResponseIDContext(ctx, c, account, promptCacheKey, result.ResponseID)
		}
		if promptCacheKey != "" && anthropicDigestChain != "" {
			s.bindOpenAICompatAnthropicDigestPromptCacheKey(account, apiKeyID, anthropicDigestChain, promptCacheKey, anthropicMatchedDigestChain)
		}
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

// handleAnthropicErrorResponse reads an upstream error and returns it in
// Anthropic error format.
func (s *OpenAIGatewayService) handleAnthropicErrorResponse(
	resp *http.Response,
	c *gin.Context,
	account *Account,
	requestedModel ...string,
) (*OpenAIForwardResult, error) {
	return s.handleCompatErrorResponse(resp, c, account, writeAnthropicError, requestedModel...)
}

func (s *OpenAIGatewayService) handleAnthropicErrorResponseContext(
	resp *http.Response,
	c gatewayctx.GatewayContext,
	account *Account,
) (*OpenAIForwardResult, error) {
	return s.handleCompatErrorResponseContext(resp, c, account, writeAnthropicErrorContext)
}

// handleAnthropicBufferedStreamingResponse reads all Responses SSE events from
// the upstream streaming response, finds the terminal event (response.completed
// / response.incomplete / response.failed), converts the complete response to
// Anthropic Messages JSON format, and writes it to the client.
// This is used when the client requested stream=false but the upstream is always
// streaming.
func (s *OpenAIGatewayService) handleAnthropicBufferedStreamingResponse(
	resp *http.Response,
	c *gin.Context,
	originalModel string,
	mappedModel string,
	startTime time.Time,
) (*OpenAIForwardResult, error) {
	return s.handleAnthropicBufferedStreamingResponseContext(resp, gatewayctx.FromGin(c), originalModel, mappedModel, startTime)
}

func (s *OpenAIGatewayService) handleAnthropicBufferedStreamingResponseContext(
	resp *http.Response,
	c gatewayctx.GatewayContext,
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
	terminalSeen := false
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
		if strings.TrimSpace(payload) == "[DONE]" {
			break
		}

		var event apicompat.ResponsesStreamEvent
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			logger.L().Warn("openai messages buffered: failed to parse event",
				zap.Error(err),
				zap.String("request_id", requestID),
			)
			continue
		}
		partial.applyEvent(&event)
		if event.Usage != nil {
			usage = copyOpenAIUsageFromResponsesUsage(event.Usage)
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
			writeAnthropicErrorContext(c, http.StatusBadRequest, "invalid_request_error", message)
			return nil, fmt.Errorf("upstream cyber policy blocked request")
		}

		// Terminal events carry the complete ResponsesResponse with output + usage.
		if openAIStreamEventTypeIsTerminal(strings.TrimSpace(event.Type)) && event.Response != nil {
			finalResponse = event.Response
			if event.Response.Usage != nil {
				usage = copyOpenAIUsageFromResponsesUsage(event.Response.Usage)
			}
			terminalSeen = true
			break
		}
	}
	if !terminalSeen {
		if frame, ok := parser.Finish(); ok {
			payload := openAICompatPayloadWithEventType(frame.Data, frame.EventType)
			var event apicompat.ResponsesStreamEvent
			if json.Unmarshal([]byte(payload), &event) == nil {
				partial.applyEvent(&event)
				if event.Usage != nil {
					usage = copyOpenAIUsageFromResponsesUsage(event.Usage)
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
					writeAnthropicErrorContext(c, http.StatusBadRequest, "invalid_request_error", message)
					return nil, fmt.Errorf("upstream cyber policy blocked request")
				}
				if openAIStreamEventTypeIsTerminal(strings.TrimSpace(event.Type)) && event.Response != nil {
					finalResponse = event.Response
					if event.Response.Usage != nil {
						usage = copyOpenAIUsageFromResponsesUsage(event.Response.Usage)
					}
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			logger.L().Warn("openai messages buffered: read error",
				zap.Error(err),
				zap.String("request_id", requestID),
			)
		}
	}

	if finalResponse == nil {
		if partial.hasUsefulOutput() {
			logger.L().Info("openai messages buffered: upstream ended without terminal event, returning partial response",
				zap.String("request_id", requestID),
			)
			finalResponse = partial.responseSnapshot()
		} else {
			writeAnthropicErrorContext(c, http.StatusBadGateway, "api_error", "Upstream stream ended without a terminal response event")
			return nil, fmt.Errorf("upstream stream ended without terminal event")
		}
	}

	anthropicResp := apicompat.ResponsesToAnthropic(finalResponse, originalModel)

	if s.responseHeaderFilter != nil {
		responseheaders.WriteFilteredHeaders(c.Header(), resp.Header, s.responseHeaderFilter)
	}
	c.SetHeader("Content-Type", "application/json; charset=utf-8")
	c.Header().Del("Content-Length")
	c.Header().Del("Transfer-Encoding")
	c.WriteJSON(http.StatusOK, anthropicResp)

	return &OpenAIForwardResult{
		RequestID:     requestID,
		ResponseID:    finalResponse.ID,
		Usage:         usage,
		Model:         originalModel,
		BillingModel:  mappedModel,
		UpstreamModel: mappedModel,
		Stream:        false,
		Duration:      time.Since(startTime),
	}, nil
}

// handleAnthropicStreamingResponse reads Responses SSE events from upstream,
// converts each to Anthropic SSE events, and writes them to the client.
// When StreamKeepaliveInterval is configured, it uses a goroutine + channel
// pattern to send Anthropic ping events during periods of upstream silence,
// preventing proxy/client timeout disconnections.
func (s *OpenAIGatewayService) handleAnthropicStreamingResponse(
	resp *http.Response,
	c *gin.Context,
	originalModel string,
	mappedModel string,
	startTime time.Time,
) (*OpenAIForwardResult, error) {
	return s.handleAnthropicStreamingResponseContext(resp, gatewayctx.FromGin(c), nil, originalModel, mappedModel, startTime)
}

func (s *OpenAIGatewayService) handleAnthropicStreamingResponseContext(
	resp *http.Response,
	c gatewayctx.GatewayContext,
	account *Account,
	originalModel string,
	mappedModel string,
	startTime time.Time,
) (*OpenAIForwardResult, error) {
	requestID := resp.Header.Get("x-request-id")

	if s.responseHeaderFilter != nil {
		responseheaders.WriteFilteredHeaders(c.Header(), resp.Header, s.responseHeaderFilter)
	}
	gatewayctx.PrepareSSE(c, gatewayctx.SSEOptions{
		ContentType:  "text/event-stream",
		CacheControl: "no-cache",
		RequestID:    requestID,
	})

	state := apicompat.NewResponsesEventToAnthropicState()
	state.Model = originalModel
	var usage OpenAIUsage
	responseID := ""
	var firstTokenMs *int
	firstChunk := true
	clientDisconnected := false
	clientOutputStarted := false
	terminalSeen := false
	var terminalErr error
	flushController := s.newOpenAIHTTPStreamFlushController()
	bufferedWriter := bufio.NewWriterSize(gatewayContextWriter{ctx: c}, 4*1024)
	flushClient := func() bool {
		if clientDisconnected {
			return false
		}
		if err := bufferedWriter.Flush(); err != nil {
			clientDisconnected = true
			return false
		}
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

	// resultWithUsage builds the final result snapshot.
	resultWithUsage := func() *OpenAIForwardResult {
		return &OpenAIForwardResult{
			RequestID:        requestID,
			ResponseID:       responseID,
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

	// processDataLine handles a complete upstream SSE payload. It returns true
	// when the client disconnects or a terminal response event is received.
	processDataLine := func(payload string) bool {
		isFirstTokenEvent := firstChunk
		if firstChunk {
			firstChunk = false
			ms := int(time.Since(startTime).Milliseconds())
			firstTokenMs = &ms
		}

		var event apicompat.ResponsesStreamEvent
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			logger.L().Warn("openai messages stream: failed to parse event",
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
		if event.Response != nil && strings.TrimSpace(event.Response.ID) != "" {
			responseID = strings.TrimSpace(event.Response.ID)
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
				frame := buildAnthropicStreamErrorSSE("invalid_request_error", message)
				if _, err := bufferedWriter.WriteString(frame); err != nil || !flushClient() {
					clientDisconnected = true
				}
			}
			terminalErr = fmt.Errorf("upstream cyber policy blocked request")
			return true
		}

		// Convert to Anthropic events
		events := apicompat.ResponsesEventToAnthropicEvents(&event, state)
		for _, evt := range events {
			if clientDisconnected {
				break
			}
			sse, err := apicompat.ResponsesAnthropicEventToSSE(evt)
			if err != nil {
				logger.L().Warn("openai messages stream: failed to marshal event",
					zap.Error(err),
					zap.String("request_id", requestID),
				)
				continue
			}
			if _, err := bufferedWriter.WriteString(sse); err != nil {
				logger.L().Info("openai messages stream: client disconnected",
					zap.String("request_id", requestID),
				)
				clientDisconnected = true
				break
			}
			clientOutputStarted = true
		}
		if len(events) > 0 && !clientDisconnected && flushController.shouldFlush(isFirstTokenEvent || isTerminal) {
			_ = flushClient()
		}
		return isTerminal
	}
	processFrame := func(frame openAICompatSSEFrame) bool {
		payload := openAICompatPayloadWithEventType(frame.Data, frame.EventType)
		return processDataLine(payload)
	}

	// finalizeStream sends any remaining Anthropic events and returns the result.
	finalizeStream := func() (*OpenAIForwardResult, error) {
		if terminalErr != nil {
			return resultWithUsage(), terminalErr
		}
		if finalEvents := apicompat.FinalizeResponsesAnthropicStream(state); len(finalEvents) > 0 && !clientDisconnected {
			for _, evt := range finalEvents {
				sse, err := apicompat.ResponsesAnthropicEventToSSE(evt)
				if err != nil {
					continue
				}
				if _, err := bufferedWriter.WriteString(sse); err != nil {
					clientDisconnected = true
					break
				}
			}
			_ = flushClient()
		}
		return resultWithUsage(), nil
	}
	missingTerminalErr := func() (*OpenAIForwardResult, error) {
		result := resultWithUsage()
		if clientDisconnected {
			return result, fmt.Errorf("stream usage incomplete: missing terminal event")
		}
		message := "OpenAI messages stream ended before a terminal event"
		if !clientOutputStarted {
			return result, s.newOpenAIStreamFailoverErrorContext(c, account, requestID, nil, message)
		}
		_ = flushClient()
		setOpsUpstreamErrorContext(c, http.StatusBadGateway, message, "")
		event := OpsUpstreamErrorEvent{
			Platform:           PlatformOpenAI,
			UpstreamStatusCode: http.StatusBadGateway,
			UpstreamRequestID:  strings.TrimSpace(requestID),
			Kind:               "stream_missing_terminal",
			Message:            message,
		}
		if account != nil {
			event.Platform = account.Platform
			event.AccountID = account.ID
			event.AccountName = account.Name
		}
		appendOpsUpstreamErrorContext(c, event)
		return result, fmt.Errorf("stream usage incomplete: missing terminal event")
	}

	// handleScanErr logs scanner errors if meaningful.
	handleScanErr := func(err error) {
		if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			logger.L().Warn("openai messages stream: read error",
				zap.Error(err),
				zap.String("request_id", requestID),
			)
		}
	}

	// ── Determine keepalive interval ──
	keepaliveInterval := time.Duration(0)
	if s.cfg != nil && s.cfg.Gateway.StreamKeepaliveInterval > 0 {
		keepaliveInterval = time.Duration(s.cfg.Gateway.StreamKeepaliveInterval) * time.Second
	}

	// ── No keepalive: fast synchronous path (no goroutine overhead) ──
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
			if frame, ok := parser.AddLine(line); ok && processFrame(frame) {
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
			if processFrame(frame) {
				return finalizeStream()
			}
		}
		handleScanErr(scanner.Err())
		if terminalSeen {
			return finalizeStream()
		}
		return missingTerminalErr()
	}

	// ── With keepalive: goroutine + channel + select ──
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
				// Upstream closed
				if frame, frameOK := parser.Finish(); frameOK {
					if strings.TrimSpace(frame.Data) == "[DONE]" {
						if terminalSeen {
							return finalizeStream()
						}
						return missingTerminalErr()
					}
					if processFrame(frame) {
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
				return missingTerminalErr()
			}
			lastDataAt = time.Now()
			if isOpenAICompatDoneSentinelLine(ev.line) {
				if terminalSeen {
					return finalizeStream()
				}
				return missingTerminalErr()
			}
			if frame, frameOK := parser.AddLine(ev.line); frameOK && processFrame(frame) {
				return finalizeStream()
			}

		case <-keepaliveTicker.C:
			if time.Since(lastDataAt) < keepaliveInterval {
				continue
			}
			// Send Anthropic-format ping event
			if clientDisconnected {
				continue
			}
			if _, err := bufferedWriter.WriteString("event: ping\ndata: {\"type\":\"ping\"}\n\n"); err != nil {
				// Client disconnected
				logger.L().Info("openai messages stream: client disconnected during keepalive",
					zap.String("request_id", requestID),
				)
				clientDisconnected = true
				continue
			}
			_ = flushClient()
		}
	}
}

// writeAnthropicError writes an error response in Anthropic Messages API format.
func writeAnthropicError(c *gin.Context, statusCode int, errType, message string) {
	writeAnthropicErrorContext(gatewayctx.FromGin(c), statusCode, errType, message)
}

func writeAnthropicErrorContext(c gatewayctx.GatewayContext, statusCode int, errType, message string) {
	if c == nil {
		return
	}
	c.WriteJSON(statusCode, gin.H{
		"type": "error",
		"error": gin.H{
			"type":    errType,
			"message": message,
		},
	})
}

func isOpenAICompatDoneSentinelLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	return trimmed == "[DONE]" || trimmed == "data: [DONE]"
}

func markOpsCyberPolicyContext(c gatewayctx.GatewayContext, mark CyberPolicyMark) {
	if c == nil {
		return
	}
	if ginContext, ok := c.Native().(*gin.Context); ok {
		MarkOpsCyberPolicy(ginContext, mark)
	}
}

func hasOpsCyberPolicyContext(c gatewayctx.GatewayContext) bool {
	if c == nil {
		return false
	}
	if ginContext, ok := c.Native().(*gin.Context); ok {
		return GetOpsCyberPolicy(ginContext) != nil
	}
	return false
}

func buildAnthropicStreamErrorSSE(errType, message string) string {
	payload, err := json.Marshal(gin.H{
		"type": "error",
		"error": gin.H{
			"type":    errType,
			"message": message,
		},
	})
	if err != nil {
		return "event: error\ndata: {\"type\":\"error\",\"error\":{\"type\":\"api_error\",\"message\":\"upstream error\"}}\n\n"
	}
	return "event: error\ndata: " + string(payload) + "\n\n"
}
