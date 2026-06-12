package service

import (
	"context"
	"strings"
	"sync"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

const (
	workspaceImageExecutionErrorDisabled            = "image_execution_disabled"
	workspaceImageExecutionErrorKillSwitch          = "image_execution_kill_switch"
	workspaceImageExecutionErrorNotAllowed          = "image_execution_not_allowed"
	workspaceImageExecutionErrorRequestCapExceeded  = "image_execution_request_cap_exceeded"
	workspaceImageExecutionErrorCapabilityMismatch  = "image_capability_mismatch"
	workspaceImageExecutionErrorImagePlanMissing    = "image_plan_missing"
	workspaceImageExecutionErrorProviderUnavailable = "image_provider_route_unavailable"
)

type WorkspaceImageExecutionGateConfig struct {
	Enabled               bool
	KillSwitch            bool
	ProviderLabel         string
	AllowedUserIDs        []int64
	AllowedModels         []string
	AllowedProviderLabels []string
	MaxRequestsPerTestRun int
}

type WorkspaceImageExecutionResponder struct {
	GateConfig WorkspaceImageExecutionGateConfig
	Boundary   WorkspaceImageProviderBoundary

	mu       sync.Mutex
	reserved int
}

func NewWorkspaceImageExecutionResponder(config WorkspaceImageExecutionGateConfig, adapter WorkspaceImageProviderAdapter) *WorkspaceImageExecutionResponder {
	return &WorkspaceImageExecutionResponder{
		GateConfig: config,
		Boundary: NewWorkspaceImageProviderBoundary(adapter, WorkspaceImageProviderRouterConfig{
			ProviderLabel: config.ProviderLabel,
			Provider:      config.ProviderLabel,
		}),
	}
}

func NewChatWorkspaceServiceWithImageExecution(repo ChatWorkspaceRepository, config WorkspaceImageExecutionGateConfig, adapter WorkspaceImageProviderAdapter) *ChatWorkspaceService {
	return NewChatWorkspaceServiceWithResponder(repo, NewWorkspaceImageExecutionResponder(config, adapter))
}

type WorkspaceImageExecutionThenFallbackResponder struct {
	ImageResponder WorkspaceAssistantResponder
	Fallback       WorkspaceAssistantResponder
}

func (r WorkspaceImageExecutionThenFallbackResponder) GenerateAssistantResponse(ctx context.Context, input WorkspaceAssistantResponseInput) (WorkspaceAssistantResponse, error) {
	capabilityPlan := workspaceCapabilityPlanFromMetadata(input.Metadata, input.Model)
	if capabilityPlan.PlannedCapability == WorkspacePlannedCapabilityImageGeneration && r.ImageResponder != nil {
		return r.ImageResponder.GenerateAssistantResponse(ctx, input)
	}
	fallback := r.Fallback
	if fallback == nil {
		fallback = WorkspaceUnavailableAssistantResponder{}
	}
	return fallback.GenerateAssistantResponse(ctx, input)
}

func NewWorkspaceImageExecutionResponderFromConfig(cfg *config.Config) *WorkspaceImageExecutionResponder {
	if workspaceImageExecutionRealProviderConfigured(cfg) {
		return NewWorkspaceImageExecutionResponder(workspaceImageRealProviderGateConfigFromConfig(cfg), NewWorkspaceOpenAICompatibleImageProviderAdapter(nil))
	}
	return NewWorkspaceImageExecutionResponder(workspaceImageExecutionGateConfigFromConfig(cfg), WorkspaceImageFakeProviderAdapter{})
}

func workspaceImageExecutionFakeConfigured(cfg *config.Config) bool {
	return cfg != nil && cfg.Workspace.ImageExecution.FakeProviderEnabled
}

func workspaceImageExecutionConfigured(cfg *config.Config) bool {
	return workspaceImageExecutionFakeConfigured(cfg) || workspaceImageExecutionRealProviderConfigured(cfg)
}

func workspaceImageExecutionRealProviderConfigured(cfg *config.Config) bool {
	if cfg == nil {
		return false
	}
	realConfig := cfg.Workspace.ImageRealProvider
	if !realConfig.Enabled || realConfig.KillSwitch {
		return false
	}
	if realConfig.StagingOnly && !isWorkspaceTextProviderNonProductionEnvironment(firstNonEmptyWorkspaceValue(realConfig.Environment, cfg.Log.Environment)) {
		return false
	}
	return strings.TrimSpace(realConfig.ProviderLabel) != ""
}

func workspaceImageExecutionGateConfigFromConfig(cfg *config.Config) WorkspaceImageExecutionGateConfig {
	if cfg == nil {
		return WorkspaceImageExecutionGateConfig{KillSwitch: true}
	}
	imageConfig := cfg.Workspace.ImageExecution
	return WorkspaceImageExecutionGateConfig{
		Enabled:               imageConfig.Enabled && imageConfig.FakeProviderEnabled,
		KillSwitch:            imageConfig.KillSwitch,
		ProviderLabel:         WorkspaceImageProviderFakeLabel,
		AllowedUserIDs:        cloneWorkspaceInt64Slice(imageConfig.AllowedUserIDs),
		AllowedModels:         cloneWorkspaceStringSlice(imageConfig.AllowedModels),
		AllowedProviderLabels: cloneWorkspaceStringSlice(imageConfig.AllowedProviderLabels),
		MaxRequestsPerTestRun: imageConfig.MaxRequestsPerTestRun,
	}
}

func workspaceImageRealProviderGateConfigFromConfig(cfg *config.Config) WorkspaceImageExecutionGateConfig {
	if cfg == nil {
		return WorkspaceImageExecutionGateConfig{KillSwitch: true}
	}
	realConfig := cfg.Workspace.ImageRealProvider
	return WorkspaceImageExecutionGateConfig{
		Enabled:               realConfig.Enabled,
		KillSwitch:            realConfig.KillSwitch,
		ProviderLabel:         strings.TrimSpace(realConfig.ProviderLabel),
		AllowedUserIDs:        cloneWorkspaceInt64Slice(realConfig.AllowedUserIDs),
		AllowedModels:         cloneWorkspaceStringSlice(realConfig.AllowedModels),
		AllowedProviderLabels: cloneWorkspaceStringSlice(realConfig.AllowedProviderLabels),
		MaxRequestsPerTestRun: realConfig.MaxRequestsPerTestRun,
	}
}

func (r *WorkspaceImageExecutionResponder) GenerateAssistantResponse(ctx context.Context, input WorkspaceAssistantResponseInput) (WorkspaceAssistantResponse, error) {
	capabilityPlan := workspaceCapabilityPlanFromMetadata(input.Metadata, input.Model)
	if capabilityPlan.PlannedCapability != WorkspacePlannedCapabilityImageGeneration {
		return WorkspaceUnavailableAssistantResponder{}.GenerateAssistantResponse(ctx, input)
	}

	modelMetadata := workspaceModelCapabilityMetadataFromMetadata(input.Metadata, input.Model)
	imagePlan := workspaceImageExperiencePlanFromMetadata(input.Metadata, input.Model, modelMetadata)
	if code := r.checkGate(input.UserID, input.Model); code != "" {
		return workspaceImageExecutionFailureResponse(r.Boundary.Normalizer, code, "Image generation is unavailable.")
	}
	if !capabilityPlan.ModelCapabilityMatched {
		return workspaceImageExecutionFailureResponse(r.Boundary.Normalizer, workspaceImageExecutionErrorCapabilityMismatch, "Selected model does not support image generation.")
	}
	if !imagePlan.Present {
		return workspaceImageExecutionFailureResponse(r.Boundary.Normalizer, workspaceImageExecutionErrorImagePlanMissing, "Image generation plan is missing.")
	}
	if code := r.reserveTestRun(); code != "" {
		return workspaceImageExecutionFailureResponse(r.Boundary.Normalizer, code, "Image generation request cap was reached.")
	}

	result := r.Boundary.GenerateAssistantImage(ctx, WorkspaceImageProviderBoundaryInput{
		CapabilityPlan:          capabilityPlan,
		ModelCapabilityMetadata: modelMetadata,
		ImageExperiencePlan:     imagePlan,
		RequestID:               workspaceMetadataString(input.Metadata, "client_request_id"),
	})
	if result.Status == WorkspaceMessageStatusFailed && workspaceMetadataString(result.Metadata, "error_code") == "image_provider_unavailable" {
		result.Metadata["error_code"] = workspaceImageExecutionErrorProviderUnavailable
	}
	result.Metadata = mergeWorkspaceImageExecutionMetadata(result.Metadata, result.Status == WorkspaceMessageStatusCompleted)
	return workspaceImageAssistantResponse(result)
}

func (r *WorkspaceImageExecutionResponder) checkGate(userID int64, model string) string {
	config := r.GateConfig
	switch {
	case !config.Enabled:
		return workspaceImageExecutionErrorDisabled
	case config.KillSwitch:
		return workspaceImageExecutionErrorKillSwitch
	case !workspaceInt64ListContains(config.AllowedUserIDs, userID):
		return workspaceImageExecutionErrorNotAllowed
	case !workspaceStringListContains(config.AllowedModels, model):
		return workspaceImageExecutionErrorNotAllowed
	case !workspaceStringListContains(config.AllowedProviderLabels, config.ProviderLabel):
		return workspaceImageExecutionErrorNotAllowed
	case config.MaxRequestsPerTestRun <= 0:
		return workspaceImageExecutionErrorNotAllowed
	}
	return ""
}

func (r *WorkspaceImageExecutionResponder) reserveTestRun() string {
	config := r.GateConfig
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.reserved >= config.MaxRequestsPerTestRun {
		return workspaceImageExecutionErrorRequestCapExceeded
	}
	r.reserved++
	return ""
}

func workspaceImageAssistantResponse(result WorkspaceImageAssistantMessageResult) (WorkspaceAssistantResponse, error) {
	return WorkspaceAssistantResponse{
		Content:     result.Content,
		MessageType: result.MessageType,
		Model:       workspaceMetadataString(result.Metadata, "model"),
		Intent:      result.Intent,
		Status:      result.Status,
		Metadata:    result.Metadata,
	}, nil
}

func workspaceImageExecutionFailureResponse(normalizer WorkspaceImageResultNormalizer, code, message string) (WorkspaceAssistantResponse, error) {
	result := normalizer.NormalizeFailure(code, message)
	result.Metadata = mergeWorkspaceImageExecutionMetadata(result.Metadata, false)
	return workspaceImageAssistantResponse(result)
}

func mergeWorkspaceImageExecutionMetadata(metadata map[string]any, providerCalled bool) map[string]any {
	out := make(map[string]any, len(metadata)+6)
	for key, value := range metadata {
		out[key] = value
	}
	out["provider_called"] = providerCalled
	out["image_execution_gate_allowed"] = providerCalled
	out["image_execution_provider_boundary"] = true
	out["image_execution_fake_provider"] = workspaceMetadataString(metadata, "provider_label") == WorkspaceImageProviderFakeLabel
	out["image_task_touched"] = false
	out["asset_upload_touched"] = false
	out["billing_touched"] = false
	return out
}

func workspaceCapabilityPlanFromMetadata(metadata map[string]any, selectedModel string) WorkspaceCapabilityPlan {
	planned := WorkspacePlannedCapability(workspaceMetadataString(metadata, "planned_capability"))
	if planned == "" {
		planned = WorkspacePlannedCapabilityUnknown
	}
	return WorkspaceCapabilityPlan{
		PlannedCapability:      planned,
		Confidence:             workspaceMetadataFloat(metadata, "planner_confidence"),
		Reason:                 workspaceMetadataString(metadata, "planner_reason"),
		PlannerVersion:         workspaceMetadataString(metadata, "planner_version"),
		SelectedModel:          strings.TrimSpace(selectedModel),
		ModelCapabilityMatched: workspaceMetadataBool(metadata, "model_capability_matched"),
		BlockReason:            firstNonEmptyWorkspaceValue(workspaceMetadataString(metadata, "planner_block_reason"), workspaceMetadataString(metadata, "model_capability_mismatch_reason")),
	}
}

func workspaceModelCapabilityMetadataFromMetadata(metadata map[string]any, model string) WorkspaceModelCapabilityMetadata {
	return WorkspaceModelCapabilityMetadata{
		ModelName:        strings.TrimSpace(model),
		ProviderLabel:    workspaceMetadataString(metadata, "model_provider_label"),
		Provider:         workspaceMetadataString(metadata, "model_provider"),
		Platform:         workspaceMetadataString(metadata, "model_platform"),
		Capabilities:     workspaceModelCapabilitiesFromMetadata(metadata["selected_model_capabilities"]),
		CapabilitySource: workspaceMetadataString(metadata, "model_capability_source"),
		Confidence:       workspaceMetadataFloat(metadata, "model_capability_confidence"),
	}
}

func workspaceImageExperiencePlanFromMetadata(metadata map[string]any, model string, modelMetadata WorkspaceModelCapabilityMetadata) WorkspaceImageExperiencePlan {
	return WorkspaceImageExperiencePlan{
		Present:                workspaceMetadataBool(metadata, "image_experience_plan_present"),
		OriginalPromptPresent:  workspaceMetadataBool(metadata, "original_prompt_present"),
		EnhancedPromptPresent:  workspaceMetadataBool(metadata, "enhanced_prompt_present"),
		NegativePromptPresent:  workspaceMetadataBool(metadata, "negative_prompt_present"),
		SubjectHint:            workspaceMetadataString(metadata, "image_subject_hint"),
		SceneHint:              workspaceMetadataString(metadata, "image_scene_hint"),
		StyleHint:              workspaceMetadataString(metadata, "image_style_hint"),
		AspectRatio:            workspaceMetadataString(metadata, "image_aspect_ratio"),
		QualityPreset:          workspaceMetadataString(metadata, "image_quality_preset"),
		EnhancerVersion:        workspaceMetadataString(metadata, "image_experience_enhancer_version"),
		Confidence:             workspaceMetadataFloat(metadata, "image_experience_confidence"),
		Reason:                 workspaceMetadataString(metadata, "image_experience_reason"),
		SelectedModel:          strings.TrimSpace(model),
		ModelCapabilityMatched: workspaceModelCapabilityListContains(modelMetadata.Capabilities, WorkspaceModelCapabilityImageGeneration),
		ModelCapabilityMismatch: workspaceMetadataString(metadata,
			"model_capability_mismatch_reason"),
	}
}

func workspaceModelCapabilitiesFromMetadata(value any) []WorkspaceModelCapability {
	switch items := value.(type) {
	case []string:
		out := make([]WorkspaceModelCapability, 0, len(items))
		for _, item := range items {
			out = append(out, WorkspaceModelCapability(strings.TrimSpace(item)))
		}
		return normalizeWorkspaceModelCapabilities(out)
	case []any:
		out := make([]WorkspaceModelCapability, 0, len(items))
		for _, item := range items {
			if value, ok := item.(string); ok {
				out = append(out, WorkspaceModelCapability(strings.TrimSpace(value)))
			}
		}
		return normalizeWorkspaceModelCapabilities(out)
	default:
		return nil
	}
}

func workspaceMetadataBool(metadata map[string]any, key string) bool {
	value, ok := metadata[key]
	if !ok {
		return false
	}
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return strings.EqualFold(strings.TrimSpace(v), "true")
	default:
		return false
	}
}

func workspaceMetadataFloat(metadata map[string]any, key string) float64 {
	value, ok := metadata[key]
	if !ok {
		return 0
	}
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	default:
		return 0
	}
}

func workspaceInt64ListContains(values []int64, target int64) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func workspaceStringListContains(values []string, target string) bool {
	target = strings.TrimSpace(target)
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), target) {
			return true
		}
	}
	return false
}
