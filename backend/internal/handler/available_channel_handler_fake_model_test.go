package handler

import (
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestAppendWorkspaceImageFakeModelChannelNoopWhenDisabled(t *testing.T) {
	channels := []userAvailableChannel{{Name: "existing"}}

	out := appendWorkspaceImageFakeModelChannel(channels, service.WorkspaceImageFakeModelExposure{})

	require.Equal(t, channels, out)
}

func TestAppendWorkspaceImageFakeModelChannelAddsSafeFakeMetadata(t *testing.T) {
	out := appendWorkspaceImageFakeModelChannel(nil, service.WorkspaceImageFakeModelExposure{
		Enabled:          true,
		Model:            service.WorkspaceImageProviderFakeModel,
		ProviderLabel:    service.WorkspaceImageProviderFakeLabel,
		Platform:         service.WorkspaceImageFakeModelPlatform,
		Capabilities:     []service.WorkspaceModelCapability{service.WorkspaceModelCapabilityImageGeneration},
		CapabilitySource: service.WorkspaceImageFakeModelExposureSource,
		Fake:             true,
		TestOnly:         true,
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
	require.True(t, model.Fake)
	require.True(t, model.TestOnly)

	encoded, err := json.Marshal(out)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "Authorization")
	require.NotContains(t, string(encoded), "token")
	require.NotContains(t, string(encoded), "cookie")
	require.NotContains(t, string(encoded), "secret")
}
