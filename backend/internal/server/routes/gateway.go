package routes

import (
	"bytes"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	appmiddleware "github.com/Wei-Shaw/sub2api/internal/middleware"
	pkghttputil "github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const (
	GatewayConsumptionRateLimitKey    = "gateway-consumption"
	GatewayConsumptionRateLimitLimit  = 600
	GatewayConsumptionRateLimitWindow = time.Minute
	gatewayConsumptionRateLimitTag    = "rl_gateway_consumption"
)

// RegisterGatewayRoutes 注册 API 网关路由（Claude/OpenAI/Gemini 兼容）
func RegisterGatewayRoutes(
	r *gin.Engine,
	h *handler.Handlers,
	apiKeyAuth middleware.APIKeyAuthMiddleware,
	apiKeyService *service.APIKeyService,
	subscriptionService *service.SubscriptionService,
	opsService *service.OpsService,
	settingService *service.SettingService,
	compositeResolver *service.CompositeRouteResolver,
	cfg *config.Config,
	redisClient *redis.Client,
) {
	gatewayRL := gatewayConsumptionRateLimit(redisClient)
	bodyLimit := middleware.RequestBodyLimit(cfg.Gateway.MaxBodySize)
	textBodyLimit := bodyLimit
	soraMaxBodySize := cfg.Gateway.SoraMaxBodySize
	if soraMaxBodySize <= 0 {
		soraMaxBodySize = cfg.Gateway.MaxBodySize
	}
	soraBodyLimit := middleware.RequestBodyLimit(soraMaxBodySize)
	clientRequestID := middleware.ClientRequestID()
	opsErrorLogger := handler.OpsErrorLoggerMiddleware(opsService)
	endpointNorm := handler.InboundEndpointMiddleware()
	compositeTarget := compositeTargetPlatformMiddleware(compositeResolver)
	compositeGeminiTarget := compositeGeminiTargetPlatformMiddleware(compositeResolver)

	// 未分组 Key 拦截中间件（按协议格式区分错误响应）
	requireGroupAnthropic := middleware.RequireGroupAssignment(settingService, middleware.AnthropicErrorWriter)
	requireGroupGoogle := middleware.RequireGroupAssignment(settingService, middleware.GoogleErrorWriter)
	idempotency := middleware.GatewayIdempotencyMiddleware(redisClient)
	guardResponsesSubpath := func(next gin.HandlerFunc) gin.HandlerFunc {
		return func(c *gin.Context) {
			if !service.IsForwardableOpenAIResponsesRequestPath(c) {
				service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonLocalPolicyDenied)
				c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
					"error": gin.H{
						"type":    "not_found_error",
						"message": "Unsupported responses subpath",
					},
				})
				return
			}
			next(c)
		}
	}

	isOpenAIResponsesCompatibleGatewayPlatform := func(c *gin.Context) bool {
		switch getGroupPlatform(c) {
		case service.PlatformOpenAI, service.PlatformGrok:
			return true
		default:
			return false
		}
	}
	isOpenAIGatewayPlatform := func(c *gin.Context) bool {
		return getGroupPlatform(c) == service.PlatformOpenAI
	}
	countTokensHandler := func(c *gin.Context) {
		switch getGroupPlatform(c) {
		case service.PlatformOpenAI:
			h.OpenAIGateway.CountTokens(c)
		case service.PlatformGrok:
			h.OpenAIGateway.GrokCountTokens(c)
		default:
			h.Gateway.CountTokens(c)
		}
	}
	modelsHandler := func(c *gin.Context) {
		if isOpenAIGatewayPlatform(c) && c.Query("client_version") != "" {
			h.OpenAIGateway.CodexModels(c)
			return
		}
		h.Gateway.Models(c)
	}
	isOpenAIOnlyEndpointGatewayPlatform := func(c *gin.Context) bool {
		return getGroupPlatform(c) == service.PlatformOpenAI
	}
	imagesHandler := func(c *gin.Context) {
		switch getGroupPlatform(c) {
		case service.PlatformOpenAI:
			h.OpenAIGateway.Images(c)
		case service.PlatformGrok:
			h.OpenAIGateway.GrokImages(c)
		default:
			service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonLocalFeatureGate)
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"type":    "not_found_error",
					"message": "Images API is not supported for this platform",
				},
			})
		}
	}
	videoGenerationHandler := func(c *gin.Context) {
		if getGroupPlatform(c) == service.PlatformGrok {
			h.OpenAIGateway.GrokVideoGeneration(c)
			return
		}
		service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonLocalFeatureGate)
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"type":    "not_found_error",
				"message": "Videos API is not supported for this platform",
			},
		})
	}
	videoStatusHandler := func(c *gin.Context) {
		// Video status requests do not carry a model, so composite groups cannot
		// be resolved by compositeTargetPlatformMiddleware. Route them through
		// the Grok handler and let scheduler/account selection enforce capacity.
		if getGroupPlatform(c) == service.PlatformGrok || getGroupPlatform(c) == service.PlatformComposite {
			h.OpenAIGateway.GrokVideoStatus(c)
			return
		}
		service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonLocalFeatureGate)
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"type":    "not_found_error",
				"message": "Videos API is not supported for this platform",
			},
		})
	}
	videoContentHandler := func(c *gin.Context) {
		// Video content requests do not carry a model, so composite groups cannot
		// be resolved by compositeTargetPlatformMiddleware. Route them through
		// the Grok handler just like video status lookups.
		if getGroupPlatform(c) == service.PlatformGrok || getGroupPlatform(c) == service.PlatformComposite {
			h.OpenAIGateway.GrokVideoContent(c)
			return
		}
		service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonLocalFeatureGate)
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"type":    "not_found_error",
				"message": "Videos API is not supported for this platform",
			},
		})
	}
	videoEditHandler := func(c *gin.Context) {
		if getGroupPlatform(c) == service.PlatformGrok {
			h.OpenAIGateway.GrokVideoEdit(c)
			return
		}
		service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonLocalFeatureGate)
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"type": "not_found_error", "message": "Videos API is not supported for this platform"}})
	}
	videoExtensionHandler := func(c *gin.Context) {
		if getGroupPlatform(c) == service.PlatformGrok {
			h.OpenAIGateway.GrokVideoExtension(c)
			return
		}
		service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonLocalFeatureGate)
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"type": "not_found_error", "message": "Videos API is not supported for this platform"}})
	}
	// /responses/*subpath 的子路径会被转发到上游同名端点之后，因此在入口就拒掉
	// 不可转发的子路径，不让它进入调度与转发流程。可转发的判定见
	// service.IsForwardableOpenAIResponsesRequestPath 及 upstream_path_guard.go。
	guardResponsesSubpath = func(next gin.HandlerFunc) gin.HandlerFunc {
		return func(c *gin.Context) {
			if !service.IsForwardableOpenAIResponsesRequestPath(c) {
				service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonLocalPolicyDenied)
				c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
					"error": gin.H{
						"type":    "not_found_error",
						"message": "Unsupported responses subpath",
					},
				})
				return
			}
			next(c)
		}
	}

	// API网关（Claude API兼容）
	gateway := r.Group("/v1")
	gateway.Use(gatewayRL)
	gateway.Use(bodyLimit)
	gateway.Use(clientRequestID)
	gateway.Use(opsErrorLogger)
	gateway.Use(endpointNorm)
	gateway.Use(gin.HandlerFunc(apiKeyAuth))
	gateway.Use(idempotency)
	gateway.Use(compositeTarget)
	gateway.Use(requireGroupAnthropic)
	{
		// /v1/messages: auto-route based on group platform
		gateway.POST("/messages", func(c *gin.Context) {
			apiKey, _ := middleware.GetAPIKeyFromContext(c)
			if shouldUseOpenAIMessagesDispatch(apiKey) || compositeResolvedToOpenAICompatible(c) {
				h.OpenAIGateway.Messages(c)
				return
			}
			h.Gateway.Messages(c)
		})
		gateway.POST("/messages/count_tokens", openAICountTokensDispatch(h))
		gateway.GET("/models", modelsHandler)
		gateway.GET("/usage", h.Gateway.Usage)
		// Live API（OpenAI 实时语音）；按分组 allow_live 开关在 handler 层限制
		gateway.POST("/live", h.OpenAIGateway.Live)
		gateway.GET("/live/:call_id", h.OpenAIGateway.LiveSideband)
		// OpenAI Responses API: auto-route based on group platform
		gateway.POST("/responses", func(c *gin.Context) {
			if isOpenAIResponsesCompatibleGatewayPlatform(c) {
				h.OpenAIGateway.Responses(c)
				return
			}
			h.Gateway.Responses(c)
		})
		gateway.POST("/responses/*subpath", guardResponsesSubpath(func(c *gin.Context) {
			if isOpenAIResponsesCompatibleGatewayPlatform(c) {
				h.OpenAIGateway.Responses(c)
				return
			}
			h.Gateway.Responses(c)
		}))
		gateway.POST("/alpha/search", textBodyLimit, h.OpenAIGateway.AlphaSearch)
		gateway.GET("/responses", func(c *gin.Context) {
			h.OpenAIGateway.ResponsesWebSocket(c)
		})
		// OpenAI Chat Completions API: auto-route based on group platform
		gateway.POST("/chat/completions", func(c *gin.Context) {
			if isOpenAIResponsesCompatibleGatewayPlatform(c) {
				h.OpenAIGateway.ChatCompletions(c)
				return
			}
			h.Gateway.ChatCompletions(c)
		})
		gateway.POST("/embeddings", textBodyLimit, func(c *gin.Context) {
			if !isOpenAIOnlyEndpointGatewayPlatform(c) {
				service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonLocalFeatureGate)
				c.JSON(http.StatusNotFound, gin.H{
					"error": gin.H{
						"type":    "not_found_error",
						"message": "Embeddings API is not supported for this platform",
					},
				})
				return
			}
			h.OpenAIGateway.Embeddings(c)
		})
		gateway.POST("/images/generations", imagesHandler)
		gateway.POST("/images/edits", imagesHandler)
		gateway.POST("/images/generations/async", h.AsyncImage.Submit)
		gateway.POST("/images/edits/async", h.AsyncImage.Submit)
		gateway.GET("/images/tasks/:task_id", h.AsyncImage.Get)
		gateway.POST("/images/batches", h.BatchImage.Submit)
		gateway.GET("/images/batches", h.BatchImage.List)
		gateway.GET("/images/batches/models", h.BatchImage.Models)
		gateway.GET("/images/batches/:id", h.BatchImage.Get)
		gateway.GET("/images/batches/:id/items", h.BatchImage.Items)
		gateway.GET("/images/batches/:id/items/:custom_id/content", h.BatchImage.ItemContent)
		gateway.GET("/images/batches/:id/download", h.BatchImage.Download)
		gateway.POST("/images/batches/:id/cancel", h.BatchImage.Cancel)
		gateway.DELETE("/images/batches/:id", h.BatchImage.DeleteRecord)
		gateway.DELETE("/images/batches/:id/outputs", h.BatchImage.DeleteOutputs)
		gateway.POST("/videos/generations", videoGenerationHandler)
		gateway.POST("/videos/edits", videoEditHandler)
		gateway.POST("/videos/extensions", videoExtensionHandler)
		gateway.GET("/videos/:request_id", videoStatusHandler)
		gateway.GET("/videos/:request_id/content", videoContentHandler)
	}

	// Gemini 原生 API 兼容层（Gemini SDK/CLI 直连）
	// Billing metadata intentionally bypasses the group-assignment gate: simple
	// mode keys have no group and the handler returns its documented 404.
	r.GET("/v1/sub2api/billing", gatewayRL, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), h.Gateway.KeyBillingInfo)

	gemini := r.Group("/v1beta")
	gemini.Use(gatewayRL)
	gemini.Use(bodyLimit)
	gemini.Use(clientRequestID)
	gemini.Use(opsErrorLogger)
	gemini.Use(endpointNorm)
	gemini.Use(middleware.APIKeyAuthWithSubscriptionGoogle(apiKeyService, subscriptionService, cfg))
	gemini.Use(compositeGeminiTarget)
	gemini.Use(requireGroupGoogle)
	{
		gemini.GET("/models", h.Gateway.GeminiV1BetaListModels)
		gemini.GET("/models/:model", h.Gateway.GeminiV1BetaGetModel)
		// Gin treats ":" as a param marker, but Gemini uses "{model}:{action}" in the same segment.
		gemini.POST("/models/*modelAction", h.Gateway.GeminiV1BetaModels)
	}

	// OpenAI Responses API（不带v1前缀的别名）— auto-route based on group platform
	responsesHandler := func(c *gin.Context) {
		if isOpenAIResponsesCompatibleGatewayPlatform(c) {
			h.OpenAIGateway.Responses(c)
			return
		}
		h.Gateway.Responses(c)
	}
	r.POST("/responses", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), compositeTarget, requireGroupAnthropic, responsesHandler)
	r.POST("/responses/*subpath", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), compositeTarget, requireGroupAnthropic, guardResponsesSubpath(responsesHandler))
	r.POST("/alpha/search", textBodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), compositeTarget, requireGroupAnthropic, h.OpenAIGateway.AlphaSearch)
	r.GET("/responses", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), compositeTarget, requireGroupAnthropic, func(c *gin.Context) {
		h.OpenAIGateway.ResponsesWebSocket(c)
	})
	r.GET("/models", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, modelsHandler)
	r.POST("/messages/count_tokens", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), compositeTarget, requireGroupAnthropic, countTokensHandler)
	r.POST("/chat/completions", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), compositeTarget, requireGroupAnthropic, func(c *gin.Context) {
		if isOpenAIResponsesCompatibleGatewayPlatform(c) {
			h.OpenAIGateway.ChatCompletions(c)
			return
		}
		h.Gateway.ChatCompletions(c)
	})
	r.POST("/embeddings", textBodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), compositeTarget, requireGroupAnthropic, func(c *gin.Context) {
		if !isOpenAIOnlyEndpointGatewayPlatform(c) {
			service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonLocalFeatureGate)
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"type": "not_found_error", "message": "Embeddings API is not supported for this platform"}})
			return
		}
		h.OpenAIGateway.Embeddings(c)
	})
	r.POST("/images/generations", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), compositeTarget, requireGroupAnthropic, imagesHandler)
	r.POST("/images/edits", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), compositeTarget, requireGroupAnthropic, imagesHandler)
	r.POST("/images/generations/async", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), compositeTarget, requireGroupAnthropic, h.AsyncImage.Submit)
	r.POST("/images/edits/async", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), compositeTarget, requireGroupAnthropic, h.AsyncImage.Submit)
	r.GET("/images/tasks/:task_id", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), compositeTarget, requireGroupAnthropic, h.AsyncImage.Get)
	r.POST("/videos/generations", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), compositeTarget, requireGroupAnthropic, videoGenerationHandler)
	r.POST("/videos/edits", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), compositeTarget, requireGroupAnthropic, videoEditHandler)
	r.POST("/videos/extensions", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), compositeTarget, requireGroupAnthropic, videoExtensionHandler)
	r.GET("/videos/:request_id", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), compositeTarget, requireGroupAnthropic, videoStatusHandler)
	r.GET("/videos/:request_id/content", bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), compositeTarget, requireGroupAnthropic, videoContentHandler)

	codexDirect := r.Group("/backend-api/codex")
	codexDirect.Use(gatewayRL, bodyLimit, clientRequestID, opsErrorLogger, endpointNorm, gin.HandlerFunc(apiKeyAuth), compositeTarget, requireGroupAnthropic)
	{
		// Live API（OpenAI 实时语音）
		codexDirect.POST("/realtime/calls", h.OpenAIGateway.Live)
		codexDirect.GET("/:call_id", h.OpenAIGateway.LiveSideband)
		codexDirect.POST("/responses", responsesHandler)
		codexDirect.POST("/responses/*subpath", guardResponsesSubpath(responsesHandler))
		codexDirect.POST("/alpha/search", textBodyLimit, h.OpenAIGateway.AlphaSearch)
		codexDirect.GET("/responses", func(c *gin.Context) {
			h.OpenAIGateway.ResponsesWebSocket(c)
		})
		codexDirect.GET("/models", h.OpenAIGateway.CodexModels)
	}
	// Claude Code bootstrap / telemetry compatibility endpoints.
	r.GET("/api/claude_cli/bootstrap", gatewayRL, clientRequestID, opsErrorLogger, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, h.Gateway.ClaudeBootstrap)
	r.GET("/api/claude_code/organizations/metrics_enabled", gatewayRL, clientRequestID, opsErrorLogger, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, h.Gateway.ClaudeMetricsEnabled)
	r.GET("/api/claude_code/settings", gatewayRL, clientRequestID, opsErrorLogger, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, h.Gateway.ClaudeManagedSettings)
	r.GET("/api/claude_code/policy_limits", gatewayRL, clientRequestID, opsErrorLogger, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, h.Gateway.ClaudePolicyLimits)
	r.GET("/api/claude_code/user_settings", gatewayRL, clientRequestID, opsErrorLogger, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, h.Gateway.ClaudeUserSettings)
	r.PUT("/api/claude_code/user_settings", gatewayRL, bodyLimit, clientRequestID, opsErrorLogger, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, h.Gateway.ClaudeUpdateUserSettings)

	// Antigravity 模型列表
	r.GET("/antigravity/models", gatewayRL, gin.HandlerFunc(apiKeyAuth), requireGroupAnthropic, h.Gateway.AntigravityModels)

	// Antigravity 专用路由（仅使用 antigravity 账户，不混合调度）
	antigravityV1 := r.Group("/antigravity/v1")
	antigravityV1.Use(gatewayRL)
	antigravityV1.Use(bodyLimit)
	antigravityV1.Use(clientRequestID)
	antigravityV1.Use(opsErrorLogger)
	antigravityV1.Use(endpointNorm)
	antigravityV1.Use(middleware.ForcePlatform(service.PlatformAntigravity))
	antigravityV1.Use(gin.HandlerFunc(apiKeyAuth))
	antigravityV1.Use(requireGroupAnthropic)
	{
		antigravityV1.POST("/messages", h.Gateway.Messages)
		antigravityV1.POST("/messages/count_tokens", h.Gateway.CountTokens)
		antigravityV1.GET("/models", h.Gateway.AntigravityModels)
		antigravityV1.GET("/usage", h.Gateway.Usage)
	}

	antigravityV1Beta := r.Group("/antigravity/v1beta")
	antigravityV1Beta.Use(gatewayRL)
	antigravityV1Beta.Use(bodyLimit)
	antigravityV1Beta.Use(clientRequestID)
	antigravityV1Beta.Use(opsErrorLogger)
	antigravityV1Beta.Use(endpointNorm)
	antigravityV1Beta.Use(middleware.ForcePlatform(service.PlatformAntigravity))
	antigravityV1Beta.Use(middleware.APIKeyAuthWithSubscriptionGoogle(apiKeyService, subscriptionService, cfg))
	antigravityV1Beta.Use(requireGroupGoogle)
	{
		antigravityV1Beta.GET("/models", h.Gateway.GeminiV1BetaListModels)
		antigravityV1Beta.GET("/models/:model", h.Gateway.GeminiV1BetaGetModel)
		antigravityV1Beta.POST("/models/*modelAction", h.Gateway.GeminiV1BetaModels)
	}

	// Sora 专用路由（强制使用 sora 平台）
	soraV1 := r.Group("/sora/v1")
	soraV1.Use(gatewayRL)
	soraV1.Use(soraBodyLimit)
	soraV1.Use(clientRequestID)
	soraV1.Use(opsErrorLogger)
	soraV1.Use(endpointNorm)
	soraV1.Use(middleware.ForcePlatform(service.PlatformSora))
	soraV1.Use(gin.HandlerFunc(apiKeyAuth))
	soraV1.Use(requireGroupAnthropic)
	{
		soraV1.POST("/chat/completions", h.SoraGateway.ChatCompletions)
		soraV1.GET("/models", h.Gateway.Models)
	}

	// Sora 媒体代理（可选 API Key 验证）
	if cfg.Gateway.SoraMediaRequireAPIKey {
		r.GET("/sora/media/*filepath", gatewayRL, gin.HandlerFunc(apiKeyAuth), h.SoraGateway.MediaProxy)
	} else {
		r.GET("/sora/media/*filepath", h.SoraGateway.MediaProxy)
	}
	// Sora 媒体代理（签名 URL，无需 API Key）
	r.GET("/sora/media-signed/*filepath", h.SoraGateway.MediaProxySigned)
}

func gatewayConsumptionRateLimit(redisClient *redis.Client) gin.HandlerFunc {
	if redisClient == nil {
		return func(c *gin.Context) {
			c.Next()
		}
	}
	return appmiddleware.NewRateLimiter(redisClient).LimitWithOptions(
		GatewayConsumptionRateLimitKey,
		GatewayConsumptionRateLimitLimit,
		GatewayConsumptionRateLimitWindow,
		appmiddleware.RateLimitOptions{FailureMode: appmiddleware.RateLimitFailOpen},
	)
}

func ExecutableGatewayRoutes(h *handler.Handlers) []gatewayctx.RouteDef {
	if h == nil || h.Gateway == nil {
		return nil
	}
	return withGatewayConsumptionRateLimit([]gatewayctx.RouteDef{
		{
			Method:  http.MethodGet,
			Path:    "/api/claude_cli/bootstrap",
			Handler: h.Gateway.ClaudeBootstrapGateway,
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"client_request_id",
				"standard_api_key_auth",
				"require_group_anthropic",
			},
		},
		{
			Method:  http.MethodGet,
			Path:    "/api/claude_code/organizations/metrics_enabled",
			Handler: h.Gateway.ClaudeMetricsEnabledGateway,
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"client_request_id",
				"standard_api_key_auth",
				"require_group_anthropic",
			},
		},
		{
			Method:  http.MethodGet,
			Path:    "/api/claude_code/settings",
			Handler: h.Gateway.ClaudeManagedSettingsGateway,
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"client_request_id",
				"standard_api_key_auth",
				"require_group_anthropic",
			},
		},
		{
			Method:  http.MethodGet,
			Path:    "/api/claude_code/policy_limits",
			Handler: h.Gateway.ClaudePolicyLimitsGateway,
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"client_request_id",
				"standard_api_key_auth",
				"require_group_anthropic",
			},
		},
		{
			Method:  http.MethodGet,
			Path:    "/api/claude_code/user_settings",
			Handler: h.Gateway.ClaudeUserSettingsGateway,
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"client_request_id",
				"standard_api_key_auth",
				"require_group_anthropic",
			},
		},
		{
			Method:  http.MethodPut,
			Path:    "/api/claude_code/user_settings",
			Handler: h.Gateway.ClaudeUpdateUserSettingsGateway,
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"gateway_body_limit",
				"client_request_id",
				"standard_api_key_auth",
				"require_group_anthropic",
			},
		},
		{
			Method:  http.MethodPost,
			Path:    "/v1/messages",
			Handler: openAIMessagesDispatchGateway(h),
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"gateway_body_limit",
				"client_request_id",
				"inbound_endpoint",
				"standard_api_key_auth",
				"require_group_anthropic",
			},
		},
		{
			Method:  http.MethodPost,
			Path:    "/v1/messages/count_tokens",
			Handler: openAICountTokensDispatchGateway(h),
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"gateway_body_limit",
				"client_request_id",
				"inbound_endpoint",
				"standard_api_key_auth",
				"require_group_anthropic",
			},
		},
		{
			Method:  http.MethodGet,
			Path:    "/v1/models",
			Handler: h.Gateway.ModelsGateway,
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"client_request_id",
				"standard_api_key_auth",
				"require_group_anthropic",
			},
		},
		{
			Method:  http.MethodPost,
			Path:    "/v1/chat/completions",
			Handler: h.OpenAIGateway.ChatCompletionsGateway,
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"gateway_body_limit",
				"client_request_id",
				"inbound_endpoint",
				"standard_api_key_auth",
				"require_group_anthropic",
			},
		},
		{
			Method:  http.MethodPost,
			Path:    "/v1/images/generations",
			Handler: openAIImagesDispatchGateway(h),
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"gateway_body_limit",
				"client_request_id",
				"inbound_endpoint",
				"standard_api_key_auth",
				"require_group_anthropic",
			},
		},
		{
			Method:  http.MethodPost,
			Path:    "/v1/images/edits",
			Handler: openAIImagesDispatchGateway(h),
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"gateway_body_limit",
				"client_request_id",
				"inbound_endpoint",
				"standard_api_key_auth",
				"require_group_anthropic",
			},
		},
		{
			Method:  http.MethodGet,
			Path:    "/v1/responses",
			Handler: h.OpenAIGateway.ResponsesWebSocketGateway,
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"gateway_body_limit",
				"client_request_id",
				"inbound_endpoint",
				"standard_api_key_auth",
				"require_group_anthropic",
			},
		},
		{
			Method:  http.MethodPost,
			Path:    "/v1/responses",
			Handler: h.OpenAIGateway.ResponsesGateway,
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"gateway_body_limit",
				"client_request_id",
				"inbound_endpoint",
				"standard_api_key_auth",
				"require_group_anthropic",
			},
		},
		{
			Method:  http.MethodPost,
			Path:    "/v1/responses/*subpath",
			Handler: h.OpenAIGateway.ResponsesGateway,
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"gateway_body_limit",
				"client_request_id",
				"inbound_endpoint",
				"standard_api_key_auth",
				"require_group_anthropic",
			},
		},
		{
			Method:  http.MethodPost,
			Path:    "/chat/completions",
			Handler: h.OpenAIGateway.ChatCompletionsGateway,
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"gateway_body_limit",
				"client_request_id",
				"inbound_endpoint",
				"standard_api_key_auth",
				"require_group_anthropic",
			},
		},
		{
			Method:  http.MethodPost,
			Path:    "/images/generations",
			Handler: openAIImagesDispatchGateway(h),
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"gateway_body_limit",
				"client_request_id",
				"inbound_endpoint",
				"standard_api_key_auth",
				"require_group_anthropic",
			},
		},
		{
			Method:  http.MethodPost,
			Path:    "/images/edits",
			Handler: openAIImagesDispatchGateway(h),
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"gateway_body_limit",
				"client_request_id",
				"inbound_endpoint",
				"standard_api_key_auth",
				"require_group_anthropic",
			},
		},
		{
			Method:  http.MethodGet,
			Path:    "/responses",
			Handler: h.OpenAIGateway.ResponsesWebSocketGateway,
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"gateway_body_limit",
				"client_request_id",
				"inbound_endpoint",
				"standard_api_key_auth",
				"require_group_anthropic",
			},
		},
		{
			Method:  http.MethodPost,
			Path:    "/responses",
			Handler: h.OpenAIGateway.ResponsesGateway,
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"gateway_body_limit",
				"client_request_id",
				"inbound_endpoint",
				"standard_api_key_auth",
				"require_group_anthropic",
			},
		},
		{
			Method:  http.MethodPost,
			Path:    "/responses/*subpath",
			Handler: h.OpenAIGateway.ResponsesGateway,
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"gateway_body_limit",
				"client_request_id",
				"inbound_endpoint",
				"standard_api_key_auth",
				"require_group_anthropic",
			},
		},
		{
			Method:  http.MethodGet,
			Path:    "/v1beta/models",
			Handler: h.Gateway.GeminiV1BetaListModelsGateway,
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"client_request_id",
				"inbound_endpoint",
				"google_api_key_auth",
				"require_group_google",
			},
		},
		{
			Method:  http.MethodGet,
			Path:    "/v1beta/models/:model",
			Handler: h.Gateway.GeminiV1BetaGetModelGateway,
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"client_request_id",
				"inbound_endpoint",
				"google_api_key_auth",
				"require_group_google",
			},
		},
		{
			Method:  http.MethodPost,
			Path:    "/v1beta/models/*modelAction",
			Handler: h.Gateway.GeminiV1BetaModelsGateway,
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"gateway_body_limit",
				"client_request_id",
				"inbound_endpoint",
				"google_api_key_auth",
				"require_group_google",
			},
		},
		{
			Method:  http.MethodGet,
			Path:    "/antigravity/models",
			Handler: h.Gateway.AntigravityModelsGateway,
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"standard_api_key_auth",
				"require_group_anthropic",
			},
		},
		{
			Method:  http.MethodGet,
			Path:    "/antigravity/v1beta/models",
			Handler: h.Gateway.GeminiV1BetaListModelsGateway,
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"client_request_id",
				"inbound_endpoint",
				"force_platform_antigravity",
				"google_api_key_auth",
				"require_group_google",
			},
		},
		{
			Method:  http.MethodGet,
			Path:    "/antigravity/v1beta/models/:model",
			Handler: h.Gateway.GeminiV1BetaGetModelGateway,
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"client_request_id",
				"inbound_endpoint",
				"force_platform_antigravity",
				"google_api_key_auth",
				"require_group_google",
			},
		},
		{
			Method:  http.MethodPost,
			Path:    "/antigravity/v1beta/models/*modelAction",
			Handler: h.Gateway.GeminiV1BetaModelsGateway,
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"gateway_body_limit",
				"client_request_id",
				"inbound_endpoint",
				"force_platform_antigravity",
				"google_api_key_auth",
				"require_group_google",
			},
		},
		{
			Method: http.MethodGet,
			Path:   "/sora/media/*filepath",
			Handler: func(c gatewayctx.GatewayContext) {
				if h.SoraGateway == nil {
					c.SetStatus(http.StatusNotFound)
					return
				}
				h.SoraGateway.MediaProxyGateway(c)
			},
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"client_request_id",
				"standard_api_key_auth",
				"require_group_anthropic",
			},
		},
		{
			Method: http.MethodGet,
			Path:   "/sora/media-signed/*filepath",
			Handler: func(c gatewayctx.GatewayContext) {
				if h.SoraGateway == nil {
					c.SetStatus(http.StatusNotFound)
					return
				}
				h.SoraGateway.MediaProxySignedGateway(c)
			},
			Middleware: []string{
				"request_logger",
				"cors",
				"security_headers",
				"client_request_id",
			},
		},
	})
}

func withGatewayConsumptionRateLimit(defs []gatewayctx.RouteDef) []gatewayctx.RouteDef {
	for i := range defs {
		if routeRequiresGatewayAPIKey(defs[i]) {
			defs[i].Middleware = append([]string{gatewayConsumptionRateLimitTag}, defs[i].Middleware...)
		}
	}
	return defs
}

func routeRequiresGatewayAPIKey(def gatewayctx.RouteDef) bool {
	for _, tag := range def.Middleware {
		if tag == "standard_api_key_auth" || tag == "google_api_key_auth" {
			return true
		}
	}
	return false
}

func openAIMessagesDispatchGateway(h *handler.Handlers) gatewayctx.HandlerFunc {
	return func(c gatewayctx.GatewayContext) {
		if c == nil || h == nil {
			writeGatewayDispatchUnavailable(c)
			return
		}
		apiKey, ok := middleware.GetAPIKeyFromGatewayContext(c)
		if !ok || apiKey == nil {
			if h.OpenAIGateway != nil {
				h.OpenAIGateway.MessagesGateway(c)
			}
			return
		}
		if shouldUseOpenAIMessagesDispatch(apiKey) {
			if h.OpenAIGateway != nil {
				h.OpenAIGateway.MessagesGateway(c)
			}
			return
		}
		if h.Gateway != nil {
			h.Gateway.MessagesGateway(c)
			return
		}
		writeGatewayDispatchUnavailable(c)
	}
}

func openAICountTokensDispatchGateway(h *handler.Handlers) gatewayctx.HandlerFunc {
	return func(c gatewayctx.GatewayContext) {
		if c == nil || h == nil {
			writeGatewayDispatchUnavailable(c)
			return
		}
		apiKey, ok := middleware.GetAPIKeyFromGatewayContext(c)
		if !ok || apiKey == nil {
			if h.Gateway != nil {
				h.Gateway.CountTokensGateway(c)
				return
			}
			writeGatewayDispatchUnavailable(c)
			return
		}
		if shouldUseOpenAIMessagesDispatch(apiKey) {
			c.WriteJSON(http.StatusNotFound, gin.H{
				"type": "error",
				"error": gin.H{
					"type":    "not_found_error",
					"message": "Token counting is not supported for this platform",
				},
			})
			return
		}
		if h.Gateway != nil {
			h.Gateway.CountTokensGateway(c)
			return
		}
		writeGatewayDispatchUnavailable(c)
	}
}

func openAICountTokensDispatch(h *handler.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey, _ := middleware.GetAPIKeyFromContext(c)
		platform := getGroupPlatform(c)
		switch {
		case platform == service.PlatformGrok:
			h.OpenAIGateway.GrokCountTokens(c)
		case platform == service.PlatformOpenAI:
			h.OpenAIGateway.CountTokens(c)
		case shouldUseOpenAIMessagesDispatch(apiKey):
			c.JSON(http.StatusNotFound, gin.H{
				"type": "error",
				"error": gin.H{
					"type":    "not_found_error",
					"message": "Token counting is not supported for this platform",
				},
			})
		default:
			h.Gateway.CountTokens(c)
		}
	}
}

func shouldUseOpenAIMessagesDispatch(apiKey *service.APIKey) bool {
	if apiKey == nil || apiKey.Group == nil {
		return false
	}
	group := apiKey.Group
	return group.Platform == service.PlatformOpenAI ||
		(group.Platform == service.PlatformAnthropic && group.AllowMessagesDispatch)
}

func openAIImagesDispatch(h *handler.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
		switch getGroupPlatform(c) {
		case service.PlatformOpenAI:
			h.OpenAIGateway.Images(c)
		case service.PlatformGrok:
			h.OpenAIGateway.GrokImages(c)
		default:
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"type":    "not_found_error",
					"message": "Images API is not supported for this platform",
				},
			})
		}
	}
}

func grokVideoDispatch(next gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if getGroupPlatform(c) != service.PlatformGrok {
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"type":    "not_found_error",
					"message": "Videos API is not supported for this platform",
				},
			})
			return
		}
		next(c)
	}
}

// grokVideoLookupDispatch also admits composite groups: video status/content
// requests do not carry a model, so compositeTargetPlatformMiddleware cannot
// resolve them. Route them through the Grok handler and let scheduler/account
// selection enforce capacity (upstream v0.1.165 behavior).
func grokVideoLookupDispatch(next gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		platform := getGroupPlatform(c)
		if platform != service.PlatformGrok && platform != service.PlatformComposite {
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"type":    "not_found_error",
					"message": "Videos API is not supported for this platform",
				},
			})
			return
		}
		next(c)
	}
}

func asyncImageSubmit(h *handler.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
		if h == nil || h.AsyncImage == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{"message": "Async image service is unavailable"}})
			return
		}
		h.AsyncImage.Submit(c)
	}
}

func asyncImageGet(h *handler.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
		if h == nil || h.AsyncImage == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{"message": "Async image service is unavailable"}})
			return
		}
		h.AsyncImage.Get(c)
	}
}

func batchImageSubmit(h *handler.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
		if h == nil || h.BatchImage == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{"message": "Batch image service is unavailable"}})
			return
		}
		h.BatchImage.Submit(c)
	}
}

func batchImageCancel(h *handler.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
		if h == nil || h.BatchImage == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{"message": "Batch image service is unavailable"}})
			return
		}
		h.BatchImage.Cancel(c)
	}
}

func batchImageGet(h *handler.Handlers) gin.HandlerFunc {
	return func(c *gin.Context) {
		if h == nil || h.BatchImage == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{"message": "Batch image service is unavailable"}})
			return
		}
		h.BatchImage.Get(c)
	}
}

func openAIImagesDispatchGateway(h *handler.Handlers) gatewayctx.HandlerFunc {
	return func(c gatewayctx.GatewayContext) {
		if c == nil || h == nil {
			writeGatewayDispatchUnavailable(c)
			return
		}
		apiKey, ok := middleware.GetAPIKeyFromGatewayContext(c)
		if !ok || apiKey == nil {
			if h.OpenAIGateway != nil {
				h.OpenAIGateway.ImagesGateway(c)
				return
			}
			writeGatewayDispatchUnavailable(c)
			return
		}
		if apiKey.Group != nil && apiKey.Group.Platform != service.PlatformOpenAI {
			c.WriteJSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"type":    "not_found_error",
					"message": "Images API is not supported for this platform",
				},
			})
			return
		}
		if h.OpenAIGateway != nil {
			h.OpenAIGateway.ImagesGateway(c)
			return
		}
		writeGatewayDispatchUnavailable(c)
	}
}

func writeGatewayDispatchUnavailable(c gatewayctx.GatewayContext) {
	if c == nil {
		return
	}
	c.WriteJSON(http.StatusServiceUnavailable, gin.H{
		"error": gin.H{
			"type":    "api_error",
			"message": "Service temporarily unavailable",
		},
	})
}

// getGroupPlatform extracts the group platform from the API Key stored in context.
func getGroupPlatform(c *gin.Context) string {
	apiKey, ok := middleware.GetAPIKeyFromContext(c)
	if !ok || apiKey.Group == nil {
		return ""
	}
	if apiKey.Group.Platform == service.PlatformComposite {
		if platform, ok := service.ResolvedTargetPlatformFromContext(c.Request.Context()); ok {
			return platform
		}
	}
	return apiKey.Group.Platform
}

// compositeResolvedToOpenAICompatible reports whether the request belongs to a
// composite group whose model routing resolved to an OpenAI-compatible target
// platform (openai/grok). Used to extend the production messages dispatch to
// composite groups without changing behavior for concrete platforms.
func compositeResolvedToOpenAICompatible(c *gin.Context) bool {
	apiKey, ok := middleware.GetAPIKeyFromContext(c)
	if !ok || apiKey == nil || apiKey.Group == nil || apiKey.Group.Platform != service.PlatformComposite {
		return false
	}
	switch getGroupPlatform(c) {
	case service.PlatformOpenAI, service.PlatformGrok:
		return true
	default:
		return false
	}
}

// compositeOpenAIOnlyGate rejects composite-group requests to OpenAI-only
// endpoints (e.g. embeddings) unless the composite route resolved to the
// openai target platform. Non-composite groups pass through unchanged and keep
// the production dispatch behavior.
func compositeOpenAIOnlyGate(c *gin.Context) {
	apiKey, ok := middleware.GetAPIKeyFromContext(c)
	if ok && apiKey != nil && apiKey.Group != nil && apiKey.Group.Platform == service.PlatformComposite &&
		getGroupPlatform(c) != service.PlatformOpenAI {
		service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonLocalFeatureGate)
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"type":    "not_found_error",
				"message": "Embeddings API is not supported for this platform",
			},
		})
		c.Abort()
		return
	}
	c.Next()
}

func compositeTargetPlatformMiddleware(resolver *service.CompositeRouteResolver) gin.HandlerFunc {
	if resolver == nil {
		resolver = service.NewCompositeRouteResolver(nil)
	}
	return func(c *gin.Context) {
		apiKey, ok := middleware.GetAPIKeyFromContext(c)
		if !ok || apiKey == nil || apiKey.Group == nil || apiKey.Group.Platform != service.PlatformComposite {
			c.Next()
			return
		}
		if c.Request == nil || c.Request.Method == http.MethodGet {
			c.Next()
			return
		}

		body, err := pkghttputil.ReadRequestBodyWithPrealloc(c.Request)
		if err != nil {
			status := http.StatusBadRequest
			message := "Failed to read request body"
			var maxErr *http.MaxBytesError
			if errors.As(err, &maxErr) {
				status = http.StatusRequestEntityTooLarge
				message = "Request body is too large"
			}
			c.JSON(status, gin.H{"error": gin.H{"type": "invalid_request_error", "message": message}})
			c.Abort()
			return
		}

		model := compositeRequestModelFromBody(c.GetHeader("Content-Type"), body)
		if model != "" {
			decision, err := resolver.Resolve(c.Request.Context(), apiKey.Group.ID, model, compositeRouteEndpointForPath(c.Request.URL.Path))
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"type": "server_error", "message": "Failed to resolve composite model route"}})
				c.Abort()
				return
			}
			if decision.Matched {
				c.Request = c.Request.WithContext(service.WithCompositeRouteDecision(c.Request.Context(), decision))
				if upstreamModel := strings.TrimSpace(decision.UpstreamModel); upstreamModel != "" && upstreamModel != model && gjson.ValidBytes(body) {
					if rewritten, rewriteErr := sjson.SetBytes(body, "model", upstreamModel); rewriteErr == nil {
						body = rewritten
					}
				}
			}
		}
		resetRequestBody(c, body)
		c.Next()
	}
}

func compositeRequestModelFromBody(contentType string, body []byte) string {
	if model := strings.TrimSpace(gjson.GetBytes(body, "model").String()); model != "" {
		return model
	}
	return compositeMultipartModelFromBody(contentType, body)
}

func compositeMultipartModelFromBody(contentType string, body []byte) string {
	mediaType, params, err := mime.ParseMediaType(strings.TrimSpace(contentType))
	if err != nil || !strings.EqualFold(mediaType, "multipart/form-data") {
		return ""
	}
	boundary := strings.TrimSpace(params["boundary"])
	if boundary == "" {
		return ""
	}
	reader := multipart.NewReader(bytes.NewReader(body), boundary)
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			return ""
		}
		if err != nil {
			return ""
		}
		if part.FormName() != "model" || part.FileName() != "" {
			continue
		}
		data, err := io.ReadAll(part)
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(data))
	}
}

func compositeGeminiTargetPlatformMiddleware(resolver *service.CompositeRouteResolver) gin.HandlerFunc {
	if resolver == nil {
		resolver = service.NewCompositeRouteResolver(nil)
	}
	return func(c *gin.Context) {
		apiKey, ok := middleware.GetAPIKeyFromContext(c)
		if ok && apiKey != nil && apiKey.Group != nil && apiKey.Group.Platform == service.PlatformComposite {
			model := compositeGeminiModelFromParams(c)
			if model != "" {
				decision, err := resolver.Resolve(c.Request.Context(), apiKey.Group.ID, model, service.CompositeRouteEndpointGemini)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"type": "server_error", "message": "Failed to resolve composite model route"}})
					c.Abort()
					return
				}
				if decision.Matched {
					c.Request = c.Request.WithContext(service.WithCompositeRouteDecision(c.Request.Context(), decision))
				}
			}
			if _, resolved := service.ResolvedTargetPlatformFromContext(c.Request.Context()); !resolved {
				c.Request = c.Request.WithContext(service.WithResolvedTargetPlatform(c.Request.Context(), service.PlatformGemini))
			}
		}
		c.Next()
	}
}

func compositeGeminiModelFromParams(c *gin.Context) string {
	if c == nil {
		return ""
	}
	if model := strings.TrimSpace(c.Param("model")); model != "" {
		return model
	}
	modelAction := strings.TrimPrefix(strings.TrimSpace(c.Param("modelAction")), "/")
	if modelAction == "" {
		return ""
	}
	if idx := strings.LastIndex(modelAction, ":"); idx >= 0 {
		return strings.TrimSpace(modelAction[:idx])
	}
	return modelAction
}

func resetRequestBody(c *gin.Context, body []byte) {
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	c.Request.ContentLength = int64(len(body))
	c.Request.Header.Set("Content-Length", strconv.Itoa(len(body)))
}

func compositeRouteEndpointForPath(path string) string {
	switch {
	case strings.Contains(path, "/messages/count_tokens"):
		return service.CompositeRouteEndpointCountTokens
	case strings.Contains(path, "/messages"):
		return service.CompositeRouteEndpointMessages
	case strings.Contains(path, "/responses"):
		return service.CompositeRouteEndpointResponses
	case strings.Contains(path, "/chat/completions"):
		return service.CompositeRouteEndpointChatCompletions
	case strings.Contains(path, "/embeddings"):
		return service.CompositeRouteEndpointEmbeddings
	case strings.Contains(path, "/images/"):
		return service.CompositeRouteEndpointImages
	case strings.Contains(path, "/v1beta/"):
		return service.CompositeRouteEndpointGemini
	default:
		return service.CompositeRouteEndpointAny
	}
}
