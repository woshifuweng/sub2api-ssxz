package middleware

import (
	"errors"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/googleapi"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// APIKeyAuthGoogle is a Google-style error wrapper for API key auth.
func APIKeyAuthGoogle(apiKeyService *service.APIKeyService, cfg *config.Config) gin.HandlerFunc {
	return APIKeyAuthWithSubscriptionGoogle(apiKeyService, nil, cfg)
}

// APIKeyAuthWithSubscriptionGoogle behaves like ApiKeyAuthWithSubscription but returns Google-style errors:
// {"error":{"code":401,"message":"...","status":"UNAUTHENTICATED"}}
//
// It is intended for Gemini native endpoints (/v1beta) to match Gemini SDK expectations.
func APIKeyAuthWithSubscriptionGoogle(apiKeyService *service.APIKeyService, subscriptionService *service.SubscriptionService, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !ApplyAPIKeyAuthWithSubscriptionGoogleContext(apiKeyService, subscriptionService, cfg, gatewayctx.FromGin(c)) {
			return
		}
		c.Next()
	}
}

// ApplyAPIKeyAuthWithSubscriptionGoogleContext applies Google-style API key
// authentication to any gateway context. It returns true only when downstream
// handling may continue.
func ApplyAPIKeyAuthWithSubscriptionGoogleContext(apiKeyService *service.APIKeyService, subscriptionService *service.SubscriptionService, cfg *config.Config, c gatewayctx.GatewayContext) bool {
	if c == nil || c.Request() == nil || apiKeyService == nil {
		return false
	}

	if rejectInvalidAuthAbuseContext(c, apiKeyService) {
		abortWithGoogleErrorContext(c, 429, "Too many invalid authentication attempts; retry later")
		return false
	}
	if apiKeyHeadersTooLargeContext(c) {
		recordInvalidAuthFailureContext(c, apiKeyService)
		markIngressRejectedContext(c, IngressRejectInvalidAPIKey)
		abortWithGoogleErrorContext(c, 401, "Invalid API key")
		return false
	}
	if value := strings.TrimSpace(c.QueryValue("api_key")); value != "" {
		recordInvalidAuthFailureContext(c, apiKeyService)
		markIngressRejectedContext(c, IngressRejectQueryAPIKeyDeprecated)
		abortWithGoogleErrorContext(c, 400, "Query parameter api_key is deprecated. Use Authorization header or key instead.")
		return false
	}

	apiKeyString := extractAPIKeyForGoogleContext(c)
	if apiKeyString == "" {
		recordInvalidAuthFailureContext(c, apiKeyService)
		if hasAPIKeyCredentialInputContext(c) {
			markIngressRejectedContext(c, IngressRejectInvalidAPIKey)
		} else {
			markIngressRejectedContext(c, IngressRejectAPIKeyRequired)
		}
		abortWithGoogleErrorContext(c, 401, "API key is required")
		return false
	}
	if len(apiKeyString) > service.MaxAPIKeyCredentialBytes {
		recordInvalidAuthFailureContext(c, apiKeyService)
		markIngressRejectedContext(c, IngressRejectInvalidAPIKey)
		abortWithGoogleErrorContext(c, 401, "Invalid API key")
		return false
	}

	apiKey, err := apiKeyService.GetByKey(c.Request().Context(), apiKeyString)
	if err != nil {
		if errors.Is(err, service.ErrAPIKeyNotFound) {
			recordInvalidAuthFailureContext(c, apiKeyService)
			markIngressRejectedContext(c, IngressRejectInvalidAPIKey)
			abortWithGoogleErrorContext(c, 401, "Invalid API key")
			return false
		}
		if errors.Is(err, service.ErrAPIKeyAuthOverloaded) {
			markIngressRejectedContext(c, IngressRejectAPIKeyAuthOverloaded)
			abortWithGoogleErrorContext(c, 503, "API key authentication is temporarily unavailable")
			return false
		}
		abortWithGoogleErrorContext(c, 500, "Failed to validate API key")
		return false
	}

	// This value is for Ops diagnostics only; it does not imply successful auth.
	setOpsFallbackAPIKeyContext(c, apiKey)

	if !apiKey.IsActive() &&
		apiKey.Status != service.StatusAPIKeyExpired &&
		apiKey.Status != service.StatusAPIKeyQuotaExhausted {
		markIngressRejectedContext(c, IngressRejectAPIKeyDisabled)
		abortWithGoogleErrorContext(c, 401, "API key is disabled")
		return false
	}
	if len(apiKey.IPWhitelist) > 0 || len(apiKey.IPBlacklist) > 0 {
		clientIP := apiKeyACLClientIP(c, cfg)
		allowed, _ := ip.CheckIPRestrictionWithCompiledRules(clientIP, apiKey.CompiledIPWhitelist, apiKey.CompiledIPBlacklist)
		if !allowed {
			service.MarkOpsClientBusinessLimitedAny(c, service.OpsClientBusinessLimitedReasonIPRestriction)
			markIngressRejectedContext(c, IngressRejectIPRestricted)
			abortWithGoogleErrorContext(c, 403, "Access denied")
			return false
		}
	}
	if apiKey.User == nil {
		abortWithGoogleErrorContext(c, 401, "User associated with API key not found")
		return false
	}
	if !apiKey.User.IsActive() {
		markIngressRejectedContext(c, IngressRejectUserInactive)
		abortWithGoogleErrorContext(c, 401, "User account is not active")
		return false
	}
	if abortIfGoogleAPIKeyGroupUnavailableContext(c, apiKey) {
		return false
	}
	if abortIfGoogleAPIKeyGroupNotAllowedContext(c, apiKey) {
		return false
	}

	// Simple mode intentionally skips balance, quota, and subscription checks.
	if cfg != nil && cfg.RunMode == config.RunModeSimple {
		setAPIKeyAuthContextValues(c, apiKey, nil)
		_ = apiKeyService.TouchLastUsed(c.Request().Context(), apiKey.ID)
		return true
	}

	switch apiKey.Status {
	case service.StatusAPIKeyQuotaExhausted:
		abortWithGoogleErrorContext(c, 429, "API key quota exhausted")
		return false
	case service.StatusAPIKeyExpired:
		abortWithGoogleErrorContext(c, 403, "API key has expired")
		return false
	}
	if apiKey.IsExpired() {
		abortWithGoogleErrorContext(c, 403, "API key has expired")
		return false
	}
	if apiKey.IsQuotaExhausted() {
		abortWithGoogleErrorContext(c, 429, "API key quota exhausted")
		return false
	}

	var subscription *service.UserSubscription
	isSubscriptionType := apiKey.Group != nil && apiKey.Group.IsSubscriptionType()
	if isSubscriptionType && subscriptionService != nil {
		subscription, err = subscriptionService.GetActiveSubscription(
			c.Request().Context(),
			apiKey.User.ID,
			apiKey.Group.ID,
		)
		if err != nil {
			abortWithGoogleErrorContext(c, 403, "No active subscription found for this group")
			return false
		}

		needsMaintenance, validateErr := subscriptionService.ValidateAndCheckLimits(subscription, apiKey.Group)
		if needsMaintenance {
			refreshed, maintenanceErr := subscriptionService.EnsureWindowMaintenance(c.Request().Context(), subscription)
			if maintenanceErr != nil {
				abortWithGoogleErrorContext(c, 500, "Failed to maintain subscription usage windows")
				return false
			}
			subscription = refreshed
			_, validateErr = subscriptionService.ValidateAndCheckLimits(subscription, apiKey.Group)
		}
		if validateErr != nil {
			status := 403
			if errors.Is(validateErr, service.ErrDailyLimitExceeded) ||
				errors.Is(validateErr, service.ErrWeeklyLimitExceeded) ||
				errors.Is(validateErr, service.ErrMonthlyLimitExceeded) {
				status = 429
			}
			abortWithGoogleErrorContext(c, status, validateErr.Error())
			return false
		}
	} else if apiKeyBalanceBelowAuthThreshold(apiKey.User.Balance, cfg) {
		abortWithGoogleErrorContext(c, 403, "Insufficient account balance")
		return false
	}

	setAPIKeyAuthContextValues(c, apiKey, subscription)
	_ = apiKeyService.TouchLastUsed(c.Request().Context(), apiKey.ID)
	return true
}

func abortIfGoogleAPIKeyGroupUnavailableContext(c gatewayctx.GatewayContext, apiKey *service.APIKey) bool {
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
	abortWithGoogleErrorContext(c, 403, message)
	return true
}

func abortIfGoogleAPIKeyGroupNotAllowedContext(c gatewayctx.GatewayContext, apiKey *service.APIKey) bool {
	if validateAPIKeyGroupAllowed(apiKey) {
		return false
	}
	service.MarkOpsClientBusinessLimitedAny(c, service.OpsClientBusinessLimitedReasonAPIKeyGroupUnavailable)
	markIngressRejectedContext(c, IngressRejectGroupNotAllowed)
	abortWithGoogleErrorContext(c, 403, "API key group is no longer allowed for this user")
	return true
}

// extractAPIKeyForGoogle extracts API key for Google/Gemini endpoints.
// Priority: x-goog-api-key > Authorization: Bearer > x-api-key > query key.
func extractAPIKeyForGoogle(c *gin.Context) string {
	return extractAPIKeyForGoogleContext(gatewayctx.FromGin(c))
}

func extractAPIKeyForGoogleContext(c gatewayctx.GatewayContext) string {
	if c == nil {
		return ""
	}
	if key := strings.TrimSpace(c.HeaderValue("x-goog-api-key")); key != "" {
		return key
	}

	auth := strings.TrimSpace(c.HeaderValue("Authorization"))
	if auth != "" {
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			if key := strings.TrimSpace(parts[1]); key != "" {
				return key
			}
		}
	}

	if key := strings.TrimSpace(c.HeaderValue("x-api-key")); key != "" {
		return key
	}
	if allowGoogleQueryKey(c.Path()) {
		if value := strings.TrimSpace(c.QueryValue("key")); value != "" {
			return value
		}
	}
	return ""
}

func abortWithGoogleError(c *gin.Context, status int, message string) {
	abortWithGoogleErrorContext(gatewayctx.FromGin(c), status, message)
}

func abortWithGoogleErrorContext(c gatewayctx.GatewayContext, status int, message string) {
	if c == nil {
		return
	}
	c.WriteJSON(status, gin.H{
		"error": gin.H{
			"code":    status,
			"message": message,
			"status":  googleapi.HTTPStatusToGoogleStatus(status),
		},
	})
	c.Abort()
}

func allowGoogleQueryKey(path string) bool {
	return strings.HasPrefix(path, "/v1beta") || strings.HasPrefix(path, "/antigravity/v1beta")
}
