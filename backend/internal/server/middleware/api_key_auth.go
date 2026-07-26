package middleware

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

const maxAPIKeyAuthorizationHeaderBytes = service.MaxAPIKeyCredentialBytes + 128

// NewAPIKeyAuthMiddleware creates the API key authentication middleware.
func NewAPIKeyAuthMiddleware(apiKeyService *service.APIKeyService, subscriptionService *service.SubscriptionService, cfg *config.Config) APIKeyAuthMiddleware {
	return APIKeyAuthMiddleware(apiKeyAuthWithSubscription(apiKeyService, subscriptionService, cfg))
}

// apiKeyAuthWithSubscription authenticates API keys and enforces billing rules.
// Authentication checks always run. Billing checks are skipped for read-only
// usage, billing metadata, and asynchronous image task endpoints.
func apiKeyAuthWithSubscription(apiKeyService *service.APIKeyService, subscriptionService *service.SubscriptionService, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !ApplyAPIKeyAuthWithSubscriptionContext(apiKeyService, subscriptionService, cfg, gatewayctx.FromGin(c)) {
			return
		}
		c.Next()
	}
}

// ApplyAPIKeyAuthWithSubscriptionContext applies API key authentication to any
// gateway context. It returns true only when downstream handling may continue.
func ApplyAPIKeyAuthWithSubscriptionContext(apiKeyService *service.APIKeyService, subscriptionService *service.SubscriptionService, cfg *config.Config, c gatewayctx.GatewayContext) bool {
	if c == nil || c.Request() == nil || apiKeyService == nil {
		return false
	}

	if rejectInvalidAuthAbuseContext(c, apiKeyService) {
		abortWithErrorContext(c, http.StatusTooManyRequests, "INVALID_AUTH_RATE_LIMITED", "Too many invalid authentication attempts; retry later")
		return false
	}

	if apiKeyHeadersTooLargeContext(c) {
		recordInvalidAuthFailureContext(c, apiKeyService)
		markIngressRejectedContext(c, IngressRejectInvalidAPIKey)
		abortWithErrorContext(c, http.StatusUnauthorized, "INVALID_API_KEY", "Invalid API key")
		return false
	}

	queryKey := strings.TrimSpace(c.QueryValue("key"))
	queryAPIKey := strings.TrimSpace(c.QueryValue("api_key"))
	if queryKey != "" || queryAPIKey != "" {
		recordInvalidAuthFailureContext(c, apiKeyService)
		markIngressRejectedContext(c, IngressRejectQueryAPIKeyDeprecated)
		abortWithErrorContext(c, http.StatusBadRequest, "api_key_in_query_deprecated", "API key in query parameter is deprecated. Please use Authorization header instead.")
		return false
	}

	apiKeyString := extractAPIKeyFromGatewayContext(c)
	if len(apiKeyString) > service.MaxAPIKeyCredentialBytes {
		recordInvalidAuthFailureContext(c, apiKeyService)
		markIngressRejectedContext(c, IngressRejectInvalidAPIKey)
		abortWithErrorContext(c, http.StatusUnauthorized, "INVALID_API_KEY", "Invalid API key")
		return false
	}
	if apiKeyString == "" {
		recordInvalidAuthFailureContext(c, apiKeyService)
		if hasAPIKeyCredentialInputContext(c) {
			markIngressRejectedContext(c, IngressRejectInvalidAPIKey)
		} else {
			markIngressRejectedContext(c, IngressRejectAPIKeyRequired)
		}
		abortWithErrorContext(c, http.StatusUnauthorized, "API_KEY_REQUIRED", "API key is required in Authorization header (Bearer scheme), x-api-key header, or x-goog-api-key header")
		return false
	}

	apiKey, err := apiKeyService.GetByKey(c.Request().Context(), apiKeyString)
	if err != nil {
		if errors.Is(err, service.ErrAPIKeyNotFound) {
			recordInvalidAuthFailureContext(c, apiKeyService)
			markIngressRejectedContext(c, IngressRejectInvalidAPIKey)
			abortWithErrorContext(c, http.StatusUnauthorized, "INVALID_API_KEY", "Invalid API key")
			return false
		}
		if errors.Is(err, service.ErrAPIKeyAuthOverloaded) {
			markIngressRejectedContext(c, IngressRejectAPIKeyAuthOverloaded)
			abortWithErrorContext(c, http.StatusServiceUnavailable, "API_KEY_AUTH_OVERLOADED", "API key authentication is temporarily unavailable")
			return false
		}
		abortWithErrorContext(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to validate API key")
		return false
	}

	// This value is for Ops diagnostics only; it does not imply successful auth.
	setOpsFallbackAPIKeyContext(c, apiKey)

	// Disabled or unknown states are always rejected. Expired and exhausted keys
	// remain eligible only for the read-only billing bypasses below.
	if !apiKey.IsActive() &&
		apiKey.Status != service.StatusAPIKeyExpired &&
		apiKey.Status != service.StatusAPIKeyQuotaExhausted {
		markIngressRejectedContext(c, IngressRejectAPIKeyDisabled)
		abortWithErrorContext(c, http.StatusUnauthorized, "API_KEY_DISABLED", "API key is disabled")
		return false
	}

	if len(apiKey.IPWhitelist) > 0 || len(apiKey.IPBlacklist) > 0 {
		clientIP := apiKeyACLClientIP(c, cfg)
		allowed, _ := ip.CheckIPRestrictionWithCompiledRules(clientIP, apiKey.CompiledIPWhitelist, apiKey.CompiledIPBlacklist)
		if !allowed {
			if clientIP == "" {
				clientIP = "unknown"
			}
			service.MarkOpsClientBusinessLimitedAny(c, service.OpsClientBusinessLimitedReasonIPRestriction)
			markIngressRejectedContext(c, IngressRejectIPRestricted)
			abortWithErrorContext(c, http.StatusForbidden, "ACCESS_DENIED", fmt.Sprintf("Access denied. Your IP is %s", clientIP))
			return false
		}
	}

	if apiKey.User == nil {
		abortWithErrorContext(c, http.StatusUnauthorized, "USER_NOT_FOUND", "User associated with API key not found")
		return false
	}
	if !apiKey.User.IsActive() {
		markIngressRejectedContext(c, IngressRejectUserInactive)
		abortWithErrorContext(c, http.StatusUnauthorized, "USER_INACTIVE", "User account is not active")
		return false
	}
	if abortIfAPIKeyGroupUnavailableContext(c, apiKey) {
		return false
	}
	if abortIfAPIKeyGroupNotAllowedContext(c, apiKey) {
		return false
	}

	requestContext := context.WithValue(c.Request().Context(), ctxkey.UserID, apiKey.User.ID)
	c.SetRequest(c.Request().WithContext(requestContext))

	billingInfoRequest := c.Path() == "/v1/sub2api/billing"
	skipBilling := billingInfoRequest ||
		isReadOnlyGatewayMetadata(c.Method(), c.Path()) ||
		isAsyncImageTaskRead(c.Method(), c.Path())

	if cfg != nil && cfg.RunMode == config.RunModeSimple {
		setAPIKeyAuthContextValues(c, apiKey, nil)
		if !billingInfoRequest {
			_ = apiKeyService.TouchLastUsed(c.Request().Context(), apiKey.ID)
		}
		return true
	}

	var subscription *service.UserSubscription
	isSubscriptionType := apiKey.Group != nil && apiKey.Group.IsSubscriptionType()
	if isSubscriptionType && subscriptionService != nil && !billingInfoRequest {
		sub, subErr := subscriptionService.GetActiveSubscription(
			c.Request().Context(),
			apiKey.User.ID,
			apiKey.Group.ID,
		)
		if subErr != nil {
			if !skipBilling {
				abortWithErrorContext(c, http.StatusForbidden, "SUBSCRIPTION_NOT_FOUND", "No active subscription found for this group")
				return false
			}
		} else {
			subscription = sub
		}
	}

	if !skipBilling {
		switch apiKey.Status {
		case service.StatusAPIKeyQuotaExhausted:
			abortWithAPIKeyQuotaErrorContext(c)
			return false
		case service.StatusAPIKeyExpired:
			abortWithErrorContext(c, http.StatusForbidden, "API_KEY_EXPIRED", "API key 已过期")
			return false
		}

		if apiKey.IsExpired() {
			abortWithErrorContext(c, http.StatusForbidden, "API_KEY_EXPIRED", "API key 已过期")
			return false
		}
		if apiKey.IsQuotaExhausted() {
			abortWithAPIKeyQuotaErrorContext(c)
			return false
		}

		if subscription != nil {
			needsMaintenance, validateErr := subscriptionService.ValidateAndCheckLimits(subscription, apiKey.Group)
			if needsMaintenance {
				refreshed, maintenanceErr := subscriptionService.EnsureWindowMaintenance(c.Request().Context(), subscription)
				if maintenanceErr != nil {
					abortWithErrorContext(c, http.StatusInternalServerError, "SUBSCRIPTION_MAINTENANCE_FAILED", "Failed to maintain subscription usage windows")
					return false
				}
				subscription = refreshed
				_, validateErr = subscriptionService.ValidateAndCheckLimits(subscription, apiKey.Group)
			}
			if validateErr != nil {
				code := "SUBSCRIPTION_INVALID"
				status := http.StatusForbidden
				if errors.Is(validateErr, service.ErrDailyLimitExceeded) ||
					errors.Is(validateErr, service.ErrWeeklyLimitExceeded) ||
					errors.Is(validateErr, service.ErrMonthlyLimitExceeded) {
					code = "USAGE_LIMIT_EXCEEDED"
					status = http.StatusTooManyRequests
				}
				abortWithErrorContext(c, status, code, validateErr.Error())
				return false
			}
		} else if apiKeyBalanceBelowAuthThreshold(apiKey.User.Balance, cfg) {
			abortWithErrorContext(c, http.StatusForbidden, "INSUFFICIENT_BALANCE", "Insufficient account balance")
			return false
		}
	}

	setAPIKeyAuthContextValues(c, apiKey, subscription)
	if !billingInfoRequest {
		_ = apiKeyService.TouchLastUsed(c.Request().Context(), apiKey.ID)
	}
	return true
}

func extractAPIKeyFromGatewayContext(c gatewayctx.GatewayContext) string {
	if c == nil {
		return ""
	}
	authHeader := strings.TrimSpace(c.HeaderValue("Authorization"))
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			if token := strings.TrimSpace(parts[1]); token != "" {
				return token
			}
		}
	}
	if token := strings.TrimSpace(c.HeaderValue("x-api-key")); token != "" {
		return token
	}
	if token := strings.TrimSpace(c.HeaderValue("x-goog-api-key")); token != "" {
		return token
	}
	return ""
}

func setAPIKeyAuthContextValues(c gatewayctx.GatewayContext, apiKey *service.APIKey, subscription *service.UserSubscription) {
	if c == nil || apiKey == nil || apiKey.User == nil {
		return
	}
	if subscription != nil {
		c.SetValue(string(ContextKeySubscription), subscription)
	}
	c.SetValue(string(ContextKeyAPIKey), apiKey)
	c.SetValue(string(ContextKeyUser), AuthSubject{
		UserID:          apiKey.User.ID,
		Concurrency:     apiKey.User.Concurrency,
		AllowedGroupIDs: cloneAuthSubjectGroupIDs(apiKey.User.AllowedGroups),
	})
	c.SetValue(string(ContextKeyUserRole), apiKey.User.Role)
	setGroupContextGateway(c, apiKey.Group)
}

func abortWithErrorContext(c gatewayctx.GatewayContext, statusCode int, code, message string) {
	if c == nil {
		return
	}
	c.WriteJSON(statusCode, NewErrorResponse(code, message))
	c.Abort()
}

func apiKeyHeadersTooLarge(c *gin.Context) bool {
	return apiKeyHeadersTooLargeContext(gatewayctx.FromGin(c))
}

func apiKeyHeadersTooLargeContext(c gatewayctx.GatewayContext) bool {
	if c == nil {
		return false
	}
	return len(c.HeaderValue("Authorization")) > maxAPIKeyAuthorizationHeaderBytes ||
		len(c.HeaderValue("x-api-key")) > service.MaxAPIKeyCredentialBytes ||
		len(c.HeaderValue("x-goog-api-key")) > service.MaxAPIKeyCredentialBytes
}

func hasAPIKeyCredentialInput(c *gin.Context) bool {
	return hasAPIKeyCredentialInputContext(gatewayctx.FromGin(c))
}

func hasAPIKeyCredentialInputContext(c gatewayctx.GatewayContext) bool {
	if c == nil {
		return false
	}
	return c.HeaderValue("Authorization") != "" ||
		c.HeaderValue("x-api-key") != "" ||
		c.HeaderValue("x-goog-api-key") != ""
}

func abortWithAPIKeyQuotaError(c *gin.Context) {
	abortWithAPIKeyQuotaErrorContext(gatewayctx.FromGin(c))
}

func abortWithAPIKeyQuotaErrorContext(c gatewayctx.GatewayContext) {
	const message = "API key 额度已用完"
	if isOpenAICompatibleAPIKeyRequestContext(c) {
		c.WriteJSON(http.StatusTooManyRequests, gin.H{
			"error": gin.H{
				"message": message,
				"type":    "insufficient_quota",
				"param":   nil,
				"code":    "insufficient_quota",
			},
		})
		c.Abort()
		return
	}
	abortWithErrorContext(c, http.StatusTooManyRequests, "API_KEY_QUOTA_EXHAUSTED", message)
}

func isOpenAICompatibleAPIKeyRequest(c *gin.Context) bool {
	return isOpenAICompatibleAPIKeyRequestContext(gatewayctx.FromGin(c))
}

func isOpenAICompatibleAPIKeyRequestContext(c gatewayctx.GatewayContext) bool {
	if c == nil {
		return false
	}
	path := strings.TrimRight(c.Path(), "/")
	for _, root := range []string{
		"/v1/responses",
		"/openai/v1/responses",
		"/responses",
		"/backend-api/codex/responses",
	} {
		if path == root || strings.HasPrefix(path, root+"/") {
			return true
		}
	}
	return false
}

func isAsyncImageTaskRead(method, path string) bool {
	if method != http.MethodGet {
		return false
	}
	return strings.HasPrefix(path, "/v1/images/tasks/") || strings.HasPrefix(path, "/images/tasks/")
}

// isReadOnlyGatewayMetadata reports whether the request only reads the caller's
// own gateway metadata: the model catalog（/v1/models 及其无前缀别名，二者
// 注册的是同一个 handler）或用量统计（/v1/usage），以及平台前缀下的同类只读
// 端点（antigravity/sora）。这些只读端点跳过下方的计费闸门（含零余额拦截）：
// 余额是消费门槛，不是身份门槛，新用户在充值前也应能验证连通性、读取自己的
// 元数据。真正产生费用的 /v1/chat/completions、/v1/messages 等推理端点不在
// 此列，仍受余额拦截。只允许 GET + 精确路径：平台前缀下的推理端点（如
// POST /antigravity/v1/messages）与元数据同前缀，禁止前缀匹配。
func isReadOnlyGatewayMetadata(method, path string) bool {
	if method != http.MethodGet {
		return false
	}
	switch strings.TrimRight(path, "/") {
	case "/v1/models", "/models", "/v1/usage",
		"/antigravity/v1/models", "/antigravity/v1/usage", "/antigravity/models",
		"/sora/v1/models":
		return true
	}
	return false
}

func rejectInvalidAuthAbuseContext(c gatewayctx.GatewayContext, apiKeyService *service.APIKeyService) bool {
	if c == nil || apiKeyService == nil {
		return false
	}
	retry, blocked := apiKeyService.CheckInvalidAuthAbuse(invalidAuthClientKeyContext(c))
	if !blocked {
		return false
	}
	retrySeconds := int(math.Ceil(retry.Seconds()))
	if retrySeconds < 1 {
		retrySeconds = 1
	}
	c.SetHeader("Retry-After", strconv.Itoa(retrySeconds))
	markIngressRejectedContext(c, IngressRejectInvalidAuthRateLimited)
	return true
}

func recordInvalidAuthFailureContext(c gatewayctx.GatewayContext, apiKeyService *service.APIKeyService) {
	if c == nil || apiKeyService == nil {
		return
	}
	apiKeyService.RecordInvalidAuthFailure(invalidAuthClientKeyContext(c))
}

func invalidAuthClientKeyContext(c gatewayctx.GatewayContext) string {
	if c == nil {
		return normalizeIngressRejectIP("")
	}
	if req := c.Request(); req != nil {
		if binding := service.SessionBindingFromContext(req.Context()); binding != nil && strings.TrimSpace(binding.IP) != "" {
			return normalizeIngressRejectIP(binding.IP)
		}
	}
	return normalizeIngressRejectIP(gatewayctx.TrustedClientIP(c))
}

func markIngressRejectedContext(c gatewayctx.GatewayContext, reason IngressRejectReason) {
	if c == nil || reason == "" {
		return
	}
	c.SetValue(ingressRejectReasonContextKey, reason)
}

func apiKeyACLClientIP(c gatewayctx.GatewayContext, cfg *config.Config) string {
	if c == nil {
		return ""
	}
	if ginContext, ok := c.Native().(*gin.Context); ok {
		return ip.GetSecurityClientIP(ginContext, cfg.TrustForwardedIPForAPIKeyACL())
	}
	if !cfg.TrustForwardedIPForAPIKeyACL() {
		return gatewayctx.TrustedClientIP(c)
	}
	return legacyForwardedAPIKeyClientIP(c, cfg.ForwardedClientIPSettings().Headers)
}

func legacyForwardedAPIKeyClientIP(c gatewayctx.GatewayContext, customHeaders []string) string {
	if c == nil || c.Request() == nil {
		return ""
	}
	request := c.Request()
	customIP, customFallback := resolveCustomAPIKeyForwardedIP(request, customHeaders)
	if customIP != "" {
		return customIP
	}
	legacyIP, legacyFallback := resolveLegacyAPIKeyForwardedIP(request)
	if legacyIP != "" {
		return legacyIP
	}
	if customFallback != "" {
		return customFallback
	}
	if legacyFallback != "" {
		return legacyFallback
	}
	return gatewayctx.TrustedClientIP(c)
}

func resolveCustomAPIKeyForwardedIP(request *http.Request, headers []string) (string, string) {
	if request == nil {
		return "", ""
	}
	var fallback string
	for _, header := range headers {
		for _, value := range request.Header.Values(header) {
			for _, candidate := range strings.Split(value, ",") {
				parsed := net.ParseIP(strings.TrimSpace(candidate))
				if parsed == nil {
					continue
				}
				normalized := parsed.String()
				if isPrivateAPIKeyClientIP(normalized) {
					if fallback == "" {
						fallback = normalized
					}
					continue
				}
				return normalized, fallback
			}
		}
	}
	return "", fallback
}

func resolveLegacyAPIKeyForwardedIP(request *http.Request) (string, string) {
	if request == nil {
		return "", ""
	}
	var fallback string
	if forwarded := normalizeAPIKeyClientIP(request.Header.Get("CF-Connecting-IP")); forwarded != "" {
		fallback = forwarded
		if !isPrivateAPIKeyClientIP(forwarded) {
			return forwarded, fallback
		}
	}
	if realIP := normalizeAPIKeyClientIP(request.Header.Get("X-Real-IP")); realIP != "" {
		if fallback == "" {
			fallback = realIP
		}
		if !isPrivateAPIKeyClientIP(realIP) {
			return realIP, fallback
		}
	}
	if forwardedFor := request.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		candidates := strings.Split(forwardedFor, ",")
		for _, candidate := range candidates {
			candidate = strings.TrimSpace(candidate)
			if candidate != "" && !isPrivateAPIKeyClientIP(candidate) {
				return normalizeAPIKeyClientIP(candidate), fallback
			}
		}
		if fallback == "" && len(candidates) > 0 {
			fallback = normalizeAPIKeyClientIP(strings.TrimSpace(candidates[0]))
		}
	}
	return "", fallback
}

func normalizeAPIKeyClientIP(value string) string {
	value = strings.TrimSpace(value)
	if host, _, err := net.SplitHostPort(value); err == nil {
		return host
	}
	return value
}

func isPrivateAPIKeyClientIP(value string) bool {
	parsed := net.ParseIP(value)
	return parsed != nil && (parsed.IsPrivate() || parsed.IsLoopback())
}

// GetAPIKeyFromContext returns the authenticated API key from a Gin context.
func GetAPIKeyFromContext(c *gin.Context) (*service.APIKey, bool) {
	return GetAPIKeyFromGatewayContext(gatewayctx.FromGin(c))
}

func GetAPIKeyFromGatewayContext(c gatewayctx.GatewayContext) (*service.APIKey, bool) {
	if c == nil {
		return nil, false
	}
	value, exists := c.Value(string(ContextKeyAPIKey))
	if !exists {
		return nil, false
	}
	apiKey, ok := value.(*service.APIKey)
	return apiKey, ok
}

// SetOpsFallbackAPIKey records a loaded key for Ops diagnostics on auth aborts.
func SetOpsFallbackAPIKey(c *gin.Context, apiKey *service.APIKey) {
	setOpsFallbackAPIKeyContext(gatewayctx.FromGin(c), apiKey)
}

func setOpsFallbackAPIKeyContext(c gatewayctx.GatewayContext, apiKey *service.APIKey) {
	if c == nil || apiKey == nil {
		return
	}
	c.SetValue(string(ContextKeyOpsFallbackAPIKey), apiKey)
}

// GetOpsFallbackAPIKey returns the Ops-only fallback key.
func GetOpsFallbackAPIKey(c *gin.Context) (*service.APIKey, bool) {
	return getOpsFallbackAPIKeyContext(gatewayctx.FromGin(c))
}

func getOpsFallbackAPIKeyContext(c gatewayctx.GatewayContext) (*service.APIKey, bool) {
	if c == nil {
		return nil, false
	}
	value, exists := c.Value(string(ContextKeyOpsFallbackAPIKey))
	if !exists {
		return nil, false
	}
	apiKey, ok := value.(*service.APIKey)
	return apiKey, ok
}

// GetSubscriptionFromContext returns subscription data from a Gin context.
func GetSubscriptionFromContext(c *gin.Context) (*service.UserSubscription, bool) {
	return GetSubscriptionFromGatewayContext(gatewayctx.FromGin(c))
}

func GetSubscriptionFromGatewayContext(c gatewayctx.GatewayContext) (*service.UserSubscription, bool) {
	if c == nil {
		return nil, false
	}
	value, exists := c.Value(string(ContextKeySubscription))
	if !exists {
		return nil, false
	}
	subscription, ok := value.(*service.UserSubscription)
	return subscription, ok
}

func setGroupContext(c *gin.Context, group *service.Group) {
	setGroupContextGateway(gatewayctx.FromGin(c), group)
}

func setGroupContextGateway(c gatewayctx.GatewayContext, group *service.Group) {
	if c == nil || c.Request() == nil || !service.IsGroupContextValid(group) {
		return
	}
	if existing, ok := c.Request().Context().Value(ctxkey.Group).(*service.Group); ok &&
		existing != nil && existing.ID == group.ID && service.IsGroupContextValid(existing) {
		return
	}
	requestContext := context.WithValue(c.Request().Context(), ctxkey.Group, group)
	c.SetRequest(c.Request().WithContext(requestContext))
}

// MinimumBalanceReserve is a conservative billing-cache threshold, not an auth
// threshold. Existing users with a positive balance must remain authorized.
func apiKeyBalanceBelowAuthThreshold(balance float64, _ *config.Config) bool {
	return balance <= 0
}

func abortIfAPIKeyGroupUnavailable(c *gin.Context, apiKey *service.APIKey) bool {
	return abortIfAPIKeyGroupUnavailableContext(gatewayctx.FromGin(c), apiKey)
}

func abortIfAPIKeyGroupUnavailableContext(c gatewayctx.GatewayContext, apiKey *service.APIKey) bool {
	code, message, ok := validateAPIKeyGroupAvailable(apiKey)
	if ok {
		return false
	}
	service.MarkOpsClientBusinessLimitedAny(c, service.OpsClientBusinessLimitedReasonAPIKeyGroupUnavailable)
	if code == "GROUP_DELETED" {
		markIngressRejectedContext(c, IngressRejectGroupDeleted)
	} else {
		markIngressRejectedContext(c, IngressRejectGroupDisabled)
	}
	abortWithErrorContext(c, http.StatusForbidden, code, message)
	return true
}

func abortIfAPIKeyGroupNotAllowed(c *gin.Context, apiKey *service.APIKey) bool {
	return abortIfAPIKeyGroupNotAllowedContext(gatewayctx.FromGin(c), apiKey)
}

func abortIfAPIKeyGroupNotAllowedContext(c gatewayctx.GatewayContext, apiKey *service.APIKey) bool {
	if validateAPIKeyGroupAllowed(apiKey) {
		return false
	}
	service.MarkOpsClientBusinessLimitedAny(c, service.OpsClientBusinessLimitedReasonAPIKeyGroupUnavailable)
	markIngressRejectedContext(c, IngressRejectGroupNotAllowed)
	abortWithErrorContext(c, http.StatusForbidden, "GROUP_NOT_ALLOWED", "API Key 所属专属分组不再允许当前用户使用")
	return true
}

func validateAPIKeyGroupAllowed(apiKey *service.APIKey) bool {
	if apiKey == nil || apiKey.GroupID == nil || apiKey.User == nil || apiKey.Group == nil {
		return true
	}
	group := apiKey.Group
	if group.IsSubscriptionType() {
		return true
	}
	return apiKey.User.CanBindGroup(group.ID, group.IsExclusive)
}

func validateAPIKeyGroupAvailable(apiKey *service.APIKey) (string, string, bool) {
	if apiKey == nil || apiKey.GroupID == nil {
		return "", "", true
	}
	group := apiKey.Group
	if group == nil || strings.EqualFold(group.Status, "deleted") {
		return "GROUP_DELETED", "API Key 所属分组已删除", false
	}
	if !group.IsActive() {
		return "GROUP_DISABLED", "API Key 所属分组已停用", false
	}
	return "", "", true
}
