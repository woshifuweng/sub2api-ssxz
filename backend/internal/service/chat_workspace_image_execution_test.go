package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type recordingWorkspaceImageExecutionAdapter struct {
	calls  int
	result WorkspaceImageProviderResult
	err    error
}

func (a *recordingWorkspaceImageExecutionAdapter) GenerateImage(_ context.Context, _ WorkspaceImageProviderRequest) (WorkspaceImageProviderResult, error) {
	a.calls++
	return a.result, a.err
}

func testWorkspaceImageExecutionGateConfig() WorkspaceImageExecutionGateConfig {
	return WorkspaceImageExecutionGateConfig{
		Enabled:               true,
		KillSwitch:            false,
		AllowedUserIDs:        []int64{10},
		AllowedModels:         []string{WorkspaceImageProviderFakeModel},
		AllowedProviderLabels: []string{WorkspaceImageProviderFakeLabel},
		MaxRequestsPerTestRun: 1,
	}
}

func testWorkspaceImageExecutionConversation(t *testing.T, svc *ChatWorkspaceService) *WorkspaceConversation {
	t.Helper()
	conversation, err := svc.CreateConversation(context.Background(), 10, WorkspaceCreateConversationInput{})
	require.NoError(t, err)
	return conversation
}

func TestWorkspaceImageExecutionTextChatUsesExistingUnavailableFlow(t *testing.T) {
	repo := newMemoryChatWorkspaceRepo()
	svc := NewChatWorkspaceServiceWithImageExecution(repo, testWorkspaceImageExecutionGateConfig(), WorkspaceImageFakeProviderAdapter{})
	conversation := testWorkspaceImageExecutionConversation(t, svc)

	_, assistantMessage, err := svc.AppendMessageWithAssistantResponse(context.Background(), 10, WorkspaceAppendMessageInput{
		ConversationID: conversation.ID,
		MessageType:    WorkspaceMessageTypeText,
		Role:           WorkspaceRoleUser,
		Content:        "hello workspace",
		Model:          "gpt-5.5",
		Intent:         WorkspaceIntentChat,
	})

	require.NoError(t, err)
	require.Equal(t, WorkspaceMessageTypeText, assistantMessage.MessageType)
	require.Equal(t, WorkspaceIntentChat, assistantMessage.Intent)
	require.Equal(t, WorkspaceAssistantUnavailableContent, assistantMessage.Content)
	require.Equal(t, false, assistantMessage.Metadata["provider_called"])
}

func TestWorkspaceImageExecutionGateDisabledBlocksBeforeAdapter(t *testing.T) {
	adapter := &recordingWorkspaceImageExecutionAdapter{}
	config := testWorkspaceImageExecutionGateConfig()
	config.Enabled = false
	svc := NewChatWorkspaceServiceWithImageExecution(newMemoryChatWorkspaceRepo(), config, adapter)
	conversation := testWorkspaceImageExecutionConversation(t, svc)

	_, assistantMessage, err := svc.AppendMessageWithAssistantResponse(context.Background(), 10, WorkspaceAppendMessageInput{
		ConversationID: conversation.ID,
		MessageType:    WorkspaceMessageTypeText,
		Role:           WorkspaceRoleUser,
		Content:        "generate image of perfume ad",
		Model:          WorkspaceImageProviderFakeModel,
		Intent:         WorkspaceIntentChat,
	})

	require.NoError(t, err)
	require.Equal(t, 0, adapter.calls)
	require.Equal(t, WorkspaceMessageTypeImage, assistantMessage.MessageType)
	require.Equal(t, WorkspaceMessageStatusFailed, assistantMessage.Status)
	require.Equal(t, workspaceImageExecutionErrorDisabled, assistantMessage.Metadata["error_code"])
	require.Equal(t, false, assistantMessage.Metadata["provider_called"])
}

func TestWorkspaceImageExecutionKillSwitchBlocksBeforeAdapter(t *testing.T) {
	adapter := &recordingWorkspaceImageExecutionAdapter{}
	config := testWorkspaceImageExecutionGateConfig()
	config.KillSwitch = true
	svc := NewChatWorkspaceServiceWithImageExecution(newMemoryChatWorkspaceRepo(), config, adapter)
	conversation := testWorkspaceImageExecutionConversation(t, svc)

	_, assistantMessage, err := svc.AppendMessageWithAssistantResponse(context.Background(), 10, WorkspaceAppendMessageInput{
		ConversationID: conversation.ID,
		MessageType:    WorkspaceMessageTypeText,
		Role:           WorkspaceRoleUser,
		Content:        "generate image of perfume ad",
		Model:          WorkspaceImageProviderFakeModel,
		Intent:         WorkspaceIntentChat,
	})

	require.NoError(t, err)
	require.Equal(t, 0, adapter.calls)
	require.Equal(t, workspaceImageExecutionErrorKillSwitch, assistantMessage.Metadata["error_code"])
	require.Equal(t, false, assistantMessage.Metadata["provider_called"])
}

func TestWorkspaceImageExecutionAllowlistBlocksBeforeAdapter(t *testing.T) {
	for name, mutate := range map[string]func(*WorkspaceImageExecutionGateConfig){
		"user_not_allowed": func(config *WorkspaceImageExecutionGateConfig) {
			config.AllowedUserIDs = []int64{20}
		},
		"model_not_allowed": func(config *WorkspaceImageExecutionGateConfig) {
			config.AllowedModels = []string{"other-image-model"}
		},
		"provider_label_not_allowed": func(config *WorkspaceImageExecutionGateConfig) {
			config.AllowedProviderLabels = []string{"other-provider"}
		},
		"missing_cap": func(config *WorkspaceImageExecutionGateConfig) {
			config.MaxRequestsPerTestRun = 0
		},
	} {
		t.Run(name, func(t *testing.T) {
			adapter := &recordingWorkspaceImageExecutionAdapter{}
			config := testWorkspaceImageExecutionGateConfig()
			mutate(&config)
			svc := NewChatWorkspaceServiceWithImageExecution(newMemoryChatWorkspaceRepo(), config, adapter)
			conversation := testWorkspaceImageExecutionConversation(t, svc)

			_, assistantMessage, err := svc.AppendMessageWithAssistantResponse(context.Background(), 10, WorkspaceAppendMessageInput{
				ConversationID: conversation.ID,
				MessageType:    WorkspaceMessageTypeText,
				Role:           WorkspaceRoleUser,
				Content:        "generate image of perfume ad",
				Model:          WorkspaceImageProviderFakeModel,
				Intent:         WorkspaceIntentChat,
			})

			require.NoError(t, err)
			require.Equal(t, 0, adapter.calls)
			require.Equal(t, workspaceImageExecutionErrorNotAllowed, assistantMessage.Metadata["error_code"])
			require.Equal(t, false, assistantMessage.Metadata["provider_called"])
		})
	}
}

func TestWorkspaceImageExecutionModelCapabilityMismatchBlocksBeforeAdapter(t *testing.T) {
	adapter := &recordingWorkspaceImageExecutionAdapter{}
	config := testWorkspaceImageExecutionGateConfig()
	config.AllowedModels = []string{"deepseek-v4-flash"}
	svc := NewChatWorkspaceServiceWithImageExecution(newMemoryChatWorkspaceRepo(), config, adapter)
	conversation := testWorkspaceImageExecutionConversation(t, svc)

	_, assistantMessage, err := svc.AppendMessageWithAssistantResponse(context.Background(), 10, WorkspaceAppendMessageInput{
		ConversationID: conversation.ID,
		MessageType:    WorkspaceMessageTypeText,
		Role:           WorkspaceRoleUser,
		Content:        "generate image of perfume ad",
		Model:          "deepseek-v4-flash",
		Intent:         WorkspaceIntentChat,
	})

	require.NoError(t, err)
	require.Equal(t, 0, adapter.calls)
	require.Equal(t, workspaceImageExecutionErrorCapabilityMismatch, assistantMessage.Metadata["error_code"])
	require.Equal(t, false, assistantMessage.Metadata["provider_called"])
}

func TestWorkspaceImageExecutionMissingImagePlanBlocksBeforeAdapter(t *testing.T) {
	adapter := &recordingWorkspaceImageExecutionAdapter{}
	responder := NewWorkspaceImageExecutionResponder(testWorkspaceImageExecutionGateConfig(), adapter)
	response, err := responder.GenerateAssistantResponse(context.Background(), WorkspaceAssistantResponseInput{
		UserID:         10,
		ConversationID: 1,
		Content:        "generate image of perfume ad",
		Model:          WorkspaceImageProviderFakeModel,
		Intent:         WorkspaceIntentChat,
		Metadata: map[string]any{
			"planned_capability":          string(WorkspacePlannedCapabilityImageGeneration),
			"model_capability_matched":    true,
			"selected_model_capabilities": []string{string(WorkspaceModelCapabilityImageGeneration)},
		},
	})

	require.NoError(t, err)
	require.Equal(t, 0, adapter.calls)
	require.Equal(t, WorkspaceMessageTypeImage, response.MessageType)
	require.Equal(t, WorkspaceMessageStatusFailed, response.Status)
	require.Equal(t, workspaceImageExecutionErrorImagePlanMissing, response.Metadata["error_code"])
}

func TestWorkspaceImageExecutionFakeProviderPersistsAssistantImageMessage(t *testing.T) {
	svc := NewChatWorkspaceServiceWithImageExecution(newMemoryChatWorkspaceRepo(), testWorkspaceImageExecutionGateConfig(), WorkspaceImageFakeProviderAdapter{})
	conversation := testWorkspaceImageExecutionConversation(t, svc)

	userMessage, assistantMessage, err := svc.AppendMessageWithAssistantResponse(context.Background(), 10, WorkspaceAppendMessageInput{
		ConversationID: conversation.ID,
		MessageType:    WorkspaceMessageTypeText,
		Role:           WorkspaceRoleUser,
		Content:        "generate image of perfume ad",
		Model:          WorkspaceImageProviderFakeModel,
		Intent:         WorkspaceIntentChat,
	})

	require.NoError(t, err)
	require.Equal(t, "image_generation", userMessage.Metadata["planned_capability"])
	require.Equal(t, WorkspaceMessageTypeImage, assistantMessage.MessageType)
	require.Equal(t, WorkspaceIntentImageGeneration, assistantMessage.Intent)
	require.Equal(t, WorkspaceMessageStatusCompleted, assistantMessage.Status)
	require.Equal(t, true, assistantMessage.Metadata["provider_called"])
	require.Equal(t, true, assistantMessage.Metadata["image_execution_gate_allowed"])
	require.Equal(t, true, assistantMessage.Metadata["image_execution_fake_provider"])
	require.Equal(t, false, assistantMessage.Metadata["image_task_touched"])
	require.Equal(t, false, assistantMessage.Metadata["asset_upload_touched"])
	require.Equal(t, false, assistantMessage.Metadata["billing_touched"])
	require.Equal(t, WorkspaceImageProviderFakeLabel, assistantMessage.Metadata["provider_label"])
	require.Equal(t, WorkspaceImageProviderFakeModel, assistantMessage.Metadata["model"])
	require.Equal(t, 1, assistantMessage.Metadata["usage_image_count"])
	require.Equal(t, "1024x1024", assistantMessage.Metadata["usage_image_size"])

	messages, err := svc.ListMessages(context.Background(), 10, conversation.ID)
	require.NoError(t, err)
	require.Len(t, messages, 2)
	require.Equal(t, WorkspaceMessageTypeImage, messages[1].MessageType)
	require.Equal(t, "image", messages[1].Metadata["result_type"])

	encoded, err := json.Marshal(messages[1])
	require.NoError(t, err)
	body := strings.ToLower(string(encoded))
	require.NotContains(t, body, "generate image of perfume ad")
	require.NotContains(t, body, "authorization")
	require.NotContains(t, body, "cookie")
	require.NotContains(t, body, "api_key")
	require.NotContains(t, body, "token")
	require.NotContains(t, body, "secret")
}

func TestWorkspaceImageExecutionRequestCapBlocksSecondImageRequest(t *testing.T) {
	adapter := WorkspaceImageFakeProviderAdapter{}
	svc := NewChatWorkspaceServiceWithImageExecution(newMemoryChatWorkspaceRepo(), testWorkspaceImageExecutionGateConfig(), adapter)
	conversation := testWorkspaceImageExecutionConversation(t, svc)
	input := WorkspaceAppendMessageInput{
		ConversationID: conversation.ID,
		MessageType:    WorkspaceMessageTypeText,
		Role:           WorkspaceRoleUser,
		Content:        "generate image of perfume ad",
		Model:          WorkspaceImageProviderFakeModel,
		Intent:         WorkspaceIntentChat,
	}

	_, firstAssistant, err := svc.AppendMessageWithAssistantResponse(context.Background(), 10, input)
	require.NoError(t, err)
	require.Equal(t, WorkspaceMessageStatusCompleted, firstAssistant.Status)

	_, secondAssistant, err := svc.AppendMessageWithAssistantResponse(context.Background(), 10, input)
	require.NoError(t, err)
	require.Equal(t, WorkspaceMessageStatusFailed, secondAssistant.Status)
	require.Equal(t, workspaceImageExecutionErrorRequestCapExceeded, secondAssistant.Metadata["error_code"])
	require.Equal(t, false, secondAssistant.Metadata["provider_called"])
}

func TestWorkspaceImageExecutionUnsafeURLReturnsSafeFailure(t *testing.T) {
	svc := NewChatWorkspaceServiceWithImageExecution(newMemoryChatWorkspaceRepo(), testWorkspaceImageExecutionGateConfig(), WorkspaceImageFakeProviderAdapter{UnsafeURL: "data:image/png;base64,abc"})
	conversation := testWorkspaceImageExecutionConversation(t, svc)

	_, assistantMessage, err := svc.AppendMessageWithAssistantResponse(context.Background(), 10, WorkspaceAppendMessageInput{
		ConversationID: conversation.ID,
		MessageType:    WorkspaceMessageTypeText,
		Role:           WorkspaceRoleUser,
		Content:        "generate image of perfume ad",
		Model:          WorkspaceImageProviderFakeModel,
		Intent:         WorkspaceIntentChat,
	})

	require.NoError(t, err)
	require.Equal(t, WorkspaceMessageStatusFailed, assistantMessage.Status)
	require.Equal(t, "image_result_invalid", assistantMessage.Metadata["error_code"])
	require.Equal(t, false, assistantMessage.Metadata["provider_called"])
	encoded, err := json.Marshal(assistantMessage)
	require.NoError(t, err)
	require.NotContains(t, strings.ToLower(string(encoded)), "data:image")
}

func TestWorkspaceImageExecutionAdapterFailureDoesNotLeakError(t *testing.T) {
	adapter := &recordingWorkspaceImageExecutionAdapter{err: errors.New("provider failed with Authorization bearer token")}
	svc := NewChatWorkspaceServiceWithImageExecution(newMemoryChatWorkspaceRepo(), testWorkspaceImageExecutionGateConfig(), adapter)
	conversation := testWorkspaceImageExecutionConversation(t, svc)

	_, assistantMessage, err := svc.AppendMessageWithAssistantResponse(context.Background(), 10, WorkspaceAppendMessageInput{
		ConversationID: conversation.ID,
		MessageType:    WorkspaceMessageTypeText,
		Role:           WorkspaceRoleUser,
		Content:        "generate image of perfume ad",
		Model:          WorkspaceImageProviderFakeModel,
		Intent:         WorkspaceIntentChat,
	})

	require.NoError(t, err)
	require.Equal(t, 1, adapter.calls)
	require.Equal(t, WorkspaceMessageStatusFailed, assistantMessage.Status)
	require.Equal(t, "image_provider_failed", assistantMessage.Metadata["error_code"])
	encoded, err := json.Marshal(assistantMessage)
	require.NoError(t, err)
	require.NotContains(t, strings.ToLower(string(encoded)), "authorization bearer token")
}
