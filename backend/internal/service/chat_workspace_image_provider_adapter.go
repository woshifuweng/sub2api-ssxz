package service

import (
	"context"
	"errors"
)

var ErrWorkspaceImageProviderFailed = errors.New("workspace image provider failed")

type WorkspaceImageProviderRequest struct {
	Route                   WorkspaceImageProviderRoute
	CapabilityPlan          WorkspaceCapabilityPlan
	ModelCapabilityMetadata WorkspaceModelCapabilityMetadata
	ImageExperiencePlan     WorkspaceImageExperiencePlan
	RequestID               string
}

type WorkspaceImageProviderAsset struct {
	ID       string
	URL      string
	MimeType string
	Width    int
	Height   int
	Provider string
	Model    string
}

type WorkspaceImageProviderUsage struct {
	ImageCount int
	ImageSize  string
}

type WorkspaceImageProviderResult struct {
	ProviderLabel string
	Model         string
	Capability    WorkspacePlannedCapability
	Assets        []WorkspaceImageProviderAsset
	Usage         WorkspaceImageProviderUsage
	LatencyMS     int64
	ErrorCode     string
	ErrorMessage  string
}

type WorkspaceImageProviderAdapter interface {
	GenerateImage(ctx context.Context, input WorkspaceImageProviderRequest) (WorkspaceImageProviderResult, error)
}

type WorkspaceImageFakeProviderAdapter struct {
	Fail      bool
	UnsafeURL string
}

func (a WorkspaceImageFakeProviderAdapter) GenerateImage(_ context.Context, input WorkspaceImageProviderRequest) (WorkspaceImageProviderResult, error) {
	if !input.Route.Available || input.Route.ProviderLabel != WorkspaceImageProviderFakeLabel {
		return WorkspaceImageProviderResult{
			ProviderLabel: input.Route.ProviderLabel,
			Model:         input.Route.Model,
			Capability:    WorkspacePlannedCapabilityImageGeneration,
			ErrorCode:     "image_provider_unavailable",
			ErrorMessage:  "Image provider is unavailable.",
		}, ErrWorkspaceImageProviderFailed
	}
	if a.Fail {
		return WorkspaceImageProviderResult{
			ProviderLabel: input.Route.ProviderLabel,
			Model:         input.Route.Model,
			Capability:    WorkspacePlannedCapabilityImageGeneration,
			ErrorCode:     "image_provider_failed",
			ErrorMessage:  "Image provider failed.",
		}, ErrWorkspaceImageProviderFailed
	}
	imageURL := "https://example.invalid/workspace-image-fake.png"
	if a.UnsafeURL != "" {
		imageURL = a.UnsafeURL
	}
	return WorkspaceImageProviderResult{
		ProviderLabel: input.Route.ProviderLabel,
		Model:         input.Route.Model,
		Capability:    WorkspacePlannedCapabilityImageGeneration,
		Assets: []WorkspaceImageProviderAsset{
			{
				ID:       "fake-image-asset",
				URL:      imageURL,
				MimeType: "image/png",
				Width:    1024,
				Height:   1024,
				Provider: input.Route.ProviderLabel,
				Model:    input.Route.Model,
			},
		},
		Usage: WorkspaceImageProviderUsage{
			ImageCount: 1,
			ImageSize:  "1024x1024",
		},
	}, nil
}

type WorkspaceImageProviderBoundaryInput struct {
	CapabilityPlan          WorkspaceCapabilityPlan
	ModelCapabilityMetadata WorkspaceModelCapabilityMetadata
	ImageExperiencePlan     WorkspaceImageExperiencePlan
	ForceRouteUnavailable   bool
	RequestID               string
}

type WorkspaceImageProviderBoundary struct {
	Router     WorkspaceImageProviderRouter
	Adapter    WorkspaceImageProviderAdapter
	Normalizer WorkspaceImageResultNormalizer
}

func NewWorkspaceImageProviderBoundary(adapter WorkspaceImageProviderAdapter, routerConfig ...WorkspaceImageProviderRouterConfig) WorkspaceImageProviderBoundary {
	if adapter == nil {
		adapter = WorkspaceImageFakeProviderAdapter{}
	}
	var config WorkspaceImageProviderRouterConfig
	if len(routerConfig) > 0 {
		config = routerConfig[0]
	}
	return WorkspaceImageProviderBoundary{
		Router:     NewWorkspaceImageProviderRouter(config),
		Adapter:    adapter,
		Normalizer: NewWorkspaceImageResultNormalizer(),
	}
}

func (b WorkspaceImageProviderBoundary) GenerateAssistantImage(ctx context.Context, input WorkspaceImageProviderBoundaryInput) WorkspaceImageAssistantMessageResult {
	route := b.Router.Route(WorkspaceImageProviderRouterInput{
		CapabilityPlan:          input.CapabilityPlan,
		ModelCapabilityMetadata: input.ModelCapabilityMetadata,
		ImageExperiencePlan:     input.ImageExperiencePlan,
		ForceUnavailable:        input.ForceRouteUnavailable,
	})
	if !route.Available {
		return b.Normalizer.NormalizeFailure(route.ErrorCode, "")
	}
	adapter := b.Adapter
	if adapter == nil {
		adapter = WorkspaceImageFakeProviderAdapter{}
	}
	result, err := adapter.GenerateImage(ctx, WorkspaceImageProviderRequest{
		Route:                   route,
		CapabilityPlan:          input.CapabilityPlan,
		ModelCapabilityMetadata: input.ModelCapabilityMetadata,
		ImageExperiencePlan:     input.ImageExperiencePlan,
		RequestID:               input.RequestID,
	})
	if err != nil || result.ErrorCode != "" {
		return b.Normalizer.NormalizeFailure(firstNonEmptyWorkspaceValue(result.ErrorCode, "image_provider_failed"), "")
	}
	message, err := b.Normalizer.NormalizeSuccess(WorkspaceImageResultNormalizationInput{
		Result:              result,
		ImageExperiencePlan: input.ImageExperiencePlan,
	})
	if err != nil {
		return b.Normalizer.NormalizeFailure("image_result_invalid", "")
	}
	return message
}
