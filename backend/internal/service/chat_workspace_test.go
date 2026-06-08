package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type fakeChatWorkspaceRepo struct {
	conversations map[int64]*ChatConversation
	messages      map[int64][]*ChatMessage
	assets        map[int64]*ChatAsset
	tasks         map[int64]*ChatImageTask
	nextID        int64
}

func newFakeChatWorkspaceRepo() *fakeChatWorkspaceRepo {
	return &fakeChatWorkspaceRepo{
		conversations: map[int64]*ChatConversation{},
		messages:      map[int64][]*ChatMessage{},
		assets:        map[int64]*ChatAsset{},
		tasks:         map[int64]*ChatImageTask{},
		nextID:        1,
	}
}

func (r *fakeChatWorkspaceRepo) next() int64 {
	id := r.nextID
	r.nextID++
	return id
}

func (r *fakeChatWorkspaceRepo) CreateConversation(_ context.Context, conversation *ChatConversation) error {
	conversation.ID = r.next()
	conversation.CreatedAt = time.Now()
	conversation.UpdatedAt = conversation.CreatedAt
	r.conversations[conversation.ID] = cloneConversation(conversation)
	return nil
}

func (r *fakeChatWorkspaceRepo) ListConversations(_ context.Context, userID int64) ([]*ChatConversation, error) {
	var out []*ChatConversation
	for _, item := range r.conversations {
		if item.UserID == userID {
			out = append(out, cloneConversation(item))
		}
	}
	return out, nil
}

func (r *fakeChatWorkspaceRepo) GetConversation(_ context.Context, userID, conversationID int64) (*ChatConversation, error) {
	item := r.conversations[conversationID]
	if item == nil || item.UserID != userID {
		return nil, ErrChatWorkspaceConversationNotFound
	}
	return cloneConversation(item), nil
}

func (r *fakeChatWorkspaceRepo) CreateMessage(_ context.Context, message *ChatMessage) error {
	message.ID = r.next()
	message.CreatedAt = time.Now()
	message.UpdatedAt = message.CreatedAt
	r.messages[message.ConversationID] = append(r.messages[message.ConversationID], cloneMessage(message))
	conversation := r.conversations[message.ConversationID]
	if conversation != nil {
		conversation.LastMessageAt = &message.CreatedAt
		conversation.UpdatedAt = message.CreatedAt
	}
	return nil
}

func (r *fakeChatWorkspaceRepo) ListMessages(_ context.Context, userID, conversationID int64) ([]*ChatMessage, error) {
	if _, err := r.GetConversation(context.Background(), userID, conversationID); err != nil {
		return nil, err
	}
	var out []*ChatMessage
	for _, item := range r.messages[conversationID] {
		out = append(out, cloneMessage(item))
	}
	return out, nil
}

func (r *fakeChatWorkspaceRepo) GetMessage(_ context.Context, userID, messageID int64) (*ChatMessage, error) {
	for _, items := range r.messages {
		for _, item := range items {
			if item.ID == messageID && item.UserID == userID {
				return cloneMessage(item), nil
			}
		}
	}
	return nil, ErrChatWorkspaceMessageNotFound
}

func (r *fakeChatWorkspaceRepo) CreateAsset(_ context.Context, asset *ChatAsset) error {
	asset.ID = r.next()
	asset.CreatedAt = time.Now()
	asset.UpdatedAt = asset.CreatedAt
	r.assets[asset.ID] = cloneAsset(asset)
	return nil
}

func (r *fakeChatWorkspaceRepo) GetAsset(_ context.Context, userID, assetID int64) (*ChatAsset, error) {
	item := r.assets[assetID]
	if item == nil || item.UserID != userID {
		return nil, ErrChatWorkspaceAssetNotFound
	}
	return cloneAsset(item), nil
}

func (r *fakeChatWorkspaceRepo) CreateImageTaskWithMessage(_ context.Context, task *ChatImageTask, message *ChatMessage) error {
	task.ID = r.next()
	message.ID = r.next()
	task.MessageID = &message.ID
	message.TaskID = &task.ID
	task.CreatedAt = time.Now()
	task.UpdatedAt = task.CreatedAt
	message.CreatedAt = task.CreatedAt
	message.UpdatedAt = task.CreatedAt
	r.tasks[task.ID] = cloneTask(task)
	r.messages[task.ConversationID] = append(r.messages[task.ConversationID], cloneMessage(message))
	return nil
}

func (r *fakeChatWorkspaceRepo) GetImageTask(_ context.Context, userID, taskID int64) (*ChatImageTask, error) {
	item := r.tasks[taskID]
	if item == nil || item.UserID != userID {
		return nil, ErrChatWorkspaceTaskNotFound
	}
	return cloneTask(item), nil
}

func cloneConversation(in *ChatConversation) *ChatConversation {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}

func cloneMessage(in *ChatMessage) *ChatMessage {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}

func cloneAsset(in *ChatAsset) *ChatAsset {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}

func cloneTask(in *ChatImageTask) *ChatImageTask {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}

func TestChatWorkspaceServiceCreatesConversationAndMessage(t *testing.T) {
	repo := newFakeChatWorkspaceRepo()
	svc := NewChatWorkspaceService(repo)

	conversation, err := svc.CreateConversation(context.Background(), 42, "First prompt")
	require.NoError(t, err)
	require.Equal(t, int64(42), conversation.UserID)
	require.Equal(t, ChatConversationStatusActive, conversation.Status)

	message, err := svc.AppendMessage(context.Background(), CreateChatMessageInput{
		UserID:         42,
		ConversationID: conversation.ID,
		MessageType:    ChatMessageTypeText,
		Role:           "user",
		Content:        "hello",
	})
	require.NoError(t, err)
	require.Equal(t, "hello", message.Content)

	messages, err := svc.ListMessages(context.Background(), 42, conversation.ID)
	require.NoError(t, err)
	require.Len(t, messages, 1)
	require.Equal(t, "user", messages[0].Role)
}

func TestChatWorkspaceServiceRejectsCrossConversationAsset(t *testing.T) {
	repo := newFakeChatWorkspaceRepo()
	svc := NewChatWorkspaceService(repo)

	left, err := svc.CreateConversation(context.Background(), 42, "left")
	require.NoError(t, err)
	right, err := svc.CreateConversation(context.Background(), 42, "right")
	require.NoError(t, err)

	asset, err := svc.RegisterAsset(context.Background(), RegisterChatAssetInput{
		UserID:         42,
		ConversationID: &left.ID,
		AssetKind:      ChatAssetKindImage,
		URL:            "data:image/png;base64,abc",
		OriginalName:   "sample.png",
		ContentType:    "image/png",
		ByteSize:       10,
	})
	require.NoError(t, err)

	_, err = svc.AppendMessage(context.Background(), CreateChatMessageInput{
		UserID:         42,
		ConversationID: right.ID,
		MessageType:    ChatMessageTypeAttachment,
		Role:           "user",
		Content:        "wrong conversation",
		AssetID:        &asset.ID,
	})
	require.ErrorIs(t, err, ErrChatWorkspaceInvalidInput)
}

func TestChatWorkspaceServiceCreatesImageTaskAndMessage(t *testing.T) {
	repo := newFakeChatWorkspaceRepo()
	svc := NewChatWorkspaceService(repo)

	conversation, err := svc.CreateConversation(context.Background(), 42, "image")
	require.NoError(t, err)

	task, err := svc.CreateImageTask(context.Background(), CreateChatImageTaskInput{
		UserID:         42,
		ConversationID: conversation.ID,
		IdempotencyKey: "image-key-1",
		Prompt:         "make a clean product image",
		Model:          "gpt-image-1",
		Ratio:          "1:1",
		Purpose:        "product",
		Style:          "clean",
	})
	require.NoError(t, err)
	require.Equal(t, ChatImageTaskStatusPending, task.TaskStatus)
	require.Equal(t, ChatImageBillingStatusNotBilled, task.BillingStatus)
	require.NotNil(t, task.MessageID)

	messages, err := svc.ListMessages(context.Background(), 42, conversation.ID)
	require.NoError(t, err)
	require.Len(t, messages, 1)
	require.Equal(t, ChatMessageTypeImageTask, messages[0].MessageType)
}

func TestChatWorkspaceServiceRejectsInvalidInput(t *testing.T) {
	repo := newFakeChatWorkspaceRepo()
	svc := NewChatWorkspaceService(repo)

	_, err := svc.CreateConversation(context.Background(), 0, "bad")
	require.ErrorIs(t, err, ErrChatWorkspaceInvalidInput)

	_, err = svc.RegisterAsset(context.Background(), RegisterChatAssetInput{
		UserID:    42,
		AssetKind: "video",
	})
	require.ErrorIs(t, err, ErrChatWorkspaceInvalidAssetKind)

	errUnexpected := errors.New("unexpected")
	require.NotErrorIs(t, errUnexpected, ErrChatWorkspaceInvalidInput)
}
