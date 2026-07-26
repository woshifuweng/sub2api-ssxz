package service

import "time"

// APIKeyAuthSnapshot API Key 认证缓存快照（仅包含认证所需字段）
type APIKeyAuthSnapshot struct {
	// Version 快照 schema 版本（apiKeyAuthSnapshotVersion）。版本不符的缓存条目
	// 按 miss 处理并回源重建，避免结构演进后旧条目以零值字段继续生效。
	Version       int                       `json:"version"`
	APIKeyID      int64                     `json:"api_key_id"`
	UserID        int64                     `json:"user_id"`
	GroupID       *int64                    `json:"group_id,omitempty"`
	Name          string                    `json:"name"`
	GroupIDs      []int64                   `json:"group_ids,omitempty"`
	AllowedModels []string                  `json:"allowed_models,omitempty"`
	Status        string                    `json:"status"`
	IPWhitelist   []string                  `json:"ip_whitelist,omitempty"`
	IPBlacklist   []string                  `json:"ip_blacklist,omitempty"`
	User          APIKeyAuthUserSnapshot    `json:"user"`
	Group         *APIKeyAuthGroupSnapshot  `json:"group,omitempty"`
	Groups        []APIKeyAuthGroupSnapshot `json:"groups,omitempty"`

	// Quota fields for API Key independent quota feature
	Quota     float64 `json:"quota"`      // Quota limit in USD (0 = unlimited)
	QuotaUsed float64 `json:"quota_used"` // Used quota amount

	// Expiration field for API Key expiration feature
	ExpiresAt *time.Time `json:"expires_at,omitempty"` // Expiration time (nil = never expires)

	// Rate limit configuration (only limits, not usage - usage read from Redis at check time)
	RateLimit5h float64 `json:"rate_limit_5h"`
	RateLimit1d float64 `json:"rate_limit_1d"`
	RateLimit7d float64 `json:"rate_limit_7d"`
}

// APIKeyAuthUserSnapshot 用户快照
type APIKeyAuthUserSnapshot struct {
	ID          int64   `json:"id"`
	Status      string  `json:"status"`
	Role        string  `json:"role"`
	Balance     float64 `json:"balance"`
	Concurrency int     `json:"concurrency"`
	// AllowedGroups 是独占分组准入白名单。缓存命中路径上 CanBindGroup 依赖它，
	// 缺失会让独占分组校验静默放行（e5c51dce9 整合曾丢过该字段）。
	AllowedGroups []int64 `json:"allowed_groups,omitempty"`
}

// APIKeyAuthGroupSnapshot 分组快照
type APIKeyAuthGroupSnapshot struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Platform string `json:"platform"`
	// IsExclusive 独占分组标记。缓存命中路径上 validateAPIKeyGroupAllowed 依赖它，
	// 缺失会让 CanBindGroup 第二参恒为 false 直接短路（e5c51dce9 整合曾丢过该字段）。
	IsExclusive                     bool     `json:"is_exclusive"`
	Status                          string   `json:"status"`
	SubscriptionType                string   `json:"subscription_type"`
	RateMultiplier                  float64  `json:"rate_multiplier"`
	PeakRateEnabled                 bool     `json:"peak_rate_enabled"`
	PeakStart                       string   `json:"peak_start"`
	PeakEnd                         string   `json:"peak_end"`
	PeakRateMultiplier              float64  `json:"peak_rate_multiplier"`
	DailyLimitUSD                   *float64 `json:"daily_limit_usd,omitempty"`
	WeeklyLimitUSD                  *float64 `json:"weekly_limit_usd,omitempty"`
	MonthlyLimitUSD                 *float64 `json:"monthly_limit_usd,omitempty"`
	ImagePrice1K                    *float64 `json:"image_price_1k,omitempty"`
	ImagePrice2K                    *float64 `json:"image_price_2k,omitempty"`
	ImagePrice4K                    *float64 `json:"image_price_4k,omitempty"`
	SoraImagePrice360               *float64 `json:"sora_image_price_360,omitempty"`
	SoraImagePrice540               *float64 `json:"sora_image_price_540,omitempty"`
	SoraVideoPricePerRequest        *float64 `json:"sora_video_price_per_request,omitempty"`
	SoraVideoPricePerRequestHD      *float64 `json:"sora_video_price_per_request_hd,omitempty"`
	WebSearchPricePerCall           *float64 `json:"web_search_price_per_call,omitempty"`
	ClaudeCodeOnly                  bool     `json:"claude_code_only"`
	FallbackGroupID                 *int64   `json:"fallback_group_id,omitempty"`
	FallbackGroupIDOnInvalidRequest *int64   `json:"fallback_group_id_on_invalid_request,omitempty"`

	// Model routing is used by gateway account selection, so it must be part of auth cache snapshot.
	// Only anthropic groups use these fields; others may leave them empty.
	ModelRouting         map[string][]int64 `json:"model_routing,omitempty"`
	ModelRoutingEnabled  bool               `json:"model_routing_enabled"`
	MCPXMLInject         bool               `json:"mcp_xml_inject"`
	AllowImageGeneration bool               `json:"allow_image_generation"`

	// 支持的模型系列（仅 antigravity 平台使用）
	SupportedModelScopes []string `json:"supported_model_scopes,omitempty"`

	// OpenAI Messages 调度配置（仅 openai 平台使用）
	AllowMessagesDispatch       bool                              `json:"allow_messages_dispatch"`
	AllowLive                   bool                              `json:"allow_live"`
	DefaultMappedModel          string                            `json:"default_mapped_model,omitempty"`
	MessagesDispatchModelConfig OpenAIMessagesDispatchModelConfig `json:"messages_dispatch_model_config,omitempty"`

	// MaxReasoningEffort OpenAI/Codex 请求的推理强度上限，空字符串表示不限制。
	MaxReasoningEffort string `json:"max_reasoning_effort,omitempty"`
	// ReasoningEffortMappings rewrites explicit effort values before the ceiling.
	ReasoningEffortMappings []ReasoningEffortMapping `json:"reasoning_effort_mappings"`
}

// APIKeyAuthCacheEntry 缓存条目，支持负缓存
type APIKeyAuthCacheEntry struct {
	NotFound bool                `json:"not_found"`
	Snapshot *APIKeyAuthSnapshot `json:"snapshot,omitempty"`
}
