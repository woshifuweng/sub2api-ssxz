package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

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
