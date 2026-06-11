//go:build unit

package service

import (
	"context"
	"strings"
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
			SettingKeyRegistrationEmailSuffixWhitelist: `["@EXAMPLE.com"," @foo.bar ","@invalid_domain",""]`,
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.Equal(t, []string{"@example.com", "@foo.bar"}, settings.RegistrationEmailSuffixWhitelist)
}

func TestSettingService_GetPublicSettings_SanitizesHomeContent(t *testing.T) {
	repo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeyHomeContent: `<div onclick="steal()">hello</div><script>alert(1)</script><a href="javascript:alert(1)">x</a>`,
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.NotContains(t, settings.HomeContent, "<script")
	require.NotContains(t, settings.HomeContent, "onclick=")
	require.NotContains(t, strings.ToLower(settings.HomeContent), "javascript:")
	require.Contains(t, settings.HomeContent, "hello")
}

func TestSettingService_GetPublicSettings_AllowsHTTPEmbeddedURL(t *testing.T) {
	repo := &settingPublicRepoStub{
		values: map[string]string{
			SettingKeyPurchaseSubscriptionEnabled: "true",
			SettingKeyPurchaseSubscriptionURL:     "http://pay.example.com/checkout",
		},
	}
	svc := NewSettingService(repo, &config.Config{})

	settings, err := svc.GetPublicSettings(context.Background())
	require.NoError(t, err)
	require.True(t, settings.PurchaseSubscriptionEnabled)
	require.Equal(t, "http://pay.example.com/checkout", settings.PurchaseSubscriptionURL)
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
