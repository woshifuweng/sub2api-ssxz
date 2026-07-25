package service

import (
	"context"
)

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

func (s *SettingService) ResolveAuthSourceGrantSettingsLegacy(context.Context, string, bool) (ProviderDefaultGrantSettings, bool, error) {
	// Local compatibility shim: until the full auth-source settings surface is migrated,
	// provider-specific grant defaults stay disabled and callers fall back to global defaults.
	return ProviderDefaultGrantSettings{}, false, nil
}

func (s *SettingService) GetAuthSourceDefaultSettingsLegacy(context.Context) (*AuthSourceDefaultSettings, error) {
	return &AuthSourceDefaultSettings{}, nil
}
