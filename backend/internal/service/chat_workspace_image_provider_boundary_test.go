package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type countingWorkspaceImageProviderAdapter struct {
	calls  int
	result WorkspaceImageProviderResult
	err    error
}

func (a *countingWorkspaceImageProviderAdapter) GenerateImage(_ context.Context, _ WorkspaceImageProviderRequest) (WorkspaceImageProviderResult, error) {
	a.calls++
	return a.result, a.err
}

func testWorkspaceImageProviderBoundaryInput() WorkspaceImageProviderBoundaryInput {
	capabilityPlan := WorkspaceCapabilityPlan{
		PlannedCapability:      WorkspacePlannedCapabilityImageGeneration,
		Confidence:             0.92,
		Reason:                 workspaceCapabilityReasonENImageGenerationKeyword,
		PlannerVersion:         WorkspaceCapabilityPlannerVersion,
		SelectedModel:          WorkspaceImageProviderFakeModel,
		ModelCapabilityMatched: true,
	}
	modelMetadata := ResolveWorkspaceModelCapabilities(WorkspaceImageProviderFakeModel, WorkspaceModelCapabilityHints{
		ProviderLabel: WorkspaceImageProviderFakeLabel,
		Capabilities:  []WorkspaceModelCapability{WorkspaceModelCapabilityImageGeneration},
	})
	imagePlan := BuildWorkspaceImageExperiencePlan(WorkspaceImageExperienceEnhancerInput{
		Text:                    "generate image of perfume ad",
		PlannedCapability:       capabilityPlan.PlannedCapability,
		SelectedModel:           WorkspaceImageProviderFakeModel,
		ModelCapabilityMetadata: modelMetadata,
	})
	return WorkspaceImageProviderBoundaryInput{
		CapabilityPlan:          capabilityPlan,
		ModelCapabilityMetadata: modelMetadata,
		ImageExperiencePlan:     imagePlan,
		RequestID:               "request-1",
	}
}

func TestWorkspaceImageProviderBoundaryFakeSuccessNormalizesAssistantImageMetadata(t *testing.T) {
	result := NewWorkspaceImageProviderBoundary(WorkspaceImageFakeProviderAdapter{}).GenerateAssistantImage(context.Background(), testWorkspaceImageProviderBoundaryInput())

	require.Equal(t, WorkspaceMessageTypeImage, result.MessageType)
	require.Equal(t, WorkspaceIntentImageGeneration, result.Intent)
	require.Equal(t, WorkspaceMessageStatusCompleted, result.Status)
	require.Equal(t, "image", result.Metadata["result_type"])
	require.Equal(t, WorkspaceImageProviderFakeLabel, result.Metadata["provider_label"])
	require.Equal(t, WorkspaceImageProviderFakeModel, result.Metadata["model"])
	require.Equal(t, 1, result.Metadata["usage_image_count"])
	require.Equal(t, "1024x1024", result.Metadata["usage_image_size"])
	require.Equal(t, true, result.Metadata["image_experience_plan_present"])
	require.Equal(t, true, result.Metadata["enhanced_prompt_present"])
	require.Equal(t, "", result.Metadata["error_code"])

	assets, ok := result.Metadata["assets"].([]any)
	require.True(t, ok)
	require.Len(t, assets, 1)
	asset, ok := assets[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "https://example.invalid/workspace-image-fake.png", asset["url"])
	require.Equal(t, WorkspaceImageProviderFakeLabel, asset["provider"])
	require.Equal(t, WorkspaceImageProviderFakeModel, asset["model"])

	encoded, err := json.Marshal(result)
	require.NoError(t, err)
	body := strings.ToLower(string(encoded))
	require.NotContains(t, body, "generate image of perfume ad")
	require.NotContains(t, body, "create a provider-agnostic image generation plan")
	require.NotContains(t, body, "cheap, cluttered")
	require.NotContains(t, body, "authorization")
	require.NotContains(t, body, "cookie")
	require.NotContains(t, body, "api_key")
	require.NotContains(t, body, "token")
	require.NotContains(t, body, "secret")
}

func TestWorkspaceImageProviderBoundaryTextChatDoesNotCallAdapter(t *testing.T) {
	input := testWorkspaceImageProviderBoundaryInput()
	input.CapabilityPlan.PlannedCapability = WorkspacePlannedCapabilityTextChat
	adapter := &countingWorkspaceImageProviderAdapter{}

	result := NewWorkspaceImageProviderBoundary(adapter).GenerateAssistantImage(context.Background(), input)

	require.Equal(t, 0, adapter.calls)
	require.Equal(t, WorkspaceMessageStatusFailed, result.Status)
	require.Equal(t, "not_applicable", result.Metadata["error_code"])
}

func TestWorkspaceImageProviderBoundaryCapabilityMismatchDoesNotCallAdapter(t *testing.T) {
	input := testWorkspaceImageProviderBoundaryInput()
	input.ModelCapabilityMetadata = ResolveWorkspaceModelCapabilities("deepseek-v4-flash", WorkspaceModelCapabilityHints{})
	adapter := &countingWorkspaceImageProviderAdapter{}

	result := NewWorkspaceImageProviderBoundary(adapter).GenerateAssistantImage(context.Background(), input)

	require.Equal(t, 0, adapter.calls)
	require.Equal(t, WorkspaceMessageStatusFailed, result.Status)
	require.Equal(t, "capability_mismatch", result.Metadata["error_code"])
}

func TestWorkspaceImageProviderBoundaryMissingImagePlanDoesNotCallAdapter(t *testing.T) {
	input := testWorkspaceImageProviderBoundaryInput()
	input.ImageExperiencePlan = WorkspaceImageExperiencePlan{}
	adapter := &countingWorkspaceImageProviderAdapter{}

	result := NewWorkspaceImageProviderBoundary(adapter).GenerateAssistantImage(context.Background(), input)

	require.Equal(t, 0, adapter.calls)
	require.Equal(t, WorkspaceMessageStatusFailed, result.Status)
	require.Equal(t, "image_plan_missing", result.Metadata["error_code"])
}

func TestWorkspaceImageProviderBoundaryRouteUnavailableDoesNotCallAdapter(t *testing.T) {
	input := testWorkspaceImageProviderBoundaryInput()
	input.ForceRouteUnavailable = true
	adapter := &countingWorkspaceImageProviderAdapter{}

	result := NewWorkspaceImageProviderBoundary(adapter).GenerateAssistantImage(context.Background(), input)

	require.Equal(t, 0, adapter.calls)
	require.Equal(t, WorkspaceMessageStatusFailed, result.Status)
	require.Equal(t, "image_provider_unavailable", result.Metadata["error_code"])
}

func TestWorkspaceImageProviderBoundaryAdapterFailureReturnsSafeFailure(t *testing.T) {
	result := NewWorkspaceImageProviderBoundary(WorkspaceImageFakeProviderAdapter{Fail: true}).GenerateAssistantImage(context.Background(), testWorkspaceImageProviderBoundaryInput())

	require.Equal(t, WorkspaceMessageStatusFailed, result.Status)
	require.Equal(t, "image_provider_failed", result.Metadata["error_code"])

	encoded, err := json.Marshal(result)
	require.NoError(t, err)
	body := strings.ToLower(string(encoded))
	require.NotContains(t, body, "stack")
	require.NotContains(t, body, "authorization")
	require.NotContains(t, body, "cookie")
	require.NotContains(t, body, "token")
	require.NotContains(t, body, "secret")
}

func TestWorkspaceImageResultNormalizerRejectsUnsafeURLs(t *testing.T) {
	normalizer := NewWorkspaceImageResultNormalizer()
	for _, unsafeURL := range []string{
		"data:image/png;base64,abc",
		"blob:https://example.test/id",
		"javascript:alert(1)",
	} {
		_, err := normalizer.NormalizeSuccess(WorkspaceImageResultNormalizationInput{
			Result: WorkspaceImageProviderResult{
				ProviderLabel: WorkspaceImageProviderFakeLabel,
				Model:         WorkspaceImageProviderFakeModel,
				Assets: []WorkspaceImageProviderAsset{
					{ID: "asset", URL: unsafeURL, MimeType: "image/png", Width: 1024, Height: 1024},
				},
				Usage: WorkspaceImageProviderUsage{ImageCount: 1, ImageSize: "1024x1024"},
			},
			ImageExperiencePlan: testWorkspaceImageProviderBoundaryInput().ImageExperiencePlan,
		})
		require.ErrorIs(t, err, ErrWorkspaceImageResultInvalid)
	}
}

func TestWorkspaceImageResultNormalizerAcceptsSafeHTTPSURL(t *testing.T) {
	normalizer := NewWorkspaceImageResultNormalizer()

	result, err := normalizer.NormalizeSuccess(WorkspaceImageResultNormalizationInput{
		Result: WorkspaceImageProviderResult{
			ProviderLabel: WorkspaceImageProviderFakeLabel,
			Model:         WorkspaceImageProviderFakeModel,
			Assets: []WorkspaceImageProviderAsset{
				{ID: "asset", URL: "https://example.invalid/image.png", MimeType: "image/png", Width: 1024, Height: 1024, Provider: WorkspaceImageProviderFakeLabel, Model: WorkspaceImageProviderFakeModel},
			},
			Usage: WorkspaceImageProviderUsage{ImageCount: 1, ImageSize: "1024x1024"},
		},
		ImageExperiencePlan: testWorkspaceImageProviderBoundaryInput().ImageExperiencePlan,
	})

	require.NoError(t, err)
	require.Equal(t, WorkspaceMessageStatusCompleted, result.Status)
	require.Equal(t, "https://example.invalid/image.png", result.Metadata["assets"].([]any)[0].(map[string]any)["url"])
}

func TestWorkspaceImageProviderBoundaryNormalizerFailureIsSafe(t *testing.T) {
	input := testWorkspaceImageProviderBoundaryInput()
	adapter := &countingWorkspaceImageProviderAdapter{
		result: WorkspaceImageProviderResult{
			ProviderLabel: WorkspaceImageProviderFakeLabel,
			Model:         WorkspaceImageProviderFakeModel,
			Assets: []WorkspaceImageProviderAsset{
				{ID: "asset", URL: "data:image/png;base64,abc", MimeType: "image/png"},
			},
			Usage: WorkspaceImageProviderUsage{ImageCount: 1, ImageSize: "1024x1024"},
		},
	}

	result := NewWorkspaceImageProviderBoundary(adapter).GenerateAssistantImage(context.Background(), input)

	require.Equal(t, 1, adapter.calls)
	require.Equal(t, WorkspaceMessageStatusFailed, result.Status)
	require.Equal(t, "image_result_invalid", result.Metadata["error_code"])
}

func TestWorkspaceImageProviderBoundaryAdapterErrorCodeFallsBackSafely(t *testing.T) {
	input := testWorkspaceImageProviderBoundaryInput()
	adapter := &countingWorkspaceImageProviderAdapter{err: errors.New("network: Authorization bearer leaked")}

	result := NewWorkspaceImageProviderBoundary(adapter).GenerateAssistantImage(context.Background(), input)

	require.Equal(t, 1, adapter.calls)
	require.Equal(t, WorkspaceMessageStatusFailed, result.Status)
	require.Equal(t, "image_provider_failed", result.Metadata["error_code"])
	encoded, err := json.Marshal(result)
	require.NoError(t, err)
	require.NotContains(t, strings.ToLower(string(encoded)), "authorization bearer leaked")
}
