package service

import (
	"context"
	"strings"
)

const (
	WorkspaceProviderNameDisabled = "workspace_provider_disabled"
)

type WorkspaceProviderCapability string

const (
	WorkspaceProviderCapabilityText            WorkspaceProviderCapability = "text"
	WorkspaceProviderCapabilityImageGeneration WorkspaceProviderCapability = "image_generation"
	WorkspaceProviderCapabilityImageEdit       WorkspaceProviderCapability = "image_edit"
	WorkspaceProviderCapabilityFileAnalysis    WorkspaceProviderCapability = "file_analysis"
	WorkspaceProviderCapabilityWeb             WorkspaceProviderCapability = "web"
	WorkspaceProviderCapabilityMemory          WorkspaceProviderCapability = "memory"
	WorkspaceProviderCapabilityToolbox         WorkspaceProviderCapability = "toolbox"
)

type WorkspaceProviderDiagnostics struct {
	RequestedModel           string                           `json:"requested_model,omitempty"`
	MappedModel              string                           `json:"mapped_model,omitempty"`
	UpstreamModel            string                           `json:"upstream_model,omitempty"`
	ProviderName             string                           `json:"provider_name,omitempty"`
	ServiceTier              string                           `json:"service_tier,omitempty"`
	SupportedCapabilities    []WorkspaceProviderCapability    `json:"supported_capabilities,omitempty"`
	DisabledCapabilityReason string                           `json:"disabled_capability_reason,omitempty"`
	PromptEnhancerUsed       bool                             `json:"prompt_enhancer_used,omitempty"`
	AuditRecord              *UpstreamQualityAuditRecord      `json:"audit_record,omitempty"`
	PromptEnhancement        *UpstreamPromptEnhancementResult `json:"prompt_enhancement,omitempty"`
}

type WorkspaceProviderRequest struct {
	UserID            int64
	ConversationID    int64
	UserMessageID     int64
	Content           string
	Model             string
	Intent            string
	Metadata          map[string]any
	Capability        WorkspaceProviderCapability
	PromptEnhancement *UpstreamPromptEnhancementResult
}

type WorkspaceProviderResponse struct {
	Content     string
	Model       string
	Intent      string
	Metadata    map[string]any
	Diagnostics WorkspaceProviderDiagnostics
}

type WorkspaceProviderAdapter interface {
	GenerateWorkspaceResponse(ctx context.Context, input WorkspaceProviderRequest) (WorkspaceProviderResponse, error)
}

type WorkspaceProviderUnavailableAdapter struct{}

func (WorkspaceProviderUnavailableAdapter) GenerateWorkspaceResponse(_ context.Context, input WorkspaceProviderRequest) (WorkspaceProviderResponse, error) {
	model := strings.TrimSpace(input.Model)
	intent := normalizeWorkspaceIntent(input.Intent)
	promptEnhanced := input.PromptEnhancement != nil
	auditRecord := BuildUpstreamQualityAuditRecord(UpstreamQualityAuditInput{
		Route:          "/api/v1/chat-workspace/conversations/:id/messages",
		Operation:      UpstreamQualityOperationTextCompletion,
		RequestedModel: model,
		MappedModel:    model,
		ProviderName:   WorkspaceProviderNameDisabled,
		Endpoint:       "/workspace-provider-disabled",
		Status:         "unavailable",
		ErrorCode:      "workspace_provider_disabled",
		Prompt:         input.Content,
		PromptEnhanced: promptEnhanced,
	})
	diagnostics := WorkspaceProviderDiagnostics{
		RequestedModel:           auditRecord.RequestedModel,
		MappedModel:              auditRecord.MappedModel,
		UpstreamModel:            auditRecord.UpstreamModel,
		ProviderName:             auditRecord.ProviderName,
		SupportedCapabilities:    []WorkspaceProviderCapability{WorkspaceProviderCapabilityText},
		DisabledCapabilityReason: "workspace provider adapter is disabled by default",
		PromptEnhancerUsed:       promptEnhanced,
		AuditRecord:              &auditRecord,
		PromptEnhancement:        input.PromptEnhancement,
	}
	return WorkspaceProviderResponse{
		Content:     WorkspaceAssistantUnavailableContent,
		Model:       model,
		Intent:      intent,
		Metadata:    workspaceProviderUnavailableMetadata(diagnostics),
		Diagnostics: diagnostics,
	}, nil
}

type WorkspaceProviderAssistantResponder struct {
	Adapter WorkspaceProviderAdapter
}

func (r WorkspaceProviderAssistantResponder) GenerateAssistantResponse(ctx context.Context, input WorkspaceAssistantResponseInput) (WorkspaceAssistantResponse, error) {
	adapter := r.Adapter
	if adapter == nil {
		adapter = WorkspaceProviderUnavailableAdapter{}
	}
	response, err := adapter.GenerateWorkspaceResponse(ctx, WorkspaceProviderRequest{
		UserID:            input.UserID,
		ConversationID:    input.ConversationID,
		UserMessageID:     input.UserMessage.ID,
		Content:           input.Content,
		Model:             input.Model,
		Intent:            input.Intent,
		Metadata:          input.Metadata,
		Capability:        WorkspaceProviderCapabilityText,
		PromptEnhancement: workspacePromptEnhancementFromMetadata(input.Metadata),
	})
	if err != nil {
		return WorkspaceAssistantResponse{}, err
	}
	return WorkspaceAssistantResponse{
		Content:  response.Content,
		Model:    response.Model,
		Intent:   response.Intent,
		Metadata: mergeWorkspaceProviderMetadata(response.Metadata, response.Diagnostics),
	}, nil
}

func NewChatWorkspaceServiceWithProviderAdapter(repo ChatWorkspaceRepository, adapter WorkspaceProviderAdapter) *ChatWorkspaceService {
	return NewChatWorkspaceServiceWithResponder(repo, WorkspaceProviderAssistantResponder{Adapter: adapter})
}

func workspaceProviderUnavailableMetadata(diagnostics WorkspaceProviderDiagnostics) map[string]any {
	return map[string]any{
		"status":                   "unavailable",
		"placeholder":              true,
		"provider_connected":       false,
		"provider_called":          false,
		"billing_touched":          false,
		"asset_touched":            false,
		"provider_adapter":         "disabled",
		"provider_name":            diagnostics.ProviderName,
		"provider_disabled_reason": diagnostics.DisabledCapabilityReason,
		"requested_model":          diagnostics.RequestedModel,
		"mapped_model":             diagnostics.MappedModel,
		"upstream_model":           diagnostics.UpstreamModel,
		"prompt_enhancer_used":     diagnostics.PromptEnhancerUsed,
	}
}

func mergeWorkspaceProviderMetadata(metadata map[string]any, diagnostics WorkspaceProviderDiagnostics) map[string]any {
	out := make(map[string]any, len(metadata)+8)
	for key, value := range metadata {
		out[key] = value
	}
	if diagnostics.ProviderName != "" {
		out["provider_name"] = diagnostics.ProviderName
	}
	if diagnostics.RequestedModel != "" {
		out["requested_model"] = diagnostics.RequestedModel
	}
	if diagnostics.MappedModel != "" {
		out["mapped_model"] = diagnostics.MappedModel
	}
	if diagnostics.UpstreamModel != "" {
		out["upstream_model"] = diagnostics.UpstreamModel
	}
	if diagnostics.DisabledCapabilityReason != "" {
		out["provider_disabled_reason"] = diagnostics.DisabledCapabilityReason
	}
	if diagnostics.AuditRecord != nil {
		out["audit_prompt_hash"] = diagnostics.AuditRecord.PromptHash
		out["audit_endpoint_label"] = diagnostics.AuditRecord.EndpointLabel
		out["audit_status"] = diagnostics.AuditRecord.Status
		out["audit_error_code"] = diagnostics.AuditRecord.ErrorCode
	}
	out["prompt_enhancer_used"] = diagnostics.PromptEnhancerUsed
	return out
}

func workspacePromptEnhancementFromMetadata(metadata map[string]any) *UpstreamPromptEnhancementResult {
	if metadata == nil {
		return nil
	}
	value, ok := metadata["prompt_enhancement"]
	if !ok {
		return nil
	}
	switch typed := value.(type) {
	case UpstreamPromptEnhancementResult:
		return &typed
	case *UpstreamPromptEnhancementResult:
		return typed
	default:
		return nil
	}
}
