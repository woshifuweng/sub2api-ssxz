package middleware

import (
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// BackendModeUserGuard blocks non-admin users from accessing user routes when backend mode is enabled.
// Must be placed AFTER JWT auth middleware so that the user role is available in context.
func BackendModeUserGuard(settingService *service.SettingService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if ApplyBackendModeUserGuardContext(settingService, gatewayctx.FromGin(c)) {
			c.Next()
		}
	}
}

func ApplyBackendModeUserGuardContext(settingService *service.SettingService, c gatewayctx.GatewayContext) bool {
	if settingService == nil || c == nil || !settingService.IsBackendModeEnabled(c.Request().Context()) {
		return true
	}
	role, _ := GetUserRoleFromGatewayContext(c)
	if role == "admin" {
		return true
	}
	c.WriteJSON(http.StatusForbidden, response.Response{Code: http.StatusForbidden, Message: "Backend mode is active. User self-service is disabled."})
	c.Abort()
	return false
}

func backendModeAllowsAuthPath(path string) bool {
	path = strings.ToLower(strings.TrimSpace(path))
	for _, suffix := range []string{
		"/auth/login",
		"/auth/login/2fa",
		"/auth/passkey/login/begin",
		"/auth/passkey/login/finish",
		"/auth/logout",
		"/auth/refresh",
	} {
		if strings.HasSuffix(path, suffix) {
			return true
		}
	}

	for _, suffix := range []string{
		"/auth/oauth/linuxdo/callback",
		"/auth/oauth/wechat/callback",
		"/auth/oauth/wechat/payment/callback",
		"/auth/oauth/oidc/callback",
		"/auth/oauth/github/callback",
		"/auth/oauth/google/callback",
		"/auth/oauth/dingtalk/callback",
		"/auth/oauth/github/complete-registration",
		"/auth/oauth/google/complete-registration",
		"/auth/oauth/linuxdo/complete-registration",
		"/auth/oauth/wechat/complete-registration",
		"/auth/oauth/oidc/complete-registration",
		"/auth/oauth/dingtalk/complete-registration",
		"/auth/oauth/linuxdo/create-account",
		"/auth/oauth/wechat/create-account",
		"/auth/oauth/oidc/create-account",
		"/auth/oauth/dingtalk/create-account",
		"/auth/oauth/linuxdo/bind-login",
		"/auth/oauth/wechat/bind-login",
		"/auth/oauth/oidc/bind-login",
		"/auth/oauth/dingtalk/bind-login",
	} {
		if strings.HasSuffix(path, suffix) {
			return true
		}
	}

	return strings.Contains(path, "/auth/oauth/pending/")
}

// BackendModeAuthGuard selectively blocks auth endpoints when backend mode is enabled.
// OAuth callbacks and pending continuations remain available so that flows
// started before backend mode was enabled can complete their handler-level checks.
func BackendModeAuthGuard(settingService *service.SettingService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if ApplyBackendModeAuthGuardContext(settingService, gatewayctx.FromGin(c)) {
			c.Next()
		}
	}
}

func ApplyBackendModeAuthGuardContext(settingService *service.SettingService, c gatewayctx.GatewayContext) bool {
	if settingService == nil || c == nil || !settingService.IsBackendModeEnabled(c.Request().Context()) {
		return true
	}
	if backendModeAllowsAuthPath(c.Path()) {
		return true
	}
	c.WriteJSON(http.StatusForbidden, response.Response{Code: http.StatusForbidden, Message: "Backend mode is active. Registration and self-service auth flows are disabled."})
	c.Abort()
	return false
}
