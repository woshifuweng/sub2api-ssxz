package service

import (
	"context"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

const (
	WorkspaceSub2APITextBridgeName = "sub2api_chat_completions"

	WorkspaceSub2APITextBridgeBlockReasonNotRealChannel = "selected_model_not_real_channel"
	WorkspaceSub2APITextBridgeBlockReasonNotTextChat    = "planned_capability_not_text_chat"
	WorkspaceSub2APITextBridgeBlockReasonFakeModel      = "selected_model_fake_or_test_only"
)

type WorkspaceSub2APITextBridge interface {
	CompleteWorkspaceText(ctx context.Context, input WorkspaceSub2APITextBridgeInput) (WorkspaceSub2APITextBridgeResult, error)
}

type WorkspaceSub2APITextBridgeInput struct {
	UserID          int64
	AllowedGroupIDs []int64
	ConversationID  int64
	UserMessageID   int64
	Content         string
	Model           string
	Metadata        map[string]any
}

type WorkspaceSub2APITextBridgeResult struct {
	Content          string
	Model            string
	UpstreamModel    string
	ProviderName     string
	RequestID        string
	LatencyMs        int64
	UsageRecorded    bool
	BillingManaged   bool
	ProviderCalled   bool
	AdditionalFields map[string]any
}

type WorkspaceSub2APITextBridgeResponder struct {
	Bridge WorkspaceSub2APITextBridge
}

func NewChatWorkspaceServiceWithSub2APITextBridge(repo ChatWorkspaceRepository, bridge WorkspaceSub2APITextBridge, resolver WorkspaceSelectedModelCatalogResolver) *ChatWorkspaceService {
	return NewChatWorkspaceServiceWithResponderAndModelCatalogResolver(repo, WorkspaceSub2APITextBridgeResponder{Bridge: bridge}, resolver)
}

func ProvideChatWorkspaceServiceWithSub2APITextBridge(repo ChatWorkspaceRepository, cfg *config.Config, channelLister WorkspaceSelectedModelCatalogChannelLister, bridge WorkspaceSub2APITextBridge) *ChatWorkspaceService {
	var resolver WorkspaceSelectedModelCatalogResolver
	if channelLister != nil {
		resolver = NewWorkspaceSelectedModelChannelCatalogResolver(cfg, channelLister)
	}
	if bridge == nil {
		return NewChatWorkspaceServiceWithResponderAndModelCatalogResolver(repo, nil, resolver)
	}
	return NewChatWorkspaceServiceWithSub2APITextBridge(repo, bridge, resolver)
}

func (r WorkspaceSub2APITextBridgeResponder) GenerateAssistantResponse(ctx context.Context, input WorkspaceAssistantResponseInput) (WorkspaceAssistantResponse, error) {
	if blockReason := workspaceSub2APITextBridgeBlockReason(input); blockReason != "" {
		return workspaceSub2APITextBridgeBlockedResponse(input, blockReason), nil
	}
	if r.Bridge == nil {
		return WorkspaceUnavailableAssistantResponder{}.GenerateAssistantResponse(ctx, input)
	}
	started := time.Now()
	result, err := r.Bridge.CompleteWorkspaceText(ctx, WorkspaceSub2APITextBridgeInput{
		UserID:          input.UserID,
		AllowedGroupIDs: cloneWorkspaceInt64Slice(input.AllowedGroupIDs),
		ConversationID:  input.ConversationID,
		UserMessageID:   input.UserMessage.ID,
		Content:         input.Content,
		Model:           input.Model,
		Metadata:        input.Metadata,
	})
	if err != nil {
		return workspaceSub2APITextBridgeFailedResponse(input, err), nil
	}
	content := strings.TrimSpace(result.Content)
	if content == "" {
		return workspaceSub2APITextBridgeFailedResponse(input, nil), nil
	}
	if result.LatencyMs <= 0 {
		result.LatencyMs = time.Since(started).Milliseconds()
	}
	return WorkspaceAssistantResponse{
		Content:     content,
		MessageType: WorkspaceMessageTypeText,
		Model:       firstNonEmptyWorkspaceValue(result.Model, input.Model),
		Intent:      WorkspaceIntentChat,
		Status:      WorkspaceMessageStatusCompleted,
		Metadata:    workspaceSub2APITextBridgeSuccessMetadata(input, result),
	}, nil
}

func workspaceSub2APITextBridgeBlockReason(input WorkspaceAssistantResponseInput) string {
	metadata := input.UserMessage.Metadata
	if workspaceMetadataString(metadata, "model_catalog_source") != WorkspaceModelCatalogSourceRealChannel {
		return WorkspaceSub2APITextBridgeBlockReasonNotRealChannel
	}
	if workspaceMetadataBool(metadata, "model_fake") || workspaceMetadataBool(metadata, "model_test_only") {
		return WorkspaceSub2APITextBridgeBlockReasonFakeModel
	}
	if planned := workspaceMetadataString(metadata, "planned_capability"); planned != "" && planned != string(WorkspacePlannedCapabilityTextChat) {
		return WorkspaceSub2APITextBridgeBlockReasonNotTextChat
	}
	if workspaceMetadataString(metadata, "pricing_status") == WorkspaceSelectedModelPricingMissing {
		return WorkspaceSelectedModelBlockReasonPricingMissing
	}
	if !workspaceMetadataBool(metadata, "model_capability_matched") {
		return WorkspaceSelectedModelBlockReasonCapabilityMismatch
	}
	return ""
}

func workspaceSub2APITextBridgeSuccessMetadata(input WorkspaceAssistantResponseInput, result WorkspaceSub2APITextBridgeResult) map[string]any {
	metadata := map[string]any{
		"status":                       WorkspaceMessageStatusCompleted,
		"placeholder":                  false,
		"provider_called":              result.ProviderCalled,
		"provider_adapter":             WorkspaceSub2APITextBridgeName,
		"provider_name":                firstNonEmptyWorkspaceValue(result.ProviderName, WorkspaceSub2APITextBridgeName),
		"requested_model":              input.Model,
		"mapped_model":                 firstNonEmptyWorkspaceValue(result.Model, input.Model),
		"upstream_model":               firstNonEmptyWorkspaceValue(result.UpstreamModel, result.Model, input.Model),
		"latency_ms":                   result.LatencyMs,
		"usage_recorded":               result.UsageRecorded,
		"billing_managed_by":           "sub2api",
		"billing_touched":              result.BillingManaged,
		"provider_routing_managed_by":  "sub2api",
		"workspace_bridge":             WorkspaceSub2APITextBridgeName,
		"model_catalog_source":         workspaceMetadataString(input.UserMessage.Metadata, "model_catalog_source"),
		"selected_model_capabilities":  input.UserMessage.Metadata["selected_model_capabilities"],
		"model_capability_matched":     workspaceMetadataBool(input.UserMessage.Metadata, "model_capability_matched"),
		"conversation_message_saved":   true,
		"workspace_provider_legacy_on": false,
	}
	if requestID := strings.TrimSpace(result.RequestID); requestID != "" {
		metadata["request_id"] = requestID
	}
	for key, value := range result.AdditionalFields {
		if strings.Contains(strings.ToLower(key), "prompt") {
			continue
		}
		metadata[key] = value
	}
	return metadata
}

func workspaceSub2APITextBridgeBlockedResponse(input WorkspaceAssistantResponseInput, reason string) WorkspaceAssistantResponse {
	metadata := workspaceProviderUnavailableMetadata(WorkspaceProviderDiagnostics{
		RequestedModel:           input.Model,
		MappedModel:              input.Model,
		ProviderName:             WorkspaceSub2APITextBridgeName,
		SupportedCapabilities:    []WorkspaceProviderCapability{WorkspaceProviderCapabilityText},
		DisabledCapabilityReason: reason,
	})
	metadata["provider_adapter"] = WorkspaceSub2APITextBridgeName
	metadata["bridge_block_reason"] = reason
	metadata["provider_called"] = false
	metadata["billing_touched"] = false
	metadata["provider_routing_touched"] = false
	return WorkspaceAssistantResponse{
		Content:     WorkspaceAssistantUnavailableContent,
		MessageType: WorkspaceMessageTypeText,
		Model:       input.Model,
		Intent:      WorkspaceIntentChat,
		Status:      WorkspaceMessageStatusCompleted,
		Metadata:    metadata,
	}
}

func workspaceSub2APITextBridgeFailedResponse(input WorkspaceAssistantResponseInput, err error) WorkspaceAssistantResponse {
	reason := "sub2api_chat_completion_failed"
	if err == nil {
		reason = "sub2api_chat_completion_empty_response"
	}
	return workspaceSub2APITextBridgeBlockedResponse(input, reason)
}
