package service

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWorkspaceModelCapabilityStaticRules(t *testing.T) {
	tests := []struct {
		name         string
		model        string
		capabilities []WorkspaceModelCapability
		notContains  WorkspaceModelCapability
	}{
		{
			name:         "deepseek text model",
			model:        "DeepSeek-V4-Flash",
			capabilities: []WorkspaceModelCapability{WorkspaceModelCapabilityTextChat},
			notContains:  WorkspaceModelCapabilityImageGeneration,
		},
		{
			name:         "gpt vision model",
			model:        "gpt-4o",
			capabilities: []WorkspaceModelCapability{WorkspaceModelCapabilityTextChat, WorkspaceModelCapabilityVision},
			notContains:  WorkspaceModelCapabilityImageGeneration,
		},
		{
			name:         "claude vision model",
			model:        "claude-3-5-sonnet",
			capabilities: []WorkspaceModelCapability{WorkspaceModelCapabilityTextChat, WorkspaceModelCapabilityVision},
			notContains:  WorkspaceModelCapabilityImageGeneration,
		},
		{
			name:         "gemini vision model",
			model:        "gemini-1.5-pro",
			capabilities: []WorkspaceModelCapability{WorkspaceModelCapabilityTextChat, WorkspaceModelCapabilityVision},
			notContains:  WorkspaceModelCapabilityImageGeneration,
		},
		{
			name:         "explicit image model",
			model:        "gpt-image-1",
			capabilities: []WorkspaceModelCapability{WorkspaceModelCapabilityImageGeneration},
			notContains:  WorkspaceModelCapabilityTextChat,
		},
		{
			name:         "unknown safe fallback",
			model:        "unknown-frontier-model",
			capabilities: []WorkspaceModelCapability{WorkspaceModelCapabilityTextChat},
			notContains:  WorkspaceModelCapabilityImageGeneration,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata := ResolveWorkspaceModelCapabilities(tt.model, WorkspaceModelCapabilityHints{})

			require.Equal(t, tt.model, metadata.ModelName)
			require.Equal(t, tt.capabilities, metadata.Capabilities)
			require.False(t, workspaceModelCapabilityListContains(metadata.Capabilities, tt.notContains))
			require.NotEmpty(t, metadata.CapabilitySource)
		})
	}
}

func TestWorkspaceModelCapabilityMatch(t *testing.T) {
	textOnly := ResolveWorkspaceModelCapabilities("deepseek-v4-flash", WorkspaceModelCapabilityHints{})

	matched, reason := WorkspaceModelCapabilitiesMatch(WorkspacePlannedCapabilityImageGeneration, textOnly)
	require.False(t, matched)
	require.Equal(t, "selected_model_does_not_support_image_generation", reason)

	matched, reason = WorkspaceModelCapabilitiesMatch(WorkspacePlannedCapabilityTextChat, textOnly)
	require.True(t, matched)
	require.Empty(t, reason)
}

func TestWorkspaceModelCapabilityUsesExplicitHints(t *testing.T) {
	metadata := ResolveWorkspaceModelCapabilities("custom-image-model", WorkspaceModelCapabilityHints{
		ProviderLabel: "custom-image-staging",
		Provider:      "custom",
		Platform:      "openai",
		Tags:          []string{"text", "vision", "image_generation", "function_calling"},
	})

	require.Equal(t, "custom-image-staging", metadata.ProviderLabel)
	require.Equal(t, "custom", metadata.Provider)
	require.Equal(t, "openai", metadata.Platform)
	require.Equal(t, workspaceModelCapabilitySourceExplicit, metadata.CapabilitySource)
	require.Contains(t, metadata.Capabilities, WorkspaceModelCapabilityTextChat)
	require.Contains(t, metadata.Capabilities, WorkspaceModelCapabilityVision)
	require.Contains(t, metadata.Capabilities, WorkspaceModelCapabilityImageGeneration)
	require.Contains(t, metadata.Capabilities, WorkspaceModelCapabilityFunctionCalling)
}

func TestWorkspaceModelCapabilityMetadataIsSafe(t *testing.T) {
	plan := WorkspaceCapabilityPlan{
		PlannedCapability:      WorkspacePlannedCapabilityImageGeneration,
		ModelCapabilityMatched: false,
		BlockReason:            "selected_model_does_not_support_image_generation",
	}
	modelMetadata := ResolveWorkspaceModelCapabilities("deepseek-v4-flash", WorkspaceModelCapabilityHints{})
	metadata := mergeWorkspaceModelCapabilityMetadata(map[string]any{"existing": "kept"}, modelMetadata, plan)

	require.Equal(t, "kept", metadata["existing"])
	require.Equal(t, []string{"text_chat"}, metadata["selected_model_capabilities"])
	require.Equal(t, "selected_model_does_not_support_image_generation", metadata["model_capability_mismatch_reason"])

	raw, err := json.Marshal(metadata)
	require.NoError(t, err)
	serialized := strings.ToLower(string(raw))
	require.NotContains(t, serialized, "prompt")
	require.NotContains(t, serialized, "authorization")
	require.NotContains(t, serialized, "cookie")
	require.NotContains(t, serialized, "token")
	require.NotContains(t, serialized, "secret")
	require.NotContains(t, serialized, "api_key")
}
