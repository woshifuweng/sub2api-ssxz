package admin

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

var semverPattern = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

// menuItemIDPattern validates custom menu item IDs: alphanumeric, hyphens, underscores only.
var menuItemIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// generateMenuItemID generates a short random hex ID for a custom menu item.
func generateMenuItemID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate menu item ID: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func scopesContainOpenID(scopes string) bool {
	for _, scope := range strings.Fields(strings.ToLower(strings.TrimSpace(scopes))) {
		if scope == "openid" {
			return true
		}
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// SettingHandler handles system settings.
type SettingHandler struct {
	settingService           *service.SettingService
	emailService             *service.EmailService
	turnstileService         *service.TurnstileService
	opsService               *service.OpsService
	paymentConfigService     *service.PaymentConfigService
	paymentService           *service.PaymentService
	userAttributeService     *service.UserAttributeService
	notificationEmailService *service.NotificationEmailService
	totpService              *service.TotpService
	userService              *service.UserService
	soraS3Storage            *service.SoraS3Storage
}

// NewSettingHandler creates a system settings handler.
func NewSettingHandler(settingService *service.SettingService, emailService *service.EmailService, turnstileService *service.TurnstileService, opsService *service.OpsService, paymentConfigService *service.PaymentConfigService, paymentService *service.PaymentService, userAttributeService *service.UserAttributeService) *SettingHandler {
	return &SettingHandler{
		settingService:       settingService,
		emailService:         emailService,
		turnstileService:     turnstileService,
		opsService:           opsService,
		paymentConfigService: paymentConfigService,
		paymentService:       paymentService,
		userAttributeService: userAttributeService,
	}
}

// SetNotificationEmailService attaches the notification template service without changing
// the constructor signature used by existing unit tests.
func (h *SettingHandler) SetNotificationEmailService(notificationEmailService *service.NotificationEmailService) {
	h.notificationEmailService = notificationEmailService
}

// SetStepUpDeps attaches the services backing the step-up switch preconditions
// (enable requires the acting admin to have TOTP enabled; disable is itself a
// step-up gated operation), without changing the constructor signature used by
// existing unit tests.
func (h *SettingHandler) SetStepUpDeps(totpService *service.TotpService, userService *service.UserService) {
	h.totpService = totpService
	h.userService = userService
}

// SetSoraS3Storage attaches the optional Sora object storage integration.
func (h *SettingHandler) SetSoraS3Storage(soraS3Storage *service.SoraS3Storage) {
	h.soraS3Storage = soraS3Storage
}

func (h *SettingHandler) GetSettings(c *gin.Context) {
	h.GetSettingsGateway(gatewayctx.FromGin(c))
}

func (h *SettingHandler) GetSettingsGateway(c gatewayctx.GatewayContext) {
	settingsDTO, err := h.loadSystemSettingsDTO(c.Request().Context(), c.HeaderValue("Origin"))
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	authSourceDefaults, err := h.settingService.GetAuthSourceDefaultSettings(c.Request().Context())
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, systemSettingsResponseData(settingsDTO, authSourceDefaults))
}

func (h *SettingHandler) loadSystemSettingsDTO(ctx context.Context, requestOrigin string) (dto.SystemSettings, error) {
	settings, err := h.settingService.GetAllSettings(ctx)
	if err != nil {
		return dto.SystemSettings{}, err
	}
	effectiveOpsMonitoringEnabled := settings.OpsMonitoringEnabled && h.opsService != nil && h.opsService.IsMonitoringEnabled(ctx)
	return h.buildSystemSettingsDTO(ctx, settings, effectiveOpsMonitoringEnabled, requestOrigin), nil
}

func (h *SettingHandler) buildSystemSettingsDTO(ctx context.Context, settings *service.SystemSettings, effectiveOpsMonitoringEnabled bool, requestOrigin string) dto.SystemSettings {
	defaultSubscriptions := make([]dto.DefaultSubscriptionSetting, 0, len(settings.DefaultSubscriptions))
	for _, sub := range settings.DefaultSubscriptions {
		defaultSubscriptions = append(defaultSubscriptions, dto.DefaultSubscriptionSetting{
			GroupID:      sub.GroupID,
			ValidityDays: sub.ValidityDays,
		})
	}

	// Load payment config
	var paymentCfg *service.PaymentConfig
	if h.paymentConfigService != nil {
		paymentCfg, _ = h.paymentConfigService.GetPaymentConfig(ctx)
	}
	if paymentCfg == nil {
		paymentCfg = &service.PaymentConfig{}
	}
	passkeyConfigured, passkeyRPID, passkeyRPOrigins := h.settingService.PasskeyConfiguration()

	payload := dto.SystemSettings{
		RegistrationEnabled:                                    settings.RegistrationEnabled,
		EmailVerifyEnabled:                                     settings.EmailVerifyEnabled,
		RegistrationEmailSuffixWhitelist:                       settings.RegistrationEmailSuffixWhitelist,
		PromoCodeEnabled:                                       settings.PromoCodeEnabled,
		PasswordResetEnabled:                                   settings.PasswordResetEnabled,
		PasswordResetEnabledStored:                             settings.PasswordResetEnabledStored,
		PasswordResetLinkBase:                                  h.settingService.ResolvePasswordResetLinkBase(ctx, requestOrigin),
		FrontendURL:                                            settings.FrontendURL,
		InvitationCodeEnabled:                                  settings.InvitationCodeEnabled,
		TotpEnabled:                                            settings.TotpEnabled,
		TotpEncryptionKeyConfigured:                            h.settingService.IsTotpEncryptionKeyConfigured(),
		PasskeyEnabled:                                         settings.PasskeyEnabled,
		PasskeyConfigured:                                      passkeyConfigured,
		PasskeyRPID:                                            passkeyRPID,
		PasskeyRPOrigins:                                       passkeyRPOrigins,
		SessionBindingEnabled:                                  settings.SessionBindingEnabled,
		StepUpEnabled:                                          settings.StepUpEnabled,
		AuditLogRetentionDays:                                  settings.AuditLogRetentionDays,
		LoginAgreementEnabled:                                  settings.LoginAgreementEnabled,
		LoginAgreementMode:                                     settings.LoginAgreementMode,
		LoginAgreementUpdatedAt:                                settings.LoginAgreementUpdatedAt,
		LoginAgreementDocuments:                                loginAgreementDocumentsToDTO(settings.LoginAgreementDocuments),
		SMTPHost:                                               settings.SMTPHost,
		SMTPPort:                                               settings.SMTPPort,
		SMTPUsername:                                           settings.SMTPUsername,
		SMTPPasswordConfigured:                                 settings.SMTPPasswordConfigured,
		SMTPFrom:                                               settings.SMTPFrom,
		SMTPFromName:                                           settings.SMTPFromName,
		SMTPUseTLS:                                             settings.SMTPUseTLS,
		TurnstileEnabled:                                       settings.TurnstileEnabled,
		TurnstileSiteKey:                                       settings.TurnstileSiteKey,
		TurnstileSecretKeyConfigured:                           settings.TurnstileSecretKeyConfigured,
		APIKeyACLTrustForwardedIP:                              settings.APIKeyACLTrustForwardedIP,
		ForwardedClientIPHeaders:                               settings.ForwardedClientIPHeaders,
		LinuxDoConnectEnabled:                                  settings.LinuxDoConnectEnabled,
		LinuxDoConnectClientID:                                 settings.LinuxDoConnectClientID,
		LinuxDoConnectClientSecretConfigured:                   settings.LinuxDoConnectClientSecretConfigured,
		LinuxDoConnectRedirectURL:                              settings.LinuxDoConnectRedirectURL,
		DingTalkConnectEnabled:                                 settings.DingTalkConnectEnabled,
		DingTalkConnectClientID:                                settings.DingTalkConnectClientID,
		DingTalkConnectClientSecretConfigured:                  settings.DingTalkConnectClientSecretConfigured,
		DingTalkConnectRedirectURL:                             settings.DingTalkConnectRedirectURL,
		DingTalkConnectCorpRestrictionPolicy:                   settings.DingTalkConnectCorpRestrictionPolicy,
		DingTalkConnectInternalCorpID:                          settings.DingTalkConnectInternalCorpID,
		DingTalkConnectBypassRegistration:                      settings.DingTalkConnectBypassRegistration,
		DingTalkConnectSyncCorpEmail:                           settings.DingTalkConnectSyncCorpEmail,
		DingTalkConnectSyncDisplayName:                         settings.DingTalkConnectSyncDisplayName,
		DingTalkConnectSyncDept:                                settings.DingTalkConnectSyncDept,
		DingTalkConnectSyncCorpEmailAttrKey:                    settings.DingTalkConnectSyncCorpEmailAttrKey,
		DingTalkConnectSyncDisplayNameAttrKey:                  settings.DingTalkConnectSyncDisplayNameAttrKey,
		DingTalkConnectSyncDeptAttrKey:                         settings.DingTalkConnectSyncDeptAttrKey,
		DingTalkConnectSyncCorpEmailAttrName:                   settings.DingTalkConnectSyncCorpEmailAttrName,
		DingTalkConnectSyncDisplayNameAttrName:                 settings.DingTalkConnectSyncDisplayNameAttrName,
		DingTalkConnectSyncDeptAttrName:                        settings.DingTalkConnectSyncDeptAttrName,
		WeChatConnectEnabled:                                   settings.WeChatConnectEnabled,
		WeChatConnectAppID:                                     settings.WeChatConnectAppID,
		WeChatConnectAppSecretConfigured:                       settings.WeChatConnectAppSecretConfigured,
		WeChatConnectOpenAppID:                                 settings.WeChatConnectOpenAppID,
		WeChatConnectOpenAppSecretConfigured:                   settings.WeChatConnectOpenAppSecretConfigured,
		WeChatConnectMPAppID:                                   settings.WeChatConnectMPAppID,
		WeChatConnectMPAppSecretConfigured:                     settings.WeChatConnectMPAppSecretConfigured,
		WeChatConnectMobileAppID:                               settings.WeChatConnectMobileAppID,
		WeChatConnectMobileAppSecretConfigured:                 settings.WeChatConnectMobileAppSecretConfigured,
		WeChatConnectOpenEnabled:                               settings.WeChatConnectOpenEnabled,
		WeChatConnectMPEnabled:                                 settings.WeChatConnectMPEnabled,
		WeChatConnectMobileEnabled:                             settings.WeChatConnectMobileEnabled,
		WeChatConnectMode:                                      settings.WeChatConnectMode,
		WeChatConnectScopes:                                    settings.WeChatConnectScopes,
		WeChatConnectRedirectURL:                               settings.WeChatConnectRedirectURL,
		WeChatConnectFrontendRedirectURL:                       settings.WeChatConnectFrontendRedirectURL,
		OIDCConnectEnabled:                                     settings.OIDCConnectEnabled,
		OIDCConnectProviderName:                                settings.OIDCConnectProviderName,
		OIDCConnectClientID:                                    settings.OIDCConnectClientID,
		OIDCConnectClientSecretConfigured:                      settings.OIDCConnectClientSecretConfigured,
		OIDCConnectIssuerURL:                                   settings.OIDCConnectIssuerURL,
		OIDCConnectDiscoveryURL:                                settings.OIDCConnectDiscoveryURL,
		OIDCConnectAuthorizeURL:                                settings.OIDCConnectAuthorizeURL,
		OIDCConnectTokenURL:                                    settings.OIDCConnectTokenURL,
		OIDCConnectUserInfoURL:                                 settings.OIDCConnectUserInfoURL,
		OIDCConnectJWKSURL:                                     settings.OIDCConnectJWKSURL,
		OIDCConnectScopes:                                      settings.OIDCConnectScopes,
		OIDCConnectRedirectURL:                                 settings.OIDCConnectRedirectURL,
		OIDCConnectFrontendRedirectURL:                         settings.OIDCConnectFrontendRedirectURL,
		OIDCConnectTokenAuthMethod:                             settings.OIDCConnectTokenAuthMethod,
		OIDCConnectUsePKCE:                                     settings.OIDCConnectUsePKCE,
		OIDCConnectValidateIDToken:                             settings.OIDCConnectValidateIDToken,
		OIDCConnectAllowedSigningAlgs:                          settings.OIDCConnectAllowedSigningAlgs,
		OIDCConnectClockSkewSeconds:                            settings.OIDCConnectClockSkewSeconds,
		OIDCConnectRequireEmailVerified:                        settings.OIDCConnectRequireEmailVerified,
		OIDCConnectUserInfoEmailPath:                           settings.OIDCConnectUserInfoEmailPath,
		OIDCConnectUserInfoIDPath:                              settings.OIDCConnectUserInfoIDPath,
		OIDCConnectUserInfoUsernamePath:                        settings.OIDCConnectUserInfoUsernamePath,
		GitHubOAuthEnabled:                                     settings.GitHubOAuthEnabled,
		GitHubOAuthClientID:                                    settings.GitHubOAuthClientID,
		GitHubOAuthClientSecretConfigured:                      settings.GitHubOAuthClientSecretConfigured,
		GitHubOAuthRedirectURL:                                 settings.GitHubOAuthRedirectURL,
		GitHubOAuthFrontendRedirectURL:                         settings.GitHubOAuthFrontendRedirectURL,
		GoogleOAuthEnabled:                                     settings.GoogleOAuthEnabled,
		GoogleOAuthClientID:                                    settings.GoogleOAuthClientID,
		GoogleOAuthClientSecretConfigured:                      settings.GoogleOAuthClientSecretConfigured,
		GoogleOAuthRedirectURL:                                 settings.GoogleOAuthRedirectURL,
		GoogleOAuthFrontendRedirectURL:                         settings.GoogleOAuthFrontendRedirectURL,
		SiteName:                                               settings.SiteName,
		SiteLogo:                                               settings.SiteLogo,
		SiteSubtitle:                                           settings.SiteSubtitle,
		APIBaseURL:                                             settings.APIBaseURL,
		ContactInfo:                                            settings.ContactInfo,
		DocURL:                                                 settings.DocURL,
		HomeContent:                                            settings.HomeContent,
		CompactHomeEnabled:                                     settings.CompactHomeEnabled,
		HideCcsImportButton:                                    settings.HideCcsImportButton,
		PurchaseSubscriptionEnabled:                            settings.PurchaseSubscriptionEnabled,
		PurchaseSubscriptionURL:                                settings.PurchaseSubscriptionURL,
		TableDefaultPageSize:                                   settings.TableDefaultPageSize,
		TablePageSizeOptions:                                   settings.TablePageSizeOptions,
		CustomMenuItems:                                        dto.ParseCustomMenuItems(settings.CustomMenuItems),
		CustomEndpoints:                                        dto.ParseCustomEndpoints(settings.CustomEndpoints),
		DefaultConcurrency:                                     settings.DefaultConcurrency,
		DefaultBalance:                                         settings.DefaultBalance,
		RiskControlEnabled:                                     settings.RiskControlEnabled,
		CyberSessionBlockEnabled:                               settings.CyberSessionBlockEnabled,
		CyberSessionBlockTTLSeconds:                            settings.CyberSessionBlockTTLSeconds,
		AffiliateRebateRate:                                    settings.AffiliateRebateRate,
		AffiliateRebateFreezeHours:                             settings.AffiliateRebateFreezeHours,
		AffiliateRebateDurationDays:                            settings.AffiliateRebateDurationDays,
		AffiliateRebatePerInviteeCap:                           settings.AffiliateRebatePerInviteeCap,
		AdminRechargeRebateEnabled:                             settings.AdminRechargeRebateEnabled,
		DefaultUserRPMLimit:                                    settings.DefaultUserRPMLimit,
		DefaultSubscriptions:                                   defaultSubscriptions,
		EnableModelFallback:                                    settings.EnableModelFallback,
		FallbackModelAnthropic:                                 settings.FallbackModelAnthropic,
		FallbackModelOpenAI:                                    settings.FallbackModelOpenAI,
		FallbackModelGemini:                                    settings.FallbackModelGemini,
		FallbackModelAntigravity:                               settings.FallbackModelAntigravity,
		EnableIdentityPatch:                                    settings.EnableIdentityPatch,
		IdentityPatchPrompt:                                    settings.IdentityPatchPrompt,
		OpsMonitoringEnabled:                                   effectiveOpsMonitoringEnabled,
		OpsRealtimeMonitoringEnabled:                           settings.OpsRealtimeMonitoringEnabled,
		OpsQueryModeDefault:                                    settings.OpsQueryModeDefault,
		OpsMetricsIntervalSeconds:                              settings.OpsMetricsIntervalSeconds,
		MinClaudeCodeVersion:                                   settings.MinClaudeCodeVersion,
		MaxClaudeCodeVersion:                                   settings.MaxClaudeCodeVersion,
		AllowUngroupedKeyScheduling:                            settings.AllowUngroupedKeyScheduling,
		BackendModeEnabled:                                     settings.BackendModeEnabled,
		EnableFingerprintUnification:                           settings.EnableFingerprintUnification,
		EnableMetadataPassthrough:                              settings.EnableMetadataPassthrough,
		EnableCCHSigning:                                       settings.EnableCCHSigning,
		EnableClaudeOAuthSystemPromptInjection:                 settings.EnableClaudeOAuthSystemPromptInjection,
		ClaudeOAuthSystemPrompt:                                settings.ClaudeOAuthSystemPrompt,
		ClaudeOAuthSystemPromptBlocks:                          settings.ClaudeOAuthSystemPromptBlocks,
		EnableAnthropicCacheTTL1hInjection:                     settings.EnableAnthropicCacheTTL1hInjection,
		RewriteMessageCacheControl:                             settings.RewriteMessageCacheControl,
		EnableClientDatelineNormalization:                      settings.EnableClientDatelineNormalization,
		AntigravityUserAgentVersion:                            settings.AntigravityUserAgentVersion,
		OpenAICodexUserAgent:                                   settings.OpenAICodexUserAgent,
		MinCodexVersion:                                        settings.MinCodexVersion,
		MaxCodexVersion:                                        settings.MaxCodexVersion,
		CodexCLIOnlyBlacklist:                                  settings.CodexCLIOnlyBlacklist,
		CodexCLIOnlyWhitelist:                                  settings.CodexCLIOnlyWhitelist,
		CodexCLIOnlyAllowAppServerClients:                      settings.CodexCLIOnlyAllowAppServerClients,
		CodexCLIOnlyEngineFingerprintSignals:                   settings.CodexCLIOnlyEngineFingerprintSignals,
		WebSearchEmulationEnabled:                              settings.WebSearchEmulationEnabled,
		PaymentVisibleMethodAlipaySource:                       settings.PaymentVisibleMethodAlipaySource,
		PaymentVisibleMethodWxpaySource:                        settings.PaymentVisibleMethodWxpaySource,
		PaymentVisibleMethodAlipayEnabled:                      settings.PaymentVisibleMethodAlipayEnabled,
		PaymentVisibleMethodWxpayEnabled:                       settings.PaymentVisibleMethodWxpayEnabled,
		OpenAILowUpstreamRatePriorityEnabled:                   settings.OpenAILowUpstreamRatePriorityEnabled,
		OpenAIOAuthSchedulingRateMultiplier:                    settings.OpenAIOAuthSchedulingRateMultiplier,
		OpenAIAdvancedSchedulerEnabled:                         settings.OpenAIAdvancedSchedulerEnabled,
		OpenAIAdvancedSchedulerStickyWeightedEnabled:           settings.OpenAIAdvancedSchedulerStickyWeightedEnabled,
		OpenAIAdvancedSchedulerSubscriptionPriorityEnabled:     settings.OpenAIAdvancedSchedulerSubscriptionPriorityEnabled,
		OpenAIAdvancedSchedulerLBTopK:                          settings.OpenAIAdvancedSchedulerLBTopK,
		OpenAIAdvancedSchedulerWeightPriority:                  settings.OpenAIAdvancedSchedulerWeightPriority,
		OpenAIAdvancedSchedulerWeightLoad:                      settings.OpenAIAdvancedSchedulerWeightLoad,
		OpenAIAdvancedSchedulerWeightQueue:                     settings.OpenAIAdvancedSchedulerWeightQueue,
		OpenAIAdvancedSchedulerWeightErrorRate:                 settings.OpenAIAdvancedSchedulerWeightErrorRate,
		OpenAIAdvancedSchedulerWeightTTFT:                      settings.OpenAIAdvancedSchedulerWeightTTFT,
		OpenAIAdvancedSchedulerWeightReset:                     settings.OpenAIAdvancedSchedulerWeightReset,
		OpenAIAdvancedSchedulerWeightQuotaHeadroom:             settings.OpenAIAdvancedSchedulerWeightQuotaHeadroom,
		OpenAIAdvancedSchedulerWeightUpstreamCost:              settings.OpenAIAdvancedSchedulerWeightUpstreamCost,
		OpenAIAdvancedSchedulerWeightPreviousResponse:          settings.OpenAIAdvancedSchedulerWeightPreviousResponse,
		OpenAIAdvancedSchedulerWeightSessionSticky:             settings.OpenAIAdvancedSchedulerWeightSessionSticky,
		OpenAIAdvancedSchedulerEffectiveLBTopK:                 settings.OpenAIAdvancedSchedulerEffectiveLBTopK,
		OpenAIAdvancedSchedulerEffectiveWeightPriority:         settings.OpenAIAdvancedSchedulerEffectiveWeightPriority,
		OpenAIAdvancedSchedulerEffectiveWeightLoad:             settings.OpenAIAdvancedSchedulerEffectiveWeightLoad,
		OpenAIAdvancedSchedulerEffectiveWeightQueue:            settings.OpenAIAdvancedSchedulerEffectiveWeightQueue,
		OpenAIAdvancedSchedulerEffectiveWeightErrorRate:        settings.OpenAIAdvancedSchedulerEffectiveWeightErrorRate,
		OpenAIAdvancedSchedulerEffectiveWeightTTFT:             settings.OpenAIAdvancedSchedulerEffectiveWeightTTFT,
		OpenAIAdvancedSchedulerEffectiveWeightReset:            settings.OpenAIAdvancedSchedulerEffectiveWeightReset,
		OpenAIAdvancedSchedulerEffectiveWeightQuotaHeadroom:    settings.OpenAIAdvancedSchedulerEffectiveWeightQuotaHeadroom,
		OpenAIAdvancedSchedulerEffectiveWeightUpstreamCost:     settings.OpenAIAdvancedSchedulerEffectiveWeightUpstreamCost,
		OpenAIAdvancedSchedulerEffectiveWeightPreviousResponse: settings.OpenAIAdvancedSchedulerEffectiveWeightPreviousResponse,
		OpenAIAdvancedSchedulerEffectiveWeightSessionSticky:    settings.OpenAIAdvancedSchedulerEffectiveWeightSessionSticky,
		BalanceLowNotifyEnabled:                                settings.BalanceLowNotifyEnabled,
		BalanceLowNotifyThreshold:                              settings.BalanceLowNotifyThreshold,
		BalanceLowNotifyRechargeURL:                            settings.BalanceLowNotifyRechargeURL,
		SubscriptionExpiryNotifyEnabled:                        settings.SubscriptionExpiryNotifyEnabled,
		AccountQuotaNotifyEnabled:                              settings.AccountQuotaNotifyEnabled,
		AccountQuotaNotifyEmails:                               dto.NotifyEmailEntriesFromService(settings.AccountQuotaNotifyEmails),
		PaymentEnabled:                                         paymentCfg.Enabled,
		PaymentMinAmount:                                       paymentCfg.MinAmount,
		PaymentMaxAmount:                                       paymentCfg.MaxAmount,
		PaymentDailyLimit:                                      paymentCfg.DailyLimit,
		PaymentOrderTimeoutMin:                                 paymentCfg.OrderTimeoutMin,
		PaymentMaxPendingOrders:                                paymentCfg.MaxPendingOrders,
		PaymentEnabledTypes:                                    paymentCfg.EnabledTypes,
		PaymentBalanceDisabled:                                 paymentCfg.BalanceDisabled,
		PaymentBalanceRechargeMultiplier:                       paymentCfg.BalanceRechargeMultiplier,
		PaymentSubscriptionUSDToCNYRate:                        paymentCfg.SubscriptionUSDToCNYRate,
		PaymentRechargeFeeRate:                                 paymentCfg.RechargeFeeRate,
		PaymentLoadBalanceStrat:                                paymentCfg.LoadBalanceStrategy,
		PaymentProductNamePrefix:                               paymentCfg.ProductNamePrefix,
		PaymentProductNameSuffix:                               paymentCfg.ProductNameSuffix,
		PaymentHelpImageURL:                                    paymentCfg.HelpImageURL,
		PaymentHelpText:                                        paymentCfg.HelpText,
		PaymentCancelRateLimitEnabled:                          paymentCfg.CancelRateLimitEnabled,
		PaymentCancelRateLimitMax:                              paymentCfg.CancelRateLimitMax,
		PaymentCancelRateLimitWindow:                           paymentCfg.CancelRateLimitWindow,
		PaymentCancelRateLimitUnit:                             paymentCfg.CancelRateLimitUnit,
		PaymentCancelRateLimitMode:                             paymentCfg.CancelRateLimitMode,
		PaymentAlipayForceQRCode:                               paymentCfg.AlipayForceQRCode,
		PaymentAlipayMobilePrecreateDeepLink:                   paymentCfg.AlipayMobilePrecreateDeepLink,

		ChannelMonitorEnabled:                settings.ChannelMonitorEnabled,
		ChannelMonitorDefaultIntervalSeconds: settings.ChannelMonitorDefaultIntervalSeconds,

		AvailableChannelsEnabled: settings.AvailableChannelsEnabled,

		ModelPlazaEnabled:     settings.ModelPlazaEnabled,
		ModelPlazaRequireAuth: settings.ModelPlazaRequireAuth,
		ModelPlazaDescription: settings.ModelPlazaDescription,

		AffiliateEnabled: settings.AffiliateEnabled,

		AllowUserViewErrorRequests: settings.AllowUserViewErrorRequests,
	}

	// OpenAI fast policy (stored under a dedicated setting key)
	if h.settingService != nil {
		if fastPolicy, err := h.settingService.GetOpenAIFastPolicySettings(ctx); err != nil {
			slog.Error("openai_fast_policy_settings_get_failed", "error", err)
		} else if fastPolicy != nil {
			payload.OpenAIFastPolicySettings = openaiFastPolicySettingsToDTO(fastPolicy)
		}

		// Load default platform quotas.
		if platformQuotas, err := h.settingService.GetDefaultPlatformQuotas(ctx); err != nil {
			slog.Error("default_platform_quotas_get_failed", "error", err)
		} else {
			payload.DefaultPlatformQuotas = platformQuotas
		}
	}

	return payload
}

// openaiFastPolicySettingsToDTO converts service -> dto for OpenAI fast policy.
func openaiFastPolicySettingsToDTO(s *service.OpenAIFastPolicySettings) *dto.OpenAIFastPolicySettings {
	if s == nil {
		return nil
	}
	rules := make([]dto.OpenAIFastPolicyRule, len(s.Rules))
	for i, r := range s.Rules {
		rules[i] = dto.OpenAIFastPolicyRule(r)
	}
	return &dto.OpenAIFastPolicySettings{Rules: rules}
}

// openaiFastPolicySettingsFromDTO converts dto -> service for OpenAI fast policy.
//
// openaiFastPolicySettingsFromDTO converts DTO rules to service settings.
func openaiFastPolicySettingsFromDTO(s *dto.OpenAIFastPolicySettings) *service.OpenAIFastPolicySettings {
	if s == nil {
		return nil
	}
	rules := make([]service.OpenAIFastPolicyRule, len(s.Rules))
	for i, r := range s.Rules {
		rules[i] = service.OpenAIFastPolicyRule(r)
		tier := strings.ToLower(strings.TrimSpace(rules[i].ServiceTier))
		if tier == "" {
			tier = service.OpenAIFastTierAny
		}
		rules[i].ServiceTier = tier
	}
	return &service.OpenAIFastPolicySettings{Rules: rules}
}

func loginAgreementDocumentsToDTO(items []service.LoginAgreementDocument) []dto.LoginAgreementDocument {
	result := make([]dto.LoginAgreementDocument, 0, len(items))
	for _, item := range items {
		result = append(result, dto.LoginAgreementDocument{
			ID:        item.ID,
			Title:     item.Title,
			ContentMD: item.ContentMD,
		})
	}
	return result
}

func loginAgreementDocumentsToService(items []dto.LoginAgreementDocument) []service.LoginAgreementDocument {
	result := make([]service.LoginAgreementDocument, 0, len(items))
	for _, item := range items {
		title := strings.TrimSpace(item.Title)
		content := strings.TrimSpace(item.ContentMD)
		if title == "" && content == "" {
			continue
		}
		result = append(result, service.LoginAgreementDocument{
			ID:        strings.TrimSpace(item.ID),
			Title:     title,
			ContentMD: content,
		})
	}
	return result
}

func systemSettingsResponseData(settings dto.SystemSettings, authSourceDefaults *service.AuthSourceDefaultSettings) map[string]any {
	data := make(map[string]any)
	raw, err := json.Marshal(settings)
	if err == nil {
		_ = json.Unmarshal(raw, &data)
	}
	if authSourceDefaults == nil {
		authSourceDefaults = &service.AuthSourceDefaultSettings{}
	}

	data["auth_source_default_email_balance"] = authSourceDefaults.Email.Balance
	data["auth_source_default_email_concurrency"] = authSourceDefaults.Email.Concurrency
	data["auth_source_default_email_subscriptions"] = authSourceDefaults.Email.Subscriptions
	data["auth_source_default_email_grant_on_signup"] = authSourceDefaults.Email.GrantOnSignup
	data["auth_source_default_email_grant_on_first_bind"] = authSourceDefaults.Email.GrantOnFirstBind
	data["auth_source_default_linuxdo_balance"] = authSourceDefaults.LinuxDo.Balance
	data["auth_source_default_linuxdo_concurrency"] = authSourceDefaults.LinuxDo.Concurrency
	data["auth_source_default_linuxdo_subscriptions"] = authSourceDefaults.LinuxDo.Subscriptions
	data["auth_source_default_linuxdo_grant_on_signup"] = authSourceDefaults.LinuxDo.GrantOnSignup
	data["auth_source_default_linuxdo_grant_on_first_bind"] = authSourceDefaults.LinuxDo.GrantOnFirstBind
	data["auth_source_default_dingtalk_balance"] = authSourceDefaults.DingTalk.Balance
	data["auth_source_default_dingtalk_concurrency"] = authSourceDefaults.DingTalk.Concurrency
	data["auth_source_default_dingtalk_subscriptions"] = authSourceDefaults.DingTalk.Subscriptions
	data["auth_source_default_dingtalk_grant_on_signup"] = authSourceDefaults.DingTalk.GrantOnSignup
	data["auth_source_default_dingtalk_grant_on_first_bind"] = authSourceDefaults.DingTalk.GrantOnFirstBind
	data["auth_source_default_oidc_balance"] = authSourceDefaults.OIDC.Balance
	data["auth_source_default_oidc_concurrency"] = authSourceDefaults.OIDC.Concurrency
	data["auth_source_default_oidc_subscriptions"] = authSourceDefaults.OIDC.Subscriptions
	data["auth_source_default_oidc_grant_on_signup"] = authSourceDefaults.OIDC.GrantOnSignup
	data["auth_source_default_oidc_grant_on_first_bind"] = authSourceDefaults.OIDC.GrantOnFirstBind
	data["auth_source_default_wechat_balance"] = authSourceDefaults.WeChat.Balance
	data["auth_source_default_wechat_concurrency"] = authSourceDefaults.WeChat.Concurrency
	data["auth_source_default_wechat_subscriptions"] = authSourceDefaults.WeChat.Subscriptions
	data["auth_source_default_wechat_grant_on_signup"] = authSourceDefaults.WeChat.GrantOnSignup
	data["auth_source_default_wechat_grant_on_first_bind"] = authSourceDefaults.WeChat.GrantOnFirstBind
	data["auth_source_default_github_balance"] = authSourceDefaults.GitHub.Balance
	data["auth_source_default_github_concurrency"] = authSourceDefaults.GitHub.Concurrency
	data["auth_source_default_github_subscriptions"] = authSourceDefaults.GitHub.Subscriptions
	data["auth_source_default_github_grant_on_signup"] = authSourceDefaults.GitHub.GrantOnSignup
	data["auth_source_default_github_grant_on_first_bind"] = authSourceDefaults.GitHub.GrantOnFirstBind
	data["auth_source_default_google_balance"] = authSourceDefaults.Google.Balance
	data["auth_source_default_google_concurrency"] = authSourceDefaults.Google.Concurrency
	data["auth_source_default_google_subscriptions"] = authSourceDefaults.Google.Subscriptions
	data["auth_source_default_google_grant_on_signup"] = authSourceDefaults.Google.GrantOnSignup
	data["auth_source_default_google_grant_on_first_bind"] = authSourceDefaults.Google.GrantOnFirstBind
	data["auth_source_default_email_platform_quotas"] = authSourceDefaults.Email.PlatformQuotas
	data["auth_source_default_linuxdo_platform_quotas"] = authSourceDefaults.LinuxDo.PlatformQuotas
	data["auth_source_default_oidc_platform_quotas"] = authSourceDefaults.OIDC.PlatformQuotas
	data["auth_source_default_wechat_platform_quotas"] = authSourceDefaults.WeChat.PlatformQuotas
	data["auth_source_default_github_platform_quotas"] = authSourceDefaults.GitHub.PlatformQuotas
	data["auth_source_default_google_platform_quotas"] = authSourceDefaults.Google.PlatformQuotas
	data["auth_source_default_dingtalk_platform_quotas"] = authSourceDefaults.DingTalk.PlatformQuotas
	data["force_email_on_third_party_signup"] = authSourceDefaults.ForceEmailOnThirdPartySignup

	return data

}
func toSoraS3SettingsDTO(settings *service.SoraS3Settings) dto.SoraS3Settings {
	if settings == nil {
		return dto.SoraS3Settings{}
	}
	return dto.SoraS3Settings{
		Enabled:                   settings.Enabled,
		Endpoint:                  settings.Endpoint,
		Region:                    settings.Region,
		Bucket:                    settings.Bucket,
		AccessKeyID:               settings.AccessKeyID,
		SecretAccessKeyConfigured: settings.SecretAccessKeyConfigured,
		Prefix:                    settings.Prefix,
		ForcePathStyle:            settings.ForcePathStyle,
		CDNURL:                    settings.CDNURL,
		DefaultStorageQuotaBytes:  settings.DefaultStorageQuotaBytes,
	}
}

func toSoraS3ProfileDTO(profile service.SoraS3Profile) dto.SoraS3Profile {
	return dto.SoraS3Profile{
		ProfileID:                 profile.ProfileID,
		Name:                      profile.Name,
		IsActive:                  profile.IsActive,
		Enabled:                   profile.Enabled,
		Endpoint:                  profile.Endpoint,
		Region:                    profile.Region,
		Bucket:                    profile.Bucket,
		AccessKeyID:               profile.AccessKeyID,
		SecretAccessKeyConfigured: profile.SecretAccessKeyConfigured,
		Prefix:                    profile.Prefix,
		ForcePathStyle:            profile.ForcePathStyle,
		CDNURL:                    profile.CDNURL,
		DefaultStorageQuotaBytes:  profile.DefaultStorageQuotaBytes,
		UpdatedAt:                 profile.UpdatedAt,
	}
}

func toTLSFingerprintProfileDTO(profile service.TLSFingerprintProfile) dto.TLSFingerprintProfile {
	return dto.TLSFingerprintProfile{
		ProfileID:    profile.ProfileID,
		Name:         profile.Name,
		Enabled:      profile.Enabled,
		EnableGREASE: profile.EnableGREASE,
		CipherSuites: profile.CipherSuites,
		Curves:       profile.Curves,
		PointFormats: profile.PointFormats,
		UpdatedAt:    profile.UpdatedAt,
	}
}

func uint16To64Slice(values []uint16) []uint64 {
	result := make([]uint64, 0, len(values))
	for _, value := range values {
		result = append(result, uint64(value))
	}
	return result
}

func uint8To64Slice(values []uint8) []uint64 {
	result := make([]uint64, 0, len(values))
	for _, value := range values {
		result = append(result, uint64(value))
	}
	return result
}

func validateTLSFingerprintSlice(field string, values []uint64) error {
	for _, value := range values {
		if value == 0 {
			return fmt.Errorf("%s cannot contain zero values", field)
		}
	}
	return nil
}

func validateSoraS3RequiredWhenEnabled(enabled bool, endpoint, bucket, accessKeyID, secretAccessKey string, hasStoredSecret bool) error {
	if !enabled {
		return nil
	}
	if strings.TrimSpace(endpoint) == "" {
		return fmt.Errorf("S3 Endpoint is required when enabled")
	}
	if strings.TrimSpace(bucket) == "" {
		return fmt.Errorf("S3 Bucket is required when enabled")
	}
	if strings.TrimSpace(accessKeyID) == "" {
		return fmt.Errorf("S3 Access Key ID is required when enabled")
	}
	if strings.TrimSpace(secretAccessKey) != "" || hasStoredSecret {
		return nil
	}
	return fmt.Errorf("S3 Secret Access Key is required when enabled")
}

type UpdateTLSFingerprintSettingsRequest struct {
	Enabled bool `json:"enabled"`
}

type CreateTLSFingerprintProfileRequest struct {
	ProfileID    string   `json:"profile_id"`
	Name         string   `json:"name"`
	Enabled      bool     `json:"enabled"`
	EnableGREASE bool     `json:"enable_grease"`
	CipherSuites []uint16 `json:"cipher_suites"`
	Curves       []uint16 `json:"curves"`
	PointFormats []uint8  `json:"point_formats"`
}

type UpdateTLSFingerprintProfileRequest struct {
	Name         string   `json:"name"`
	Enabled      bool     `json:"enabled"`
	EnableGREASE bool     `json:"enable_grease"`
	CipherSuites []uint16 `json:"cipher_suites"`
	Curves       []uint16 `json:"curves"`
	PointFormats []uint8  `json:"point_formats"`
}

func (h *SettingHandler) GetTLSFingerprintSettings(c *gin.Context) {
	h.GetTLSFingerprintSettingsGateway(gatewayctx.FromGin(c))
}

func (h *SettingHandler) GetTLSFingerprintSettingsGateway(c gatewayctx.GatewayContext) {
	result, err := h.settingService.ListTLSFingerprintProfiles(c.Request().Context())
	if err != nil {
		response.ErrorFromContext(c, err)
		return
	}
	items := make([]dto.TLSFingerprintProfile, 0, len(result.Items))
	for idx := range result.Items {
		items = append(items, toTLSFingerprintProfileDTO(result.Items[idx]))
	}
	response.SuccessContext(c, dto.ListTLSFingerprintProfilesResponse{
		Enabled: result.Enabled,
		Items:   items,
	})
}

func (h *SettingHandler) UpdateTLSFingerprintSettings(c *gin.Context) {
	h.UpdateTLSFingerprintSettingsGateway(gatewayctx.FromGin(c))
}

func (h *SettingHandler) UpdateTLSFingerprintSettingsGateway(c gatewayctx.GatewayContext) {
	var req UpdateTLSFingerprintSettingsRequest
	if err := c.BindJSON(&req); err != nil {
		response.ErrorContext(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}
	if err := h.settingService.SetTLSFingerprintSettings(c.Request().Context(), &service.TLSFingerprintSettings{
		Enabled: req.Enabled,
	}); err != nil {
		response.ErrorFromContext(c, err)
		return
	}
	response.SuccessContext(c, dto.TLSFingerprintSettings{Enabled: req.Enabled})
}

func (h *SettingHandler) ListTLSFingerprintProfiles(c *gin.Context) {
	h.ListTLSFingerprintProfilesGateway(gatewayctx.FromGin(c))
}

func (h *SettingHandler) ListTLSFingerprintProfilesGateway(c gatewayctx.GatewayContext) {
	result, err := h.settingService.ListTLSFingerprintProfiles(c.Request().Context())
	if err != nil {
		response.ErrorFromContext(c, err)
		return
	}
	items := make([]dto.TLSFingerprintProfile, 0, len(result.Items))
	for idx := range result.Items {
		items = append(items, toTLSFingerprintProfileDTO(result.Items[idx]))
	}
	response.SuccessContext(c, dto.ListTLSFingerprintProfilesResponse{
		Enabled: result.Enabled,
		Items:   items,
	})
}

func (h *SettingHandler) CreateTLSFingerprintProfile(c *gin.Context) {
	h.CreateTLSFingerprintProfileGateway(gatewayctx.FromGin(c))
}

func (h *SettingHandler) CreateTLSFingerprintProfileGateway(c gatewayctx.GatewayContext) {
	var req CreateTLSFingerprintProfileRequest
	if err := c.BindJSON(&req); err != nil {
		response.ErrorContext(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}
	if strings.TrimSpace(req.ProfileID) == "" {
		response.ErrorContext(c, http.StatusBadRequest, "Profile ID is required")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		response.ErrorContext(c, http.StatusBadRequest, "Name is required")
		return
	}
	if err := validateTLSFingerprintSlice("cipher_suites", uint16To64Slice(req.CipherSuites)); err != nil {
		response.ErrorContext(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateTLSFingerprintSlice("curves", uint16To64Slice(req.Curves)); err != nil {
		response.ErrorContext(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateTLSFingerprintSlice("point_formats", uint8To64Slice(req.PointFormats)); err != nil {
		response.ErrorContext(c, http.StatusBadRequest, err.Error())
		return
	}

	created, err := h.settingService.CreateTLSFingerprintProfile(c.Request().Context(), &service.TLSFingerprintProfile{
		ProfileID:    req.ProfileID,
		Name:         req.Name,
		Enabled:      req.Enabled,
		EnableGREASE: req.EnableGREASE,
		CipherSuites: req.CipherSuites,
		Curves:       req.Curves,
		PointFormats: req.PointFormats,
	})
	if err != nil {
		response.ErrorFromContext(c, err)
		return
	}
	response.SuccessContext(c, toTLSFingerprintProfileDTO(*created))
}

func (h *SettingHandler) UpdateTLSFingerprintProfile(c *gin.Context) {
	h.UpdateTLSFingerprintProfileGateway(gatewayctx.FromGin(c))
}

func (h *SettingHandler) UpdateTLSFingerprintProfileGateway(c gatewayctx.GatewayContext) {
	profileID := strings.TrimSpace(c.PathParam("profile_id"))
	if profileID == "" {
		response.ErrorContext(c, http.StatusBadRequest, "Profile ID is required")
		return
	}
	var req UpdateTLSFingerprintProfileRequest
	if err := c.BindJSON(&req); err != nil {
		response.ErrorContext(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		response.ErrorContext(c, http.StatusBadRequest, "Name is required")
		return
	}
	if err := validateTLSFingerprintSlice("cipher_suites", uint16To64Slice(req.CipherSuites)); err != nil {
		response.ErrorContext(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateTLSFingerprintSlice("curves", uint16To64Slice(req.Curves)); err != nil {
		response.ErrorContext(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateTLSFingerprintSlice("point_formats", uint8To64Slice(req.PointFormats)); err != nil {
		response.ErrorContext(c, http.StatusBadRequest, err.Error())
		return
	}

	updated, err := h.settingService.UpdateTLSFingerprintProfile(c.Request().Context(), profileID, &service.TLSFingerprintProfile{
		Name:         req.Name,
		Enabled:      req.Enabled,
		EnableGREASE: req.EnableGREASE,
		CipherSuites: req.CipherSuites,
		Curves:       req.Curves,
		PointFormats: req.PointFormats,
	})
	if err != nil {
		response.ErrorFromContext(c, err)
		return
	}
	response.SuccessContext(c, toTLSFingerprintProfileDTO(*updated))
}

func (h *SettingHandler) DeleteTLSFingerprintProfile(c *gin.Context) {
	h.DeleteTLSFingerprintProfileGateway(gatewayctx.FromGin(c))
}

func (h *SettingHandler) DeleteTLSFingerprintProfileGateway(c gatewayctx.GatewayContext) {
	profileID := strings.TrimSpace(c.PathParam("profile_id"))
	if profileID == "" {
		response.ErrorContext(c, http.StatusBadRequest, "Profile ID is required")
		return
	}
	if err := h.settingService.DeleteTLSFingerprintProfile(c.Request().Context(), profileID); err != nil {
		response.ErrorFromContext(c, err)
		return
	}
	response.SuccessContext(c, gin.H{"deleted": true})
}

func findSoraS3ProfileByID(items []service.SoraS3Profile, profileID string) *service.SoraS3Profile {
	for idx := range items {
		if items[idx].ProfileID == profileID {
			return &items[idx]
		}
	}
	return nil
}

// GET /api/v1/admin/settings/sora-s3
func (h *SettingHandler) GetSoraS3Settings(c *gin.Context) {
	h.GetSoraS3SettingsGateway(gatewayctx.FromGin(c))
}

func (h *SettingHandler) GetSoraS3SettingsGateway(c gatewayctx.GatewayContext) {
	settings, err := h.settingService.GetSoraS3Settings(c.Request().Context())
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, toSoraS3SettingsDTO(settings))
}

func (h *SettingHandler) ListSoraS3Profiles(c *gin.Context) {
	h.ListSoraS3ProfilesGateway(gatewayctx.FromGin(c))
}

func (h *SettingHandler) ListSoraS3ProfilesGateway(c gatewayctx.GatewayContext) {
	result, err := h.settingService.ListSoraS3Profiles(c.Request().Context())
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	items := make([]dto.SoraS3Profile, 0, len(result.Items))
	for idx := range result.Items {
		items = append(items, toSoraS3ProfileDTO(result.Items[idx]))
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, dto.ListSoraS3ProfilesResponse{
		ActiveProfileID: result.ActiveProfileID,
		Items:           items,
	})
}

// UpdateSoraS3SettingsRequest updates or tests Sora S3 settings.
type UpdateSoraS3SettingsRequest struct {
	ProfileID                string `json:"profile_id"`
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
}

type CreateSoraS3ProfileRequest struct {
	ProfileID                string `json:"profile_id"`
	Name                     string `json:"name"`
	SetActive                bool   `json:"set_active"`
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
}

type UpdateSoraS3ProfileRequest struct {
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
}

// POST /api/v1/admin/settings/sora-s3/profiles
func (h *SettingHandler) CreateSoraS3Profile(c *gin.Context) {
	h.CreateSoraS3ProfileGateway(gatewayctx.FromGin(c))
}

func (h *SettingHandler) CreateSoraS3ProfileGateway(c gatewayctx.GatewayContext) {
	var req CreateSoraS3ProfileRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	if req.DefaultStorageQuotaBytes < 0 {
		req.DefaultStorageQuotaBytes = 0
	}
	if strings.TrimSpace(req.Name) == "" {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Name is required")
		return
	}
	if strings.TrimSpace(req.ProfileID) == "" {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Profile ID is required")
		return
	}
	if err := validateSoraS3RequiredWhenEnabled(req.Enabled, req.Endpoint, req.Bucket, req.AccessKeyID, req.SecretAccessKey, false); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, err.Error())
		return
	}

	created, err := h.settingService.CreateSoraS3Profile(c.Request().Context(), &service.SoraS3Profile{
		ProfileID:                req.ProfileID,
		Name:                     req.Name,
		Enabled:                  req.Enabled,
		Endpoint:                 req.Endpoint,
		Region:                   req.Region,
		Bucket:                   req.Bucket,
		AccessKeyID:              req.AccessKeyID,
		SecretAccessKey:          req.SecretAccessKey,
		Prefix:                   req.Prefix,
		ForcePathStyle:           req.ForcePathStyle,
		CDNURL:                   req.CDNURL,
		DefaultStorageQuotaBytes: req.DefaultStorageQuotaBytes,
	}, req.SetActive)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	response.SuccessContext(gatewayJSONResponder{ctx: c}, toSoraS3ProfileDTO(*created))
}

// PUT /api/v1/admin/settings/sora-s3/profiles/:profile_id
func (h *SettingHandler) UpdateSoraS3Profile(c *gin.Context) {
	h.UpdateSoraS3ProfileGateway(gatewayctx.FromGin(c))
}

func (h *SettingHandler) UpdateSoraS3ProfileGateway(c gatewayctx.GatewayContext) {
	profileID := strings.TrimSpace(c.PathParam("profile_id"))
	if profileID == "" {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Profile ID is required")
		return
	}

	var req UpdateSoraS3ProfileRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	if req.DefaultStorageQuotaBytes < 0 {
		req.DefaultStorageQuotaBytes = 0
	}
	if strings.TrimSpace(req.Name) == "" {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Name is required")
		return
	}

	existingList, err := h.settingService.ListSoraS3Profiles(c.Request().Context())
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	existing := findSoraS3ProfileByID(existingList.Items, profileID)
	if existing == nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, service.ErrSoraS3ProfileNotFound)
		return
	}
	if err := validateSoraS3RequiredWhenEnabled(req.Enabled, req.Endpoint, req.Bucket, req.AccessKeyID, req.SecretAccessKey, existing.SecretAccessKeyConfigured); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, err.Error())
		return
	}

	updated, updateErr := h.settingService.UpdateSoraS3Profile(c.Request().Context(), profileID, &service.SoraS3Profile{
		Name:                     req.Name,
		Enabled:                  req.Enabled,
		Endpoint:                 req.Endpoint,
		Region:                   req.Region,
		Bucket:                   req.Bucket,
		AccessKeyID:              req.AccessKeyID,
		SecretAccessKey:          req.SecretAccessKey,
		Prefix:                   req.Prefix,
		ForcePathStyle:           req.ForcePathStyle,
		CDNURL:                   req.CDNURL,
		DefaultStorageQuotaBytes: req.DefaultStorageQuotaBytes,
	})
	if updateErr != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, updateErr)
		return
	}

	response.SuccessContext(gatewayJSONResponder{ctx: c}, toSoraS3ProfileDTO(*updated))
}

// DELETE /api/v1/admin/settings/sora-s3/profiles/:profile_id
func (h *SettingHandler) DeleteSoraS3Profile(c *gin.Context) {
	h.DeleteSoraS3ProfileGateway(gatewayctx.FromGin(c))
}

func (h *SettingHandler) DeleteSoraS3ProfileGateway(c gatewayctx.GatewayContext) {
	profileID := strings.TrimSpace(c.PathParam("profile_id"))
	if profileID == "" {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Profile ID is required")
		return
	}
	if err := h.settingService.DeleteSoraS3Profile(c.Request().Context(), profileID); err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, map[string]any{"deleted": true})
}

// POST /api/v1/admin/settings/sora-s3/profiles/:profile_id/activate
func (h *SettingHandler) SetActiveSoraS3Profile(c *gin.Context) {
	h.SetActiveSoraS3ProfileGateway(gatewayctx.FromGin(c))
}

func (h *SettingHandler) SetActiveSoraS3ProfileGateway(c gatewayctx.GatewayContext) {
	profileID := strings.TrimSpace(c.PathParam("profile_id"))
	if profileID == "" {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Profile ID is required")
		return
	}
	active, err := h.settingService.SetActiveSoraS3Profile(c.Request().Context(), profileID)
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, toSoraS3ProfileDTO(*active))
}

// PUT /api/v1/admin/settings/sora-s3
func (h *SettingHandler) UpdateSoraS3Settings(c *gin.Context) {
	h.UpdateSoraS3SettingsGateway(gatewayctx.FromGin(c))
}

func (h *SettingHandler) UpdateSoraS3SettingsGateway(c gatewayctx.GatewayContext) {
	var req UpdateSoraS3SettingsRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	existing, err := h.settingService.GetSoraS3Settings(c.Request().Context())
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	if req.DefaultStorageQuotaBytes < 0 {
		req.DefaultStorageQuotaBytes = 0
	}
	if err := validateSoraS3RequiredWhenEnabled(req.Enabled, req.Endpoint, req.Bucket, req.AccessKeyID, req.SecretAccessKey, existing.SecretAccessKeyConfigured); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, err.Error())
		return
	}

	settings := &service.SoraS3Settings{
		Enabled:                  req.Enabled,
		Endpoint:                 req.Endpoint,
		Region:                   req.Region,
		Bucket:                   req.Bucket,
		AccessKeyID:              req.AccessKeyID,
		SecretAccessKey:          req.SecretAccessKey,
		Prefix:                   req.Prefix,
		ForcePathStyle:           req.ForcePathStyle,
		CDNURL:                   req.CDNURL,
		DefaultStorageQuotaBytes: req.DefaultStorageQuotaBytes,
	}
	if err := h.settingService.SetSoraS3Settings(c.Request().Context(), settings); err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}

	updatedSettings, err := h.settingService.GetSoraS3Settings(c.Request().Context())
	if err != nil {
		response.ErrorFromContext(gatewayJSONResponder{ctx: c}, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, toSoraS3SettingsDTO(updatedSettings))
}

func (h *SettingHandler) TestSoraS3Connection(c *gin.Context) {
	h.TestSoraS3ConnectionGateway(gatewayctx.FromGin(c))
}

func (h *SettingHandler) TestSoraS3ConnectionGateway(c gatewayctx.GatewayContext) {
	if h.soraS3Storage == nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, 500, "S3 存储服务未初始化")
		return
	}

	var req UpdateSoraS3SettingsRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}
	if !req.Enabled {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "S3 未启用，无法测试连接")
		return
	}

	if req.SecretAccessKey == "" {
		if req.ProfileID != "" {
			profiles, err := h.settingService.ListSoraS3Profiles(c.Request().Context())
			if err == nil {
				profile := findSoraS3ProfileByID(profiles.Items, req.ProfileID)
				if profile != nil {
					req.SecretAccessKey = profile.SecretAccessKey
				}
			}
		}
		if req.SecretAccessKey == "" {
			existing, err := h.settingService.GetSoraS3Settings(c.Request().Context())
			if err == nil {
				req.SecretAccessKey = existing.SecretAccessKey
			}
		}
	}

	testCfg := &service.SoraS3Settings{
		Enabled:         true,
		Endpoint:        req.Endpoint,
		Region:          req.Region,
		Bucket:          req.Bucket,
		AccessKeyID:     req.AccessKeyID,
		SecretAccessKey: req.SecretAccessKey,
		Prefix:          req.Prefix,
		ForcePathStyle:  req.ForcePathStyle,
		CDNURL:          req.CDNURL,
	}
	if err := h.soraS3Storage.TestConnectionWithSettings(c.Request().Context(), testCfg); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, 400, "S3 连接测试失败: "+err.Error())
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, map[string]any{"message": "S3 连接成功"})
}
