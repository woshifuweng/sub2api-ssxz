package service

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

const (
	WorkspaceImageFakeModelExposureSource = "workspace_image_fake_execution_gate"
	WorkspaceImageFakeModelPlatform       = WorkspaceImageProviderFakeLabel

	WorkspaceImageRealModelCapabilitySource = "workspace_image_real_channel"
	WorkspaceImageRealModelProvider         = "openai-compatible-images"

	WorkspaceModelCatalogSourceRealChannel    = "real_channel"
	WorkspaceModelCatalogSourceFakeGate       = "fake_gate"
	WorkspaceModelCatalogSourceEnvGate        = "env_gate"
	WorkspaceModelCatalogSourceStaticFallback = "static_fallback"
	WorkspaceModelCatalogSourceUnknown        = "unknown"
)

type WorkspaceImageFakeModelExposure struct {
	Enabled            bool
	Model              string
	ProviderLabel      string
	Platform           string
	Capabilities       []WorkspaceModelCapability
	CapabilitySource   string
	ModelCatalogSource string
	Fake               bool
	TestOnly           bool
}

type WorkspaceImageAvailableModelExposure struct {
	Model              string
	ProviderLabel      string
	Provider           string
	Platform           string
	Capabilities       []WorkspaceModelCapability
	CapabilitySource   string
	ModelCatalogSource string
	Fake               bool
	TestOnly           bool
	StagingOnly        bool
}

type WorkspaceTextAvailableModelExposure struct {
	Model              string
	ProviderLabel      string
	Provider           string
	Platform           string
	Capabilities       []WorkspaceModelCapability
	CapabilitySource   string
	ModelCatalogSource string
}

func (s *SettingService) GetWorkspaceImageFakeModelExposure(userID int64) WorkspaceImageFakeModelExposure {
	if s == nil {
		return WorkspaceImageFakeModelExposure{}
	}
	return workspaceImageFakeModelExposureFromConfig(s.cfg, userID)
}

func (s *SettingService) GetWorkspaceImageRealChannelModelExposure(userID int64, model SupportedModel) WorkspaceImageAvailableModelExposure {
	if s == nil {
		return WorkspaceImageAvailableModelExposure{}
	}
	return workspaceImageRealChannelModelExposureFromConfig(s.cfg, userID, model)
}

func (s *SettingService) GetWorkspaceTextRealChannelModelExposure(userID int64, model SupportedModel) WorkspaceTextAvailableModelExposure {
	if s == nil {
		return WorkspaceTextAvailableModelExposure{}
	}
	return workspaceTextRealChannelModelExposureFromConfig(s.cfg, userID, model)
}

func workspaceImageFakeModelExposureFromConfig(cfg *config.Config, userID int64) WorkspaceImageFakeModelExposure {
	if cfg == nil || userID <= 0 {
		return WorkspaceImageFakeModelExposure{}
	}
	imageConfig := cfg.Workspace.ImageExecution
	if !imageConfig.Enabled || !imageConfig.FakeProviderEnabled {
		return WorkspaceImageFakeModelExposure{}
	}
	if imageConfig.MaxRequestsPerTestRun <= 0 {
		return WorkspaceImageFakeModelExposure{}
	}
	if !workspaceImageFakeExposureNonProduction(cfg) {
		return WorkspaceImageFakeModelExposure{}
	}
	if !workspaceImageFakeExposureInt64Contains(imageConfig.AllowedUserIDs, userID) {
		return WorkspaceImageFakeModelExposure{}
	}
	if !workspaceImageFakeExposureStringContains(imageConfig.AllowedModels, WorkspaceImageProviderFakeModel) {
		return WorkspaceImageFakeModelExposure{}
	}
	if !workspaceImageFakeExposureStringContains(imageConfig.AllowedProviderLabels, WorkspaceImageProviderFakeLabel) {
		return WorkspaceImageFakeModelExposure{}
	}
	return WorkspaceImageFakeModelExposure{
		Enabled:            true,
		Model:              WorkspaceImageProviderFakeModel,
		ProviderLabel:      WorkspaceImageProviderFakeLabel,
		Platform:           WorkspaceImageFakeModelPlatform,
		Capabilities:       []WorkspaceModelCapability{WorkspaceModelCapabilityImageGeneration},
		CapabilitySource:   WorkspaceImageFakeModelExposureSource,
		ModelCatalogSource: WorkspaceModelCatalogSourceFakeGate,
		Fake:               true,
		TestOnly:           true,
	}
}

func workspaceImageRealChannelModelExposureFromConfig(cfg *config.Config, userID int64, model SupportedModel) WorkspaceImageAvailableModelExposure {
	if cfg == nil || userID <= 0 {
		return WorkspaceImageAvailableModelExposure{}
	}
	realConfig := cfg.Workspace.ImageRealProvider
	if !realConfig.Enabled {
		return WorkspaceImageAvailableModelExposure{}
	}
	if realConfig.MaxRequestsPerTestRun <= 0 {
		return WorkspaceImageAvailableModelExposure{}
	}
	if realConfig.ProviderLabel == "" {
		return WorkspaceImageAvailableModelExposure{}
	}
	if realConfig.StagingOnly && !workspaceImageRealExposureNonProduction(cfg) {
		return WorkspaceImageAvailableModelExposure{}
	}
	if !workspaceImageFakeExposureInt64Contains(realConfig.AllowedUserIDs, userID) {
		return WorkspaceImageAvailableModelExposure{}
	}
	if !workspaceImageFakeExposureStringContains(realConfig.AllowedProviderLabels, realConfig.ProviderLabel) {
		return WorkspaceImageAvailableModelExposure{}
	}
	if !workspaceImageFakeExposureStringContains(realConfig.AllowedModels, model.Name) {
		return WorkspaceImageAvailableModelExposure{}
	}
	if !workspaceRealImageModelPricingConfigured(model.Pricing) {
		return WorkspaceImageAvailableModelExposure{}
	}
	metadata := ResolveWorkspaceModelCapabilities(model.Name, WorkspaceModelCapabilityHints{
		ProviderLabel: realConfig.ProviderLabel,
		Provider:      WorkspaceImageRealModelProvider,
		Platform:      model.Platform,
	})
	if !workspaceModelCapabilityListContains(metadata.Capabilities, WorkspaceModelCapabilityImageGeneration) {
		return WorkspaceImageAvailableModelExposure{}
	}
	return WorkspaceImageAvailableModelExposure{
		Model:              strings.TrimSpace(model.Name),
		ProviderLabel:      realConfig.ProviderLabel,
		Provider:           WorkspaceImageRealModelProvider,
		Platform:           strings.TrimSpace(model.Platform),
		Capabilities:       metadata.Capabilities,
		CapabilitySource:   WorkspaceImageRealModelCapabilitySource,
		ModelCatalogSource: WorkspaceModelCatalogSourceRealChannel,
		StagingOnly:        realConfig.StagingOnly,
	}
}

func workspaceTextRealChannelModelExposureFromConfig(cfg *config.Config, userID int64, model SupportedModel) WorkspaceTextAvailableModelExposure {
	if cfg == nil || userID <= 0 {
		return WorkspaceTextAvailableModelExposure{}
	}
	textConfig := cfg.Workspace.TextProvider
	if textConfig.BetaAllowlist.Enabled && !workspaceImageFakeExposureStringContains(textConfig.BetaAllowlist.AllowedModels, model.Name) {
		return WorkspaceTextAvailableModelExposure{}
	}
	metadata := ResolveWorkspaceModelCapabilities(model.Name, WorkspaceModelCapabilityHints{
		ProviderLabel: textConfig.TestProviderLabel,
		Provider:      strings.TrimSpace(model.Platform),
		Platform:      strings.TrimSpace(model.Platform),
	})
	if !workspaceModelCapabilityListContains(metadata.Capabilities, WorkspaceModelCapabilityTextChat) {
		return WorkspaceTextAvailableModelExposure{}
	}
	if workspaceModelCapabilityListContains(metadata.Capabilities, WorkspaceModelCapabilityImageGeneration) {
		return WorkspaceTextAvailableModelExposure{}
	}
	return WorkspaceTextAvailableModelExposure{
		Model:              strings.TrimSpace(model.Name),
		ProviderLabel:      metadata.ProviderLabel,
		Provider:           metadata.Provider,
		Platform:           metadata.Platform,
		Capabilities:       metadata.Capabilities,
		CapabilitySource:   metadata.CapabilitySource,
		ModelCatalogSource: WorkspaceModelCatalogSourceRealChannel,
	}
}

func workspaceRealImageModelPricingConfigured(pricing *ChannelModelPricing) bool {
	if pricing == nil {
		return false
	}
	if pricing.BillingMode == BillingModeImage {
		return true
	}
	if pricing.ImageOutputPrice != nil || pricing.PerRequestPrice != nil {
		return true
	}
	return len(pricing.Intervals) > 0
}

func workspaceImageFakeExposureNonProduction(cfg *config.Config) bool {
	environment := strings.TrimSpace(cfg.Workspace.TextProvider.Environment)
	if environment == "" {
		environment = strings.TrimSpace(cfg.Log.Environment)
	}
	return isWorkspaceTextProviderNonProductionEnvironment(environment)
}

func workspaceImageRealExposureNonProduction(cfg *config.Config) bool {
	environment := strings.TrimSpace(cfg.Workspace.ImageRealProvider.Environment)
	if environment == "" {
		environment = strings.TrimSpace(cfg.Log.Environment)
	}
	return isWorkspaceTextProviderNonProductionEnvironment(environment)
}

func workspaceImageFakeExposureStringContains(values []string, target string) bool {
	target = strings.ToLower(strings.TrimSpace(target))
	if target == "" {
		return false
	}
	for _, value := range values {
		if strings.ToLower(strings.TrimSpace(value)) == target {
			return true
		}
	}
	return false
}

func workspaceImageFakeExposureInt64Contains(values []int64, target int64) bool {
	if target <= 0 {
		return false
	}
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
