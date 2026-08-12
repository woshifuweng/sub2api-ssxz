package service

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

type HTTPUpstreamMetricsSnapshot struct {
	IsolationMode          string `json:"isolation_mode"`
	ActiveClients          int    `json:"active_clients"`
	CacheHitTotal          int64  `json:"cache_hit_total"`
	CacheMissTotal         int64  `json:"cache_miss_total"`
	ClientCreateTotal      int64  `json:"client_create_total"`
	EvictIdleTotal         int64  `json:"evict_idle_total"`
	EvictLRUTotal          int64  `json:"evict_lru_total"`
	EvictConfigChangeTotal int64  `json:"evict_config_change_total"`
	LimitRejectTotal       int64  `json:"limit_reject_total"`
}

type HTTPUpstreamMetricsProvider interface {
	SnapshotMetrics() HTTPUpstreamMetricsSnapshot
}

type LoginAgreementDocument struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	ContentMD string `json:"content_md"`
}

type AdminBindAuthIdentityInput struct {
	ProviderType    string
	ProviderKey     string
	ProviderSubject string
	Issuer          *string
	Metadata        map[string]any
	Channel         *AdminBindAuthIdentityChannelInput
}

type AdminBindAuthIdentityChannelInput struct {
	Channel        string
	ChannelAppID   string
	ChannelSubject string
	Metadata       map[string]any
}

type AdminBoundAuthIdentity struct {
	UserID          int64                          `json:"user_id"`
	ProviderType    string                         `json:"provider_type"`
	ProviderKey     string                         `json:"provider_key"`
	ProviderSubject string                         `json:"provider_subject"`
	VerifiedAt      *time.Time                     `json:"verified_at,omitempty"`
	Issuer          *string                        `json:"issuer,omitempty"`
	Metadata        map[string]any                 `json:"metadata"`
	CreatedAt       time.Time                      `json:"created_at"`
	UpdatedAt       time.Time                      `json:"updated_at"`
	Channel         *AdminBoundAuthIdentityChannel `json:"channel,omitempty"`
}

type AdminBoundAuthIdentityChannel struct {
	Channel        string         `json:"channel"`
	ChannelAppID   string         `json:"channel_app_id"`
	ChannelSubject string         `json:"channel_subject"`
	Metadata       map[string]any `json:"metadata"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

type ShadowOptions struct {
	Name        string
	Priority    int
	Concurrency int
	GroupIDs    []int64
}

type BulkUpdateAccountFilters struct {
	Platform    string
	Type        string
	Status      string
	Group       string
	Search      string
	PrivacyMode string
}

type UserRPMStatus struct {
	UserRPMUsed  int                  `json:"user_rpm_used"`
	UserRPMLimit int                  `json:"user_rpm_limit"`
	PerGroup     []UserGroupRPMStatus `json:"per_group"`
}

type UserGroupRPMStatus struct {
	GroupID   int64  `json:"group_id"`
	GroupName string `json:"group_name"`
	Used      int    `json:"used"`
	Limit     int    `json:"limit"`
	Source    string `json:"source"`
}

type WebSearchManagerBuilder func(cfg *WebSearchEmulationConfig, proxyURLs map[int64]string)

type OpenAIFastPolicyRule struct {
	ServiceTier          string   `json:"service_tier"`
	Action               string   `json:"action"`
	Scope                string   `json:"scope"`
	UserIDs              []int64  `json:"user_ids,omitempty"`
	ErrorMessage         string   `json:"error_message,omitempty"`
	ModelWhitelist       []string `json:"model_whitelist,omitempty"`
	FallbackAction       string   `json:"fallback_action,omitempty"`
	FallbackErrorMessage string   `json:"fallback_error_message,omitempty"`
}

type OpenAIFastPolicySettings struct {
	Rules []OpenAIFastPolicyRule `json:"rules"`
}

type RateLimit429CooldownSettings struct {
	Enabled         bool `json:"enabled"`
	CooldownSeconds int  `json:"cooldown_seconds"`
}

// BatchImageBalanceHoldCommand describes the idempotent image-billing hold
// operation used by the repository layer.
type BatchImageBalanceHoldCommand struct {
	RequestID          string
	APIKeyID           int64
	RequestFingerprint string
	RequestPayloadHash string
	UserID             int64
	BatchID            string
	HoldAmount         float64
	ActualAmount       float64
}

func (c *BatchImageBalanceHoldCommand) Normalize() {
	if c == nil {
		return
	}
	c.RequestID = strings.TrimSpace(c.RequestID)
	c.BatchID = strings.TrimSpace(c.BatchID)
	if strings.TrimSpace(c.RequestFingerprint) == "" {
		c.RequestFingerprint = buildBatchImageBalanceHoldFingerprint(c)
	}
}

func buildBatchImageBalanceHoldFingerprint(c *BatchImageBalanceHoldCommand) string {
	if c == nil {
		return ""
	}
	raw := fmt.Sprintf(
		"%d|%d|%s|%0.10f|%0.10f",
		c.UserID,
		c.APIKeyID,
		strings.TrimSpace(c.BatchID),
		c.HoldAmount,
		c.ActualAmount,
	)
	if payloadHash := strings.TrimSpace(c.RequestPayloadHash); payloadHash != "" {
		raw += "|" + payloadHash
	}
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

type BatchImageBalanceHoldResult struct {
	Applied       bool
	NewBalance    *float64
	FrozenBalance *float64
}

type OpsRetryAttempt struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"created_at"`

	RequestedByUserID int64  `json:"requested_by_user_id"`
	SourceErrorID     int64  `json:"source_error_id"`
	Mode              string `json:"mode"`
	PinnedAccountID   *int64 `json:"pinned_account_id"`
	PinnedAccountName string `json:"pinned_account_name"`

	Status     string     `json:"status"`
	StartedAt  *time.Time `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at"`
	DurationMs *int64     `json:"duration_ms"`

	Success           *bool   `json:"success"`
	HTTPStatusCode    *int    `json:"http_status_code"`
	UpstreamRequestID *string `json:"upstream_request_id"`
	UsedAccountID     *int64  `json:"used_account_id"`
	UsedAccountName   string  `json:"used_account_name"`
	ResponsePreview   *string `json:"response_preview"`
	ResponseTruncated *bool   `json:"response_truncated"`

	ResultRequestID *string `json:"result_request_id"`
	ResultErrorID   *int64  `json:"result_error_id"`
	ErrorMessage    *string `json:"error_message"`
}

type OpsRetryResult struct {
	AttemptID int64  `json:"attempt_id"`
	Mode      string `json:"mode"`
	Status    string `json:"status"`

	PinnedAccountID *int64 `json:"pinned_account_id"`
	UsedAccountID   *int64 `json:"used_account_id"`

	HTTPStatusCode    int    `json:"http_status_code"`
	UpstreamRequestID string `json:"upstream_request_id"`
	ResponsePreview   string `json:"response_preview"`
	ResponseTruncated bool   `json:"response_truncated"`
	ErrorMessage      string `json:"error_message"`

	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
	DurationMs int64     `json:"duration_ms"`
}

type DefaultPlatformQuotaSetting struct {
	DailyLimitUSD   *float64 `json:"daily"`
	WeeklyLimitUSD  *float64 `json:"weekly"`
	MonthlyLimitUSD *float64 `json:"monthly"`
}

type authSourceDefaultKeySet struct {
	source           string
	balance          string
	concurrency      string
	subscriptions    string
	grantOnSignup    string
	grantOnFirstBind string
	platformQuotas   string
}
