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

func testAvailableChannelTextCatalogConfig() *config.Config {
	return &config.Config{
		Log: config.LogConfig{Environment: "staging"},
		Workspace: config.WorkspaceConfig{
			TextProvider: config.WorkspaceTextProviderConfig{
				TestProviderLabel: "deepseek-staging",
				BetaAllowlist: config.WorkspaceTextProviderBetaConfig{
					Enabled:       true,
					AllowedModels: []string{"deepseek-v4-flash"},
				},
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

func TestToUserSupportedModelsAddsGeminiImageMetadataForAllowedRealChannelModel(t *testing.T) {
	price := 0.03
	cfg := testAvailableChannelRealImageCatalogConfig()
	cfg.Workspace.ImageRealProvider.AllowedModels = []string{"gemini-2.5-flash-image"}
	settingService := service.NewSettingService(nil, cfg)

	out := toUserSupportedModels([]service.SupportedModel{{
		Name:     "gemini-2.5-flash-image",
		Platform: "gemini",
		Pricing: &service.ChannelModelPricing{
			BillingMode:      service.BillingModeImage,
			ImageOutputPrice: &price,
		},
	}}, map[string]struct{}{"gemini": {}}, settingService, 1)

	require.Len(t, out, 1)
	model := out[0]
	require.Equal(t, "gemini-2.5-flash-image", model.Name)
	require.Equal(t, "workspace-openai-compatible-image-staging", model.ProviderLabel)
	require.Equal(t, service.WorkspaceImageRealModelProvider, model.Provider)
	require.Equal(t, []string{"image_generation"}, model.Capabilities)
	require.Equal(t, service.WorkspaceImageRealModelCapabilitySource, model.CapabilitySource)
	require.Equal(t, service.WorkspaceModelCatalogSourceRealChannel, model.ModelCatalogSource)
	require.Equal(t, service.WorkspaceSelectedModelPricingConfigured, model.PricingStatus)
	require.Contains(t, model.UsageSupport, "image_count")
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
	require.Contains(t, out[0].Capabilities, "text_chat")
	require.Equal(t, service.WorkspaceModelCatalogSourceRealChannel, out[0].ModelCatalogSource)
	require.Equal(t, service.WorkspaceSelectedModelPricingConfigured, out[0].PricingStatus)
	require.Contains(t, out[0].UsageSupport, "token")
}

func TestToUserSupportedModelsOnlyMarksBetaAllowlistedTextModelsSelectable(t *testing.T) {
	settingService := service.NewSettingService(nil, testAvailableChannelTextCatalogConfig())

	out := toUserSupportedModels([]service.SupportedModel{
		{
			Name:     "deepseek-v4-flash",
			Platform: "openai",
			Pricing:  &service.ChannelModelPricing{BillingMode: service.BillingModeToken},
		},
		{
			Name:     "gpt-5.5",
			Platform: "openai",
			Pricing:  &service.ChannelModelPricing{BillingMode: service.BillingModeToken},
		},
	}, map[string]struct{}{"openai": {}}, settingService, 1)

	require.Len(t, out, 2)
	require.Equal(t, "deepseek-v4-flash", out[0].Name)
	require.Contains(t, out[0].Capabilities, "text_chat")
	require.Equal(t, service.WorkspaceModelCatalogSourceRealChannel, out[0].ModelCatalogSource)
	require.Equal(t, "gpt-5.5", out[1].Name)
	require.NotContains(t, out[1].Capabilities, "text_chat")
	require.Empty(t, out[1].ModelCatalogSource)
}

func TestBuildPlatformSectionsDeepSeekOnlyDoesNotExposeSyntheticImageModels(t *testing.T) {
	price := 0.01
	settingService := service.NewSettingService(nil, testAvailableChannelRealImageCatalogConfig())

	sections := buildPlatformSections(service.AvailableChannel{
		Name: "DeepSeek Staging",
		Groups: []service.AvailableGroupRef{
			{ID: 10, Name: "default", Platform: "openai"},
		},
		SupportedModels: []service.SupportedModel{
			{
				Name:     "deepseek-v4-flash",
				Platform: "openai",
				Pricing: &service.ChannelModelPricing{
					BillingMode: service.BillingModeToken,
					InputPrice:  &price,
					OutputPrice: &price,
				},
			},
			{
				Name:     "deepseek-chat",
				Platform: "openai",
				Pricing: &service.ChannelModelPricing{
					BillingMode: service.BillingModeToken,
					InputPrice:  &price,
					OutputPrice: &price,
				},
			},
		},
	}, []userAvailableGroup{{ID: 10, Name: "default", Platform: "openai"}}, settingService, 1)

	require.Len(t, sections, 1)
	modelNames := availableChannelTestModelNames(sections[0].SupportedModels)
	require.ElementsMatch(t, []string{"deepseek-v4-flash", "deepseek-chat"}, modelNames)
	require.NotContains(t, modelNames, "gpt-image-1")
	require.NotContains(t, modelNames, "gpt-image-2")
	require.NotContains(t, modelNames, "claude-3-5-sonnet")
	require.NotContains(t, modelNames, "gemini-2.5-pro")

	for _, model := range sections[0].SupportedModels {
		require.Equal(t, service.WorkspaceModelCatalogSourceRealChannel, model.ModelCatalogSource)
		require.Contains(t, model.Capabilities, "text_chat")
		require.NotContains(t, model.Capabilities, "image_generation")
	}
}

func TestToUserSupportedModelsDoesNotCreateStaticImageModelFromCapabilityHelper(t *testing.T) {
	metadata := service.ResolveWorkspaceModelCapabilities("gpt-image-1", service.WorkspaceModelCapabilityHints{})
	require.Contains(t, metadata.Capabilities, service.WorkspaceModelCapabilityImageGeneration)

	out := toUserSupportedModels([]service.SupportedModel{{
		Name:     "deepseek-v4-flash",
		Platform: "openai",
		Pricing:  &service.ChannelModelPricing{BillingMode: service.BillingModeToken},
	}}, map[string]struct{}{"openai": {}}, service.NewSettingService(nil, testAvailableChannelRealImageCatalogConfig()), 1)

	require.Len(t, out, 1)
	require.Equal(t, "deepseek-v4-flash", out[0].Name)
	require.NotEqual(t, "gpt-image-1", out[0].Name)
	require.NotContains(t, availableChannelTestModelNames(out), "gpt-image-1")
	require.NotContains(t, out[0].Capabilities, "image_generation")
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
	require.Equal(t, service.WorkspaceSelectedModelPricingMissing, out[0].PricingStatus)
}

func TestToUserSupportedModelsAddsRealChannelTextMetadata(t *testing.T) {
	out := toUserSupportedModels([]service.SupportedModel{{
		Name:     "claude-3-5-sonnet",
		Platform: "anthropic",
		Pricing:  &service.ChannelModelPricing{BillingMode: service.BillingModeToken},
	}}, map[string]struct{}{"anthropic": {}}, nil, 1)

	require.Len(t, out, 1)
	model := out[0]
	require.Equal(t, "claude-3-5-sonnet", model.Name)
	require.Equal(t, "anthropic", model.Platform)
	require.Equal(t, "anthropic", model.Provider)
	require.Equal(t, service.WorkspaceModelCatalogSourceRealChannel, model.ModelCatalogSource)
	require.Contains(t, model.Capabilities, "text_chat")
	require.Contains(t, model.Capabilities, "vision")
	require.Equal(t, service.WorkspaceSelectedModelPricingConfigured, model.PricingStatus)
	require.Contains(t, model.UsageSupport, "token")
}

func availableChannelTestModelNames(models []userSupportedModel) []string {
	names := make([]string, 0, len(models))
	for _, model := range models {
		names = append(names, model.Name)
	}
	return names
}
