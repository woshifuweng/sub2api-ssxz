package service

import (
	"strings"
)

const (
	settingKeyAvailableChannelsEnabled         = "available_channels_enabled"
	settingKeyChannelMonitorEnabled            = "channel_monitor_enabled"
	settingKeyWeChatConnectEnabled             = "wechat_connect_enabled"
	settingKeyWeChatConnectAppID               = "wechat_connect_app_id"
	settingKeyWeChatConnectAppSecret           = "wechat_connect_app_secret"
	settingKeyWeChatConnectOpenAppID           = "wechat_connect_open_app_id"
	settingKeyWeChatConnectOpenAppSecret       = "wechat_connect_open_app_secret"
	settingKeyWeChatConnectMPAppID             = "wechat_connect_mp_app_id"
	settingKeyWeChatConnectMPAppSecret         = "wechat_connect_mp_app_secret"
	settingKeyWeChatConnectMobileAppID         = "wechat_connect_mobile_app_id"
	settingKeyWeChatConnectMobileAppSecret     = "wechat_connect_mobile_app_secret"
	settingKeyWeChatConnectOpenEnabled         = "wechat_connect_open_enabled"
	settingKeyWeChatConnectMPEnabled           = "wechat_connect_mp_enabled"
	settingKeyWeChatConnectMobileEnabled       = "wechat_connect_mobile_enabled"
	settingKeyWeChatConnectMode                = "wechat_connect_mode"
	settingKeyWeChatConnectScopes              = "wechat_connect_scopes"
	settingKeyWeChatConnectRedirectURL         = "wechat_connect_redirect_url"
	settingKeyWeChatConnectFrontendRedirectURL = "wechat_connect_frontend_redirect_url"
	settingKeyOIDCConnectEnabled               = "oidc_connect_enabled"
	settingKeyOIDCConnectProviderName          = "oidc_connect_provider_name"
	settingKeyOIDCConnectClientID              = "oidc_connect_client_id"
	settingKeyOIDCConnectClientSecret          = "oidc_connect_client_secret"
	settingKeyOIDCConnectIssuerURL             = "oidc_connect_issuer_url"
	settingKeyOIDCConnectDiscoveryURL          = "oidc_connect_discovery_url"
	settingKeyOIDCConnectAuthorizeURL          = "oidc_connect_authorize_url"
	settingKeyOIDCConnectTokenURL              = "oidc_connect_token_url"
	settingKeyOIDCConnectUserInfoURL           = "oidc_connect_userinfo_url"
	settingKeyOIDCConnectJWKSURL               = "oidc_connect_jwks_url"
	settingKeyOIDCConnectScopes                = "oidc_connect_scopes"
	settingKeyOIDCConnectRedirectURL           = "oidc_connect_redirect_url"
	settingKeyOIDCConnectFrontendRedirectURL   = "oidc_connect_frontend_redirect_url"
	settingKeyOIDCConnectTokenAuthMethod       = "oidc_connect_token_auth_method"
	settingKeyOIDCConnectUsePKCE               = "oidc_connect_use_pkce"
	settingKeyOIDCConnectValidateIDToken       = "oidc_connect_validate_id_token"
	settingKeyOIDCConnectAllowedSigningAlgs    = "oidc_connect_allowed_signing_algs"
	settingKeyOIDCConnectClockSkewSeconds      = "oidc_connect_clock_skew_seconds"
	settingKeyOIDCConnectRequireEmailVerified  = "oidc_connect_require_email_verified"
	settingKeyOIDCConnectUserInfoEmailPath     = "oidc_connect_userinfo_email_path"
	settingKeyOIDCConnectUserInfoIDPath        = "oidc_connect_userinfo_id_path"
	settingKeyOIDCConnectUserInfoUsernamePath  = "oidc_connect_userinfo_username_path"
	defaultOIDCFrontendRedirect                = "/auth/oidc/callback"
	defaultOIDCAllowedSigningAlgs              = "RS256,ES256,PS256"
)

func (s *SettingService) getAvailableChannelsRuntimeOverride() (AvailableChannelsRuntime, bool) {
	if s == nil || s.cfg == nil || !s.cfg.Workspace.AvailableChannels.StagingOverrideEnabled {
		return AvailableChannelsRuntime{}, false
	}
	if !s.isNonProductionWorkspaceRuntime() {
		return AvailableChannelsRuntime{}, false
	}
	return AvailableChannelsRuntime{Enabled: true}, true
}

func (s *SettingService) getSoraClientRuntimeOverride() (bool, bool) {
	if s == nil || s.cfg == nil || !s.cfg.Workspace.SoraClient.StagingOverrideEnabled {
		return false, false
	}
	if !s.isNonProductionWorkspaceRuntime() {
		return false, false
	}
	return true, true
}

func (s *SettingService) isNonProductionWorkspaceRuntime() bool {
	environment := strings.TrimSpace(s.cfg.Workspace.TextProvider.Environment)
	if environment == "" {
		environment = strings.TrimSpace(s.cfg.Log.Environment)
	}
	return isWorkspaceTextProviderNonProductionEnvironment(environment)
}

func parseWeChatCapabilityFlag(raw string, fallback bool) bool {
	if strings.TrimSpace(raw) == "" {
		return fallback
	}
	return strings.TrimSpace(raw) == "true"
}

func normalizeWeChatStoredMode(openEnabled, mpEnabled, mobileEnabled bool, mode string) string {
	mode = normalizeWeChatConnectModeSetting(mode)
	switch mode {
	case "open":
		if openEnabled {
			return "open"
		}
	case "mp":
		if mpEnabled {
			return "mp"
		}
	case "mobile":
		if mobileEnabled {
			return "mobile"
		}
	}
	switch {
	case openEnabled:
		return "open"
	case mpEnabled:
		return "mp"
	case mobileEnabled:
		return "mobile"
	default:
		return mode
	}
}
