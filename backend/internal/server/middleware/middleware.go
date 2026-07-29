package middleware

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/googleapi"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// ContextKey 定义上下文键类型
type ContextKey string

const (
	// ContextKeyUser 用户上下文键
	ContextKeyUser ContextKey = "user"
	// ContextKeyUserRole 当前用户角色（string）
	ContextKeyUserRole ContextKey = "user_role"
	// ContextKeyAPIKey API密钥上下文键
	ContextKeyAPIKey ContextKey = "api_key"
	// ContextKeySubscription 订阅上下文键
	ContextKeySubscription ContextKey = "subscription"
	// ContextKeyForcePlatform 强制平台（用于 /antigravity 路由）
	ContextKeyForcePlatform ContextKey = "force_platform"
	// ContextKeyOpsFallbackAPIKey 运维错误日志专用回退键。
	// 鉴权早退（分组停用/删除、Key 停用/过期/额度、用户停用、IP 限制等）时，
	// apiKey 已加载但尚未写入 ContextKeyAPIKey；该键让 Ops 错误日志仍能取到
	// user/group/platform。仅供 Ops 错误日志读取，不代表请求已通过鉴权。
	ContextKeyOpsFallbackAPIKey ContextKey = "ops_fallback_api_key"
)

// ForcePlatform 返回设置强制平台的中间件
// 同时设置 request.Context（供 Service 使用）和 gin.Context（供 Handler 快速检查）
func ForcePlatform(platform string) gin.HandlerFunc {
	return func(c *gin.Context) {
		SetForcePlatformContext(gatewayctx.FromGin(c), platform)
		c.Next()
	}
}

func SetForcePlatformContext(c gatewayctx.GatewayContext, platform string) {
	if c == nil || c.Request() == nil {
		return
	}
	ctx := context.WithValue(c.Request().Context(), ctxkey.ForcePlatform, platform)
	c.SetRequest(c.Request().WithContext(ctx))
	c.SetValue(string(ContextKeyForcePlatform), platform)
}

// HasForcePlatform 检查是否有强制平台（用于 Handler 跳过分组检查）
func HasForcePlatform(c *gin.Context) bool {
	_, exists := c.Get(string(ContextKeyForcePlatform))
	return exists
}

func HasForcePlatformContext(c gatewayctx.GatewayContext) bool {
	if c == nil {
		return false
	}
	_, exists := c.Value(string(ContextKeyForcePlatform))
	return exists
}

// GetForcePlatformFromContext 从 gin.Context 获取强制平台
func GetForcePlatformFromContext(c *gin.Context) (string, bool) {
	return GetForcePlatformFromGatewayContext(gatewayctx.FromGin(c))
}

func GetForcePlatformFromGatewayContext(c gatewayctx.GatewayContext) (string, bool) {
	if c == nil {
		return "", false
	}
	value, exists := c.Value(string(ContextKeyForcePlatform))
	if !exists {
		return "", false
	}
	platform, ok := value.(string)
	return platform, ok
}

// ErrorResponse 标准错误响应结构
type ErrorResponse struct {
	Error   ErrorDetail `json:"error"`
	Code    string      `json:"-"`
	Message string      `json:"-"`
}

type ErrorDetail struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

func (r *ErrorResponse) UnmarshalJSON(data []byte) error {
	var payload struct {
		Error   ErrorDetail `json:"error"`
		Code    string      `json:"code"`
		Message string      `json:"message"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}
	r.Error = payload.Error
	r.Code = payload.Code
	r.Message = payload.Message
	if r.Code == "" {
		r.Code = r.Error.Code
	}
	if r.Message == "" {
		r.Message = r.Error.Message
	}
	return nil
}

func errorResponseType(code string) string {
	switch code {
	case "INSUFFICIENT_BALANCE", "API_KEY_QUOTA_EXHAUSTED", "USAGE_LIMIT_EXCEEDED":
		return "billing_error"
	case "INVALID_API_KEY", "API_KEY_REQUIRED", "API_KEY_DISABLED", "USER_INACTIVE", "USER_NOT_FOUND", "UNAUTHORIZED":
		return "authentication_error"
	case "API_KEY_EXPIRED", "SUBSCRIPTION_NOT_FOUND", "SUBSCRIPTION_INVALID", "ACCESS_DENIED":
		return "permission_error"
	default:
		return "api_error"
	}
}

// NewErrorResponse 创建错误响应
func NewErrorResponse(code, message string) ErrorResponse {
	return ErrorResponse{
		Error: ErrorDetail{
			Message: message,
			Type:    errorResponseType(code),
			Code:    code,
		},
		Code:    code,
		Message: message,
	}
}

// AbortWithError 中断请求并返回JSON错误
func AbortWithError(c *gin.Context, statusCode int, code, message string) {
	c.JSON(statusCode, NewErrorResponse(code, message))
	c.Abort()
}

// abortWithOpenAIQuotaError writes the OpenAI-compatible insufficient quota response.
func abortWithOpenAIQuotaError(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, gin.H{
		"error": gin.H{
			"message": message,
			"type":    "insufficient_quota",
			"param":   nil,
			"code":    "insufficient_quota",
		},
	})
	c.Abort()
}

// ──────────────────────────────────────────────────────────
// RequireGroupAssignment — 未分组 Key 拦截中间件
// ──────────────────────────────────────────────────────────

// GatewayErrorWriter 定义网关错误响应格式（不同协议使用不同格式）
type GatewayErrorWriter func(c *gin.Context, status int, message string)
type GatewayErrorWriterContext func(c gatewayctx.GatewayContext, status int, message string)

// AnthropicErrorWriter 按 Anthropic API 规范输出错误
func AnthropicErrorWriter(c *gin.Context, status int, message string) {
	AnthropicErrorWriterContext(gatewayctx.FromGin(c), status, message)
}

func AnthropicErrorWriterContext(c gatewayctx.GatewayContext, status int, message string) {
	if c == nil {
		return
	}
	c.WriteJSON(status, gin.H{
		"type":  "error",
		"error": gin.H{"type": "permission_error", "message": message},
	})
}

// GoogleErrorWriter 按 Google API 规范输出错误
func GoogleErrorWriter(c *gin.Context, status int, message string) {
	GoogleErrorWriterContext(gatewayctx.FromGin(c), status, message)
}

func GoogleErrorWriterContext(c gatewayctx.GatewayContext, status int, message string) {
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
}

// RequireGroupAssignment 检查 API Key 是否已分配到分组，
// 如果未分组且系统设置不允许未分组 Key 调度则返回 403。
func RequireGroupAssignment(settingService *service.SettingService, writeError GatewayErrorWriter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := gatewayctx.FromGin(c)
		if RequireGroupAssignmentContext(settingService, func(gc gatewayctx.GatewayContext, status int, message string) {
			writeError(c, status, message)
		}, ctx) {
			c.Next()
			return
		}
		c.Abort()
	}
}

func RequireGroupAssignmentContext(settingService *service.SettingService, writeError GatewayErrorWriterContext, c gatewayctx.GatewayContext) bool {
	apiKey, ok := GetAPIKeyFromGatewayContext(c)
	if !ok || apiKey.GroupID != nil {
		return true
	}
	if settingService.IsUngroupedKeySchedulingAllowed(c.Request().Context()) {
		return true
	}
	service.MarkOpsClientBusinessLimitedAny(c, service.OpsClientBusinessLimitedReasonAPIKeyGroupUnassigned)
	c.SetValue(ingressRejectReasonContextKey, IngressRejectGroupUnassigned)
	if writeError != nil {
		writeError(c, http.StatusForbidden, "API Key is not assigned to any group and cannot be used. Please contact the administrator to assign it to a group.")
	}
	return false
}
