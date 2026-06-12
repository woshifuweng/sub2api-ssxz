package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	testWorkspaceOpenAICompatibleImageProviderLabel = "workspace-openai-compatible-image-staging"
	testWorkspaceOpenAICompatibleImageModel         = "gpt-image-1"
)

type recordingWorkspaceOpenAICompatibleImageExecutor struct {
	calls    int
	request  WorkspaceOpenAICompatibleImageGenerationRequest
	response WorkspaceOpenAICompatibleImageGenerationResponse
	err      error
}

func (e *recordingWorkspaceOpenAICompatibleImageExecutor) GenerateImage(_ context.Context, request WorkspaceOpenAICompatibleImageGenerationRequest) (WorkspaceOpenAICompatibleImageGenerationResponse, error) {
	e.calls++
	e.request = request
	return e.response, e.err
}

func testWorkspaceOpenAICompatibleImageProviderRequest() WorkspaceImageProviderRequest {
	modelMetadata := ResolveWorkspaceModelCapabilities(testWorkspaceOpenAICompatibleImageModel, WorkspaceModelCapabilityHints{
		ProviderLabel: testWorkspaceOpenAICompatibleImageProviderLabel,
		Provider:      WorkspaceOpenAICompatibleImageProviderKind,
		Capabilities:  []WorkspaceModelCapability{WorkspaceModelCapabilityImageGeneration},
	})
	capabilityPlan := ApplyWorkspaceModelCapabilityMatch(WorkspaceCapabilityPlan{
		PlannedCapability: WorkspacePlannedCapabilityImageGeneration,
		SelectedModel:     testWorkspaceOpenAICompatibleImageModel,
	}, modelMetadata)
	imagePlan := BuildWorkspaceImageExperiencePlan(WorkspaceImageExperienceEnhancerInput{
		Text:                    "generate image of perfume ad with secret-token",
		PlannedCapability:       WorkspacePlannedCapabilityImageGeneration,
		SelectedModel:           testWorkspaceOpenAICompatibleImageModel,
		ModelCapabilityMetadata: modelMetadata,
	})
	return WorkspaceImageProviderRequest{
		Route: WorkspaceImageProviderRoute{
			ProviderLabel: testWorkspaceOpenAICompatibleImageProviderLabel,
			Provider:      WorkspaceOpenAICompatibleImageProviderKind,
			Model:         testWorkspaceOpenAICompatibleImageModel,
			Capability:    WorkspacePlannedCapabilityImageGeneration,
			Reason:        workspaceImageProviderReasonConfiguredRoute,
			Available:     true,
		},
		CapabilityPlan:          capabilityPlan,
		ModelCapabilityMetadata: modelMetadata,
		ImageExperiencePlan:     imagePlan,
		RequestID:               "request-1",
	}
}

func TestWorkspaceOpenAICompatibleImageProviderAdapterBuildsRequestAndNormalizesResult(t *testing.T) {
	executor := &recordingWorkspaceOpenAICompatibleImageExecutor{
		response: WorkspaceOpenAICompatibleImageGenerationResponse{
			ProviderLabel: testWorkspaceOpenAICompatibleImageProviderLabel,
			Model:         testWorkspaceOpenAICompatibleImageModel,
			Images: []WorkspaceOpenAICompatibleImage{
				{
					ID:       "image-1",
					URL:      "https://example.invalid/generated.png",
					MimeType: "image/png",
					Width:    1024,
					Height:   1024,
				},
			},
			Usage:     WorkspaceImageProviderUsage{ImageCount: 1, ImageSize: "1024x1024"},
			LatencyMS: 1078,
		},
	}
	input := testWorkspaceOpenAICompatibleImageProviderRequest()

	result, err := NewWorkspaceOpenAICompatibleImageProviderAdapter(executor).GenerateImage(context.Background(), input)

	require.NoError(t, err)
	require.Equal(t, 1, executor.calls)
	require.Equal(t, testWorkspaceOpenAICompatibleImageProviderLabel, executor.request.ProviderLabel)
	require.Equal(t, testWorkspaceOpenAICompatibleImageModel, executor.request.Model)
	require.Equal(t, "1024x1536", executor.request.Size)
	require.Equal(t, 1, executor.request.Count)
	require.NotEmpty(t, executor.request.Prompt)
	require.NotEmpty(t, executor.request.NegativePrompt)
	require.Equal(t, testWorkspaceOpenAICompatibleImageProviderLabel, result.ProviderLabel)
	require.Equal(t, testWorkspaceOpenAICompatibleImageModel, result.Model)
	require.Equal(t, 1, result.Usage.ImageCount)
	require.Equal(t, "1024x1024", result.Usage.ImageSize)
	require.Equal(t, int64(1078), result.LatencyMS)
	require.Len(t, result.Assets, 1)
	require.Equal(t, "https://example.invalid/generated.png", result.Assets[0].URL)
}

func TestWorkspaceOpenAICompatibleImageProviderAdapterNilExecutorFailsClosed(t *testing.T) {
	result, err := NewWorkspaceOpenAICompatibleImageProviderAdapter(nil).GenerateImage(context.Background(), testWorkspaceOpenAICompatibleImageProviderRequest())

	require.ErrorIs(t, err, ErrWorkspaceImageProviderFailed)
	require.Equal(t, "image_provider_unavailable", result.ErrorCode)
}

func TestWorkspaceOpenAICompatibleImageProviderAdapterFailureIsSanitized(t *testing.T) {
	executor := &recordingWorkspaceOpenAICompatibleImageExecutor{err: errors.New("provider failed with Authorization bearer token")}

	result, err := NewWorkspaceOpenAICompatibleImageProviderAdapter(executor).GenerateImage(context.Background(), testWorkspaceOpenAICompatibleImageProviderRequest())

	require.ErrorIs(t, err, ErrWorkspaceImageProviderFailed)
	require.Equal(t, "image_provider_failed", result.ErrorCode)
	encoded, marshalErr := json.Marshal(result)
	require.NoError(t, marshalErr)
	body := strings.ToLower(string(encoded))
	require.NotContains(t, body, "authorization bearer")
	require.NotContains(t, body, "token")
	require.NotContains(t, body, "cookie")
	require.NotContains(t, body, "secret")
}

func TestWorkspaceOpenAICompatibleImageProviderBoundaryMetadataExcludesFullPromptAndSecrets(t *testing.T) {
	executor := &recordingWorkspaceOpenAICompatibleImageExecutor{
		response: WorkspaceOpenAICompatibleImageGenerationResponse{
			ProviderLabel: testWorkspaceOpenAICompatibleImageProviderLabel,
			Model:         testWorkspaceOpenAICompatibleImageModel,
			Images: []WorkspaceOpenAICompatibleImage{
				{ID: "image-1", URL: "https://example.invalid/generated.png", MimeType: "image/png", Width: 1024, Height: 1024},
			},
			Usage: WorkspaceImageProviderUsage{ImageCount: 1, ImageSize: "1024x1024"},
		},
	}
	input := testWorkspaceImageProviderBoundaryInput()
	input.ModelCapabilityMetadata = ResolveWorkspaceModelCapabilities(testWorkspaceOpenAICompatibleImageModel, WorkspaceModelCapabilityHints{
		ProviderLabel: testWorkspaceOpenAICompatibleImageProviderLabel,
		Provider:      WorkspaceOpenAICompatibleImageProviderKind,
		Capabilities:  []WorkspaceModelCapability{WorkspaceModelCapabilityImageGeneration},
	})
	input.CapabilityPlan.SelectedModel = testWorkspaceOpenAICompatibleImageModel
	input.CapabilityPlan.ModelCapabilityMatched = true
	input.ImageExperiencePlan = BuildWorkspaceImageExperiencePlan(WorkspaceImageExperienceEnhancerInput{
		Text:                    "generate image of perfume ad with secret-token",
		PlannedCapability:       WorkspacePlannedCapabilityImageGeneration,
		SelectedModel:           testWorkspaceOpenAICompatibleImageModel,
		ModelCapabilityMetadata: input.ModelCapabilityMetadata,
	})

	result := NewWorkspaceImageProviderBoundary(
		NewWorkspaceOpenAICompatibleImageProviderAdapter(executor),
		WorkspaceImageProviderRouterConfig{
			ProviderLabel: testWorkspaceOpenAICompatibleImageProviderLabel,
			Provider:      WorkspaceOpenAICompatibleImageProviderKind,
		},
	).GenerateAssistantImage(context.Background(), input)

	require.Equal(t, WorkspaceMessageStatusCompleted, result.Status)
	require.Equal(t, testWorkspaceOpenAICompatibleImageProviderLabel, result.Metadata["provider_label"])
	require.Equal(t, testWorkspaceOpenAICompatibleImageModel, result.Metadata["model"])
	require.Equal(t, 1, result.Metadata["usage_image_count"])
	encoded, err := json.Marshal(result)
	require.NoError(t, err)
	body := strings.ToLower(string(encoded))
	require.NotContains(t, body, "generate image of perfume ad")
	require.NotContains(t, body, "secret-token")
	require.NotContains(t, body, "authorization")
	require.NotContains(t, body, "cookie")
	require.NotContains(t, body, "api_key")
	require.NotContains(t, body, "token")
	require.NotContains(t, body, "secret")
}
