package service

const (
	WorkspaceImageProviderFakeLabel = "workspace-image-fake"
	WorkspaceImageProviderFakeModel = "workspace-image-fake-model"

	workspaceImageProviderReasonFakeRoute = "fake_image_provider_route"
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

type WorkspaceImageProviderRouter struct{}

func NewWorkspaceImageProviderRouter() WorkspaceImageProviderRouter {
	return WorkspaceImageProviderRouter{}
}

func (WorkspaceImageProviderRouter) Route(input WorkspaceImageProviderRouterInput) WorkspaceImageProviderRoute {
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
	return WorkspaceImageProviderRoute{
		ProviderLabel: WorkspaceImageProviderFakeLabel,
		Provider:      WorkspaceImageProviderFakeLabel,
		Model:         WorkspaceImageProviderFakeModel,
		Capability:    WorkspacePlannedCapabilityImageGeneration,
		Reason:        workspaceImageProviderReasonFakeRoute,
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
