package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type recordingWorkspaceSub2APITextBridge struct {
	calls     int
	lastInput WorkspaceSub2APITextBridgeInput
	result    WorkspaceSub2APITextBridgeResult
	err       error
}

func (b *recordingWorkspaceSub2APITextBridge) CompleteWorkspaceText(_ context.Context, input WorkspaceSub2APITextBridgeInput) (WorkspaceSub2APITextBridgeResult, error) {
	b.calls++
	b.lastInput = input
	if b.err != nil {
		return WorkspaceSub2APITextBridgeResult{}, b.err
	}
	if strings.TrimSpace(b.result.Content) != "" {
		return b.result, nil
	}
	return WorkspaceSub2APITextBridgeResult{
		Content:        "STAGING_TEXT_OK",
		Model:          input.Model,
		UpstreamModel:  input.Model,
		ProviderName:   "sub2api-openai-compatible",
		RequestID:      "req-test",
		LatencyMs:      12,
		UsageRecorded:  true,
		BillingManaged: true,
		ProviderCalled: true,
	}, nil
}

type recordingWorkspaceWebSearchService struct {
	calls   int
	lastReq WorkspaceToolRequest
	result  WorkspaceToolResult
	err     error
}

func (s *recordingWorkspaceWebSearchService) SearchWeb(_ context.Context, req WorkspaceToolRequest) (WorkspaceToolResult, error) {
	s.calls++
	s.lastReq = req
	if s.err != nil {
		return s.result, s.err
	}
	return s.result, nil
}

func TestWorkspaceSub2APITextBridgeRunsForRealChannelDeepSeekTextModel(t *testing.T) {
	repo := newMemoryChatWorkspaceRepo()
	resolver := NewWorkspaceSelectedModelChannelCatalogResolver(testWorkspaceSelectedModelCatalogRealImageConfig(), testWorkspaceSelectedModelChannelLister{
		channels: []AvailableChannel{testWorkspaceSelectedModelDeepSeekChannel(10)},
	})
	bridge := &recordingWorkspaceSub2APITextBridge{}
	svc := NewChatWorkspaceServiceWithSub2APITextBridge(repo, bridge, resolver)
	conversation, err := svc.CreateConversation(context.Background(), 1, WorkspaceCreateConversationInput{})
	require.NoError(t, err)

	userMessage, assistantMessage, err := svc.AppendMessageWithAssistantResponse(context.Background(), 1, WorkspaceAppendMessageInput{
		ConversationID:  conversation.ID,
		MessageType:     WorkspaceMessageTypeText,
		Role:            WorkspaceRoleUser,
		Content:         "请只回复：STAGING_TEXT_OK",
		Model:           "deepseek-v4-flash",
		Intent:          WorkspaceIntentChat,
		AllowedGroupIDs: []int64{10},
	})

	require.NoError(t, err)
	require.NotNil(t, userMessage)
	require.NotNil(t, assistantMessage)
	require.Equal(t, 1, bridge.calls)
	require.Equal(t, int64(1), bridge.lastInput.UserID)
	require.Equal(t, conversation.ID, bridge.lastInput.ConversationID)
	require.Equal(t, userMessage.ID, bridge.lastInput.UserMessageID)
	require.Equal(t, "deepseek-v4-flash", bridge.lastInput.Model)
	require.Equal(t, WorkspaceModelCatalogSourceRealChannel, bridge.lastInput.Metadata["model_catalog_source"])
	require.Equal(t, "STAGING_TEXT_OK", assistantMessage.Content)
	require.Equal(t, WorkspaceSub2APITextBridgeName, assistantMessage.Metadata["provider_adapter"])
	require.Equal(t, "sub2api", assistantMessage.Metadata["billing_managed_by"])
	require.Equal(t, "sub2api", assistantMessage.Metadata["provider_routing_managed_by"])
	require.Equal(t, true, assistantMessage.Metadata["usage_recorded"])
	require.Equal(t, true, assistantMessage.Metadata["billing_touched"])
	_, userMessageBills := userMessage.Metadata["billing_touched"]
	require.False(t, userMessageBills)
}

func TestWorkspaceSub2APITextBridgeDoesNotCallWebSearchWhenNotRequested(t *testing.T) {
	repo := newMemoryChatWorkspaceRepo()
	resolver := NewWorkspaceSelectedModelChannelCatalogResolver(testWorkspaceSelectedModelCatalogRealImageConfig(), testWorkspaceSelectedModelChannelLister{
		channels: []AvailableChannel{testWorkspaceSelectedModelDeepSeekChannel(10)},
	})
	bridge := &recordingWorkspaceSub2APITextBridge{}
	webSearch := &recordingWorkspaceWebSearchService{}
	svc := NewChatWorkspaceServiceWithSub2APITextBridgeAndWebSearch(repo, bridge, webSearch, resolver)
	conversation, err := svc.CreateConversation(context.Background(), 1, WorkspaceCreateConversationInput{})
	require.NoError(t, err)

	_, assistantMessage, err := svc.AppendMessageWithAssistantResponse(context.Background(), 1, WorkspaceAppendMessageInput{
		ConversationID:  conversation.ID,
		MessageType:     WorkspaceMessageTypeText,
		Role:            WorkspaceRoleUser,
		Content:         "plain text only",
		Model:           "deepseek-v4-flash",
		Intent:          WorkspaceIntentChat,
		AllowedGroupIDs: []int64{10},
	})

	require.NoError(t, err)
	require.Equal(t, 0, webSearch.calls)
	require.Equal(t, 1, bridge.calls)
	require.Empty(t, bridge.lastInput.SystemMessages)
	require.NotContains(t, assistantMessage.Metadata, workspaceWebSearchCitationsKey)
}

func TestWorkspaceSub2APITextBridgeInjectsWebSearchCitationsIntoBridgeAndMetadata(t *testing.T) {
	repo := newMemoryChatWorkspaceRepo()
	resolver := NewWorkspaceSelectedModelChannelCatalogResolver(testWorkspaceSelectedModelCatalogRealImageConfig(), testWorkspaceSelectedModelChannelLister{
		channels: []AvailableChannel{testWorkspaceSelectedModelDeepSeekChannel(10)},
	})
	bridge := &recordingWorkspaceSub2APITextBridge{}
	webSearch := &recordingWorkspaceWebSearchService{
		result: WorkspaceToolResult{
			Tool:   WorkspaceToolWebSearch,
			Status: WorkspaceToolStatusCompleted,
			WebSearch: &WebSearchResult{
				Summary: "Match schedule from FIFA and ESPN.",
				Citations: []Citation{
					{Index: 1, Title: "FIFA Schedule", Domain: "fifa.com", URL: "https://www.fifa.com/schedule", Snippet: "Opening match details.", RetrievedAt: time.Unix(0, 0).UTC()},
					{Index: 2, Title: "ESPN Fixtures", Domain: "espn.com", URL: "https://www.espn.com/fixtures", Snippet: "Same-day fixtures and kickoff times.", RetrievedAt: time.Unix(0, 0).UTC()},
				},
			},
			Citations: []Citation{
				{Index: 1, Title: "FIFA Schedule", Domain: "fifa.com", URL: "https://www.fifa.com/schedule", Snippet: "Opening match details.", RetrievedAt: time.Unix(0, 0).UTC()},
				{Index: 2, Title: "ESPN Fixtures", Domain: "espn.com", URL: "https://www.espn.com/fixtures", Snippet: "Same-day fixtures and kickoff times.", RetrievedAt: time.Unix(0, 0).UTC()},
			},
			UsageLog: WorkspaceToolUsageLogPayload{
				Tool:         WorkspaceToolWebSearch,
				Provider:     "jina",
				Status:       WorkspaceToolStatusCompleted,
				ResultCount:  2,
				ReadURLCount: 2,
				LatencyMS:    15,
				CreatedAt:    time.Unix(0, 0).UTC(),
			},
		},
	}
	svc := NewChatWorkspaceServiceWithSub2APITextBridgeAndWebSearch(repo, bridge, webSearch, resolver)
	conversation, err := svc.CreateConversation(context.Background(), 1, WorkspaceCreateConversationInput{})
	require.NoError(t, err)

	_, assistantMessage, err := svc.AppendMessageWithAssistantResponse(context.Background(), 1, WorkspaceAppendMessageInput{
		ConversationID:  conversation.ID,
		MessageType:     WorkspaceMessageTypeText,
		Role:            WorkspaceRoleUser,
		Content:         "今天 2026 世界杯有哪些比赛？请给出来源。",
		Model:           "deepseek-v4-flash",
		Intent:          WorkspaceIntentChat,
		Metadata:        map[string]any{workspaceWebSearchRequestedKey: true},
		AllowedGroupIDs: []int64{10},
	})

	require.NoError(t, err)
	require.Equal(t, 1, webSearch.calls)
	require.Equal(t, 1, bridge.calls)
	require.Len(t, bridge.lastInput.SystemMessages, 1)
	require.Contains(t, bridge.lastInput.SystemMessages[0], "[1] FIFA Schedule")
	require.Contains(t, bridge.lastInput.SystemMessages[0], "cite them inline as [1], [2]")
	require.Equal(t, true, assistantMessage.Metadata[workspaceWebSearchRequestedKey])
	require.Equal(t, true, assistantMessage.Metadata[workspaceWebSearchUsedKey])
	require.Equal(t, "jina", assistantMessage.Metadata[workspaceWebSearchProviderKey])
	require.Equal(t, 2, assistantMessage.Metadata[workspaceWebSearchResultCountKey])
	citations, ok := assistantMessage.Metadata[workspaceWebSearchCitationsKey].([]map[string]any)
	require.True(t, ok)
	require.Len(t, citations, 2)
	require.Equal(t, "fifa.com", citations[0]["domain"])
	data, err := json.Marshal(assistantMessage.Metadata)
	require.NoError(t, err)
	require.NotContains(t, strings.ToLower(string(data)), "authorization")
}

func TestWorkspaceSub2APITextBridgeUsesSavedUserMessageMetadataForWebSearchRequest(t *testing.T) {
	bridge := &recordingWorkspaceSub2APITextBridge{}
	webSearch := &recordingWorkspaceWebSearchService{
		result: WorkspaceToolResult{
			Tool:   WorkspaceToolWebSearch,
			Status: WorkspaceToolStatusCompleted,
			WebSearch: &WebSearchResult{
				Summary: "Match schedule from FIFA and ESPN.",
			},
			Citations: []Citation{
				{Index: 1, Title: "FIFA Schedule", Domain: "fifa.com", URL: "https://www.fifa.com/schedule", Snippet: "Opening match details.", RetrievedAt: time.Unix(0, 0).UTC()},
			},
			UsageLog: WorkspaceToolUsageLogPayload{
				Tool:         WorkspaceToolWebSearch,
				Provider:     "jina",
				Status:       WorkspaceToolStatusCompleted,
				ResultCount:  1,
				ReadURLCount: 1,
				LatencyMS:    15,
				CreatedAt:    time.Unix(0, 0).UTC(),
			},
		},
	}
	responder := WorkspaceSub2APITextBridgeResponder{Bridge: bridge, WebSearch: webSearch}

	assistantMessage, err := responder.GenerateAssistantResponse(context.Background(), WorkspaceAssistantResponseInput{
		UserID:         1,
		ConversationID: 16,
		UserMessage: WorkspaceMessage{
			ID:      65,
			Content: "今天 2026 世界杯有哪些比赛？请给出来源。",
			Model:   "deepseek-v4-flash",
			Intent:  WorkspaceIntentChat,
			Status:  WorkspaceMessageStatusCompleted,
			Metadata: map[string]any{
				workspaceWebSearchRequestedKey: true,
				"model_catalog_source":         WorkspaceModelCatalogSourceRealChannel,
				"planned_capability":           string(WorkspacePlannedCapabilityTextChat),
				"pricing_status":               WorkspaceSelectedModelPricingConfigured,
				"model_capability_matched":     true,
			},
			CreatedAt: time.Unix(0, 0).UTC(),
		},
		Content:  "今天 2026 世界杯有哪些比赛？请给出来源。",
		Model:    "deepseek-v4-flash",
		Intent:   WorkspaceIntentChat,
		Metadata: map[string]any{},
	})

	require.NoError(t, err)
	require.Equal(t, 1, webSearch.calls)
	require.Equal(t, 1, bridge.calls)
	require.Len(t, bridge.lastInput.SystemMessages, 1)
	require.Equal(t, true, assistantMessage.Metadata[workspaceWebSearchUsedKey])
	require.Equal(t, 1, assistantMessage.Metadata[workspaceWebSearchResultCountKey])
}

func TestWorkspaceSub2APITextBridgeAcceptsStringTrueWebSearchRequest(t *testing.T) {
	bridge := &recordingWorkspaceSub2APITextBridge{}
	webSearch := &recordingWorkspaceWebSearchService{
		result: WorkspaceToolResult{
			Tool:   WorkspaceToolWebSearch,
			Status: WorkspaceToolStatusCompleted,
			WebSearch: &WebSearchResult{
				Summary: "Match schedule from FIFA.",
			},
			Citations: []Citation{
				{Index: 1, Title: "FIFA Schedule", Domain: "fifa.com", URL: "https://www.fifa.com/schedule", Snippet: "Opening match details.", RetrievedAt: time.Unix(0, 0).UTC()},
			},
			UsageLog: WorkspaceToolUsageLogPayload{
				Tool:         WorkspaceToolWebSearch,
				Provider:     "jina",
				Status:       WorkspaceToolStatusCompleted,
				ResultCount:  1,
				ReadURLCount: 1,
				LatencyMS:    15,
				CreatedAt:    time.Unix(0, 0).UTC(),
			},
		},
	}
	responder := WorkspaceSub2APITextBridgeResponder{Bridge: bridge, WebSearch: webSearch}

	assistantMessage, err := responder.GenerateAssistantResponse(context.Background(), WorkspaceAssistantResponseInput{
		UserID:         1,
		ConversationID: 16,
		UserMessage: WorkspaceMessage{
			ID:      65,
			Content: "今天 2026 世界杯有哪些比赛？请给出来源。",
			Model:   "deepseek-v4-flash",
			Intent:  WorkspaceIntentChat,
			Status:  WorkspaceMessageStatusCompleted,
			Metadata: map[string]any{
				workspaceWebSearchRequestedKey: "true",
				"model_catalog_source":         WorkspaceModelCatalogSourceRealChannel,
				"planned_capability":           string(WorkspacePlannedCapabilityTextChat),
				"pricing_status":               WorkspaceSelectedModelPricingConfigured,
				"model_capability_matched":     true,
			},
			CreatedAt: time.Unix(0, 0).UTC(),
		},
		Content:  "今天 2026 世界杯有哪些比赛？请给出来源。",
		Model:    "deepseek-v4-flash",
		Intent:   WorkspaceIntentChat,
		Metadata: map[string]any{},
	})

	require.NoError(t, err)
	require.Equal(t, 1, webSearch.calls)
	require.Equal(t, 1, bridge.calls)
	require.Equal(t, true, assistantMessage.Metadata[workspaceWebSearchUsedKey])
}

func TestWorkspaceSub2APITextBridgeWebSearchFailureReturnsExplicitUnavailableMessage(t *testing.T) {
	repo := newMemoryChatWorkspaceRepo()
	resolver := NewWorkspaceSelectedModelChannelCatalogResolver(testWorkspaceSelectedModelCatalogRealImageConfig(), testWorkspaceSelectedModelChannelLister{
		channels: []AvailableChannel{testWorkspaceSelectedModelDeepSeekChannel(10)},
	})
	bridge := &recordingWorkspaceSub2APITextBridge{}
	webSearch := &recordingWorkspaceWebSearchService{
		result: WorkspaceToolResult{
			Tool:      WorkspaceToolWebSearch,
			Status:    WorkspaceToolStatusUnavailable,
			ErrorCode: WorkspaceToolErrorProviderUnavailable,
			Message:   "web search provider unavailable",
			UsageLog: WorkspaceToolUsageLogPayload{
				Tool:      WorkspaceToolWebSearch,
				Provider:  "jina",
				Status:    WorkspaceToolStatusUnavailable,
				ErrorCode: WorkspaceToolErrorProviderUnavailable,
			},
		},
		err: ErrWorkspaceToolUnavailable,
	}
	svc := NewChatWorkspaceServiceWithSub2APITextBridgeAndWebSearch(repo, bridge, webSearch, resolver)
	conversation, err := svc.CreateConversation(context.Background(), 1, WorkspaceCreateConversationInput{})
	require.NoError(t, err)

	_, assistantMessage, err := svc.AppendMessageWithAssistantResponse(context.Background(), 1, WorkspaceAppendMessageInput{
		ConversationID:  conversation.ID,
		MessageType:     WorkspaceMessageTypeText,
		Role:            WorkspaceRoleUser,
		Content:         "search for current fixtures",
		Model:           "deepseek-v4-flash",
		Intent:          WorkspaceIntentChat,
		Metadata:        map[string]any{workspaceWebSearchRequestedKey: true},
		AllowedGroupIDs: []int64{10},
	})

	require.NoError(t, err)
	require.Equal(t, 1, webSearch.calls)
	require.Zero(t, bridge.calls)
	require.Equal(t, workspaceWebSearchUnavailableContent, assistantMessage.Content)
	require.Equal(t, false, assistantMessage.Metadata[workspaceWebSearchUsedKey])
	require.Equal(t, WorkspaceToolErrorProviderUnavailable, assistantMessage.Metadata[workspaceWebSearchErrorCodeKey])
	require.Equal(t, false, assistantMessage.Metadata["provider_called"])
	require.Equal(t, 0, assistantMessage.Metadata[workspaceWebSearchResultCountKey])
	citations, ok := assistantMessage.Metadata[workspaceWebSearchCitationsKey].([]map[string]any)
	require.True(t, ok)
	require.Len(t, citations, 0)
}

func TestWorkspaceSub2APITextBridgeWebSearchGateDisabledFailsClosed(t *testing.T) {
	repo := newMemoryChatWorkspaceRepo()
	resolver := NewWorkspaceSelectedModelChannelCatalogResolver(testWorkspaceSelectedModelCatalogRealImageConfig(), testWorkspaceSelectedModelChannelLister{
		channels: []AvailableChannel{testWorkspaceSelectedModelDeepSeekChannel(10)},
	})
	bridge := &recordingWorkspaceSub2APITextBridge{}
	webSearch := &recordingWorkspaceWebSearchService{
		result: WorkspaceToolResult{
			Tool:      WorkspaceToolWebSearch,
			Status:    WorkspaceToolStatusUnavailable,
			ErrorCode: WorkspaceToolErrorKillSwitch,
			Message:   "Web search is unavailable.",
			UsageLog: WorkspaceToolUsageLogPayload{
				Tool:      WorkspaceToolWebSearch,
				Provider:  "jina",
				Status:    WorkspaceToolStatusUnavailable,
				ErrorCode: WorkspaceToolErrorKillSwitch,
			},
		},
		err: ErrWorkspaceToolUnavailable,
	}
	svc := NewChatWorkspaceServiceWithSub2APITextBridgeAndWebSearch(repo, bridge, webSearch, resolver)
	conversation, err := svc.CreateConversation(context.Background(), 1, WorkspaceCreateConversationInput{})
	require.NoError(t, err)

	_, assistantMessage, err := svc.AppendMessageWithAssistantResponse(context.Background(), 1, WorkspaceAppendMessageInput{
		ConversationID:  conversation.ID,
		MessageType:     WorkspaceMessageTypeText,
		Role:            WorkspaceRoleUser,
		Content:         "search under gate",
		Model:           "deepseek-v4-flash",
		Intent:          WorkspaceIntentChat,
		Metadata:        map[string]any{workspaceWebSearchRequestedKey: true},
		AllowedGroupIDs: []int64{10},
	})

	require.NoError(t, err)
	require.Zero(t, bridge.calls)
	require.Equal(t, workspaceWebSearchUnavailableContent, assistantMessage.Content)
	require.Equal(t, WorkspaceToolErrorKillSwitch, assistantMessage.Metadata[workspaceWebSearchErrorCodeKey])
	require.Equal(t, false, assistantMessage.Metadata[workspaceWebSearchUsedKey])
}

func TestWorkspaceSub2APITextBridgeDoesNotClaimBillingWhenGatewayReportsMissingUsage(t *testing.T) {
	repo := newMemoryChatWorkspaceRepo()
	resolver := NewWorkspaceSelectedModelChannelCatalogResolver(testWorkspaceSelectedModelCatalogRealImageConfig(), testWorkspaceSelectedModelChannelLister{
		channels: []AvailableChannel{testWorkspaceSelectedModelDeepSeekChannel(10)},
	})
	bridge := &recordingWorkspaceSub2APITextBridge{result: WorkspaceSub2APITextBridgeResult{
		Content:        "STAGING_TEXT_OK",
		Model:          "deepseek-v4-flash",
		UpstreamModel:  "deepseek-v4-flash",
		ProviderName:   WorkspaceSub2APITextBridgeName,
		RequestID:      "req-missing-usage",
		ProviderCalled: true,
		UsageRecorded:  false,
		BillingManaged: false,
		AdditionalFields: map[string]any{
			"usage_status":   "usage_missing",
			"billing_status": "billing_not_recorded",
		},
	}}
	svc := NewChatWorkspaceServiceWithSub2APITextBridge(repo, bridge, resolver)
	conversation, err := svc.CreateConversation(context.Background(), 1, WorkspaceCreateConversationInput{})
	require.NoError(t, err)

	_, assistantMessage, err := svc.AppendMessageWithAssistantResponse(context.Background(), 1, WorkspaceAppendMessageInput{
		ConversationID:  conversation.ID,
		MessageType:     WorkspaceMessageTypeText,
		Role:            WorkspaceRoleUser,
		Content:         "璇峰彧鍥炲锛歋TAGING_TEXT_OK",
		Model:           "deepseek-v4-flash",
		Intent:          WorkspaceIntentChat,
		AllowedGroupIDs: []int64{10},
	})

	require.NoError(t, err)
	require.Equal(t, 1, bridge.calls)
	require.Equal(t, "STAGING_TEXT_OK", assistantMessage.Content)
	require.Equal(t, true, assistantMessage.Metadata["provider_called"])
	require.Equal(t, false, assistantMessage.Metadata["usage_recorded"])
	require.Equal(t, false, assistantMessage.Metadata["billing_touched"])
	require.Equal(t, "usage_missing", assistantMessage.Metadata["usage_status"])
	require.Equal(t, "billing_not_recorded", assistantMessage.Metadata["billing_status"])
}

func TestWorkspaceSub2APITextBridgeMissingAPIKeyShowsClearMessage(t *testing.T) {
	repo := newMemoryChatWorkspaceRepo()
	resolver := NewWorkspaceSelectedModelChannelCatalogResolver(testWorkspaceSelectedModelCatalogRealImageConfig(), testWorkspaceSelectedModelChannelLister{
		channels: []AvailableChannel{testWorkspaceSelectedModelDeepSeekChannel(10)},
	})
	bridge := &recordingWorkspaceSub2APITextBridge{err: ErrWorkspaceSub2APITextBridgeMissingAPIKey}
	svc := NewChatWorkspaceServiceWithSub2APITextBridge(repo, bridge, resolver)
	conversation, err := svc.CreateConversation(context.Background(), 1, WorkspaceCreateConversationInput{})
	require.NoError(t, err)

	_, assistantMessage, err := svc.AppendMessageWithAssistantResponse(context.Background(), 1, WorkspaceAppendMessageInput{
		ConversationID:  conversation.ID,
		MessageType:     WorkspaceMessageTypeText,
		Role:            WorkspaceRoleUser,
		Content:         "hello",
		Model:           "deepseek-v4-flash",
		Intent:          WorkspaceIntentChat,
		AllowedGroupIDs: []int64{10},
	})

	require.NoError(t, err)
	require.Equal(t, 1, bridge.calls)
	require.Equal(t, WorkspaceSub2APITextBridgeMissingAPIKeyContent, assistantMessage.Content)
	require.Equal(t, "sub2api_api_key_missing", assistantMessage.Metadata["bridge_block_reason"])
	require.Equal(t, false, assistantMessage.Metadata["provider_called"])
	require.Equal(t, false, assistantMessage.Metadata["billing_touched"])
}

func TestWorkspaceSub2APITextBridgeModelNotAllowedShowsClearMessage(t *testing.T) {
	repo := newMemoryChatWorkspaceRepo()
	resolver := NewWorkspaceSelectedModelChannelCatalogResolver(testWorkspaceSelectedModelCatalogRealImageConfig(), testWorkspaceSelectedModelChannelLister{
		channels: []AvailableChannel{testWorkspaceSelectedModelDeepSeekChannel(10)},
	})
	bridge := &recordingWorkspaceSub2APITextBridge{err: ErrWorkspaceSub2APITextBridgeModelNotAllowed}
	svc := NewChatWorkspaceServiceWithSub2APITextBridge(repo, bridge, resolver)
	conversation, err := svc.CreateConversation(context.Background(), 1, WorkspaceCreateConversationInput{})
	require.NoError(t, err)

	_, assistantMessage, err := svc.AppendMessageWithAssistantResponse(context.Background(), 1, WorkspaceAppendMessageInput{
		ConversationID:  conversation.ID,
		MessageType:     WorkspaceMessageTypeText,
		Role:            WorkspaceRoleUser,
		Content:         "hello",
		Model:           "deepseek-v4-flash",
		Intent:          WorkspaceIntentChat,
		AllowedGroupIDs: []int64{10},
	})

	require.NoError(t, err)
	require.Equal(t, 1, bridge.calls)
	require.Equal(t, WorkspaceSub2APITextBridgeModelNotAllowedContent, assistantMessage.Content)
	require.Equal(t, "sub2api_api_key_model_not_allowed", assistantMessage.Metadata["bridge_block_reason"])
	require.Equal(t, false, assistantMessage.Metadata["provider_called"])
	require.Equal(t, false, assistantMessage.Metadata["billing_touched"])
}

func TestWorkspaceSub2APITextBridgeRejectsUnknownModelBeforeBridge(t *testing.T) {
	repo := newMemoryChatWorkspaceRepo()
	resolver := NewWorkspaceSelectedModelChannelCatalogResolver(testWorkspaceSelectedModelCatalogRealImageConfig(), testWorkspaceSelectedModelChannelLister{
		channels: []AvailableChannel{testWorkspaceSelectedModelDeepSeekChannel(10)},
	})
	bridge := &recordingWorkspaceSub2APITextBridge{}
	svc := NewChatWorkspaceServiceWithSub2APITextBridge(repo, bridge, resolver)
	conversation, err := svc.CreateConversation(context.Background(), 1, WorkspaceCreateConversationInput{})
	require.NoError(t, err)

	_, _, err = svc.AppendMessageWithAssistantResponse(context.Background(), 1, WorkspaceAppendMessageInput{
		ConversationID:  conversation.ID,
		MessageType:     WorkspaceMessageTypeText,
		Role:            WorkspaceRoleUser,
		Content:         "hello",
		Model:           "env-only-model",
		Intent:          WorkspaceIntentChat,
		AllowedGroupIDs: []int64{10},
	})

	require.ErrorIs(t, err, ErrWorkspaceInvalidModel)
	require.Zero(t, bridge.calls)
}

func TestWorkspaceSub2APITextBridgeBlocksFakeOrImageModelWithoutCallingBridge(t *testing.T) {
	repo := newMemoryChatWorkspaceRepo()
	resolver := NewWorkspaceSelectedModelChannelCatalogResolver(testWorkspaceSelectedModelCatalogFakeConfig(), testWorkspaceSelectedModelChannelLister{})
	bridge := &recordingWorkspaceSub2APITextBridge{}
	svc := NewChatWorkspaceServiceWithSub2APITextBridge(repo, bridge, resolver)
	conversation, err := svc.CreateConversation(context.Background(), 1, WorkspaceCreateConversationInput{})
	require.NoError(t, err)

	_, assistantMessage, err := svc.AppendMessageWithAssistantResponse(context.Background(), 1, WorkspaceAppendMessageInput{
		ConversationID:  conversation.ID,
		MessageType:     WorkspaceMessageTypeText,
		Role:            WorkspaceRoleUser,
		Content:         "generate image of premium product",
		Model:           WorkspaceImageProviderFakeModel,
		Intent:          WorkspaceIntentChat,
		AllowedGroupIDs: []int64{10},
	})

	require.NoError(t, err)
	require.NotNil(t, assistantMessage)
	require.Zero(t, bridge.calls)
	require.Equal(t, WorkspaceAssistantUnavailableContent, assistantMessage.Content)
	require.NotEqual(t, "", assistantMessage.Metadata["bridge_block_reason"])
	require.Equal(t, false, assistantMessage.Metadata["provider_called"])
	require.Equal(t, false, assistantMessage.Metadata["billing_touched"])
}

func TestWorkspaceSub2APITextBridgeBlocksImageCapabilityWithoutCallingBridge(t *testing.T) {
	repo := newMemoryChatWorkspaceRepo()
	resolver := NewWorkspaceSelectedModelChannelCatalogResolver(testWorkspaceSelectedModelCatalogRealImageConfig(), testWorkspaceSelectedModelChannelLister{
		channels: []AvailableChannel{testWorkspaceSelectedModelAvailableChannel("gpt-image-1", 10, true, WorkspaceModelCapabilityImageGeneration)},
	})
	bridge := &recordingWorkspaceSub2APITextBridge{}
	svc := NewChatWorkspaceServiceWithSub2APITextBridge(repo, bridge, resolver)
	conversation, err := svc.CreateConversation(context.Background(), 1, WorkspaceCreateConversationInput{})
	require.NoError(t, err)

	_, assistantMessage, err := svc.AppendMessageWithAssistantResponse(context.Background(), 1, WorkspaceAppendMessageInput{
		ConversationID:  conversation.ID,
		MessageType:     WorkspaceMessageTypeText,
		Role:            WorkspaceRoleUser,
		Content:         "generate image of premium product",
		Model:           "gpt-image-1",
		Intent:          WorkspaceIntentChat,
		AllowedGroupIDs: []int64{10},
	})

	require.NoError(t, err)
	require.NotNil(t, assistantMessage)
	require.Zero(t, bridge.calls)
	require.Equal(t, WorkspaceAssistantUnavailableContent, assistantMessage.Content)
	require.Equal(t, WorkspaceSub2APITextBridgeBlockReasonNotTextChat, assistantMessage.Metadata["bridge_block_reason"])
}

func TestWorkspaceSub2APITextBridgeFailureDoesNotExposeSecretsOrMarkBilling(t *testing.T) {
	repo := newMemoryChatWorkspaceRepo()
	resolver := NewWorkspaceSelectedModelChannelCatalogResolver(testWorkspaceSelectedModelCatalogRealImageConfig(), testWorkspaceSelectedModelChannelLister{
		channels: []AvailableChannel{testWorkspaceSelectedModelDeepSeekChannel(10)},
	})
	bridge := &recordingWorkspaceSub2APITextBridge{err: errors.New("upstream Authorization Bearer sk-secret cookie=session failed")}
	svc := NewChatWorkspaceServiceWithSub2APITextBridge(repo, bridge, resolver)
	conversation, err := svc.CreateConversation(context.Background(), 1, WorkspaceCreateConversationInput{})
	require.NoError(t, err)

	_, assistantMessage, err := svc.AppendMessageWithAssistantResponse(context.Background(), 1, WorkspaceAppendMessageInput{
		ConversationID:  conversation.ID,
		MessageType:     WorkspaceMessageTypeText,
		Role:            WorkspaceRoleUser,
		Content:         "hello",
		Model:           "deepseek-v4-flash",
		Intent:          WorkspaceIntentChat,
		AllowedGroupIDs: []int64{10},
	})

	require.NoError(t, err)
	require.Equal(t, 1, bridge.calls)
	require.Equal(t, WorkspaceAssistantUnavailableContent, assistantMessage.Content)
	require.Equal(t, "sub2api_chat_completion_failed", assistantMessage.Metadata["bridge_block_reason"])
	require.Equal(t, false, assistantMessage.Metadata["provider_called"])
	require.Equal(t, false, assistantMessage.Metadata["billing_touched"])

	data, err := json.Marshal(assistantMessage.Metadata)
	require.NoError(t, err)
	body := strings.ToLower(string(data))
	require.NotContains(t, body, "sk-secret")
	require.NotContains(t, body, "authorization")
	require.NotContains(t, body, "cookie=session")
}
