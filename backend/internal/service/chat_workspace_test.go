package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type memoryChatWorkspaceRepo struct {
	nextConversationID int64
	nextMessageID      int64
	conversations      map[int64]*WorkspaceConversation
	messages           map[int64][]WorkspaceMessage
}

func newMemoryChatWorkspaceRepo() *memoryChatWorkspaceRepo {
	return &memoryChatWorkspaceRepo{
		nextConversationID: 1,
		nextMessageID:      1,
		conversations:      make(map[int64]*WorkspaceConversation),
		messages:           make(map[int64][]WorkspaceMessage),
	}
}

func (r *memoryChatWorkspaceRepo) ListConversations(_ context.Context, userID int64) ([]WorkspaceConversation, error) {
	out := make([]WorkspaceConversation, 0)
	for _, conversation := range r.conversations {
		if conversation.UserID == userID {
			out = append(out, *conversation)
		}
	}
	return out, nil
}

func (r *memoryChatWorkspaceRepo) CreateConversation(_ context.Context, userID int64, title string) (*WorkspaceConversation, error) {
	now := time.Now().UTC()
	conversation := &WorkspaceConversation{
		ID:        r.nextConversationID,
		UserID:    userID,
		Title:     title,
		Status:    "active",
		CreatedAt: now,
		UpdatedAt: now,
	}
	r.nextConversationID++
	r.conversations[conversation.ID] = conversation
	return conversation, nil
}

func (r *memoryChatWorkspaceRepo) GetConversation(_ context.Context, userID, conversationID int64) (*WorkspaceConversation, error) {
	conversation := r.conversations[conversationID]
	if conversation == nil || conversation.UserID != userID {
		return nil, ErrWorkspaceConversationNotFound
	}
	cp := *conversation
	return &cp, nil
}

func (r *memoryChatWorkspaceRepo) ListMessages(_ context.Context, userID, conversationID int64) ([]WorkspaceMessage, error) {
	if _, err := r.GetConversation(context.Background(), userID, conversationID); err != nil {
		return nil, err
	}
	items := r.messages[conversationID]
	out := make([]WorkspaceMessage, len(items))
	copy(out, items)
	return out, nil
}

func (r *memoryChatWorkspaceRepo) AppendMessage(_ context.Context, userID int64, input WorkspaceAppendMessageInput, titleIfEmpty string) (*WorkspaceMessage, error) {
	conversation, err := r.GetConversation(context.Background(), userID, input.ConversationID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	msg := WorkspaceMessage{
		ID:             r.nextMessageID,
		ConversationID: input.ConversationID,
		UserID:         userID,
		MessageType:    input.MessageType,
		Role:           input.Role,
		Content:        input.Content,
		Model:          input.Model,
		Intent:         input.Intent,
		Status:         input.Status,
		Metadata:       input.Metadata,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	r.nextMessageID++
	r.messages[input.ConversationID] = append(r.messages[input.ConversationID], msg)
	conversation.LastMessageAt = &now
	conversation.UpdatedAt = now
	if conversation.Title == "" {
		conversation.Title = titleIfEmpty
		r.conversations[conversation.ID].Title = titleIfEmpty
	}
	r.conversations[conversation.ID].LastMessageAt = &now
	r.conversations[conversation.ID].UpdatedAt = now
	return &msg, nil
}

func TestChatWorkspaceServiceTextLoopPersistsAndScopesToUser(t *testing.T) {
	repo := newMemoryChatWorkspaceRepo()
	svc := NewChatWorkspaceService(repo)

	own, err := svc.CreateConversation(context.Background(), 10, WorkspaceCreateConversationInput{})
	require.NoError(t, err)
	other, err := svc.CreateConversation(context.Background(), 20, WorkspaceCreateConversationInput{Title: "Other"})
	require.NoError(t, err)

	msg, err := svc.AppendMessage(context.Background(), 10, WorkspaceAppendMessageInput{
		ConversationID: own.ID,
		MessageType:    WorkspaceMessageTypeText,
		Role:           WorkspaceRoleUser,
		Content:        "hello workspace",
		Model:          "gpt-5.5",
		Intent:         WorkspaceIntentChat,
	})
	require.NoError(t, err)
	require.Equal(t, own.ID, msg.ConversationID)
	require.Equal(t, int64(10), msg.UserID)

	messages, err := svc.ListMessages(context.Background(), 10, own.ID)
	require.NoError(t, err)
	require.Len(t, messages, 1)
	require.Equal(t, "hello workspace", messages[0].Content)

	conversations, err := svc.ListConversations(context.Background(), 10)
	require.NoError(t, err)
	require.Len(t, conversations, 1)
	require.Equal(t, "hello workspace", conversations[0].Title)

	_, err = svc.GetConversation(context.Background(), 10, other.ID)
	require.ErrorIs(t, err, ErrWorkspaceConversationNotFound)

	_, err = svc.AppendMessage(context.Background(), 10, WorkspaceAppendMessageInput{
		ConversationID: other.ID,
		MessageType:    WorkspaceMessageTypeText,
		Role:           WorkspaceRoleUser,
		Content:        "cross user",
		Model:          "gpt-5.5",
		Intent:         WorkspaceIntentChat,
	})
	require.ErrorIs(t, err, ErrWorkspaceConversationNotFound)
}

func TestChatWorkspaceServiceRejectsInvalidModelIntentAndDisabledCapabilities(t *testing.T) {
	repo := newMemoryChatWorkspaceRepo()
	svc := NewChatWorkspaceService(repo)
	conversation, err := svc.CreateConversation(context.Background(), 10, WorkspaceCreateConversationInput{})
	require.NoError(t, err)

	base := WorkspaceAppendMessageInput{
		ConversationID: conversation.ID,
		MessageType:    WorkspaceMessageTypeText,
		Role:           WorkspaceRoleUser,
		Content:        "hello",
		Model:          "gpt-5.5",
		Intent:         WorkspaceIntentChat,
	}

	invalidModel := base
	invalidModel.Model = "unknown model"
	_, err = svc.AppendMessage(context.Background(), 10, invalidModel)
	require.ErrorIs(t, err, ErrWorkspaceInvalidModel)

	invalidIntent := base
	invalidIntent.Intent = "vision"
	_, err = svc.AppendMessage(context.Background(), 10, invalidIntent)
	require.ErrorIs(t, err, ErrWorkspaceCapabilityDisabled)

	unknownIntent := base
	unknownIntent.Intent = "custom"
	_, err = svc.AppendMessage(context.Background(), 10, unknownIntent)
	require.ErrorIs(t, err, ErrWorkspaceInvalidIntent)
}

func TestChatWorkspaceServiceAllowsDeepSeekStagingTextModel(t *testing.T) {
	repo := newMemoryChatWorkspaceRepo()
	svc := NewChatWorkspaceService(repo)
	conversation, err := svc.CreateConversation(context.Background(), 10, WorkspaceCreateConversationInput{})
	require.NoError(t, err)

	msg, err := svc.AppendMessage(context.Background(), 10, WorkspaceAppendMessageInput{
		ConversationID: conversation.ID,
		MessageType:    WorkspaceMessageTypeText,
		Role:           WorkspaceRoleUser,
		Content:        "hello deepseek staging",
		Model:          "deepseek-v4-flash",
		Intent:         WorkspaceIntentChat,
	})
	require.NoError(t, err)
	require.Equal(t, "deepseek-v4-flash", msg.Model)
	require.Equal(t, WorkspaceIntentChat, msg.Intent)
	require.Equal(t, WorkspaceMessageStatusCompleted, msg.Status)
}

func TestChatWorkspaceServicePersistsCapabilityPlanMetadataWithoutRouting(t *testing.T) {
	repo := newMemoryChatWorkspaceRepo()
	svc := NewChatWorkspaceService(repo)
	conversation, err := svc.CreateConversation(context.Background(), 10, WorkspaceCreateConversationInput{})
	require.NoError(t, err)

	msg, err := svc.AppendMessage(context.Background(), 10, WorkspaceAppendMessageInput{
		ConversationID: conversation.ID,
		MessageType:    WorkspaceMessageTypeText,
		Role:           WorkspaceRoleUser,
		Content:        "\u5e2e\u6211\u751f\u6210\u4e00\u5f20\u9ad8\u7ea7\u611f\u4ea7\u54c1\u56fe",
		Model:          "gpt-5.5",
		Intent:         WorkspaceIntentChat,
		Metadata: map[string]any{
			"client_request_id": "safe-client-request",
		},
	})
	require.NoError(t, err)

	require.Equal(t, WorkspaceMessageTypeText, msg.MessageType)
	require.Equal(t, WorkspaceRoleUser, msg.Role)
	require.Equal(t, WorkspaceIntentChat, msg.Intent)
	require.Equal(t, "safe-client-request", msg.Metadata["client_request_id"])
	require.Equal(t, "image_generation", msg.Metadata["planned_capability"])
	require.Equal(t, WorkspaceCapabilityPlannerVersion, msg.Metadata["planner_version"])
	require.Equal(t, workspaceCapabilityReasonZHImageGenerationKeyword, msg.Metadata["planner_reason"])
	require.Equal(t, "gpt-5.5", msg.Metadata["selected_model"])
	require.Equal(t, false, msg.Metadata["model_capability_matched"])
	require.Equal(t, []string{"text_chat", "vision"}, msg.Metadata["selected_model_capabilities"])
	require.Equal(t, "selected_model_does_not_support_image_generation", msg.Metadata["planner_block_reason"])
	require.Equal(t, "selected_model_does_not_support_image_generation", msg.Metadata["model_capability_mismatch_reason"])
	require.Equal(t, true, msg.Metadata["image_experience_plan_present"])
	require.Equal(t, WorkspaceImageExperienceEnhancerVersion, msg.Metadata["image_experience_enhancer_version"])
	require.Equal(t, "product", msg.Metadata["image_subject_hint"])
	require.Equal(t, "commercial product image", msg.Metadata["image_scene_hint"])
	require.Equal(t, "commercial premium product photography", msg.Metadata["image_style_hint"])
	require.Equal(t, "1:1", msg.Metadata["image_aspect_ratio"])
	require.Equal(t, "commercial", msg.Metadata["image_quality_preset"])
	require.Equal(t, true, msg.Metadata["enhanced_prompt_present"])
	require.Equal(t, true, msg.Metadata["negative_prompt_present"])
	require.NotContains(t, msg.Metadata, "assets")
	require.NotContains(t, msg.Metadata, "provider_called")
	require.NotContains(t, msg.Metadata, "image_task_id")
	require.NotContains(t, msg.Metadata, "enhanced_prompt")
	require.NotContains(t, msg.Metadata, "negative_prompt")

	messages, err := svc.ListMessages(context.Background(), 10, conversation.ID)
	require.NoError(t, err)
	require.Len(t, messages, 1)
	require.Equal(t, "image_generation", messages[0].Metadata["planned_capability"])
	require.Equal(t, WorkspaceIntentChat, messages[0].Intent)
}

func TestChatWorkspaceServicePersistsTextCapabilityPlanMetadata(t *testing.T) {
	repo := newMemoryChatWorkspaceRepo()
	svc := NewChatWorkspaceService(repo)
	conversation, err := svc.CreateConversation(context.Background(), 10, WorkspaceCreateConversationInput{})
	require.NoError(t, err)

	msg, err := svc.AppendMessage(context.Background(), 10, WorkspaceAppendMessageInput{
		ConversationID: conversation.ID,
		MessageType:    WorkspaceMessageTypeText,
		Role:           WorkspaceRoleUser,
		Content:        "\u5e2e\u6211\u603b\u7ed3\u4e00\u4e0b",
		Model:          "gpt-5.5",
		Intent:         WorkspaceIntentChat,
	})
	require.NoError(t, err)

	require.Equal(t, "text_chat", msg.Metadata["planned_capability"])
	require.Equal(t, workspaceCapabilityReasonDefaultTextChat, msg.Metadata["planner_reason"])
	require.Equal(t, true, msg.Metadata["model_capability_matched"])
	require.Equal(t, []string{"text_chat", "vision"}, msg.Metadata["selected_model_capabilities"])
	require.NotContains(t, msg.Metadata, "planner_block_reason")
	require.NotContains(t, msg.Metadata, "image_experience_plan_present")
}

func TestChatWorkspaceServicePersistsModelCapabilityMismatchMetadata(t *testing.T) {
	repo := newMemoryChatWorkspaceRepo()
	svc := NewChatWorkspaceService(repo)
	conversation, err := svc.CreateConversation(context.Background(), 10, WorkspaceCreateConversationInput{})
	require.NoError(t, err)

	msg, err := svc.AppendMessage(context.Background(), 10, WorkspaceAppendMessageInput{
		ConversationID: conversation.ID,
		MessageType:    WorkspaceMessageTypeText,
		Role:           WorkspaceRoleUser,
		Content:        "\u5e2e\u6211\u751f\u6210\u4e00\u5f20\u9ad8\u7ea7\u611f\u4ea7\u54c1\u56fe",
		Model:          "deepseek-v4-flash",
		Intent:         WorkspaceIntentChat,
	})
	require.NoError(t, err)

	require.Equal(t, "image_generation", msg.Metadata["planned_capability"])
	require.Equal(t, "deepseek-v4-flash", msg.Metadata["selected_model"])
	require.Equal(t, []string{"text_chat"}, msg.Metadata["selected_model_capabilities"])
	require.Equal(t, false, msg.Metadata["model_capability_matched"])
	require.Equal(t, "selected_model_does_not_support_image_generation", msg.Metadata["model_capability_mismatch_reason"])
	require.Equal(t, true, msg.Metadata["image_experience_plan_present"])
	require.Equal(t, "1:1", msg.Metadata["image_aspect_ratio"])
	require.NotContains(t, msg.Metadata, "provider_called")
	require.NotContains(t, msg.Metadata, "image_task_id")
}

func TestChatWorkspaceServiceRejectsUnsafePayloadsAndNonTextMessages(t *testing.T) {
	repo := newMemoryChatWorkspaceRepo()
	svc := NewChatWorkspaceService(repo)
	conversation, err := svc.CreateConversation(context.Background(), 10, WorkspaceCreateConversationInput{})
	require.NoError(t, err)

	_, err = svc.AppendMessage(context.Background(), 10, WorkspaceAppendMessageInput{
		ConversationID: conversation.ID,
		MessageType:    WorkspaceMessageTypeText,
		Role:           WorkspaceRoleUser,
		Content:        "data:image/png;base64,abc",
		Model:          "gpt-5.5",
		Intent:         WorkspaceIntentChat,
	})
	require.ErrorIs(t, err, ErrWorkspaceInvalidMessage)

	_, err = svc.AppendMessage(context.Background(), 10, WorkspaceAppendMessageInput{
		ConversationID: conversation.ID,
		MessageType:    WorkspaceMessageTypeText,
		Role:           WorkspaceRoleUser,
		Content:        "hello",
		Model:          "gpt-5.5",
		Intent:         WorkspaceIntentChat,
		Metadata: map[string]any{
			"preview_url": "data:image/png;base64,abc",
		},
	})
	require.ErrorIs(t, err, ErrWorkspaceInvalidMessage)

	_, err = svc.AppendMessage(context.Background(), 10, WorkspaceAppendMessageInput{
		ConversationID: conversation.ID,
		MessageType:    WorkspaceMessageTypeText,
		Role:           WorkspaceRoleUser,
		Content:        "describe this image",
		Model:          "gpt-5.5",
		Intent:         WorkspaceIntentChat,
		Metadata: map[string]any{
			"assets": []any{
				map[string]any{
					"id":  "asset-1",
					"url": "https://cdn.example.test/user-upload.png",
				},
			},
		},
	})
	require.ErrorIs(t, err, ErrWorkspaceAttachmentsDisabled)

	_, err = svc.AppendMessage(context.Background(), 10, WorkspaceAppendMessageInput{
		ConversationID: conversation.ID,
		MessageType:    "attachment",
		Role:           WorkspaceRoleUser,
		Content:        "hello",
		Model:          "gpt-5.5",
		Intent:         WorkspaceIntentChat,
	})
	require.ErrorIs(t, err, ErrWorkspaceInvalidMessage)
}

func TestChatWorkspaceServiceAppendAssistantMessagePersistsUnderCurrentUser(t *testing.T) {
	repo := newMemoryChatWorkspaceRepo()
	svc := NewChatWorkspaceService(repo)
	conversation, err := svc.CreateConversation(context.Background(), 10, WorkspaceCreateConversationInput{})
	require.NoError(t, err)
	other, err := svc.CreateConversation(context.Background(), 20, WorkspaceCreateConversationInput{})
	require.NoError(t, err)

	msg, err := svc.AppendAssistantMessage(context.Background(), 10, WorkspaceAppendAssistantMessageInput{
		ConversationID: conversation.ID,
		Content:        "assistant reply",
		Model:          "gpt-5.5",
		Intent:         WorkspaceIntentChat,
	})
	require.NoError(t, err)
	require.Equal(t, WorkspaceRoleAssistant, msg.Role)
	require.Equal(t, WorkspaceMessageTypeText, msg.MessageType)
	require.Equal(t, WorkspaceMessageStatusCompleted, msg.Status)
	require.Equal(t, int64(10), msg.UserID)

	messages, err := svc.ListMessages(context.Background(), 10, conversation.ID)
	require.NoError(t, err)
	require.Len(t, messages, 1)
	require.Equal(t, "assistant reply", messages[0].Content)

	_, err = svc.AppendAssistantMessage(context.Background(), 10, WorkspaceAppendAssistantMessageInput{
		ConversationID: other.ID,
		Content:        "cross user assistant reply",
		Model:          "gpt-5.5",
		Intent:         WorkspaceIntentChat,
	})
	require.ErrorIs(t, err, ErrWorkspaceConversationNotFound)
}

func TestChatWorkspaceServiceAppendAssistantImageMessagePersistsMetadata(t *testing.T) {
	repo := newMemoryChatWorkspaceRepo()
	svc := NewChatWorkspaceService(repo)
	conversation, err := svc.CreateConversation(context.Background(), 10, WorkspaceCreateConversationInput{})
	require.NoError(t, err)

	msg, err := svc.AppendAssistantMessage(context.Background(), 10, WorkspaceAppendAssistantMessageInput{
		ConversationID: conversation.ID,
		MessageType:    WorkspaceMessageTypeImage,
		Content:        "Generated image is ready.",
		Model:          "gpt-5.5",
		Intent:         WorkspaceIntentImageGeneration,
		Metadata: map[string]any{
			"capability":              WorkspaceIntentImageGeneration,
			"result_type":             "image",
			"status":                  WorkspaceMessageStatusCompleted,
			"enhanced_prompt_present": true,
			"prompt_present":          true,
			"provider_called":         false,
			"assets": []any{
				map[string]any{
					"id":        "asset-1",
					"url":       "https://cdn.example.test/workspace/image-1.png",
					"mime_type": "image/png",
					"width":     float64(1024),
					"height":    float64(1024),
					"provider":  "placeholder-provider",
					"model":     "placeholder-model",
				},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, WorkspaceMessageTypeImage, msg.MessageType)
	require.Equal(t, WorkspaceRoleAssistant, msg.Role)
	require.Equal(t, WorkspaceIntentImageGeneration, msg.Intent)
	require.Equal(t, WorkspaceMessageStatusCompleted, msg.Status)
	require.Equal(t, false, msg.Metadata["provider_called"])

	messages, err := svc.ListMessages(context.Background(), 10, conversation.ID)
	require.NoError(t, err)
	require.Len(t, messages, 1)
	require.Equal(t, WorkspaceMessageTypeImage, messages[0].MessageType)
	require.Equal(t, WorkspaceIntentImageGeneration, messages[0].Intent)
	require.Equal(t, "image", messages[0].Metadata["result_type"])

	encoded, err := json.Marshal(messages[0])
	require.NoError(t, err)
	body := strings.ToLower(string(encoded))
	require.NotContains(t, body, "authorization")
	require.NotContains(t, body, "cookie")
	require.NotContains(t, body, "api_key")
	require.NotContains(t, body, "secret")
}

func TestChatWorkspaceServiceAppendAssistantImageMessageAllowsFailedState(t *testing.T) {
	repo := newMemoryChatWorkspaceRepo()
	svc := NewChatWorkspaceService(repo)
	conversation, err := svc.CreateConversation(context.Background(), 10, WorkspaceCreateConversationInput{})
	require.NoError(t, err)

	msg, err := svc.AppendAssistantMessage(context.Background(), 10, WorkspaceAppendAssistantMessageInput{
		ConversationID: conversation.ID,
		MessageType:    WorkspaceMessageTypeImage,
		Content:        "Image generation failed. Please try again.",
		Model:          "gpt-5.5",
		Intent:         WorkspaceIntentImageGeneration,
		Status:         WorkspaceMessageStatusFailed,
		Metadata: map[string]any{
			"capability":    WorkspaceIntentImageGeneration,
			"result_type":   "image",
			"status":        WorkspaceMessageStatusFailed,
			"error_code":    "provider_unavailable",
			"error_message": "Image generation failed. Please try again.",
			"assets": []any{
				map[string]any{
					"url": "https://cdn.example.test/workspace/failed-placeholder.png",
				},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, WorkspaceMessageStatusFailed, msg.Status)
	require.Equal(t, "provider_unavailable", msg.Metadata["error_code"])
}

func TestChatWorkspaceServiceAppendAssistantImageMessageRejectsUnsafeMetadata(t *testing.T) {
	repo := newMemoryChatWorkspaceRepo()
	svc := NewChatWorkspaceService(repo)
	conversation, err := svc.CreateConversation(context.Background(), 10, WorkspaceCreateConversationInput{})
	require.NoError(t, err)

	_, err = svc.AppendAssistantMessage(context.Background(), 10, WorkspaceAppendAssistantMessageInput{
		ConversationID: conversation.ID,
		MessageType:    WorkspaceMessageTypeImage,
		Content:        "Generated image is ready.",
		Model:          "gpt-5.5",
		Intent:         WorkspaceIntentImageGeneration,
		Metadata: map[string]any{
			"result_type": "image",
			"assets": []any{
				map[string]any{
					"url": "data:image/png;base64,abc",
				},
			},
		},
	})
	require.ErrorIs(t, err, ErrWorkspaceInvalidMessage)
}

func TestChatWorkspaceServiceAppendAssistantMessageRejectsUnsafeInputs(t *testing.T) {
	repo := newMemoryChatWorkspaceRepo()
	svc := NewChatWorkspaceService(repo)
	conversation, err := svc.CreateConversation(context.Background(), 10, WorkspaceCreateConversationInput{})
	require.NoError(t, err)

	_, err = svc.AppendAssistantMessage(context.Background(), 10, WorkspaceAppendAssistantMessageInput{
		ConversationID: conversation.ID,
		Content:        "assistant reply",
		Model:          "unknown model",
		Intent:         WorkspaceIntentChat,
	})
	require.ErrorIs(t, err, ErrWorkspaceInvalidModel)

	_, err = svc.AppendAssistantMessage(context.Background(), 10, WorkspaceAppendAssistantMessageInput{
		ConversationID: conversation.ID,
		Content:        "assistant reply",
		Model:          "gpt-5.5",
		Intent:         "image_generation",
	})
	require.ErrorIs(t, err, ErrWorkspaceCapabilityDisabled)

	_, err = svc.AppendAssistantMessage(context.Background(), 10, WorkspaceAppendAssistantMessageInput{
		ConversationID: conversation.ID,
		Content:        "data:image/png;base64,abc",
		Model:          "gpt-5.5",
		Intent:         WorkspaceIntentChat,
	})
	require.ErrorIs(t, err, ErrWorkspaceInvalidMessage)
}

func TestChatWorkspaceServiceAppendMessageWithAssistantResponsePersistsUnavailableAssistant(t *testing.T) {
	repo := newMemoryChatWorkspaceRepo()
	responder := &recordingWorkspaceAssistantResponder{
		response: WorkspaceAssistantResponse{
			Content: WorkspaceAssistantUnavailableContent,
			Model:   "gpt-5.5",
			Intent:  WorkspaceIntentChat,
			Metadata: map[string]any{
				"status":             "unavailable",
				"placeholder":        true,
				"provider_called":    false,
				"billing_touched":    false,
				"asset_touched":      false,
				"provider_connected": false,
			},
		},
	}
	svc := NewChatWorkspaceServiceWithResponder(repo, responder)
	conversation, err := svc.CreateConversation(context.Background(), 10, WorkspaceCreateConversationInput{})
	require.NoError(t, err)

	userMessage, assistantMessage, err := svc.AppendMessageWithAssistantResponse(context.Background(), 10, WorkspaceAppendMessageInput{
		ConversationID: conversation.ID,
		MessageType:    WorkspaceMessageTypeText,
		Role:           WorkspaceRoleUser,
		Content:        "hello workspace",
		Model:          "gpt-5.5",
		Intent:         WorkspaceIntentChat,
	})
	require.NoError(t, err)
	require.NotNil(t, userMessage)
	require.NotNil(t, assistantMessage)
	require.Equal(t, 1, responder.calls)
	require.Equal(t, int64(10), responder.lastInput.UserID)
	require.Equal(t, conversation.ID, responder.lastInput.ConversationID)
	require.Equal(t, userMessage.ID, responder.lastInput.UserMessage.ID)
	require.Equal(t, WorkspaceRoleAssistant, assistantMessage.Role)
	require.Equal(t, WorkspaceMessageTypeText, assistantMessage.MessageType)
	require.Equal(t, WorkspaceMessageStatusCompleted, assistantMessage.Status)
	require.Equal(t, WorkspaceAssistantUnavailableContent, assistantMessage.Content)
	require.Equal(t, "unavailable", assistantMessage.Metadata["status"])
	require.Equal(t, true, assistantMessage.Metadata["placeholder"])
	require.Equal(t, false, assistantMessage.Metadata["provider_called"])
	require.Equal(t, false, assistantMessage.Metadata["billing_touched"])
	require.Equal(t, false, assistantMessage.Metadata["asset_touched"])

	messages, err := svc.ListMessages(context.Background(), 10, conversation.ID)
	require.NoError(t, err)
	require.Len(t, messages, 2)
	require.Equal(t, WorkspaceRoleUser, messages[0].Role)
	require.Equal(t, WorkspaceRoleAssistant, messages[1].Role)
}

func TestChatWorkspaceServiceAppendMessageWithAssistantResponseDoesNotRunResponderForRejectedInput(t *testing.T) {
	repo := newMemoryChatWorkspaceRepo()
	responder := &recordingWorkspaceAssistantResponder{}
	svc := NewChatWorkspaceServiceWithResponder(repo, responder)
	conversation, err := svc.CreateConversation(context.Background(), 10, WorkspaceCreateConversationInput{})
	require.NoError(t, err)

	_, _, err = svc.AppendMessageWithAssistantResponse(context.Background(), 10, WorkspaceAppendMessageInput{
		ConversationID: conversation.ID,
		MessageType:    WorkspaceMessageTypeText,
		Role:           WorkspaceRoleUser,
		Content:        "hello workspace",
		Model:          "unknown model",
		Intent:         WorkspaceIntentChat,
	})
	require.ErrorIs(t, err, ErrWorkspaceInvalidModel)

	_, _, err = svc.AppendMessageWithAssistantResponse(context.Background(), 10, WorkspaceAppendMessageInput{
		ConversationID: conversation.ID,
		MessageType:    WorkspaceMessageTypeText,
		Role:           WorkspaceRoleUser,
		Content:        "draw image",
		Model:          "gpt-5.5",
		Intent:         "image_generation",
	})
	require.ErrorIs(t, err, ErrWorkspaceCapabilityDisabled)

	_, _, err = svc.AppendMessageWithAssistantResponse(context.Background(), 10, WorkspaceAppendMessageInput{
		ConversationID: conversation.ID,
		MessageType:    WorkspaceMessageTypeText,
		Role:           WorkspaceRoleUser,
		Content:        "hello workspace",
		Model:          "gpt-5.5",
		Intent:         "custom",
	})
	require.ErrorIs(t, err, ErrWorkspaceInvalidIntent)

	require.Zero(t, responder.calls)
	messages, err := svc.ListMessages(context.Background(), 10, conversation.ID)
	require.NoError(t, err)
	require.Empty(t, messages)
}

func TestWorkspaceUnavailableAssistantResponderDoesNotCallProviderOrBilling(t *testing.T) {
	response, err := WorkspaceUnavailableAssistantResponder{}.GenerateAssistantResponse(context.Background(), WorkspaceAssistantResponseInput{
		UserID:         10,
		ConversationID: 1,
		Content:        "hello workspace",
		Model:          "gpt-5.5",
		Intent:         WorkspaceIntentChat,
	})
	require.NoError(t, err)
	require.Equal(t, WorkspaceAssistantUnavailableContent, response.Content)
	require.Equal(t, false, response.Metadata["provider_connected"])
	require.Equal(t, false, response.Metadata["provider_called"])
	require.Equal(t, false, response.Metadata["billing_touched"])
	require.Equal(t, false, response.Metadata["asset_touched"])
	require.Equal(t, WorkspaceProviderNameDisabled, response.Metadata["provider_name"])
	require.Equal(t, "workspace_provider_disabled", response.Metadata["audit_error_code"])
	require.NotEmpty(t, response.Metadata["audit_prompt_hash"])
}

func TestWorkspaceProviderUnavailableAdapterBuildsSafeDiagnostics(t *testing.T) {
	response, err := WorkspaceProviderUnavailableAdapter{}.GenerateWorkspaceResponse(context.Background(), WorkspaceProviderRequest{
		UserID:         10,
		ConversationID: 1,
		UserMessageID:  2,
		Content:        "hello Authorization: Bearer sk-secret access_token=abc cookie=session",
		Model:          "gpt-5.5",
		Intent:         WorkspaceIntentChat,
		Capability:     WorkspaceProviderCapabilityText,
		PromptEnhancement: &UpstreamPromptEnhancementResult{
			PromptHash:           "prompt-hash",
			EnhancerVersion:      "offline-prompt-enhancer-v1",
			BenchmarkSampleID:    "text-strict-json-output",
			SuggestedForProvider: false,
		},
	})
	require.NoError(t, err)
	require.Equal(t, WorkspaceAssistantUnavailableContent, response.Content)
	require.Equal(t, WorkspaceProviderNameDisabled, response.Diagnostics.ProviderName)
	require.Equal(t, "workspace provider adapter is disabled by default", response.Diagnostics.DisabledCapabilityReason)
	require.True(t, response.Diagnostics.PromptEnhancerUsed)
	require.NotNil(t, response.Diagnostics.AuditRecord)
	require.Equal(t, "workspace_provider_disabled", response.Diagnostics.AuditRecord.ErrorCode)
	require.Equal(t, "/workspace-provider-disabled", response.Diagnostics.AuditRecord.EndpointLabel)
	require.Equal(t, "gpt-5.5", response.Diagnostics.AuditRecord.RequestedModel)
	require.Equal(t, "gpt-5.5", response.Diagnostics.AuditRecord.MappedModel)
	require.Empty(t, response.Diagnostics.AuditRecord.UpstreamModel)

	encoded, err := json.Marshal(response)
	require.NoError(t, err)
	body := strings.ToLower(string(encoded))
	require.NotContains(t, body, "sk-secret")
	require.NotContains(t, body, "access_token=abc")
	require.NotContains(t, body, "cookie=session")
	require.NotContains(t, body, "authorization: bearer")
	require.NotContains(t, body, "provider_key")
	require.NotContains(t, body, "base_url")
}

func TestChatWorkspaceServiceProviderAdapterBoundaryPersistsFakeAssistantInTests(t *testing.T) {
	repo := newMemoryChatWorkspaceRepo()
	adapter := &recordingWorkspaceProviderAdapter{
		response: WorkspaceProviderResponse{
			Content: "fake adapter response for tests",
			Model:   "gpt-5.5",
			Intent:  WorkspaceIntentChat,
			Metadata: map[string]any{
				"provider_called": false,
				"fake_adapter":    true,
			},
			Diagnostics: WorkspaceProviderDiagnostics{
				RequestedModel:        "gpt-5.5",
				MappedModel:           "gpt-5.5",
				UpstreamModel:         "gpt-5.5-test-double",
				ProviderName:          "workspace_fake_provider",
				SupportedCapabilities: []WorkspaceProviderCapability{WorkspaceProviderCapabilityText},
			},
		},
	}
	svc := NewChatWorkspaceServiceWithProviderAdapter(repo, adapter)
	conversation, err := svc.CreateConversation(context.Background(), 10, WorkspaceCreateConversationInput{})
	require.NoError(t, err)

	userMessage, assistantMessage, err := svc.AppendMessageWithAssistantResponse(context.Background(), 10, WorkspaceAppendMessageInput{
		ConversationID: conversation.ID,
		MessageType:    WorkspaceMessageTypeText,
		Role:           WorkspaceRoleUser,
		Content:        "hello workspace",
		Model:          "gpt-5.5",
		Intent:         WorkspaceIntentChat,
		Metadata: map[string]any{
			"prompt_enhancement": UpstreamPromptEnhancementResult{
				PromptHash:        "hash",
				BenchmarkSampleID: "text-code-explanation",
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, userMessage)
	require.NotNil(t, assistantMessage)
	require.Equal(t, 1, adapter.calls)
	require.Equal(t, int64(10), adapter.lastInput.UserID)
	require.Equal(t, conversation.ID, adapter.lastInput.ConversationID)
	require.Equal(t, userMessage.ID, adapter.lastInput.UserMessageID)
	require.Equal(t, WorkspaceProviderCapabilityText, adapter.lastInput.Capability)
	require.NotNil(t, adapter.lastInput.PromptEnhancement)
	require.Equal(t, "text-code-explanation", adapter.lastInput.PromptEnhancement.BenchmarkSampleID)
	require.Equal(t, WorkspaceRoleAssistant, assistantMessage.Role)
	require.Equal(t, "fake adapter response for tests", assistantMessage.Content)
	require.Equal(t, "workspace_fake_provider", assistantMessage.Metadata["provider_name"])
	require.Equal(t, "gpt-5.5-test-double", assistantMessage.Metadata["upstream_model"])
	require.Equal(t, false, assistantMessage.Metadata["provider_called"])
}

func TestChatWorkspaceServiceProviderAdapterNotCalledForRejectedInputs(t *testing.T) {
	repo := newMemoryChatWorkspaceRepo()
	adapter := &recordingWorkspaceProviderAdapter{}
	svc := NewChatWorkspaceServiceWithProviderAdapter(repo, adapter)
	conversation, err := svc.CreateConversation(context.Background(), 10, WorkspaceCreateConversationInput{})
	require.NoError(t, err)

	_, _, err = svc.AppendMessageWithAssistantResponse(context.Background(), 10, WorkspaceAppendMessageInput{
		ConversationID: conversation.ID,
		MessageType:    WorkspaceMessageTypeText,
		Role:           WorkspaceRoleUser,
		Content:        "hello workspace",
		Model:          "unknown model",
		Intent:         WorkspaceIntentChat,
	})
	require.ErrorIs(t, err, ErrWorkspaceInvalidModel)

	_, _, err = svc.AppendMessageWithAssistantResponse(context.Background(), 10, WorkspaceAppendMessageInput{
		ConversationID: conversation.ID,
		MessageType:    WorkspaceMessageTypeText,
		Role:           WorkspaceRoleUser,
		Content:        "draw an image",
		Model:          "gpt-5.5",
		Intent:         "image_generation",
	})
	require.ErrorIs(t, err, ErrWorkspaceCapabilityDisabled)

	_, _, err = svc.AppendMessageWithAssistantResponse(context.Background(), 10, WorkspaceAppendMessageInput{
		ConversationID: conversation.ID,
		MessageType:    WorkspaceMessageTypeText,
		Role:           WorkspaceRoleUser,
		Content:        "hello workspace",
		Model:          "gpt-5.5",
		Intent:         "custom",
	})
	require.ErrorIs(t, err, ErrWorkspaceInvalidIntent)
	require.Zero(t, adapter.calls)
}

type recordingWorkspaceAssistantResponder struct {
	calls     int
	lastInput WorkspaceAssistantResponseInput
	response  WorkspaceAssistantResponse
	err       error
}

func (r *recordingWorkspaceAssistantResponder) GenerateAssistantResponse(_ context.Context, input WorkspaceAssistantResponseInput) (WorkspaceAssistantResponse, error) {
	r.calls++
	r.lastInput = input
	if r.err != nil {
		return WorkspaceAssistantResponse{}, r.err
	}
	if r.response.Content != "" {
		return r.response, nil
	}
	return WorkspaceUnavailableAssistantResponder{}.GenerateAssistantResponse(context.Background(), input)
}

type recordingWorkspaceProviderAdapter struct {
	calls     int
	lastInput WorkspaceProviderRequest
	response  WorkspaceProviderResponse
	err       error
}

func (r *recordingWorkspaceProviderAdapter) GenerateWorkspaceResponse(_ context.Context, input WorkspaceProviderRequest) (WorkspaceProviderResponse, error) {
	r.calls++
	r.lastInput = input
	if r.err != nil {
		return WorkspaceProviderResponse{}, r.err
	}
	if r.response.Content != "" {
		return r.response, nil
	}
	return WorkspaceProviderUnavailableAdapter{}.GenerateWorkspaceResponse(context.Background(), input)
}
