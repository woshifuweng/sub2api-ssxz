package service

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

const (
	WorkspaceImageFakeModelExposureSource = "workspace_image_fake_execution_gate"
	WorkspaceImageFakeModelPlatform       = WorkspaceImageProviderFakeLabel

	WorkspaceImageRealModelExposureSource = "workspace_image_real_provider_gate"
	WorkspaceImageRealModelPlatform       = "OpenAI-compatible Images"
	WorkspaceImageRealModelProvider       = "openai-compatible-images"
)

type WorkspaceImageFakeModelExposure struct {
	Enabled          bool
	Model            string
	ProviderLabel    string
	Platform         string
	Capabilities     []WorkspaceModelCapability
	CapabilitySource string
	Fake             bool
	TestOnly         bool
}

type WorkspaceImageAvailableModelExposure struct {
	Model            string
	ProviderLabel    string
	Provider         string
	Platform         string
	Capabilities     []WorkspaceModelCapability
	CapabilitySource string
	Fake             bool
	TestOnly         bool
	StagingOnly      bool
}

type WorkspaceImageRealModelExposure struct {
	Enabled bool
	Models  []WorkspaceImageAvailableModelExposure
}

func (s *SettingService) GetWorkspaceImageFakeModelExposure(userID int64) WorkspaceImageFakeModelExposure {
	if s == nil {
		return WorkspaceImageFakeModelExposure{}
	}
	return workspaceImageFakeModelExposureFromConfig(s.cfg, userID)
}

func (s *SettingService) GetWorkspaceImageRealModelExposure(userID int64) WorkspaceImageRealModelExposure {
	if s == nil {
		return WorkspaceImageRealModelExposure{}
	}
	return workspaceImageRealModelExposureFromConfig(s.cfg, userID)
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
		Enabled:          true,
		Model:            WorkspaceImageProviderFakeModel,
		ProviderLabel:    WorkspaceImageProviderFakeLabel,
		Platform:         WorkspaceImageFakeModelPlatform,
		Capabilities:     []WorkspaceModelCapability{WorkspaceModelCapabilityImageGeneration},
		CapabilitySource: WorkspaceImageFakeModelExposureSource,
		Fake:             true,
		TestOnly:         true,
	}
}

func workspaceImageRealModelExposureFromConfig(cfg *config.Config, userID int64) WorkspaceImageRealModelExposure {
	if cfg == nil || userID <= 0 {
		return WorkspaceImageRealModelExposure{}
	}
	realConfig := cfg.Workspace.ImageRealProvider
	if !realConfig.Enabled {
		return WorkspaceImageRealModelExposure{}
	}
	if realConfig.MaxRequestsPerTestRun <= 0 {
		return WorkspaceImageRealModelExposure{}
	}
	if realConfig.ProviderLabel == "" {
		return WorkspaceImageRealModelExposure{}
	}
	if realConfig.StagingOnly && !workspaceImageRealExposureNonProduction(cfg) {
		return WorkspaceImageRealModelExposure{}
	}
	if !workspaceImageFakeExposureInt64Contains(realConfig.AllowedUserIDs, userID) {
		return WorkspaceImageRealModelExposure{}
	}
	if !workspaceImageFakeExposureStringContains(realConfig.AllowedProviderLabels, realConfig.ProviderLabel) {
		return WorkspaceImageRealModelExposure{}
	}

	models := make([]WorkspaceImageAvailableModelExposure, 0, len(realConfig.AllowedModels))
	for _, allowedModel := range realConfig.AllowedModels {
		allowedModel = strings.TrimSpace(allowedModel)
		if allowedModel == "" {
			continue
		}
		metadata := ResolveWorkspaceModelCapabilities(allowedModel, WorkspaceModelCapabilityHints{
			ProviderLabel: realConfig.ProviderLabel,
			Provider:      WorkspaceImageRealModelProvider,
			Platform:      WorkspaceImageRealModelPlatform,
		})
		if !workspaceModelCapabilityListContains(metadata.Capabilities, WorkspaceModelCapabilityImageGeneration) {
			continue
		}
		models = append(models, WorkspaceImageAvailableModelExposure{
			Model:            allowedModel,
			ProviderLabel:    realConfig.ProviderLabel,
			Provider:         WorkspaceImageRealModelProvider,
			Platform:         WorkspaceImageRealModelPlatform,
			Capabilities:     metadata.Capabilities,
			CapabilitySource: WorkspaceImageRealModelExposureSource,
			StagingOnly:      realConfig.StagingOnly,
		})
	}
	if len(models) == 0 {
		return WorkspaceImageRealModelExposure{}
	}
	return WorkspaceImageRealModelExposure{
		Enabled: true,
		Models:  models,
	}
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
