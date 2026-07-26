package routes

import (
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

func ExecutableUserRoutes(h *handler.Handlers) []gatewayctx.RouteDef {
	if h == nil {
		return nil
	}
	out := make([]gatewayctx.RouteDef, 0, 12)
	if h.User != nil {
		out = append(out,
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/user/profile", Handler: h.User.GetProfileGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/user/avatar", Handler: h.User.GetAvatarGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/user/avatar", Handler: h.User.UpdateAvatarGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/user/password", Handler: h.User.ChangePasswordGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/user", Handler: h.User.UpdateProfileGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
		)
	}
	if h.APIKey != nil {
		out = append(out,
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/keys", Handler: h.APIKey.ListGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/keys/:id", Handler: h.APIKey.GetByIDGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/keys/:id/reveal", Handler: h.APIKey.RevealGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/keys", Handler: h.APIKey.CreateGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodPut, Path: "/api/v1/keys/:id", Handler: h.APIKey.UpdateGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodDelete, Path: "/api/v1/keys/:id", Handler: h.APIKey.DeleteGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/groups/available", Handler: h.APIKey.GetAvailableGroupsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/groups/rates", Handler: h.APIKey.GetUserGroupRatesGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
		)
	}
	if h.Announcement != nil {
		out = append(out,
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/announcements", Handler: h.Announcement.ListGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/announcements/:id/read", Handler: h.Announcement.MarkReadGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
		)
	}
	if h.Redeem != nil {
		out = append(out,
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/redeem", Handler: h.Redeem.RedeemGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/redeem/history", Handler: h.Redeem.GetHistoryGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
		)
	}
	if h.Subscription != nil {
		out = append(out,
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/subscriptions", Handler: h.Subscription.ListGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/subscriptions/active", Handler: h.Subscription.GetActiveGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/subscriptions/progress", Handler: h.Subscription.GetProgressGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/subscriptions/summary", Handler: h.Subscription.GetSummaryGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
		)
	}
	if h.Usage != nil {
		out = append(out,
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/usage", Handler: h.Usage.ListGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/usage/:id", Handler: h.Usage.GetByIDGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/usage/stats", Handler: h.Usage.StatsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/usage/dashboard/stats", Handler: h.Usage.DashboardStatsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/usage/dashboard/trend", Handler: h.Usage.DashboardTrendGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/usage/dashboard/models", Handler: h.Usage.DashboardModelsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/usage/dashboard/api-keys-usage", Handler: h.Usage.DashboardAPIKeysUsageGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
		)
	}
	if h.ImageStudio != nil {
		out = append(out, gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/image-studio/generate", Handler: h.ImageStudio.GenerateGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}})
	}
	if h.ChatStudio != nil {
		out = append(out, gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/chat-studio/complete", Handler: h.ChatStudio.CompleteGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}})
	}
	if h.ChatWorkspace != nil {
		out = append(out,
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/chat-workspace/conversations", Handler: h.ChatWorkspace.ListConversationsGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/chat-workspace/conversations", Handler: h.ChatWorkspace.CreateConversationGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/chat-workspace/conversations/:id", Handler: h.ChatWorkspace.GetConversationGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/chat-workspace/conversations/:id/messages", Handler: h.ChatWorkspace.ListMessagesGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/chat-workspace/conversations/:id/messages", Handler: h.ChatWorkspace.AppendMessageGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
		)
	}
	if h.Totp != nil {
		out = append(out,
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/user/totp/status", Handler: h.Totp.GetStatusGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodGet, Path: "/api/v1/user/totp/verification-method", Handler: h.Totp.GetVerificationMethodGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/user/totp/send-code", Handler: h.Totp.SendVerifyCodeGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/user/totp/setup", Handler: h.Totp.InitiateSetupGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/user/totp/enable", Handler: h.Totp.EnableGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
			gatewayctx.RouteDef{Method: http.MethodPost, Path: "/api/v1/user/totp/disable", Handler: h.Totp.DisableGateway, Middleware: []string{"request_logger", "cors", "security_headers", "client_request_id", "jwt_auth", "backend_mode_user_guard"}},
		)
	}
	out = append(out, executableUserFeatureRoutes(h)...)
	return out
}

// RegisterUserRoutes 注册用户相关路由（需要认证）
func RegisterUserRoutes(
	v1 *gin.RouterGroup,
	h *handler.Handlers,
	jwtAuth middleware.JWTAuthMiddleware,
	auditLog middleware.AuditLogMiddleware,
	settingService *service.SettingService,
) {
	authenticated := v1.Group("")
	authenticated.Use(gin.HandlerFunc(jwtAuth))
	authenticated.Use(middleware.BackendModeUserGuard(settingService))
	// 用户管理面变更类操作入审计（含 TOTP 启用/禁用、step-up 验证、密码修改等安全事件）
	authenticated.Use(gin.HandlerFunc(auditLog))
	{
		// 用户接口
		user := authenticated.Group("/user")
		{
			user.GET("/profile", h.User.GetProfile)
			user.GET("/avatar", h.User.GetAvatar)
			user.PUT("/avatar", h.User.UpdateAvatar)
			user.PUT("/password", h.User.ChangePassword)
			user.PUT("", h.User.UpdateProfile)
			user.GET("/aff", h.User.GetAffiliate)
			user.POST("/aff/transfer", h.User.TransferAffiliateQuota)
			user.POST("/account-bindings/email/send-code", h.User.SendEmailBindingCode)
			user.POST("/account-bindings/email", h.User.BindEmailIdentity)
			user.DELETE("/account-bindings/:provider", h.User.UnbindIdentity)
			user.POST("/auth-identities/bind/start", h.User.StartIdentityBinding)
			user.GET("/api-keys/:id/usage/daily", h.Usage.GetMyAPIKeyDailyUsage)
			user.GET("/platform-quotas", h.User.GetMyPlatformQuotas)

			// 通知邮箱管理
			notifyEmail := user.Group("/notify-email")
			{
				notifyEmail.POST("/send-code", h.User.SendNotifyEmailCode)
				notifyEmail.POST("/verify", h.User.VerifyNotifyEmail)
				notifyEmail.PUT("/toggle", h.User.ToggleNotifyEmail)
				notifyEmail.DELETE("", h.User.RemoveNotifyEmail)
			}

			// TOTP 双因素认证
			totp := user.Group("/totp")
			{
				totp.GET("/status", h.Totp.GetStatus)
				totp.GET("/verification-method", h.Totp.GetVerificationMethod)
				totp.POST("/send-code", h.Totp.SendVerifyCode)
				totp.POST("/setup", h.Totp.InitiateSetup)
				totp.POST("/enable", h.Totp.Enable)
				totp.POST("/disable", h.Totp.Disable)
				// 敏感操作二次验证：授予当前会话一段时间的 step-up 权限
				totp.POST("/step-up", h.Totp.StepUp)
			}
		}

		// API Key管理
		keys := authenticated.Group("/keys")
		{
			keys.GET("", h.APIKey.List)
			keys.GET("/:id", h.APIKey.GetByID)
			keys.POST("/:id/reveal", h.APIKey.Reveal)
			keys.POST("", h.APIKey.Create)
			keys.PUT("/:id", h.APIKey.Update)
			keys.DELETE("/:id", h.APIKey.Delete)
		}

		// 用户可用分组（非管理员接口）
		groups := authenticated.Group("/groups")
		{
			groups.GET("/available", h.APIKey.GetAvailableGroups)
			groups.GET("/rates", h.APIKey.GetUserGroupRates)
		}

		// 用户可用渠道（非管理员接口）
		channels := authenticated.Group("/channels")
		{
			channels.GET("/available", h.AvailableChannel.List)
		}

		// 使用记录
		usage := authenticated.Group("/usage")
		{
			usage.GET("", h.Usage.List)
			usage.GET("/errors", h.Usage.ListErrors)
			usage.GET("/errors/:id", h.Usage.GetErrorDetail)
			usage.GET("/:id", h.Usage.GetByID)
			usage.GET("/stats", h.Usage.Stats)
			// User dashboard endpoints
			usage.GET("/dashboard/stats", h.Usage.DashboardStats)
			usage.GET("/dashboard/trend", h.Usage.DashboardTrend)
			usage.GET("/dashboard/models", h.Usage.DashboardModels)
			usage.GET("/dashboard/snapshot-v2", h.Usage.DashboardSnapshotV2)
			usage.POST("/dashboard/api-keys-usage", h.Usage.DashboardAPIKeysUsage)
		}

		// 公告（用户可见）
		announcements := authenticated.Group("/announcements")
		{
			announcements.GET("", h.Announcement.List)
			announcements.POST("/:id/read", h.Announcement.MarkRead)
		}

		// 卡密兑换
		redeem := authenticated.Group("/redeem")
		{
			redeem.POST("", h.Redeem.Redeem)
			redeem.GET("/history", h.Redeem.GetHistory)
		}

		// 用户订阅
		subscriptions := authenticated.Group("/subscriptions")
		{
			subscriptions.GET("", h.Subscription.List)
			subscriptions.GET("/active", h.Subscription.GetActive)
			subscriptions.GET("/progress", h.Subscription.GetProgress)
			subscriptions.GET("/summary", h.Subscription.GetSummary)
		}

		// 渠道监控（用户只读）
		monitors := authenticated.Group("/channel-monitors")
		{
			monitors.GET("", h.ChannelMonitor.List)
			monitors.GET("/:id/status", h.ChannelMonitor.GetStatus)
		}
		if h.ImageStudio != nil {
			imageStudio := authenticated.Group("/image-studio")
			{
				imageStudio.POST("/generate", h.ImageStudio.Generate)
			}
		}
		if h.ChatStudio != nil {
			chatStudio := authenticated.Group("/chat-studio")
			{
				chatStudio.POST("/complete", h.ChatStudio.Complete)
			}
		}
		if h.ChatWorkspace != nil {
			chatWorkspace := authenticated.Group("/chat-workspace")
			{
				chatWorkspace.GET("/conversations", h.ChatWorkspace.ListConversations)
				chatWorkspace.POST("/conversations", h.ChatWorkspace.CreateConversation)
				chatWorkspace.GET("/conversations/:id", h.ChatWorkspace.GetConversation)
				chatWorkspace.GET("/conversations/:id/messages", h.ChatWorkspace.ListMessages)
				chatWorkspace.POST("/conversations/:id/messages", h.ChatWorkspace.AppendMessage)
			}
		}

		// 注意：/user/aff、/user/aff/transfer、/channels/available、
		// /channel-monitors 等路由已在上方内联注册（生产版本，路由更全）。
		// registerUserFeatureRoutes 是换底座时引入的等价辅助函数，与内联注册重复，
		// 会导致 gin "handlers are already registered" panic，故不再调用。
	}
}
