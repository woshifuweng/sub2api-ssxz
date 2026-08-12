package service

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"golang.org/x/sync/singleflight"
)

var (
	ErrRegistrationDisabled          = infraerrors.Forbidden("REGISTRATION_DISABLED", "registration is currently disabled")
	ErrSettingNotFound               = infraerrors.NotFound("SETTING_NOT_FOUND", "setting not found")
	ErrSoraS3ProfileNotFound         = infraerrors.NotFound("SORA_S3_PROFILE_NOT_FOUND", "sora s3 profile not found")
	ErrSoraS3ProfileExists           = infraerrors.Conflict("SORA_S3_PROFILE_EXISTS", "sora s3 profile already exists")
	ErrTLSFingerprintProfileNotFound = infraerrors.NotFound("TLS_FINGERPRINT_PROFILE_NOT_FOUND", "tls fingerprint profile not found")
	ErrTLSFingerprintProfileExists   = infraerrors.Conflict("TLS_FINGERPRINT_PROFILE_EXISTS", "tls fingerprint profile already exists")
	ErrDefaultSubGroupInvalid        = infraerrors.BadRequest(
		"DEFAULT_SUBSCRIPTION_GROUP_INVALID",
		"default subscription group must exist and be subscription type",
	)
	ErrDefaultSubGroupDuplicate = infraerrors.BadRequest(
		"DEFAULT_SUBSCRIPTION_GROUP_DUPLICATE",
		"default subscription group cannot be duplicated",
	)
)

type SettingRepository interface {
	Get(ctx context.Context, key string) (*Setting, error)
	GetValue(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string) error
	GetMultiple(ctx context.Context, keys []string) (map[string]string, error)
	SetMultiple(ctx context.Context, settings map[string]string) error
	GetAll(ctx context.Context) (map[string]string, error)
	Delete(ctx context.Context, key string) error
}

// cachedVersionBounds 缓存 Claude Code 版本号上下限（进程内缓存，60s TTL）
type cachedVersionBoundsLegacy struct {
	min       string // 空字符串 = 不检查
	max       string // 空字符串 = 不检查
	expiresAt int64  // unix nano
}

// versionBoundsCache 版本号上下限进程内缓存
var versionBoundsCacheLegacy atomic.Value // *cachedVersionBoundsLegacy

// versionBoundsSF 防止缓存过期时 thundering herd
var versionBoundsSFLegacy singleflight.Group

// versionBoundsCacheTTL 缓存有效期
const versionBoundsCacheTTLLegacy = 60 * time.Second

// versionBoundsErrorTTL DB 错误时的短缓存，快速重试
const versionBoundsErrorTTLLegacy = 5 * time.Second

// versionBoundsDBTimeout singleflight 内 DB 查询超时，独立于请求 context
const versionBoundsDBTimeoutLegacy = 5 * time.Second

// cachedBackendMode Backend Mode cache (in-process, 60s TTL)
type cachedBackendModeLegacy struct {
	value     bool
	expiresAt int64 // unix nano
}

var backendModeCacheLegacy atomic.Value // *cachedBackendModeLegacy
var backendModeSFLegacy singleflight.Group

const backendModeCacheTTLLegacy = 60 * time.Second
const backendModeErrorTTLLegacy = 5 * time.Second
const backendModeDBTimeoutLegacy = 5 * time.Second
const settingHotCacheTTL = 60 * time.Second
const settingHotErrorTTL = 5 * time.Second
const settingHotDBTimeout = 5 * time.Second

const adminAPIKeyRecordVersion = 2

var (
	homeContentScriptTagPattern  = regexp.MustCompile(`(?is)<script\b[^>]*>.*?</script\s*>`)
	homeContentEventAttrPattern  = regexp.MustCompile(`(?i)\son[a-z0-9_-]+\s*=\s*(".*?"|'.*?'|[^\s>]+)`)
	homeContentJavascriptPattern = regexp.MustCompile(`(?i)javascript:`)
)

type adminAPIKeyRecord struct {
	Version           int    `json:"version"`
	KeyHash           string `json:"key_hash"`
	MaskedKey         string `json:"masked_key"`
	AdminUserID       int64  `json:"admin_user_id"`
	AdminTokenVersion int64  `json:"admin_token_version"`
}

type AdminAPIKeyBinding struct {
	AdminUserID       int64
	AdminTokenVersion int64
	Legacy            bool
}

type settingHotCacheEntry struct {
	Value     string
	Found     bool
	ExpiresAt int64
}

var settingHotKeys = map[string]struct{}{
	SettingKeyOpsMonitoringEnabled:         {},
	SettingKeyOpsRealtimeMonitoringEnabled: {},
	SettingKeyStreamTimeoutSettings:        {},
	SettingKeyAutoDelete401Accounts:        {},
	SettingKeyAutoDelete429Accounts:        {},
	SettingKeyAutoDeleteUselessProxies:     {},
	SettingKeyAllowUngroupedKeyScheduling:  {},
	SettingKeyFrontendURL:                  {},
	SettingKeyAPIBaseURL:                   {},
	SettingKeyAffiliateEnabled:             {},
}

// DefaultSubscriptionGroupReader validates group references used by default subscriptions.
type DefaultSubscriptionGroupReader interface {
	GetByID(ctx context.Context, id int64) (*Group, error)
}

// SettingService 系统设置服务
type SettingService struct {
	settingRepo                 SettingRepository
	defaultSubGroupReader       DefaultSubscriptionGroupReader
	proxyRepo                   ProxyRepository // for resolving websearch provider proxy URLs
	cfg                         *config.Config
	onUpdate                    func() // Callback when settings are updated (for cache invalidation)
	onS3Update                  func() // Callback when Sora S3 settings are updated
	onTLSFingerprintUpdate      func() // Callback when TLS fingerprint settings are updated
	version                     string // Application version
	hotSettings                 sync.Map
	hotSettingsSF               singleflight.Group
	webSearchManagerBuilder     WebSearchManagerBuilder
	antigravityUAVersionCache   atomic.Value // *cachedAntigravityUserAgentVersion
	antigravityUAVersionSF      singleflight.Group
	openAICodexUACache          atomic.Value // *cachedOpenAICodexUserAgent
	openAICodexUASF             singleflight.Group
	openAICodexVersionCache     atomic.Value // *cachedOpenAICodexClientVersion
	openAICodexVersionSF        singleflight.Group
	codexRestrictionPolicyCache atomic.Value // *cachedCodexRestrictionPolicy
	codexRestrictionPolicySF    singleflight.Group
	backendModeCache            atomic.Value // *cachedBackendMode
	backendModeSF               singleflight.Group

	cyberSessionBlockRuntimeCache atomic.Value // *cachedCyberSessionBlockRuntime
	cyberSessionBlockRuntimeSF    singleflight.Group

	// panelRateLimitCache 面板 API 限流配置进程内缓存（*cachedPanelRateLimitSettings）。
	// 面板每个认证请求都会读取，禁止在热路径上直接访问 DB。
	panelRateLimitCache atomic.Value
	panelRateLimitSF    singleflight.Group

	// openAIQuotaAutoPauseSettingsCache holds the most recently observed quota auto-pause
	// settings. GetOpenAIQuotaAutoPauseSettings reads this atomic.Value on the request hot
	// path without ever blocking on the DB; when the cached entry expires, a background
	// goroutine refreshes it via openAIQuotaAutoPauseSettingsSF (stale-while-revalidate).
	// This per-service field also gives tests natural isolation — each SettingService
	// instance owns its own cache, no shared package-level state.
	openAIQuotaAutoPauseSettingsCache atomic.Value // *cachedOpenAIQuotaAutoPauseSettings
	openAIQuotaAutoPauseSettingsSF    singleflight.Group
}

// NewSettingService 创建系统设置服务实例
func NewSettingService(settingRepo SettingRepository, cfg *config.Config) *SettingService {
	return &SettingService{
		settingRepo: settingRepo,
		cfg:         cfg,
	}
}

// SetProxyRepository injects a proxy repo for resolving websearch provider proxy URLs.
func (s *SettingService) SetProxyRepository(repo ProxyRepository) {
	s.proxyRepo = repo
}

func (s *SettingService) isHotSettingKey(key string) bool {
	_, ok := settingHotKeys[key]
	return ok
}

func (s *SettingService) invalidateHotSettingKeys(keys ...string) {
	if s == nil {
		return
	}
	if len(keys) == 0 {
		s.hotSettings.Range(func(key, _ any) bool {
			s.hotSettings.Delete(key)
			return true
		})
		return
	}
	for _, key := range keys {
		if strings.TrimSpace(key) == "" {
			continue
		}
		s.hotSettings.Delete(key)
	}
}

func (s *SettingService) getSettingValueCached(ctx context.Context, key string) (string, error) {
	if s == nil || s.settingRepo == nil {
		return "", ErrSettingNotFound
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if !s.isHotSettingKey(key) {
		return s.settingRepo.GetValue(ctx, key)
	}
	if cached, ok := s.hotSettings.Load(key); ok {
		if entry, ok := cached.(*settingHotCacheEntry); ok && entry != nil && time.Now().UnixNano() < entry.ExpiresAt {
			if entry.Found {
				return entry.Value, nil
			}
			return "", ErrSettingNotFound
		}
	}

	value, err, _ := s.hotSettingsSF.Do(key, func() (any, error) {
		if cached, ok := s.hotSettings.Load(key); ok {
			if entry, ok := cached.(*settingHotCacheEntry); ok && entry != nil && time.Now().UnixNano() < entry.ExpiresAt {
				return entry, nil
			}
		}
		dbCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), settingHotDBTimeout)
		defer cancel()

		loaded, loadErr := s.settingRepo.GetValue(dbCtx, key)
		if loadErr != nil {
			if errors.Is(loadErr, ErrSettingNotFound) {
				entry := &settingHotCacheEntry{
					Found:     false,
					ExpiresAt: time.Now().Add(settingHotCacheTTL).UnixNano(),
				}
				s.hotSettings.Store(key, entry)
				return entry, nil
			}
			return nil, loadErr
		}
		entry := &settingHotCacheEntry{
			Value:     loaded,
			Found:     true,
			ExpiresAt: time.Now().Add(settingHotCacheTTL).UnixNano(),
		}
		s.hotSettings.Store(key, entry)
		return entry, nil
	})
	if err != nil {
		return "", err
	}
	entry, _ := value.(*settingHotCacheEntry)
	if entry == nil || !entry.Found {
		return "", ErrSettingNotFound
	}
	return entry.Value, nil
}

func (s *SettingService) setSettingValue(ctx context.Context, key, value string) error {
	if s == nil || s.settingRepo == nil {
		return ErrSettingNotFound
	}
	if err := s.settingRepo.Set(ctx, key, value); err != nil {
		return err
	}
	s.invalidateHotSettingKeys(key)
	if s.onUpdate != nil {
		s.onUpdate()
	}
	return nil
}

func (s *SettingService) setMultipleSettingValues(ctx context.Context, updates map[string]string) error {
	if s == nil || s.settingRepo == nil {
		return ErrSettingNotFound
	}
	if err := s.settingRepo.SetMultiple(ctx, updates); err != nil {
		return err
	}
	keys := make([]string, 0, len(updates))
	for key := range updates {
		keys = append(keys, key)
	}
	s.invalidateHotSettingKeys(keys...)
	if s.onUpdate != nil {
		s.onUpdate()
	}
	return nil
}

func (s *SettingService) deleteSettingValue(ctx context.Context, key string) error {
	if s == nil || s.settingRepo == nil {
		return ErrSettingNotFound
	}
	if err := s.settingRepo.Delete(ctx, key); err != nil {
		return err
	}
	s.invalidateHotSettingKeys(key)
	if s.onUpdate != nil {
		s.onUpdate()
	}
	return nil
}

// SetDefaultSubscriptionGroupReader injects an optional group reader for default subscription validation.
func (s *SettingService) SetDefaultSubscriptionGroupReader(reader DefaultSubscriptionGroupReader) {
	s.defaultSubGroupReader = reader
}

// GetAllSettings 获取所有系统设置
func (s *SettingService) GetAllSettings(ctx context.Context) (*SystemSettings, error) {
	settings, err := s.settingRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("get all settings: %w", err)
	}

	return s.parseSettings(settings), nil
}

// GetFrontendURL 获取前端基础URL（数据库优先，fallback 到配置文件）
func (s *SettingService) GetFrontendURLLegacy(ctx context.Context) string {
	val, err := s.getSettingValueCached(ctx, SettingKeyFrontendURL)
	if err == nil && strings.TrimSpace(val) != "" {
		return strings.TrimSpace(val)
	}
	return s.cfg.Server.FrontendURL
}

// ResolvePasswordResetLinkBase resolves the public base URL used in password reset links.
func (s *SettingService) ResolvePasswordResetLinkBase(ctx context.Context, requestOrigin string) string {
	return ResolvePasswordResetBaseURL(s.GetFrontendURLLegacy(ctx), requestOrigin, s.cfg)
}

// GetPublicSettings 获取公开设置（无需登录）
func (s *SettingService) GetPublicSettingsLegacy(ctx context.Context) (*PublicSettings, error) {
	keys := []string{
		SettingKeyRegistrationEnabled,
		SettingKeyEmailVerifyEnabled,
		SettingKeyRegistrationEmailSuffixWhitelist,
		SettingKeyPromoCodeEnabled,
		SettingKeyPasswordResetEnabled,
		SettingKeyInvitationCodeEnabled,
		SettingKeyTotpEnabled,
		SettingKeyTurnstileEnabled,
		SettingKeyTurnstileSiteKey,
		SettingKeySiteName,
		SettingKeySiteLogo,
		SettingKeySiteSubtitle,
		SettingKeyAPIBaseURL,
		SettingKeyContactInfo,
		SettingKeyDocURL,
		SettingKeyHomeContent,
		SettingKeyHideCcsImportButton,
		SettingKeyPurchaseSubscriptionEnabled,
		SettingKeyPurchaseSubscriptionURL,
		SettingKeyPurchaseLinkCNY10,
		SettingKeyPurchaseLinkCNY30,
		SettingKeyPurchaseLinkCNY100,
		SettingPaymentEnabled,
		SettingKeySoraClientEnabled,
		SettingKeyCustomMenuItems,
		SettingKeyLinuxDoConnectEnabled,
		settingKeyWeChatConnectEnabled,
		settingKeyWeChatConnectOpenEnabled,
		settingKeyWeChatConnectMPEnabled,
		settingKeyWeChatConnectMobileEnabled,
		settingKeyOIDCConnectEnabled,
		settingKeyOIDCConnectProviderName,
		settingKeyChannelMonitorEnabled,
		settingKeyAvailableChannelsEnabled,
		SettingKeyAffiliateEnabled,
		SettingKeyBackendModeEnabled,
	}

	settings, err := s.settingRepo.GetMultiple(ctx, keys)
	if err != nil {
		return nil, fmt.Errorf("get public settings: %w", err)
	}

	linuxDoEnabled := false
	if raw, ok := settings[SettingKeyLinuxDoConnectEnabled]; ok {
		linuxDoEnabled = raw == "true"
	} else {
		linuxDoEnabled = s.cfg != nil && s.cfg.LinuxDo.Enabled
	}

	// Password reset requires email verification to be enabled
	emailVerifyEnabled := settings[SettingKeyEmailVerifyEnabled] == "true"
	passwordResetEnabled := emailVerifyEnabled && settings[SettingKeyPasswordResetEnabled] == "true"
	registrationEmailSuffixWhitelist := ParseRegistrationEmailSuffixWhitelist(
		settings[SettingKeyRegistrationEmailSuffixWhitelist],
	)
	weChatOAuth := s.effectiveWeChatConnectOAuthConfig(settings)
	oidcProviderName := ""
	oidcEnabled := false
	if s != nil && s.cfg != nil {
		oidcEnabled = s.cfg.OIDC.Enabled
		oidcProviderName = strings.TrimSpace(s.cfg.OIDC.ProviderName)
	}
	if raw, ok := settings[settingKeyOIDCConnectEnabled]; ok {
		oidcEnabled = strings.TrimSpace(raw) == "true"
	}
	if value := strings.TrimSpace(settings[settingKeyOIDCConnectProviderName]); value != "" {
		oidcProviderName = value
	}
	availableChannelsEnabled := settings[settingKeyAvailableChannelsEnabled] == "true"
	if runtime, ok := s.getAvailableChannelsRuntimeOverride(); ok {
		availableChannelsEnabled = runtime.Enabled
	}
	soraClientEnabled := settings[SettingKeySoraClientEnabled] == "true"
	if enabled, ok := s.getSoraClientRuntimeOverride(); ok {
		soraClientEnabled = enabled
	}
	webSearchSettings := PublicWorkspaceWebSearchSettings{}
	if s != nil && s.cfg != nil {
		webSearchSettings.Provider = strings.TrimSpace(s.cfg.Workspace.WebSearch.Provider)
		webSearchSettings.Available = s.cfg.Workspace.WebSearch.Enabled && !s.cfg.Workspace.WebSearch.KillSwitch
	}

	return &PublicSettings{
		RegistrationEnabled:                  settings[SettingKeyRegistrationEnabled] == "true",
		EmailVerifyEnabled:                   emailVerifyEnabled,
		RegistrationEmailSuffixWhitelist:     registrationEmailSuffixWhitelist,
		PromoCodeEnabled:                     settings[SettingKeyPromoCodeEnabled] != "false", // 默认启用
		PasswordResetEnabled:                 passwordResetEnabled,
		InvitationCodeEnabled:                settings[SettingKeyInvitationCodeEnabled] == "true",
		TotpEnabled:                          settings[SettingKeyTotpEnabled] == "true",
		TurnstileEnabled:                     settings[SettingKeyTurnstileEnabled] == "true",
		TurnstileSiteKey:                     settings[SettingKeyTurnstileSiteKey],
		SiteName:                             s.getStringOrDefault(settings, SettingKeySiteName, "Sub2API"),
		SiteLogo:                             settings[SettingKeySiteLogo],
		SiteSubtitle:                         s.getStringOrDefault(settings, SettingKeySiteSubtitle, "Subscription to API Conversion Platform"),
		APIBaseURL:                           settings[SettingKeyAPIBaseURL],
		ContactInfo:                          settings[SettingKeyContactInfo],
		DocURL:                               settings[SettingKeyDocURL],
		HomeContent:                          sanitizePublicHomeContent(settings[SettingKeyHomeContent]),
		HideCcsImportButton:                  settings[SettingKeyHideCcsImportButton] == "true",
		PurchaseSubscriptionEnabled:          settings[SettingKeyPurchaseSubscriptionEnabled] == "true",
		PurchaseSubscriptionURL:              sanitizePublicEmbeddedURL(settings[SettingKeyPurchaseSubscriptionURL]),
		PurchaseLinkCNY10:                    sanitizePublicEmbeddedURL(settings[SettingKeyPurchaseLinkCNY10]),
		PurchaseLinkCNY30:                    sanitizePublicEmbeddedURL(settings[SettingKeyPurchaseLinkCNY30]),
		PurchaseLinkCNY100:                   sanitizePublicEmbeddedURL(settings[SettingKeyPurchaseLinkCNY100]),
		PaymentEnabled:                       settings[SettingPaymentEnabled] == "true",
		SoraClientEnabled:                    soraClientEnabled,
		CustomMenuItems:                      settings[SettingKeyCustomMenuItems],
		LinuxDoOAuthEnabled:                  linuxDoEnabled,
		WeChatOAuthEnabled:                   weChatOAuth.Enabled,
		WeChatOAuthOpenEnabled:               weChatOAuth.OpenEnabled,
		WeChatOAuthMPEnabled:                 weChatOAuth.MPEnabled,
		WeChatOAuthMobileEnabled:             weChatOAuth.MobileEnabled,
		OIDCOAuthEnabled:                     oidcEnabled,
		OIDCOAuthProviderName:                oidcProviderName,
		ChannelMonitorEnabled:                settings[settingKeyChannelMonitorEnabled] == "true",
		ChannelMonitorDefaultIntervalSeconds: 60,
		AvailableChannelsEnabled:             availableChannelsEnabled,
		WebSearch:                            webSearchSettings,
		AffiliateEnabled:                     settings[SettingKeyAffiliateEnabled] == "true",
		BackendModeEnabled:                   settings[SettingKeyBackendModeEnabled] == "true",
	}, nil
}

func sanitizePublicHomeContent(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if isAbsoluteHTTPURL(raw) {
		return raw
	}
	sanitized := homeContentScriptTagPattern.ReplaceAllString(raw, "")
	sanitized = homeContentEventAttrPattern.ReplaceAllString(sanitized, "")
	sanitized = homeContentJavascriptPattern.ReplaceAllString(sanitized, "blocked:")
	return strings.TrimSpace(sanitized)
}

func isAbsoluteHTTPURL(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || !u.IsAbs() || strings.TrimSpace(u.Host) == "" {
		return false
	}
	return strings.EqualFold(u.Scheme, "http") || strings.EqualFold(u.Scheme, "https")
}

func sanitizePublicEmbeddedURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if err := config.ValidateAbsoluteHTTPURL(raw); err != nil {
		return ""
	}
	return raw
}

// SetOnUpdateCallback sets a callback function to be called when settings are updated
// This is used for cache invalidation (e.g., HTML cache in frontend server)
func (s *SettingService) SetOnUpdateCallback(callback func()) {
	s.onUpdate = callback
}

// SetOnS3UpdateCallback 设置 Sora S3 配置变更时的回调函数（用于刷新 S3 客户端缓存）。
func (s *SettingService) SetOnS3UpdateCallback(callback func()) {
	s.onS3Update = callback
}

// SetOnTLSFingerprintUpdateCallback 设置 TLS 指纹配置变更时的回调函数。
func (s *SettingService) SetOnTLSFingerprintUpdateCallback(callback func()) {
	s.onTLSFingerprintUpdate = callback
}

// SetVersion sets the application version for injection into public settings
func (s *SettingService) SetVersion(version string) {
	s.version = version
}

// GetPublicSettingsForInjection returns public settings in a format suitable for HTML injection
// This implements the web.PublicSettingsProvider interface
func (s *SettingService) GetPublicSettingsForInjectionLegacy(ctx context.Context) (any, error) {
	settings, err := s.GetPublicSettings(ctx)
	if err != nil {
		return nil, err
	}

	// Return a struct that matches the frontend's expected format
	return &struct {
		RegistrationEnabled                  bool            `json:"registration_enabled"`
		EmailVerifyEnabled                   bool            `json:"email_verify_enabled"`
		RegistrationEmailSuffixWhitelist     []string        `json:"registration_email_suffix_whitelist"`
		PromoCodeEnabled                     bool            `json:"promo_code_enabled"`
		PasswordResetEnabled                 bool            `json:"password_reset_enabled"`
		InvitationCodeEnabled                bool            `json:"invitation_code_enabled"`
		TotpEnabled                          bool            `json:"totp_enabled"`
		TurnstileEnabled                     bool            `json:"turnstile_enabled"`
		TurnstileSiteKey                     string          `json:"turnstile_site_key,omitempty"`
		SiteName                             string          `json:"site_name"`
		SiteLogo                             string          `json:"site_logo,omitempty"`
		SiteSubtitle                         string          `json:"site_subtitle,omitempty"`
		APIBaseURL                           string          `json:"api_base_url,omitempty"`
		ContactInfo                          string          `json:"contact_info,omitempty"`
		DocURL                               string          `json:"doc_url,omitempty"`
		HomeContent                          string          `json:"home_content,omitempty"`
		HideCcsImportButton                  bool            `json:"hide_ccs_import_button"`
		PurchaseSubscriptionEnabled          bool            `json:"purchase_subscription_enabled"`
		PurchaseSubscriptionURL              string          `json:"purchase_subscription_url,omitempty"`
		PurchaseLinkCNY10                    string          `json:"purchase_link_cny_10,omitempty"`
		PurchaseLinkCNY30                    string          `json:"purchase_link_cny_30,omitempty"`
		PurchaseLinkCNY100                   string          `json:"purchase_link_cny_100,omitempty"`
		PaymentEnabled                       bool            `json:"payment_enabled"`
		SoraClientEnabled                    bool            `json:"sora_client_enabled"`
		CustomMenuItems                      json.RawMessage `json:"custom_menu_items"`
		LinuxDoOAuthEnabled                  bool            `json:"linuxdo_oauth_enabled"`
		WeChatOAuthEnabled                   bool            `json:"wechat_oauth_enabled"`
		WeChatOAuthOpenEnabled               bool            `json:"wechat_oauth_open_enabled"`
		WeChatOAuthMPEnabled                 bool            `json:"wechat_oauth_mp_enabled"`
		WeChatOAuthMobileEnabled             bool            `json:"wechat_oauth_mobile_enabled"`
		OIDCOAuthEnabled                     bool            `json:"oidc_oauth_enabled"`
		OIDCOAuthProviderName                string          `json:"oidc_oauth_provider_name,omitempty"`
		ChannelMonitorEnabled                bool            `json:"channel_monitor_enabled"`
		ChannelMonitorDefaultIntervalSeconds int             `json:"channel_monitor_default_interval_seconds"`
		AvailableChannelsEnabled             bool            `json:"available_channels_enabled"`
		AffiliateEnabled                     bool            `json:"affiliate_enabled"`
		BackendModeEnabled                   bool            `json:"backend_mode_enabled"`
		Version                              string          `json:"version,omitempty"`
	}{
		RegistrationEnabled:                  settings.RegistrationEnabled,
		EmailVerifyEnabled:                   settings.EmailVerifyEnabled,
		RegistrationEmailSuffixWhitelist:     settings.RegistrationEmailSuffixWhitelist,
		PromoCodeEnabled:                     settings.PromoCodeEnabled,
		PasswordResetEnabled:                 settings.PasswordResetEnabled,
		InvitationCodeEnabled:                settings.InvitationCodeEnabled,
		TotpEnabled:                          settings.TotpEnabled,
		TurnstileEnabled:                     settings.TurnstileEnabled,
		TurnstileSiteKey:                     settings.TurnstileSiteKey,
		SiteName:                             settings.SiteName,
		SiteLogo:                             settings.SiteLogo,
		SiteSubtitle:                         settings.SiteSubtitle,
		APIBaseURL:                           settings.APIBaseURL,
		ContactInfo:                          settings.ContactInfo,
		DocURL:                               settings.DocURL,
		HomeContent:                          settings.HomeContent,
		HideCcsImportButton:                  settings.HideCcsImportButton,
		PurchaseSubscriptionEnabled:          settings.PurchaseSubscriptionEnabled,
		PurchaseSubscriptionURL:              settings.PurchaseSubscriptionURL,
		PurchaseLinkCNY10:                    settings.PurchaseLinkCNY10,
		PurchaseLinkCNY30:                    settings.PurchaseLinkCNY30,
		PurchaseLinkCNY100:                   settings.PurchaseLinkCNY100,
		PaymentEnabled:                       settings.PaymentEnabled,
		SoraClientEnabled:                    settings.SoraClientEnabled,
		CustomMenuItems:                      filterUserVisibleMenuItems(settings.CustomMenuItems),
		LinuxDoOAuthEnabled:                  settings.LinuxDoOAuthEnabled,
		WeChatOAuthEnabled:                   settings.WeChatOAuthEnabled,
		WeChatOAuthOpenEnabled:               settings.WeChatOAuthOpenEnabled,
		WeChatOAuthMPEnabled:                 settings.WeChatOAuthMPEnabled,
		WeChatOAuthMobileEnabled:             settings.WeChatOAuthMobileEnabled,
		OIDCOAuthEnabled:                     settings.OIDCOAuthEnabled,
		OIDCOAuthProviderName:                settings.OIDCOAuthProviderName,
		ChannelMonitorEnabled:                settings.ChannelMonitorEnabled,
		ChannelMonitorDefaultIntervalSeconds: settings.ChannelMonitorDefaultIntervalSeconds,
		AvailableChannelsEnabled:             settings.AvailableChannelsEnabled,
		AffiliateEnabled:                     settings.AffiliateEnabled,
		BackendModeEnabled:                   settings.BackendModeEnabled,
		Version:                              s.version,
	}, nil
}

// filterUserVisibleMenuItems filters out admin-only menu items from a raw JSON
// array string, returning only items with visibility != "admin".
func filterUserVisibleMenuItemsLegacy(raw string) json.RawMessage {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return json.RawMessage("[]")
	}
	var items []struct {
		Visibility string `json:"visibility"`
		URL        string `json:"url"`
	}
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return json.RawMessage("[]")
	}

	// Parse full items to preserve all fields
	var fullItems []json.RawMessage
	if err := json.Unmarshal([]byte(raw), &fullItems); err != nil {
		return json.RawMessage("[]")
	}

	var filtered []json.RawMessage
	for i, item := range items {
		if item.Visibility != "admin" && extractOriginFromURL(item.URL) != "" {
			filtered = append(filtered, fullItems[i])
		}
	}
	if len(filtered) == 0 {
		return json.RawMessage("[]")
	}
	result, err := json.Marshal(filtered)
	if err != nil {
		return json.RawMessage("[]")
	}
	return result
}

// GetFrameSrcOrigins returns deduplicated http(s) origins from purchase_subscription_url
// and all custom_menu_items URLs. Used by the router layer for CSP frame-src injection.
func (s *SettingService) GetFrameSrcOriginsLegacy(ctx context.Context) ([]string, error) {
	settings, err := s.GetPublicSettings(ctx)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{})
	var origins []string

	addOrigin := func(rawURL string) {
		if origin := extractOriginFromURL(rawURL); origin != "" {
			if _, ok := seen[origin]; !ok {
				seen[origin] = struct{}{}
				origins = append(origins, origin)
			}
		}
	}

	// purchase subscription URL
	if settings.PurchaseSubscriptionEnabled {
		addOrigin(settings.PurchaseSubscriptionURL)
	}

	// all custom menu items (including admin-only, since CSP must allow all iframes)
	for _, item := range parseCustomMenuItemURLs(settings.CustomMenuItems) {
		addOrigin(item)
	}

	return origins, nil
}

// extractOriginFromURL returns the scheme+host origin from rawURL.
// Only http and https schemes are accepted.
func extractOriginFromURLLegacy(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return ""
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

// parseCustomMenuItemURLs extracts URLs from a raw JSON array of custom menu items.
func parseCustomMenuItemURLsLegacy(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return nil
	}
	var items []struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil
	}
	urls := make([]string, 0, len(items))
	for _, item := range items {
		if item.URL != "" {
			urls = append(urls, item.URL)
		}
	}
	return urls
}

// UpdateSettings 更新系统设置

// IsRegistrationEnabled 检查是否开放注册

// IsBackendModeEnabled checks if backend mode is enabled
// Uses in-process atomic.Value cache with 60s TTL, zero-lock hot path
func (s *SettingService) IsBackendModeEnabledLegacy(ctx context.Context) bool {
	if s == nil || s.settingRepo == nil {
		return false
	}
	if cached, ok := backendModeCache.Load().(*cachedBackendMode); ok && cached != nil {
		if time.Now().UnixNano() < cached.expiresAt {
			return cached.value
		}
	}
	result, _, _ := backendModeSF.Do("backend_mode", func() (any, error) {
		if cached, ok := backendModeCache.Load().(*cachedBackendMode); ok && cached != nil {
			if time.Now().UnixNano() < cached.expiresAt {
				return cached.value, nil
			}
		}
		dbCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), backendModeDBTimeout)
		defer cancel()
		value, err := s.settingRepo.GetValue(dbCtx, SettingKeyBackendModeEnabled)
		if err != nil {
			if errors.Is(err, ErrSettingNotFound) {
				// Setting not yet created (fresh install) - default to disabled with full TTL
				backendModeCache.Store(&cachedBackendMode{
					value:     false,
					expiresAt: time.Now().Add(backendModeCacheTTL).UnixNano(),
				})
				return false, nil
			}
			slog.Warn("failed to get backend_mode_enabled setting", "error", err)
			backendModeCache.Store(&cachedBackendMode{
				value:     false,
				expiresAt: time.Now().Add(backendModeErrorTTL).UnixNano(),
			})
			return false, nil
		}
		enabled := value == "true"
		backendModeCache.Store(&cachedBackendMode{
			value:     enabled,
			expiresAt: time.Now().Add(backendModeCacheTTL).UnixNano(),
		})
		return enabled, nil
	})
	if val, ok := result.(bool); ok {
		return val
	}
	return false
}

// IsEmailVerifyEnabled 检查是否开启邮件验证

// GetRegistrationEmailSuffixWhitelist returns normalized registration email suffix whitelist.

// IsPromoCodeEnabled 检查是否启用优惠码功能

// IsInvitationCodeEnabled 检查是否启用邀请码注册功能

// IsPasswordResetEnabled 检查是否启用密码重置功能
// 要求：必须同时开启邮件验证

// IsTotpEnabled 检查是否启用 TOTP 双因素认证功能

// IsTotpEncryptionKeyConfigured 检查 TOTP 加密密钥是否已手动配置
// 只有手动配置了密钥才允许在管理后台启用 TOTP 功能

// GetSiteName 获取网站名称

// GetDefaultConcurrency 获取默认并发量

// GetDefaultBalance 获取默认余额

// GetDefaultSubscriptions 获取新用户默认订阅配置列表。

// InitializeDefaultSettings 初始化默认设置
func (s *SettingService) InitializeDefaultSettingsLegacy(ctx context.Context) error {
	// 检查是否已有设置
	_, err := s.settingRepo.GetValue(ctx, SettingKeyRegistrationEnabled)
	if err == nil {
		// 已有设置，不需要初始化
		return nil
	}
	if !errors.Is(err, ErrSettingNotFound) {
		return fmt.Errorf("check existing settings: %w", err)
	}

	// 初始化默认设置
	defaults := map[string]string{
		SettingKeyRegistrationEnabled:              "true",
		SettingKeyEmailVerifyEnabled:               "false",
		SettingKeyRegistrationEmailSuffixWhitelist: "[]",
		SettingKeyPromoCodeEnabled:                 "true", // 默认启用优惠码功能
		SettingKeySiteName:                         "Sub2API",
		SettingKeySiteLogo:                         "",
		SettingKeyPurchaseSubscriptionEnabled:      "false",
		SettingKeyPurchaseSubscriptionURL:          "",
		SettingKeyPurchaseLinkCNY10:                "",
		SettingKeyPurchaseLinkCNY30:                "",
		SettingKeyPurchaseLinkCNY100:               "",
		SettingKeySoraClientEnabled:                "false",
		SettingKeyCustomMenuItems:                  "[]",
		SettingKeyDefaultConcurrency:               strconv.Itoa(s.cfg.Default.UserConcurrency),
		SettingKeyDefaultBalance:                   strconv.FormatFloat(s.cfg.Default.UserBalance, 'f', 8, 64),
		SettingKeyAffiliateRebateRate:              strconv.FormatFloat(AffiliateRebateRateDefault, 'f', 8, 64),
		SettingKeyAffiliateRebateFreezeHours:       strconv.Itoa(AffiliateRebateFreezeHoursDefault),
		SettingKeyAffiliateRebateDurationDays:      strconv.Itoa(AffiliateRebateDurationDaysDefault),
		SettingKeyAffiliateRebatePerInviteeCap:     strconv.FormatFloat(AffiliateRebatePerInviteeCapDefault, 'f', 2, 64),
		SettingKeyAffiliateEnabled:                 "false",
		SettingKeyDefaultSubscriptions:             "[]",
		SettingKeySMTPPort:                         "587",
		SettingKeySMTPUseTLS:                       "false",
		// Model fallback defaults
		SettingKeyEnableModelFallback:      "false",
		SettingKeyFallbackModelAnthropic:   "claude-3-5-sonnet-20241022",
		SettingKeyFallbackModelOpenAI:      "gpt-4o",
		SettingKeyFallbackModelGemini:      "gemini-2.5-pro",
		SettingKeyFallbackModelAntigravity: "gemini-2.5-pro",
		// Identity patch defaults
		SettingKeyEnableIdentityPatch: "true",
		SettingKeyIdentityPatchPrompt: "",

		// Ops monitoring defaults (vNext)
		SettingKeyOpsMonitoringEnabled:         "true",
		SettingKeyOpsRealtimeMonitoringEnabled: "true",
		SettingKeyOpsQueryModeDefault:          "auto",
		SettingKeyOpsMetricsIntervalSeconds:    "60",

		// Claude Code version check (default: empty = disabled)
		SettingKeyMinClaudeCodeVersion: "",
		SettingKeyMaxClaudeCodeVersion: "",

		// 分组隔离（默认不允许未分组 Key 调度）
		SettingKeyAllowUngroupedKeyScheduling: "false",
		SettingKeyAutoDelete401Accounts:       "false",
		SettingKeyAutoDelete429Accounts:       "false",
		SettingKeyAutoDeleteUselessProxies:    "false",
	}

	return s.setMultipleSettingValues(ctx, defaults)
}

// parseSettings 解析设置到结构体
func (s *SettingService) parseSettingsLegacy(settings map[string]string) *SystemSettings {
	emailVerifyEnabled := settings[SettingKeyEmailVerifyEnabled] == "true"
	result := &SystemSettings{
		RegistrationEnabled:              settings[SettingKeyRegistrationEnabled] == "true",
		EmailVerifyEnabled:               emailVerifyEnabled,
		RegistrationEmailSuffixWhitelist: ParseRegistrationEmailSuffixWhitelist(settings[SettingKeyRegistrationEmailSuffixWhitelist]),
		PromoCodeEnabled:                 settings[SettingKeyPromoCodeEnabled] != "false", // 默认启用
		PasswordResetEnabled:             emailVerifyEnabled && settings[SettingKeyPasswordResetEnabled] == "true",
		FrontendURL:                      settings[SettingKeyFrontendURL],
		InvitationCodeEnabled:            settings[SettingKeyInvitationCodeEnabled] == "true",
		TotpEnabled:                      settings[SettingKeyTotpEnabled] == "true",
		SMTPHost:                         settings[SettingKeySMTPHost],
		SMTPUsername:                     settings[SettingKeySMTPUsername],
		SMTPFrom:                         settings[SettingKeySMTPFrom],
		SMTPFromName:                     settings[SettingKeySMTPFromName],
		SMTPUseTLS:                       settings[SettingKeySMTPUseTLS] == "true",
		SMTPPasswordConfigured:           settings[SettingKeySMTPPassword] != "",
		TurnstileEnabled:                 settings[SettingKeyTurnstileEnabled] == "true",
		TurnstileSiteKey:                 settings[SettingKeyTurnstileSiteKey],
		TurnstileSecretKeyConfigured:     settings[SettingKeyTurnstileSecretKey] != "",
		SiteName:                         s.getStringOrDefault(settings, SettingKeySiteName, "Sub2API"),
		SiteLogo:                         settings[SettingKeySiteLogo],
		SiteSubtitle:                     s.getStringOrDefault(settings, SettingKeySiteSubtitle, "Subscription to API Conversion Platform"),
		APIBaseURL:                       settings[SettingKeyAPIBaseURL],
		ContactInfo:                      settings[SettingKeyContactInfo],
		DocURL:                           settings[SettingKeyDocURL],
		HomeContent:                      settings[SettingKeyHomeContent],
		HideCcsImportButton:              settings[SettingKeyHideCcsImportButton] == "true",
		PurchaseSubscriptionEnabled:      settings[SettingKeyPurchaseSubscriptionEnabled] == "true",
		PurchaseSubscriptionURL:          strings.TrimSpace(settings[SettingKeyPurchaseSubscriptionURL]),
		PurchaseLinkCNY10:                sanitizePublicEmbeddedURL(settings[SettingKeyPurchaseLinkCNY10]),
		PurchaseLinkCNY30:                sanitizePublicEmbeddedURL(settings[SettingKeyPurchaseLinkCNY30]),
		PurchaseLinkCNY100:               sanitizePublicEmbeddedURL(settings[SettingKeyPurchaseLinkCNY100]),
		SoraClientEnabled:                settings[SettingKeySoraClientEnabled] == "true",
		CustomMenuItems:                  settings[SettingKeyCustomMenuItems],
		BackendModeEnabled:               settings[SettingKeyBackendModeEnabled] == "true",
	}

	// 解析整数类型
	if port, err := strconv.Atoi(settings[SettingKeySMTPPort]); err == nil {
		result.SMTPPort = port
	} else {
		result.SMTPPort = 587
	}

	if concurrency, err := strconv.Atoi(settings[SettingKeyDefaultConcurrency]); err == nil {
		result.DefaultConcurrency = concurrency
	} else {
		result.DefaultConcurrency = s.cfg.Default.UserConcurrency
	}

	// 解析浮点数类型
	if balance, err := strconv.ParseFloat(settings[SettingKeyDefaultBalance], 64); err == nil {
		result.DefaultBalance = balance
	} else {
		result.DefaultBalance = s.cfg.Default.UserBalance
	}
	if rebateRate, err := strconv.ParseFloat(settings[SettingKeyAffiliateRebateRate], 64); err == nil {
		result.AffiliateRebateRate = clampAffiliateRebateRate(rebateRate)
	} else {
		result.AffiliateRebateRate = AffiliateRebateRateDefault
	}
	result.AffiliateRebateFreezeHours = AffiliateRebateFreezeHoursDefault
	if freezeHours, err := strconv.Atoi(settings[SettingKeyAffiliateRebateFreezeHours]); err == nil && freezeHours >= 0 {
		if freezeHours > AffiliateRebateFreezeHoursMax {
			freezeHours = AffiliateRebateFreezeHoursMax
		}
		result.AffiliateRebateFreezeHours = freezeHours
	}
	result.AffiliateRebateDurationDays = AffiliateRebateDurationDaysDefault
	if durationDays, err := strconv.Atoi(settings[SettingKeyAffiliateRebateDurationDays]); err == nil && durationDays >= 0 {
		if durationDays > AffiliateRebateDurationDaysMax {
			durationDays = AffiliateRebateDurationDaysMax
		}
		result.AffiliateRebateDurationDays = durationDays
	}
	result.AffiliateRebatePerInviteeCap = AffiliateRebatePerInviteeCapDefault
	if perInviteeCap, err := strconv.ParseFloat(settings[SettingKeyAffiliateRebatePerInviteeCap], 64); err == nil && perInviteeCap >= 0 {
		result.AffiliateRebatePerInviteeCap = perInviteeCap
	}
	result.DefaultSubscriptions = parseDefaultSubscriptions(settings[SettingKeyDefaultSubscriptions])

	// 敏感信息直接返回，方便测试连接时使用
	result.SMTPPassword = settings[SettingKeySMTPPassword]
	result.TurnstileSecretKey = settings[SettingKeyTurnstileSecretKey]

	// LinuxDo Connect 设置：
	// - 兼容 config.yaml/env（避免老部署因为未迁移到数据库设置而被意外关闭）
	// - 支持在后台“系统设置”中覆盖并持久化（存储于 DB）
	linuxDoBase := config.LinuxDoConnectConfig{}
	if s.cfg != nil {
		linuxDoBase = s.cfg.LinuxDo
	}

	if raw, ok := settings[SettingKeyLinuxDoConnectEnabled]; ok {
		result.LinuxDoConnectEnabled = raw == "true"
	} else {
		result.LinuxDoConnectEnabled = linuxDoBase.Enabled
	}

	if v, ok := settings[SettingKeyLinuxDoConnectClientID]; ok && strings.TrimSpace(v) != "" {
		result.LinuxDoConnectClientID = strings.TrimSpace(v)
	} else {
		result.LinuxDoConnectClientID = linuxDoBase.ClientID
	}

	if v, ok := settings[SettingKeyLinuxDoConnectRedirectURL]; ok && strings.TrimSpace(v) != "" {
		result.LinuxDoConnectRedirectURL = strings.TrimSpace(v)
	} else {
		result.LinuxDoConnectRedirectURL = linuxDoBase.RedirectURL
	}

	result.LinuxDoConnectClientSecret = strings.TrimSpace(settings[SettingKeyLinuxDoConnectClientSecret])
	if result.LinuxDoConnectClientSecret == "" {
		result.LinuxDoConnectClientSecret = strings.TrimSpace(linuxDoBase.ClientSecret)
	}
	result.LinuxDoConnectClientSecretConfigured = result.LinuxDoConnectClientSecret != ""

	// Model fallback settings
	result.EnableModelFallback = settings[SettingKeyEnableModelFallback] == "true"
	result.FallbackModelAnthropic = s.getStringOrDefault(settings, SettingKeyFallbackModelAnthropic, "claude-3-5-sonnet-20241022")
	result.FallbackModelOpenAI = s.getStringOrDefault(settings, SettingKeyFallbackModelOpenAI, "gpt-4o")
	result.FallbackModelGemini = s.getStringOrDefault(settings, SettingKeyFallbackModelGemini, "gemini-2.5-pro")
	result.FallbackModelAntigravity = s.getStringOrDefault(settings, SettingKeyFallbackModelAntigravity, "gemini-2.5-pro")

	// Identity patch settings (default: enabled, to preserve existing behavior)
	if v, ok := settings[SettingKeyEnableIdentityPatch]; ok && v != "" {
		result.EnableIdentityPatch = v == "true"
	} else {
		result.EnableIdentityPatch = true
	}
	result.IdentityPatchPrompt = settings[SettingKeyIdentityPatchPrompt]

	// Ops monitoring settings (default: enabled, fail-open)
	result.OpsMonitoringEnabled = !isFalseSettingValue(settings[SettingKeyOpsMonitoringEnabled])
	result.OpsRealtimeMonitoringEnabled = !isFalseSettingValue(settings[SettingKeyOpsRealtimeMonitoringEnabled])
	result.OpsQueryModeDefault = string(ParseOpsQueryMode(settings[SettingKeyOpsQueryModeDefault]))
	result.OpsMetricsIntervalSeconds = 60
	if raw := strings.TrimSpace(settings[SettingKeyOpsMetricsIntervalSeconds]); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil {
			if v < 60 {
				v = 60
			}
			if v > 3600 {
				v = 3600
			}
			result.OpsMetricsIntervalSeconds = v
		}
	}

	// Claude Code version check
	result.MinClaudeCodeVersion = settings[SettingKeyMinClaudeCodeVersion]
	result.MaxClaudeCodeVersion = settings[SettingKeyMaxClaudeCodeVersion]

	// 分组隔离
	result.AllowUngroupedKeyScheduling = settings[SettingKeyAllowUngroupedKeyScheduling] == "true"
	result.AutoDelete401Accounts = settings[SettingKeyAutoDelete401Accounts] == "true"
	result.AutoDelete429Accounts = settings[SettingKeyAutoDelete429Accounts] == "true"
	result.AutoDeleteUselessProxies = settings[SettingKeyAutoDeleteUselessProxies] == "true"
	result.AffiliateEnabled = settings[SettingKeyAffiliateEnabled] == "true"

	return result
}

func clampAffiliateRebateRateLegacy(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return AffiliateRebateRateDefault
	}
	if value < AffiliateRebateRateMin {
		return AffiliateRebateRateMin
	}
	if value > AffiliateRebateRateMax {
		return AffiliateRebateRateMax
	}
	return value
}

func isFalseSettingValueLegacy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "false", "0", "off", "disabled":
		return true
	default:
		return false
	}
}

func parseDefaultSubscriptionsLegacy(raw string) []DefaultSubscriptionSetting {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	var items []DefaultSubscriptionSetting
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil
	}

	normalized := make([]DefaultSubscriptionSetting, 0, len(items))
	for _, item := range items {
		if item.GroupID <= 0 || item.ValidityDays <= 0 {
			continue
		}
		if item.ValidityDays > MaxValidityDays {
			item.ValidityDays = MaxValidityDays
		}
		normalized = append(normalized, item)
	}

	return normalized
}

// getStringOrDefault 获取字符串值或默认值
func (s *SettingService) getStringOrDefault(settings map[string]string, key, defaultValue string) string {
	if value, ok := settings[key]; ok && value != "" {
		return value
	}
	return defaultValue
}

// IsTurnstileEnabled 检查是否启用 Turnstile 验证

// GetTurnstileSecretKey 获取 Turnstile Secret Key

// IsIdentityPatchEnabled 检查是否启用身份补丁（Claude -> Gemini systemInstruction 注入）

// GetIdentityPatchPrompt 获取自定义身份补丁提示词（为空表示使用内置默认模板）

func hashAdminAPIKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

func maskAdminAPIKey(key string) string {
	if len(key) > 14 {
		return key[:10] + "..." + key[len(key)-4:]
	}
	return key
}

func parseAdminAPIKeyRecord(raw string) (*adminAPIKeyRecord, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" || !strings.HasPrefix(raw, "{") {
		return nil, false
	}
	var record adminAPIKeyRecord
	if err := json.Unmarshal([]byte(raw), &record); err != nil {
		return nil, false
	}
	if record.Version != adminAPIKeyRecordVersion || record.KeyHash == "" || record.AdminUserID <= 0 {
		return nil, false
	}
	if record.MaskedKey == "" {
		record.MaskedKey = "configured"
	}
	return &record, true
}

// GenerateAdminAPIKey 生成新的管理员 API Key

// GetAdminAPIKeyStatus 获取管理员 API Key 状态
// 返回脱敏的 key、是否存在、错误

// ValidateAdminAPIKey 校验管理员 API Key 并返回其绑定主体。
// 旧版纯文本 Key 会被视为需要轮换，不再接受认证。
func (s *SettingService) ValidateAdminAPIKey(ctx context.Context, presentedKey string) (*AdminAPIKeyBinding, error) {
	presentedKey = strings.TrimSpace(presentedKey)
	if presentedKey == "" {
		return nil, nil
	}

	storedKey, err := s.getSettingValueCached(ctx, SettingKeyAdminAPIKey)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if stored := strings.TrimSpace(storedKey); stored != "" {
		if record, ok := parseAdminAPIKeyRecord(stored); ok {
			if subtle.ConstantTimeCompare([]byte(hashAdminAPIKey(presentedKey)), []byte(record.KeyHash)) != 1 {
				return nil, nil
			}
			return &AdminAPIKeyBinding{
				AdminUserID:       record.AdminUserID,
				AdminTokenVersion: record.AdminTokenVersion,
			}, nil
		}
		return nil, nil
	}
	return nil, nil
}

// DeleteAdminAPIKey 删除管理员 API Key

// IsModelFallbackEnabled 检查是否启用模型兜底机制

// GetFallbackModel 获取指定平台的兜底模型

// GetLinuxDoConnectOAuthConfig 返回用于登录的"最终生效" LinuxDo Connect 配置。
//
// 优先级：
// - 若对应系统设置键存在，则覆盖 config.yaml/env 的值
// - 否则回退到 config.yaml/env 的值
func (s *SettingService) GetLinuxDoConnectOAuthConfigLegacy(ctx context.Context) (config.LinuxDoConnectConfig, error) {
	if s == nil || s.cfg == nil {
		return config.LinuxDoConnectConfig{}, infraerrors.ServiceUnavailable("CONFIG_NOT_READY", "config not loaded")
	}

	effective := s.cfg.LinuxDo

	keys := []string{
		SettingKeyLinuxDoConnectEnabled,
		SettingKeyLinuxDoConnectClientID,
		SettingKeyLinuxDoConnectClientSecret,
		SettingKeyLinuxDoConnectRedirectURL,
	}
	settings, err := s.settingRepo.GetMultiple(ctx, keys)
	if err != nil {
		return config.LinuxDoConnectConfig{}, fmt.Errorf("get linuxdo connect settings: %w", err)
	}

	if raw, ok := settings[SettingKeyLinuxDoConnectEnabled]; ok {
		effective.Enabled = raw == "true"
	}
	if v, ok := settings[SettingKeyLinuxDoConnectClientID]; ok && strings.TrimSpace(v) != "" {
		effective.ClientID = strings.TrimSpace(v)
	}
	if v, ok := settings[SettingKeyLinuxDoConnectClientSecret]; ok && strings.TrimSpace(v) != "" {
		effective.ClientSecret = strings.TrimSpace(v)
	}
	if v, ok := settings[SettingKeyLinuxDoConnectRedirectURL]; ok && strings.TrimSpace(v) != "" {
		effective.RedirectURL = strings.TrimSpace(v)
	}

	if !effective.Enabled {
		return config.LinuxDoConnectConfig{}, infraerrors.NotFound("OAUTH_DISABLED", "oauth login is disabled")
	}

	// 基础健壮性校验（避免把用户重定向到一个必然失败或不安全的 OAuth 流程里）。
	if strings.TrimSpace(effective.ClientID) == "" {
		return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth client id not configured")
	}
	if strings.TrimSpace(effective.AuthorizeURL) == "" {
		return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth authorize url not configured")
	}
	if strings.TrimSpace(effective.TokenURL) == "" {
		return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth token url not configured")
	}
	if strings.TrimSpace(effective.UserInfoURL) == "" {
		return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth userinfo url not configured")
	}
	if strings.TrimSpace(effective.RedirectURL) == "" {
		return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth redirect url not configured")
	}
	if strings.TrimSpace(effective.FrontendRedirectURL) == "" {
		return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth frontend redirect url not configured")
	}

	if err := config.ValidateAbsoluteHTTPURL(effective.AuthorizeURL); err != nil {
		return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth authorize url invalid")
	}
	if err := config.ValidateAbsoluteHTTPURL(effective.TokenURL); err != nil {
		return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth token url invalid")
	}
	if err := config.ValidateAbsoluteHTTPURL(effective.UserInfoURL); err != nil {
		return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth userinfo url invalid")
	}
	if err := config.ValidateAbsoluteHTTPURL(effective.RedirectURL); err != nil {
		return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth redirect url invalid")
	}
	if err := config.ValidateFrontendRedirectURL(effective.FrontendRedirectURL); err != nil {
		return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth frontend redirect url invalid")
	}

	method := strings.ToLower(strings.TrimSpace(effective.TokenAuthMethod))
	switch method {
	case "", "client_secret_post", "client_secret_basic":
		if strings.TrimSpace(effective.ClientSecret) == "" {
			return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth client secret not configured")
		}
	case "none":
		if !effective.UsePKCE {
			return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth pkce must be enabled when token_auth_method=none")
		}
	default:
		return config.LinuxDoConnectConfig{}, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth token_auth_method invalid")
	}

	return effective, nil
}

// GetOverloadCooldownSettings 获取529过载冷却配置

// SetOverloadCooldownSettings 设置529过载冷却配置

// GetStreamTimeoutSettings 获取流超时处理配置

// IsUngroupedKeySchedulingAllowed 查询是否允许未分组 Key 调度
func (s *SettingService) IsUngroupedKeySchedulingAllowedLegacy(ctx context.Context) bool {
	value, err := s.getSettingValueCached(ctx, SettingKeyAllowUngroupedKeyScheduling)
	if err != nil {
		return false // fail-closed: 查询失败时默认不允许
	}
	return value == "true"
}

func (s *SettingService) IsAutoDelete401AccountsEnabled(ctx context.Context) bool {
	value, err := s.getSettingValueCached(ctx, SettingKeyAutoDelete401Accounts)
	if err != nil {
		return false
	}
	return value == "true"
}

func (s *SettingService) IsAutoDelete429AccountsEnabled(ctx context.Context) bool {
	value, err := s.getSettingValueCached(ctx, SettingKeyAutoDelete429Accounts)
	if err != nil {
		return false
	}
	return value == "true"
}

func (s *SettingService) IsAutoDeleteUselessProxiesEnabled(ctx context.Context) bool {
	value, err := s.getSettingValueCached(ctx, SettingKeyAutoDeleteUselessProxies)
	if err != nil {
		return false
	}
	return value == "true"
}

// GetClaudeCodeVersionBounds 获取 Claude Code 版本号上下限要求
// 使用进程内 atomic.Value 缓存，60 秒 TTL，热路径零锁开销
// singleflight 防止缓存过期时 thundering herd
// 返回空字符串表示不做对应方向的版本检查
func (s *SettingService) GetClaudeCodeVersionBoundsLegacy(ctx context.Context) (min, max string) {
	if cached, ok := versionBoundsCache.Load().(*cachedVersionBounds); ok {
		if time.Now().UnixNano() < cached.expiresAt {
			return cached.min, cached.max
		}
	}
	// singleflight: 同一时刻只有一个 goroutine 查询 DB，其余复用结果
	type bounds struct{ min, max string }
	result, err, _ := versionBoundsSF.Do("version_bounds", func() (any, error) {
		// 二次检查，避免排队的 goroutine 重复查询
		if cached, ok := versionBoundsCache.Load().(*cachedVersionBounds); ok {
			if time.Now().UnixNano() < cached.expiresAt {
				return bounds{cached.min, cached.max}, nil
			}
		}
		// 使用独立 context：断开请求取消链，避免客户端断连导致空值被长期缓存
		dbCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), versionBoundsDBTimeout)
		defer cancel()
		values, err := s.settingRepo.GetMultiple(dbCtx, []string{
			SettingKeyMinClaudeCodeVersion,
			SettingKeyMaxClaudeCodeVersion,
		})
		if err != nil {
			// fail-open: DB 错误时不阻塞请求，但记录日志并使用短 TTL 快速重试
			slog.Warn("failed to get claude code version bounds setting, skipping version check", "error", err)
			versionBoundsCache.Store(&cachedVersionBounds{
				min:       "",
				max:       "",
				expiresAt: time.Now().Add(versionBoundsErrorTTL).UnixNano(),
			})
			return bounds{"", ""}, nil
		}
		b := bounds{
			min: values[SettingKeyMinClaudeCodeVersion],
			max: values[SettingKeyMaxClaudeCodeVersion],
		}
		versionBoundsCache.Store(&cachedVersionBounds{
			min:       b.min,
			max:       b.max,
			expiresAt: time.Now().Add(versionBoundsCacheTTL).UnixNano(),
		})
		return b, nil
	})
	if err != nil {
		return "", ""
	}
	b, ok := result.(bounds)
	if !ok {
		return "", ""
	}
	return b.min, b.max
}

// GetRectifierSettings 获取请求整流器配置
func (s *SettingService) GetRectifierSettingsLegacy(ctx context.Context) (*RectifierSettings, error) {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyRectifierSettings)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return DefaultRectifierSettings(), nil
		}
		return nil, fmt.Errorf("get rectifier settings: %w", err)
	}
	if value == "" {
		return DefaultRectifierSettings(), nil
	}

	var settings RectifierSettings
	if err := json.Unmarshal([]byte(value), &settings); err != nil {
		return DefaultRectifierSettings(), nil
	}

	return &settings, nil
}

// SetRectifierSettings 设置请求整流器配置
func (s *SettingService) SetRectifierSettingsLegacy(ctx context.Context, settings *RectifierSettings) error {
	if settings == nil {
		return fmt.Errorf("settings cannot be nil")
	}

	data, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("marshal rectifier settings: %w", err)
	}

	return s.settingRepo.Set(ctx, SettingKeyRectifierSettings, string(data))
}

// IsSignatureRectifierEnabled 判断签名整流是否启用（总开关 && 签名子开关）

// IsBudgetRectifierEnabled 判断 Budget 整流是否启用（总开关 && Budget 子开关）

// GetBetaPolicySettings 获取 Beta 策略配置

// SetBetaPolicySettings 设置 Beta 策略配置

// SetStreamTimeoutSettings 设置流超时处理配置

type soraS3ProfilesStore struct {
	ActiveProfileID string                   `json:"active_profile_id"`
	Items           []soraS3ProfileStoreItem `json:"items"`
}

type soraS3ProfileStoreItem struct {
	ProfileID                string `json:"profile_id"`
	Name                     string `json:"name"`
	Enabled                  bool   `json:"enabled"`
	Endpoint                 string `json:"endpoint"`
	Region                   string `json:"region"`
	Bucket                   string `json:"bucket"`
	AccessKeyID              string `json:"access_key_id"`
	SecretAccessKey          string `json:"secret_access_key"`
	Prefix                   string `json:"prefix"`
	ForcePathStyle           bool   `json:"force_path_style"`
	CDNURL                   string `json:"cdn_url"`
	DefaultStorageQuotaBytes int64  `json:"default_storage_quota_bytes"`
	UpdatedAt                string `json:"updated_at"`
}

type tlsFingerprintProfilesStore struct {
	Enabled bool                             `json:"enabled"`
	Items   []tlsFingerprintProfileStoreItem `json:"items"`
}

type tlsFingerprintProfileStoreItem struct {
	ProfileID    string   `json:"profile_id"`
	Name         string   `json:"name"`
	Enabled      bool     `json:"enabled"`
	EnableGREASE bool     `json:"enable_grease"`
	CipherSuites []uint16 `json:"cipher_suites"`
	Curves       []uint16 `json:"curves"`
	PointFormats []uint8  `json:"point_formats"`
	UpdatedAt    string   `json:"updated_at"`
}

// GetTLSFingerprintSettings 获取 TLS 指纹全局开关。
func (s *SettingService) GetTLSFingerprintSettings(ctx context.Context) (*TLSFingerprintSettings, error) {
	store, err := s.loadTLSFingerprintProfilesStore(ctx)
	if err != nil {
		return nil, err
	}
	return &TLSFingerprintSettings{Enabled: store.Enabled}, nil
}

// SetTLSFingerprintSettings 更新 TLS 指纹全局开关。
func (s *SettingService) SetTLSFingerprintSettings(ctx context.Context, settings *TLSFingerprintSettings) error {
	if settings == nil {
		return fmt.Errorf("settings cannot be nil")
	}
	store, err := s.loadTLSFingerprintProfilesStore(ctx)
	if err != nil {
		return err
	}
	store.Enabled = settings.Enabled
	return s.persistTLSFingerprintProfilesStore(ctx, store)
}

// RefreshTLSFingerprintRuntime 初始化或刷新 TLS 指纹运行时 registry。
func (s *SettingService) RefreshTLSFingerprintRuntime(ctx context.Context) error {
	store, err := s.loadTLSFingerprintProfilesStore(ctx)
	if err != nil {
		return err
	}
	s.refreshTLSFingerprintRuntimeFromStore(store)
	return nil
}

// ListTLSFingerprintProfiles 获取 TLS 指纹 Profile 列表。
func (s *SettingService) ListTLSFingerprintProfiles(ctx context.Context) (*TLSFingerprintProfileList, error) {
	store, err := s.loadTLSFingerprintProfilesStore(ctx)
	if err != nil {
		return nil, err
	}
	return convertTLSFingerprintProfilesStore(store), nil
}

// CreateTLSFingerprintProfile 创建 TLS 指纹 Profile。
func (s *SettingService) CreateTLSFingerprintProfile(ctx context.Context, profile *TLSFingerprintProfile) (*TLSFingerprintProfile, error) {
	if profile == nil {
		return nil, fmt.Errorf("profile cannot be nil")
	}
	profileID := strings.TrimSpace(profile.ProfileID)
	if profileID == "" {
		return nil, infraerrors.BadRequest("TLS_FINGERPRINT_PROFILE_ID_REQUIRED", "profile_id is required")
	}
	name := strings.TrimSpace(profile.Name)
	if name == "" {
		return nil, infraerrors.BadRequest("TLS_FINGERPRINT_PROFILE_NAME_REQUIRED", "name is required")
	}

	store, err := s.loadTLSFingerprintProfilesStore(ctx)
	if err != nil {
		return nil, err
	}
	if hasTLSFingerprintProfileID(store.Items, profileID) {
		return nil, ErrTLSFingerprintProfileExists
	}

	now := time.Now().UTC().Format(time.RFC3339)
	store.Items = append(store.Items, tlsFingerprintProfileStoreItem{
		ProfileID:    profileID,
		Name:         name,
		Enabled:      profile.Enabled,
		EnableGREASE: profile.EnableGREASE,
		CipherSuites: normalizeUint16Slice(profile.CipherSuites),
		Curves:       normalizeUint16Slice(profile.Curves),
		PointFormats: normalizeUint8Slice(profile.PointFormats),
		UpdatedAt:    now,
	})

	if err := s.persistTLSFingerprintProfilesStore(ctx, store); err != nil {
		return nil, err
	}

	profiles := convertTLSFingerprintProfilesStore(store)
	created := findTLSFingerprintProfileByID(profiles.Items, profileID)
	if created == nil {
		return nil, ErrTLSFingerprintProfileNotFound
	}
	return created, nil
}

// UpdateTLSFingerprintProfile 更新 TLS 指纹 Profile。
func (s *SettingService) UpdateTLSFingerprintProfile(ctx context.Context, profileID string, profile *TLSFingerprintProfile) (*TLSFingerprintProfile, error) {
	if profile == nil {
		return nil, fmt.Errorf("profile cannot be nil")
	}
	targetID := strings.TrimSpace(profileID)
	if targetID == "" {
		return nil, infraerrors.BadRequest("TLS_FINGERPRINT_PROFILE_ID_REQUIRED", "profile_id is required")
	}
	name := strings.TrimSpace(profile.Name)
	if name == "" {
		return nil, infraerrors.BadRequest("TLS_FINGERPRINT_PROFILE_NAME_REQUIRED", "name is required")
	}

	store, err := s.loadTLSFingerprintProfilesStore(ctx)
	if err != nil {
		return nil, err
	}
	targetIndex := findTLSFingerprintProfileIndex(store.Items, targetID)
	if targetIndex < 0 {
		return nil, ErrTLSFingerprintProfileNotFound
	}

	store.Items[targetIndex] = tlsFingerprintProfileStoreItem{
		ProfileID:    targetID,
		Name:         name,
		Enabled:      profile.Enabled,
		EnableGREASE: profile.EnableGREASE,
		CipherSuites: normalizeUint16Slice(profile.CipherSuites),
		Curves:       normalizeUint16Slice(profile.Curves),
		PointFormats: normalizeUint8Slice(profile.PointFormats),
		UpdatedAt:    time.Now().UTC().Format(time.RFC3339),
	}

	if err := s.persistTLSFingerprintProfilesStore(ctx, store); err != nil {
		return nil, err
	}

	profiles := convertTLSFingerprintProfilesStore(store)
	updated := findTLSFingerprintProfileByID(profiles.Items, targetID)
	if updated == nil {
		return nil, ErrTLSFingerprintProfileNotFound
	}
	return updated, nil
}

// DeleteTLSFingerprintProfile 删除 TLS 指纹 Profile。
func (s *SettingService) DeleteTLSFingerprintProfile(ctx context.Context, profileID string) error {
	targetID := strings.TrimSpace(profileID)
	if targetID == "" {
		return infraerrors.BadRequest("TLS_FINGERPRINT_PROFILE_ID_REQUIRED", "profile_id is required")
	}

	store, err := s.loadTLSFingerprintProfilesStore(ctx)
	if err != nil {
		return err
	}
	if len(store.Items) <= 1 {
		return infraerrors.BadRequest("TLS_FINGERPRINT_PROFILE_LAST_DELETE_FORBIDDEN", "cannot delete the last tls fingerprint profile")
	}

	targetIndex := findTLSFingerprintProfileIndex(store.Items, targetID)
	if targetIndex < 0 {
		return ErrTLSFingerprintProfileNotFound
	}

	store.Items = append(store.Items[:targetIndex], store.Items[targetIndex+1:]...)
	return s.persistTLSFingerprintProfilesStore(ctx, store)
}

func (s *SettingService) loadTLSFingerprintProfilesStore(ctx context.Context) (*tlsFingerprintProfilesStore, error) {
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyTLSFingerprintProfiles)
	if err == nil {
		trimmed := strings.TrimSpace(raw)
		if trimmed != "" {
			var store tlsFingerprintProfilesStore
			if unmarshalErr := json.Unmarshal([]byte(trimmed), &store); unmarshalErr == nil {
				normalized := normalizeTLSFingerprintProfilesStore(store)
				return &normalized, nil
			}
		}
		cfgStore := s.tlsFingerprintProfilesStoreFromConfig()
		if cfgStore != nil {
			normalized := normalizeTLSFingerprintProfilesStore(*cfgStore)
			if persistErr := s.persistTLSFingerprintProfilesStore(ctx, &normalized); persistErr != nil {
				return nil, persistErr
			}
			return &normalized, nil
		}
		empty := normalizeTLSFingerprintProfilesStore(tlsFingerprintProfilesStore{})
		return &empty, nil
	}
	if !errors.Is(err, ErrSettingNotFound) {
		return nil, fmt.Errorf("get tls fingerprint profiles: %w", err)
	}

	cfgStore := s.tlsFingerprintProfilesStoreFromConfig()
	if cfgStore != nil {
		normalized := normalizeTLSFingerprintProfilesStore(*cfgStore)
		if persistErr := s.persistTLSFingerprintProfilesStore(ctx, &normalized); persistErr != nil {
			return nil, persistErr
		}
		return &normalized, nil
	}

	empty := normalizeTLSFingerprintProfilesStore(tlsFingerprintProfilesStore{})
	return &empty, nil
}

func (s *SettingService) tlsFingerprintProfilesStoreFromConfig() *tlsFingerprintProfilesStore {
	if s == nil || s.cfg == nil {
		return nil
	}
	cfg := s.cfg.Gateway.TLSFingerprint
	items := make([]tlsFingerprintProfileStoreItem, 0, len(cfg.Profiles))
	for profileID, profile := range cfg.Profiles {
		items = append(items, tlsFingerprintProfileStoreItem{
			ProfileID:    strings.TrimSpace(profileID),
			Name:         strings.TrimSpace(profile.Name),
			Enabled:      true,
			EnableGREASE: profile.EnableGREASE,
			CipherSuites: normalizeUint16Slice(profile.CipherSuites),
			Curves:       normalizeUint16Slice(profile.Curves),
			PointFormats: normalizeUint16ToUint8Slice(profile.PointFormats),
		})
	}
	if len(items) == 0 && !cfg.Enabled {
		return nil
	}
	return &tlsFingerprintProfilesStore{
		Enabled: cfg.Enabled,
		Items:   items,
	}
}

func (s *SettingService) persistTLSFingerprintProfilesStore(ctx context.Context, store *tlsFingerprintProfilesStore) error {
	if store == nil {
		return fmt.Errorf("tls fingerprint profiles store cannot be nil")
	}
	normalized := normalizeTLSFingerprintProfilesStore(*store)
	data, err := json.Marshal(normalized)
	if err != nil {
		return fmt.Errorf("marshal tls fingerprint profiles: %w", err)
	}
	if err := s.setSettingValue(ctx, SettingKeyTLSFingerprintProfiles, string(data)); err != nil {
		return err
	}
	s.refreshTLSFingerprintRuntimeFromStore(&normalized)
	return nil
}

func (s *SettingService) refreshTLSFingerprintRuntimeFromStore(store *tlsFingerprintProfilesStore) {
	profiles := make(map[string]*tlsfingerprint.Profile)
	if store != nil && store.Enabled {
		for _, item := range store.Items {
			if !item.Enabled || strings.TrimSpace(item.ProfileID) == "" {
				continue
			}
			profiles[item.ProfileID] = &tlsfingerprint.Profile{
				Name:         item.Name,
				EnableGREASE: item.EnableGREASE,
				CipherSuites: append([]uint16(nil), item.CipherSuites...),
				Curves:       append([]uint16(nil), item.Curves...),
				PointFormats: uint8ToUint16Slice(item.PointFormats),
			}
		}
	}
	tlsfingerprint.ReplaceGlobalRegistryProfiles(profiles)
	if s != nil && s.onTLSFingerprintUpdate != nil {
		s.onTLSFingerprintUpdate()
	}
}

func normalizeTLSFingerprintProfilesStore(store tlsFingerprintProfilesStore) tlsFingerprintProfilesStore {
	normalized := tlsFingerprintProfilesStore{
		Enabled: store.Enabled,
		Items:   make([]tlsFingerprintProfileStoreItem, 0, len(store.Items)),
	}
	seen := make(map[string]struct{}, len(store.Items))
	now := time.Now().UTC().Format(time.RFC3339)
	for idx := range store.Items {
		item := store.Items[idx]
		item.ProfileID = strings.TrimSpace(item.ProfileID)
		if item.ProfileID == "" {
			item.ProfileID = fmt.Sprintf("profile-%d", idx+1)
		}
		if _, exists := seen[item.ProfileID]; exists {
			continue
		}
		seen[item.ProfileID] = struct{}{}
		item.Name = strings.TrimSpace(item.Name)
		if item.Name == "" {
			item.Name = item.ProfileID
		}
		item.CipherSuites = normalizeUint16Slice(item.CipherSuites)
		item.Curves = normalizeUint16Slice(item.Curves)
		item.PointFormats = normalizeUint8Slice(item.PointFormats)
		item.UpdatedAt = strings.TrimSpace(item.UpdatedAt)
		if item.UpdatedAt == "" {
			item.UpdatedAt = now
		}
		normalized.Items = append(normalized.Items, item)
	}
	sort.SliceStable(normalized.Items, func(i, j int) bool {
		return normalized.Items[i].ProfileID < normalized.Items[j].ProfileID
	})
	return normalized
}

func convertTLSFingerprintProfilesStore(store *tlsFingerprintProfilesStore) *TLSFingerprintProfileList {
	if store == nil {
		return &TLSFingerprintProfileList{}
	}
	items := make([]TLSFingerprintProfile, 0, len(store.Items))
	for _, item := range store.Items {
		items = append(items, TLSFingerprintProfile{
			ProfileID:    item.ProfileID,
			Name:         item.Name,
			Enabled:      item.Enabled,
			EnableGREASE: item.EnableGREASE,
			CipherSuites: append([]uint16(nil), item.CipherSuites...),
			Curves:       append([]uint16(nil), item.Curves...),
			PointFormats: append([]uint8(nil), item.PointFormats...),
			UpdatedAt:    item.UpdatedAt,
		})
	}
	return &TLSFingerprintProfileList{
		Enabled: store.Enabled,
		Items:   items,
	}
}

func findTLSFingerprintProfileIndex(items []tlsFingerprintProfileStoreItem, profileID string) int {
	for idx := range items {
		if items[idx].ProfileID == profileID {
			return idx
		}
	}
	return -1
}

func hasTLSFingerprintProfileID(items []tlsFingerprintProfileStoreItem, profileID string) bool {
	return findTLSFingerprintProfileIndex(items, profileID) >= 0
}

func findTLSFingerprintProfileByID(items []TLSFingerprintProfile, profileID string) *TLSFingerprintProfile {
	for idx := range items {
		if items[idx].ProfileID == profileID {
			return &items[idx]
		}
	}
	return nil
}

func normalizeUint16Slice(values []uint16) []uint16 {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[uint16]struct{}, len(values))
	normalized := make([]uint16, 0, len(values))
	for _, value := range values {
		if value == 0 {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func normalizeUint8Slice(values []uint8) []uint8 {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[uint8]struct{}, len(values))
	normalized := make([]uint8, 0, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func normalizeUint16ToUint8Slice(values []uint16) []uint8 {
	if len(values) == 0 {
		return nil
	}
	converted := make([]uint8, 0, len(values))
	for _, value := range values {
		if value <= 255 {
			converted = append(converted, uint8(value))
		}
	}
	return normalizeUint8Slice(converted)
}

func uint8ToUint16Slice(values []uint8) []uint16 {
	if len(values) == 0 {
		return nil
	}
	converted := make([]uint16, len(values))
	for i, value := range values {
		converted[i] = uint16(value)
	}
	return converted
}

// GetSoraS3Settings 获取 Sora S3 存储配置（兼容旧单配置语义：返回当前激活配置）
func (s *SettingService) GetSoraS3Settings(ctx context.Context) (*SoraS3Settings, error) {
	profiles, err := s.ListSoraS3Profiles(ctx)
	if err != nil {
		return nil, err
	}

	activeProfile := pickActiveSoraS3Profile(profiles.Items, profiles.ActiveProfileID)
	if activeProfile == nil {
		return &SoraS3Settings{}, nil
	}

	return &SoraS3Settings{
		Enabled:                   activeProfile.Enabled,
		Endpoint:                  activeProfile.Endpoint,
		Region:                    activeProfile.Region,
		Bucket:                    activeProfile.Bucket,
		AccessKeyID:               activeProfile.AccessKeyID,
		SecretAccessKey:           activeProfile.SecretAccessKey,
		SecretAccessKeyConfigured: activeProfile.SecretAccessKeyConfigured,
		Prefix:                    activeProfile.Prefix,
		ForcePathStyle:            activeProfile.ForcePathStyle,
		CDNURL:                    activeProfile.CDNURL,
		DefaultStorageQuotaBytes:  activeProfile.DefaultStorageQuotaBytes,
	}, nil
}

// SetSoraS3Settings 更新 Sora S3 存储配置（兼容旧单配置语义：写入当前激活配置）
func (s *SettingService) SetSoraS3Settings(ctx context.Context, settings *SoraS3Settings) error {
	if settings == nil {
		return fmt.Errorf("settings cannot be nil")
	}

	store, err := s.loadSoraS3ProfilesStore(ctx)
	if err != nil {
		return err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	activeIndex := findSoraS3ProfileIndex(store.Items, store.ActiveProfileID)
	if activeIndex < 0 {
		activeID := "default"
		if hasSoraS3ProfileID(store.Items, activeID) {
			activeID = fmt.Sprintf("default-%d", time.Now().Unix())
		}
		store.Items = append(store.Items, soraS3ProfileStoreItem{
			ProfileID: activeID,
			Name:      "Default",
			UpdatedAt: now,
		})
		store.ActiveProfileID = activeID
		activeIndex = len(store.Items) - 1
	}

	active := store.Items[activeIndex]
	active.Enabled = settings.Enabled
	active.Endpoint = strings.TrimSpace(settings.Endpoint)
	active.Region = strings.TrimSpace(settings.Region)
	active.Bucket = strings.TrimSpace(settings.Bucket)
	active.AccessKeyID = strings.TrimSpace(settings.AccessKeyID)
	active.Prefix = strings.TrimSpace(settings.Prefix)
	active.ForcePathStyle = settings.ForcePathStyle
	active.CDNURL = strings.TrimSpace(settings.CDNURL)
	active.DefaultStorageQuotaBytes = maxInt64(settings.DefaultStorageQuotaBytes, 0)
	if settings.SecretAccessKey != "" {
		active.SecretAccessKey = settings.SecretAccessKey
	}
	active.UpdatedAt = now
	store.Items[activeIndex] = active

	return s.persistSoraS3ProfilesStore(ctx, store)
}

// ListSoraS3Profiles 获取 Sora S3 多配置列表
func (s *SettingService) ListSoraS3Profiles(ctx context.Context) (*SoraS3ProfileList, error) {
	store, err := s.loadSoraS3ProfilesStore(ctx)
	if err != nil {
		return nil, err
	}
	return convertSoraS3ProfilesStore(store), nil
}

// CreateSoraS3Profile 创建 Sora S3 配置
func (s *SettingService) CreateSoraS3Profile(ctx context.Context, profile *SoraS3Profile, setActive bool) (*SoraS3Profile, error) {
	if profile == nil {
		return nil, fmt.Errorf("profile cannot be nil")
	}

	profileID := strings.TrimSpace(profile.ProfileID)
	if profileID == "" {
		return nil, infraerrors.BadRequest("SORA_S3_PROFILE_ID_REQUIRED", "profile_id is required")
	}
	name := strings.TrimSpace(profile.Name)
	if name == "" {
		return nil, infraerrors.BadRequest("SORA_S3_PROFILE_NAME_REQUIRED", "name is required")
	}

	store, err := s.loadSoraS3ProfilesStore(ctx)
	if err != nil {
		return nil, err
	}
	if hasSoraS3ProfileID(store.Items, profileID) {
		return nil, ErrSoraS3ProfileExists
	}

	now := time.Now().UTC().Format(time.RFC3339)
	store.Items = append(store.Items, soraS3ProfileStoreItem{
		ProfileID:                profileID,
		Name:                     name,
		Enabled:                  profile.Enabled,
		Endpoint:                 strings.TrimSpace(profile.Endpoint),
		Region:                   strings.TrimSpace(profile.Region),
		Bucket:                   strings.TrimSpace(profile.Bucket),
		AccessKeyID:              strings.TrimSpace(profile.AccessKeyID),
		SecretAccessKey:          profile.SecretAccessKey,
		Prefix:                   strings.TrimSpace(profile.Prefix),
		ForcePathStyle:           profile.ForcePathStyle,
		CDNURL:                   strings.TrimSpace(profile.CDNURL),
		DefaultStorageQuotaBytes: maxInt64(profile.DefaultStorageQuotaBytes, 0),
		UpdatedAt:                now,
	})

	if setActive || store.ActiveProfileID == "" {
		store.ActiveProfileID = profileID
	}

	if err := s.persistSoraS3ProfilesStore(ctx, store); err != nil {
		return nil, err
	}

	profiles := convertSoraS3ProfilesStore(store)
	created := findSoraS3ProfileByID(profiles.Items, profileID)
	if created == nil {
		return nil, ErrSoraS3ProfileNotFound
	}
	return created, nil
}

// UpdateSoraS3Profile 更新 Sora S3 配置
func (s *SettingService) UpdateSoraS3Profile(ctx context.Context, profileID string, profile *SoraS3Profile) (*SoraS3Profile, error) {
	if profile == nil {
		return nil, fmt.Errorf("profile cannot be nil")
	}

	targetID := strings.TrimSpace(profileID)
	if targetID == "" {
		return nil, infraerrors.BadRequest("SORA_S3_PROFILE_ID_REQUIRED", "profile_id is required")
	}

	store, err := s.loadSoraS3ProfilesStore(ctx)
	if err != nil {
		return nil, err
	}

	targetIndex := findSoraS3ProfileIndex(store.Items, targetID)
	if targetIndex < 0 {
		return nil, ErrSoraS3ProfileNotFound
	}

	target := store.Items[targetIndex]
	name := strings.TrimSpace(profile.Name)
	if name == "" {
		return nil, infraerrors.BadRequest("SORA_S3_PROFILE_NAME_REQUIRED", "name is required")
	}
	target.Name = name
	target.Enabled = profile.Enabled
	target.Endpoint = strings.TrimSpace(profile.Endpoint)
	target.Region = strings.TrimSpace(profile.Region)
	target.Bucket = strings.TrimSpace(profile.Bucket)
	target.AccessKeyID = strings.TrimSpace(profile.AccessKeyID)
	target.Prefix = strings.TrimSpace(profile.Prefix)
	target.ForcePathStyle = profile.ForcePathStyle
	target.CDNURL = strings.TrimSpace(profile.CDNURL)
	target.DefaultStorageQuotaBytes = maxInt64(profile.DefaultStorageQuotaBytes, 0)
	if profile.SecretAccessKey != "" {
		target.SecretAccessKey = profile.SecretAccessKey
	}
	target.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	store.Items[targetIndex] = target

	if err := s.persistSoraS3ProfilesStore(ctx, store); err != nil {
		return nil, err
	}

	profiles := convertSoraS3ProfilesStore(store)
	updated := findSoraS3ProfileByID(profiles.Items, targetID)
	if updated == nil {
		return nil, ErrSoraS3ProfileNotFound
	}
	return updated, nil
}

// DeleteSoraS3Profile 删除 Sora S3 配置
func (s *SettingService) DeleteSoraS3Profile(ctx context.Context, profileID string) error {
	targetID := strings.TrimSpace(profileID)
	if targetID == "" {
		return infraerrors.BadRequest("SORA_S3_PROFILE_ID_REQUIRED", "profile_id is required")
	}

	store, err := s.loadSoraS3ProfilesStore(ctx)
	if err != nil {
		return err
	}

	targetIndex := findSoraS3ProfileIndex(store.Items, targetID)
	if targetIndex < 0 {
		return ErrSoraS3ProfileNotFound
	}

	store.Items = append(store.Items[:targetIndex], store.Items[targetIndex+1:]...)
	if store.ActiveProfileID == targetID {
		store.ActiveProfileID = ""
		if len(store.Items) > 0 {
			store.ActiveProfileID = store.Items[0].ProfileID
		}
	}

	return s.persistSoraS3ProfilesStore(ctx, store)
}

// SetActiveSoraS3Profile 设置激活的 Sora S3 配置
func (s *SettingService) SetActiveSoraS3Profile(ctx context.Context, profileID string) (*SoraS3Profile, error) {
	targetID := strings.TrimSpace(profileID)
	if targetID == "" {
		return nil, infraerrors.BadRequest("SORA_S3_PROFILE_ID_REQUIRED", "profile_id is required")
	}

	store, err := s.loadSoraS3ProfilesStore(ctx)
	if err != nil {
		return nil, err
	}

	targetIndex := findSoraS3ProfileIndex(store.Items, targetID)
	if targetIndex < 0 {
		return nil, ErrSoraS3ProfileNotFound
	}

	store.ActiveProfileID = targetID
	store.Items[targetIndex].UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := s.persistSoraS3ProfilesStore(ctx, store); err != nil {
		return nil, err
	}

	profiles := convertSoraS3ProfilesStore(store)
	active := pickActiveSoraS3Profile(profiles.Items, profiles.ActiveProfileID)
	if active == nil {
		return nil, ErrSoraS3ProfileNotFound
	}
	return active, nil
}

func (s *SettingService) loadSoraS3ProfilesStore(ctx context.Context) (*soraS3ProfilesStore, error) {
	raw, err := s.settingRepo.GetValue(ctx, SettingKeySoraS3Profiles)
	if err == nil {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			return &soraS3ProfilesStore{}, nil
		}
		var store soraS3ProfilesStore
		if unmarshalErr := json.Unmarshal([]byte(trimmed), &store); unmarshalErr != nil {
			legacy, legacyErr := s.getLegacySoraS3Settings(ctx)
			if legacyErr != nil {
				return nil, fmt.Errorf("unmarshal sora s3 profiles: %w", unmarshalErr)
			}
			if isEmptyLegacySoraS3Settings(legacy) {
				return &soraS3ProfilesStore{}, nil
			}
			now := time.Now().UTC().Format(time.RFC3339)
			return &soraS3ProfilesStore{
				ActiveProfileID: "default",
				Items: []soraS3ProfileStoreItem{
					{
						ProfileID:                "default",
						Name:                     "Default",
						Enabled:                  legacy.Enabled,
						Endpoint:                 strings.TrimSpace(legacy.Endpoint),
						Region:                   strings.TrimSpace(legacy.Region),
						Bucket:                   strings.TrimSpace(legacy.Bucket),
						AccessKeyID:              strings.TrimSpace(legacy.AccessKeyID),
						SecretAccessKey:          legacy.SecretAccessKey,
						Prefix:                   strings.TrimSpace(legacy.Prefix),
						ForcePathStyle:           legacy.ForcePathStyle,
						CDNURL:                   strings.TrimSpace(legacy.CDNURL),
						DefaultStorageQuotaBytes: maxInt64(legacy.DefaultStorageQuotaBytes, 0),
						UpdatedAt:                now,
					},
				},
			}, nil
		}
		normalized := normalizeSoraS3ProfilesStore(store)
		return &normalized, nil
	}

	if !errors.Is(err, ErrSettingNotFound) {
		return nil, fmt.Errorf("get sora s3 profiles: %w", err)
	}

	legacy, legacyErr := s.getLegacySoraS3Settings(ctx)
	if legacyErr != nil {
		return nil, legacyErr
	}
	if isEmptyLegacySoraS3Settings(legacy) {
		return &soraS3ProfilesStore{}, nil
	}

	now := time.Now().UTC().Format(time.RFC3339)
	return &soraS3ProfilesStore{
		ActiveProfileID: "default",
		Items: []soraS3ProfileStoreItem{
			{
				ProfileID:                "default",
				Name:                     "Default",
				Enabled:                  legacy.Enabled,
				Endpoint:                 strings.TrimSpace(legacy.Endpoint),
				Region:                   strings.TrimSpace(legacy.Region),
				Bucket:                   strings.TrimSpace(legacy.Bucket),
				AccessKeyID:              strings.TrimSpace(legacy.AccessKeyID),
				SecretAccessKey:          legacy.SecretAccessKey,
				Prefix:                   strings.TrimSpace(legacy.Prefix),
				ForcePathStyle:           legacy.ForcePathStyle,
				CDNURL:                   strings.TrimSpace(legacy.CDNURL),
				DefaultStorageQuotaBytes: maxInt64(legacy.DefaultStorageQuotaBytes, 0),
				UpdatedAt:                now,
			},
		},
	}, nil
}

func (s *SettingService) persistSoraS3ProfilesStore(ctx context.Context, store *soraS3ProfilesStore) error {
	if store == nil {
		return fmt.Errorf("sora s3 profiles store cannot be nil")
	}

	normalized := normalizeSoraS3ProfilesStore(*store)
	data, err := json.Marshal(normalized)
	if err != nil {
		return fmt.Errorf("marshal sora s3 profiles: %w", err)
	}

	updates := map[string]string{
		SettingKeySoraS3Profiles: string(data),
	}

	active := pickActiveSoraS3ProfileFromStore(normalized.Items, normalized.ActiveProfileID)
	if active == nil {
		updates[SettingKeySoraS3Enabled] = "false"
		updates[SettingKeySoraS3Endpoint] = ""
		updates[SettingKeySoraS3Region] = ""
		updates[SettingKeySoraS3Bucket] = ""
		updates[SettingKeySoraS3AccessKeyID] = ""
		updates[SettingKeySoraS3Prefix] = ""
		updates[SettingKeySoraS3ForcePathStyle] = "false"
		updates[SettingKeySoraS3CDNURL] = ""
		updates[SettingKeySoraDefaultStorageQuotaBytes] = "0"
		updates[SettingKeySoraS3SecretAccessKey] = ""
	} else {
		updates[SettingKeySoraS3Enabled] = strconv.FormatBool(active.Enabled)
		updates[SettingKeySoraS3Endpoint] = strings.TrimSpace(active.Endpoint)
		updates[SettingKeySoraS3Region] = strings.TrimSpace(active.Region)
		updates[SettingKeySoraS3Bucket] = strings.TrimSpace(active.Bucket)
		updates[SettingKeySoraS3AccessKeyID] = strings.TrimSpace(active.AccessKeyID)
		updates[SettingKeySoraS3Prefix] = strings.TrimSpace(active.Prefix)
		updates[SettingKeySoraS3ForcePathStyle] = strconv.FormatBool(active.ForcePathStyle)
		updates[SettingKeySoraS3CDNURL] = strings.TrimSpace(active.CDNURL)
		updates[SettingKeySoraDefaultStorageQuotaBytes] = strconv.FormatInt(maxInt64(active.DefaultStorageQuotaBytes, 0), 10)
		updates[SettingKeySoraS3SecretAccessKey] = active.SecretAccessKey
	}

	if err := s.setMultipleSettingValues(ctx, updates); err != nil {
		return err
	}

	if s.onS3Update != nil {
		s.onS3Update()
	}
	return nil
}

func (s *SettingService) getLegacySoraS3Settings(ctx context.Context) (*SoraS3Settings, error) {
	keys := []string{
		SettingKeySoraS3Enabled,
		SettingKeySoraS3Endpoint,
		SettingKeySoraS3Region,
		SettingKeySoraS3Bucket,
		SettingKeySoraS3AccessKeyID,
		SettingKeySoraS3SecretAccessKey,
		SettingKeySoraS3Prefix,
		SettingKeySoraS3ForcePathStyle,
		SettingKeySoraS3CDNURL,
		SettingKeySoraDefaultStorageQuotaBytes,
	}

	values, err := s.settingRepo.GetMultiple(ctx, keys)
	if err != nil {
		return nil, fmt.Errorf("get legacy sora s3 settings: %w", err)
	}

	result := &SoraS3Settings{
		Enabled:                   values[SettingKeySoraS3Enabled] == "true",
		Endpoint:                  values[SettingKeySoraS3Endpoint],
		Region:                    values[SettingKeySoraS3Region],
		Bucket:                    values[SettingKeySoraS3Bucket],
		AccessKeyID:               values[SettingKeySoraS3AccessKeyID],
		SecretAccessKey:           values[SettingKeySoraS3SecretAccessKey],
		SecretAccessKeyConfigured: values[SettingKeySoraS3SecretAccessKey] != "",
		Prefix:                    values[SettingKeySoraS3Prefix],
		ForcePathStyle:            values[SettingKeySoraS3ForcePathStyle] == "true",
		CDNURL:                    values[SettingKeySoraS3CDNURL],
	}
	if v, parseErr := strconv.ParseInt(values[SettingKeySoraDefaultStorageQuotaBytes], 10, 64); parseErr == nil {
		result.DefaultStorageQuotaBytes = v
	}
	return result, nil
}

func normalizeSoraS3ProfilesStore(store soraS3ProfilesStore) soraS3ProfilesStore {
	seen := make(map[string]struct{}, len(store.Items))
	normalized := soraS3ProfilesStore{
		ActiveProfileID: strings.TrimSpace(store.ActiveProfileID),
		Items:           make([]soraS3ProfileStoreItem, 0, len(store.Items)),
	}
	now := time.Now().UTC().Format(time.RFC3339)

	for idx := range store.Items {
		item := store.Items[idx]
		item.ProfileID = strings.TrimSpace(item.ProfileID)
		if item.ProfileID == "" {
			item.ProfileID = fmt.Sprintf("profile-%d", idx+1)
		}
		if _, exists := seen[item.ProfileID]; exists {
			continue
		}
		seen[item.ProfileID] = struct{}{}

		item.Name = strings.TrimSpace(item.Name)
		if item.Name == "" {
			item.Name = item.ProfileID
		}
		item.Endpoint = strings.TrimSpace(item.Endpoint)
		item.Region = strings.TrimSpace(item.Region)
		item.Bucket = strings.TrimSpace(item.Bucket)
		item.AccessKeyID = strings.TrimSpace(item.AccessKeyID)
		item.Prefix = strings.TrimSpace(item.Prefix)
		item.CDNURL = strings.TrimSpace(item.CDNURL)
		item.DefaultStorageQuotaBytes = maxInt64(item.DefaultStorageQuotaBytes, 0)
		item.UpdatedAt = strings.TrimSpace(item.UpdatedAt)
		if item.UpdatedAt == "" {
			item.UpdatedAt = now
		}
		normalized.Items = append(normalized.Items, item)
	}

	if len(normalized.Items) == 0 {
		normalized.ActiveProfileID = ""
		return normalized
	}

	if findSoraS3ProfileIndex(normalized.Items, normalized.ActiveProfileID) >= 0 {
		return normalized
	}

	normalized.ActiveProfileID = normalized.Items[0].ProfileID
	return normalized
}

func convertSoraS3ProfilesStore(store *soraS3ProfilesStore) *SoraS3ProfileList {
	if store == nil {
		return &SoraS3ProfileList{}
	}
	items := make([]SoraS3Profile, 0, len(store.Items))
	for idx := range store.Items {
		item := store.Items[idx]
		items = append(items, SoraS3Profile{
			ProfileID:                 item.ProfileID,
			Name:                      item.Name,
			IsActive:                  item.ProfileID == store.ActiveProfileID,
			Enabled:                   item.Enabled,
			Endpoint:                  item.Endpoint,
			Region:                    item.Region,
			Bucket:                    item.Bucket,
			AccessKeyID:               item.AccessKeyID,
			SecretAccessKey:           item.SecretAccessKey,
			SecretAccessKeyConfigured: item.SecretAccessKey != "",
			Prefix:                    item.Prefix,
			ForcePathStyle:            item.ForcePathStyle,
			CDNURL:                    item.CDNURL,
			DefaultStorageQuotaBytes:  item.DefaultStorageQuotaBytes,
			UpdatedAt:                 item.UpdatedAt,
		})
	}
	return &SoraS3ProfileList{
		ActiveProfileID: store.ActiveProfileID,
		Items:           items,
	}
}

func pickActiveSoraS3Profile(items []SoraS3Profile, activeProfileID string) *SoraS3Profile {
	for idx := range items {
		if items[idx].ProfileID == activeProfileID {
			return &items[idx]
		}
	}
	if len(items) == 0 {
		return nil
	}
	return &items[0]
}

func findSoraS3ProfileByID(items []SoraS3Profile, profileID string) *SoraS3Profile {
	for idx := range items {
		if items[idx].ProfileID == profileID {
			return &items[idx]
		}
	}
	return nil
}

func pickActiveSoraS3ProfileFromStore(items []soraS3ProfileStoreItem, activeProfileID string) *soraS3ProfileStoreItem {
	for idx := range items {
		if items[idx].ProfileID == activeProfileID {
			return &items[idx]
		}
	}
	if len(items) == 0 {
		return nil
	}
	return &items[0]
}

func findSoraS3ProfileIndex(items []soraS3ProfileStoreItem, profileID string) int {
	for idx := range items {
		if items[idx].ProfileID == profileID {
			return idx
		}
	}
	return -1
}

func hasSoraS3ProfileID(items []soraS3ProfileStoreItem, profileID string) bool {
	return findSoraS3ProfileIndex(items, profileID) >= 0
}

func isEmptyLegacySoraS3Settings(settings *SoraS3Settings) bool {
	if settings == nil {
		return true
	}
	if settings.Enabled {
		return false
	}
	if strings.TrimSpace(settings.Endpoint) != "" {
		return false
	}
	if strings.TrimSpace(settings.Region) != "" {
		return false
	}
	if strings.TrimSpace(settings.Bucket) != "" {
		return false
	}
	if strings.TrimSpace(settings.AccessKeyID) != "" {
		return false
	}
	if settings.SecretAccessKey != "" {
		return false
	}
	if strings.TrimSpace(settings.Prefix) != "" {
		return false
	}
	if strings.TrimSpace(settings.CDNURL) != "" {
		return false
	}
	return settings.DefaultStorageQuotaBytes == 0
}

func maxInt64(value int64, min int64) int64 {
	if value < min {
		return min
	}
	return value
}
