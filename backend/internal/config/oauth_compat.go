package config

// WeChatConnectConfig keeps the SSXZ WeChat compatibility fields that are
// referenced by the merged configuration surface.
type WeChatConnectConfig struct {
	Enabled             bool   `mapstructure:"enabled"`
	AppID               string `mapstructure:"app_id"`
	AppSecret           string `mapstructure:"app_secret"`
	OpenAppID           string `mapstructure:"open_app_id"`
	OpenAppSecret       string `mapstructure:"open_app_secret"`
	MPAppID             string `mapstructure:"mp_app_id"`
	MPAppSecret         string `mapstructure:"mp_app_secret"`
	MobileAppID         string `mapstructure:"mobile_app_id"`
	MobileAppSecret     string `mapstructure:"mobile_app_secret"`
	OpenEnabled         bool   `mapstructure:"open_enabled"`
	MPEnabled           bool   `mapstructure:"mp_enabled"`
	MobileEnabled       bool   `mapstructure:"mobile_enabled"`
	Mode                string `mapstructure:"mode"`
	Scopes              string `mapstructure:"scopes"`
	RedirectURL         string `mapstructure:"redirect_url"`
	FrontendRedirectURL string `mapstructure:"frontend_redirect_url"`
}

// OIDCConnectConfig keeps the SSXZ OIDC compatibility fields that are
// referenced by the merged configuration surface.
type OIDCConnectConfig struct {
	Enabled                 bool   `mapstructure:"enabled"`
	ProviderName            string `mapstructure:"provider_name"`
	ClientID                string `mapstructure:"client_id"`
	ClientSecret            string `mapstructure:"client_secret"`
	IssuerURL               string `mapstructure:"issuer_url"`
	DiscoveryURL            string `mapstructure:"discovery_url"`
	AuthorizeURL            string `mapstructure:"authorize_url"`
	TokenURL                string `mapstructure:"token_url"`
	UserInfoURL             string `mapstructure:"userinfo_url"`
	JWKSURL                 string `mapstructure:"jwks_url"`
	Scopes                  string `mapstructure:"scopes"`
	RedirectURL             string `mapstructure:"redirect_url"`
	FrontendRedirectURL     string `mapstructure:"frontend_redirect_url"`
	TokenAuthMethod         string `mapstructure:"token_auth_method"`
	UsePKCE                 bool   `mapstructure:"use_pkce"`
	ValidateIDToken         bool   `mapstructure:"validate_id_token"`
	UsePKCEExplicit         bool   `mapstructure:"-" yaml:"-"`
	ValidateIDTokenExplicit bool   `mapstructure:"-" yaml:"-"`
	AllowedSigningAlgs      string `mapstructure:"allowed_signing_algs"`
	ClockSkewSeconds        int    `mapstructure:"clock_skew_seconds"`
	RequireEmailVerified    bool   `mapstructure:"require_email_verified"`
	UserInfoEmailPath       string `mapstructure:"userinfo_email_path"`
	UserInfoIDPath          string `mapstructure:"userinfo_id_path"`
	UserInfoUsernamePath    string `mapstructure:"userinfo_username_path"`
}

// DingTalkConnectConfig is retained from the v0.1.171 base. Unlike WeChat and
// OIDC, the P branch had no compatibility type to restore during the merge.
type DingTalkConnectConfig struct {
	Enabled             bool   `mapstructure:"enabled"`
	ClientID            string `mapstructure:"client_id"`
	ClientSecret        string `mapstructure:"client_secret"`
	AuthorizeURL        string `mapstructure:"authorize_url"`
	TokenURL            string `mapstructure:"token_url"`
	UserInfoURL         string `mapstructure:"userinfo_url"`
	Scopes              string `mapstructure:"scopes"`
	RedirectURL         string `mapstructure:"redirect_url"`
	FrontendRedirectURL string `mapstructure:"frontend_redirect_url"`

	DingTalkAppKind string `mapstructure:"dingtalk_app_kind"`
	AppType         string `mapstructure:"app_type"`

	CorpRestrictionPolicy   string `mapstructure:"corp_restriction_policy"`
	InternalCorpID          string `mapstructure:"internal_corp_id"`
	BypassRegistration      bool   `mapstructure:"bypass_registration"`
	SyncCorpEmail           bool   `mapstructure:"sync_corp_email"`
	SyncDisplayName         bool   `mapstructure:"sync_display_name"`
	SyncDept                bool   `mapstructure:"sync_dept"`
	SyncCorpEmailAttrKey    string `mapstructure:"sync_corp_email_attr_key"`
	SyncDisplayNameAttrKey  string `mapstructure:"sync_display_name_attr_key"`
	SyncDeptAttrKey         string `mapstructure:"sync_dept_attr_key"`
	SyncCorpEmailAttrName   string `mapstructure:"sync_corp_email_attr_name"`
	SyncDisplayNameAttrName string `mapstructure:"sync_display_name_attr_name"`
	SyncDeptAttrName        string `mapstructure:"sync_dept_attr_name"`

	RequireEmail            bool   `mapstructure:"require_email"`
	UsernameOverwritePolicy string `mapstructure:"username_overwrite_policy"`

	UsernameAttributeKey         string   `mapstructure:"username_attribute_key"`
	EnableAttributeMatching      bool     `mapstructure:"enable_attribute_matching"`
	EnableAttributeSync          bool     `mapstructure:"enable_attribute_sync"`
	AttributeSyncFields          []string `mapstructure:"attribute_sync_fields"`
	AttributeSyncOverwritePolicy string   `mapstructure:"attribute_sync_overwrite_policy"`
}
