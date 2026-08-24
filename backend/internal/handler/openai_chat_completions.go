package handler

import (
	"context"
	"errors"
	"net/http"
	"time"

	pkghttputil "github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

// ChatCompletions handles OpenAI Chat Completions API requests.
// POST /v1/chat/completions
func (h *OpenAIGatewayHandler) ChatCompletions(c *gin.Context) {
	h.ChatCompletionsGateway(gatewayctx.FromGin(c))
}

func (h *OpenAIGatewayHandler) ChatCompletionsGateway(c gatewayctx.GatewayContext) {
	streamStarted := false
	defer h.recoverChatCompletionsPanicContext(c, &streamStarted)

	requestStart := time.Now()

	apiKey, ok := middleware2.GetAPIKeyFromGatewayContext(c)
	if !ok {
		h.errorResponseGateway(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}

	subject, ok := middleware2.GetAuthSubjectFromGatewayContext(c)
	if !ok {
		h.errorResponseGateway(c, http.StatusInternalServerError, "api_error", "User context not found")
		return
	}
	reqLog := requestLoggerContext(
		c,
		"handler.openai_gateway.chat_completions",
		zap.Int64("user_id", subject.UserID),
		zap.Int64("api_key_id", apiKey.ID),
		zap.Any("group_id", apiKey.GroupID),
	)

	if !h.ensureResponsesDependenciesGateway(c, reqLog) {
		return
	}

	body, err := pkghttputil.ReadRequestBodyWithPrealloc(c.Request())
	if err != nil {
		if maxErr, ok := extractMaxBytesError(err); ok {
			h.errorResponseGateway(c, http.StatusRequestEntityTooLarge, "invalid_request_error", buildBodyTooLargeMessage(maxErr.Limit))
			return
		}
		h.errorResponseGateway(c, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
		return
	}
	if len(body) == 0 {
		h.errorResponseGateway(c, http.StatusBadRequest, "invalid_request_error", "Request body is empty")
		return
	}

	if !gjson.ValidBytes(body) {
		h.errorResponseGateway(c, http.StatusBadRequest, "invalid_request_error", "Failed to parse request body")
		return
	}
	body, _, err = service.EnforceUnboundedTokenRequestLimit(body, "max_tokens", "max_completion_tokens", "max_tokens")
	if err != nil {
		h.errorResponseGateway(c, http.StatusBadRequest, "invalid_request_error", "Failed to apply output token limit")
		return
	}

	modelResult := gjson.GetBytes(body, "model")
	if !modelResult.Exists() || modelResult.Type != gjson.String || modelResult.String() == "" {
		h.errorResponseGateway(c, http.StatusBadRequest, "invalid_request_error", "model is required")
		return
	}
	reqModel := modelResult.String()
	ensureCompositeTargetPlatformContext(c, apiKey, reqModel)
	if !openAICompatibleTextTargetAllowedContext(c, apiKey, reqModel) {
		h.errorResponseGateway(c, http.StatusBadRequest, "invalid_request_error", "Model is not supported by this OpenAI-compatible endpoint for composite groups")
		return
	}
	if service.IsGPTImageGenerationModel(reqModel) {
		h.errorResponseGateway(c, http.StatusBadRequest, "invalid_request_error", "This model is not supported on the Chat Completions endpoint")
		return
	}
	if !apiKeyAllowsRequestedModel(apiKey, reqModel) {
		h.errorResponseGateway(c, http.StatusBadRequest, "invalid_request_error", apiKeyModelRestrictionMessage(apiKey, reqModel))
		return
	}
	reqStream, ok := parseOpenAICompatibleStream(body)
	if !ok {
		h.errorResponseGateway(c, http.StatusBadRequest, "invalid_request_error", invalidStreamFieldTypeMessage)
		return
	}

	reqLog = reqLog.With(zap.String("model", reqModel), zap.Bool("stream", reqStream))

	setOpsRequestContextGateway(c, reqModel, reqStream, body)
	setOpsEndpointContextGateway(c, "", int16(service.RequestTypeFromLegacy(reqStream, false)))
	if decision := h.checkSecurityAuditContext(c, reqLog, apiKey, subject, service.ContentModerationProtocolOpenAIChat, reqModel, body); decision != nil && !decision.AllowNextStage {
		h.openAISecurityAuditErrorContext(c, decision)
		return
	}
	if h.rejectIfCyberSessionBlockedContext(c, apiKey, body, reqModel, cyberBlockFormatChat) {
		return
	}
	channelMapping, _ := h.gatewayService.ResolveChannelMappingAndRestrict(c.Context(), apiKey.GroupID, reqModel)

	if h.errorPassthroughService != nil {
		service.BindErrorPassthroughServiceContext(c, h.errorPassthroughService)
	}

	subscription, _ := middleware2.GetSubscriptionFromGatewayContext(c)

	service.SetOpsLatencyMsContext(c, service.OpsAuthLatencyMsKey, time.Since(requestStart).Milliseconds())
	routingStart := time.Now()

	userReleaseFunc, acquired := h.acquireResponsesUserSlotContext(c, subject.UserID, subject.Concurrency, reqStream, &streamStarted, reqLog)
	if !acquired {
		return
	}
	if userReleaseFunc != nil {
		defer userReleaseFunc()
	}

	if !h.checkOpenAITokenBillingEligibilityContext(
		c,
		reqLog,
		apiKey,
		subscription,
		reqModel,
		body,
		streamStarted,
		"openai_chat_completions",
		h.handleStreamingAwareErrorContext,
	) {
		return
	}
	ccPricingCtx, pricingAt := h.gatewayService.WithOpenAIRequestPricingContext(c.Context(), apiKey.GroupID)
	c.SetRequest(c.Request().WithContext(ccPricingCtx))

	sessionHash := h.resolveOpenAIStickySessionHashContext(c, apiKey, subject.UserID, reqModel, reqStream, body)
	promptCacheKey := h.gatewayService.ExtractSessionIDContext(c, body)

	failoverPolicy := h.resolveOpenAIFailoverPolicy(body, reqStream)
	maxAccountSwitches := failoverPolicy.MaxSwitches
	if maxAccountSwitches != h.maxAccountSwitches || !failoverPolicy.AllowSameAccountRetry {
		reqLog.Info("openai_chat_completions.streaming_failover_policy_applied",
			zap.Int("body_bytes", len(body)),
			zap.Bool("stream", reqStream),
			zap.Int("effective_max_switches", maxAccountSwitches),
			zap.Bool("allow_same_account_retry", failoverPolicy.AllowSameAccountRetry),
		)
	}
	switchCount := 0
	firstOutputTimeoutSwitchCount := 0
	failedAccountIDs := make(map[int64]struct{})
	sameAccountRetryCount := make(map[int64]int)
	profitVetoCount := 0
	var lastFailoverErr *service.UpstreamFailoverError
	var oauth429FailoverState service.OpenAIOAuth429FailoverState

	for {
		if !openAIRequestAllowsFailoverReplayContext(c) {
			return
		}
		if failoverClientGoneContext(c) {
			return
		}
		c.SetValue("openai_chat_completions_fallback_model", "")
		reqLog.Debug("openai_chat_completions.account_selecting", zap.Int("excluded_account_count", len(failedAccountIDs)))
		groupSelection, err := selectOpenAIAPIKeyGroup(
			c.Context(),
			apiKey,
			"",
			sessionHash,
			reqModel,
			failedAccountIDs,
			service.OpenAIUpstreamTransportAny,
			h.selectOpenAIAccountWithSchedulerForCapability(apiKey, service.OpenAIEndpointCapabilityChatCompletions, false, true),
		)
		var (
			selection        *service.AccountSelectionResult
			scheduleDecision service.OpenAIAccountScheduleDecision
			selectedAPIKey   = apiKey
		)
		if err == nil && groupSelection != nil {
			selection = groupSelection.Selection
			scheduleDecision = groupSelection.Decision
			selectedAPIKey = groupSelection.APIKey
		}
		if err != nil {
			reqLog.Warn("openai_chat_completions.account_select_failed",
				zap.Error(err),
				zap.Int("excluded_account_count", len(failedAccountIDs)),
			)
			if len(failedAccountIDs) == 0 {
				defaultModel := resolveOpenAISelectionFallbackModel(apiKey, reqModel)
				if defaultModel != "" && defaultModel != reqModel {
					reqLog.Info("openai_chat_completions.fallback_to_default_model",
						zap.String("default_mapped_model", defaultModel),
					)
					groupSelection, err = selectOpenAIAPIKeyGroup(
						c.Context(),
						apiKey,
						"",
						sessionHash,
						defaultModel,
						failedAccountIDs,
						service.OpenAIUpstreamTransportAny,
						h.selectOpenAIAccountWithSchedulerForCapability(apiKey, service.OpenAIEndpointCapabilityChatCompletions, false, true),
					)
					if err == nil && groupSelection != nil {
						selection = groupSelection.Selection
						scheduleDecision = groupSelection.Decision
						selectedAPIKey = groupSelection.APIKey
						c.SetValue("openai_chat_completions_fallback_model", defaultModel)
					}
				}
				if err != nil {
					classification := classifyOpenAICompatibleNoAccountErrorGateway(c, h.gatewayService, apiKey, reqModel, reqModel)
					h.handleStreamingAwareErrorContext(c, classification.Status, classification.ErrType, classification.Message, streamStarted)
					return
				}
			} else {
				var action FailoverAction
				failedAccountIDs, action = handleOpenAISelectionExhausted(
					c.Context(),
					failedAccountIDs,
					lastFailoverErr,
					switchCount,
					maxAccountSwitches,
				)
				switch action {
				case FailoverContinue:
					continue
				case FailoverCanceled:
					return
				default:
					if lastFailoverErr != nil {
						h.handleFailoverExhaustedContext(c, lastFailoverErr, streamStarted)
					} else {
						h.handleStreamingAwareErrorContext(c, http.StatusBadGateway, "api_error", "Upstream request failed", streamStarted)
					}
					return
				}
			}
		}
		if selection == nil || selection.Account == nil {
			classification := classifyOpenAICompatibleNoAccountErrorGateway(c, h.gatewayService, apiKey, reqModel, reqModel)
			h.handleStreamingAwareErrorContext(c, classification.Status, classification.ErrType, classification.Message, streamStarted)
			return
		}
		account := selection.Account
		sessionHash = ensureOpenAIPoolModeSessionHash(sessionHash, account)
		reqLog.Debug("openai_chat_completions.account_selected", zap.Int64("account_id", account.ID), zap.String("account_name", account.Name))
		_ = scheduleDecision
		setOpsSelectedAccountGateway(c, account.ID, account.Platform)

		accountReleaseFunc, slotResult := h.acquireResponsesAccountSlotWithProfitGateContext(c, selectedAPIKey.GroupID, sessionHash, selection, reqStream, &streamStarted, reqLog)
		if slotResult == openAISlotAcquireProfitVetoed {
			if !recordOpenAIProfitVeto(failedAccountIDs, account.ID, &profitVetoCount) {
				reqLog.Warn("openai_chat_completions.profit_veto_attempts_exhausted", zap.Int("profit_veto_count", profitVetoCount))
				h.handleStreamingAwareErrorContext(c, http.StatusServiceUnavailable, "api_error", profitVetoExhaustedMessage, streamStarted)
				return
			}
			continue
		}
		if slotResult != openAISlotAcquireOK {
			return
		}

		service.SetOpsLatencyMsContext(c, service.OpsRoutingLatencyMsKey, time.Since(routingStart).Milliseconds())
		forwardStart := time.Now()
		writerSizeBeforeForward := service.OpenAICompactKeepaliveAdjustedWrittenSizeContext(c)

		defaultMappedModel := resolveOpenAIForwardDefaultMappedModel(selectedAPIKey, getContextStringGateway(c, "openai_chat_completions_fallback_model"))
		forwardBody := openAIModelMappedBody(body, channelMapping.Mapped, channelMapping.MappedModel, h.gatewayService.ReplaceModelInBody)
		var result *service.OpenAIForwardResult
		if credentialErr := h.preflightGrokCredentialContext(c.Context(), c, account); credentialErr != nil {
			err = credentialErr
		} else {
			result, err = h.gatewayService.ForwardAsChatCompletionsContext(c.Context(), c, account, forwardBody, promptCacheKey, defaultMappedModel)
		}
		usageChannelMapping := usageChannelMappingForAPIKey(c.Context(), h.gatewayService, selectedAPIKey, reqModel)
		h.recordCyberPolicyIfMarkedContext(c, selectedAPIKey, account, subscription, reqModel, err != nil, body, clientRequestedUsageFieldsContext(c, usageChannelMapping, reqModel, ""))

		forwardDurationMs := time.Since(forwardStart).Milliseconds()
		if accountReleaseFunc != nil {
			accountReleaseFunc()
		}
		upstreamLatencyMs, _ := getContextInt64Gateway(c, service.OpsUpstreamLatencyMsKey)
		responseLatencyMs := forwardDurationMs
		if upstreamLatencyMs > 0 && forwardDurationMs > upstreamLatencyMs {
			responseLatencyMs = forwardDurationMs - upstreamLatencyMs
		}
		service.SetOpsLatencyMsContext(c, service.OpsResponseLatencyMsKey, responseLatencyMs)
		if err == nil && result != nil && result.FirstTokenMs != nil {
			service.SetOpsLatencyMsContext(c, service.OpsTimeToFirstTokenMsKey, int64(*result.FirstTokenMs))
		}
		// #5148 对齐：错误返回携带的部分 result（流中断前上游已计量的 usage）照常
		// 入账；failover 错误恒定 result=nil，不会重复计费。
		submitChatUsage := func(res *service.OpenAIForwardResult) {
			if res == nil {
				return
			}
			userAgent := c.HeaderValue("User-Agent")
			clientIP := c.ClientIP()
			quotaPlatform := service.QuotaPlatform(c.Context(), selectedAPIKey)
			sessionID := extractClientSessionIDContext(c)
			cyberBlocked := getOpsCyberPolicyContext(c) != nil
			h.submitOpenAIUsageRecordTask(c.Context(), res, func(ctx context.Context) {
				if err := h.gatewayService.RecordUsage(ctx, &service.OpenAIRecordUsageInput{
					Result:             res,
					APIKey:             selectedAPIKey,
					User:               selectedAPIKey.User,
					Account:            account,
					Subscription:       subscription,
					InboundEndpoint:    GetInboundEndpointContext(c),
					UpstreamEndpoint:   resolveOpenAIUpstreamEndpointContext(c, account, res),
					UserAgent:          userAgent,
					IPAddress:          clientIP,
					RequestPayloadHash: service.HashUsageRequestPayload(body),
					APIKeyService:      h.apiKeyService,
					QuotaPlatform:      quotaPlatform,
					SessionID:          sessionID,
					ChannelUsageFields: clientRequestedUsageFieldsContext(c, usageChannelMapping, reqModel, res.UpstreamModel),
					PricingAt:          pricingAt,
					CyberBlocked:       cyberBlocked,
				}); err != nil {
					logger.L().With(
						zap.String("component", "handler.openai_gateway.chat_completions"),
						zap.Int64("user_id", subject.UserID),
						zap.Int64("api_key_id", selectedAPIKey.ID),
						zap.Any("group_id", selectedAPIKey.GroupID),
						zap.String("model", reqModel),
						zap.Int64("account_id", account.ID),
					).Error("openai_chat_completions.record_usage_failed", zap.Error(err))
				}
			})
		}
		if err != nil {
			var failoverErr *service.UpstreamFailoverError
			if errors.As(err, &failoverErr) {
				failoverErr = adjustOpenAINetworkFailoverCooldown(failoverErr)
				failoverErr = adjustOpenAIFailoverForLargeStreamingRequest(failoverErr, body, reqStream, h.cfg)
				if !openAIForwardMayFailoverContext(c, writerSizeBeforeForward, failoverErr) {
					h.handleFailoverExhaustedContext(c, failoverErr, true)
					return
				}
				if failoverClientGoneContext(c) {
					return
				}
				if failoverErr.SafeToFailoverAfterWrite && c.ResponseWritten() {
					streamStarted = true
				}
				h.gatewayService.RegisterOpenAIRuntimeFailure(account, failoverErr)
				if failoverErr.ShouldReportAccountScheduleFailure() {
					h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, false, nil)
				}
				if !failoverErr.ShouldRetryNextAccount() {
					h.handleFailoverExhaustedContext(c, failoverErr, streamStarted)
					return
				}
				if c.ResponseSize() != writerSizeBeforeForward {
					h.handleFailoverExhaustedContext(c, failoverErr, true)
					return
				}
				if openAIFirstOutputFailoverExhausted(failoverErr, &firstOutputTimeoutSwitchCount) {
					h.handleFailoverExhaustedContext(c, failoverErr, streamStarted)
					return
				}
				// Pool mode: retry on the same account
				if failoverPolicy.AllowSameAccountRetry && failoverErr.RetryableOnSameAccount {
					retryLimit := account.GetPoolModeRetryCount()
					if sameAccountRetryCount[account.ID] < retryLimit {
						sameAccountRetryCount[account.ID]++
						reqLog.Warn("openai_chat_completions.pool_mode_same_account_retry",
							zap.Int64("account_id", account.ID),
							zap.Int("upstream_status", failoverErr.StatusCode),
							zap.Int("retry_limit", retryLimit),
							zap.Int("retry_count", sameAccountRetryCount[account.ID]),
							zap.Duration("retry_delay", sameAccountRetryDelayFor(failoverErr, sameAccountRetryCount[account.ID])),
						)
						retryDelay := sameAccountRetryDelayFor(failoverErr, sameAccountRetryCount[account.ID])
						select {
						case <-c.Context().Done():
							return
						case <-time.After(retryDelay):
						}
						continue
					}
				}
				_ = h.gatewayService.ClearStickySessionForAccount(c.Context(), selectedAPIKey.GroupID, sessionHash, account.ID)
				h.gatewayService.TempUnscheduleRetryableError(c.Context(), account.ID, failoverErr)
				h.gatewayService.RecordOpenAIAccountSwitch()
				failedAccountIDs[account.ID] = struct{}{}
				lastFailoverErr = failoverErr
				if switchCount >= maxAccountSwitches {
					h.handleFailoverExhaustedContext(c, failoverErr, streamStarted)
					return
				}
				switchCount++
				if h.gatewayService.ShouldStopOpenAIOAuth429Failover(account, failoverErr.StatusCode, switchCount, &oauth429FailoverState) {
					h.handleFailoverExhaustedContext(c, failoverErr, streamStarted)
					return
				}
				reqLog.Warn("openai_chat_completions.upstream_failover_switching",
					zap.Int64("account_id", account.ID),
					zap.Int("upstream_status", failoverErr.StatusCode),
					zap.Int("switch_count", switchCount),
					zap.Int("max_switches", maxAccountSwitches),
				)
				continue
			}
			h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, false, nil)
			wroteFallback := h.ensureForwardErrorResponseContext(c, streamStarted)
			reqLog.Warn("openai_chat_completions.forward_failed",
				zap.Int64("account_id", account.ID),
				zap.Bool("fallback_error_response_written", wroteFallback),
				zap.Error(err),
			)
			submitChatUsage(result)
			return
		}
		if result != nil {
			h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, true, result.FirstTokenMs)
		} else {
			h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, true, nil)
		}
		h.gatewayService.MarkOpenAIAccountHealthy(account)
		if err := h.gatewayService.PromoteStickySession(c.Context(), selectedAPIKey.GroupID, sessionHash, account.ID); err != nil {
			reqLog.Warn("openai_chat_completions.promote_sticky_session_failed", zap.Int64("account_id", account.ID), zap.Error(err))
		}
		submitChatUsage(result)
		reqLog.Debug("openai_chat_completions.request_completed",
			zap.Int64("account_id", account.ID),
			zap.Int("switch_count", switchCount),
		)
		return
	}
}

func (h *OpenAIGatewayHandler) recoverChatCompletionsPanicContext(c gatewayctx.GatewayContext, streamStarted *bool) {
	recovered := recover()
	if recovered == nil {
		return
	}
	started := streamStarted != nil && *streamStarted
	requestLoggerContext(c, "handler.openai_gateway.chat_completions").Error(
		"openai.chat_completions_panic_recovered",
		zap.Bool("stream_started", started),
		zap.Any("panic", recovered),
	)
	if !started {
		h.errorResponseGateway(c, http.StatusInternalServerError, "api_error", "Internal server error")
	}
}
