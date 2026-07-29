package service

// Setting keys introduced by the Wei-Shaw v0.1.163 base that are absent from
// the production customization's domain_constants.go.
const (
	AdminRechargeRebateEnabledDefault = false
	defaultAuthSourceBalance          = 0
	defaultAuthSourceConcurrency      = 5
	defaultWeChatConnectMode          = "open"
	defaultWeChatConnectScopes        = "snsapi_login"
	defaultGitHubOAuthAuthorize       = "https://github.com/login/oauth/authorize"
	defaultGitHubOAuthToken           = "https://github.com/login/oauth/access_token"
	defaultGitHubOAuthUserInfo        = "https://api.github.com/user"
	defaultGitHubOAuthEmails          = "https://api.github.com/user/emails"
	defaultGitHubOAuthScopes          = "read:user user:email"
	defaultGitHubOAuthFrontend        = "/auth/oauth/callback"
	defaultGoogleOAuthAuthorize       = "https://accounts.google.com/o/oauth2/v2/auth"
	defaultGoogleOAuthToken           = "https://oauth2.googleapis.com/token"
	defaultGoogleOAuthUserInfo        = "https://openidconnect.googleapis.com/v1/userinfo"
	defaultGoogleOAuthScopes          = "openid email profile"
	defaultGoogleOAuthFrontend        = "/auth/oauth/callback"
	defaultLoginAgreementMode         = "modal"
	defaultLoginAgreementDate         = "2026-03-31"

	SettingKeyAffiliateAdminRechargeEnabled             = "affiliate_admin_recharge_enabled"
	SettingKeyCyberSessionBlockEnabled                  = "cyber_session_block_enabled"
	SettingKeyCyberSessionBlockTTLSeconds               = "cyber_session_block_ttl_seconds"
	SettingKeyLoginAgreementEnabled                     = "login_agreement_enabled"
	SettingKeyLoginAgreementMode                        = "login_agreement_mode"
	SettingKeyLoginAgreementUpdatedAt                   = "login_agreement_updated_at"
	SettingKeyLoginAgreementDocuments                   = "login_agreement_documents"
	SettingKeyAPIKeyACLTrustForwardedIP                 = "api_key_acl_trust_forwarded_ip"
	SettingKeyForwardedClientIPHeaders                  = "forwarded_client_ip_headers"
	settingKeyForwardedClientIPModeV2                   = "forwarded_client_ip_mode_v2_migrated"
	SettingKeySessionBindingEnabled                     = "session_binding_enabled"
	SettingKeyStepUpEnabled                             = "step_up_enabled"
	SettingKeyAuditLogRetentionDays                     = "audit_log_retention_days"
	SettingKeyDingTalkConnectEnabled                    = "dingtalk_connect_enabled"
	SettingKeyDingTalkConnectClientID                   = "dingtalk_connect_client_id"
	SettingKeyDingTalkConnectClientSecret               = "dingtalk_connect_client_secret"
	SettingKeyDingTalkConnectRedirectURL                = "dingtalk_connect_redirect_url"
	SettingKeyDingTalkConnectCorpRestrictionPolicy      = "dingtalk_connect_corp_restriction_policy"
	SettingKeyDingTalkConnectInternalCorpID             = "dingtalk_connect_internal_corp_id"
	SettingKeyDingTalkConnectBypassRegistration         = "dingtalk_connect_bypass_registration"
	SettingKeyDingTalkConnectSyncCorpEmail              = "dingtalk_connect_sync_corp_email"
	SettingKeyDingTalkConnectSyncDisplayName            = "dingtalk_connect_sync_display_name"
	SettingKeyDingTalkConnectSyncDept                   = "dingtalk_connect_sync_dept"
	SettingKeyDingTalkConnectSyncCorpEmailAttrKey       = "dingtalk_connect_sync_corp_email_attr_key"
	SettingKeyDingTalkConnectSyncDisplayNameAttrKey     = "dingtalk_connect_sync_display_name_attr_key"
	SettingKeyDingTalkConnectSyncDeptAttrKey            = "dingtalk_connect_sync_dept_attr_key"
	SettingKeyDingTalkConnectSyncCorpEmailAttrName      = "dingtalk_connect_sync_corp_email_attr_name"
	SettingKeyDingTalkConnectSyncDisplayNameAttrName    = "dingtalk_connect_sync_display_name_attr_name"
	SettingKeyDingTalkConnectSyncDeptAttrName           = "dingtalk_connect_sync_dept_attr_name"
	SettingKeyWeChatConnectEnabled                      = "wechat_connect_enabled"
	SettingKeyWeChatConnectAppID                        = "wechat_connect_app_id"
	SettingKeyWeChatConnectAppSecret                    = "wechat_connect_app_secret"
	SettingKeyWeChatConnectOpenAppID                    = "wechat_connect_open_app_id"
	SettingKeyWeChatConnectOpenAppSecret                = "wechat_connect_open_app_secret"
	SettingKeyWeChatConnectMPAppID                      = "wechat_connect_mp_app_id"
	SettingKeyWeChatConnectMPAppSecret                  = "wechat_connect_mp_app_secret"
	SettingKeyWeChatConnectMobileAppID                  = "wechat_connect_mobile_app_id"
	SettingKeyWeChatConnectMobileAppSecret              = "wechat_connect_mobile_app_secret"
	SettingKeyWeChatConnectOpenEnabled                  = "wechat_connect_open_enabled"
	SettingKeyWeChatConnectMPEnabled                    = "wechat_connect_mp_enabled"
	SettingKeyWeChatConnectMobileEnabled                = "wechat_connect_mobile_enabled"
	SettingKeyWeChatConnectMode                         = "wechat_connect_mode"
	SettingKeyWeChatConnectScopes                       = "wechat_connect_scopes"
	SettingKeyWeChatConnectRedirectURL                  = "wechat_connect_redirect_url"
	SettingKeyWeChatConnectFrontendRedirectURL          = "wechat_connect_frontend_redirect_url"
	SettingKeyOIDCConnectEnabled                        = "oidc_connect_enabled"
	SettingKeyOIDCConnectProviderName                   = "oidc_connect_provider_name"
	SettingKeyOIDCConnectClientID                       = "oidc_connect_client_id"
	SettingKeyOIDCConnectClientSecret                   = "oidc_connect_client_secret"
	SettingKeyOIDCConnectIssuerURL                      = "oidc_connect_issuer_url"
	SettingKeyOIDCConnectDiscoveryURL                   = "oidc_connect_discovery_url"
	SettingKeyOIDCConnectAuthorizeURL                   = "oidc_connect_authorize_url"
	SettingKeyOIDCConnectTokenURL                       = "oidc_connect_token_url"
	SettingKeyOIDCConnectUserInfoURL                    = "oidc_connect_userinfo_url"
	SettingKeyOIDCConnectJWKSURL                        = "oidc_connect_jwks_url"
	SettingKeyOIDCConnectScopes                         = "oidc_connect_scopes"
	SettingKeyOIDCConnectRedirectURL                    = "oidc_connect_redirect_url"
	SettingKeyOIDCConnectFrontendRedirectURL            = "oidc_connect_frontend_redirect_url"
	SettingKeyOIDCConnectTokenAuthMethod                = "oidc_connect_token_auth_method"
	SettingKeyOIDCConnectUsePKCE                        = "oidc_connect_use_pkce"
	SettingKeyOIDCConnectValidateIDToken                = "oidc_connect_validate_id_token"
	SettingKeyOIDCConnectAllowedSigningAlgs             = "oidc_connect_allowed_signing_algs"
	SettingKeyOIDCConnectClockSkewSeconds               = "oidc_connect_clock_skew_seconds"
	SettingKeyOIDCConnectRequireEmailVerified           = "oidc_connect_require_email_verified"
	SettingKeyOIDCConnectUserInfoEmailPath              = "oidc_connect_userinfo_email_path"
	SettingKeyOIDCConnectUserInfoIDPath                 = "oidc_connect_userinfo_id_path"
	SettingKeyOIDCConnectUserInfoUsernamePath           = "oidc_connect_userinfo_username_path"
	SettingKeyGitHubOAuthEnabled                        = "github_oauth_enabled"
	SettingKeyGitHubOAuthClientID                       = "github_oauth_client_id"
	SettingKeyGitHubOAuthClientSecret                   = "github_oauth_client_secret"
	SettingKeyGitHubOAuthRedirectURL                    = "github_oauth_redirect_url"
	SettingKeyGitHubOAuthFrontendRedirectURL            = "github_oauth_frontend_redirect_url"
	SettingKeyGoogleOAuthEnabled                        = "google_oauth_enabled"
	SettingKeyGoogleOAuthClientID                       = "google_oauth_client_id"
	SettingKeyGoogleOAuthClientSecret                   = "google_oauth_client_secret"
	SettingKeyGoogleOAuthRedirectURL                    = "google_oauth_redirect_url"
	SettingKeyGoogleOAuthFrontendRedirectURL            = "google_oauth_frontend_redirect_url"
	SettingKeyTableDefaultPageSize                      = "table_default_page_size"
	SettingKeyTablePageSizeOptions                      = "table_page_size_options"
	SettingKeyCustomEndpoints                           = "custom_endpoints"
	SettingKeyDefaultUserRPMLimit                       = "default_user_rpm_limit"
	SettingKeyAuthSourceDefaultEmailBalance             = "auth_source_default_email_balance"
	SettingKeyAuthSourceDefaultEmailConcurrency         = "auth_source_default_email_concurrency"
	SettingKeyAuthSourceDefaultEmailSubscriptions       = "auth_source_default_email_subscriptions"
	SettingKeyAuthSourceDefaultEmailGrantOnSignup       = "auth_source_default_email_grant_on_signup"
	SettingKeyAuthSourceDefaultEmailGrantOnFirstBind    = "auth_source_default_email_grant_on_first_bind"
	SettingKeyAuthSourceDefaultLinuxDoBalance           = "auth_source_default_linuxdo_balance"
	SettingKeyAuthSourceDefaultLinuxDoConcurrency       = "auth_source_default_linuxdo_concurrency"
	SettingKeyAuthSourceDefaultLinuxDoSubscriptions     = "auth_source_default_linuxdo_subscriptions"
	SettingKeyAuthSourceDefaultLinuxDoGrantOnSignup     = "auth_source_default_linuxdo_grant_on_signup"
	SettingKeyAuthSourceDefaultLinuxDoGrantOnFirstBind  = "auth_source_default_linuxdo_grant_on_first_bind"
	SettingKeyAuthSourceDefaultOIDCBalance              = "auth_source_default_oidc_balance"
	SettingKeyAuthSourceDefaultOIDCConcurrency          = "auth_source_default_oidc_concurrency"
	SettingKeyAuthSourceDefaultOIDCSubscriptions        = "auth_source_default_oidc_subscriptions"
	SettingKeyAuthSourceDefaultOIDCGrantOnSignup        = "auth_source_default_oidc_grant_on_signup"
	SettingKeyAuthSourceDefaultOIDCGrantOnFirstBind     = "auth_source_default_oidc_grant_on_first_bind"
	SettingKeyAuthSourceDefaultWeChatBalance            = "auth_source_default_wechat_balance"
	SettingKeyAuthSourceDefaultWeChatConcurrency        = "auth_source_default_wechat_concurrency"
	SettingKeyAuthSourceDefaultWeChatSubscriptions      = "auth_source_default_wechat_subscriptions"
	SettingKeyAuthSourceDefaultWeChatGrantOnSignup      = "auth_source_default_wechat_grant_on_signup"
	SettingKeyAuthSourceDefaultWeChatGrantOnFirstBind   = "auth_source_default_wechat_grant_on_first_bind"
	SettingKeyAuthSourceDefaultGitHubBalance            = "auth_source_default_github_balance"
	SettingKeyAuthSourceDefaultGitHubConcurrency        = "auth_source_default_github_concurrency"
	SettingKeyAuthSourceDefaultGitHubSubscriptions      = "auth_source_default_github_subscriptions"
	SettingKeyAuthSourceDefaultGitHubGrantOnSignup      = "auth_source_default_github_grant_on_signup"
	SettingKeyAuthSourceDefaultGitHubGrantOnFirstBind   = "auth_source_default_github_grant_on_first_bind"
	SettingKeyAuthSourceDefaultGoogleBalance            = "auth_source_default_google_balance"
	SettingKeyAuthSourceDefaultGoogleConcurrency        = "auth_source_default_google_concurrency"
	SettingKeyAuthSourceDefaultGoogleSubscriptions      = "auth_source_default_google_subscriptions"
	SettingKeyAuthSourceDefaultGoogleGrantOnSignup      = "auth_source_default_google_grant_on_signup"
	SettingKeyAuthSourceDefaultGoogleGrantOnFirstBind   = "auth_source_default_google_grant_on_first_bind"
	SettingKeyAuthSourceDefaultDingTalkBalance          = "auth_source_default_dingtalk_balance"
	SettingKeyAuthSourceDefaultDingTalkConcurrency      = "auth_source_default_dingtalk_concurrency"
	SettingKeyAuthSourceDefaultDingTalkSubscriptions    = "auth_source_default_dingtalk_subscriptions"
	SettingKeyAuthSourceDefaultDingTalkGrantOnSignup    = "auth_source_default_dingtalk_grant_on_signup"
	SettingKeyAuthSourceDefaultDingTalkGrantOnFirstBind = "auth_source_default_dingtalk_grant_on_first_bind"
	SettingKeyForceEmailOnThirdPartySignup              = "force_email_on_third_party_signup"
	SettingKeyRateLimit429CooldownSettings              = "rate_limit_429_cooldown_settings"
	SettingKeyOpenAIFastPolicySettings                  = "openai_fast_policy_settings"
	SettingKeyMinCodexVersion                           = "min_codex_version"
	SettingKeyMaxCodexVersion                           = "max_codex_version"
	SettingKeyCodexCLIOnlyBlacklist                     = "codex_cli_only_blacklist"
	SettingKeyCodexCLIOnlyWhitelist                     = "codex_cli_only_whitelist"
	SettingKeyCodexCLIOnlyAllowAppServerClients         = "codex_cli_only_allow_app_server_clients"
	SettingKeyCodexCLIOnlyAllowBodyEngineFingerprint    = "codex_cli_only_allow_body_engine_fingerprint"
	SettingKeyCodexCLIOnlyEngineFingerprintSignals      = "codex_cli_only_engine_fingerprint_signals"
	SettingKeyEnableFingerprintUnification              = "enable_fingerprint_unification"
	SettingKeyEnableMetadataPassthrough                 = "enable_metadata_passthrough"
	SettingKeyEnableCCHSigning                          = "enable_cch_signing"
	SettingKeyEnableClaudeOAuthSystemPromptInjection    = "enable_claude_oauth_system_prompt_injection"
	SettingKeyClaudeOAuthSystemPrompt                   = "claude_oauth_system_prompt"
	SettingKeyClaudeOAuthSystemPromptBlocks             = "claude_oauth_system_prompt_blocks"
	SettingKeyEnableAnthropicCacheTTL1hInjection        = "enable_anthropic_cache_ttl_1h_injection"
	SettingKeyEnableClientDatelineNormalization         = "enable_client_dateline_normalization"
	SettingKeyRewriteMessageCacheControl                = "rewrite_message_cache_control"
	SettingKeyAntigravityUserAgentVersion               = "antigravity_user_agent_version"
	SettingKeyOpenAICodexUserAgent                      = "openai_codex_user_agent"
	SettingKeyOpenAIAllowClaudeCodeCodexPlugin          = "openai_allow_claude_code_codex_plugin"
	SettingKeySubscriptionExpiryNotifyEnabled           = "subscription_expiry_notify_enabled"
	SettingKeyWebSearchEmulationConfig                  = "web_search_emulation_config"
	SettingKeyDefaultPlatformQuotas                     = "default_platform_quotas"
	SettingKeyAllowUserViewErrorRequests                = "allow_user_view_error_requests"
)

var AllowedQuotaPlatforms = []string{
	PlatformAnthropic,
	PlatformOpenAI,
	PlatformGemini,
	PlatformAntigravity,
	PlatformGrok,
}

func IsAllowedQuotaPlatform(platform string) bool {
	for _, allowed := range AllowedQuotaPlatforms {
		if platform == allowed {
			return true
		}
	}
	return false
}

func SettingKeyAuthSourcePlatformQuotas(source string) string {
	return "auth_source_default_" + source + "_platform_quotas"
}

var (
	emailAuthSourceDefaultKeys = authSourceDefaultKeySet{
		source: "email", balance: SettingKeyAuthSourceDefaultEmailBalance,
		concurrency:      SettingKeyAuthSourceDefaultEmailConcurrency,
		subscriptions:    SettingKeyAuthSourceDefaultEmailSubscriptions,
		grantOnSignup:    SettingKeyAuthSourceDefaultEmailGrantOnSignup,
		grantOnFirstBind: SettingKeyAuthSourceDefaultEmailGrantOnFirstBind,
		platformQuotas:   SettingKeyAuthSourcePlatformQuotas("email"),
	}
	linuxDoAuthSourceDefaultKeys = authSourceDefaultKeySet{
		source: "linuxdo", balance: SettingKeyAuthSourceDefaultLinuxDoBalance,
		concurrency:      SettingKeyAuthSourceDefaultLinuxDoConcurrency,
		subscriptions:    SettingKeyAuthSourceDefaultLinuxDoSubscriptions,
		grantOnSignup:    SettingKeyAuthSourceDefaultLinuxDoGrantOnSignup,
		grantOnFirstBind: SettingKeyAuthSourceDefaultLinuxDoGrantOnFirstBind,
		platformQuotas:   SettingKeyAuthSourcePlatformQuotas("linuxdo"),
	}
	oidcAuthSourceDefaultKeys = authSourceDefaultKeySet{
		source: "oidc", balance: SettingKeyAuthSourceDefaultOIDCBalance,
		concurrency:      SettingKeyAuthSourceDefaultOIDCConcurrency,
		subscriptions:    SettingKeyAuthSourceDefaultOIDCSubscriptions,
		grantOnSignup:    SettingKeyAuthSourceDefaultOIDCGrantOnSignup,
		grantOnFirstBind: SettingKeyAuthSourceDefaultOIDCGrantOnFirstBind,
		platformQuotas:   SettingKeyAuthSourcePlatformQuotas("oidc"),
	}
	weChatAuthSourceDefaultKeys = authSourceDefaultKeySet{
		source: "wechat", balance: SettingKeyAuthSourceDefaultWeChatBalance,
		concurrency:      SettingKeyAuthSourceDefaultWeChatConcurrency,
		subscriptions:    SettingKeyAuthSourceDefaultWeChatSubscriptions,
		grantOnSignup:    SettingKeyAuthSourceDefaultWeChatGrantOnSignup,
		grantOnFirstBind: SettingKeyAuthSourceDefaultWeChatGrantOnFirstBind,
		platformQuotas:   SettingKeyAuthSourcePlatformQuotas("wechat"),
	}
	gitHubAuthSourceDefaultKeys = authSourceDefaultKeySet{
		source: "github", balance: SettingKeyAuthSourceDefaultGitHubBalance,
		concurrency:      SettingKeyAuthSourceDefaultGitHubConcurrency,
		subscriptions:    SettingKeyAuthSourceDefaultGitHubSubscriptions,
		grantOnSignup:    SettingKeyAuthSourceDefaultGitHubGrantOnSignup,
		grantOnFirstBind: SettingKeyAuthSourceDefaultGitHubGrantOnFirstBind,
		platformQuotas:   SettingKeyAuthSourcePlatformQuotas("github"),
	}
	googleAuthSourceDefaultKeys = authSourceDefaultKeySet{
		source: "google", balance: SettingKeyAuthSourceDefaultGoogleBalance,
		concurrency:      SettingKeyAuthSourceDefaultGoogleConcurrency,
		subscriptions:    SettingKeyAuthSourceDefaultGoogleSubscriptions,
		grantOnSignup:    SettingKeyAuthSourceDefaultGoogleGrantOnSignup,
		grantOnFirstBind: SettingKeyAuthSourceDefaultGoogleGrantOnFirstBind,
		platformQuotas:   SettingKeyAuthSourcePlatformQuotas("google"),
	}
	dingTalkAuthSourceDefaultKeys = authSourceDefaultKeySet{
		source: "dingtalk", balance: SettingKeyAuthSourceDefaultDingTalkBalance,
		concurrency:      SettingKeyAuthSourceDefaultDingTalkConcurrency,
		subscriptions:    SettingKeyAuthSourceDefaultDingTalkSubscriptions,
		grantOnSignup:    SettingKeyAuthSourceDefaultDingTalkGrantOnSignup,
		grantOnFirstBind: SettingKeyAuthSourceDefaultDingTalkGrantOnFirstBind,
		platformQuotas:   SettingKeyAuthSourcePlatformQuotas("dingtalk"),
	}
)

func DefaultRateLimit429CooldownSettings() *RateLimit429CooldownSettings {
	return &RateLimit429CooldownSettings{Enabled: true, CooldownSeconds: 5}
}

func DefaultOpenAIFastPolicySettings() *OpenAIFastPolicySettings {
	return &OpenAIFastPolicySettings{Rules: []OpenAIFastPolicyRule{}}
}
