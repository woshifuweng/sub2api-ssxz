package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	pkghttputil "github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	rustsidecar "github.com/Wei-Shaw/sub2api/internal/rustbridge/sidecar"
	"github.com/Wei-Shaw/sub2api/internal/securityaudit"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"go.uber.org/zap"
)

func usageRecordContext(parent context.Context, base context.Context) context.Context {
	if base == nil {
		base = context.Background()
	}
	if parent == nil {
		return base
	}
	if clientRequestID, _ := parent.Value(ctxkey.ClientRequestID).(string); strings.TrimSpace(clientRequestID) != "" {
		base = context.WithValue(base, ctxkey.ClientRequestID, strings.TrimSpace(clientRequestID))
	}
	if requestID, _ := parent.Value(ctxkey.RequestID).(string); strings.TrimSpace(requestID) != "" {
		base = context.WithValue(base, ctxkey.RequestID, strings.TrimSpace(requestID))
	}
	return base
}

func wrapUsageRecordTaskContext(parent context.Context, task service.UsageRecordTask) service.UsageRecordTask {
	if task == nil {
		return nil
	}
	return func(ctx context.Context) {
		task(usageRecordContext(parent, ctx))
	}
}

func openAICompatibleRequestPlatform(ctx context.Context, apiKey *service.APIKey) string {
	if platform, ok := service.ResolvedTargetPlatformFromContext(ctx); ok {
		if platform == service.PlatformGrok {
			return service.PlatformGrok
		}
		return service.PlatformOpenAI
	}
	if apiKey != nil && apiKey.Group != nil && apiKey.Group.Platform == service.PlatformGrok {
		return service.PlatformGrok
	}
	return service.PlatformOpenAI
}

func allowOpenAICompatibleMessagesDispatch(apiKey *service.APIKey) bool {
	if apiKey == nil || apiKey.Group == nil {
		return true
	}
	if apiKey.Group.Platform == service.PlatformGrok {
		return true
	}
	return apiKey.Group.AllowMessagesDispatch
}

const maxOpenAIFirstOutputTimeoutSwitches = 1

func openAIForwardSucceededForScheduling(result *service.OpenAIForwardResult) bool {
	return result.SucceededForScheduling()
}

func resolveOpenAIMessagesDispatchMappedModel(apiKey *service.APIKey, requestedModel string) string {
	if apiKey == nil || apiKey.Group == nil {
		return ""
	}
	return strings.TrimSpace(apiKey.Group.ResolveMessagesDispatchModel(requestedModel))
}

func seedOpenAIForwardImageIntentHint(c *gin.Context, channelMapped bool, imageIntent bool) {
	if channelMapped {
		return
	}
	service.SetOpenAIImageIntentHint(c, imageIntent)
}

type openAIModelBodyReplaceFunc func(body []byte, newModel string) []byte

func openAIModelMappedBody(body []byte, mapped bool, mappedModel string, replace openAIModelBodyReplaceFunc) []byte {
	if !mapped || strings.TrimSpace(mappedModel) == "" || replace == nil {
		return body
	}
	return replace(body, mappedModel)
}

func newOpenAIModelMappedBodyCache(body []byte, replace openAIModelBodyReplaceFunc) func(bool, string) []byte {
	var (
		cachedModel string
		cachedBody  []byte
	)
	return func(mapped bool, mappedModel string) []byte {
		if !mapped || strings.TrimSpace(mappedModel) == "" || replace == nil {
			return body
		}
		if cachedBody == nil || cachedModel != mappedModel {
			cachedModel = mappedModel
			cachedBody = replace(body, mappedModel)
		}
		return cachedBody
	}
}

func credentialFailoverClientResponse(failoverErr *service.UpstreamFailoverError) (int, string) {
	_ = failoverErr
	return http.StatusServiceUnavailable, service.GrokCredentialUnavailableClientMessage
}

func copyFailoverRetryAfter(c *gin.Context, headers http.Header) {
	if c == nil || headers == nil {
		return
	}
	retryAfter := strings.TrimSpace(headers.Get("Retry-After"))
	if retryAfter == "" || len(retryAfter) > 128 || strings.ContainsAny(retryAfter, "\r\n") || !isSafeRetryAfter(retryAfter) {
		return
	}
	c.Header("Retry-After", retryAfter)
}

func isSafeRetryAfter(value string) bool {
	digitsOnly := true
	for _, char := range value {
		if char < '0' || char > '9' {
			digitsOnly = false
			break
		}
	}
	if digitsOnly {
		seconds, err := strconv.ParseUint(value, 10, 32)
		return err == nil && seconds <= uint64((7*24*time.Hour)/time.Second)
	}
	retryAt, err := http.ParseTime(value)
	if err != nil {
		return false
	}
	return !retryAt.After(time.Now().Add(7 * 24 * time.Hour))
}

// OpenAIGatewayHandler handles OpenAI API gateway requests
type OpenAIGatewayHandler struct {
	gatewayService             *service.OpenAIGatewayService
	billingCacheService        *service.BillingCacheService
	apiKeyService              *service.APIKeyService
	usageRecordWorkerPool      *service.UsageRecordWorkerPool
	errorPassthroughService    *service.ErrorPassthroughService
	contentModerationService   *service.ContentModerationService
	securityAuditCoordinator   *securityaudit.Coordinator
	grokMediaEligibilityProber grokMediaEligibilityProber
	opsService                 *service.OpsService
	concurrencyHelper          *ConcurrencyHelper
	imageLimiter               *imageConcurrencyLimiter
	maxAccountSwitches         int
	cfg                        *config.Config
}

type grokMediaEligibilityProber interface {
	ProbeMediaEligibility(ctx context.Context, accountID int64) (bool, string, error)
}

const (
	openAIStreamLargeBodyThresholdBytes    = 64 * 1024
	openAIStreamXLargeBodyThresholdBytes   = 256 * 1024
	openAIStreamHugeBodyThresholdBytes     = 1024 * 1024
	openAIStreamProxyFailureCooldown       = 20 * time.Minute
	openAIStreamAccountNetworkCooldown     = 12 * time.Minute
	openAIStreamLongContextTimeoutCooldown = 6 * time.Minute
)

type openAIFailoverPolicy struct {
	MaxSwitches           int
	AllowSameAccountRetry bool
}

func applyOpenAIRemoteCompactFailoverPolicy(policy openAIFailoverPolicy, remoteCompact bool) openAIFailoverPolicy {
	if !remoteCompact {
		return policy
	}
	policy.AllowSameAccountRetry = false
	return policy
}

func shouldRetryOpenAIRemoteCompactSilently(
	failoverErr *service.UpstreamFailoverError,
	previousResponseID string,
	switchCount int,
) bool {
	if failoverErr == nil || switchCount > 0 {
		return false
	}
	// previous_response_id binds server-side history to one account/session;
	// retrying on another account is unlikely to be seamless.
	if strings.TrimSpace(previousResponseID) != "" {
		return false
	}
	switch failoverErr.StatusCode {
	case http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusTooManyRequests, 554:
		return false
	}
	reason := strings.ToLower(strings.TrimSpace(failoverErr.TempUnscheduleReason))
	bodyText := strings.ToLower(strings.TrimSpace(string(failoverErr.ResponseBody)))
	if failoverErr.FailedProxyID > 0 ||
		strings.Contains(reason, "proxy") ||
		strings.Contains(reason, "network") ||
		strings.Contains(reason, "timeout") ||
		strings.Contains(bodyText, "context deadline exceeded") ||
		strings.Contains(bodyText, "timeout") ||
		strings.Contains(bodyText, "connection refused") ||
		strings.Contains(bodyText, "connection reset") ||
		strings.Contains(bodyText, "socks connect") ||
		strings.Contains(bodyText, "eof") {
		return true
	}
	switch failoverErr.StatusCode {
	case 0, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func shouldSwitchOpenAIRemoteCompactAccount(
	failoverErr *service.UpstreamFailoverError,
	previousResponseID string,
	switchCount int,
	maxSwitches int,
) bool {
	if failoverErr == nil || maxSwitches <= 0 || switchCount >= maxSwitches {
		return false
	}
	if strings.TrimSpace(previousResponseID) != "" {
		return false
	}

	statusCode := failoverErr.StatusCode
	switch statusCode {
	case http.StatusUnauthorized, http.StatusTooManyRequests, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout, 529:
		return true
	}
	if statusCode >= 500 {
		return true
	}

	reason := strings.ToLower(strings.TrimSpace(failoverErr.TempUnscheduleReason))
	bodyText := strings.ToLower(strings.TrimSpace(string(failoverErr.ResponseBody)))
	return failoverErr.FailedProxyID > 0 ||
		strings.Contains(reason, "proxy") ||
		strings.Contains(reason, "network") ||
		strings.Contains(reason, "timeout") ||
		strings.Contains(bodyText, "context deadline exceeded") ||
		strings.Contains(bodyText, "timeout") ||
		strings.Contains(bodyText, "connection refused") ||
		strings.Contains(bodyText, "connection reset") ||
		strings.Contains(bodyText, "socks connect") ||
		strings.Contains(bodyText, "eof")
}

func resolveOpenAIForwardDefaultMappedModel(apiKey *service.APIKey, fallbackModel string) string {
	if fallbackModel = strings.TrimSpace(fallbackModel); fallbackModel != "" {
		return fallbackModel
	}
	if apiKey == nil || apiKey.Group == nil {
		return ""
	}
	return strings.TrimSpace(apiKey.Group.DefaultMappedModel)
}

func resolveOpenAISelectionFallbackModel(_ *service.APIKey, _ string) string {
	// Account selection must match the requested model. Silently retrying a group
	// default can route an unsupported client model to an unrelated upstream.
	return ""
}

func handleOpenAISelectionExhausted(
	ctx context.Context,
	failedAccountIDs map[int64]struct{},
	lastFailoverErr *service.UpstreamFailoverError,
	switchCount int,
	maxSwitches int,
) (map[int64]struct{}, FailoverAction) {
	fs := NewFailoverState(maxSwitches, false)
	fs.FailedAccountIDs = failedAccountIDs
	fs.LastFailoverErr = lastFailoverErr
	fs.SwitchCount = switchCount
	return fs.FailedAccountIDs, fs.HandleSelectionExhausted(ctx)
}

func clampOpenAIMaxSwitches(limit, cap int) int {
	if limit < 0 {
		limit = 0
	}
	if cap < 0 {
		cap = 0
	}
	if limit == 0 || cap == 0 {
		return 0
	}
	if limit < cap {
		return limit
	}
	return cap
}

func deriveOpenAIStreamingFallbackSeed(c *gin.Context, apiKey *service.APIKey, userID int64, reqModel string) string {
	return deriveOpenAIStreamingFallbackSeedContext(gatewayctx.FromGin(c), apiKey, userID, reqModel)
}

func deriveOpenAIStreamingFallbackSeedContext(ctx gatewayctx.GatewayContext, apiKey *service.APIKey, userID int64, reqModel string) string {
	if apiKey == nil || userID <= 0 {
		return ""
	}

	groupID := int64(0)
	if apiKey.GroupID != nil {
		groupID = *apiKey.GroupID
	}
	clientIP := strings.TrimSpace(ctx.ClientIP())

	return fmt.Sprintf(
		"openai_http_stream:%d:%d:%d:%s:%s:%s",
		userID,
		apiKey.ID,
		groupID,
		strings.TrimSpace(reqModel),
		clientIP,
		strings.TrimSpace(ctx.HeaderValue("User-Agent")),
	)
}

func (h *OpenAIGatewayHandler) resolveOpenAIStickySessionHash(
	c *gin.Context,
	apiKey *service.APIKey,
	userID int64,
	reqModel string,
	reqStream bool,
	body []byte,
) string {
	return h.resolveOpenAIStickySessionHashContext(gatewayctx.FromGin(c), apiKey, userID, reqModel, reqStream, body)
}

func (h *OpenAIGatewayHandler) resolveOpenAIStickySessionHashContext(
	ctx gatewayctx.GatewayContext,
	apiKey *service.APIKey,
	userID int64,
	reqModel string,
	reqStream bool,
	body []byte,
) string {
	if h == nil || h.gatewayService == nil {
		return ""
	}
	if !reqStream {
		return h.gatewayService.GenerateSessionHashContext(ctx, body)
	}

	seed := deriveOpenAIStreamingFallbackSeedContext(ctx, apiKey, userID, reqModel)
	return h.gatewayService.GenerateSessionHashWithFallbackContext(ctx, body, seed)
}

func (h *OpenAIGatewayHandler) resolveOpenAIFailoverPolicy(body []byte, reqStream bool) openAIFailoverPolicy {
	limit := h.maxAccountSwitches
	policy := openAIFailoverPolicy{
		MaxSwitches:           limit,
		AllowSameAccountRetry: true,
	}
	if !reqStream {
		return policy
	}

	large, xlarge, huge := h.openAIStreamingBodyThresholds()
	bodySize := len(body)
	switch {
	case huge > 0 && bodySize >= huge:
		policy.MaxSwitches = clampOpenAIMaxSwitches(limit, 0)
		policy.AllowSameAccountRetry = false
	case xlarge > 0 && bodySize >= xlarge:
		policy.MaxSwitches = clampOpenAIMaxSwitches(limit, 1)
		policy.AllowSameAccountRetry = false
	case large > 0 && bodySize >= large:
		policy.MaxSwitches = clampOpenAIMaxSwitches(limit, 2)
		policy.AllowSameAccountRetry = false
	}
	return policy
}

func adjustOpenAIFailoverForLargeStreamingRequest(
	failoverErr *service.UpstreamFailoverError,
	body []byte,
	reqStream bool,
	cfg *config.Config,
) *service.UpstreamFailoverError {
	if failoverErr == nil || !reqStream {
		return failoverErr
	}
	large, xlarge, huge := resolveOpenAIStreamingThresholds(cfg)
	bodySize := len(body)
	if large <= 0 || bodySize < large {
		return failoverErr
	}

	adjusted := *failoverErr
	adjusted.RetryableOnSameAccount = false
	if adjusted.TempUnscheduleFor <= 0 {
		switch {
		case huge > 0 && bodySize >= huge:
			adjusted.TempUnscheduleFor = openAIStreamLongContextTimeoutCooldown
		case xlarge > 0 && bodySize >= xlarge:
			adjusted.TempUnscheduleFor = 4 * time.Minute
		default:
			adjusted.TempUnscheduleFor = 2 * time.Minute
		}
		if strings.TrimSpace(adjusted.TempUnscheduleReason) == "" {
			adjusted.TempUnscheduleReason = "openai streaming long-context failover cooldown"
		}
	}
	return &adjusted
}

func resolveOpenAIStreamingThresholds(cfg *config.Config) (int, int, int) {
	large := openAIStreamLargeBodyThresholdBytes
	xlarge := openAIStreamXLargeBodyThresholdBytes
	huge := openAIStreamHugeBodyThresholdBytes
	if cfg == nil {
		return large, xlarge, huge
	}
	streaming := cfg.Gateway.OpenAI.Streaming
	if streaming.LargeBodyThresholdBytes > 0 {
		large = streaming.LargeBodyThresholdBytes
	}
	if streaming.XLargeBodyThresholdBytes > 0 {
		xlarge = streaming.XLargeBodyThresholdBytes
	}
	if streaming.HugeBodyThresholdBytes > 0 {
		huge = streaming.HugeBodyThresholdBytes
	}
	return large, xlarge, huge
}

func (h *OpenAIGatewayHandler) openAIStreamingBodyThresholds() (int, int, int) {
	if h == nil {
		return resolveOpenAIStreamingThresholds(nil)
	}
	return resolveOpenAIStreamingThresholds(h.cfg)
}

func adjustOpenAINetworkFailoverCooldown(
	failoverErr *service.UpstreamFailoverError,
) *service.UpstreamFailoverError {
	if failoverErr == nil {
		return nil
	}
	if !strings.Contains(strings.ToLower(strings.TrimSpace(failoverErr.Error())), "failover") {
		return failoverErr
	}
	if !strings.Contains(strings.ToLower(strings.TrimSpace(failoverErr.TempUnscheduleReason)), "proxy") &&
		!strings.Contains(strings.ToLower(strings.TrimSpace(failoverErr.TempUnscheduleReason)), "network") {
		return failoverErr
	}
	adjusted := *failoverErr
	if adjusted.FailedProxyID > 0 || strings.TrimSpace(adjusted.FailedProxyURL) != "" {
		adjusted.TempUnscheduleFor = openAIStreamProxyFailureCooldown
		adjusted.TempUnscheduleReason = "upstream request failed via proxy/network (auto temp-unschedule 20m)"
		return &adjusted
	}
	if adjusted.TempUnscheduleFor < openAIStreamAccountNetworkCooldown {
		adjusted.TempUnscheduleFor = openAIStreamAccountNetworkCooldown
		adjusted.TempUnscheduleReason = "upstream request failed via network (auto temp-unschedule 12m)"
	}
	return &adjusted
}

func openAICompatibleTextTargetAllowed(c *gin.Context, apiKey *service.APIKey, model string) bool {
	return compositeTargetPlatformAllowed(c, apiKey, model, service.PlatformOpenAI, service.PlatformGrok)
}

// ensureCompositeTargetPlatformContext 桥接 gatewayctx 与 gin-based composite 解析。
// 非 gin 传输不携带 composite 上下文，保持 v0.1.163 行为（composite 分组仅在 gin 入口生效）。
func ensureCompositeTargetPlatformContext(c gatewayctx.GatewayContext, apiKey *service.APIKey, model string) {
	if c == nil {
		return
	}
	if native, ok := c.Native().(*gin.Context); ok && native != nil {
		ensureCompositeTargetPlatform(native, apiKey, model)
	}
}

func openAICompatibleTextTargetAllowedContext(c gatewayctx.GatewayContext, apiKey *service.APIKey, model string) bool {
	if c != nil {
		if native, ok := c.Native().(*gin.Context); ok && native != nil {
			return openAICompatibleTextTargetAllowed(native, apiKey, model)
		}
	}
	// 非 gin 传输：composite 分组无法解析目标平台，仅 composite 分组拒绝。
	return apiKey == nil || apiKey.Group == nil || apiKey.Group.Platform != service.PlatformComposite
}

func compositeTargetPlatformAllowedContext(c gatewayctx.GatewayContext, apiKey *service.APIKey, model string, allowed ...string) bool {
	if c != nil {
		if native, ok := c.Native().(*gin.Context); ok && native != nil {
			return compositeTargetPlatformAllowed(native, apiKey, model, allowed...)
		}
	}
	return apiKey == nil || apiKey.Group == nil || apiKey.Group.Platform != service.PlatformComposite
}

func extractClientSessionIDContext(c gatewayctx.GatewayContext) string {
	if c == nil {
		return ""
	}
	if native, ok := c.Native().(*gin.Context); ok && native != nil {
		return service.ExtractClientSessionID(native)
	}
	return ""
}

func clientRequestedUsageFieldsContext(c gatewayctx.GatewayContext, mapping service.ChannelMappingResult, fallbackModel, upstreamModel string) service.ChannelUsageFields {
	if c != nil {
		if native, ok := c.Native().(*gin.Context); ok && native != nil {
			return clientRequestedUsageFields(native, mapping, fallbackModel, upstreamModel)
		}
	}
	return mapping.ToUsageFields(fallbackModel, upstreamModel)
}

// NewOpenAIGatewayHandler creates a new OpenAIGatewayHandler
func NewOpenAIGatewayHandler(
	gatewayService *service.OpenAIGatewayService,
	concurrencyService *service.ConcurrencyService,
	billingCacheService *service.BillingCacheService,
	apiKeyService *service.APIKeyService,
	usageRecordWorkerPool *service.UsageRecordWorkerPool,
	errorPassthroughService *service.ErrorPassthroughService,
	contentModerationService *service.ContentModerationService,
	opsService *service.OpsService,
	cfg *config.Config,
) *OpenAIGatewayHandler {
	pingInterval := time.Duration(0)
	maxAccountSwitches := 3
	if cfg != nil {
		pingInterval = time.Duration(cfg.Concurrency.PingInterval) * time.Second
		if cfg.Gateway.MaxAccountSwitches > 0 {
			maxAccountSwitches = cfg.Gateway.MaxAccountSwitches
		}
	}
	return &OpenAIGatewayHandler{
		gatewayService:           gatewayService,
		billingCacheService:      billingCacheService,
		apiKeyService:            apiKeyService,
		usageRecordWorkerPool:    usageRecordWorkerPool,
		errorPassthroughService:  errorPassthroughService,
		contentModerationService: contentModerationService,
		opsService:               opsService,
		concurrencyHelper:        NewConcurrencyHelper(concurrencyService, SSEPingFormatComment, pingInterval),
		imageLimiter:             &imageConcurrencyLimiter{},
		maxAccountSwitches:       maxAccountSwitches,
		cfg:                      cfg,
	}
}

func (h *OpenAIGatewayHandler) checkOpenAITokenBillingEligibilityContext(
	c gatewayctx.GatewayContext,
	reqLog *zap.Logger,
	apiKey *service.APIKey,
	subscription *service.UserSubscription,
	model string,
	body []byte,
	streamStarted bool,
	logEvent string,
	writeErr func(gatewayctx.GatewayContext, int, string, string, bool),
) bool {
	if h == nil || h.billingCacheService == nil {
		return false
	}
	estimatedCost, estimateErr := h.gatewayService.EstimateOpenAITokenRequestCost(c.Context(), model, body, apiKey, apiKey.User)
	if estimateErr != nil && reqLog != nil {
		reqLog.Warn(logEvent+".token_cost_estimate_failed", zap.Error(estimateErr))
	}

	var err error
	if estimatedCost != nil && estimatedCost.ActualCost > 0 {
		err = h.billingCacheService.CheckBillingEligibilityForCost(c.Context(), apiKey.User, apiKey, apiKey.Group, subscription, estimatedCost.ActualCost)
	} else {
		err = h.billingCacheService.CheckBillingEligibility(c.Context(), apiKey.User, apiKey, apiKey.Group, subscription, service.QuotaPlatform(c.Context(), apiKey))
	}
	if err == nil {
		return true
	}

	if reqLog != nil {
		reqLog.Info(logEvent+".billing_eligibility_check_failed", zap.Error(err))
	}
	status, code, message, retryAfter := billingErrorDetails(err)
	if retryAfter > 0 {
		c.SetHeader("Retry-After", strconv.Itoa(retryAfter))
	}
	writeErr(c, status, code, message, streamStarted)
	return false
}

// Responses handles OpenAI Responses API endpoint
// POST /openai/v1/responses
func (h *OpenAIGatewayHandler) Responses(c *gin.Context) {
	h.ResponsesGateway(gatewayctx.FromGin(c))
}

func (h *OpenAIGatewayHandler) ResponsesGateway(transportCtx gatewayctx.GatewayContext) {
	// 局部兜底：确保该 handler 内部任何 panic 都不会击穿到进程级。
	streamStarted := false
	defer h.recoverResponsesPanicContext(transportCtx, &streamStarted)
	compactStartedAt := time.Now()
	defer h.logOpenAIRemoteCompactOutcomeContext(transportCtx, compactStartedAt)
	remoteCompact := isOpenAIRemoteCompactPathContext(transportCtx)
	setOpenAIClientTransportHTTPGateway(transportCtx)

	requestStart := time.Now()

	// Get apiKey and user from context (set by ApiKeyAuth middleware)
	apiKey, ok := middleware2.GetAPIKeyFromGatewayContext(transportCtx)
	if !ok {
		h.errorResponseGateway(transportCtx, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}

	subject, ok := middleware2.GetAuthSubjectFromGatewayContext(transportCtx)
	if !ok {
		h.errorResponseGateway(transportCtx, http.StatusInternalServerError, "api_error", "User context not found")
		return
	}
	reqLog := requestLoggerContext(
		transportCtx,
		"handler.openai_gateway.responses",
		zap.Int64("user_id", subject.UserID),
		zap.Int64("api_key_id", apiKey.ID),
		zap.Any("group_id", apiKey.GroupID),
	)
	if !h.ensureResponsesDependenciesGateway(transportCtx, reqLog) {
		return
	}

	// Read request body
	body, err := pkghttputil.ReadRequestBodyWithPrealloc(transportCtx.Request())
	if err != nil {
		if maxErr, ok := extractMaxBytesError(err); ok {
			h.errorResponseGateway(transportCtx, http.StatusRequestEntityTooLarge, "invalid_request_error", buildBodyTooLargeMessage(maxErr.Limit))
			return
		}
		h.errorResponseGateway(transportCtx, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
		return
	}

	if len(body) == 0 {
		h.errorResponseGateway(transportCtx, http.StatusBadRequest, "invalid_request_error", "Request body is empty")
		return
	}
	body, compactRequestOK := h.normalizeOpenAIResponsesCompactRequestContext(transportCtx, reqLog, body)
	if !compactRequestOK {
		h.errorResponseGateway(transportCtx, http.StatusBadRequest, "invalid_request_error", "Failed to normalize compact request body")
		return
	}
	remoteCompact = isOpenAIRemoteCompactPathContext(transportCtx)

	setOpsRequestContextGateway(transportCtx, "", false, body)
	sessionHashBody := body
	if service.IsOpenAIResponsesCompactPathForTestContext(transportCtx) {
		if compactSeed := strings.TrimSpace(gjson.GetBytes(body, "prompt_cache_key").String()); compactSeed != "" {
			transportCtx.SetValue(service.OpenAICompactSessionSeedKeyForTest(), compactSeed)
		}
		normalizedCompactBody, normalizedCompact, compactErr := service.NormalizeOpenAICompactRequestBodyForTest(body)
		if compactErr != nil {
			h.errorResponseGateway(transportCtx, http.StatusBadRequest, "invalid_request_error", "Failed to normalize compact request body")
			return
		}
		if normalizedCompact {
			body = normalizedCompactBody
		}
	}

	// 校验请求体 JSON 合法性
	if !gjson.ValidBytes(body) {
		h.errorResponseGateway(transportCtx, http.StatusBadRequest, "invalid_request_error", "Failed to parse request body")
		return
	}
	body, _, err = service.EnforceUnboundedTokenRequestLimit(body, "max_output_tokens", "max_output_tokens")
	if err != nil {
		h.errorResponseGateway(transportCtx, http.StatusBadRequest, "invalid_request_error", "Failed to apply output token limit")
		return
	}

	// 使用 gjson 只读提取字段做校验，避免完整 Unmarshal
	modelResult := gjson.GetBytes(body, "model")
	if !modelResult.Exists() || modelResult.Type != gjson.String || modelResult.String() == "" {
		h.errorResponseGateway(transportCtx, http.StatusBadRequest, "invalid_request_error", "model is required")
		return
	}
	reqModel := modelResult.String()
	ensureCompositeTargetPlatformContext(transportCtx, apiKey, reqModel)
	if !openAICompatibleTextTargetAllowedContext(transportCtx, apiKey, reqModel) {
		h.errorResponseGateway(transportCtx, http.StatusBadRequest, "invalid_request_error", "Model is not supported by this OpenAI-compatible endpoint for composite groups")
		return
	}
	requestPlatform := openAICompatibleRequestPlatform(transportCtx.Context(), apiKey)
	imageIntent := service.IsExplicitImageGenerationIntent("/v1/responses", reqModel, body)
	if requestPlatform == service.PlatformGrok {
		imageIntent = service.IsImageGenerationIntentForPlatform("/v1/responses", reqModel, body, requestPlatform)
	}
	if imageIntent {
		transportCtx.SetRequest(transportCtx.Request().WithContext(service.WithOpenAIImageGenerationIntent(transportCtx.Context())))
		if !service.GroupAllowsImageGeneration(apiKey.Group) {
			h.errorResponseGateway(transportCtx, http.StatusForbidden, "permission_error", service.ImageGenerationPermissionMessage())
			return
		}
	}
	canonicalModel, catalogEnforced, catalogAvailable := canonicalCustomerGatewayModelForAPIKey(apiKey, reqModel)
	if catalogEnforced && !catalogAvailable {
		h.errorResponseGateway(transportCtx, http.StatusBadRequest, "invalid_request_error", customerGatewayModelUnavailableMessage(reqModel))
		return
	}
	if catalogEnforced && canonicalModel != reqModel {
		body, err = sjson.SetBytes(body, "model", canonicalModel)
		if err != nil {
			h.errorResponseGateway(transportCtx, http.StatusBadRequest, "invalid_request_error", "Failed to normalize model")
			return
		}
		reqModel = canonicalModel
	}
	if !apiKeyAllowsRequestedModel(apiKey, reqModel) {
		h.errorResponseGateway(transportCtx, http.StatusBadRequest, "invalid_request_error", apiKeyModelNotAllowedMessage(reqModel))
		return
	}

	streamResult := gjson.GetBytes(body, "stream")
	if streamResult.Exists() && streamResult.Type != gjson.True && streamResult.Type != gjson.False {
		h.errorResponseGateway(transportCtx, http.StatusBadRequest, "invalid_request_error", "invalid stream field type")
		return
	}
	reqStream := streamResult.Bool()
	reqLog = reqLog.With(zap.String("model", reqModel), zap.Bool("stream", reqStream))
	previousResponseID := strings.TrimSpace(gjson.GetBytes(body, "previous_response_id").String())
	if previousResponseID != "" {
		previousResponseIDKind := service.ClassifyOpenAIPreviousResponseIDKind(previousResponseID)
		reqLog = reqLog.With(
			zap.Bool("has_previous_response_id", true),
			zap.String("previous_response_id_kind", previousResponseIDKind),
			zap.Int("previous_response_id_len", len(previousResponseID)),
		)
		if previousResponseIDKind == service.OpenAIPreviousResponseIDKindMessageID {
			reqLog.Warn("openai.request_validation_failed",
				zap.String("reason", "previous_response_id_looks_like_message_id"),
			)
			h.errorResponseGateway(transportCtx, http.StatusBadRequest, "invalid_request_error", "previous_response_id must be a response.id (resp_*), not a message id")
			return
		}
		reqLog.Warn("openai.request_validation_failed",
			zap.String("reason", "previous_response_id_requires_wsv2"),
		)
		h.errorResponseGateway(transportCtx, http.StatusBadRequest, "invalid_request_error", "previous_response_id is only supported on Responses WebSocket v2")
		return
	}

	setOpsRequestContextGateway(transportCtx, reqModel, reqStream, body)
	setOpsEndpointContextGateway(transportCtx, "", int16(service.RequestTypeFromLegacy(reqStream, false)))
	channelMapping, _ := h.gatewayService.ResolveChannelMappingAndRestrict(transportCtx.Context(), apiKey.GroupID, reqModel)
	if decision := h.checkSecurityAuditContext(transportCtx, reqLog, apiKey, subject, service.ContentModerationProtocolOpenAIResponses, reqModel, body); decision != nil && !decision.AllowNextStage {
		h.openAISecurityAuditErrorContext(transportCtx, decision)
		return
	}
	if imageIntent {
		imageReleaseFunc, acquired := h.acquireImageGenerationSlotContext(transportCtx, streamStarted)
		if !acquired {
			return
		}
		if imageReleaseFunc != nil {
			defer imageReleaseFunc()
		}
	}

	// 提前校验 function_call_output 是否具备可关联上下文，避免上游 400。
	if !h.validateFunctionCallOutputRequestContext(transportCtx, body, reqLog) {
		return
	}

	// 绑定错误透传服务，允许 service 层在非 failover 错误场景复用规则。
	if h.errorPassthroughService != nil {
		service.BindErrorPassthroughServiceContext(transportCtx, h.errorPassthroughService)
	}

	// Get subscription info (may be nil)
	subscription, _ := middleware2.GetSubscriptionFromGatewayContext(transportCtx)

	service.SetOpsLatencyMsContext(transportCtx, service.OpsAuthLatencyMsKey, time.Since(requestStart).Milliseconds())
	routingStart := time.Now()

	userReleaseFunc, acquired := h.acquireResponsesUserSlotContext(transportCtx, subject.UserID, subject.Concurrency, reqStream, &streamStarted, reqLog)
	if !acquired {
		return
	}
	// 确保请求取消时也会释放槽位，避免长连接被动中断造成泄漏
	if userReleaseFunc != nil {
		defer userReleaseFunc()
	}

	// 2. Re-check billing eligibility after wait
	if !h.checkOpenAITokenBillingEligibilityContext(
		transportCtx,
		reqLog,
		apiKey,
		subscription,
		reqModel,
		body,
		streamStarted,
		"openai",
		h.handleStreamingAwareErrorContext,
	) {
		return
	}

	// Generate sticky session hash. For streaming requests, fall back to a stable
	// per-user/per-key seed when the client did not send session headers.
	sessionHash := h.resolveOpenAIStickySessionHashContext(transportCtx, apiKey, subject.UserID, reqModel, reqStream, sessionHashBody)

	failoverPolicy := h.resolveOpenAIFailoverPolicy(body, reqStream)
	failoverPolicy = applyOpenAIRemoteCompactFailoverPolicy(failoverPolicy, remoteCompact)
	maxAccountSwitches := failoverPolicy.MaxSwitches
	if maxAccountSwitches != h.maxAccountSwitches || !failoverPolicy.AllowSameAccountRetry {
		reqLog.Info("openai.streaming_failover_policy_applied",
			zap.Int("body_bytes", len(body)),
			zap.Bool("stream", reqStream),
			zap.Bool("remote_compact", remoteCompact),
			zap.Int("effective_max_switches", maxAccountSwitches),
			zap.Bool("allow_same_account_retry", failoverPolicy.AllowSameAccountRetry),
		)
	}
	switchCount := 0
	firstOutputTimeoutSwitchCount := 0
	failedAccountIDs := make(map[int64]struct{})
	sameAccountRetryCount := make(map[int64]int)
	var lastFailoverErr *service.UpstreamFailoverError

	for {
		if !openAIRequestAllowsFailoverReplayContext(transportCtx) {
			return
		}
		transportCtx.SetValue("openai_responses_fallback_model", "")
		// Select account supporting the requested model
		reqLog.Debug("openai.account_selecting", zap.Int("excluded_account_count", len(failedAccountIDs)))
		groupSelection, err := selectOpenAIAPIKeyGroupWithCompact(
			transportCtx.Context(),
			apiKey,
			previousResponseID,
			sessionHash,
			reqModel,
			failedAccountIDs,
			service.OpenAIUpstreamTransportAny,
			remoteCompact,
			h.gatewayService.SelectAccountWithSchedulerWithCompact,
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
			reqLog.Warn("openai.account_select_failed",
				zap.Error(err),
				zap.Int("excluded_account_count", len(failedAccountIDs)),
			)
			if len(failedAccountIDs) == 0 {
				fallbackModel := resolveOpenAISelectionFallbackModel(apiKey, reqModel)
				if fallbackModel != "" {
					reqLog.Info("openai.fallback_to_default_model",
						zap.String("default_mapped_model", fallbackModel),
					)
					groupSelection, err = selectOpenAIAPIKeyGroupWithCompact(
						transportCtx.Context(),
						apiKey,
						previousResponseID,
						sessionHash,
						fallbackModel,
						failedAccountIDs,
						service.OpenAIUpstreamTransportAny,
						remoteCompact,
						h.gatewayService.SelectAccountWithSchedulerWithCompact,
					)
					if err == nil && groupSelection != nil {
						selection = groupSelection.Selection
						scheduleDecision = groupSelection.Decision
						selectedAPIKey = groupSelection.APIKey
						transportCtx.SetValue("openai_responses_fallback_model", fallbackModel)
					}
				}
				if err != nil {
					if errors.Is(err, service.ErrNoAvailableCompactAccounts) {
						h.handleStreamingAwareErrorContext(transportCtx, http.StatusServiceUnavailable, "compact_not_supported", "No available OpenAI accounts support /responses/compact", streamStarted)
						return
					}
					h.handleStreamingAwareErrorContext(transportCtx, http.StatusServiceUnavailable, "no_available_account", "No available upstream account supports the requested model", streamStarted)
					return
				}
			} else {
				var action FailoverAction
				failedAccountIDs, action = handleOpenAISelectionExhausted(
					transportCtx.Context(),
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
						h.handleFailoverExhaustedContext(transportCtx, lastFailoverErr, streamStarted)
					} else {
						h.handleFailoverExhaustedSimpleContext(transportCtx, 502, streamStarted)
					}
					return
				}
			}
		}
		if selection == nil || selection.Account == nil {
			h.handleStreamingAwareErrorContext(transportCtx, http.StatusServiceUnavailable, "no_available_account", "No available upstream account supports the requested model", streamStarted)
			return
		}
		if previousResponseID != "" && selection != nil && selection.Account != nil {
			reqLog.Debug("openai.account_selected_with_previous_response_id", zap.Int64("account_id", selection.Account.ID))
		}
		reqLog.Debug("openai.account_schedule_decision",
			zap.String("layer", scheduleDecision.Layer),
			zap.Bool("sticky_previous_hit", scheduleDecision.StickyPreviousHit),
			zap.Bool("sticky_session_hit", scheduleDecision.StickySessionHit),
			zap.Int("candidate_count", scheduleDecision.CandidateCount),
			zap.Int("top_k", scheduleDecision.TopK),
			zap.Int64("latency_ms", scheduleDecision.LatencyMs),
			zap.Float64("load_skew", scheduleDecision.LoadSkew),
		)
		account := selection.Account
		sessionHash = ensureOpenAIPoolModeSessionHash(sessionHash, account)
		reqLog.Debug("openai.account_selected", zap.Int64("account_id", account.ID), zap.String("account_name", account.Name))
		setOpsSelectedAccountGateway(transportCtx, account.ID, account.Platform)

		accountReleaseFunc, acquired := h.acquireResponsesAccountSlotContext(transportCtx, selectedAPIKey.GroupID, sessionHash, selection, reqStream, &streamStarted, reqLog)
		if !acquired {
			return
		}

		// Forward request
		service.SetOpsLatencyMsContext(transportCtx, service.OpsRoutingLatencyMsKey, time.Since(routingStart).Milliseconds())
		forwardStart := time.Now()
		writerSizeBeforeForward := service.OpenAICompactKeepaliveAdjustedWrittenSizeContext(transportCtx)
		defaultMappedModel := resolveOpenAIForwardDefaultMappedModel(selectedAPIKey, getContextStringGateway(transportCtx, "openai_responses_fallback_model"))
		forwardBody := body
		if channelMapping.Mapped {
			forwardBody = h.gatewayService.ReplaceModelInBody(body, channelMapping.MappedModel)
		}
		result, err := h.gatewayService.ForwardContext(transportCtx.Context(), transportCtx, account, forwardBody, defaultMappedModel)
		forwardDurationMs := time.Since(forwardStart).Milliseconds()
		if accountReleaseFunc != nil {
			accountReleaseFunc()
		}
		upstreamLatencyMs, _ := getContextInt64Gateway(transportCtx, service.OpsUpstreamLatencyMsKey)
		responseLatencyMs := forwardDurationMs
		if upstreamLatencyMs > 0 && forwardDurationMs > upstreamLatencyMs {
			responseLatencyMs = forwardDurationMs - upstreamLatencyMs
		}
		service.SetOpsLatencyMsContext(transportCtx, service.OpsResponseLatencyMsKey, responseLatencyMs)
		if err == nil && result != nil && result.FirstTokenMs != nil {
			service.SetOpsLatencyMsContext(transportCtx, service.OpsTimeToFirstTokenMsKey, int64(*result.FirstTokenMs))
		}
		if err != nil {
			var failoverErr *service.UpstreamFailoverError
			if errors.As(err, &failoverErr) {
				failoverErr = adjustOpenAINetworkFailoverCooldown(failoverErr)
				failoverErr = adjustOpenAIFailoverForLargeStreamingRequest(failoverErr, body, reqStream, h.cfg)
				if !openAIForwardMayFailoverContext(transportCtx, writerSizeBeforeForward, failoverErr) {
					h.handleFailoverExhaustedContext(transportCtx, failoverErr, true)
					return
				}
				if failoverErr.SafeToFailoverAfterWrite && transportCtx.ResponseWritten() {
					streamStarted = true
				}
				h.gatewayService.RegisterOpenAIRuntimeFailure(account, failoverErr)
				if failoverErr.ShouldReportAccountScheduleFailure() {
					h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, false, nil)
				}
				if !failoverErr.ShouldRetryNextAccount() {
					h.handleFailoverExhaustedContext(transportCtx, failoverErr, streamStarted)
					return
				}
				if openAIFirstOutputFailoverExhausted(failoverErr, &firstOutputTimeoutSwitchCount) {
					h.handleFailoverExhaustedContext(transportCtx, failoverErr, streamStarted)
					return
				}
				if remoteCompact {
					if failoverErr.TempUnscheduleFor > 0 || failoverErr.StatusCode == http.StatusUnauthorized {
						_ = h.gatewayService.ClearStickySessionForAccount(transportCtx.Context(), selectedAPIKey.GroupID, sessionHash, account.ID)
						h.gatewayService.TempUnscheduleRetryableError(transportCtx.Context(), account.ID, failoverErr)
					}
					if shouldSwitchOpenAIRemoteCompactAccount(failoverErr, previousResponseID, switchCount, maxAccountSwitches) {
						h.gatewayService.RecordOpenAIAccountSwitch()
						failedAccountIDs[account.ID] = struct{}{}
						lastFailoverErr = failoverErr
						switchCount++
						reqLog.Warn("openai.remote_compact_retrying",
							zap.Int64("account_id", account.ID),
							zap.Int("upstream_status", failoverErr.StatusCode),
							zap.Int("switch_count", switchCount),
							zap.String("reason", strings.TrimSpace(failoverErr.TempUnscheduleReason)),
						)
						continue
					}
					h.handleRemoteCompactFailureContext(transportCtx, failoverErr, streamStarted)
					return
				}
				// 池模式：同账号重试
				if failoverPolicy.AllowSameAccountRetry && failoverErr.RetryableOnSameAccount {
					retryLimit := account.GetPoolModeRetryCount()
					if sameAccountRetryCount[account.ID] < retryLimit {
						sameAccountRetryCount[account.ID]++
						reqLog.Warn("openai.pool_mode_same_account_retry",
							zap.Int64("account_id", account.ID),
							zap.Int("upstream_status", failoverErr.StatusCode),
							zap.Int("retry_limit", retryLimit),
							zap.Int("retry_count", sameAccountRetryCount[account.ID]),
						)
						select {
						case <-transportCtx.Context().Done():
							return
						case <-time.After(sameAccountRetryDelay):
						}
						continue
					}
				}
				_ = h.gatewayService.ClearStickySessionForAccount(transportCtx.Context(), selectedAPIKey.GroupID, sessionHash, account.ID)
				h.gatewayService.TempUnscheduleRetryableError(transportCtx.Context(), account.ID, failoverErr)
				h.gatewayService.RecordOpenAIAccountSwitch()
				failedAccountIDs[account.ID] = struct{}{}
				lastFailoverErr = failoverErr
				if switchCount >= maxAccountSwitches {
					h.handleFailoverExhaustedContext(transportCtx, failoverErr, streamStarted)
					return
				}
				switchCount++
				reqLog.Warn("openai.upstream_failover_switching",
					zap.Int64("account_id", account.ID),
					zap.Int("upstream_status", failoverErr.StatusCode),
					zap.Int("switch_count", switchCount),
					zap.Int("max_switches", maxAccountSwitches),
				)
				continue
			}
			h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, false, nil)
			wroteFallback := h.ensureForwardErrorResponseContext(transportCtx, streamStarted)
			fields := []zap.Field{
				zap.Int64("account_id", account.ID),
				zap.Bool("fallback_error_response_written", wroteFallback),
				zap.Error(err),
			}
			if shouldLogOpenAIForwardFailureAsWarnContext(transportCtx, wroteFallback) {
				reqLog.Warn("openai.forward_failed", fields...)
				return
			}
			reqLog.Error("openai.forward_failed", fields...)
			return
		}
		if result != nil {
			if account.Type == service.AccountTypeOAuth && !account.IsOpenAIChatWebMode() {
				h.gatewayService.UpdateCodexUsageSnapshotFromHeaders(transportCtx.Context(), account.ID, result.ResponseHeaders)
			}
			h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, true, result.FirstTokenMs)
		} else {
			h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, true, nil)
		}
		h.gatewayService.MarkOpenAIAccountHealthy(account)
		if err := h.gatewayService.PromoteStickySession(transportCtx.Context(), selectedAPIKey.GroupID, sessionHash, account.ID); err != nil {
			reqLog.Warn("openai.promote_sticky_session_failed", zap.Int64("account_id", account.ID), zap.Error(err))
		}

		// 捕获请求信息（用于异步记录，避免在 goroutine 中访问 gin.Context）
		userAgent := strings.TrimSpace(transportCtx.HeaderValue("User-Agent"))
		clientIP := strings.TrimSpace(transportCtx.ClientIP())
		requestPayloadHash := service.HashUsageRequestPayload(body)
		quotaPlatform := service.QuotaPlatform(transportCtx.Context(), selectedAPIKey)
		sessionID := extractClientSessionIDContext(transportCtx)

		// 使用量记录通过有界 worker 池提交，避免请求热路径创建无界 goroutine。
		h.submitUsageRecordTask(transportCtx.Context(), func(ctx context.Context) {
			if err := h.gatewayService.RecordUsage(ctx, &service.OpenAIRecordUsageInput{
				Result:             result,
				APIKey:             selectedAPIKey,
				User:               selectedAPIKey.User,
				Account:            account,
				Subscription:       subscription,
				InboundEndpoint:    GetInboundEndpointContext(transportCtx),
				UpstreamEndpoint:   resolveOpenAIUpstreamEndpointContext(transportCtx, account, result),
				UserAgent:          userAgent,
				IPAddress:          clientIP,
				RequestPayloadHash: requestPayloadHash,
				APIKeyService:      h.apiKeyService,
				QuotaPlatform:      quotaPlatform,
				SessionID:          sessionID,
				ChannelUsageFields: clientRequestedUsageFieldsContext(transportCtx, channelMapping, reqModel, result.UpstreamModel),
			}); err != nil {
				logger.L().With(
					zap.String("component", "handler.openai_gateway.responses"),
					zap.Int64("user_id", subject.UserID),
					zap.Int64("api_key_id", selectedAPIKey.ID),
					zap.Any("group_id", selectedAPIKey.GroupID),
					zap.String("model", reqModel),
					zap.Int64("account_id", account.ID),
				).Error("openai.record_usage_failed", zap.Error(err))
			}
		})
		reqLog.Debug("openai.request_completed",
			zap.Int64("account_id", account.ID),
			zap.Int("switch_count", switchCount),
		)
		return
	}
}

func isOpenAIRemoteCompactPath(c *gin.Context) bool {
	return isOpenAIRemoteCompactPathContext(gatewayctx.FromGin(c))
}

func isOpenAIRemoteCompactPathContext(c gatewayctx.GatewayContext) bool {
	if c == nil || c.Request() == nil || c.Request().URL == nil {
		return false
	}
	normalizedPath := strings.TrimRight(strings.TrimSpace(c.Request().URL.Path), "/")
	return strings.HasSuffix(normalizedPath, "/responses/compact")
}

func (h *OpenAIGatewayHandler) logOpenAIRemoteCompactOutcome(c *gin.Context, startedAt time.Time) {
	h.logOpenAIRemoteCompactOutcomeContext(gatewayctx.FromGin(c), startedAt)
}

func (h *OpenAIGatewayHandler) logOpenAIRemoteCompactOutcomeContext(c gatewayctx.GatewayContext, startedAt time.Time) {
	if !isOpenAIRemoteCompactPathContext(c) {
		return
	}

	var (
		ctx  = context.Background()
		path string
	)
	if c != nil {
		if req := c.Request(); req != nil {
			ctx = req.Context()
			if req.URL != nil {
				path = strings.TrimSpace(req.URL.Path)
			}
		}
	}
	status := responseStatusFromGatewayContext(c)
	if status <= 0 {
		status = http.StatusOK
	}

	outcome := "failed"
	if status >= 200 && status < 300 {
		outcome = "succeeded"
	}
	latencyMs := time.Since(startedAt).Milliseconds()
	if latencyMs < 0 {
		latencyMs = 0
	}

	fields := []zap.Field{
		zap.String("component", "handler.openai_gateway.responses"),
		zap.Bool("remote_compact", true),
		zap.String("compact_outcome", outcome),
		zap.Int("status_code", status),
		zap.Int64("latency_ms", latencyMs),
		zap.String("path", path),
		zap.Bool("force_codex_cli", h != nil && h.cfg != nil && h.cfg.Gateway.ForceCodexCLI),
	}

	if c != nil {
		if userAgent := strings.TrimSpace(c.HeaderValue("User-Agent")); userAgent != "" {
			fields = append(fields, zap.String("request_user_agent", userAgent))
		}
		if v, ok := c.Value(opsModelKey); ok {
			if model, ok := v.(string); ok && strings.TrimSpace(model) != "" {
				fields = append(fields, zap.String("request_model", strings.TrimSpace(model)))
			}
		}
		if v, ok := c.Value(opsAccountIDKey); ok {
			if accountID, ok := v.(int64); ok && accountID > 0 {
				fields = append(fields, zap.Int64("account_id", accountID))
			}
		}
		if upstreamRequestID := responseHeaderValue(c, "x-request-id"); upstreamRequestID != "" {
			fields = append(fields, zap.String("upstream_request_id", upstreamRequestID))
		} else if upstreamRequestID := responseHeaderValue(c, "X-Request-Id"); upstreamRequestID != "" {
			fields = append(fields, zap.String("upstream_request_id", upstreamRequestID))
		}
	}

	log := logger.FromContext(ctx).With(fields...)
	if outcome == "succeeded" {
		log.Info("codex.remote_compact.succeeded")
		return
	}
	log.Warn("codex.remote_compact.failed")
}

func responseStatusFromGatewayContext(c gatewayctx.GatewayContext) int {
	if c == nil {
		return 0
	}
	if raw, ok := c.Value(service.OpsUpstreamStatusCodeKey); ok {
		switch typed := raw.(type) {
		case int:
			return typed
		case int64:
			return int(typed)
		}
	}
	switch native := c.Native().(type) {
	case *gin.Context:
		if native != nil && native.Writer != nil {
			return native.Writer.Status()
		}
	case interface{ StatusCode() int }:
		return native.StatusCode()
	case interface{ Status() int }:
		return native.Status()
	}
	return 0
}

func responseHeaderValue(c gatewayctx.GatewayContext, name string) string {
	if c == nil {
		return ""
	}
	if header := c.Header(); header != nil {
		if value := strings.TrimSpace(header.Get(name)); value != "" {
			return value
		}
	}
	return strings.TrimSpace(c.HeaderValue(name))
}

// Messages handles Anthropic Messages API requests routed to OpenAI platform.
// POST /v1/messages (when group platform is OpenAI)
func (h *OpenAIGatewayHandler) Messages(c *gin.Context) {
	h.MessagesGateway(gatewayctx.FromGin(c))
}

func (h *OpenAIGatewayHandler) MessagesGateway(transportCtx gatewayctx.GatewayContext) {
	streamStarted := false
	defer h.recoverAnthropicMessagesPanicContext(transportCtx, &streamStarted)

	requestStart := time.Now()

	apiKey, ok := middleware2.GetAPIKeyFromGatewayContext(transportCtx)
	if !ok {
		h.anthropicErrorResponseContext(transportCtx, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}

	subject, ok := middleware2.GetAuthSubjectFromGatewayContext(transportCtx)
	if !ok {
		h.anthropicErrorResponseContext(transportCtx, http.StatusInternalServerError, "api_error", "User context not found")
		return
	}
	reqLog := requestLoggerContext(
		transportCtx,
		"handler.openai_gateway.messages",
		zap.Int64("user_id", subject.UserID),
		zap.Int64("api_key_id", apiKey.ID),
		zap.Any("group_id", apiKey.GroupID),
	)

	// 检查分组是否允许 /v1/messages 调度
	if !allowOpenAICompatibleMessagesDispatch(apiKey) {
		h.anthropicErrorResponseContext(transportCtx, http.StatusForbidden, "permission_error",
			"This group does not allow /v1/messages dispatch")
		return
	}

	if !h.ensureResponsesDependenciesGateway(transportCtx, reqLog) {
		return
	}

	body, err := pkghttputil.ReadRequestBodyWithPrealloc(transportCtx.Request())
	if err != nil {
		if maxErr, ok := extractMaxBytesError(err); ok {
			h.anthropicErrorResponseContext(transportCtx, http.StatusRequestEntityTooLarge, "invalid_request_error", buildBodyTooLargeMessage(maxErr.Limit))
			return
		}
		h.anthropicErrorResponseContext(transportCtx, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
		return
	}
	if len(body) == 0 {
		h.anthropicErrorResponseContext(transportCtx, http.StatusBadRequest, "invalid_request_error", "Request body is empty")
		return
	}

	if !gjson.ValidBytes(body) {
		h.anthropicErrorResponseContext(transportCtx, http.StatusBadRequest, "invalid_request_error", "Failed to parse request body")
		return
	}
	body, _, err = service.EnforceUnboundedTokenRequestLimit(body, "max_tokens", "max_tokens")
	if err != nil {
		h.anthropicErrorResponseContext(transportCtx, http.StatusBadRequest, "invalid_request_error", "Failed to apply output token limit")
		return
	}

	modelResult := gjson.GetBytes(body, "model")
	if !modelResult.Exists() || modelResult.Type != gjson.String || modelResult.String() == "" {
		h.anthropicErrorResponseContext(transportCtx, http.StatusBadRequest, "invalid_request_error", "model is required")
		return
	}
	reqModel := modelResult.String()
	if !apiKeyAllowsRequestedModel(apiKey, reqModel) {
		h.anthropicErrorResponseContext(transportCtx, http.StatusBadRequest, "invalid_request_error", apiKeyModelNotAllowedMessage(reqModel))
		return
	}
	ensureCompositeTargetPlatformContext(transportCtx, apiKey, reqModel)
	if !openAICompatibleTextTargetAllowedContext(transportCtx, apiKey, reqModel) {
		h.anthropicErrorResponseContext(transportCtx, http.StatusBadRequest, "invalid_request_error", "Model is not supported by this OpenAI-compatible endpoint for composite groups")
		return
	}
	reqStream := gjson.GetBytes(body, "stream").Bool()

	reqLog = reqLog.With(zap.String("model", reqModel), zap.Bool("stream", reqStream))

	setOpsRequestContextGateway(transportCtx, reqModel, reqStream, body)
	setOpsEndpointContextGateway(transportCtx, "", int16(service.RequestTypeFromLegacy(reqStream, false)))
	channelMappingMsg, _ := h.gatewayService.ResolveChannelMappingAndRestrict(transportCtx.Context(), apiKey.GroupID, reqModel)
	if decision := h.checkSecurityAuditContext(transportCtx, reqLog, apiKey, subject, service.ContentModerationProtocolAnthropicMessages, reqModel, body); decision != nil && !decision.AllowNextStage {
		h.anthropicSecurityAuditErrorContext(transportCtx, decision)
		return
	}

	// 绑定错误透传服务，允许 service 层在非 failover 错误场景复用规则。
	if h.errorPassthroughService != nil {
		service.BindErrorPassthroughServiceContext(transportCtx, h.errorPassthroughService)
	}

	subscription, _ := middleware2.GetSubscriptionFromGatewayContext(transportCtx)

	service.SetOpsLatencyMsContext(transportCtx, service.OpsAuthLatencyMsKey, time.Since(requestStart).Milliseconds())
	routingStart := time.Now()

	userReleaseFunc, acquired := h.acquireResponsesUserSlotContext(transportCtx, subject.UserID, subject.Concurrency, reqStream, &streamStarted, reqLog)
	if !acquired {
		return
	}
	if userReleaseFunc != nil {
		defer userReleaseFunc()
	}

	if !h.checkOpenAITokenBillingEligibilityContext(
		transportCtx,
		reqLog,
		apiKey,
		subscription,
		reqModel,
		body,
		streamStarted,
		"openai_messages",
		h.anthropicStreamingAwareErrorContext,
	) {
		return
	}

	sessionHash := h.resolveOpenAIStickySessionHashContext(transportCtx, apiKey, subject.UserID, reqModel, reqStream, body)
	promptCacheKey := h.gatewayService.ExtractSessionIDContext(transportCtx, body)
	sessionHash, promptCacheKey = resolveOpenAIMessagesMetadataSession(sessionHash, promptCacheKey, reqModel, body)

	failoverPolicy := h.resolveOpenAIFailoverPolicy(body, reqStream)
	maxAccountSwitches := failoverPolicy.MaxSwitches
	if maxAccountSwitches != h.maxAccountSwitches || !failoverPolicy.AllowSameAccountRetry {
		reqLog.Info("openai_messages.streaming_failover_policy_applied",
			zap.Int("body_bytes", len(body)),
			zap.Bool("stream", reqStream),
			zap.Int("effective_max_switches", maxAccountSwitches),
			zap.Bool("allow_same_account_retry", failoverPolicy.AllowSameAccountRetry),
		)
	}
	switchCount := 0
	failedAccountIDs := make(map[int64]struct{})
	sameAccountRetryCount := make(map[int64]int)
	var lastFailoverErr *service.UpstreamFailoverError

	for {
		// 清除上一次迭代的降级模型标记，避免残留影响本次迭代
		transportCtx.SetValue("openai_messages_fallback_model", "")
		reqLog.Debug("openai_messages.account_selecting", zap.Int("excluded_account_count", len(failedAccountIDs)))
		groupSelection, err := selectOpenAIAPIKeyGroup(
			transportCtx.Context(),
			apiKey,
			"",
			sessionHash,
			reqModel,
			failedAccountIDs,
			service.OpenAIUpstreamTransportAny,
			h.gatewayService.SelectAccountWithScheduler,
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
			reqLog.Warn("openai_messages.account_select_failed",
				zap.Error(err),
				zap.Int("excluded_account_count", len(failedAccountIDs)),
			)
			// 首次调度失败 + 有默认映射模型 → 用默认模型重试
			if len(failedAccountIDs) == 0 {
				defaultModel := resolveOpenAISelectionFallbackModel(apiKey, reqModel)
				if defaultModel != "" && defaultModel != reqModel {
					reqLog.Info("openai_messages.fallback_to_default_model",
						zap.String("default_mapped_model", defaultModel),
					)
					groupSelection, err = selectOpenAIAPIKeyGroup(
						transportCtx.Context(),
						apiKey,
						"",
						sessionHash,
						defaultModel,
						failedAccountIDs,
						service.OpenAIUpstreamTransportAny,
						h.gatewayService.SelectAccountWithScheduler,
					)
					if err == nil && groupSelection != nil {
						selection = groupSelection.Selection
						scheduleDecision = groupSelection.Decision
						selectedAPIKey = groupSelection.APIKey
						transportCtx.SetValue("openai_messages_fallback_model", defaultModel)
					}
				}
				if err != nil {
					h.anthropicStreamingAwareErrorContext(transportCtx, http.StatusServiceUnavailable, "no_available_account", "No available upstream account supports the requested model", streamStarted)
					return
				}
			} else {
				var action FailoverAction
				failedAccountIDs, action = handleOpenAISelectionExhausted(
					transportCtx.Context(),
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
						h.handleAnthropicFailoverExhaustedContext(transportCtx, lastFailoverErr, streamStarted)
					} else {
						h.anthropicStreamingAwareErrorContext(transportCtx, http.StatusBadGateway, "api_error", "Upstream request failed", streamStarted)
					}
					return
				}
			}
		}
		if selection == nil || selection.Account == nil {
			h.anthropicStreamingAwareErrorContext(transportCtx, http.StatusServiceUnavailable, "no_available_account", "No available upstream account supports the requested model", streamStarted)
			return
		}
		account := selection.Account
		sessionHash = ensureOpenAIPoolModeSessionHash(sessionHash, account)
		reqLog.Debug("openai_messages.account_selected", zap.Int64("account_id", account.ID), zap.String("account_name", account.Name))
		_ = scheduleDecision
		setOpsSelectedAccountGateway(transportCtx, account.ID, account.Platform)

		accountReleaseFunc, acquired := h.acquireResponsesAccountSlotContext(transportCtx, selectedAPIKey.GroupID, sessionHash, selection, reqStream, &streamStarted, reqLog)
		if !acquired {
			return
		}

		service.SetOpsLatencyMsContext(transportCtx, service.OpsRoutingLatencyMsKey, time.Since(routingStart).Milliseconds())
		forwardStart := time.Now()

		// Forward 层需要始终拿到 group 默认映射模型，这样未命中账号级映射的
		// Claude 兼容模型才不会在后续 Codex 规范化中意外退化到 gpt-5.1。
		defaultMappedModel := resolveOpenAIForwardDefaultMappedModel(selectedAPIKey, getContextStringGateway(transportCtx, "openai_messages_fallback_model"))
		forwardBody := body
		if channelMappingMsg.Mapped {
			forwardBody = h.gatewayService.ReplaceModelInBody(body, channelMappingMsg.MappedModel)
		}
		result, err := h.gatewayService.ForwardAsAnthropicContext(transportCtx.Context(), transportCtx, account, forwardBody, promptCacheKey, defaultMappedModel)

		forwardDurationMs := time.Since(forwardStart).Milliseconds()
		if accountReleaseFunc != nil {
			accountReleaseFunc()
		}
		upstreamLatencyMs, _ := getContextInt64Gateway(transportCtx, service.OpsUpstreamLatencyMsKey)
		responseLatencyMs := forwardDurationMs
		if upstreamLatencyMs > 0 && forwardDurationMs > upstreamLatencyMs {
			responseLatencyMs = forwardDurationMs - upstreamLatencyMs
		}
		service.SetOpsLatencyMsContext(transportCtx, service.OpsResponseLatencyMsKey, responseLatencyMs)
		if err == nil && result != nil && result.FirstTokenMs != nil {
			service.SetOpsLatencyMsContext(transportCtx, service.OpsTimeToFirstTokenMsKey, int64(*result.FirstTokenMs))
		}
		if err != nil {
			var failoverErr *service.UpstreamFailoverError
			if errors.As(err, &failoverErr) {
				failoverErr = adjustOpenAINetworkFailoverCooldown(failoverErr)
				failoverErr = adjustOpenAIFailoverForLargeStreamingRequest(failoverErr, body, reqStream, h.cfg)
				h.gatewayService.RegisterOpenAIRuntimeFailure(account, failoverErr)
				h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, false, nil)
				// 池模式：同账号重试
				if failoverPolicy.AllowSameAccountRetry && failoverErr.RetryableOnSameAccount {
					retryLimit := account.GetPoolModeRetryCount()
					if sameAccountRetryCount[account.ID] < retryLimit {
						sameAccountRetryCount[account.ID]++
						reqLog.Warn("openai_messages.pool_mode_same_account_retry",
							zap.Int64("account_id", account.ID),
							zap.Int("upstream_status", failoverErr.StatusCode),
							zap.Int("retry_limit", retryLimit),
							zap.Int("retry_count", sameAccountRetryCount[account.ID]),
						)
						select {
						case <-transportCtx.Context().Done():
							return
						case <-time.After(sameAccountRetryDelay):
						}
						continue
					}
				}
				_ = h.gatewayService.ClearStickySessionForAccount(transportCtx.Context(), selectedAPIKey.GroupID, sessionHash, account.ID)
				h.gatewayService.TempUnscheduleRetryableError(transportCtx.Context(), account.ID, failoverErr)
				h.gatewayService.RecordOpenAIAccountSwitch()
				failedAccountIDs[account.ID] = struct{}{}
				lastFailoverErr = failoverErr
				if switchCount >= maxAccountSwitches {
					h.handleAnthropicFailoverExhaustedContext(transportCtx, failoverErr, streamStarted)
					return
				}
				switchCount++
				reqLog.Warn("openai_messages.upstream_failover_switching",
					zap.Int64("account_id", account.ID),
					zap.Int("upstream_status", failoverErr.StatusCode),
					zap.Int("switch_count", switchCount),
					zap.Int("max_switches", maxAccountSwitches),
				)
				continue
			}
			h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, false, nil)
			wroteFallback := h.ensureAnthropicErrorResponseContext(transportCtx, streamStarted)
			reqLog.Warn("openai_messages.forward_failed",
				zap.Int64("account_id", account.ID),
				zap.Bool("fallback_error_response_written", wroteFallback),
				zap.Error(err),
			)
			return
		}
		if result != nil {
			h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, true, result.FirstTokenMs)
		} else {
			h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, true, nil)
		}
		h.gatewayService.MarkOpenAIAccountHealthy(account)
		if err := h.gatewayService.PromoteStickySession(transportCtx.Context(), selectedAPIKey.GroupID, sessionHash, account.ID); err != nil {
			reqLog.Warn("openai_messages.promote_sticky_session_failed", zap.Int64("account_id", account.ID), zap.Error(err))
		}

		userAgent := strings.TrimSpace(transportCtx.HeaderValue("User-Agent"))
		clientIP := strings.TrimSpace(transportCtx.ClientIP())
		requestPayloadHash := service.HashUsageRequestPayload(body)
		quotaPlatform := service.QuotaPlatform(transportCtx.Context(), selectedAPIKey)
		sessionID := extractClientSessionIDContext(transportCtx)

		h.submitUsageRecordTask(transportCtx.Context(), func(ctx context.Context) {
			if err := h.gatewayService.RecordUsage(ctx, &service.OpenAIRecordUsageInput{
				Result:             result,
				APIKey:             selectedAPIKey,
				User:               selectedAPIKey.User,
				Account:            account,
				Subscription:       subscription,
				InboundEndpoint:    GetInboundEndpointContext(transportCtx),
				UpstreamEndpoint:   resolveOpenAIUpstreamEndpointContext(transportCtx, account, result),
				UserAgent:          userAgent,
				IPAddress:          clientIP,
				RequestPayloadHash: requestPayloadHash,
				APIKeyService:      h.apiKeyService,
				QuotaPlatform:      quotaPlatform,
				SessionID:          sessionID,
				ChannelUsageFields: clientRequestedUsageFieldsContext(transportCtx, channelMappingMsg, reqModel, result.UpstreamModel),
			}); err != nil {
				logger.L().With(
					zap.String("component", "handler.openai_gateway.messages"),
					zap.Int64("user_id", subject.UserID),
					zap.Int64("api_key_id", selectedAPIKey.ID),
					zap.Any("group_id", selectedAPIKey.GroupID),
					zap.String("model", reqModel),
					zap.Int64("account_id", account.ID),
				).Error("openai_messages.record_usage_failed", zap.Error(err))
			}
		})
		reqLog.Debug("openai_messages.request_completed",
			zap.Int64("account_id", account.ID),
			zap.Int("switch_count", switchCount),
		)
		return
	}
}

// anthropicErrorResponse writes an error in Anthropic Messages API format.
func (h *OpenAIGatewayHandler) anthropicErrorResponse(c *gin.Context, status int, errType, message string) {
	h.anthropicErrorResponseContext(gatewayctx.FromGin(c), status, errType, message)
}

func (h *OpenAIGatewayHandler) anthropicErrorResponseContext(c gatewayctx.GatewayContext, status int, errType, message string) {
	if c == nil {
		return
	}
	c.WriteJSON(status, gin.H{
		"type": "error",
		"error": gin.H{
			"type":    errType,
			"message": message,
		},
	})
}

// anthropicStreamingAwareError handles errors that may occur during streaming,
// using Anthropic SSE error format.
func (h *OpenAIGatewayHandler) anthropicStreamingAwareError(c *gin.Context, status int, errType, message string, streamStarted bool) {
	h.anthropicStreamingAwareErrorContext(gatewayctx.FromGin(c), status, errType, message, streamStarted)
}

func (h *OpenAIGatewayHandler) anthropicStreamingAwareErrorContext(c gatewayctx.GatewayContext, status int, errType, message string, streamStarted bool) {
	if c == nil {
		return
	}
	if streamStarted || service.RequestUsesAnthropicSSE(c) {
		_ = gatewayctx.WriteSSEEvent(c, "error", gin.H{
			"type": "error",
			"error": gin.H{
				"type":    errType,
				"message": message,
			},
		})
		return
	}
	h.anthropicErrorResponseContext(c, status, errType, message)
}

// handleAnthropicFailoverExhausted maps upstream failover errors to Anthropic format.
func (h *OpenAIGatewayHandler) handleAnthropicFailoverExhausted(c *gin.Context, failoverErr *service.UpstreamFailoverError, streamStarted bool) {
	h.handleAnthropicFailoverExhaustedContext(gatewayctx.FromGin(c), failoverErr, streamStarted)
}

func (h *OpenAIGatewayHandler) handleAnthropicFailoverExhaustedContext(c gatewayctx.GatewayContext, failoverErr *service.UpstreamFailoverError, streamStarted bool) {
	status, errType, errMsg := h.mapUpstreamError(failoverErr.StatusCode)
	h.anthropicStreamingAwareErrorContext(c, status, errType, errMsg, streamStarted)
}

// ensureAnthropicErrorResponse writes a fallback Anthropic error if no response was written.
func (h *OpenAIGatewayHandler) ensureAnthropicErrorResponse(c *gin.Context, streamStarted bool) bool {
	return h.ensureAnthropicErrorResponseContext(gatewayctx.FromGin(c), streamStarted)
}

func (h *OpenAIGatewayHandler) ensureAnthropicErrorResponseContext(c gatewayctx.GatewayContext, streamStarted bool) bool {
	if c == nil || service.RequestPayloadStarted(c) {
		return false
	}
	if c.ResponseWritten() && !streamStarted && !service.RequestUsesAnthropicSSE(c) && !service.RequestUsesBufferedJSON(c) {
		return false
	}
	h.anthropicStreamingAwareErrorContext(c, http.StatusBadGateway, "api_error", "Upstream request failed", streamStarted)
	return true
}

func (h *OpenAIGatewayHandler) validateFunctionCallOutputRequest(c *gin.Context, body []byte, reqLog *zap.Logger) bool {
	return h.validateFunctionCallOutputRequestContext(gatewayctx.FromGin(c), body, reqLog)
}

func (h *OpenAIGatewayHandler) validateFunctionCallOutputRequestContext(c gatewayctx.GatewayContext, body []byte, reqLog *zap.Logger) bool {
	if !gjson.GetBytes(body, `input.#(type=="function_call_output")`).Exists() {
		return true
	}

	var reqBody map[string]any
	if err := json.Unmarshal(body, &reqBody); err != nil {
		// 保持原有容错语义：解析失败时跳过预校验，沿用后续上游校验结果。
		return true
	}

	c.SetValue(service.OpenAIParsedRequestBodyKey, reqBody)
	validation := service.ValidateFunctionCallOutputContext(reqBody)
	if !validation.HasFunctionCallOutput {
		return true
	}

	previousResponseID, _ := reqBody["previous_response_id"].(string)
	if strings.TrimSpace(previousResponseID) != "" || validation.HasToolCallContext {
		return true
	}

	if validation.HasFunctionCallOutputMissingCallID {
		reqLog.Warn("openai.request_validation_failed",
			zap.String("reason", "function_call_output_missing_call_id"),
		)
		h.errorResponseGateway(c, http.StatusBadRequest, "invalid_request_error", "function_call_output requires call_id on HTTP requests; continuation via previous_response_id is only supported on Responses WebSocket v2")
		return false
	}
	if validation.HasItemReferenceForAllCallIDs {
		return true
	}

	reqLog.Warn("openai.request_validation_failed",
		zap.String("reason", "function_call_output_missing_item_reference"),
	)
	h.errorResponseGateway(c, http.StatusBadRequest, "invalid_request_error", "function_call_output requires item_reference ids matching each call_id on HTTP requests; continuation via previous_response_id is only supported on Responses WebSocket v2")
	return false
}

func (h *OpenAIGatewayHandler) acquireResponsesUserSlot(
	c *gin.Context,
	userID int64,
	userConcurrency int,
	reqStream bool,
	streamStarted *bool,
	reqLog *zap.Logger,
) (func(), bool) {
	return h.acquireResponsesUserSlotContext(gatewayctx.FromGin(c), userID, userConcurrency, reqStream, streamStarted, reqLog)
}

func (h *OpenAIGatewayHandler) acquireResponsesUserSlotContext(
	c gatewayctx.GatewayContext,
	userID int64,
	userConcurrency int,
	reqStream bool,
	streamStarted *bool,
	reqLog *zap.Logger,
) (func(), bool) {
	ctx := c.Context()
	userReleaseFunc, userAcquired, err := h.concurrencyHelper.TryAcquireUserSlot(ctx, userID, userConcurrency)
	if err != nil {
		reqLog.Warn("openai.user_slot_acquire_failed", zap.Error(err))
		h.handleConcurrencyErrorContext(c, err, "user", *streamStarted)
		return nil, false
	}
	if userAcquired {
		return wrapReleaseOnDone(ctx, userReleaseFunc), true
	}

	maxWait := service.CalculateMaxWait(userConcurrency)
	canWait, waitErr := h.concurrencyHelper.IncrementWaitCount(ctx, userID, maxWait)
	if waitErr != nil {
		reqLog.Warn("openai.user_wait_counter_increment_failed", zap.Error(waitErr))
		// 按现有降级语义：等待计数异常时放行后续抢槽流程
	} else if !canWait {
		reqLog.Info("openai.user_wait_queue_full", zap.Int("max_wait", maxWait))
		h.errorResponseGateway(c, http.StatusTooManyRequests, "rate_limit_error", "Too many pending requests, please retry later")
		return nil, false
	}

	waitCounted := waitErr == nil && canWait
	defer func() {
		if waitCounted {
			h.concurrencyHelper.DecrementWaitCount(ctx, userID)
		}
	}()

	userReleaseFunc, err = h.concurrencyHelper.AcquireUserSlotWithWaitContext(c, userID, userConcurrency, reqStream, streamStarted)
	if err != nil {
		reqLog.Warn("openai.user_slot_acquire_failed_after_wait", zap.Error(err))
		h.handleConcurrencyErrorContext(c, err, "user", *streamStarted)
		return nil, false
	}

	// 槽位获取成功后，立刻退出等待计数。
	if waitCounted {
		h.concurrencyHelper.DecrementWaitCount(ctx, userID)
		waitCounted = false
	}
	return wrapReleaseOnDone(ctx, userReleaseFunc), true
}

func (h *OpenAIGatewayHandler) acquireResponsesAccountSlot(
	c *gin.Context,
	groupID *int64,
	sessionHash string,
	selection *service.AccountSelectionResult,
	reqStream bool,
	streamStarted *bool,
	reqLog *zap.Logger,
) (func(), bool) {
	return h.acquireResponsesAccountSlotContext(gatewayctx.FromGin(c), groupID, sessionHash, selection, reqStream, streamStarted, reqLog)
}

func (h *OpenAIGatewayHandler) acquireResponsesAccountSlotContext(
	c gatewayctx.GatewayContext,
	groupID *int64,
	sessionHash string,
	selection *service.AccountSelectionResult,
	reqStream bool,
	streamStarted *bool,
	reqLog *zap.Logger,
) (func(), bool) {
	if selection == nil || selection.Account == nil {
		h.handleStreamingAwareErrorContext(c, http.StatusServiceUnavailable, "api_error", "No available accounts", *streamStarted)
		return nil, false
	}

	ctx := c.Context()
	account := selection.Account
	if selection.Acquired {
		return wrapReleaseOnDone(ctx, selection.ReleaseFunc), true
	}
	if selection.WaitPlan == nil {
		h.handleStreamingAwareErrorContext(c, http.StatusServiceUnavailable, "api_error", "No available accounts", *streamStarted)
		return nil, false
	}

	fastReleaseFunc, fastAcquired, err := h.concurrencyHelper.TryAcquireAccountSlot(
		ctx,
		account.ID,
		selection.WaitPlan.MaxConcurrency,
	)
	if err != nil {
		reqLog.Warn("openai.account_slot_quick_acquire_failed", zap.Int64("account_id", account.ID), zap.Error(err))
		h.handleConcurrencyErrorContext(c, err, "account", *streamStarted)
		return nil, false
	}
	if fastAcquired {
		if err := h.gatewayService.BindStickySession(ctx, groupID, sessionHash, account.ID); err != nil {
			reqLog.Warn("openai.bind_sticky_session_failed", zap.Int64("account_id", account.ID), zap.Error(err))
		}
		return wrapReleaseOnDone(ctx, fastReleaseFunc), true
	}

	canWait, waitErr := h.concurrencyHelper.IncrementAccountWaitCount(ctx, account.ID, selection.WaitPlan.MaxWaiting)
	if waitErr != nil {
		reqLog.Warn("openai.account_wait_counter_increment_failed", zap.Int64("account_id", account.ID), zap.Error(waitErr))
	} else if !canWait {
		reqLog.Info("openai.account_wait_queue_full",
			zap.Int64("account_id", account.ID),
			zap.Int("max_waiting", selection.WaitPlan.MaxWaiting),
		)
		h.handleStreamingAwareErrorContext(c, http.StatusTooManyRequests, "rate_limit_error", "Too many pending requests, please retry later", *streamStarted)
		return nil, false
	}

	accountWaitCounted := waitErr == nil && canWait
	releaseWait := func() {
		if accountWaitCounted {
			h.concurrencyHelper.DecrementAccountWaitCount(ctx, account.ID)
			accountWaitCounted = false
		}
	}
	defer releaseWait()

	accountReleaseFunc, err := h.concurrencyHelper.AcquireAccountSlotWithWaitTimeoutContext(
		c,
		account.ID,
		selection.WaitPlan.MaxConcurrency,
		selection.WaitPlan.Timeout,
		reqStream,
		streamStarted,
	)
	if err != nil {
		reqLog.Warn("openai.account_slot_acquire_failed", zap.Int64("account_id", account.ID), zap.Error(err))
		h.handleConcurrencyErrorContext(c, err, "account", *streamStarted)
		return nil, false
	}

	// Slot acquired: no longer waiting in queue.
	releaseWait()
	if err := h.gatewayService.BindStickySession(ctx, groupID, sessionHash, account.ID); err != nil {
		reqLog.Warn("openai.bind_sticky_session_failed", zap.Int64("account_id", account.ID), zap.Error(err))
	}
	return wrapReleaseOnDone(ctx, accountReleaseFunc), true
}

// ResponsesWebSocket handles OpenAI Responses API WebSocket ingress endpoint
// GET /openai/v1/responses (Upgrade: websocket)
func (h *OpenAIGatewayHandler) ResponsesWebSocket(c *gin.Context) {
	h.ResponsesWebSocketGateway(gatewayctx.FromGin(c))
}

func (h *OpenAIGatewayHandler) ResponsesWebSocketGateway(transportCtx gatewayctx.GatewayContext) {
	if transportCtx == nil {
		return
	}

	if !isOpenAIWSUpgradeRequest(transportCtx.Request()) {
		h.errorResponseGateway(transportCtx, http.StatusUpgradeRequired, "invalid_request_error", "WebSocket upgrade required (Upgrade: websocket)")
		return
	}
	setOpenAIClientTransportWSGateway(transportCtx)

	apiKey, ok := middleware2.GetAPIKeyFromGatewayContext(transportCtx)
	if !ok {
		h.errorResponseGateway(transportCtx, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}
	subject, ok := middleware2.GetAuthSubjectFromGatewayContext(transportCtx)
	if !ok {
		h.errorResponseGateway(transportCtx, http.StatusInternalServerError, "api_error", "User context not found")
		return
	}

	reqLog := requestLoggerContext(
		transportCtx,
		"handler.openai_gateway.responses_ws",
		zap.Int64("user_id", subject.UserID),
		zap.Int64("api_key_id", apiKey.ID),
		zap.Any("group_id", apiKey.GroupID),
		zap.Bool("openai_ws_mode", true),
	)
	if !h.ensureResponsesDependenciesGateway(transportCtx, reqLog) {
		return
	}
	reqLog.Info("openai.websocket_ingress_started")
	clientIP := strings.TrimSpace(transportCtx.ClientIP())
	userAgent := strings.TrimSpace(transportCtx.HeaderValue("User-Agent"))
	ctx := transportCtx.Context()
	maxIngressConnections := 0
	if h.cfg != nil {
		maxIngressConnections = h.cfg.Gateway.OpenAIWS.MaxIngressConnectionsPerAPIKey
	}
	ingressLease, ingressLeaseAcquired, ingressLeaseErr := h.concurrencyHelper.AcquireOpenAIWSIngressLease(ctx, apiKey.ID, maxIngressConnections)
	if ingressLeaseErr != nil {
		reqLog.Error("openai.websocket_ingress_lease_acquire_failed", zap.Error(ingressLeaseErr))
		h.errorResponseGateway(transportCtx, http.StatusServiceUnavailable, "service_unavailable", "WebSocket ingress capacity is temporarily unavailable")
		return
	}
	if !ingressLeaseAcquired {
		reqLog.Info("openai.websocket_ingress_capacity_rejected", zap.Int("max_ingress_connections_per_api_key", maxIngressConnections))
		transportCtx.SetHeader("Retry-After", "5")
		h.errorResponseGateway(transportCtx, http.StatusTooManyRequests, "rate_limit_error", "Too many open WebSocket connections, please retry later")
		return
	}
	if ingressLease != nil {
		defer ingressLease.Release()
		ctx = ingressLease.Context()
		transportCtx.SetRequest(transportCtx.Request().WithContext(ctx))
	}

	if handled := h.tryProxyResponsesWebSocketViaRustSidecar(transportCtx, reqLog); handled {
		return
	}

	acceptedConn, err := transportCtx.AcceptWebSocket(gatewayctx.WebSocketAcceptOptions{
		CompressionEnabled: true,
	})
	if err != nil {
		reqLog.Warn("openai.websocket_accept_failed",
			zap.Error(err),
			zap.String("client_ip", clientIP),
			zap.String("request_user_agent", userAgent),
			zap.String("upgrade_header", strings.TrimSpace(transportCtx.HeaderValue("Upgrade"))),
			zap.String("connection_header", strings.TrimSpace(transportCtx.HeaderValue("Connection"))),
			zap.String("sec_websocket_version", strings.TrimSpace(transportCtx.HeaderValue("Sec-WebSocket-Version"))),
			zap.Bool("has_sec_websocket_key", strings.TrimSpace(transportCtx.HeaderValue("Sec-WebSocket-Key")) != ""),
		)
		return
	}
	wsConn, ok := acceptedConn.Native().(*coderws.Conn)
	if !ok || wsConn == nil {
		reqLog.Warn("openai.websocket_accept_failed", zap.String("reason", "gatewayctx returned unexpected websocket implementation"))
		return
	}
	defer func() {
		_ = wsConn.CloseNow()
	}()
	wsConn.SetReadLimit(service.ResolveOpenAIWSClientReadLimitBytes(h.cfg))

	firstMessageTimeout := service.ResolveOpenAIWSClientFirstMessageTimeout(h.cfg)
	msgType, firstMessage, err := service.ReadOpenAIWSClientMessage(
		ctx,
		wsConn,
		firstMessageTimeout,
		coderws.StatusPolicyViolation,
		"missing first response.create message",
	)
	if err != nil {
		if errors.Is(context.Cause(ctx), service.ErrOpenAIWSIngressLeaseLost) {
			reqLog.Warn("openai.websocket_ingress_lease_lost_before_first_message", zap.Error(err))
			closeOpenAIClientWS(wsConn, coderws.StatusTryAgainLater, "websocket ingress capacity lease lost; please reconnect")
			return
		}
		closeStatus, closeReason := summarizeWSCloseErrorForLog(err)
		reqLog.Warn("openai.websocket_read_first_message_failed",
			zap.Error(err),
			zap.String("client_ip", clientIP),
			zap.String("close_status", closeStatus),
			zap.String("close_reason", closeReason),
			zap.Duration("read_timeout", firstMessageTimeout),
		)
		closeOpenAIClientWS(wsConn, coderws.StatusPolicyViolation, "missing first response.create message")
		return
	}
	if msgType != coderws.MessageText && msgType != coderws.MessageBinary {
		closeOpenAIClientWS(wsConn, coderws.StatusPolicyViolation, "unsupported websocket message type")
		return
	}
	if !gjson.ValidBytes(firstMessage) {
		closeOpenAIClientWS(wsConn, coderws.StatusPolicyViolation, "invalid JSON payload")
		return
	}
	firstMessage, _, err = service.EnforceUnboundedTokenRequestLimit(firstMessage, "max_output_tokens", "max_output_tokens")
	if err != nil {
		closeOpenAIClientWS(wsConn, coderws.StatusPolicyViolation, "invalid response.create payload")
		return
	}

	reqModel := strings.TrimSpace(gjson.GetBytes(firstMessage, "model").String())
	if reqModel == "" {
		closeOpenAIClientWS(wsConn, coderws.StatusPolicyViolation, "model is required in first response.create payload")
		return
	}
	canonicalModel, catalogEnforced, catalogAvailable := canonicalCustomerGatewayModelForAPIKey(apiKey, reqModel)
	if catalogEnforced && !catalogAvailable {
		closeOpenAIClientWS(wsConn, coderws.StatusPolicyViolation, customerGatewayModelUnavailableMessage(reqModel))
		return
	}
	if catalogEnforced && canonicalModel != reqModel {
		firstMessage, err = sjson.SetBytes(firstMessage, "model", canonicalModel)
		if err != nil {
			closeOpenAIClientWS(wsConn, coderws.StatusPolicyViolation, "failed to normalize model")
			return
		}
		reqModel = canonicalModel
	}
	if !apiKeyAllowsRequestedModel(apiKey, reqModel) {
		closeOpenAIClientWS(wsConn, coderws.StatusPolicyViolation, apiKeyModelNotAllowedMessage(reqModel))
		return
	}
	ensureCompositeTargetPlatformContext(transportCtx, apiKey, reqModel)
	ctx = transportCtx.Context()
	if apiKey.Group != nil && apiKey.Group.Platform == service.PlatformComposite {
		platform, ok := service.ResolvedTargetPlatformFromContext(ctx)
		if !ok || (platform != service.PlatformOpenAI && platform != service.PlatformGrok) {
			closeOpenAIClientWS(wsConn, coderws.StatusPolicyViolation, "Responses WebSocket API only supports OpenAI-compatible models for composite groups")
			return
		}
	}
	previousResponseID := strings.TrimSpace(gjson.GetBytes(firstMessage, "previous_response_id").String())
	previousResponseIDKind := service.ClassifyOpenAIPreviousResponseIDKind(previousResponseID)
	if previousResponseID != "" && previousResponseIDKind == service.OpenAIPreviousResponseIDKindMessageID {
		closeOpenAIClientWS(wsConn, coderws.StatusPolicyViolation, "previous_response_id must be a response.id (resp_*), not a message id")
		return
	}
	reqLog = reqLog.With(
		zap.Bool("ws_ingress", true),
		zap.String("model", reqModel),
		zap.Bool("has_previous_response_id", previousResponseID != ""),
		zap.String("previous_response_id_kind", previousResponseIDKind),
	)
	setOpsRequestContextGateway(transportCtx, reqModel, true, firstMessage)
	setOpsEndpointContextGateway(transportCtx, "", int16(service.RequestTypeWSV2))
	channelMappingWS, _ := h.gatewayService.ResolveChannelMappingAndRestrict(ctx, apiKey.GroupID, reqModel)
	if decision := h.checkSecurityAuditStageContext(transportCtx, reqLog, apiKey, subject, service.ContentModerationProtocolOpenAIResponses, reqModel, firstMessage, "first_turn"); decision != nil && !decision.AllowNextStage {
		closeOpenAIClientWS(wsConn, coderws.StatusPolicyViolation, securityAuditMessage(decision))
		return
	}

	var currentUserRelease func()
	var currentAccountRelease func()
	releaseAccountSlot := func() {
		if currentAccountRelease != nil {
			currentAccountRelease()
			currentAccountRelease = nil
		}
	}
	releaseTurnSlots := func() {
		releaseAccountSlot()
		if currentUserRelease != nil {
			currentUserRelease()
			currentUserRelease = nil
		}
	}
	// 必须尽早注册，确保任何 early return 都能释放已获取的并发槽位。
	defer releaseTurnSlots()

	userReleaseFunc, userAcquired, err := h.concurrencyHelper.TryAcquireUserSlotForAPIKey(ctx, subject.UserID, subject.Concurrency, apiKey.ID)
	if err != nil {
		reqLog.Warn("openai.websocket_user_slot_acquire_failed", zap.Error(err))
		closeOpenAIClientWS(wsConn, coderws.StatusInternalError, "failed to acquire user concurrency slot")
		return
	}
	if !userAcquired {
		closeOpenAIClientWS(wsConn, coderws.StatusTryAgainLater, "too many concurrent requests, please retry later")
		return
	}
	currentUserRelease = wrapReleaseOnDone(ctx, userReleaseFunc)
	ensureUserSlotHeld := func() bool {
		if currentUserRelease != nil {
			return true
		}
		release, acquired, acquireErr := h.concurrencyHelper.TryAcquireUserSlotForAPIKey(ctx, subject.UserID, subject.Concurrency, apiKey.ID)
		if acquireErr != nil {
			reqLog.Warn("openai.websocket_user_slot_reacquire_failed", zap.Error(acquireErr))
			closeOpenAIClientWS(wsConn, coderws.StatusInternalError, "failed to acquire user concurrency slot")
			return false
		}
		if !acquired {
			closeOpenAIClientWS(wsConn, coderws.StatusTryAgainLater, "too many concurrent requests, please retry later")
			return false
		}
		currentUserRelease = wrapReleaseOnDone(ctx, release)
		return true
	}

	subscription, _ := middleware2.GetSubscriptionFromGatewayContext(transportCtx)
	checkWebSocketTokenBilling := func(payload []byte, model string) error {
		reason := "billing check failed"
		if h.checkOpenAITokenBillingEligibilityContext(
			transportCtx,
			reqLog,
			apiKey,
			subscription,
			model,
			payload,
			false,
			"openai.websocket",
			func(_ gatewayctx.GatewayContext, _ int, _ string, message string, _ bool) {
				message = strings.TrimSpace(message)
				if message != "" {
					reason = "billing check failed: " + message
				}
			},
		) {
			return nil
		}
		return service.NewOpenAIWSClientCloseError(coderws.StatusPolicyViolation, reason, nil)
	}
	if err := checkWebSocketTokenBilling(firstMessage, reqModel); err != nil {
		var closeErr *service.OpenAIWSClientCloseError
		if errors.As(err, &closeErr) {
			closeOpenAIClientWS(wsConn, closeErr.StatusCode(), closeErr.Reason())
			return
		}
		closeOpenAIClientWS(wsConn, coderws.StatusPolicyViolation, "billing check failed")
		return
	}

	sessionHash := h.gatewayService.GenerateSessionHashWithFallbackContext(
		transportCtx,
		firstMessage,
		openAIWSIngressFallbackSessionSeed(subject.UserID, apiKey.ID, apiKey.GroupID),
	)
	maxAccountSwitches := h.maxAccountSwitches
	switchCount := 0
	failedAccountIDs := make(map[int64]struct{})
	var lastFailoverErr *service.UpstreamFailoverError
	var oauth429FailoverState service.OpenAIOAuth429FailoverState
	prepareWebSocketFailover := func(account *service.Account, selectedAPIKey *service.APIKey, failoverErr *service.UpstreamFailoverError) bool {
		if account == nil || selectedAPIKey == nil || failoverErr == nil || ctx.Err() != nil {
			return false
		}
		h.gatewayService.RegisterOpenAIRuntimeFailure(account, failoverErr)
		if failoverErr.ShouldReportAccountScheduleFailure() {
			h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, false, nil)
		}
		releaseAccountSlot()
		if !failoverErr.ShouldRetryNextAccount() {
			closeOpenAIWSFailoverExhausted(wsConn, failoverErr)
			return false
		}
		_ = h.gatewayService.ClearStickySessionForAccount(ctx, selectedAPIKey.GroupID, sessionHash, account.ID)
		h.gatewayService.TempUnscheduleRetryableError(ctx, account.ID, failoverErr)
		h.gatewayService.RecordOpenAIAccountSwitch()
		failedAccountIDs[account.ID] = struct{}{}
		lastFailoverErr = failoverErr
		if switchCount >= maxAccountSwitches {
			closeOpenAIWSFailoverExhausted(wsConn, failoverErr)
			return false
		}
		switchCount++
		if h.gatewayService.ShouldStopOpenAIOAuth429Failover(account, failoverErr.StatusCode, switchCount, &oauth429FailoverState) {
			closeOpenAIWSFailoverExhausted(wsConn, failoverErr)
			return false
		}
		reqLog.Warn("openai.websocket_upstream_failover_switching",
			zap.Int64("account_id", account.ID),
			zap.Int("upstream_status", failoverErr.StatusCode),
			zap.Int("switch_count", switchCount),
			zap.Int("max_switches", maxAccountSwitches),
		)
		return ensureUserSlotHeld()
	}

retryWebSocketAccount:
	groupSelection, err := selectOpenAIAPIKeyGroup(
		ctx,
		apiKey,
		previousResponseID,
		sessionHash,
		reqModel,
		failedAccountIDs,
		service.OpenAIUpstreamTransportResponsesWebsocketV2,
		h.gatewayService.SelectAccountWithScheduler,
	)
	if err != nil {
		reqLog.Warn("openai.websocket_account_select_failed", zap.Error(err), zap.Int("excluded_account_count", len(failedAccountIDs)))
		if lastFailoverErr != nil {
			closeOpenAIWSFailoverExhausted(wsConn, lastFailoverErr)
		} else {
			closeOpenAIClientWS(wsConn, coderws.StatusTryAgainLater, "no available account")
		}
		return
	}
	selection := groupSelection.Selection
	scheduleDecision := groupSelection.Decision
	selectedAPIKey := groupSelection.APIKey
	if selection == nil || selection.Account == nil {
		if lastFailoverErr != nil {
			closeOpenAIWSFailoverExhausted(wsConn, lastFailoverErr)
		} else {
			closeOpenAIClientWS(wsConn, coderws.StatusTryAgainLater, "no available account")
		}
		return
	}

	account := selection.Account
	accountMaxConcurrency := account.Concurrency
	if selection.WaitPlan != nil && selection.WaitPlan.MaxConcurrency > 0 {
		accountMaxConcurrency = selection.WaitPlan.MaxConcurrency
	}
	accountReleaseFunc := selection.ReleaseFunc
	if !selection.Acquired {
		if selection.WaitPlan == nil {
			closeOpenAIClientWS(wsConn, coderws.StatusTryAgainLater, "account is busy, please retry later")
			return
		}
		fastReleaseFunc, fastAcquired, err := h.concurrencyHelper.TryAcquireAccountSlot(
			ctx,
			account.ID,
			selection.WaitPlan.MaxConcurrency,
		)
		if err != nil {
			reqLog.Warn("openai.websocket_account_slot_acquire_failed", zap.Int64("account_id", account.ID), zap.Error(err))
			closeOpenAIClientWS(wsConn, coderws.StatusInternalError, "failed to acquire account concurrency slot")
			return
		}
		if !fastAcquired {
			closeOpenAIClientWS(wsConn, coderws.StatusTryAgainLater, "account is busy, please retry later")
			return
		}
		accountReleaseFunc = fastReleaseFunc
	}
	currentAccountRelease = wrapReleaseOnDone(ctx, accountReleaseFunc)
	if err := h.gatewayService.BindStickySession(ctx, selectedAPIKey.GroupID, sessionHash, account.ID); err != nil {
		reqLog.Warn("openai.websocket_bind_sticky_session_failed", zap.Int64("account_id", account.ID), zap.Error(err))
	}

	token, _, err := h.gatewayService.GetAccessToken(ctx, account)
	if err != nil {
		reqLog.Warn("openai.websocket_get_access_token_failed", zap.Int64("account_id", account.ID), zap.Error(err))
		var failoverErr *service.UpstreamFailoverError
		if errors.As(err, &failoverErr) {
			if prepareWebSocketFailover(account, selectedAPIKey, failoverErr) {
				goto retryWebSocketAccount
			}
			return
		}
		closeOpenAIClientWS(wsConn, coderws.StatusInternalError, "failed to get access token")
		return
	}

	reqLog.Debug("openai.websocket_account_selected",
		zap.Int64("account_id", account.ID),
		zap.String("account_name", account.Name),
		zap.String("schedule_layer", scheduleDecision.Layer),
		zap.Int("candidate_count", scheduleDecision.CandidateCount),
	)

	quotaPlatform := service.QuotaPlatform(transportCtx.Context(), selectedAPIKey)
	sessionID := extractClientSessionIDContext(transportCtx)
	hooks := &service.OpenAIWSIngressHooks{
		InitialRequestModel: reqModel,
		BeforeTurn: func(turn int) error {
			if turn == 1 {
				return nil
			}
			// 防御式清理：避免异常路径下旧槽位覆盖导致泄漏。
			releaseTurnSlots()
			// 非首轮 turn 需要重新抢占并发槽位，避免长连接空闲占槽。
			userReleaseFunc, userAcquired, err := h.concurrencyHelper.TryAcquireUserSlotForAPIKey(ctx, subject.UserID, subject.Concurrency, apiKey.ID)
			if err != nil {
				return service.NewOpenAIWSClientCloseError(coderws.StatusInternalError, "failed to acquire user concurrency slot", err)
			}
			if !userAcquired {
				return service.NewOpenAIWSClientCloseError(coderws.StatusTryAgainLater, "too many concurrent requests, please retry later", nil)
			}
			accountReleaseFunc, accountAcquired, err := h.concurrencyHelper.TryAcquireAccountSlot(ctx, account.ID, accountMaxConcurrency)
			if err != nil {
				if userReleaseFunc != nil {
					userReleaseFunc()
				}
				return service.NewOpenAIWSClientCloseError(coderws.StatusInternalError, "failed to acquire account concurrency slot", err)
			}
			if !accountAcquired {
				if userReleaseFunc != nil {
					userReleaseFunc()
				}
				return service.NewOpenAIWSClientCloseError(coderws.StatusTryAgainLater, "account is busy, please retry later", nil)
			}
			currentUserRelease = wrapReleaseOnDone(ctx, userReleaseFunc)
			currentAccountRelease = wrapReleaseOnDone(ctx, accountReleaseFunc)
			return nil
		},
		BeforeTurnPayload: func(turn int, payload []byte, model string) error {
			if turn == 1 {
				return nil
			}
			if decision := h.checkSecurityAuditStageContext(transportCtx, reqLog, apiKey, subject, service.ContentModerationProtocolOpenAIResponses, model, payload, "subsequent_turn"); decision != nil && !decision.AllowNextStage {
				return service.NewOpenAIWSClientCloseError(coderws.StatusPolicyViolation, securityAuditMessage(decision), nil)
			}
			return checkWebSocketTokenBilling(payload, model)
		},
		AfterTurn: func(turn int, result *service.OpenAIForwardResult, turnErr error) {
			releaseTurnSlots()
			if turnErr != nil || result == nil {
				return
			}
			if account.Type == service.AccountTypeOAuth && !account.IsOpenAIChatWebMode() {
				h.gatewayService.UpdateCodexUsageSnapshotFromHeaders(ctx, account.ID, result.ResponseHeaders)
			}
			h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, true, result.FirstTokenMs)
			h.submitUsageRecordTask(transportCtx.Context(), func(taskCtx context.Context) {
				if err := h.gatewayService.RecordUsage(taskCtx, &service.OpenAIRecordUsageInput{
					Result:             result,
					APIKey:             selectedAPIKey,
					User:               selectedAPIKey.User,
					Account:            account,
					Subscription:       subscription,
					InboundEndpoint:    GetInboundEndpointContext(transportCtx),
					UpstreamEndpoint:   resolveOpenAIUpstreamEndpointContext(transportCtx, account, result),
					UserAgent:          userAgent,
					IPAddress:          clientIP,
					RequestPayloadHash: service.HashUsageRequestPayload(firstMessage),
					APIKeyService:      h.apiKeyService,
					QuotaPlatform:      quotaPlatform,
					SessionID:          sessionID,
					ChannelUsageFields: clientRequestedUsageFieldsContext(transportCtx, channelMappingWS, reqModel, result.UpstreamModel),
				}); err != nil {
					reqLog.Error("openai.websocket_record_usage_failed",
						zap.Int64("account_id", account.ID),
						zap.String("request_id", result.RequestID),
						zap.Error(err),
					)
				}
			})
		},
	}

	ginCtx, ok := transportCtx.Native().(*gin.Context)
	if !ok || ginCtx == nil {
		closeOpenAIClientWS(wsConn, coderws.StatusInternalError, "unsupported websocket gateway transport")
		return
	}
	wsFirstMessage := firstMessage
	if channelMappingWS.Mapped {
		wsFirstMessage = h.gatewayService.ReplaceModelInBody(firstMessage, channelMappingWS.MappedModel)
	}
	if err := h.gatewayService.ProxyResponsesWebSocketFromClient(ctx, ginCtx, wsConn, account, token, wsFirstMessage, hooks); err != nil {
		var failoverErr *service.UpstreamFailoverError
		if errors.As(err, &failoverErr) {
			if prepareWebSocketFailover(account, selectedAPIKey, failoverErr) {
				goto retryWebSocketAccount
			}
			return
		}
		if errors.Is(context.Cause(ctx), service.ErrOpenAIWSIngressLeaseLost) {
			reqLog.Warn("openai.websocket_ingress_lease_lost", zap.Int64("account_id", account.ID), zap.Error(err))
			closeOpenAIClientWS(wsConn, coderws.StatusTryAgainLater, "websocket ingress capacity lease lost; please reconnect")
			return
		}
		var closeErr *service.OpenAIWSClientCloseError
		if errors.As(err, &closeErr) && closeErr.StatusCode() == coderws.StatusNormalClosure {
			reqLog.Info("openai.websocket_ingress_closed_normally", zap.Int64("account_id", account.ID), zap.String("reason", closeErr.Reason()))
			closeOpenAIClientWS(wsConn, closeErr.StatusCode(), closeErr.Reason())
			return
		}
		h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, false, nil)
		closeStatus, closeReason := summarizeWSCloseErrorForLog(err)
		reqLog.Warn("openai.websocket_proxy_failed",
			zap.Int64("account_id", account.ID),
			zap.Error(err),
			zap.String("close_status", closeStatus),
			zap.String("close_reason", closeReason),
		)
		if errors.As(err, &closeErr) {
			closeOpenAIClientWS(wsConn, closeErr.StatusCode(), closeErr.Reason())
			return
		}
		closeOpenAIClientWS(wsConn, coderws.StatusInternalError, "upstream websocket proxy failed")
		return
	}
	reqLog.Info("openai.websocket_ingress_closed", zap.Int64("account_id", account.ID))
}

func (h *OpenAIGatewayHandler) shouldUseRustSidecarResponsesWS(transportCtx gatewayctx.GatewayContext) bool {
	if h == nil || h.cfg == nil || transportCtx == nil {
		return false
	}
	if !h.cfg.Rust.Sidecar.Enabled || !h.cfg.Rust.Sidecar.ResponsesWSEnabled {
		return false
	}
	return !rustsidecar.HasBypassHeader(transportCtx.Request())
}

func (h *OpenAIGatewayHandler) tryProxyResponsesWebSocketViaRustSidecar(transportCtx gatewayctx.GatewayContext, reqLog *zap.Logger) bool {
	if !h.shouldUseRustSidecarResponsesWS(transportCtx) {
		return false
	}

	socketPath := strings.TrimSpace(h.cfg.Rust.Sidecar.SocketPath)
	var hijacker http.Hijacker
	switch native := transportCtx.Native().(type) {
	case *gin.Context:
		hj, ok := native.Writer.(http.Hijacker)
		if ok {
			hijacker = hj
		}
	case http.Hijacker:
		hijacker = native
	}
	if hijacker == nil {
		if reqLog != nil {
			reqLog.Warn("openai.websocket_rust_sidecar_unavailable", zap.String("reason", "no_hijacker"))
		}
		if h.cfg.Rust.Sidecar.FailClosed {
			return true
		}
		return false
	}

	if err := rustsidecar.TunnelUpgradedRequest(transportCtx.Request(), hijacker, socketPath); err != nil {
		if reqLog != nil {
			reqLog.Warn("openai.websocket_rust_sidecar_proxy_failed", zap.Error(err))
		}
		if h.cfg.Rust.Sidecar.FailClosed {
			return true
		}
		return false
	}
	if reqLog != nil {
		reqLog.Info("openai.websocket_routed_via_rust_sidecar")
	}
	return true
}

func (h *OpenAIGatewayHandler) recoverResponsesPanic(c *gin.Context, streamStarted *bool) {
	h.handleRecoveredResponsesPanicContext(gatewayctx.FromGin(c), streamStarted, recover())
}

func (h *OpenAIGatewayHandler) recoverResponsesPanicContext(c gatewayctx.GatewayContext, streamStarted *bool) {
	h.handleRecoveredResponsesPanicContext(c, streamStarted, recover())
}

func (h *OpenAIGatewayHandler) handleRecoveredResponsesPanicContext(c gatewayctx.GatewayContext, streamStarted *bool, recovered any) {
	if recovered == nil {
		return
	}

	started := false
	if streamStarted != nil {
		started = *streamStarted
	}
	wroteFallback := h.ensureForwardErrorResponseContext(c, started)
	requestLoggerContext(c, "handler.openai_gateway.responses").Error(
		"openai.responses_panic_recovered",
		zap.Bool("fallback_error_response_written", wroteFallback),
		zap.Any("panic", recovered),
		zap.ByteString("stack", debug.Stack()),
	)
}

// recoverAnthropicMessagesPanic recovers from panics in the Anthropic Messages
// handler and returns an Anthropic-formatted error response.
func (h *OpenAIGatewayHandler) recoverAnthropicMessagesPanic(c *gin.Context, streamStarted *bool) {
	h.handleRecoveredAnthropicMessagesPanicContext(gatewayctx.FromGin(c), streamStarted, recover())
}

func (h *OpenAIGatewayHandler) recoverAnthropicMessagesPanicContext(c gatewayctx.GatewayContext, streamStarted *bool) {
	h.handleRecoveredAnthropicMessagesPanicContext(c, streamStarted, recover())
}

func (h *OpenAIGatewayHandler) handleRecoveredAnthropicMessagesPanicContext(c gatewayctx.GatewayContext, streamStarted *bool, recovered any) {
	if recovered == nil {
		return
	}

	started := streamStarted != nil && *streamStarted
	requestLoggerContext(c, "handler.openai_gateway.messages").Error(
		"openai.messages_panic_recovered",
		zap.Bool("stream_started", started),
		zap.Any("panic", recovered),
		zap.ByteString("stack", debug.Stack()),
	)
	if !started {
		h.anthropicErrorResponseContext(c, http.StatusInternalServerError, "api_error", "Internal server error")
	}
}

func (h *OpenAIGatewayHandler) ensureResponsesDependencies(c *gin.Context, reqLog *zap.Logger) bool {
	missing := h.missingResponsesDependencies()
	if len(missing) == 0 {
		return true
	}

	if reqLog == nil {
		reqLog = requestLogger(c, "handler.openai_gateway.responses")
	}
	reqLog.Error("openai.handler_dependencies_missing", zap.Strings("missing_dependencies", missing))

	if c != nil && c.Writer != nil && !c.Writer.Written() {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": gin.H{
				"type":    "api_error",
				"message": "Service temporarily unavailable",
			},
		})
	}
	return false
}

func (h *OpenAIGatewayHandler) missingResponsesDependencies() []string {
	missing := make([]string, 0, 5)
	if h == nil {
		return append(missing, "handler")
	}
	if h.gatewayService == nil {
		missing = append(missing, "gatewayService")
	}
	if h.billingCacheService == nil {
		missing = append(missing, "billingCacheService")
	}
	if h.apiKeyService == nil {
		missing = append(missing, "apiKeyService")
	}
	if h.concurrencyHelper == nil || h.concurrencyHelper.concurrencyService == nil {
		missing = append(missing, "concurrencyHelper")
	}
	return missing
}

func getContextInt64(c *gin.Context, key string) (int64, bool) {
	return getContextInt64Gateway(gatewayctx.FromGin(c), key)
}

func getContextInt64Gateway(c gatewayctx.GatewayContext, key string) (int64, bool) {
	if c == nil || key == "" {
		return 0, false
	}
	v, ok := c.Value(key)
	if !ok {
		return 0, false
	}
	switch t := v.(type) {
	case int64:
		return t, true
	case int:
		return int64(t), true
	case int32:
		return int64(t), true
	case float64:
		return int64(t), true
	default:
		return 0, false
	}
}

func getContextStringGateway(c gatewayctx.GatewayContext, key string) string {
	if c == nil || key == "" {
		return ""
	}
	v, ok := c.Value(key)
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}

func shouldMaskOpenAIUpstreamError(statusCode int) bool {
	switch statusCode {
	case http.StatusTooManyRequests, 529, http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func (h *OpenAIGatewayHandler) submitUsageRecordTask(parent context.Context, task service.UsageRecordTask) {
	if task == nil {
		return
	}
	task = wrapUsageRecordTaskContext(parent, task)
	if h.usageRecordWorkerPool != nil {
		h.usageRecordWorkerPool.Submit(task)
		return
	}
	// 回退路径：worker 池未注入时同步执行，避免退回到无界 goroutine 模式。
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	defer func() {
		if recovered := recover(); recovered != nil {
			logger.L().With(
				zap.String("component", "handler.openai_gateway.responses"),
				zap.Any("panic", recovered),
			).Error("openai.usage_record_task_panic_recovered")
		}
	}()
	task(ctx)
}

// handleConcurrencyError handles concurrency-related errors with proper 429 response
func (h *OpenAIGatewayHandler) handleConcurrencyError(c *gin.Context, err error, slotType string, streamStarted bool) {
	h.handleConcurrencyErrorContext(gatewayctx.FromGin(c), err, slotType, streamStarted)
}

func (h *OpenAIGatewayHandler) handleConcurrencyErrorContext(c gatewayctx.GatewayContext, err error, slotType string, streamStarted bool) {
	h.handleStreamingAwareErrorContext(c, http.StatusTooManyRequests, "rate_limit_error",
		fmt.Sprintf("Concurrency limit exceeded for %s, please retry later", slotType), streamStarted)
}

func (h *OpenAIGatewayHandler) handleFailoverExhausted(c *gin.Context, failoverErr *service.UpstreamFailoverError, streamStarted bool) {
	h.handleFailoverExhaustedContext(gatewayctx.FromGin(c), failoverErr, streamStarted)
}

func (h *OpenAIGatewayHandler) handleFailoverExhaustedContext(c gatewayctx.GatewayContext, failoverErr *service.UpstreamFailoverError, streamStarted bool) {
	if failoverErr == nil {
		h.handleStreamingAwareErrorContext(c, http.StatusBadGateway, "upstream_error", "Upstream request failed", streamStarted)
		return
	}
	if failoverErr.IsOpenAIRequestBodyTooLarge() {
		service.SetOpsUpstreamErrorContext(c, http.StatusRequestEntityTooLarge, service.OpenAIRequestBodyTooLargeClientMessage, "")
		h.handleStreamingAwareErrorContext(
			c,
			http.StatusRequestEntityTooLarge,
			"invalid_request_error",
			service.OpenAIRequestBodyTooLargeClientMessage,
			streamStarted,
		)
		return
	}
	statusCode := failoverErr.StatusCode
	responseBody := failoverErr.ResponseBody

	// 先检查透传规则
	if h.errorPassthroughService != nil && len(responseBody) > 0 {
		if rule := h.errorPassthroughService.MatchRule("openai", statusCode, responseBody); rule != nil {
			// 确定响应状态码
			respCode := statusCode
			if !rule.PassthroughCode && rule.ResponseCode != nil {
				respCode = *rule.ResponseCode
			}

			// 确定响应消息
			msg := service.ExtractUpstreamErrorMessage(responseBody)
			if !rule.PassthroughBody && rule.CustomMessage != nil {
				msg = *rule.CustomMessage
			}

			if rule.SkipMonitoring {
				c.SetValue(service.OpsSkipPassthroughKey, true)
			}

			h.handleStreamingAwareErrorContext(c, respCode, "upstream_error", msg, streamStarted)
			return
		}
	}

	if service.WriteOpenAICompactErrorAfterResponseStartedContext(c, "upstream_error", firstNonEmpty(service.ExtractUpstreamErrorMessage(responseBody), "Upstream compact request failed")) {
		return
	}

	// 记录原始上游状态码，以便 ops 错误日志捕获真实的上游错误
	upstreamMsg := service.ExtractUpstreamErrorMessage(responseBody)
	service.SetOpsUpstreamErrorContext(c, statusCode, upstreamMsg, "")

	if isOpenAIPassthroughFailoverContext(c) && statusCode >= http.StatusInternalServerError {
		h.handleStreamingAwareErrorContext(c, http.StatusBadGateway, "upstream_error", "Upstream service temporarily unavailable", streamStarted)
		return
	}

	// 使用默认的错误映射
	status, errType, errMsg := h.mapUpstreamError(statusCode)
	h.handleStreamingAwareErrorContext(c, status, errType, errMsg, streamStarted)
}

func (h *OpenAIGatewayHandler) handleRemoteCompactFailure(c *gin.Context, failoverErr *service.UpstreamFailoverError, streamStarted bool) {
	h.handleRemoteCompactFailureContext(gatewayctx.FromGin(c), failoverErr, streamStarted)
}

func (h *OpenAIGatewayHandler) handleRemoteCompactFailureContext(c gatewayctx.GatewayContext, failoverErr *service.UpstreamFailoverError, streamStarted bool) {
	if failoverErr == nil {
		h.handleStreamingAwareErrorContext(c, http.StatusBadGateway, "upstream_error", "Upstream compact request failed", streamStarted)
		return
	}

	statusCode := failoverErr.StatusCode
	if statusCode < 400 || statusCode > 599 {
		statusCode = http.StatusBadGateway
	}
	responseBody := failoverErr.ResponseBody

	if h.errorPassthroughService != nil && len(responseBody) > 0 {
		if rule := h.errorPassthroughService.MatchRule("openai", statusCode, responseBody); rule != nil {
			respCode := statusCode
			if !rule.PassthroughCode && rule.ResponseCode != nil {
				respCode = *rule.ResponseCode
			}

			msg := service.ExtractUpstreamErrorMessage(responseBody)
			if !rule.PassthroughBody && rule.CustomMessage != nil {
				msg = *rule.CustomMessage
			}

			if rule.SkipMonitoring {
				c.SetValue(service.OpsSkipPassthroughKey, true)
			}

			h.handleStreamingAwareErrorContext(c, respCode, "upstream_error", msg, streamStarted)
			return
		}
	}

	upstreamMsg := strings.TrimSpace(service.ExtractUpstreamErrorMessage(responseBody))
	if upstreamMsg == "" {
		upstreamMsg = strings.TrimSpace(failoverErr.Error())
	}
	if upstreamMsg == "" {
		_, _, mappedMsg := h.mapUpstreamError(statusCode)
		upstreamMsg = mappedMsg
	}
	service.SetOpsUpstreamErrorContext(c, statusCode, upstreamMsg, "")

	if shouldMaskOpenAIUpstreamError(statusCode) {
		mappedStatus, mappedType, mappedMsg := h.mapUpstreamError(statusCode)
		h.handleStreamingAwareErrorContext(c, mappedStatus, mappedType, mappedMsg, streamStarted)
		return
	}

	errType := "upstream_error"
	switch statusCode {
	case http.StatusBadRequest:
		errType = "invalid_request_error"
	case http.StatusUnauthorized:
		errType = "authentication_error"
	case http.StatusTooManyRequests:
		errType = "rate_limit_error"
	}

	h.handleStreamingAwareErrorContext(c, statusCode, errType, upstreamMsg, streamStarted)
}

// handleFailoverExhaustedSimple 简化版本，用于没有响应体的情况
func (h *OpenAIGatewayHandler) handleFailoverExhaustedSimple(c *gin.Context, statusCode int, streamStarted bool) {
	h.handleFailoverExhaustedSimpleContext(gatewayctx.FromGin(c), statusCode, streamStarted)
}

func (h *OpenAIGatewayHandler) handleFailoverExhaustedSimpleContext(c gatewayctx.GatewayContext, statusCode int, streamStarted bool) {
	status, errType, errMsg := h.mapUpstreamError(statusCode)
	service.SetOpsUpstreamErrorContext(c, statusCode, errMsg, "")
	h.handleStreamingAwareErrorContext(c, status, errType, errMsg, streamStarted)
}

func (h *OpenAIGatewayHandler) mapUpstreamError(statusCode int) (int, string, string) {
	switch statusCode {
	case 401:
		return http.StatusBadGateway, "upstream_error", "Upstream authentication failed, please contact administrator"
	case 403:
		return http.StatusBadGateway, "upstream_error", "Upstream access forbidden, please contact administrator"
	case 429:
		return http.StatusServiceUnavailable, "api_error", "Service temporarily unavailable"
	case 529:
		return http.StatusServiceUnavailable, "api_error", "Service temporarily unavailable"
	case 500, 502, 503, 504:
		return http.StatusServiceUnavailable, "api_error", "Service temporarily unavailable"
	default:
		return http.StatusBadGateway, "upstream_error", "Upstream request failed"
	}
}

func isOpenAIPassthroughFailoverContext(c gatewayctx.GatewayContext) bool {
	if c == nil {
		return false
	}
	value, ok := c.Value(service.OpenAIPassthroughFailoverKey)
	if !ok {
		return false
	}
	marked, _ := value.(bool)
	return marked
}

// handleStreamingAwareError handles errors that may occur after streaming has started
func (h *OpenAIGatewayHandler) handleStreamingAwareError(c *gin.Context, status int, errType, message string, streamStarted bool) {
	h.handleStreamingAwareErrorWithCode(c, status, errType, "", message, streamStarted, false)
}

func (h *OpenAIGatewayHandler) handleStreamingAwareErrorWithCode(
	c *gin.Context,
	status int,
	errType string,
	code string,
	message string,
	streamStarted bool,
	countTowardsSLA bool,
) {
	if c == nil {
		return
	}
	if service.StopOpenAICompactSSEKeepaliveCommitted(c) {
		streamStarted = true
	}
	if streamStarted {
		if countTowardsSLA {
			service.MarkOpsStreamFailure(c, errType, code, message, status)
		} else {
			service.MarkOpsStreamError(c, errType, message, status)
		}
		if inboundIsResponses(c) && writeResponsesFailedSSE(c, errType, message) {
			return
		}
		_ = gatewayctx.WriteSSEEvent(gatewayctx.FromGin(c), "error", gin.H{
			"error": gin.H{
				"type":    errType,
				"code":    code,
				"message": message,
			},
		})
		return
	}

	c.JSON(status, gin.H{
		"error": gin.H{
			"type":    errType,
			"code":    code,
			"message": message,
		},
	})
}

func (h *OpenAIGatewayHandler) handleStreamingAwareErrorContext(c gatewayctx.GatewayContext, status int, errType, message string, streamStarted bool) {
	if c == nil {
		return
	}
	if streamStarted || service.RequestUsesOpenAISSE(c) || responseUsesSSEContext(c) {
		if inboundIsResponsesContext(c) && writeResponsesFailedSSEContext(c, errType, message) {
			return
		}
		writeLegacyStreamErrorContext(c, "error", errType, message)
		return
	}

	// Normal case: return JSON response with proper status code
	h.errorResponseGateway(c, status, errType, message)
}

// ensureForwardErrorResponse 在 Forward 返回错误但尚未写响应时补写统一错误响应。
func (h *OpenAIGatewayHandler) ensureForwardErrorResponse(c *gin.Context, streamStarted bool) bool {
	if service.OpenAIImagesJSONKeepalivePresent(c) {
		return service.WriteOpenAIImagesUpstreamErrorResponse(c, &service.OpenAIImagesUpstreamError{
			StatusCode: http.StatusBadGateway,
			ErrorType:  "upstream_error",
			Message:    "Upstream request failed",
		})
	}
	return h.ensureForwardErrorResponseContext(gatewayctx.FromGin(c), streamStarted)
}

func (h *OpenAIGatewayHandler) ensureForwardErrorResponseContext(c gatewayctx.GatewayContext, streamStarted bool) bool {
	if c == nil || service.RequestPayloadStarted(c) {
		return false
	}
	if c.ResponseWritten() && !streamStarted && !service.RequestUsesOpenAISSE(c) && !responseUsesSSEContext(c) && !service.RequestUsesBufferedJSON(c) {
		return false
	}
	h.handleStreamingAwareErrorContext(c, http.StatusBadGateway, "upstream_error", "Upstream request failed", streamStarted)
	return true
}

func shouldLogOpenAIForwardFailureAsWarn(c *gin.Context, wroteFallback bool) bool {
	return shouldLogOpenAIForwardFailureAsWarnContext(gatewayctx.FromGin(c), wroteFallback)
}

func shouldLogOpenAIForwardFailureAsWarnContext(c gatewayctx.GatewayContext, wroteFallback bool) bool {
	if wroteFallback {
		return false
	}
	if c == nil {
		return false
	}
	return c.ResponseWritten()
}

// errorResponse returns OpenAI API format error response
func (h *OpenAIGatewayHandler) errorResponse(c *gin.Context, status int, errType, message string) {
	h.errorResponseGateway(gatewayctx.FromGin(c), status, errType, message)
}

func setOpenAIClientTransportHTTP(c *gin.Context) {
	service.SetOpenAIClientTransport(c, service.OpenAIClientTransportHTTP)
}

func setOpenAIClientTransportHTTPGateway(c gatewayctx.GatewayContext) {
	service.SetOpenAIClientTransportContext(c, service.OpenAIClientTransportHTTP)
}

func setOpenAIClientTransportWS(c *gin.Context) {
	service.SetOpenAIClientTransport(c, service.OpenAIClientTransportWS)
}

func setOpenAIClientTransportWSGateway(c gatewayctx.GatewayContext) {
	service.SetOpenAIClientTransportContext(c, service.OpenAIClientTransportWS)
}

func (h *OpenAIGatewayHandler) errorResponseGateway(c gatewayctx.GatewayContext, status int, errType, message string) {
	if c == nil {
		return
	}
	c.WriteJSON(status, gin.H{
		"error": gin.H{
			"type":    errType,
			"message": message,
		},
	})
}

func (h *OpenAIGatewayHandler) ensureResponsesDependenciesGateway(c gatewayctx.GatewayContext, reqLog *zap.Logger) bool {
	missing := h.missingResponsesDependencies()
	if len(missing) == 0 {
		return true
	}

	if reqLog == nil {
		reqLog = requestLoggerContext(c, "handler.openai_gateway.responses")
	}
	reqLog.Error("openai.handler_dependencies_missing", zap.Strings("missing_dependencies", missing))
	h.errorResponseGateway(c, http.StatusServiceUnavailable, "api_error", "Service temporarily unavailable")
	return false
}

func ensureOpenAIPoolModeSessionHash(sessionHash string, account *service.Account) string {
	if sessionHash != "" || account == nil || !account.IsPoolMode() {
		return sessionHash
	}
	// 为当前请求生成一次性粘性会话键，确保同账号重试不会重新负载均衡到其他账号。
	return "openai-pool-retry-" + uuid.NewString()
}

func openAIWSIngressFallbackSessionSeed(userID, apiKeyID int64, groupID *int64) string {
	gid := int64(0)
	if groupID != nil {
		gid = *groupID
	}
	return fmt.Sprintf("openai_ws_ingress:%d:%d:%d", gid, userID, apiKeyID)
}

func isOpenAIWSUpgradeRequest(r *http.Request) bool {
	if r == nil {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(r.Header.Get("Upgrade")), "websocket") {
		return false
	}
	return strings.Contains(strings.ToLower(strings.TrimSpace(r.Header.Get("Connection"))), "upgrade")
}

func closeOpenAIClientWS(conn *coderws.Conn, status coderws.StatusCode, reason string) {
	if conn == nil {
		return
	}
	reason = strings.TrimSpace(reason)
	if len(reason) > 120 {
		reason = reason[:120]
	}
	_ = conn.Close(status, reason)
	_ = conn.CloseNow()
}

func closeOpenAIWSFailoverExhausted(conn *coderws.Conn, failoverErr *service.UpstreamFailoverError) {
	if failoverErr == nil {
		closeOpenAIClientWS(conn, coderws.StatusInternalError, "upstream websocket proxy failed")
		return
	}
	if failoverErr.Stage == service.GatewayFailureStageAccountAuth {
		closeOpenAIClientWS(conn, coderws.StatusTryAgainLater, service.GrokCredentialUnavailableClientMessage)
		return
	}
	switch failoverErr.StatusCode {
	case http.StatusTooManyRequests:
		closeOpenAIClientWS(conn, coderws.StatusTryAgainLater, "upstream rate limit exceeded, please retry later")
	case 529, http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		closeOpenAIClientWS(conn, coderws.StatusTryAgainLater, "upstream service temporarily unavailable")
	case http.StatusUnauthorized, http.StatusForbidden:
		closeOpenAIClientWS(conn, coderws.StatusPolicyViolation, "upstream websocket authentication failed")
	default:
		closeOpenAIClientWS(conn, coderws.StatusInternalError, "upstream websocket proxy failed")
	}
}

func summarizeWSCloseErrorForLog(err error) (string, string) {
	if err == nil {
		return "-", "-"
	}
	statusCode := coderws.CloseStatus(err)
	if statusCode == -1 {
		return "-", "-"
	}
	closeStatus := fmt.Sprintf("%d(%s)", int(statusCode), statusCode.String())
	closeReason := "-"
	var closeErr coderws.CloseError
	if errors.As(err, &closeErr) {
		reason := strings.TrimSpace(closeErr.Reason)
		if reason != "" {
			closeReason = reason
		}
	}
	return closeStatus, closeReason
}
