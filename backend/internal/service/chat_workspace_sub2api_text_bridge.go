package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

const (
	WorkspaceSub2APITextBridgeName = "sub2api_chat_completions"

	WorkspaceSub2APITextBridgeBlockReasonNotRealChannel = "selected_model_not_real_channel"
	WorkspaceSub2APITextBridgeBlockReasonNotTextChat    = "planned_capability_not_text_chat"
	WorkspaceSub2APITextBridgeBlockReasonFakeModel      = "selected_model_fake_or_test_only"

	WorkspaceSub2APITextBridgeMissingAPIKeyContent   = "当前账户没有可用 API Key，请先在开发者 API 中创建或启用 API Key 后再使用工作台。"
	WorkspaceSub2APITextBridgeModelNotAllowedContent = "当前 API Key 不允许使用所选模型，请检查开发者 API Key 的模型权限。"
)

var (
	ErrWorkspaceSub2APITextBridgeMissingAPIKey   = errors.New("workspace sub2api text bridge missing usable api key")
	ErrWorkspaceSub2APITextBridgeModelNotAllowed = errors.New("workspace sub2api text bridge api key model not allowed")
)

type WorkspaceSub2APITextBridge interface {
	CompleteWorkspaceText(ctx context.Context, input WorkspaceSub2APITextBridgeInput) (WorkspaceSub2APITextBridgeResult, error)
}

type WorkspaceWebSearchService interface {
	SearchWeb(ctx context.Context, req WorkspaceToolRequest) (WorkspaceToolResult, error)
}

type WorkspaceSub2APITextBridgeInput struct {
	UserID          int64
	AllowedGroupIDs []int64
	ConversationID  int64
	UserMessageID   int64
	Content         string
	Model           string
	SystemMessages  []string
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
	Bridge    WorkspaceSub2APITextBridge
	WebSearch WorkspaceWebSearchService
}

func NewChatWorkspaceServiceWithSub2APITextBridge(repo ChatWorkspaceRepository, bridge WorkspaceSub2APITextBridge, resolver WorkspaceSelectedModelCatalogResolver) *ChatWorkspaceService {
	return NewChatWorkspaceServiceWithResponderAndModelCatalogResolver(repo, WorkspaceSub2APITextBridgeResponder{Bridge: bridge}, resolver)
}

func NewChatWorkspaceServiceWithSub2APITextBridgeAndWebSearch(repo ChatWorkspaceRepository, bridge WorkspaceSub2APITextBridge, webSearch WorkspaceWebSearchService, resolver WorkspaceSelectedModelCatalogResolver) *ChatWorkspaceService {
	return NewChatWorkspaceServiceWithResponderAndModelCatalogResolver(repo, WorkspaceSub2APITextBridgeResponder{
		Bridge:    bridge,
		WebSearch: webSearch,
	}, resolver)
}

func ProvideChatWorkspaceServiceWithSub2APITextBridge(repo ChatWorkspaceRepository, cfg *config.Config, channelLister WorkspaceSelectedModelCatalogChannelLister, bridge WorkspaceSub2APITextBridge, webSearch WorkspaceWebSearchService) *ChatWorkspaceService {
	var resolver WorkspaceSelectedModelCatalogResolver
	if channelLister != nil {
		resolver = NewWorkspaceSelectedModelChannelCatalogResolver(cfg, channelLister)
	}
	if bridge == nil {
		return NewChatWorkspaceServiceWithResponderAndModelCatalogResolver(repo, nil, resolver)
	}
	return NewChatWorkspaceServiceWithSub2APITextBridgeAndWebSearch(repo, bridge, webSearch, resolver)
}

func (r WorkspaceSub2APITextBridgeResponder) GenerateAssistantResponse(ctx context.Context, input WorkspaceAssistantResponseInput) (WorkspaceAssistantResponse, error) {
	if blockReason := workspaceSub2APITextBridgeBlockReason(input); blockReason != "" {
		return workspaceSub2APITextBridgeBlockedResponse(input, blockReason), nil
	}
	if r.Bridge == nil {
		return WorkspaceUnavailableAssistantResponder{}.GenerateAssistantResponse(ctx, input)
	}

	systemMessages, searchMetadata, blockedResponse, err := r.prepareWebSearch(ctx, input)
	if err != nil {
		return WorkspaceAssistantResponse{}, err
	}
	if blockedResponse != nil {
		return *blockedResponse, nil
	}
	if len(searchMetadata) > 0 {
		input.Metadata = workspaceCloneMetadata(input.Metadata)
		mergeWorkspaceMetadata(input.Metadata, searchMetadata)
		input.UserMessage.Metadata = workspaceCloneMetadata(input.UserMessage.Metadata)
		mergeWorkspaceMetadata(input.UserMessage.Metadata, searchMetadata)
	}
	started := time.Now()
	result, err := r.Bridge.CompleteWorkspaceText(ctx, WorkspaceSub2APITextBridgeInput{
		UserID:          input.UserID,
		AllowedGroupIDs: cloneWorkspaceInt64Slice(input.AllowedGroupIDs),
		ConversationID:  input.ConversationID,
		UserMessageID:   input.UserMessage.ID,
		Content:         input.Content,
		Model:           input.Model,
		SystemMessages:  systemMessages,
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
	mergeWorkspaceMetadata(metadata, workspaceAssistantWebSearchMetadata(input.Metadata))
	return metadata
}

func workspaceSub2APITextBridgeBlockedResponse(input WorkspaceAssistantResponseInput, reason string) WorkspaceAssistantResponse {
	return workspaceSub2APITextBridgeBlockedResponseWithContent(input, reason, WorkspaceAssistantUnavailableContent)
}

func workspaceSub2APITextBridgeBlockedResponseWithContent(input WorkspaceAssistantResponseInput, reason, content string) WorkspaceAssistantResponse {
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
	mergeWorkspaceMetadata(metadata, workspaceAssistantWebSearchMetadata(input.Metadata))
	return WorkspaceAssistantResponse{
		Content:     firstNonEmptyWorkspaceValue(content, WorkspaceAssistantUnavailableContent),
		MessageType: WorkspaceMessageTypeText,
		Model:       input.Model,
		Intent:      WorkspaceIntentChat,
		Status:      WorkspaceMessageStatusCompleted,
		Metadata:    metadata,
	}
}

func workspaceAssistantWebSearchMetadata(metadata map[string]any) map[string]any {
	if len(metadata) == 0 || !workspaceMetadataBool(metadata, workspaceWebSearchRequestedKey) {
		return nil
	}
	out := map[string]any{
		workspaceWebSearchRequestedKey:   metadata[workspaceWebSearchRequestedKey],
		workspaceWebSearchUsedKey:        metadata[workspaceWebSearchUsedKey],
		workspaceWebSearchStatusKey:      metadata[workspaceWebSearchStatusKey],
		workspaceWebSearchProviderKey:    metadata[workspaceWebSearchProviderKey],
		workspaceWebSearchErrorCodeKey:   metadata[workspaceWebSearchErrorCodeKey],
		workspaceWebSearchSummaryKey:     metadata[workspaceWebSearchSummaryKey],
		workspaceWebSearchCitationsKey:   metadata[workspaceWebSearchCitationsKey],
		workspaceWebSearchResultCountKey: metadata[workspaceWebSearchResultCountKey],
		workspaceWebSearchToolLogKey:     metadata[workspaceWebSearchToolLogKey],
	}
	return out
}

func workspaceSub2APITextBridgeFailedResponse(input WorkspaceAssistantResponseInput, err error) WorkspaceAssistantResponse {
	reason := "sub2api_chat_completion_failed"
	content := WorkspaceAssistantUnavailableContent
	if err == nil {
		reason = "sub2api_chat_completion_empty_response"
	} else if errors.Is(err, ErrWorkspaceSub2APITextBridgeMissingAPIKey) {
		reason = "sub2api_api_key_missing"
		content = WorkspaceSub2APITextBridgeMissingAPIKeyContent
	} else if errors.Is(err, ErrWorkspaceSub2APITextBridgeModelNotAllowed) {
		reason = "sub2api_api_key_model_not_allowed"
		content = WorkspaceSub2APITextBridgeModelNotAllowedContent
	}
	return workspaceSub2APITextBridgeBlockedResponseWithContent(input, reason, content)
}
