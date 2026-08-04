package service

import (
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/domain"
)

// Status constants
const (
	StatusActive   = domain.StatusActive
	StatusDisabled = domain.StatusDisabled
	StatusError    = domain.StatusError
	StatusUnused   = domain.StatusUnused
	StatusUsed     = domain.StatusUsed
	StatusExpired  = domain.StatusExpired
)

// Role constants
const (
	RoleAdmin = domain.RoleAdmin
	RoleUser  = domain.RoleUser
)

// Affiliate rebate settings.
const (
	AffiliateRebateRateDefault          = 20.0
	AffiliateRebateRateMin              = 0.0
	AffiliateRebateRateMax              = 100.0
	AffiliateEnabledDefault             = false
	AffiliateRebateFreezeHoursDefault   = 0
	AffiliateRebateFreezeHoursMax       = 720
	AffiliateRebateDurationDaysDefault  = 0
	AffiliateRebateDurationDaysMax      = 3650
	AffiliateRebatePerInviteeCapDefault = 0.0
)

// Platform constants
const (
	PlatformAnthropic   = domain.PlatformAnthropic
	PlatformOpenAI      = domain.PlatformOpenAI
	PlatformGemini      = domain.PlatformGemini
	PlatformGrok        = domain.PlatformGrok
	PlatformAntigravity = domain.PlatformAntigravity
	PlatformSora        = domain.PlatformSora
	PlatformKiro        = domain.PlatformKiro
	PlatformComposite   = domain.PlatformComposite
)

// Account type constants
const (
	AccountTypeOAuth      = domain.AccountTypeOAuth      // OAuth类型账号（full scope: profile + inference）
	AccountTypeSetupToken = domain.AccountTypeSetupToken // Setup Token类型账号（inference only scope）
	AccountTypeAPIKey     = domain.AccountTypeAPIKey     // API Key类型账号
	AccountTypeUpstream   = domain.AccountTypeUpstream   // 上游透传类型账号（通过 Base URL + API Key 连接上游）
	AccountTypeBedrock    = domain.AccountTypeBedrock    // AWS Bedrock 类型账号（通过 SigV4 签名或 API Key 连接 Bedrock，由 credentials.auth_mode 区分）
)

const AccountTypeServiceAccount = domain.AccountTypeServiceAccount

// Redeem type constants
const (
	RedeemTypeBalance          = domain.RedeemTypeBalance
	RedeemTypeConcurrency      = domain.RedeemTypeConcurrency
	RedeemTypeSubscription     = domain.RedeemTypeSubscription
	RedeemTypeInvitation       = domain.RedeemTypeInvitation
	RedeemTypeAffiliateBalance = "affiliate_balance"
)

// PromoCode status constants
const (
	PromoCodeStatusActive   = domain.PromoCodeStatusActive
	PromoCodeStatusDisabled = domain.PromoCodeStatusDisabled
)

// Admin adjustment type constants
const (
	AdjustmentTypeAdminBalance     = domain.AdjustmentTypeAdminBalance     // 管理员调整余额
	AdjustmentTypeAdminConcurrency = domain.AdjustmentTypeAdminConcurrency // 管理员调整并发数
)

// Group subscription type constants
const (
	SubscriptionTypeStandard     = domain.SubscriptionTypeStandard     // 标准计费模式（按余额扣费）
	SubscriptionTypeSubscription = domain.SubscriptionTypeSubscription // 订阅模式（按限额控制）
)

// Subscription status constants
const (
	SubscriptionStatusActive    = domain.SubscriptionStatusActive
	SubscriptionStatusExpired   = domain.SubscriptionStatusExpired
	SubscriptionStatusSuspended = domain.SubscriptionStatusSuspended
	// SubscriptionStatusRevoked is the API-visible state for soft-deleted subscriptions.
	SubscriptionStatusRevoked = "revoked"
)

// LinuxDoConnectSyntheticEmailDomain 是 LinuxDo Connect 用户的合成邮箱后缀（RFC 保留域名）。
const LinuxDoConnectSyntheticEmailDomain = "@linuxdo-connect.invalid"

// OIDCConnectSyntheticEmailDomain 是 OIDC 用户的合成邮箱后缀（RFC 保留域名）。
const OIDCConnectSyntheticEmailDomain = "@oidc-connect.invalid"

// WeChatConnectSyntheticEmailDomain 是 WeChat Connect 用户的合成邮箱后缀（RFC 保留域名）。
const WeChatConnectSyntheticEmailDomain = "@wechat-connect.invalid"

// DingTalkConnectSyntheticEmailDomain is the reserved synthetic-email suffix for DingTalk identities.
const DingTalkConnectSyntheticEmailDomain = "@dingtalk-connect.invalid"

// Setting keys
const (
	// 注册设置
	SettingKeyRegistrationEnabled              = "registration_enabled"                // 是否开放注册
	SettingKeyEmailVerifyEnabled               = "email_verify_enabled"                // 是否开启邮件验证
	SettingKeyRegistrationEmailSuffixWhitelist = "registration_email_suffix_whitelist" // 注册邮箱后缀白名单（JSON 数组）
	SettingKeyPromoCodeEnabled                 = "promo_code_enabled"                  // 是否启用优惠码功能
	SettingKeyPasswordResetEnabled             = "password_reset_enabled"              // 是否启用忘记密码功能（需要先开启邮件验证）
	SettingKeyFrontendURL                      = "frontend_url"                        // 前端基础URL，用于生成邮件中的重置密码链接
	SettingKeyInvitationCodeEnabled            = "invitation_code_enabled"             // 是否启用邀请码注册
	SettingKeyAffiliateEnabled                 = "affiliate_enabled"                   // 邀请返利功能总开关
	SettingKeyAffiliateRebateRate              = "affiliate_rebate_rate"               // 邀请返利比例（百分比，0-100）
	SettingKeyAffiliateRebateFreezeHours       = "affiliate_rebate_freeze_hours"       // 返利冻结期（小时，0=不冻结）
	SettingKeyAffiliateRebateDurationDays      = "affiliate_rebate_duration_days"      // 返利有效期（天，0=永久）
	SettingKeyAffiliateRebatePerInviteeCap     = "affiliate_rebate_per_invitee_cap"    // 单人返利上限（0=无上限）

	// 邮件服务设置
	SettingKeySMTPHost     = "smtp_host"      // SMTP服务器地址
	SettingKeySMTPPort     = "smtp_port"      // SMTP端口
	SettingKeySMTPUsername = "smtp_username"  // SMTP用户名
	SettingKeySMTPPassword = "smtp_password"  // SMTP密码（加密存储）
	SettingKeySMTPFrom     = "smtp_from"      // 发件人地址
	SettingKeySMTPFromName = "smtp_from_name" // 发件人名称
	SettingKeySMTPUseTLS   = "smtp_use_tls"   // 是否使用TLS

	// Cloudflare Turnstile 设置
	SettingKeyTurnstileEnabled   = "turnstile_enabled"    // 是否启用 Turnstile 验证
	SettingKeyTurnstileSiteKey   = "turnstile_site_key"   // Turnstile Site Key
	SettingKeyTurnstileSecretKey = "turnstile_secret_key" // Turnstile Secret Key

	// 腾讯天御验证码设置
	SettingKeyTencentCaptchaEnabled        = "tencent_captcha_enabled"
	SettingKeyTencentCaptchaAppID          = "tencent_captcha_app_id"
	SettingKeyTencentCaptchaAppSecretKey   = "tencent_captcha_app_secret_key"
	SettingKeyTencentCaptchaCloudSecretID  = "tencent_captcha_cloud_secret_id"
	SettingKeyTencentCaptchaCloudSecretKey = "tencent_captcha_cloud_secret_key"

	// 阿里云验证码 2.0 设置（与 Turnstile、腾讯天御互斥，同一时间仅可启用一家）
	SettingKeyAliyunCaptchaEnabled         = "aliyun_captcha_enabled"           // 是否启用阿里云验证码
	SettingKeyAliyunCaptchaAccessKeyID     = "aliyun_captcha_access_key_id"     // 阿里云 AccessKey ID
	SettingKeyAliyunCaptchaAccessKeySecret = "aliyun_captcha_access_key_secret" // 阿里云 AccessKey Secret
	SettingKeyAliyunCaptchaSceneID         = "aliyun_captcha_scene_id"          // 验证场景 ID（所有认证流程共用）
	SettingKeyAliyunCaptchaPrefix          = "aliyun_captcha_prefix"            // 身份标，前端 SDK 初始化用
	SettingKeyAliyunCaptchaRegion          = "aliyun_captcha_region"            // 地域："cn"|"sgp"，决定前端脚本区域与服务端接入点

	// API Key IP 访问控制设置
	SettingKeyAPIKeyACLTrustForwardedIP = "api_key_acl_trust_forwarded_ip" // API Key IP 白/黑名单是否信任转发 IP
	SettingKeyForwardedClientIPHeaders  = "forwarded_client_ip_headers"    // 自定义 CDN 客户端 IP 请求头（JSON 数组）
	settingKeyForwardedClientIPModeV2   = "forwarded_client_ip_mode_v2_migrated"

	// TOTP 双因素认证设置
	SettingKeyTotpEnabled    = "totp_enabled"    // 是否启用 TOTP 2FA 功能
	SettingKeyPasskeyEnabled = "passkey_enabled" // 是否启用 Passkey 登录（仍要求有效的 WebAuthn 部署配置）

	// LinuxDo Connect OAuth 登录设置
	SettingKeyLinuxDoConnectEnabled      = "linuxdo_connect_enabled"
	SettingKeyLinuxDoConnectClientID     = "linuxdo_connect_client_id"
	SettingKeyLinuxDoConnectClientSecret = "linuxdo_connect_client_secret"
	SettingKeyLinuxDoConnectRedirectURL  = "linuxdo_connect_redirect_url"

	// OEM设置
	SettingKeySoraClientEnabled           = "sora_client_enabled" // 是否启用 Sora 客户端（管理员手动控制）
	SettingKeyPanelRateLimitSettings      = "panel_rate_limit_settings"
	SettingKeySiteName                    = "site_name"                     // 网站名称
	SettingKeySiteLogo                    = "site_logo"                     // 网站Logo (base64)
	SettingKeySiteSubtitle                = "site_subtitle"                 // 网站副标题
	SettingKeyAPIBaseURL                  = "api_base_url"                  // API端点地址（用于客户端配置和导入）
	SettingKeyContactInfo                 = "contact_info"                  // 客服联系方式
	SettingKeyDocURL                      = "doc_url"                       // 文档链接
	SettingKeyHomeContent                 = "home_content"                  // 首页内容（支持 Markdown/HTML，或 URL 作为 iframe src）
	SettingKeyCompactHomeEnabled          = "compact_home_enabled"          // 是否启用内置简洁首页
	SettingKeyHideCcsImportButton         = "hide_ccs_import_button"        // 是否隐藏 API Keys 页面的导入 CCS 按钮
	SettingKeyPurchaseSubscriptionEnabled = "purchase_subscription_enabled" // 是否展示"购买订阅"页面入口
	SettingKeyPurchaseSubscriptionURL     = "purchase_subscription_url"     // "购买订阅"页面 URL（作为 iframe src）
	SettingKeyPurchaseLinkCNY10           = "purchase_link_cny_10"
	SettingKeyPurchaseLinkCNY30           = "purchase_link_cny_30"
	SettingKeyPurchaseLinkCNY100          = "purchase_link_cny_100"
	SettingKeyCustomMenuItems             = "custom_menu_items" // 自定义菜单项（JSON 数组）

	// 默认配置
	SettingKeyDefaultConcurrency   = "default_concurrency"   // 新用户默认并发量
	SettingKeyDefaultBalance       = "default_balance"       // 新用户默认余额
	SettingKeyDefaultSubscriptions = "default_subscriptions" // 新用户默认订阅列表（JSON）

	// 管理员 API Key
	SettingKeyAdminAPIKey = "admin_api_key" // 全局管理员 API Key（用于外部系统集成）

	// Gemini 配额策略（JSON）
	SettingKeyGeminiQuotaPolicy = "gemini_quota_policy"

	// Model fallback settings
	SettingKeyEnableModelFallback      = "enable_model_fallback"
	SettingKeyFallbackModelAnthropic   = "fallback_model_anthropic"
	SettingKeyFallbackModelOpenAI      = "fallback_model_openai"
	SettingKeyFallbackModelGemini      = "fallback_model_gemini"
	SettingKeyFallbackModelAntigravity = "fallback_model_antigravity"

	// Request identity patch (Claude -> Gemini systemInstruction injection)
	SettingKeyEnableIdentityPatch = "enable_identity_patch"
	SettingKeyIdentityPatchPrompt = "identity_patch_prompt"

	// =========================
	// Ops Monitoring (vNext)
	// =========================

	// SettingKeyOpsMonitoringEnabled is a DB-backed soft switch to enable/disable ops module at runtime.
	SettingKeyOpsMonitoringEnabled = "ops_monitoring_enabled"

	// SettingKeyOpsRealtimeMonitoringEnabled controls realtime features (e.g. WS/QPS push).
	SettingKeyOpsRealtimeMonitoringEnabled = "ops_realtime_monitoring_enabled"

	// SettingKeyOpsQueryModeDefault controls the default query mode for ops dashboard (auto/raw/preagg).
	SettingKeyOpsQueryModeDefault = "ops_query_mode_default"

	// SettingKeyOpsEmailNotificationConfig stores JSON config for ops email notifications.
	SettingKeyOpsEmailNotificationConfig = "ops_email_notification_config"

	// SettingKeyOpsAlertRuntimeSettings stores JSON config for ops alert evaluator runtime settings.
	SettingKeyOpsAlertRuntimeSettings = "ops_alert_runtime_settings"

	// SettingKeyOpsMetricsIntervalSeconds controls the ops metrics collector interval (>=60).
	SettingKeyOpsMetricsIntervalSeconds = "ops_metrics_interval_seconds"

	// SettingKeyOpsAdvancedSettings stores JSON config for ops advanced settings (data retention, aggregation).
	SettingKeyOpsAdvancedSettings = "ops_advanced_settings"

	// SettingKeyOpsRuntimeLogConfig stores JSON config for runtime log settings.
	SettingKeyOpsRuntimeLogConfig = "ops_runtime_log_config"

	// =========================
	// Ollama Cloud Usage
	// =========================

	// SettingKeyChannelMonitorEnabled is a DB-backed soft switch for the channel monitor feature.
	// When false: runner skips scheduling and user-facing endpoints return an empty list.
	SettingKeyChannelMonitorEnabled = "channel_monitor_enabled"

	// SettingKeyChannelMonitorDefaultIntervalSeconds controls the default interval (seconds)
	// pre-filled when creating a new channel monitor from the admin UI. Range: [15, 3600].
	SettingKeyChannelMonitorDefaultIntervalSeconds = "channel_monitor_default_interval_seconds"

	// SettingKeyAvailableChannelsEnabled is a DB-backed soft switch for the "Available Channels"
	// user-facing aggregate view. When false: user endpoint returns an empty list and the
	// sidebar entry is hidden. Defaults to false (opt-in feature).
	SettingKeyAvailableChannelsEnabled = "available_channels_enabled"

	// SettingKeyModelPlazaEnabled is a DB-backed soft switch for the Model Plaza page
	// (public group/model pricing showcase). When false: the plaza endpoint returns 404
	// and the header entry is hidden. Defaults to false (opt-in feature).
	SettingKeyModelPlazaEnabled = "model_plaza_enabled"

	// SettingKeyModelPlazaRequireAuth controls whether the Model Plaza page requires a
	// logged-in user. When false the page is public and anonymous visitors see only
	// non-exclusive groups.
	SettingKeyModelPlazaRequireAuth = "model_plaza_require_auth"

	// SettingKeyModelPlazaDescription stores the Markdown blurb rendered at the top of
	// the Model Plaza page (global pricing notes, exchange rate, promotions, ...).
	SettingKeyModelPlazaDescription = "model_plaza_description"

	// SettingKeyUpstreamBillingProbeSettings stores the global enable switch and interval
	// for probing remote Sub2API API-key billing metadata.
	SettingKeyUpstreamBillingProbeSettings = "upstream_billing_probe_settings"

	// SettingKeyOllamaCloudUsageSettings stores the opt-in global runner switch and interval.
	SettingKeyOllamaCloudUsageSettings = "ollama_cloud_usage_settings"

	// =========================
	// Overload Cooldown (529)
	// =========================

	// SettingKeyOverloadCooldownSettings stores JSON config for 529 overload cooldown handling.
	SettingKeyOverloadCooldownSettings = "overload_cooldown_settings"

	// =========================
	// Stream Timeout Handling
	// =========================

	// SettingKeyStreamTimeoutSettings stores JSON config for stream timeout handling.
	SettingKeyStreamTimeoutSettings = "stream_timeout_settings"

	// =========================
	// Request Rectifier (请求整流器)
	// =========================

	// SettingKeyRectifierSettings stores JSON config for rectifier settings (thinking signature + budget).
	SettingKeyRectifierSettings = "rectifier_settings"

	// =========================
	// Beta Policy Settings
	// =========================

	// SettingKeyBetaPolicySettings stores JSON config for beta policy rules.
	SettingKeyBetaPolicySettings = "beta_policy_settings"

	// =========================
	// TLS Fingerprint Profiles
	// =========================

	// =========================
	// Sora S3 存储配置
	// =========================

	SettingKeySoraS3Enabled         = "sora_s3_enabled"           // 是否启用 Sora S3 存储
	SettingKeySoraS3Endpoint        = "sora_s3_endpoint"          // S3 端点地址
	SettingKeySoraS3Region          = "sora_s3_region"            // S3 区域
	SettingKeySoraS3Bucket          = "sora_s3_bucket"            // S3 存储桶名称
	SettingKeySoraS3AccessKeyID     = "sora_s3_access_key_id"     // S3 Access Key ID
	SettingKeySoraS3SecretAccessKey = "sora_s3_secret_access_key" // S3 Secret Access Key（加密存储）
	SettingKeySoraS3Prefix          = "sora_s3_prefix"            // S3 对象键前缀
	SettingKeySoraS3ForcePathStyle  = "sora_s3_force_path_style"  // 是否强制 Path Style（兼容 MinIO 等）
	SettingKeySoraS3CDNURL          = "sora_s3_cdn_url"           // CDN 加速 URL（可选）
	// SettingKeyTLSFingerprintProfiles stores JSON config for DB-backed TLS fingerprint settings.
	SettingKeyTLSFingerprintProfiles = "tls_fingerprint_profiles"
	SettingKeySoraS3Profiles         = "sora_s3_profiles" // Sora S3 多配置（JSON）

	// =========================
	// Sora 用户存储配额
	// =========================

	SettingKeySoraDefaultStorageQuotaBytes = "sora_default_storage_quota_bytes" // 新用户默认 Sora 存储配额（字节）

	// =========================
	// Claude Code Version Check
	// =========================

	// SettingKeyMinClaudeCodeVersion 最低 Claude Code 版本号要求 (semver, 如 "2.1.0"，空值=不检查)
	SettingKeyMinClaudeCodeVersion = "min_claude_code_version"

	// SettingKeyMaxClaudeCodeVersion 最高 Claude Code 版本号限制 (semver, 如 "3.0.0"，空值=不检查)
	SettingKeyMaxClaudeCodeVersion = "max_claude_code_version"

	// SettingKeyAllowUngroupedKeyScheduling 允许未分组 API Key 调度（默认 false：未分组 Key 返回 403）
	SettingKeyAllowUngroupedKeyScheduling = "allow_ungrouped_key_scheduling"

	// 自动清理策略
	SettingKeyAutoDelete401Accounts    = "auto_delete_401_accounts"
	SettingKeyAutoDelete429Accounts    = "auto_delete_429_accounts"
	SettingKeyAutoDeleteUselessProxies = "auto_delete_useless_proxies"

	// SettingKeyOpenAILowUpstreamRatePriorityEnabled controls legacy scheduling by upstream token cost.
	SettingKeyOpenAILowUpstreamRatePriorityEnabled = "openai_low_upstream_rate_priority_enabled"
	// SettingKeyOpenAIOAuthSchedulingRateMultiplier is the reference cost multiplier for OAuth accounts.
	SettingKeyOpenAIOAuthSchedulingRateMultiplier                = "openai_oauth_scheduling_rate_multiplier"
	SettingKeyOpenAIAdvancedSchedulerStickyWeightedEnabled       = "openai_advanced_scheduler_sticky_weighted_enabled"
	SettingKeyOpenAIAdvancedSchedulerSubscriptionPriorityEnabled = "openai_advanced_scheduler_subscription_priority_enabled"
	SettingKeyOpenAIAdvancedSchedulerLBTopK                      = "openai_advanced_scheduler_lb_top_k"
	SettingKeyOpenAIAdvancedSchedulerWeightPriority              = "openai_advanced_scheduler_weight_priority"
	SettingKeyOpenAIAdvancedSchedulerWeightLoad                  = "openai_advanced_scheduler_weight_load"
	SettingKeyOpenAIAdvancedSchedulerWeightQueue                 = "openai_advanced_scheduler_weight_queue"
	SettingKeyOpenAIAdvancedSchedulerWeightErrorRate             = "openai_advanced_scheduler_weight_error_rate"
	SettingKeyOpenAIAdvancedSchedulerWeightTTFT                  = "openai_advanced_scheduler_weight_ttft"
	SettingKeyOpenAIAdvancedSchedulerWeightReset                 = "openai_advanced_scheduler_weight_reset"
	SettingKeyOpenAIAdvancedSchedulerWeightQuotaHeadroom         = "openai_advanced_scheduler_weight_quota_headroom"
	SettingKeyOpenAIAdvancedSchedulerWeightUpstreamCost          = "openai_advanced_scheduler_weight_upstream_cost"
	SettingKeyOpenAIAdvancedSchedulerWeightPreviousResponse      = "openai_advanced_scheduler_weight_previous_response"
	SettingKeyOpenAIAdvancedSchedulerWeightSessionSticky         = "openai_advanced_scheduler_weight_session_sticky"

	// SettingKeyBackendModeEnabled Backend 模式：禁用用户注册和自助服务，仅管理员可登录
	SettingKeyBackendModeEnabled = "backend_mode_enabled"

	// Gateway Forwarding Behavior
	// SettingKeyEnableFingerprintUnification 是否统一 OAuth 账号的 X-Stainless-* 指纹头（默认 true）
	SettingKeyEnableFingerprintUnification = "enable_fingerprint_unification"
	// SettingKeyEnableMetadataPassthrough 是否透传客户端原始 metadata.user_id（默认 false）
	SettingKeyEnableMetadataPassthrough = "enable_metadata_passthrough"
	// SettingKeyEnableCCHSigning 已废弃（no-op）：新版 Claude Code CLI 已取消 cch 签名字段，
	// 网关随之不再注入/签名 cch（见 buildBillingAttributionText）。保留该 key 仅为向后兼容，
	// 开关不再产生任何效果。
	SettingKeyEnableCCHSigning = "enable_cch_signing"
	// SettingKeyEnableClaudeOAuthSystemPromptInjection 是否对 Claude OAuth mimic 路径注入 Claude Code system blocks（默认 true）
	SettingKeyEnableClaudeOAuthSystemPromptInjection = "enable_claude_oauth_system_prompt_injection"
	// SettingKeyClaudeOAuthSystemPrompt Claude OAuth mimic 路径注入的通用扩展 system prompt（空值使用内置默认）
	SettingKeyClaudeOAuthSystemPrompt = "claude_oauth_system_prompt"
	// SettingKeyClaudeOAuthSystemPromptBlocks Claude OAuth mimic 路径注入的 system blocks JSON 配置（空值使用内置默认）
	SettingKeyClaudeOAuthSystemPromptBlocks = "claude_oauth_system_prompt_blocks"
	// SettingKeyEnableAnthropicCacheTTL1hInjection 是否对 Anthropic OAuth/SetupToken 请求体注入 1h cache_control ttl（默认 false）
	SettingKeyEnableAnthropicCacheTTL1hInjection = "enable_anthropic_cache_ttl_1h_injection"
	// SettingKeyEnableClientDatelineNormalization 是否对 Anthropic OAuth/SetupToken 账号
	// 的 /v1/messages 请求体做客户端 dateline 归一化（默认 true）。
	// 归一化把 system prompt / <system-reminder> 块中 "Today's date is …" 语句里的
	// 非 ASCII 撇号与 "/" 日期分隔符还原为 ASCII 撇号 + "-" 分隔符，抹除某些客户端
	// 在检测到非官方 base URL 时注入的 3 bit 隐写指纹。仅适用于 Anthropic OAuth/SetupToken
	// 账号；API Key 账号不受影响。
	SettingKeyEnableClientDatelineNormalization = "enable_client_dateline_normalization"
	// SettingKeyRewriteMessageCacheControl 是否改写 messages[*].content[*].cache_control（默认 false）
	SettingKeyRewriteMessageCacheControl = "rewrite_message_cache_control"
	// SettingKeyAntigravityUserAgentVersion Antigravity 上游 User-Agent 版本号（空值使用环境变量/默认值）
	SettingKeyAntigravityUserAgentVersion = "antigravity_user_agent_version"
	// SettingKeyOpenAICodexUserAgent OpenAI Codex 完整 User-Agent（空值使用内置默认）
	// 当客户端 UA 被识别为浏览器（Chrome/Firefox/Safari/Edge 等）时，转发给 OpenAI 上游前会替换为此值，
	// 用于避免 Cloudflare 对浏览器型 UA 的质询拦截。
	SettingKeyOpenAICodexUserAgent = "openai_codex_user_agent"
	// SettingKeyOpenAICodexClientVersion 网关对 ChatGPT 上游声明的 Codex 客户端版本号（管理员覆写）。
	// 空值表示跟随自动同步值；自动同步也没有结果时回退到内置常量。
	// 上游在容量紧张时按客户端身份分优先级降载，陈旧版本会被优先丢弃，故该值需保持跟随官方发布。
	SettingKeyOpenAICodexClientVersion = "openai_codex_client_version"
	// SettingKeyOpenAICodexClientVersionSynced 自动同步任务写入的官方 Codex 最新稳定版版本号。
	// 由 OpenAICodexVersionSyncService 独占写入，面板只读展示；管理员覆写请用
	// SettingKeyOpenAICodexClientVersion。
	SettingKeyOpenAICodexClientVersionSynced = "openai_codex_client_version_synced"
	// SettingKeyOpenAICodexVersionAutoSyncEnabled 是否启用 Codex 客户端版本号自动同步（默认 true）。
	SettingKeyOpenAICodexVersionAutoSyncEnabled = "openai_codex_version_auto_sync_enabled"
	// SettingKeyOpenAIAllowClaudeCodeCodexPlugin 已废弃：历史全局开关只作为升级迁移输入读取。
	// 迁移后等价规则写入 SettingKeyCodexCLIOnlyWhitelist，不再参与运行时判定。
	SettingKeyOpenAIAllowClaudeCodeCodexPlugin = "openai_allow_claude_code_codex_plugin"

	// 余额不足提醒
	SettingKeyBalanceLowNotifyEnabled     = "balance_low_notify_enabled"      // 全局开关
	SettingKeyBalanceLowNotifyThreshold   = "balance_low_notify_threshold"    // 默认阈值（USD）
	SettingKeyBalanceLowNotifyRechargeURL = "balance_low_notify_recharge_url" // 充值页面 URL

	// 订阅到期提醒
	SettingKeySubscriptionExpiryNotifyEnabled = "subscription_expiry_notify_enabled" // 订阅到期提醒全局开关，默认开启

	// 账号限额通知
	SettingKeyAccountQuotaNotifyEnabled = "account_quota_notify_enabled" // 全局开关
	SettingKeyAccountQuotaNotifyEmails  = "account_quota_notify_emails"  // 管理员通知邮箱列表（JSON 数组）

	// Web Search Emulation
	SettingKeyWebSearchEmulationConfig = "web_search_emulation_config" // JSON 配置
)

// SettingKeyDefaultPlatformQuotas —— 系统全局：每用户 × 平台日/周/月 USD 上限（JSON）。
// 值为 map[platform]{daily,weekly,monthly}，null/缺省 = 不限制；0 = 禁用；>0 = USD 上限。
const SettingKeyDefaultPlatformQuotas = "default_platform_quotas"

// SettingKeyAuthSourcePlatformQuotas 返回某 auth source 的 platform quota JSON key。
// 形如 auth_source_default_{source}_platform_quotas
func SettingKeyAuthSourcePlatformQuotas(source string) string {
	return fmt.Sprintf("auth_source_default_%s_platform_quotas", source)
}

// QuotaDimension constants for spark shadow accounts.
const (
	QuotaDimensionGlobal = "global"
	QuotaDimensionSpark  = "spark"
)

// AdminAPIKeyPrefix is the prefix for admin API keys (distinct from user "sk-" keys).
const AdminAPIKeyPrefix = "admin-"

const featureKeyWebSearchEmulation = "web_search_emulation"
