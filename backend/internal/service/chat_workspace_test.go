package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type workspaceRepoStub struct {
	conversation WorkspaceConversation
	messages     []WorkspaceMessage
}

func (r *workspaceRepoStub) ListConversations(_ context.Context, userID int64) ([]WorkspaceConversation, error) {
	if userID != r.conversation.UserID {
		return []WorkspaceConversation{}, nil
	}
	return []WorkspaceConversation{r.conversation}, nil
}

func (r *workspaceRepoStub) CreateConversation(_ context.Context, userID int64, title string) (*WorkspaceConversation, error) {
	r.conversation = WorkspaceConversation{ID: 7, UserID: userID, Title: title, Status: "active", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	return &r.conversation, nil
}

func (r *workspaceRepoStub) GetConversation(_ context.Context, userID, conversationID int64) (*WorkspaceConversation, error) {
	if userID != r.conversation.UserID || conversationID != r.conversation.ID {
		return nil, ErrWorkspaceConversationNotFound
	}
	copy := r.conversation
	return &copy, nil
}

func (r *workspaceRepoStub) ListMessages(ctx context.Context, userID, conversationID int64) ([]WorkspaceMessage, error) {
	if _, err := r.GetConversation(ctx, userID, conversationID); err != nil {
		return nil, err
	}
	return append([]WorkspaceMessage(nil), r.messages...), nil
}

func (r *workspaceRepoStub) AppendMessage(ctx context.Context, userID int64, input WorkspaceAppendMessageInput, title string) (*WorkspaceMessage, error) {
	if _, err := r.GetConversation(ctx, userID, input.ConversationID); err != nil {
		return nil, err
	}
	message := WorkspaceMessage{
		ID: int64(len(r.messages) + 1), ConversationID: input.ConversationID, UserID: userID,
		MessageType: input.MessageType, Role: input.Role, Content: input.Content,
		Model: input.Model, Intent: input.Intent, Status: input.Status, Metadata: input.Metadata,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	r.messages = append(r.messages, message)
	if r.conversation.Title == "" {
		r.conversation.Title = title
	}
	return &message, nil
}

type workspaceResponderStub struct {
	result WorkspaceAssistantResponse
	err    error
}

func (r workspaceResponderStub) GenerateAssistantResponse(context.Context, WorkspaceAssistantResponseInput) (WorkspaceAssistantResponse, error) {
	return r.result, r.err
}

func TestChatWorkspaceServiceAppendsTextConversationForOwner(t *testing.T) {
	repo := &workspaceRepoStub{conversation: WorkspaceConversation{ID: 7, UserID: 42, Status: "active"}}
	svc := NewChatWorkspaceServiceWithResponder(repo, workspaceResponderStub{result: WorkspaceAssistantResponse{Content: "已收到", Model: "gpt-5.4"}})

	userMessage, assistantMessage, err := svc.AppendMessageWithAssistantResponse(context.Background(), 42, WorkspaceAppendMessageInput{
		ConversationID: 7,
		MessageType:    WorkspaceMessageTypeText,
		Role:           WorkspaceRoleUser,
		Content:        "测试一下",
		Model:          "gpt-5.4",
		Intent:         WorkspaceIntentChat,
	})

	require.NoError(t, err)
	require.Equal(t, WorkspaceRoleUser, userMessage.Role)
	require.Equal(t, WorkspaceRoleAssistant, assistantMessage.Role)
	require.Equal(t, "已收到", assistantMessage.Content)
	require.Equal(t, "测试一下", repo.conversation.Title)
	require.Len(t, repo.messages, 2)
}

func TestChatWorkspaceServiceRejectsCrossTenantConversation(t *testing.T) {
	repo := &workspaceRepoStub{conversation: WorkspaceConversation{ID: 7, UserID: 42, Status: "active"}}
	svc := NewChatWorkspaceServiceWithResponder(repo, workspaceResponderStub{})

	_, _, err := svc.AppendMessageWithAssistantResponse(context.Background(), 99, WorkspaceAppendMessageInput{
		ConversationID: 7,
		Content:        "不能访问",
		Model:          "gpt-5.4",
	})

	require.ErrorIs(t, err, ErrWorkspaceConversationNotFound)
	require.Empty(t, repo.messages)
}

func TestChatWorkspaceServiceRejectsAttachmentsBeforeCallingProvider(t *testing.T) {
	repo := &workspaceRepoStub{conversation: WorkspaceConversation{ID: 7, UserID: 42, Status: "active"}}
	svc := NewChatWorkspaceServiceWithResponder(repo, workspaceResponderStub{err: errors.New("must not be called")})

	_, _, err := svc.AppendMessageWithAssistantResponse(context.Background(), 42, WorkspaceAppendMessageInput{
		ConversationID: 7,
		Content:        "带附件",
		Model:          "gpt-5.4",
		Metadata:       map[string]any{"attachments": []any{"secret"}},
	})

	require.ErrorIs(t, err, ErrWorkspaceAttachmentsDisabled)
	require.Empty(t, repo.messages)
}
