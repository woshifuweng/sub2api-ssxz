package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newChatWorkspaceHandlerTestRouter(userID int64) (*gin.Engine, *service.ChatWorkspaceService) {
	gin.SetMode(gin.TestMode)
	repo := newHandlerMemoryChatWorkspaceRepo()
	svc := service.NewChatWorkspaceService(repo)
	h := NewChatWorkspaceHandler(svc)

	router := gin.New()
	router.Use(middleware.RequestLogger(), middleware.ClientRequestID())
	if userID > 0 {
		router.Use(func(c *gin.Context) {
			c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: userID})
			c.Next()
		})
	}
	group := router.Group("/api/v1/chat-workspace")
	group.GET("/conversations", h.ListConversations)
	group.POST("/conversations", h.CreateConversation)
	group.GET("/conversations/:id", h.GetConversation)
	group.GET("/conversations/:id/messages", h.ListMessages)
	group.POST("/conversations/:id/messages", h.AppendMessage)
	return router, svc
}

func TestChatWorkspaceHandlerRejectsUnauthenticatedAccess(t *testing.T) {
	router, _ := newChatWorkspaceHandlerTestRouter(0)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/chat-workspace/conversations", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.NotEmpty(t, rec.Header().Get("X-Request-ID"))
	var envelope chatWorkspaceErrorEnvelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	require.Equal(t, workspaceReasonUnauthenticated, envelope.Reason)
	require.NotEmpty(t, envelope.Metadata["request_id"])
	require.NotEmpty(t, envelope.Metadata["client_request_id"])
	require.NotContains(t, rec.Body.String(), "stack")
	require.NotContains(t, rec.Body.String(), "Authorization")
	require.NotContains(t, rec.Body.String(), "api_key")
	require.NotContains(t, rec.Body.String(), "token")
	require.NotContains(t, rec.Body.String(), "secret")
}

func TestChatWorkspaceHandlerCreatesConversationAndRestoresMessages(t *testing.T) {
	router, _ := newChatWorkspaceHandlerTestRouter(42)

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/chat-workspace/conversations", bytes.NewBufferString(`{"title":""}`))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)
	require.Equal(t, http.StatusCreated, createRec.Code)

	var createEnvelope struct {
		Code int `json:"code"`
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(createRec.Body.Bytes(), &createEnvelope))
	require.Equal(t, 0, createEnvelope.Code)
	require.Positive(t, createEnvelope.Data.ID)

	appendBody := `{"message_type":"text","role":"user","content":"hello","model":"gpt-5.5","intent":"chat"}`
	appendReq := httptest.NewRequest(http.MethodPost, "/api/v1/chat-workspace/conversations/1/messages", bytes.NewBufferString(appendBody))
	appendReq.Header.Set("Content-Type", "application/json")
	appendRec := httptest.NewRecorder()
	router.ServeHTTP(appendRec, appendReq)
	require.Equal(t, http.StatusCreated, appendRec.Code)
	require.Contains(t, appendRec.Body.String(), `"role":"user"`)
	require.NotContains(t, appendRec.Body.String(), "AI response provider is not connected yet")

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/chat-workspace/conversations/1/messages", nil)
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)
	require.Equal(t, http.StatusOK, listRec.Code)
	require.Contains(t, listRec.Body.String(), "hello")
	require.Contains(t, listRec.Body.String(), service.WorkspaceAssistantUnavailableContent)
	require.NotContains(t, listRec.Body.String(), "AI response provider is not connected yet")
	require.Contains(t, listRec.Body.String(), `"role":"assistant"`)
	require.Contains(t, listRec.Body.String(), `"provider_called":false`)
	require.Contains(t, listRec.Body.String(), `"billing_touched":false`)
}

func TestChatWorkspaceHandlerRejectsDisabledCapabilityWithoutLeakingInternals(t *testing.T) {
	router, _ := newChatWorkspaceHandlerTestRouter(42)

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/chat-workspace/conversations", bytes.NewBufferString(`{}`))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)
	require.Equal(t, http.StatusCreated, createRec.Code)

	appendBody := `{"message_type":"text","role":"user","content":"draw","model":"gpt-5.5","intent":"image_generation"}`
	appendReq := httptest.NewRequest(http.MethodPost, "/api/v1/chat-workspace/conversations/1/messages", bytes.NewBufferString(appendBody))
	appendReq.Header.Set("Content-Type", "application/json")
	appendRec := httptest.NewRecorder()
	router.ServeHTTP(appendRec, appendReq)

	require.Equal(t, http.StatusBadRequest, appendRec.Code)
	var errorEnvelope chatWorkspaceErrorEnvelope
	require.NoError(t, json.Unmarshal(appendRec.Body.Bytes(), &errorEnvelope))
	require.Equal(t, workspaceReasonCapabilityUnavailable, errorEnvelope.Reason)
	require.NotEmpty(t, errorEnvelope.Metadata["request_id"])
	require.NotEmpty(t, errorEnvelope.Metadata["client_request_id"])
	require.Contains(t, appendRec.Body.String(), "Capability is not available")
	require.NotContains(t, appendRec.Body.String(), "SQL")
	require.NotContains(t, appendRec.Body.String(), "provider")
	require.NotContains(t, appendRec.Body.String(), "Authorization")
	require.NotContains(t, appendRec.Body.String(), "api_key")
	require.NotContains(t, appendRec.Body.String(), "token")
	require.NotContains(t, appendRec.Body.String(), "secret")

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/chat-workspace/conversations/1/messages", nil)
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)
	require.Equal(t, http.StatusOK, listRec.Code)
	require.NotContains(t, listRec.Body.String(), `"role":"assistant"`)
}

type chatWorkspaceErrorEnvelope struct {
	Code     int               `json:"code"`
	Message  string            `json:"message"`
	Reason   string            `json:"reason"`
	Metadata map[string]string `json:"metadata"`
}

type handlerMemoryChatWorkspaceRepo struct {
	nextConversationID int64
	nextMessageID      int64
	conversations      map[int64]*service.WorkspaceConversation
	messages           map[int64][]service.WorkspaceMessage
}

func newHandlerMemoryChatWorkspaceRepo() *handlerMemoryChatWorkspaceRepo {
	return &handlerMemoryChatWorkspaceRepo{
		nextConversationID: 1,
		nextMessageID:      1,
		conversations:      make(map[int64]*service.WorkspaceConversation),
		messages:           make(map[int64][]service.WorkspaceMessage),
	}
}

func (r *handlerMemoryChatWorkspaceRepo) ListConversations(_ context.Context, userID int64) ([]service.WorkspaceConversation, error) {
	out := make([]service.WorkspaceConversation, 0)
	for _, conversation := range r.conversations {
		if conversation.UserID == userID {
			out = append(out, *conversation)
		}
	}
	return out, nil
}

func (r *handlerMemoryChatWorkspaceRepo) CreateConversation(_ context.Context, userID int64, title string) (*service.WorkspaceConversation, error) {
	now := time.Now().UTC()
	conversation := &service.WorkspaceConversation{
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

func (r *handlerMemoryChatWorkspaceRepo) GetConversation(_ context.Context, userID, conversationID int64) (*service.WorkspaceConversation, error) {
	conversation := r.conversations[conversationID]
	if conversation == nil || conversation.UserID != userID {
		return nil, service.ErrWorkspaceConversationNotFound
	}
	cp := *conversation
	return &cp, nil
}

func (r *handlerMemoryChatWorkspaceRepo) ListMessages(_ context.Context, userID, conversationID int64) ([]service.WorkspaceMessage, error) {
	if _, err := r.GetConversation(context.Background(), userID, conversationID); err != nil {
		return nil, err
	}
	items := r.messages[conversationID]
	out := make([]service.WorkspaceMessage, len(items))
	copy(out, items)
	return out, nil
}

func (r *handlerMemoryChatWorkspaceRepo) AppendMessage(_ context.Context, userID int64, input service.WorkspaceAppendMessageInput, titleIfEmpty string) (*service.WorkspaceMessage, error) {
	conversation, err := r.GetConversation(context.Background(), userID, input.ConversationID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	msg := service.WorkspaceMessage{
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

var _ service.ChatWorkspaceRepository = (*handlerMemoryChatWorkspaceRepo)(nil)
