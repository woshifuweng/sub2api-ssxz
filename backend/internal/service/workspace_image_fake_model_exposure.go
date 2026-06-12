package service

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

const (
	WorkspaceImageFakeModelExposureSource = "workspace_image_fake_execution_gate"
	WorkspaceImageFakeModelPlatform       = WorkspaceImageProviderFakeLabel
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

func (s *SettingService) GetWorkspaceImageFakeModelExposure(userID int64) WorkspaceImageFakeModelExposure {
	if s == nil {
		return WorkspaceImageFakeModelExposure{}
	}
	return workspaceImageFakeModelExposureFromConfig(s.cfg, userID)
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

func workspaceImageFakeExposureNonProduction(cfg *config.Config) bool {
	environment := strings.TrimSpace(cfg.Workspace.TextProvider.Environment)
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
