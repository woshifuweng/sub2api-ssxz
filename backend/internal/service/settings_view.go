package service

type SystemSettings struct {
	RegistrationEnabled              bool
	EmailVerifyEnabled               bool
	RegistrationEmailSuffixWhitelist []string
	PromoCodeEnabled                 bool
	PasswordResetEnabled             bool
	FrontendURL                      string
	InvitationCodeEnabled            bool
	TotpEnabled                      bool // TOTP 双因素认证

	SMTPHost               string
	SMTPPort               int
	SMTPUsername           string
	SMTPPassword           string
	SMTPPasswordConfigured bool
	SMTPFrom               string
	SMTPFromName           string
	SMTPUseTLS             bool

	TurnstileEnabled             bool
	TurnstileSiteKey             string
	TurnstileSecretKey           string
	TurnstileSecretKeyConfigured bool

	// LinuxDo Connect OAuth 登录
	LinuxDoConnectEnabled                bool
	LinuxDoConnectClientID               string
	LinuxDoConnectClientSecret           string
	LinuxDoConnectClientSecretConfigured bool
	LinuxDoConnectRedirectURL            string

	SiteName                    string
	SiteLogo                    string
	SiteSubtitle                string
	APIBaseURL                  string
	ContactInfo                 string
	DocURL                      string
	HomeContent                 string
	HideCcsImportButton         bool
	PurchaseSubscriptionEnabled bool
	PurchaseSubscriptionURL     string
	PurchaseLinkCNY10           string
	PurchaseLinkCNY30           string
	PurchaseLinkCNY100          string
	SoraClientEnabled           bool
	CustomMenuItems             string // JSON array of custom menu items

	DefaultConcurrency           int
	DefaultBalance               float64
	AffiliateRebateRate          float64
	AffiliateRebateFreezeHours   int
	AffiliateRebateDurationDays  int
	AffiliateRebatePerInviteeCap float64
	DefaultSubscriptions         []DefaultSubscriptionSetting

	// Model fallback configuration
	EnableModelFallback      bool   `json:"enable_model_fallback"`
	FallbackModelAnthropic   string `json:"fallback_model_anthropic"`
	FallbackModelOpenAI      string `json:"fallback_model_openai"`
	FallbackModelGemini      string `json:"fallback_model_gemini"`
	FallbackModelAntigravity string `json:"fallback_model_antigravity"`

	// Identity patch configuration (Claude -> Gemini)
	EnableIdentityPatch bool   `json:"enable_identity_patch"`
	IdentityPatchPrompt string `json:"identity_patch_prompt"`

	// Ops monitoring (vNext)
	OpsMonitoringEnabled         bool
	OpsRealtimeMonitoringEnabled bool
	OpsQueryModeDefault          string
	OpsMetricsIntervalSeconds    int

	// Claude Code version check
	MinClaudeCodeVersion string
	MaxClaudeCodeVersion string

	// 分组隔离：允许未分组 Key 调度（默认 false → 403）
	AllowUngroupedKeyScheduling bool

	AutoDelete401Accounts    bool
	AutoDelete429Accounts    bool
	AutoDeleteUselessProxies bool

	// Backend 模式：禁用用户注册和自助服务，仅管理员可登录
	BackendModeEnabled bool

	AffiliateEnabled bool

	SessionBindingEnabled   bool
	StepUpEnabled           bool
	AuditLogRetentionDays   int
	LoginAgreementEnabled   bool
	LoginAgreementMode      string
	LoginAgreementUpdatedAt string
	LoginAgreementDocuments []LoginAgreementDocument

	APIKeyACLTrustForwardedIP bool
	ForwardedClientIPHeaders  []string

	DingTalkConnectEnabled                 bool
	DingTalkConnectClientID                string
	DingTalkConnectClientSecret            string
	DingTalkConnectClientSecretConfigured  bool
	DingTalkConnectRedirectURL             string
	DingTalkConnectCorpRestrictionPolicy   string
	DingTalkConnectInternalCorpID          string
	DingTalkConnectBypassRegistration      bool
	DingTalkConnectSyncCorpEmail           bool
	DingTalkConnectSyncDisplayName         bool
	DingTalkConnectSyncDept                bool
	DingTalkConnectSyncCorpEmailAttrKey    string
	DingTalkConnectSyncDisplayNameAttrKey  string
	DingTalkConnectSyncDeptAttrKey         string
	DingTalkConnectSyncCorpEmailAttrName   string
	DingTalkConnectSyncDisplayNameAttrName string
	DingTalkConnectSyncDeptAttrName        string

	WeChatConnectEnabled                   bool
	WeChatConnectAppID                     string
	WeChatConnectAppSecret                 string
	WeChatConnectAppSecretConfigured       bool
	WeChatConnectOpenAppID                 string
	WeChatConnectOpenAppSecret             string
	WeChatConnectOpenAppSecretConfigured   bool
	WeChatConnectMPAppID                   string
	WeChatConnectMPAppSecret               string
	WeChatConnectMPAppSecretConfigured     bool
	WeChatConnectMobileAppID               string
	WeChatConnectMobileAppSecret           string
	WeChatConnectMobileAppSecretConfigured bool
	WeChatConnectOpenEnabled               bool
	WeChatConnectMPEnabled                 bool
	WeChatConnectMobileEnabled             bool
	WeChatConnectMode                      string
	WeChatConnectScopes                    string
	WeChatConnectRedirectURL               string
	WeChatConnectFrontendRedirectURL       string

	OIDCConnectEnabled                bool
	OIDCConnectProviderName           string
	OIDCConnectClientID               string
	OIDCConnectClientSecret           string
	OIDCConnectClientSecretConfigured bool
	OIDCConnectIssuerURL              string
	OIDCConnectDiscoveryURL           string
	OIDCConnectAuthorizeURL           string
	OIDCConnectTokenURL               string
	OIDCConnectUserInfoURL            string
	OIDCConnectJWKSURL                string
	OIDCConnectScopes                 string
	OIDCConnectRedirectURL            string
	OIDCConnectFrontendRedirectURL    string
	OIDCConnectTokenAuthMethod        string
	OIDCConnectUsePKCE                bool
	OIDCConnectValidateIDToken        bool
	OIDCConnectAllowedSigningAlgs     string
	OIDCConnectClockSkewSeconds       int
	OIDCConnectRequireEmailVerified   bool
	OIDCConnectUserInfoEmailPath      string
	OIDCConnectUserInfoIDPath         string
	OIDCConnectUserInfoUsernamePath   string

	GitHubOAuthEnabled                bool
	GitHubOAuthClientID               string
	GitHubOAuthClientSecret           string
	GitHubOAuthClientSecretConfigured bool
	GitHubOAuthRedirectURL            string
	GitHubOAuthFrontendRedirectURL    string
	GoogleOAuthEnabled                bool
	GoogleOAuthClientID               string
	GoogleOAuthClientSecret           string
	GoogleOAuthClientSecretConfigured bool
	GoogleOAuthRedirectURL            string
	GoogleOAuthFrontendRedirectURL    string

	TableDefaultPageSize        int
	TablePageSizeOptions        []int
	CustomEndpoints             string
	RiskControlEnabled          bool
	CyberSessionBlockEnabled    bool
	CyberSessionBlockTTLSeconds int
	AdminRechargeRebateEnabled  bool
	DefaultUserRPMLimit         int

	ChannelMonitorEnabled                bool `json:"channel_monitor_enabled"`
	ChannelMonitorDefaultIntervalSeconds int  `json:"channel_monitor_default_interval_seconds"`
	AvailableChannelsEnabled             bool `json:"available_channels_enabled"`

	EnableFingerprintUnification           bool
	EnableMetadataPassthrough              bool
	EnableCCHSigning                       bool
	EnableClaudeOAuthSystemPromptInjection bool
	ClaudeOAuthSystemPrompt                string
	ClaudeOAuthSystemPromptBlocks          string
	EnableAnthropicCacheTTL1hInjection     bool
	EnableClientDatelineNormalization      bool
	RewriteMessageCacheControl             bool
	AntigravityUserAgentVersion            string
	OpenAICodexUserAgent                   string
	MinCodexVersion                        string
	MaxCodexVersion                        string
	CodexCLIOnlyBlacklist                  string
	CodexCLIOnlyWhitelist                  string
	CodexCLIOnlyAllowAppServerClients      bool
	CodexCLIOnlyEngineFingerprintSignals   string
	WebSearchEmulationEnabled              bool

	PaymentVisibleMethodAlipaySource  string
	PaymentVisibleMethodWxpaySource   string
	PaymentVisibleMethodAlipayEnabled bool
	PaymentVisibleMethodWxpayEnabled  bool

	OpenAILowUpstreamRatePriorityEnabled                   bool
	OpenAIOAuthSchedulingRateMultiplier                    float64
	OpenAIAdvancedSchedulerEnabled                         bool
	OpenAIAdvancedSchedulerStickyWeightedEnabled           bool
	OpenAIAdvancedSchedulerSubscriptionPriorityEnabled     bool
	OpenAIAdvancedSchedulerLBTopK                          string
	OpenAIAdvancedSchedulerWeightPriority                  string
	OpenAIAdvancedSchedulerWeightLoad                      string
	OpenAIAdvancedSchedulerWeightQueue                     string
	OpenAIAdvancedSchedulerWeightErrorRate                 string
	OpenAIAdvancedSchedulerWeightTTFT                      string
	OpenAIAdvancedSchedulerWeightReset                     string
	OpenAIAdvancedSchedulerWeightQuotaHeadroom             string
	OpenAIAdvancedSchedulerWeightUpstreamCost              string
	OpenAIAdvancedSchedulerWeightPreviousResponse          string
	OpenAIAdvancedSchedulerWeightSessionSticky             string
	OpenAIAdvancedSchedulerEffectiveLBTopK                 string
	OpenAIAdvancedSchedulerEffectiveWeightPriority         string
	OpenAIAdvancedSchedulerEffectiveWeightLoad             string
	OpenAIAdvancedSchedulerEffectiveWeightQueue            string
	OpenAIAdvancedSchedulerEffectiveWeightErrorRate        string
	OpenAIAdvancedSchedulerEffectiveWeightTTFT             string
	OpenAIAdvancedSchedulerEffectiveWeightReset            string
	OpenAIAdvancedSchedulerEffectiveWeightQuotaHeadroom    string
	OpenAIAdvancedSchedulerEffectiveWeightUpstreamCost     string
	OpenAIAdvancedSchedulerEffectiveWeightPreviousResponse string
	OpenAIAdvancedSchedulerEffectiveWeightSessionSticky    string

	BalanceLowNotifyEnabled         bool
	BalanceLowNotifyThreshold       float64
	BalanceLowNotifyRechargeURL     string
	SubscriptionExpiryNotifyEnabled bool
	AccountQuotaNotifyEnabled       bool
	AccountQuotaNotifyEmails        []NotifyEmailEntry
	DefaultPlatformQuotas           map[string]*DefaultPlatformQuotaSetting `json:"default_platform_quotas"`
	AllowUserViewErrorRequests      bool
}

type DefaultSubscriptionSetting struct {
	GroupID      int64 `json:"group_id"`
	ValidityDays int   `json:"validity_days"`
}

type PublicSettings struct {
	RegistrationEnabled              bool
	EmailVerifyEnabled               bool
	RegistrationEmailSuffixWhitelist []string
	PromoCodeEnabled                 bool
	PasswordResetEnabled             bool
	InvitationCodeEnabled            bool
	TotpEnabled                      bool // TOTP 双因素认证
	TurnstileEnabled                 bool
	TurnstileSiteKey                 string
	SiteName                         string
	SiteLogo                         string
	SiteSubtitle                     string
	APIBaseURL                       string
	ContactInfo                      string
	DocURL                           string
	HomeContent                      string
	HideCcsImportButton              bool

	PurchaseSubscriptionEnabled bool
	PurchaseSubscriptionURL     string
	PurchaseLinkCNY10           string
	PurchaseLinkCNY30           string
	PurchaseLinkCNY100          string
	PaymentEnabled              bool
	SoraClientEnabled           bool
	CustomMenuItems             string // JSON array of custom menu items

	LinuxDoOAuthEnabled      bool
	WeChatOAuthEnabled       bool
	WeChatOAuthOpenEnabled   bool
	WeChatOAuthMPEnabled     bool
	WeChatOAuthMobileEnabled bool
	OIDCOAuthEnabled         bool
	OIDCOAuthProviderName    string

	ChannelMonitorEnabled                bool
	ChannelMonitorDefaultIntervalSeconds int
	AvailableChannelsEnabled             bool
	WebSearch                            PublicWorkspaceWebSearchSettings
	AffiliateEnabled                     bool

	BackendModeEnabled bool
	Version            string

	ForceEmailOnThirdPartySignup bool
	LoginAgreementEnabled        bool
	LoginAgreementMode           string
	LoginAgreementUpdatedAt      string
	LoginAgreementRevision       string
	LoginAgreementDocuments      []LoginAgreementDocument
	TableDefaultPageSize         int
	TablePageSizeOptions         []int
	CustomEndpoints              string
	DingTalkOAuthEnabled         bool
	GitHubOAuthEnabled           bool
	GoogleOAuthEnabled           bool
	BalanceLowNotifyEnabled      bool
	AccountQuotaNotifyEnabled    bool
	BalanceLowNotifyThreshold    float64
	BalanceLowNotifyRechargeURL  string
	RiskControlEnabled           bool `json:"risk_control_enabled"`
	AllowUserViewErrorRequests   bool `json:"allow_user_view_error_requests"`
}

type PublicWorkspaceWebSearchSettings struct {
	Available bool   `json:"available"`
	Provider  string `json:"provider,omitempty"`
}

// SoraS3Settings Sora S3 存储配置
type SoraS3Settings struct {
	Enabled                   bool   `json:"enabled"`
	Endpoint                  string `json:"endpoint"`
	Region                    string `json:"region"`
	Bucket                    string `json:"bucket"`
	AccessKeyID               string `json:"access_key_id"`
	SecretAccessKey           string `json:"secret_access_key"`            // 仅内部使用，不直接返回前端
	SecretAccessKeyConfigured bool   `json:"secret_access_key_configured"` // 前端展示用
	Prefix                    string `json:"prefix"`
	ForcePathStyle            bool   `json:"force_path_style"`
	CDNURL                    string `json:"cdn_url"`
	DefaultStorageQuotaBytes  int64  `json:"default_storage_quota_bytes"`
}

// SoraS3Profile Sora S3 多配置项（服务内部模型）
type SoraS3Profile struct {
	ProfileID                 string `json:"profile_id"`
	Name                      string `json:"name"`
	IsActive                  bool   `json:"is_active"`
	Enabled                   bool   `json:"enabled"`
	Endpoint                  string `json:"endpoint"`
	Region                    string `json:"region"`
	Bucket                    string `json:"bucket"`
	AccessKeyID               string `json:"access_key_id"`
	SecretAccessKey           string `json:"-"`                            // 仅内部使用，不直接返回前端
	SecretAccessKeyConfigured bool   `json:"secret_access_key_configured"` // 前端展示用
	Prefix                    string `json:"prefix"`
	ForcePathStyle            bool   `json:"force_path_style"`
	CDNURL                    string `json:"cdn_url"`
	DefaultStorageQuotaBytes  int64  `json:"default_storage_quota_bytes"`
	UpdatedAt                 string `json:"updated_at"`
}

// SoraS3ProfileList Sora S3 多配置列表
type SoraS3ProfileList struct {
	ActiveProfileID string          `json:"active_profile_id"`
	Items           []SoraS3Profile `json:"items"`
}

// TLSFingerprintSettings TLS 指纹全局设置。
type TLSFingerprintSettings struct {
	Enabled bool `json:"enabled"`
}

// TLSFingerprintProfile TLS 指纹 Profile（服务内部模型）。
type TLSFingerprintProfile struct {
	ProfileID    string   `json:"profile_id"`
	Name         string   `json:"name"`
	Enabled      bool     `json:"enabled"`
	EnableGREASE bool     `json:"enable_grease"`
	CipherSuites []uint16 `json:"cipher_suites"`
	Curves       []uint16 `json:"curves"`
	PointFormats []uint8  `json:"point_formats"`
	UpdatedAt    string   `json:"updated_at"`
}

// TLSFingerprintProfileList TLS 指纹配置列表。
type TLSFingerprintProfileList struct {
	Enabled bool                    `json:"enabled"`
	Items   []TLSFingerprintProfile `json:"items"`
}

// StreamTimeoutSettings 流超时处理配置（仅控制超时后的处理方式，超时判定由网关配置控制）
type StreamTimeoutSettings struct {
	// Enabled 是否启用流超时处理
	Enabled bool `json:"enabled"`
	// Action 超时后的处理方式: "temp_unsched" | "error" | "none"
	Action string `json:"action"`
	// TempUnschedMinutes 临时不可调度持续时间（分钟）
	TempUnschedMinutes int `json:"temp_unsched_minutes"`
	// ThresholdCount 触发阈值次数（累计多少次超时才触发）
	ThresholdCount int `json:"threshold_count"`
	// ThresholdWindowMinutes 阈值窗口时间（分钟）
	ThresholdWindowMinutes int `json:"threshold_window_minutes"`
}

// StreamTimeoutAction 流超时处理方式常量
const (
	StreamTimeoutActionTempUnsched = "temp_unsched" // 临时不可调度
	StreamTimeoutActionError       = "error"        // 标记为错误状态
	StreamTimeoutActionNone        = "none"         // 不处理
)

// DefaultStreamTimeoutSettings 返回默认的流超时配置
func DefaultStreamTimeoutSettings() *StreamTimeoutSettings {
	return &StreamTimeoutSettings{
		Enabled:                false,
		Action:                 StreamTimeoutActionTempUnsched,
		TempUnschedMinutes:     5,
		ThresholdCount:         3,
		ThresholdWindowMinutes: 10,
	}
}

// RectifierSettings 请求整流器配置
type RectifierSettings struct {
	Enabled                  bool     `json:"enabled"`                    // 总开关
	ThinkingSignatureEnabled bool     `json:"thinking_signature_enabled"` // Thinking 签名整流
	ThinkingBudgetEnabled    bool     `json:"thinking_budget_enabled"`    // Thinking Budget 整流
	APIKeySignatureEnabled   bool     `json:"apikey_signature_enabled"`   // API Key 签名整流开关
	APIKeySignaturePatterns  []string `json:"apikey_signature_patterns"`  // API Key 自定义匹配关键词
}

// DefaultRectifierSettings 返回默认的整流器配置（全部启用）
func DefaultRectifierSettings() *RectifierSettings {
	return &RectifierSettings{
		Enabled:                  true,
		ThinkingSignatureEnabled: true,
		ThinkingBudgetEnabled:    true,
	}
}

// Beta Policy 策略常量
const (
	BetaPolicyActionPass   = "pass"   // 透传，不做任何处理
	BetaPolicyActionFilter = "filter" // 过滤，从 beta header 中移除该 token
	BetaPolicyActionBlock  = "block"  // 拦截，直接返回错误

	BetaPolicyScopeAll     = "all"     // 所有账号类型
	BetaPolicyScopeOAuth   = "oauth"   // 仅 OAuth 账号
	BetaPolicyScopeAPIKey  = "apikey"  // 仅 API Key 账号
	BetaPolicyScopeBedrock = "bedrock" // 仅 AWS Bedrock 账号
)

// BetaPolicyRule 单条 Beta 策略规则
type BetaPolicyRule struct {
	BetaToken            string   `json:"beta_token"`                       // beta token 值
	Action               string   `json:"action"`                           // "pass" | "filter" | "block"
	Scope                string   `json:"scope"`                            // "all" | "oauth" | "apikey" | "bedrock"
	ErrorMessage         string   `json:"error_message,omitempty"`          // 自定义错误消息 (action=block 时生效)
	ModelWhitelist       []string `json:"model_whitelist,omitempty"`        // 模型匹配模式列表（为空=对所有模型生效）
	FallbackAction       string   `json:"fallback_action,omitempty"`        // 未匹配白名单的模型的处理方式
	FallbackErrorMessage string   `json:"fallback_error_message,omitempty"` // 未匹配白名单时的自定义错误消息
}

// BetaPolicySettings Beta 策略配置
type BetaPolicySettings struct {
	Rules []BetaPolicyRule `json:"rules"`
}

// OverloadCooldownSettings 529过载冷却配置
type OverloadCooldownSettings struct {
	// Enabled 是否在收到529时暂停账号调度
	Enabled bool `json:"enabled"`
	// CooldownMinutes 冷却时长（分钟）
	CooldownMinutes int `json:"cooldown_minutes"`
}

// DefaultOverloadCooldownSettings 返回默认的过载冷却配置（启用，10分钟）
func DefaultOverloadCooldownSettings() *OverloadCooldownSettings {
	return &OverloadCooldownSettings{
		Enabled:         true,
		CooldownMinutes: 10,
	}
}

// DefaultBetaPolicySettings 返回默认的 Beta 策略配置
func DefaultBetaPolicySettings() *BetaPolicySettings {
	return &BetaPolicySettings{
		Rules: []BetaPolicyRule{
			{
				BetaToken: "fast-mode-2026-02-01",
				Action:    BetaPolicyActionFilter,
				Scope:     BetaPolicyScopeAll,
			},
			{
				BetaToken: "context-1m-2025-08-07",
				Action:    BetaPolicyActionPass,
				Scope:     BetaPolicyScopeAll,
				ModelWhitelist: []string{
					"claude-sonnet-5",
					"claude-sonnet-5-*",
					"claude-sonnet-5@*",
					"us.anthropic.claude-sonnet-5*",
					"eu.anthropic.claude-sonnet-5*",
					"apac.anthropic.claude-sonnet-5*",
					"jp.anthropic.claude-sonnet-5*",
					"au.anthropic.claude-sonnet-5*",
					"us-gov.anthropic.claude-sonnet-5*",
					"global.anthropic.claude-sonnet-5*",
					"anthropic.claude-sonnet-5*",
				},
				FallbackAction: BetaPolicyActionFilter,
			},
		},
	}
}
