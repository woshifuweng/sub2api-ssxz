package service

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWorkspaceCapabilityPlannerDetectsImageGeneration(t *testing.T) {
	planner := NewWorkspaceCapabilityPlanner()

	tests := []struct {
		name   string
		text   string
		reason string
	}{
		{
			name:   "zh product image",
			text:   "\u5e2e\u6211\u751f\u6210\u4e00\u5f20\u9ad8\u7ea7\u611f\u4ea7\u54c1\u56fe",
			reason: workspaceCapabilityReasonZHImageGenerationKeyword,
		},
		{
			name:   "zh cover image",
			text:   "\u753b\u4e00\u5f20\u5c0f\u7ea2\u4e66\u5c01\u9762",
			reason: workspaceCapabilityReasonZHImageGenerationKeyword,
		},
		{
			name:   "en image",
			text:   "generate image of perfume ad",
			reason: workspaceCapabilityReasonENImageGenerationKeyword,
		},
		{
			name:   "en poster",
			text:   "make a poster for a coffee shop",
			reason: workspaceCapabilityReasonENImageGenerationKeyword,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := planner.Plan(WorkspaceCapabilityPlannerInput{
				Text:          tt.text,
				SelectedModel: "gpt-5.5",
			})

			require.Equal(t, WorkspacePlannedCapabilityImageGeneration, plan.PlannedCapability)
			require.Equal(t, tt.reason, plan.Reason)
			require.Equal(t, WorkspaceCapabilityPlannerVersion, plan.PlannerVersion)
			require.Equal(t, "gpt-5.5", plan.SelectedModel)
			require.False(t, plan.ModelCapabilityMatched)
			require.Equal(t, workspaceCapabilityBlockModelCapabilityUnavailable, plan.BlockReason)
			require.Greater(t, plan.Confidence, 0.9)
		})
	}
}

func TestWorkspaceCapabilityPlannerDetectsTextChatAndUnknown(t *testing.T) {
	planner := NewWorkspaceCapabilityPlanner()

	for _, text := range []string{
		"\u4f60\u597d",
		"\u5e2e\u6211\u603b\u7ed3\u4e00\u4e0b",
		"\u7ffb\u8bd1\u8fd9\u6bb5\u8bdd",
	} {
		plan := planner.Plan(WorkspaceCapabilityPlannerInput{
			Text:          text,
			SelectedModel: "gpt-5.5",
		})

		require.Equal(t, WorkspacePlannedCapabilityTextChat, plan.PlannedCapability)
		require.Equal(t, workspaceCapabilityReasonDefaultTextChat, plan.Reason)
		require.True(t, plan.ModelCapabilityMatched)
		require.Empty(t, plan.BlockReason)
	}

	empty := planner.Plan(WorkspaceCapabilityPlannerInput{
		Text:          "   ",
		SelectedModel: "gpt-5.5",
	})
	require.Equal(t, WorkspacePlannedCapabilityUnknown, empty.PlannedCapability)
	require.Equal(t, workspaceCapabilityReasonEmptyText, empty.Reason)
	require.False(t, empty.ModelCapabilityMatched)
}

func TestWorkspaceCapabilityPlannerMetadataIsSafeAndDoesNotCopyPrompt(t *testing.T) {
	prompt := "\u5e2e\u6211\u751f\u6210\u4e00\u5f20\u9ad8\u7ea7\u611f\u4ea7\u54c1\u56fe"
	plan := NewWorkspaceCapabilityPlanner().Plan(WorkspaceCapabilityPlannerInput{
		Text:          prompt,
		SelectedModel: "gpt-5.5",
	})

	metadata := mergeWorkspaceCapabilityPlanMetadata(map[string]any{"existing": "kept"}, plan)

	require.Equal(t, "kept", metadata["existing"])
	require.Equal(t, "image_generation", metadata["planned_capability"])
	require.Equal(t, "workspace_capability_planner_v1", metadata["planner_version"])
	require.Equal(t, "gpt-5.5", metadata["selected_model"])
	require.Equal(t, false, metadata["model_capability_matched"])

	raw, err := json.Marshal(metadata)
	require.NoError(t, err)
	serialized := strings.ToLower(string(raw))
	require.NotContains(t, serialized, prompt)
	require.NotContains(t, serialized, "authorization")
	require.NotContains(t, serialized, "cookie")
	require.NotContains(t, serialized, "token")
	require.NotContains(t, serialized, "secret")
	require.NotContains(t, serialized, "api_key")
}
