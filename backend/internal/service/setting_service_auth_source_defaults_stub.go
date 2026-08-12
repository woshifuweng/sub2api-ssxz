package service

type ProviderDefaultGrantSettings struct {
	Balance          float64
	Concurrency      int
	Subscriptions    []DefaultSubscriptionSetting
	GrantOnSignup    bool
	GrantOnFirstBind bool
	PlatformQuotas   map[string]*DefaultPlatformQuotaSetting
}

type AuthSourceDefaultSettings struct {
	Email                        ProviderDefaultGrantSettings
	LinuxDo                      ProviderDefaultGrantSettings
	OIDC                         ProviderDefaultGrantSettings
	WeChat                       ProviderDefaultGrantSettings
	GitHub                       ProviderDefaultGrantSettings
	Google                       ProviderDefaultGrantSettings
	DingTalk                     ProviderDefaultGrantSettings
	ForceEmailOnThirdPartySignup bool
}
