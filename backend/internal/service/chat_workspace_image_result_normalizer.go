package service

import (
	"errors"
	"net/url"
	"strings"
)

var ErrWorkspaceImageResultInvalid = errors.New("workspace image result is invalid")

type WorkspaceImageResultNormalizer struct{}

type WorkspaceImageResultNormalizationInput struct {
	Result              WorkspaceImageProviderResult
	ImageExperiencePlan WorkspaceImageExperiencePlan
}

type WorkspaceImageAssistantMessageResult struct {
	Content     string
	MessageType string
	Intent      string
	Status      string
	Metadata    map[string]any
}

func NewWorkspaceImageResultNormalizer() WorkspaceImageResultNormalizer {
	return WorkspaceImageResultNormalizer{}
}

func (WorkspaceImageResultNormalizer) NormalizeSuccess(input WorkspaceImageResultNormalizationInput) (WorkspaceImageAssistantMessageResult, error) {
	if len(input.Result.Assets) == 0 {
		return WorkspaceImageAssistantMessageResult{}, ErrWorkspaceImageResultInvalid
	}
	assets := make([]any, 0, len(input.Result.Assets))
	for _, asset := range input.Result.Assets {
		if !isSafeWorkspaceImageProviderURL(asset.URL) {
			return WorkspaceImageAssistantMessageResult{}, ErrWorkspaceImageResultInvalid
		}
		assets = append(assets, map[string]any{
			"id":        strings.TrimSpace(asset.ID),
			"url":       strings.TrimSpace(asset.URL),
			"mime_type": firstNonEmptyWorkspaceValue(asset.MimeType, "image/png"),
			"width":     asset.Width,
			"height":    asset.Height,
			"provider":  strings.TrimSpace(asset.Provider),
			"model":     strings.TrimSpace(asset.Model),
		})
	}
	metadata := map[string]any{
		"capability":                    WorkspaceIntentImageGeneration,
		"result_type":                   "image",
		"status":                        WorkspaceMessageStatusCompleted,
		"provider_label":                strings.TrimSpace(input.Result.ProviderLabel),
		"model":                         strings.TrimSpace(input.Result.Model),
		"assets":                        assets,
		"image_experience_plan_present": input.ImageExperiencePlan.Present,
		"enhanced_prompt_present":       input.ImageExperiencePlan.EnhancedPromptPresent,
		"usage_image_count":             input.Result.Usage.ImageCount,
		"usage_image_size":              input.Result.Usage.ImageSize,
		"latency_ms":                    input.Result.LatencyMS,
		"error_code":                    "",
	}
	if metadataContainsUnsafeInlinePayload(metadata) {
		return WorkspaceImageAssistantMessageResult{}, ErrWorkspaceImageResultInvalid
	}
	return WorkspaceImageAssistantMessageResult{
		Content:     "Generated image is ready.",
		MessageType: WorkspaceMessageTypeImage,
		Intent:      WorkspaceIntentImageGeneration,
		Status:      WorkspaceMessageStatusCompleted,
		Metadata:    metadata,
	}, nil
}

func (WorkspaceImageResultNormalizer) NormalizeFailure(errorCode, errorMessage string) WorkspaceImageAssistantMessageResult {
	errorCode = firstNonEmptyWorkspaceValue(errorCode, "image_provider_failed")
	errorMessage = firstNonEmptyWorkspaceValue(errorMessage, "Image generation failed. Please try again.")
	return WorkspaceImageAssistantMessageResult{
		Content:     errorMessage,
		MessageType: WorkspaceMessageTypeImage,
		Intent:      WorkspaceIntentImageGeneration,
		Status:      WorkspaceMessageStatusFailed,
		Metadata: map[string]any{
			"capability":    WorkspaceIntentImageGeneration,
			"result_type":   "image",
			"status":        WorkspaceMessageStatusFailed,
			"error_code":    errorCode,
			"error_message": errorMessage,
		},
	}
}

func isSafeWorkspaceImageProviderURL(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || containsUnsafeInlinePayload(value) {
		return false
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return false
	}
	return parsed.Scheme == "https" && parsed.Host != ""
}
