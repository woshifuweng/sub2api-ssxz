package service

import (
	"context"
	"strings"
)

const (
	WorkspaceOpenAICompatibleImageProviderKind = "openai-compatible-images"

	workspaceOpenAICompatibleImageDefaultCount = 1
	workspaceOpenAICompatibleImageDefaultSize  = "1024x1024"
)

type WorkspaceOpenAICompatibleImageGenerationRequest struct {
	RequestID      string
	ProviderLabel  string
	Model          string
	Prompt         string
	NegativePrompt string
	Size           string
	Count          int
	QualityPreset  string
}

type WorkspaceOpenAICompatibleImageGenerationResponse struct {
	ProviderLabel string
	Model         string
	Images        []WorkspaceOpenAICompatibleImage
	Usage         WorkspaceImageProviderUsage
	LatencyMS     int64
	ErrorCode     string
	ErrorMessage  string
}

type WorkspaceOpenAICompatibleImage struct {
	ID       string
	URL      string
	MimeType string
	Width    int
	Height   int
}

type WorkspaceOpenAICompatibleImageExecutor interface {
	GenerateImage(ctx context.Context, request WorkspaceOpenAICompatibleImageGenerationRequest) (WorkspaceOpenAICompatibleImageGenerationResponse, error)
}

type WorkspaceOpenAICompatibleImageProviderAdapter struct {
	Executor WorkspaceOpenAICompatibleImageExecutor
}

func NewWorkspaceOpenAICompatibleImageProviderAdapter(executor WorkspaceOpenAICompatibleImageExecutor) WorkspaceOpenAICompatibleImageProviderAdapter {
	return WorkspaceOpenAICompatibleImageProviderAdapter{Executor: executor}
}

func (a WorkspaceOpenAICompatibleImageProviderAdapter) GenerateImage(ctx context.Context, input WorkspaceImageProviderRequest) (WorkspaceImageProviderResult, error) {
	if !input.Route.Available || strings.TrimSpace(input.Route.ProviderLabel) == "" {
		return workspaceOpenAICompatibleImageProviderFailure(input, "image_provider_unavailable"), ErrWorkspaceImageProviderFailed
	}
	if a.Executor == nil {
		return workspaceOpenAICompatibleImageProviderFailure(input, "image_provider_unavailable"), ErrWorkspaceImageProviderFailed
	}

	request := buildWorkspaceOpenAICompatibleImageRequest(input)
	response, err := a.Executor.GenerateImage(ctx, request)
	if err != nil {
		return workspaceOpenAICompatibleImageProviderFailure(input, "image_provider_failed"), ErrWorkspaceImageProviderFailed
	}
	if response.ErrorCode != "" {
		return WorkspaceImageProviderResult{
			ProviderLabel: firstNonEmptyWorkspaceValue(response.ProviderLabel, input.Route.ProviderLabel),
			Model:         firstNonEmptyWorkspaceValue(response.Model, input.Route.Model),
			Capability:    WorkspacePlannedCapabilityImageGeneration,
			ErrorCode:     response.ErrorCode,
			ErrorMessage:  "Image provider failed.",
		}, ErrWorkspaceImageProviderFailed
	}

	assets := make([]WorkspaceImageProviderAsset, 0, len(response.Images))
	for _, image := range response.Images {
		assets = append(assets, WorkspaceImageProviderAsset{
			ID:       strings.TrimSpace(image.ID),
			URL:      strings.TrimSpace(image.URL),
			MimeType: firstNonEmptyWorkspaceValue(image.MimeType, "image/png"),
			Width:    image.Width,
			Height:   image.Height,
			Provider: firstNonEmptyWorkspaceValue(response.ProviderLabel, input.Route.ProviderLabel),
			Model:    firstNonEmptyWorkspaceValue(response.Model, input.Route.Model),
		})
	}

	usage := response.Usage
	if usage.ImageCount == 0 {
		usage.ImageCount = len(assets)
	}
	if usage.ImageSize == "" {
		usage.ImageSize = workspaceOpenAICompatibleImageSizeFromPlan(input.ImageExperiencePlan)
	}
	return WorkspaceImageProviderResult{
		ProviderLabel: firstNonEmptyWorkspaceValue(response.ProviderLabel, input.Route.ProviderLabel),
		Model:         firstNonEmptyWorkspaceValue(response.Model, input.Route.Model),
		Capability:    WorkspacePlannedCapabilityImageGeneration,
		Assets:        assets,
		Usage:         usage,
		LatencyMS:     response.LatencyMS,
	}, nil
}

func buildWorkspaceOpenAICompatibleImageRequest(input WorkspaceImageProviderRequest) WorkspaceOpenAICompatibleImageGenerationRequest {
	count := workspaceOpenAICompatibleImageDefaultCount
	return WorkspaceOpenAICompatibleImageGenerationRequest{
		RequestID:      strings.TrimSpace(input.RequestID),
		ProviderLabel:  strings.TrimSpace(input.Route.ProviderLabel),
		Model:          strings.TrimSpace(input.Route.Model),
		Prompt:         strings.TrimSpace(input.ImageExperiencePlan.EnhancedPrompt),
		NegativePrompt: strings.TrimSpace(input.ImageExperiencePlan.NegativePrompt),
		Size:           workspaceOpenAICompatibleImageSizeFromPlan(input.ImageExperiencePlan),
		Count:          count,
		QualityPreset:  strings.TrimSpace(input.ImageExperiencePlan.QualityPreset),
	}
}

func workspaceOpenAICompatibleImageSizeFromPlan(plan WorkspaceImageExperiencePlan) string {
	switch strings.TrimSpace(plan.AspectRatio) {
	case "16:9":
		return "1792x1024"
	case "4:5", "3:4":
		return "1024x1536"
	default:
		return workspaceOpenAICompatibleImageDefaultSize
	}
}

func workspaceOpenAICompatibleImageProviderFailure(input WorkspaceImageProviderRequest, code string) WorkspaceImageProviderResult {
	return WorkspaceImageProviderResult{
		ProviderLabel: strings.TrimSpace(input.Route.ProviderLabel),
		Model:         strings.TrimSpace(input.Route.Model),
		Capability:    WorkspacePlannedCapabilityImageGeneration,
		ErrorCode:     code,
		ErrorMessage:  "Image provider failed.",
	}
}
