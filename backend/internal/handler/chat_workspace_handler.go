package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type ChatWorkspaceHandler struct {
	workspaceService *service.ChatWorkspaceService
}

type createWorkspaceConversationRequest struct {
	Title string `json:"title"`
}

type appendWorkspaceMessageRequest struct {
	MessageType string         `json:"message_type"`
	Role        string         `json:"role"`
	Content     string         `json:"content"`
	Model       string         `json:"model"`
	Intent      string         `json:"intent"`
	Metadata    map[string]any `json:"metadata"`
}

func NewChatWorkspaceHandler(workspaceService *service.ChatWorkspaceService) *ChatWorkspaceHandler {
	return &ChatWorkspaceHandler{workspaceService: workspaceService}
}

func (h *ChatWorkspaceHandler) ListConversations(c *gin.Context) {
	h.ListConversationsGateway(gatewayctx.FromGin(c))
}

func (h *ChatWorkspaceHandler) ListConversationsGateway(c gatewayctx.GatewayContext) {
	subject, ok := middleware2.GetAuthSubjectFromGatewayContext(c)
	if !ok {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusUnauthorized, "User not authenticated")
		return
	}

	conversations, err := h.workspaceService.ListConversations(c.Request().Context(), subject.UserID)
	if err != nil {
		writeChatWorkspaceError(c, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, conversations)
}

func (h *ChatWorkspaceHandler) CreateConversation(c *gin.Context) {
	h.CreateConversationGateway(gatewayctx.FromGin(c))
}

func (h *ChatWorkspaceHandler) CreateConversationGateway(c gatewayctx.GatewayContext) {
	subject, ok := middleware2.GetAuthSubjectFromGatewayContext(c)
	if !ok {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var req createWorkspaceConversationRequest
	if err := c.BindJSON(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request")
		return
	}

	conversation, err := h.workspaceService.CreateConversation(c.Request().Context(), subject.UserID, service.WorkspaceCreateConversationInput{
		Title: req.Title,
	})
	if err != nil {
		writeChatWorkspaceError(c, err)
		return
	}
	response.CreatedContext(gatewayJSONResponder{ctx: c}, conversation)
}

func (h *ChatWorkspaceHandler) GetConversation(c *gin.Context) {
	h.GetConversationGateway(gatewayctx.FromGin(c))
}

func (h *ChatWorkspaceHandler) GetConversationGateway(c gatewayctx.GatewayContext) {
	subject, conversationID, ok := h.authenticatedConversationID(c)
	if !ok {
		return
	}
	conversation, err := h.workspaceService.GetConversation(c.Request().Context(), subject.UserID, conversationID)
	if err != nil {
		writeChatWorkspaceError(c, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, conversation)
}

func (h *ChatWorkspaceHandler) ListMessages(c *gin.Context) {
	h.ListMessagesGateway(gatewayctx.FromGin(c))
}

func (h *ChatWorkspaceHandler) ListMessagesGateway(c gatewayctx.GatewayContext) {
	subject, conversationID, ok := h.authenticatedConversationID(c)
	if !ok {
		return
	}
	messages, err := h.workspaceService.ListMessages(c.Request().Context(), subject.UserID, conversationID)
	if err != nil {
		writeChatWorkspaceError(c, err)
		return
	}
	response.SuccessContext(gatewayJSONResponder{ctx: c}, messages)
}

func (h *ChatWorkspaceHandler) AppendMessage(c *gin.Context) {
	h.AppendMessageGateway(gatewayctx.FromGin(c))
}

func (h *ChatWorkspaceHandler) AppendMessageGateway(c gatewayctx.GatewayContext) {
	subject, conversationID, ok := h.authenticatedConversationID(c)
	if !ok {
		return
	}

	var req appendWorkspaceMessageRequest
	if err := c.BindJSON(&req); err != nil {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid request")
		return
	}

	msg, err := h.workspaceService.AppendMessage(c.Request().Context(), subject.UserID, service.WorkspaceAppendMessageInput{
		ConversationID: conversationID,
		MessageType:    req.MessageType,
		Role:           req.Role,
		Content:        req.Content,
		Model:          req.Model,
		Intent:         req.Intent,
		Metadata:       req.Metadata,
	})
	if err != nil {
		writeChatWorkspaceError(c, err)
		return
	}
	response.CreatedContext(gatewayJSONResponder{ctx: c}, msg)
}

func (h *ChatWorkspaceHandler) authenticatedConversationID(c gatewayctx.GatewayContext) (middleware2.AuthSubject, int64, bool) {
	subject, ok := middleware2.GetAuthSubjectFromGatewayContext(c)
	if !ok {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusUnauthorized, "User not authenticated")
		return middleware2.AuthSubject{}, 0, false
	}
	id, err := strconv.ParseInt(c.PathParam("id"), 10, 64)
	if err != nil || id <= 0 {
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Invalid conversation id")
		return middleware2.AuthSubject{}, 0, false
	}
	return subject, id, true
}

func writeChatWorkspaceError(c gatewayctx.GatewayContext, err error) {
	switch {
	case errors.Is(err, service.ErrWorkspaceConversationNotFound):
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusNotFound, "Conversation not found")
	case errors.Is(err, service.ErrWorkspaceInvalidModel):
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Model is not available for workspace chat")
	case errors.Is(err, service.ErrWorkspaceInvalidIntent):
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Intent is not available for workspace chat")
	case errors.Is(err, service.ErrWorkspaceCapabilityDisabled):
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Capability is not available in this workspace phase")
	case errors.Is(err, service.ErrWorkspaceInvalidMessage):
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusBadRequest, "Message is invalid")
	default:
		response.ErrorContext(gatewayJSONResponder{ctx: c}, http.StatusInternalServerError, "Workspace service unavailable")
	}
}
