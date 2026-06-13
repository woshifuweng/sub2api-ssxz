package handler

import (
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func testAvailableChannelRealImageCatalogConfig() *config.Config {
	return &config.Config{
		Log: config.LogConfig{Environment: "production"},
		Workspace: config.WorkspaceConfig{
			ImageRealProvider: config.WorkspaceImageRealProviderConfig{
				Enabled:               true,
				KillSwitch:            true,
				StagingOnly:           true,
				Environment:           "staging",
				ProviderLabel:         "workspace-openai-compatible-image-staging",
				AllowedUserIDs:        []int64{1},
				AllowedModels:         []string{"gpt-image-1"},
				AllowedProviderLabels: []string{"workspace-openai-compatible-image-staging"},
				MaxRequestsPerTestRun: 1,
			},
		},
	}
}

func TestAppendWorkspaceImageFakeModelChannelNoopWhenDisabled(t *testing.T) {
	channels := []userAvailableChannel{{Name: "existing"}}

	out := appendWorkspaceImageFakeModelChannel(channels, service.WorkspaceImageFakeModelExposure{})

	require.Equal(t, channels, out)
}

func TestAppendWorkspaceImageFakeModelChannelAddsSafeFakeMetadata(t *testing.T) {
	out := appendWorkspaceImageFakeModelChannel(nil, service.WorkspaceImageFakeModelExposure{
		Enabled:            true,
		Model:              service.WorkspaceImageProviderFakeModel,
		ProviderLabel:      service.WorkspaceImageProviderFakeLabel,
		Platform:           service.WorkspaceImageFakeModelPlatform,
		Capabilities:       []service.WorkspaceModelCapability{service.WorkspaceModelCapabilityImageGeneration},
		CapabilitySource:   service.WorkspaceImageFakeModelExposureSource,
		ModelCatalogSource: service.WorkspaceModelCatalogSourceFakeGate,
		Fake:               true,
		TestOnly:           true,
	})

	require.Len(t, out, 1)
	require.Equal(t, "Workspace Image Fake", out[0].Name)
	require.Len(t, out[0].Platforms, 1)
	require.Equal(t, service.WorkspaceImageProviderFakeLabel, out[0].Platforms[0].Platform)
	require.Len(t, out[0].Platforms[0].SupportedModels, 1)

	model := out[0].Platforms[0].SupportedModels[0]
	require.Equal(t, service.WorkspaceImageProviderFakeModel, model.Name)
	require.Equal(t, service.WorkspaceImageProviderFakeLabel, model.ProviderLabel)
	require.Equal(t, []string{"image_generation"}, model.Capabilities)
	require.Equal(t, service.WorkspaceModelCatalogSourceFakeGate, model.ModelCatalogSource)
	require.True(t, model.Fake)
	require.True(t, model.TestOnly)

	encoded, err := json.Marshal(out)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "Authorization")
	require.NotContains(t, string(encoded), "token")
	require.NotContains(t, string(encoded), "cookie")
	require.NotContains(t, string(encoded), "secret")
}

func TestToUserSupportedModelsAddsRealChannelImageMetadataOnlyForRealSupportedModel(t *testing.T) {
	price := 0.02
	settingService := service.NewSettingService(nil, testAvailableChannelRealImageCatalogConfig())

	out := toUserSupportedModels([]service.SupportedModel{{
		Name:     "gpt-image-1",
		Platform: "openai",
		Pricing: &service.ChannelModelPricing{
			BillingMode:      service.BillingModeImage,
			ImageOutputPrice: &price,
		},
	}}, map[string]struct{}{"openai": {}}, settingService, 1)

	require.Len(t, out, 1)
	model := out[0]
	require.Equal(t, "gpt-image-1", model.Name)
	require.Equal(t, "workspace-openai-compatible-image-staging", model.ProviderLabel)
	require.Equal(t, service.WorkspaceImageRealModelProvider, model.Provider)
	require.Equal(t, []string{"image_generation"}, model.Capabilities)
	require.Equal(t, service.WorkspaceImageRealModelCapabilitySource, model.CapabilitySource)
	require.Equal(t, service.WorkspaceModelCatalogSourceRealChannel, model.ModelCatalogSource)
	require.False(t, model.Fake)
	require.False(t, model.TestOnly)
	require.True(t, model.StagingOnly)

	encoded, err := json.Marshal(out)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "Authorization")
	require.NotContains(t, string(encoded), "token")
	require.NotContains(t, string(encoded), "cookie")
	require.NotContains(t, string(encoded), "secret")
}

func TestToUserSupportedModelsDoesNotCreateRealImageModelFromEnvGate(t *testing.T) {
	settingService := service.NewSettingService(nil, testAvailableChannelRealImageCatalogConfig())

	out := toUserSupportedModels([]service.SupportedModel{{
		Name:     "gpt-5.5",
		Platform: "openai",
		Pricing:  &service.ChannelModelPricing{BillingMode: service.BillingModeToken},
	}}, map[string]struct{}{"openai": {}}, settingService, 1)

	require.Len(t, out, 1)
	require.Equal(t, "gpt-5.5", out[0].Name)
	require.Empty(t, out[0].Capabilities)
	require.Empty(t, out[0].ModelCatalogSource)
}

func TestToUserSupportedModelsRequiresRealImagePricing(t *testing.T) {
	settingService := service.NewSettingService(nil, testAvailableChannelRealImageCatalogConfig())

	out := toUserSupportedModels([]service.SupportedModel{{
		Name:     "gpt-image-1",
		Platform: "openai",
		Pricing:  nil,
	}}, map[string]struct{}{"openai": {}}, settingService, 1)

	require.Len(t, out, 1)
	require.Empty(t, out[0].Capabilities)
	require.Empty(t, out[0].ModelCatalogSource)
}
