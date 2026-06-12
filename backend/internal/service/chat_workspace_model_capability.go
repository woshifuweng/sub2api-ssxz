package service

import "strings"

type WorkspaceModelCapability string

const (
	WorkspaceModelCapabilityTextChat        WorkspaceModelCapability = "text_chat"
	WorkspaceModelCapabilityVision          WorkspaceModelCapability = "vision"
	WorkspaceModelCapabilityImageGeneration WorkspaceModelCapability = "image_generation"
	WorkspaceModelCapabilityImageEdit       WorkspaceModelCapability = "image_edit"
	WorkspaceModelCapabilityFunctionCalling WorkspaceModelCapability = "function_calling"
	WorkspaceModelCapabilityTool            WorkspaceModelCapability = "tool"

	workspaceModelCapabilitySourceExplicit     = "explicit_metadata"
	workspaceModelCapabilitySourceStaticRule   = "static_rule"
	workspaceModelCapabilitySourceSafeFallback = "safe_fallback"
)

type WorkspaceModelCapabilityHints struct {
	ProviderLabel           string
	Provider                string
	Platform                string
	Capabilities            []WorkspaceModelCapability
	Tags                    []string
	SupportsVision          bool
	SupportsFunctionCalling bool
}

type WorkspaceModelCapabilityMetadata struct {
	ModelName        string
	ProviderLabel    string
	Provider         string
	Platform         string
	Capabilities     []WorkspaceModelCapability
	CapabilitySource string
	Confidence       float64
}

func ResolveWorkspaceModelCapabilities(modelName string, hints WorkspaceModelCapabilityHints) WorkspaceModelCapabilityMetadata {
	modelName = strings.TrimSpace(modelName)
	metadata := WorkspaceModelCapabilityMetadata{
		ModelName:        modelName,
		ProviderLabel:    strings.TrimSpace(hints.ProviderLabel),
		Provider:         strings.TrimSpace(hints.Provider),
		Platform:         strings.TrimSpace(hints.Platform),
		Capabilities:     normalizeWorkspaceModelCapabilities(hints.Capabilities),
		CapabilitySource: workspaceModelCapabilitySourceExplicit,
		Confidence:       0.95,
	}
	if len(metadata.Capabilities) > 0 {
		return metadata
	}

	metadata.Capabilities = capabilitiesFromWorkspaceModelHints(hints)
	if len(metadata.Capabilities) > 0 {
		metadata.CapabilitySource = workspaceModelCapabilitySourceExplicit
		metadata.Confidence = 0.85
		return metadata
	}

	normalized := strings.ToLower(modelName)
	switch {
	case normalized == "":
		metadata.Capabilities = nil
		metadata.CapabilitySource = workspaceModelCapabilitySourceSafeFallback
		metadata.Confidence = 0
	case isWorkspaceImageGenerationModelName(normalized):
		metadata.Capabilities = []WorkspaceModelCapability{WorkspaceModelCapabilityImageGeneration}
		metadata.CapabilitySource = workspaceModelCapabilitySourceStaticRule
		metadata.Confidence = 0.78
	case isWorkspaceDeepSeekTextModelName(normalized):
		metadata.Capabilities = []WorkspaceModelCapability{WorkspaceModelCapabilityTextChat}
		metadata.CapabilitySource = workspaceModelCapabilitySourceStaticRule
		metadata.Confidence = 0.82
	case isWorkspaceVisionTextModelName(normalized):
		metadata.Capabilities = []WorkspaceModelCapability{WorkspaceModelCapabilityTextChat, WorkspaceModelCapabilityVision}
		metadata.CapabilitySource = workspaceModelCapabilitySourceStaticRule
		metadata.Confidence = 0.72
	default:
		metadata.Capabilities = []WorkspaceModelCapability{WorkspaceModelCapabilityTextChat}
		metadata.CapabilitySource = workspaceModelCapabilitySourceSafeFallback
		metadata.Confidence = 0.35
	}
	return metadata
}

func ApplyWorkspaceModelCapabilityMatch(plan WorkspaceCapabilityPlan, metadata WorkspaceModelCapabilityMetadata) WorkspaceCapabilityPlan {
	matched, reason := WorkspaceModelCapabilitiesMatch(plan.PlannedCapability, metadata)
	plan.ModelCapabilityMatched = matched
	plan.BlockReason = reason
	return plan
}

func WorkspaceModelCapabilitiesMatch(planned WorkspacePlannedCapability, metadata WorkspaceModelCapabilityMetadata) (bool, string) {
	required := workspaceCapabilityRequiredModelCapability(planned)
	if required == "" {
		return false, "planned_capability_unknown"
	}
	if workspaceModelCapabilityListContains(metadata.Capabilities, required) {
		return true, ""
	}
	return false, "selected_model_does_not_support_" + string(planned)
}

func mergeWorkspaceModelCapabilityMetadata(metadata map[string]any, modelMetadata WorkspaceModelCapabilityMetadata, plan WorkspaceCapabilityPlan) map[string]any {
	out := make(map[string]any, len(metadata)+8)
	for key, value := range metadata {
		out[key] = value
	}
	out["selected_model_capabilities"] = workspaceModelCapabilityStrings(modelMetadata.Capabilities)
	out["model_capability_source"] = modelMetadata.CapabilitySource
	out["model_capability_confidence"] = modelMetadata.Confidence
	if modelMetadata.ProviderLabel != "" {
		out["model_provider_label"] = modelMetadata.ProviderLabel
	}
	if modelMetadata.Provider != "" {
		out["model_provider"] = modelMetadata.Provider
	}
	if modelMetadata.Platform != "" {
		out["model_platform"] = modelMetadata.Platform
	}
	if !plan.ModelCapabilityMatched && strings.TrimSpace(plan.BlockReason) != "" {
		out["model_capability_mismatch_reason"] = plan.BlockReason
	}
	return out
}

func workspaceCapabilityRequiredModelCapability(planned WorkspacePlannedCapability) WorkspaceModelCapability {
	switch planned {
	case WorkspacePlannedCapabilityTextChat:
		return WorkspaceModelCapabilityTextChat
	case WorkspacePlannedCapabilityImageGeneration:
		return WorkspaceModelCapabilityImageGeneration
	case WorkspacePlannedCapabilityImageEdit:
		return WorkspaceModelCapabilityImageEdit
	case WorkspacePlannedCapabilityVision:
		return WorkspaceModelCapabilityVision
	case WorkspacePlannedCapabilityFunctionCalling:
		return WorkspaceModelCapabilityFunctionCalling
	case WorkspacePlannedCapabilityTool:
		return WorkspaceModelCapabilityTool
	default:
		return ""
	}
}

func capabilitiesFromWorkspaceModelHints(hints WorkspaceModelCapabilityHints) []WorkspaceModelCapability {
	capabilities := make([]WorkspaceModelCapability, 0, 2)
	for _, tag := range hints.Tags {
		switch strings.ToLower(strings.TrimSpace(tag)) {
		case string(WorkspaceModelCapabilityTextChat), "chat", "text":
			capabilities = append(capabilities, WorkspaceModelCapabilityTextChat)
		case string(WorkspaceModelCapabilityVision), "image_understanding":
			capabilities = append(capabilities, WorkspaceModelCapabilityVision)
		case string(WorkspaceModelCapabilityImageGeneration), "image", "images":
			capabilities = append(capabilities, WorkspaceModelCapabilityImageGeneration)
		case string(WorkspaceModelCapabilityImageEdit):
			capabilities = append(capabilities, WorkspaceModelCapabilityImageEdit)
		case string(WorkspaceModelCapabilityFunctionCalling), "function", "tools":
			capabilities = append(capabilities, WorkspaceModelCapabilityFunctionCalling)
		case string(WorkspaceModelCapabilityTool):
			capabilities = append(capabilities, WorkspaceModelCapabilityTool)
		}
	}
	if hints.SupportsVision {
		capabilities = append(capabilities, WorkspaceModelCapabilityVision)
	}
	if hints.SupportsFunctionCalling {
		capabilities = append(capabilities, WorkspaceModelCapabilityFunctionCalling)
	}
	return normalizeWorkspaceModelCapabilities(capabilities)
}

func normalizeWorkspaceModelCapabilities(capabilities []WorkspaceModelCapability) []WorkspaceModelCapability {
	out := make([]WorkspaceModelCapability, 0, len(capabilities))
	seen := map[WorkspaceModelCapability]struct{}{}
	for _, capability := range capabilities {
		capability = WorkspaceModelCapability(strings.TrimSpace(string(capability)))
		if capability == "" {
			continue
		}
		if _, ok := seen[capability]; ok {
			continue
		}
		seen[capability] = struct{}{}
		out = append(out, capability)
	}
	return out
}

func workspaceModelCapabilityStrings(capabilities []WorkspaceModelCapability) []string {
	out := make([]string, 0, len(capabilities))
	for _, capability := range capabilities {
		if strings.TrimSpace(string(capability)) == "" {
			continue
		}
		out = append(out, string(capability))
	}
	return out
}

func workspaceModelCapabilityListContains(capabilities []WorkspaceModelCapability, target WorkspaceModelCapability) bool {
	for _, capability := range capabilities {
		if capability == target {
			return true
		}
	}
	return false
}

func isWorkspaceDeepSeekTextModelName(model string) bool {
	return strings.Contains(model, "deepseek")
}

func isWorkspaceVisionTextModelName(model string) bool {
	return strings.Contains(model, "gpt-4o") ||
		strings.Contains(model, "gpt-4.1") ||
		strings.Contains(model, "gpt-5") ||
		strings.Contains(model, "claude-3") ||
		strings.Contains(model, "claude-sonnet") ||
		strings.Contains(model, "claude-opus") ||
		strings.Contains(model, "claude-haiku") ||
		strings.Contains(model, "gemini")
}

func isWorkspaceImageGenerationModelName(model string) bool {
	return strings.Contains(model, "workspace-image-fake") ||
		strings.Contains(model, "gpt-image") ||
		strings.Contains(model, "dall-e") ||
		strings.Contains(model, "imagen") ||
		strings.Contains(model, "flux") ||
		strings.Contains(model, "midjourney") ||
		strings.Contains(model, "stable-diffusion") ||
		strings.Contains(model, "sdxl")
}
