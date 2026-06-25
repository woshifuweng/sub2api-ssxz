package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/internal/util/logredact"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const (
	workspaceReasonUnauthenticated       = "WORKSPACE_UNAUTHENTICATED"
	workspaceReasonInvalidRequest        = "WORKSPACE_INVALID_REQUEST"
	workspaceReasonInvalidConversationID = "WORKSPACE_INVALID_CONVERSATION_ID"
	workspaceReasonConversationNotFound  = "WORKSPACE_CONVERSATION_NOT_FOUND"
	workspaceReasonModelUnavailable      = "WORKSPACE_MODEL_UNAVAILABLE"
	workspaceReasonIntentUnavailable     = "WORKSPACE_INTENT_UNAVAILABLE"
	workspaceReasonCapabilityUnavailable = "WORKSPACE_CAPABILITY_UNAVAILABLE"
	workspaceReasonAttachmentsDisabled   = "WORKSPACE_ATTACHMENTS_DISABLED"
	workspaceReasonInvalidMessage        = "WORKSPACE_INVALID_MESSAGE"
	workspaceReasonServiceUnavailable    = "WORKSPACE_SERVICE_UNAVAILABLE"
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
		writeChatWorkspaceStatus(c, http.StatusUnauthorized, "User not authenticated", workspaceReasonUnauthenticated)
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
		writeChatWorkspaceStatus(c, http.StatusUnauthorized, "User not authenticated", workspaceReasonUnauthenticated)
		return
	}

	var req createWorkspaceConversationRequest
	if err := c.BindJSON(&req); err != nil {
		writeChatWorkspaceStatus(c, http.StatusBadRequest, "Invalid request", workspaceReasonInvalidRequest)
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
		writeChatWorkspaceStatus(c, http.StatusBadRequest, "Invalid request", workspaceReasonInvalidRequest)
		return
	}

	msg, _, err := h.workspaceService.AppendMessageWithAssistantResponse(c.Request().Context(), subject.UserID, service.WorkspaceAppendMessageInput{
		ConversationID:  conversationID,
		MessageType:     req.MessageType,
		Role:            req.Role,
		Content:         req.Content,
		Model:           req.Model,
		Intent:          req.Intent,
		Metadata:        req.Metadata,
		AllowedGroupIDs: subject.AllowedGroupIDs,
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
		writeChatWorkspaceStatus(c, http.StatusUnauthorized, "User not authenticated", workspaceReasonUnauthenticated)
		return middleware2.AuthSubject{}, 0, false
	}
	id, err := strconv.ParseInt(c.PathParam("id"), 10, 64)
	if err != nil || id <= 0 {
		writeChatWorkspaceStatus(c, http.StatusBadRequest, "Invalid conversation id", workspaceReasonInvalidConversationID)
		return middleware2.AuthSubject{}, 0, false
	}
	return subject, id, true
}

func writeChatWorkspaceError(c gatewayctx.GatewayContext, err error) {
	switch {
	case errors.Is(err, service.ErrWorkspaceConversationNotFound):
		writeChatWorkspaceStatusWithError(c, http.StatusNotFound, "Conversation not found", workspaceReasonConversationNotFound, err)
	case errors.Is(err, service.ErrWorkspaceInvalidModel):
		writeChatWorkspaceStatusWithError(c, http.StatusBadRequest, "Model is not available for workspace chat", workspaceReasonModelUnavailable, err)
	case errors.Is(err, service.ErrWorkspaceInvalidIntent):
		writeChatWorkspaceStatusWithError(c, http.StatusBadRequest, "Intent is not available for workspace chat", workspaceReasonIntentUnavailable, err)
	case errors.Is(err, service.ErrWorkspaceCapabilityDisabled):
		writeChatWorkspaceStatusWithError(c, http.StatusBadRequest, "Capability is not available in this workspace phase", workspaceReasonCapabilityUnavailable, err)
	case errors.Is(err, service.ErrWorkspaceAttachmentsDisabled):
		writeChatWorkspaceStatusWithError(c, http.StatusBadRequest, "Attachments are disabled in workspace text beta", workspaceReasonAttachmentsDisabled, err)
	case errors.Is(err, service.ErrWorkspaceInvalidMessage):
		writeChatWorkspaceStatusWithError(c, http.StatusBadRequest, "Message is invalid", workspaceReasonInvalidMessage, err)
	default:
		writeChatWorkspaceStatusWithError(c, http.StatusInternalServerError, "Workspace service unavailable", workspaceReasonServiceUnavailable, err)
	}
}

func writeChatWorkspaceStatus(c gatewayctx.GatewayContext, status int, message, reason string) {
	writeChatWorkspaceStatusWithError(c, status, message, reason, nil)
}

func writeChatWorkspaceStatusWithError(c gatewayctx.GatewayContext, status int, message, reason string, err error) {
	logChatWorkspaceError(c, status, reason, err)
	response.ErrorWithDetailsContext(gatewayJSONResponder{ctx: c}, status, message, reason, chatWorkspaceErrorMetadata(c))
}

func chatWorkspaceErrorMetadata(c gatewayctx.GatewayContext) map[string]string {
	if c == nil || c.Request() == nil {
		return nil
	}

	metadata := make(map[string]string, 2)
	if requestID, ok := c.Request().Context().Value(ctxkey.RequestID).(string); ok {
		if requestID = strings.TrimSpace(requestID); requestID != "" {
			metadata["request_id"] = requestID
		}
	}
	if clientRequestID, ok := c.Request().Context().Value(ctxkey.ClientRequestID).(string); ok {
		if clientRequestID = strings.TrimSpace(clientRequestID); clientRequestID != "" {
			metadata["client_request_id"] = clientRequestID
		}
	}
	if len(metadata) == 0 {
		return nil
	}
	return metadata
}

func logChatWorkspaceError(c gatewayctx.GatewayContext, status int, reason string, err error) {
	if c == nil || c.Request() == nil {
		return
	}

	fields := []zap.Field{
		zap.String("component", "chat_workspace"),
		zap.Int("status", status),
		zap.String("reason", strings.TrimSpace(reason)),
		zap.String("method", c.Method()),
		zap.String("path", c.Path()),
	}
	if subject, ok := middleware2.GetAuthSubjectFromGatewayContext(c); ok {
		fields = append(fields, zap.Int64("user_id", subject.UserID))
	}
	if conversationID, parseErr := strconv.ParseInt(c.PathParam("id"), 10, 64); parseErr == nil && conversationID > 0 {
		fields = append(fields, zap.Int64("conversation_id", conversationID))
	}
	if err != nil {
		fields = append(fields, zap.String("error", logredact.RedactText(err.Error(), "authorization", "api_key", "token", "secret", "cookie")))
	}

	entry := logger.FromContext(c.Request().Context())
	if status >= http.StatusInternalServerError {
		entry.Error("chat workspace request failed", fields...)
		return
	}
	entry.Warn("chat workspace request rejected", fields...)
}
