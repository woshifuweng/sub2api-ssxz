package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

var (
	errWorkspaceNoUsableAPIKey = errors.New("workspace has no usable api key")
	errWorkspaceGatewayFailed  = errors.New("workspace gateway request failed")
)

type ChatWorkspaceHandler struct {
	service *service.ChatWorkspaceService
}

func NewChatWorkspaceHandler(workspaceService *service.ChatWorkspaceService) *ChatWorkspaceHandler {
	return &ChatWorkspaceHandler{service: workspaceService}
}

func ProvideChatWorkspaceService(
	repo service.ChatWorkspaceRepository,
	apiKeyService *service.APIKeyService,
	openAIGateway *OpenAIGatewayHandler,
) *service.ChatWorkspaceService {
	responder := &workspaceGatewayResponder{
		apiKeyService: apiKeyService,
		gateway:       openAIGateway,
	}
	return service.NewChatWorkspaceServiceWithResponder(repo, responder)
}

func (h *ChatWorkspaceHandler) ListConversations(c *gin.Context) {
	userID, ok := workspaceUserID(c)
	if !ok {
		return
	}
	items, err := h.service.ListConversations(c.Request.Context(), userID)
	if err != nil {
		workspaceError(c, err)
		return
	}
	response.Success(c, items)
}

func (h *ChatWorkspaceHandler) CreateConversation(c *gin.Context) {
	userID, ok := workspaceUserID(c)
	if !ok {
		return
	}
	var body struct {
		Title string `json:"title"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, "对话标题格式不正确")
		return
	}
	item, err := h.service.CreateConversation(c.Request.Context(), userID, service.WorkspaceCreateConversationInput{Title: body.Title})
	if err != nil {
		workspaceError(c, err)
		return
	}
	response.Created(c, item)
}

func (h *ChatWorkspaceHandler) GetConversation(c *gin.Context) {
	userID, conversationID, ok := workspaceConversationContext(c)
	if !ok {
		return
	}
	item, err := h.service.GetConversation(c.Request.Context(), userID, conversationID)
	if err != nil {
		workspaceError(c, err)
		return
	}
	response.Success(c, item)
}

func (h *ChatWorkspaceHandler) ListMessages(c *gin.Context) {
	userID, conversationID, ok := workspaceConversationContext(c)
	if !ok {
		return
	}
	items, err := h.service.ListMessages(c.Request.Context(), userID, conversationID)
	if err != nil {
		workspaceError(c, err)
		return
	}
	response.Success(c, items)
}

func (h *ChatWorkspaceHandler) AppendMessage(c *gin.Context) {
	userID, conversationID, ok := workspaceConversationContext(c)
	if !ok {
		return
	}
	var body struct {
		MessageType string         `json:"message_type"`
		Role        string         `json:"role"`
		Content     string         `json:"content"`
		Model       string         `json:"model"`
		Intent      string         `json:"intent"`
		Metadata    map[string]any `json:"metadata"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, "消息格式不正确")
		return
	}
	userMessage, assistantMessage, err := h.service.AppendMessageWithAssistantResponse(
		c.Request.Context(),
		userID,
		service.WorkspaceAppendMessageInput{
			ConversationID: conversationID,
			MessageType:    body.MessageType,
			Role:           body.Role,
			Content:        body.Content,
			Model:          body.Model,
			Intent:         body.Intent,
			Metadata:       body.Metadata,
		},
	)
	if err != nil {
		workspaceError(c, err)
		return
	}
	response.Success(c, gin.H{
		"user_message":      userMessage,
		"assistant_message": assistantMessage,
	})
}

func workspaceUserID(c *gin.Context) (int64, bool) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Error(c, http.StatusUnauthorized, "请先登录")
		return 0, false
	}
	return subject.UserID, true
}

func workspaceConversationContext(c *gin.Context) (int64, int64, bool) {
	userID, ok := workspaceUserID(c)
	if !ok {
		return 0, 0, false
	}
	conversationID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || conversationID <= 0 {
		response.Error(c, http.StatusBadRequest, "对话编号不正确")
		return 0, 0, false
	}
	return userID, conversationID, true
}

func workspaceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrWorkspaceConversationNotFound):
		response.Error(c, http.StatusNotFound, "对话不存在")
	case errors.Is(err, service.ErrWorkspaceInvalidModel):
		response.Error(c, http.StatusBadRequest, "所选模型不可用")
	case errors.Is(err, service.ErrWorkspaceInvalidIntent),
		errors.Is(err, service.ErrWorkspaceInvalidMessage):
		response.Error(c, http.StatusBadRequest, "消息内容不正确")
	case errors.Is(err, service.ErrWorkspaceAttachmentsDisabled),
		errors.Is(err, service.ErrWorkspaceCapabilityDisabled):
		response.Error(c, http.StatusBadRequest, "当前对话页仅支持文字消息")
	case errors.Is(err, errWorkspaceNoUsableAPIKey):
		response.Error(c, http.StatusForbidden, "没有找到可用于该模型的单分组 API Key，请先在 API 密钥页创建或启用")
	default:
		response.Error(c, http.StatusServiceUnavailable, "AI 回复暂时失败，请稍后重试；未完成的回复不会按成功请求扣费")
	}
}

type workspaceGatewayResponder struct {
	apiKeyService *service.APIKeyService
	gateway       *OpenAIGatewayHandler
}

func (r *workspaceGatewayResponder) GenerateAssistantResponse(ctx context.Context, input service.WorkspaceAssistantResponseInput) (service.WorkspaceAssistantResponse, error) {
	if r == nil || r.apiKeyService == nil || r.gateway == nil {
		return service.WorkspaceAssistantResponse{}, errWorkspaceGatewayFailed
	}
	apiKey, user, err := r.selectAPIKey(ctx, input.UserID, input.Model)
	if err != nil {
		return service.WorkspaceAssistantResponse{}, err
	}
	body, err := json.Marshal(gin.H{
		"model":  input.Model,
		"stream": false,
		"messages": []gin.H{{
			"role":    "user",
			"content": input.Content,
		}},
	})
	if err != nil {
		return service.WorkspaceAssistantResponse{}, errWorkspaceGatewayFailed
	}

	recorder := httptest.NewRecorder()
	requestContext := context.WithValue(ctx, ctxkey.UserID, user.ID)
	requestContext = context.WithValue(requestContext, ctxkey.Group, apiKey.Group)
	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body)).WithContext(requestContext)
	request.Header.Set("Content-Type", "application/json")
	request.RemoteAddr = "127.0.0.1:0"
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = request
	ginContext.Set(string(middleware2.ContextKeyAPIKey), apiKey)
	ginContext.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID, Concurrency: user.Concurrency})
	ginContext.Set(string(middleware2.ContextKeyUserRole), user.Role)
	r.gateway.ChatCompletions(ginContext)

	if recorder.Code < http.StatusOK || recorder.Code >= http.StatusMultipleChoices {
		return service.WorkspaceAssistantResponse{}, fmt.Errorf("%w: status %d", errWorkspaceGatewayFailed, recorder.Code)
	}
	var payload struct {
		ID      string `json:"id"`
		Choices []struct {
			FinishReason string `json:"finish_reason"`
			Message      struct {
				Content any `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil || len(payload.Choices) == 0 {
		return service.WorkspaceAssistantResponse{}, errWorkspaceGatewayFailed
	}
	content := workspaceResponseText(payload.Choices[0].Message.Content)
	if content == "" {
		return service.WorkspaceAssistantResponse{}, errWorkspaceGatewayFailed
	}
	return service.WorkspaceAssistantResponse{
		Content: content,
		Model:   input.Model,
		Status:  service.WorkspaceMessageStatusCompleted,
		Metadata: map[string]any{
			"provider":      "sub2api",
			"request_id":    payload.ID,
			"finish_reason": payload.Choices[0].FinishReason,
		},
	}, nil
}

func (r *workspaceGatewayResponder) selectAPIKey(ctx context.Context, userID int64, model string) (*service.APIKey, *service.User, error) {
	params := pagination.DefaultPagination()
	params.PageSize = 1000
	keys, _, err := r.apiKeyService.List(ctx, userID, params, service.APIKeyListFilters{})
	if err != nil {
		return nil, nil, errWorkspaceGatewayFailed
	}
	for i := range keys {
		candidate := &keys[i]
		if candidate.Key == "" || !candidate.IsActive() || candidate.IsExpired() || candidate.IsQuotaExhausted() || !candidate.AllowsModel(model) {
			continue
		}
		if len(candidate.IPWhitelist) > 0 || len(candidate.IPBlacklist) > 0 {
			continue
		}
		if len(service.NormalizeAPIKeyGroupIDs(candidate.GroupID, candidate.GroupIDs)) != 1 {
			continue
		}
		validated, user, validateErr := r.apiKeyService.ValidateKey(ctx, candidate.Key)
		if validateErr != nil || validated == nil || user == nil || user.ID != userID || user.Balance <= 0 {
			continue
		}
		if validated.Group == nil || !service.IsGroupContextValid(validated.Group) || !validated.Group.IsActive() || validated.Group.IsSubscriptionType() {
			continue
		}
		validated.User = user
		return validated, user, nil
	}
	return nil, nil, errWorkspaceNoUsableAPIKey
}

func workspaceResponseText(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case []any:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			if object, ok := item.(map[string]any); ok {
				if text, ok := object["text"].(string); ok && strings.TrimSpace(text) != "" {
					parts = append(parts, strings.TrimSpace(text))
				}
			}
		}
		return strings.Join(parts, "\n")
	default:
		return ""
	}
}

var _ service.WorkspaceAssistantResponder = (*workspaceGatewayResponder)(nil)
