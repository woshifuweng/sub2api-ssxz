package service

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func testWorkspaceImageFakeModelExposureConfig() *config.Config {
	return &config.Config{
		Log: config.LogConfig{Environment: "production"},
		Workspace: config.WorkspaceConfig{
			TextProvider: config.WorkspaceTextProviderConfig{Environment: "staging"},
			ImageExecution: config.WorkspaceImageExecutionConfig{
				Enabled:               true,
				KillSwitch:            true,
				FakeProviderEnabled:   true,
				AllowedUserIDs:        []int64{1},
				AllowedModels:         []string{WorkspaceImageProviderFakeModel},
				AllowedProviderLabels: []string{WorkspaceImageProviderFakeLabel},
				MaxRequestsPerTestRun: 3,
			},
		},
	}
}

func TestWorkspaceImageFakeModelExposureDefaultFailClosed(t *testing.T) {
	exposure := workspaceImageFakeModelExposureFromConfig(&config.Config{}, 1)
	require.False(t, exposure.Enabled)
}

func TestWorkspaceImageFakeModelExposureRequiresFakeProviderConfig(t *testing.T) {
	for name, mutate := range map[string]func(*config.Config){
		"fake_disabled": func(cfg *config.Config) {
			cfg.Workspace.ImageExecution.FakeProviderEnabled = false
		},
		"user_not_allowed": func(cfg *config.Config) {
			cfg.Workspace.ImageExecution.AllowedUserIDs = []int64{2}
		},
		"model_not_allowed": func(cfg *config.Config) {
			cfg.Workspace.ImageExecution.AllowedModels = []string{"other-image-model"}
		},
		"provider_not_allowed": func(cfg *config.Config) {
			cfg.Workspace.ImageExecution.AllowedProviderLabels = []string{"other-provider"}
		},
		"missing_cap": func(cfg *config.Config) {
			cfg.Workspace.ImageExecution.MaxRequestsPerTestRun = 0
		},
		"production_environment": func(cfg *config.Config) {
			cfg.Workspace.TextProvider.Environment = ""
			cfg.Log.Environment = "production"
		},
	} {
		t.Run(name, func(t *testing.T) {
			cfg := testWorkspaceImageFakeModelExposureConfig()
			mutate(cfg)

			exposure := workspaceImageFakeModelExposureFromConfig(cfg, 1)
			require.False(t, exposure.Enabled)
		})
	}
}

func TestWorkspaceImageFakeModelExposureAllowedUserReturnsMetadata(t *testing.T) {
	cfg := testWorkspaceImageFakeModelExposureConfig()
	cfg.Workspace.ImageExecution.KillSwitch = true

	exposure := workspaceImageFakeModelExposureFromConfig(cfg, 1)

	require.True(t, exposure.Enabled)
	require.Equal(t, WorkspaceImageProviderFakeModel, exposure.Model)
	require.Equal(t, WorkspaceImageProviderFakeLabel, exposure.ProviderLabel)
	require.Equal(t, []WorkspaceModelCapability{WorkspaceModelCapabilityImageGeneration}, exposure.Capabilities)
	require.Equal(t, WorkspaceImageFakeModelExposureSource, exposure.CapabilitySource)
	require.True(t, exposure.Fake)
	require.True(t, exposure.TestOnly)
}
