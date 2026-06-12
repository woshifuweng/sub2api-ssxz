package service

import "strings"

type WorkspacePlannedCapability string

const (
	WorkspacePlannedCapabilityTextChat        WorkspacePlannedCapability = "text_chat"
	WorkspacePlannedCapabilityImageGeneration WorkspacePlannedCapability = "image_generation"
	WorkspacePlannedCapabilityImageEdit       WorkspacePlannedCapability = "image_edit"
	WorkspacePlannedCapabilityVision          WorkspacePlannedCapability = "vision"
	WorkspacePlannedCapabilityFunctionCalling WorkspacePlannedCapability = "function_calling"
	WorkspacePlannedCapabilityTool            WorkspacePlannedCapability = "tool"
	WorkspacePlannedCapabilityUnknown         WorkspacePlannedCapability = "unknown"

	WorkspaceCapabilityPlannerVersion = "workspace_capability_planner_v1"

	workspaceCapabilityReasonEmptyText                 = "empty_text"
	workspaceCapabilityReasonDefaultTextChat           = "default_text_chat"
	workspaceCapabilityReasonZHImageGenerationKeyword  = "zh_image_generation_keyword"
	workspaceCapabilityReasonENImageGenerationKeyword  = "en_image_generation_keyword"
	workspaceCapabilityBlockModelCapabilityUnavailable = "model_capability_registry_missing"
)

type WorkspaceCapabilityPlannerInput struct {
	Text                string
	SelectedModel       string
	AttachmentCount     int
	ContextMessageCount int
}

type WorkspaceCapabilityPlan struct {
	PlannedCapability      WorkspacePlannedCapability
	Confidence             float64
	Reason                 string
	PlannerVersion         string
	SelectedModel          string
	ModelCapabilityMatched bool
	BlockReason            string
}

type WorkspaceCapabilityPlanner struct{}

func NewWorkspaceCapabilityPlanner() WorkspaceCapabilityPlanner {
	return WorkspaceCapabilityPlanner{}
}

func (WorkspaceCapabilityPlanner) Plan(input WorkspaceCapabilityPlannerInput) WorkspaceCapabilityPlan {
	text := strings.TrimSpace(input.Text)
	selectedModel := strings.TrimSpace(input.SelectedModel)
	plan := WorkspaceCapabilityPlan{
		PlannedCapability:      WorkspacePlannedCapabilityUnknown,
		Confidence:             0,
		Reason:                 workspaceCapabilityReasonEmptyText,
		PlannerVersion:         WorkspaceCapabilityPlannerVersion,
		SelectedModel:          selectedModel,
		ModelCapabilityMatched: false,
	}
	if text == "" {
		return plan
	}

	if reason, ok := matchWorkspaceImageGenerationKeyword(text); ok {
		plan.PlannedCapability = WorkspacePlannedCapabilityImageGeneration
		plan.Confidence = 0.92
		plan.Reason = reason
		plan.BlockReason = workspaceCapabilityBlockModelCapabilityUnavailable
		return plan
	}

	plan.PlannedCapability = WorkspacePlannedCapabilityTextChat
	plan.Confidence = 0.72
	plan.Reason = workspaceCapabilityReasonDefaultTextChat
	plan.ModelCapabilityMatched = isAllowedWorkspaceModel(selectedModel)
	return plan
}

func mergeWorkspaceCapabilityPlanMetadata(metadata map[string]any, plan WorkspaceCapabilityPlan) map[string]any {
	out := make(map[string]any, len(metadata)+7)
	for key, value := range metadata {
		out[key] = value
	}
	out["planned_capability"] = string(plan.PlannedCapability)
	out["planner_confidence"] = plan.Confidence
	out["planner_reason"] = plan.Reason
	out["planner_version"] = plan.PlannerVersion
	out["selected_model"] = plan.SelectedModel
	out["model_capability_matched"] = plan.ModelCapabilityMatched
	if strings.TrimSpace(plan.BlockReason) != "" {
		out["planner_block_reason"] = plan.BlockReason
	}
	return out
}

func matchWorkspaceImageGenerationKeyword(text string) (string, bool) {
	for _, keyword := range []string{
		"\u751f\u6210\u4e00\u5f20",
		"\u753b\u4e00\u5f20",
		"\u8bbe\u8ba1\u4e00\u5f20",
		"\u505a\u4e00\u5f20\u56fe",
		"\u751f\u6210\u56fe\u7247",
		"\u751f\u6210\u6d77\u62a5",
		"\u753b\u56fe",
		"\u751f\u56fe",
	} {
		if strings.Contains(text, keyword) {
			return workspaceCapabilityReasonZHImageGenerationKeyword, true
		}
	}

	lower := strings.ToLower(text)
	for _, keyword := range []string{
		"generate image",
		"create image",
		"draw an image",
		"make a poster",
		"image of",
		"poster of",
	} {
		if strings.Contains(lower, keyword) {
			return workspaceCapabilityReasonENImageGenerationKeyword, true
		}
	}
	return "", false
}
