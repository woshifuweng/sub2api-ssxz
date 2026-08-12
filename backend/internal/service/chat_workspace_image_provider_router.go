package service

const (
	WorkspaceImageProviderFakeLabel = "workspace-image-fake"
	WorkspaceImageProviderFakeModel = "workspace-image-fake-model"

	workspaceImageProviderReasonFakeRoute       = "fake_image_provider_route"
	workspaceImageProviderReasonConfiguredRoute = "configured_image_provider_route"
)

type WorkspaceImageProviderRoute struct {
	ProviderLabel string
	Provider      string
	Model         string
	Capability    WorkspacePlannedCapability
	Reason        string
	Available     bool
	ErrorCode     string
}

type WorkspaceImageProviderRouterInput struct {
	CapabilityPlan          WorkspaceCapabilityPlan
	ModelCapabilityMetadata WorkspaceModelCapabilityMetadata
	ImageExperiencePlan     WorkspaceImageExperiencePlan
	ForceUnavailable        bool
}

type WorkspaceImageProviderRouterConfig struct {
	ProviderLabel string
	Provider      string
}

type WorkspaceImageProviderRouter struct {
	Config WorkspaceImageProviderRouterConfig
}

func NewWorkspaceImageProviderRouter(config ...WorkspaceImageProviderRouterConfig) WorkspaceImageProviderRouter {
	router := WorkspaceImageProviderRouter{}
	if len(config) > 0 {
		router.Config = config[0]
	}
	return router
}

func (r WorkspaceImageProviderRouter) Route(input WorkspaceImageProviderRouterInput) WorkspaceImageProviderRoute {
	if input.CapabilityPlan.PlannedCapability != WorkspacePlannedCapabilityImageGeneration {
		return workspaceUnavailableImageProviderRoute("not_applicable")
	}
	if matched, _ := WorkspaceModelCapabilitiesMatch(WorkspacePlannedCapabilityImageGeneration, input.ModelCapabilityMetadata); !matched {
		return workspaceUnavailableImageProviderRoute("capability_mismatch")
	}
	if !input.ImageExperiencePlan.Present {
		return workspaceUnavailableImageProviderRoute("image_plan_missing")
	}
	if input.ForceUnavailable {
		return workspaceUnavailableImageProviderRoute("image_provider_unavailable")
	}
	if input.ModelCapabilityMetadata.ModelName == WorkspaceImageProviderFakeModel {
		return WorkspaceImageProviderRoute{
			ProviderLabel: WorkspaceImageProviderFakeLabel,
			Provider:      WorkspaceImageProviderFakeLabel,
			Model:         WorkspaceImageProviderFakeModel,
			Capability:    WorkspacePlannedCapabilityImageGeneration,
			Reason:        workspaceImageProviderReasonFakeRoute,
			Available:     true,
		}
	}
	providerLabel := firstNonEmptyWorkspaceValue(input.ModelCapabilityMetadata.ProviderLabel, r.Config.ProviderLabel)
	if providerLabel == "" {
		return workspaceUnavailableImageProviderRoute("image_provider_unavailable")
	}
	provider := firstNonEmptyWorkspaceValue(input.ModelCapabilityMetadata.Provider, r.Config.Provider)
	model := firstNonEmptyWorkspaceValue(input.ModelCapabilityMetadata.ModelName, input.CapabilityPlan.SelectedModel)
	if model == "" {
		return workspaceUnavailableImageProviderRoute("image_provider_unavailable")
	}
	return WorkspaceImageProviderRoute{
		ProviderLabel: providerLabel,
		Provider:      provider,
		Model:         model,
		Capability:    WorkspacePlannedCapabilityImageGeneration,
		Reason:        workspaceImageProviderReasonConfiguredRoute,
		Available:     true,
	}
}

func workspaceUnavailableImageProviderRoute(errorCode string) WorkspaceImageProviderRoute {
	return WorkspaceImageProviderRoute{
		Capability: WorkspacePlannedCapabilityImageGeneration,
		Reason:     errorCode,
		ErrorCode:  errorCode,
	}
}
