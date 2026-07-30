//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type settingPublicRepoStub struct {
	values map[string]string
}

func (s *settingPublicRepoStub) Get(ctx context.Context, key string) (*Setting, error) {
	panic("unexpected Get call")
}

func (s *settingPublicRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	if value, ok := s.values[key]; ok {
		return value, nil
	}
	return "", ErrSettingNotFound
}

func (s *settingPublicRepoStub) Set(ctx context.Context, key, value string) error {
	panic("unexpected Set call")
}

func (s *settingPublicRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

func (s *settingPublicRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *settingPublicRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *settingPublicRepoStub) Delete(ctx context.Context, key string) error {
	panic("unexpected Delete call")
}

func TestSettingService_GetPublicSettings_ExposesRegistrationEmailSuffixWhitelist(t *testing.T) {
	repo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeyRegistrationEnabled:              "true",
			SettingKeyEmailVerifyEnabled:               "true",
			SettingKeyRegistrationEmailSuffixWhitelist: `["@EXAMPLE.com"," @foo.bar ","*.EDU.CN","@invalid_domain",""]`,
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.Equal(t, []string{"@example.com", "@foo.bar", "*.edu.cn"}, settings.RegistrationEmailSuffixWhitelist)
}

func TestSettingService_GetPublicSettings_ExposesTablePreferences(t *testing.T) {
	repo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeyTableDefaultPageSize: "50",
			SettingKeyTablePageSizeOptions: "[20,50,100]",
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.Equal(t, 50, settings.TableDefaultPageSize)
	require.Equal(t, []int{20, 50, 100}, settings.TablePageSizeOptions)
}

func TestSettingService_GetPublicSettings_ExposesCompactHomeEnabled(t *testing.T) {
	repo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeyCompactHomeEnabled: "true",
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())

	require.NoError(t, err)
	require.True(t, settings.CompactHomeEnabled)

	missingSettings, err := NewSettingService(&settingPublicRepoStub{values: map[string]string{}}, &config.Config{}).
		GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.False(t, missingSettings.CompactHomeEnabled)
}

func TestSettingService_GetPublicSettings_ExposesForceEmailOnThirdPartySignup(t *testing.T) {
	repo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeyForceEmailOnThirdPartySignup: "true",
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.True(t, settings.ForceEmailOnThirdPartySignup)
}

func TestSettingService_GetPublicSettings_ExposesAllowUserViewErrorRequests(t *testing.T) {
	repo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeyAllowUserViewErrorRequests: "true",
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.True(t, settings.AllowUserViewErrorRequests)
}

func TestSettingService_GetPublicSettings_ExposesWeChatOAuthModeCapabilities(t *testing.T) {
	svc := NewSettingService(&settingPublicRepoStub{
		values: map[string]string{
			SettingKeyWeChatConnectEnabled:             "true",
			SettingKeyWeChatConnectAppID:               "wx-mp-app",
			SettingKeyWeChatConnectAppSecret:           "wx-mp-secret",
			SettingKeyWeChatConnectMode:                "mp",
			SettingKeyWeChatConnectScopes:              "snsapi_base",
			SettingKeyWeChatConnectOpenEnabled:         "true",
			SettingKeyWeChatConnectMPEnabled:           "true",
			SettingKeyWeChatConnectRedirectURL:         "https://api.example.com/api/v1/auth/oauth/wechat/callback",
			SettingKeyWeChatConnectFrontendRedirectURL: "/auth/wechat/callback",
		},
	}, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.True(t, settings.WeChatOAuthEnabled)
	require.True(t, settings.WeChatOAuthOpenEnabled)
	require.True(t, settings.WeChatOAuthMPEnabled)
}

func TestSettingService_GetPublicSettings_DoesNotExposeMobileOnlyWeChatAsWebOAuthAvailable(t *testing.T) {
	svc := NewSettingService(&settingPublicRepoStub{
		values: map[string]string{
			SettingKeyWeChatConnectEnabled:             "true",
			SettingKeyWeChatConnectMobileEnabled:       "true",
			SettingKeyWeChatConnectMode:                "mobile",
			SettingKeyWeChatConnectMobileAppID:         "wx-mobile-app",
			SettingKeyWeChatConnectMobileAppSecret:     "wx-mobile-secret",
			SettingKeyWeChatConnectFrontendRedirectURL: "/auth/wechat/callback",
		},
	}, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.False(t, settings.WeChatOAuthEnabled)
	require.False(t, settings.WeChatOAuthOpenEnabled)
	require.False(t, settings.WeChatOAuthMPEnabled)
	require.True(t, settings.WeChatOAuthMobileEnabled)
}

func TestSettingService_GetPublicSettings_FallsBackToConfigForWeChatOAuthCapabilities(t *testing.T) {
	svc := NewSettingService(&settingPublicRepoStub{values: map[string]string{}}, &config.Config{
		WeChat: config.WeChatConnectConfig{
			Enabled:             true,
			OpenEnabled:         true,
			OpenAppID:           "wx-open-config",
			OpenAppSecret:       "wx-open-secret",
			FrontendRedirectURL: "/auth/wechat/config-callback",
		},
	})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.True(t, settings.WeChatOAuthEnabled)
	require.True(t, settings.WeChatOAuthOpenEnabled)
	require.False(t, settings.WeChatOAuthMPEnabled)
	require.False(t, settings.WeChatOAuthMobileEnabled)
}

func TestSettingService_AvailableChannelsDefaultUsesStoredSetting(t *testing.T) {
	repo := &settingPublicRepoStub{
		values: map[string]string{
			settingKeyAvailableChannelsEnabled: "false",
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	runtime := svc.GetAvailableChannelsRuntime(context.Background())
	require.False(t, runtime.Enabled)

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.False(t, settings.AvailableChannelsEnabled)
}

func TestSettingService_PublicSettingsExposeWebSearchUnavailableByDefault(t *testing.T) {
	repo := &settingPublicRepoStub{values: map[string]string{}}
	svc := NewSettingService(repo, &config.Config{
		Workspace: config.WorkspaceConfig{
			WebSearch: config.WorkspaceWebSearchConfig{
				Provider:   "jina",
				Enabled:    false,
				KillSwitch: true,
			},
		},
	})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.False(t, settings.WebSearch.Available)
	require.Equal(t, "jina", settings.WebSearch.Provider)
}

func TestSettingService_AvailableChannelsStagingOverrideRequiresNonProduction(t *testing.T) {
	repo := &settingPublicRepoStub{
		values: map[string]string{
			settingKeyAvailableChannelsEnabled: "false",
		},
	}
	svc := NewSettingService(repo, &config.Config{
		Log: config.LogConfig{Environment: "production"},
		Workspace: config.WorkspaceConfig{
			AvailableChannels: config.WorkspaceAvailableChannelsConfig{
				StagingOverrideEnabled: true,
			},
		},
	})

	runtime := svc.GetAvailableChannelsRuntime(context.Background())
	require.False(t, runtime.Enabled)

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.False(t, settings.AvailableChannelsEnabled)
}

func TestSettingService_AvailableChannelsStagingOverrideEnablesRuntimeAndPublicSettings(t *testing.T) {
	repo := &settingPublicRepoStub{
		values: map[string]string{
			settingKeyAvailableChannelsEnabled: "false",
		},
	}
	svc := NewSettingService(repo, &config.Config{
		Log: config.LogConfig{Environment: "production"},
		Workspace: config.WorkspaceConfig{
			TextProvider: config.WorkspaceTextProviderConfig{
				Environment: "staging",
			},
			AvailableChannels: config.WorkspaceAvailableChannelsConfig{
				StagingOverrideEnabled: true,
			},
		},
	})

	runtime := svc.GetAvailableChannelsRuntime(context.Background())
	require.True(t, runtime.Enabled)

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.True(t, settings.AvailableChannelsEnabled)
}

func TestSettingService_SoraClientStagingOverrideRequiresNonProduction(t *testing.T) {
	repo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeySoraClientEnabled: "false",
		},
	}
	svc := NewSettingService(repo, &config.Config{
		Log: config.LogConfig{Environment: "production"},
		Workspace: config.WorkspaceConfig{
			SoraClient: config.WorkspaceSoraClientConfig{
				StagingOverrideEnabled: true,
			},
		},
	})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.False(t, settings.SoraClientEnabled)
}

func TestSettingService_SoraClientStagingOverrideEnablesPublicSettings(t *testing.T) {
	repo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeySoraClientEnabled: "false",
		},
	}
	svc := NewSettingService(repo, &config.Config{
		Log: config.LogConfig{Environment: "production"},
		Workspace: config.WorkspaceConfig{
			TextProvider: config.WorkspaceTextProviderConfig{
				Environment: "staging",
			},
			SoraClient: config.WorkspaceSoraClientConfig{
				StagingOverrideEnabled: true,
			},
		},
	})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.True(t, settings.SoraClientEnabled)
}

func TestSettingService_PublicSettingsInjectionPreservesUnionAndSanitizesValues(t *testing.T) {
	repo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeyForceEmailOnThirdPartySignup: "true",
			SettingKeyHomeContent:                  `<section onclick="alert(1)"><script>alert(2)</script>safe</section>`,
			SettingKeyPurchaseSubscriptionURL:      "https://example.com/subscribe",
			SettingKeyPurchaseLinkCNY10:            "https://example.com/10",
			SettingKeyPurchaseLinkCNY30:            "javascript:alert(1)",
			SettingKeyPurchaseLinkCNY100:           "https://example.com/100",
			SettingKeySoraClientEnabled:            "true",
		},
	}
	svc := NewSettingService(repo, &config.Config{
		Workspace: config.WorkspaceConfig{
			WebSearch: config.WorkspaceWebSearchConfig{
				Enabled:  true,
				Provider: "jina",
			},
		},
	})

	publicSettings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.True(t, publicSettings.ForceEmailOnThirdPartySignup)
	require.Equal(t, "https://example.com/subscribe", publicSettings.PurchaseSubscriptionURL)
	require.Equal(t, "https://example.com/10", publicSettings.PurchaseLinkCNY10)
	require.Empty(t, publicSettings.PurchaseLinkCNY30)
	require.Equal(t, "https://example.com/100", publicSettings.PurchaseLinkCNY100)
	require.True(t, publicSettings.SoraClientEnabled)
	require.Equal(t, PublicWorkspaceWebSearchSettings{Available: true, Provider: "jina"}, publicSettings.WebSearch)
	require.NotContains(t, publicSettings.HomeContent, "<script")
	require.NotContains(t, publicSettings.HomeContent, "onclick")

	rawInjection, err := svc.GetPublicSettingsForInjection(context.Background())
	require.NoError(t, err)
	injection, ok := rawInjection.(*PublicSettingsInjectionPayload)
	require.True(t, ok, "unexpected injection payload type %T", rawInjection)
	require.Equal(t, publicSettings.ForceEmailOnThirdPartySignup, injection.ForceEmailOnThirdPartySignup)
	require.Equal(t, publicSettings.PurchaseSubscriptionURL, injection.PurchaseSubscriptionURL)
	require.Equal(t, publicSettings.PurchaseLinkCNY10, injection.PurchaseLinkCNY10)
	require.Equal(t, publicSettings.PurchaseLinkCNY30, injection.PurchaseLinkCNY30)
	require.Equal(t, publicSettings.PurchaseLinkCNY100, injection.PurchaseLinkCNY100)
	require.Equal(t, publicSettings.SoraClientEnabled, injection.SoraClientEnabled)
	require.Equal(t, publicSettings.WebSearch, injection.WebSearch)
	require.Equal(t, publicSettings.HomeContent, injection.HomeContent)
}
