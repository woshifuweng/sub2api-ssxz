package service

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWorkspaceImageExperienceEnhancerBuildsCommercialPlans(t *testing.T) {
	tests := []struct {
		name        string
		text        string
		subject     string
		scene       string
		style       string
		aspectRatio string
		reason      string
	}{
		{
			name:        "premium perfume product",
			text:        "\u5e2e\u6211\u751f\u6210\u4e00\u5f20\u9ad8\u7ea7\u611f\u9999\u6c34\u4ea7\u54c1\u56fe",
			subject:     "perfume product",
			scene:       "commercial product image",
			style:       "commercial premium product photography",
			aspectRatio: "1:1",
			reason:      workspaceImageExperienceReasonCommercialProduct,
		},
		{
			name:        "xiaohongshu cover",
			text:        "\u753b\u4e00\u5f20\u5c0f\u7ea2\u4e66\u5c01\u9762",
			subject:     "social media cover",
			scene:       "vertical social media cover",
			style:       "clean premium social cover",
			aspectRatio: "4:5",
			reason:      workspaceImageExperienceReasonSocialCover,
		},
		{
			name:        "perfume ad",
			text:        "generate image of perfume ad",
			subject:     "perfume product",
			scene:       "commercial advertisement",
			style:       "premium marketing poster",
			aspectRatio: "4:5",
			reason:      workspaceImageExperienceReasonPoster,
		},
		{
			name:        "ecommerce main image",
			text:        "\u505a\u4e00\u5f20\u7535\u5546\u4e3b\u56fe",
			subject:     "product",
			scene:       "commercial product image",
			style:       "commercial premium product photography",
			aspectRatio: "1:1",
			reason:      workspaceImageExperienceReasonCommercialProduct,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := BuildWorkspaceImageExperiencePlan(WorkspaceImageExperienceEnhancerInput{
				Text:                    tt.text,
				PlannedCapability:       WorkspacePlannedCapabilityImageGeneration,
				SelectedModel:           "gpt-image-1",
				ModelCapabilityMetadata: ResolveWorkspaceModelCapabilities("gpt-image-1", WorkspaceModelCapabilityHints{}),
			})

			require.True(t, plan.Present)
			require.True(t, plan.OriginalPromptPresent)
			require.True(t, plan.EnhancedPromptPresent)
			require.True(t, plan.NegativePromptPresent)
			require.Equal(t, WorkspaceImageExperienceEnhancerVersion, plan.EnhancerVersion)
			require.Equal(t, tt.subject, plan.SubjectHint)
			require.Equal(t, tt.scene, plan.SceneHint)
			require.Equal(t, tt.style, plan.StyleHint)
			require.Equal(t, tt.aspectRatio, plan.AspectRatio)
			require.Equal(t, "commercial", plan.QualityPreset)
			require.Equal(t, tt.reason, plan.Reason)
			require.Contains(t, plan.EnhancedPrompt, tt.subject)
			require.NotEmpty(t, plan.NegativePrompt)
		})
	}
}

func TestWorkspaceImageExperienceEnhancerSkipsNonImageCapabilities(t *testing.T) {
	for _, capability := range []WorkspacePlannedCapability{
		WorkspacePlannedCapabilityTextChat,
		WorkspacePlannedCapabilityUnknown,
	} {
		plan := BuildWorkspaceImageExperiencePlan(WorkspaceImageExperienceEnhancerInput{
			Text:                    "hello",
			PlannedCapability:       capability,
			SelectedModel:           "gpt-5.5",
			ModelCapabilityMetadata: ResolveWorkspaceModelCapabilities("gpt-5.5", WorkspaceModelCapabilityHints{}),
		})

		require.False(t, plan.Present)
		require.Equal(t, workspaceImageExperienceReasonNotImageGeneration, plan.Reason)
		require.Empty(t, plan.EnhancedPrompt)
	}
}

func TestWorkspaceImageExperienceEnhancerCarriesSelectedModelWithoutProviderExecution(t *testing.T) {
	modelMetadata := ResolveWorkspaceModelCapabilities("deepseek-v4-flash", WorkspaceModelCapabilityHints{})
	plan := BuildWorkspaceImageExperiencePlan(WorkspaceImageExperienceEnhancerInput{
		Text:                    "\u5e2e\u6211\u751f\u6210\u4e00\u5f20\u9ad8\u7ea7\u611f\u4ea7\u54c1\u56fe",
		PlannedCapability:       WorkspacePlannedCapabilityImageGeneration,
		SelectedModel:           "deepseek-v4-flash",
		ModelCapabilityMetadata: modelMetadata,
	})

	require.True(t, plan.Present)
	require.Equal(t, "deepseek-v4-flash", plan.SelectedModel)
	require.False(t, plan.ModelCapabilityMatched)
	require.Equal(t, "selected_model_does_not_support_image_generation", plan.ModelCapabilityMismatch)

	metadata := mergeWorkspaceImageExperiencePlanMetadata(map[string]any{
		"existing": "kept",
	}, plan)

	require.Equal(t, "kept", metadata["existing"])
	require.Equal(t, true, metadata["image_experience_plan_present"])
	require.Equal(t, "product", metadata["image_subject_hint"])
	require.Equal(t, "commercial premium product photography", metadata["image_style_hint"])
	require.Equal(t, "1:1", metadata["image_aspect_ratio"])
	require.Equal(t, true, metadata["enhanced_prompt_present"])
	require.NotContains(t, metadata, "enhanced_prompt")
	require.NotContains(t, metadata, "negative_prompt")
	require.NotContains(t, metadata, "provider_called")
	require.NotContains(t, metadata, "image_task_id")
	require.NotContains(t, metadata, "assets")
}

func TestWorkspaceImageExperienceMetadataIsSafe(t *testing.T) {
	prompt := "\u5e2e\u6211\u751f\u6210\u4e00\u5f20\u9ad8\u7ea7\u611f\u9999\u6c34\u4ea7\u54c1\u56fe"
	plan := BuildWorkspaceImageExperiencePlan(WorkspaceImageExperienceEnhancerInput{
		Text:                    prompt,
		PlannedCapability:       WorkspacePlannedCapabilityImageGeneration,
		SelectedModel:           "gpt-image-1",
		ModelCapabilityMetadata: ResolveWorkspaceModelCapabilities("gpt-image-1", WorkspaceModelCapabilityHints{}),
	})

	metadata := mergeWorkspaceImageExperiencePlanMetadata(map[string]any{}, plan)
	raw, err := json.Marshal(metadata)
	require.NoError(t, err)
	serialized := strings.ToLower(string(raw))

	require.NotContains(t, serialized, strings.ToLower(prompt))
	require.NotContains(t, serialized, strings.ToLower(plan.EnhancedPrompt))
	require.NotContains(t, serialized, strings.ToLower(plan.NegativePrompt))
	require.NotContains(t, serialized, "authorization")
	require.NotContains(t, serialized, "cookie")
	require.NotContains(t, serialized, "token")
	require.NotContains(t, serialized, "secret")
	require.NotContains(t, serialized, "api_key")
}
