package service

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	WorkspaceMessageTypeText = "text"

	WorkspaceRoleUser      = "user"
	WorkspaceRoleAssistant = "assistant"

	WorkspaceIntentChat = "chat"

	WorkspaceMessageStatusCompleted = "completed"
	WorkspaceMessageStatusFailed    = "failed"

	WorkspaceAssistantUnavailableContent = "当前模型暂不可用，请切换其他模型，或联系管理员检查 API Key、分组和上游账号配置。本次未完成回复，不会按成功回复扣费。"

	workspaceMaxTitleLength   = 255
	workspaceMaxContentLength = 12000
)

var (
	ErrWorkspaceConversationNotFound = errors.New("workspace conversation not found")
	ErrWorkspaceInvalidModel         = errors.New("workspace model is not available")
	ErrWorkspaceInvalidIntent        = errors.New("workspace intent is not available")
	ErrWorkspaceInvalidMessage       = errors.New("workspace message is invalid")
	ErrWorkspaceAttachmentsDisabled  = errors.New("workspace attachments are disabled")
	ErrWorkspaceCapabilityDisabled   = errors.New("workspace capability is disabled")
)

type WorkspaceConversation struct {
	ID            int64      `json:"id"`
	UserID        int64      `json:"-"`
	Title         string     `json:"title"`
	Status        string     `json:"status"`
	LastMessageAt *time.Time `json:"last_message_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type WorkspaceMessage struct {
	ID             int64          `json:"id"`
	ConversationID int64          `json:"conversation_id"`
	UserID         int64          `json:"-"`
	MessageType    string         `json:"message_type"`
	Role           string         `json:"role"`
	Content        string         `json:"content"`
	Model          string         `json:"model,omitempty"`
	Intent         string         `json:"intent"`
	Status         string         `json:"status"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

type WorkspaceCreateConversationInput struct {
	Title string
}

type WorkspaceAppendMessageInput struct {
	ConversationID  int64
	MessageType     string
	Role            string
	Content         string
	Model           string
	Intent          string
	Status          string
	Metadata        map[string]any
	AllowedGroupIDs []int64
}

type WorkspaceAssistantResponseInput struct {
	UserID          int64
	AllowedGroupIDs []int64
	ConversationID  int64
	UserMessage     WorkspaceMessage
	Content         string
	Model           string
	Intent          string
	Metadata        map[string]any
}

type WorkspaceAssistantResponse struct {
	Content  string
	Model    string
	Status   string
	Metadata map[string]any
}

type WorkspaceAssistantResponder interface {
	GenerateAssistantResponse(context.Context, WorkspaceAssistantResponseInput) (WorkspaceAssistantResponse, error)
}

type ChatWorkspaceRepository interface {
	ListConversations(context.Context, int64) ([]WorkspaceConversation, error)
	CreateConversation(context.Context, int64, string) (*WorkspaceConversation, error)
	GetConversation(context.Context, int64, int64) (*WorkspaceConversation, error)
	ListMessages(context.Context, int64, int64) ([]WorkspaceMessage, error)
	AppendMessage(context.Context, int64, WorkspaceAppendMessageInput, string) (*WorkspaceMessage, error)
}

type ChatWorkspaceService struct {
	repo      ChatWorkspaceRepository
	responder WorkspaceAssistantResponder
}

func NewChatWorkspaceService(repo ChatWorkspaceRepository) *ChatWorkspaceService {
	return NewChatWorkspaceServiceWithResponder(repo, nil)
}

func NewChatWorkspaceServiceWithResponder(repo ChatWorkspaceRepository, responder WorkspaceAssistantResponder) *ChatWorkspaceService {
	return &ChatWorkspaceService{repo: repo, responder: responder}
}

func (s *ChatWorkspaceService) ListConversations(ctx context.Context, userID int64) ([]WorkspaceConversation, error) {
	if s == nil || s.repo == nil || userID <= 0 {
		return nil, ErrWorkspaceConversationNotFound
	}
	return s.repo.ListConversations(ctx, userID)
}

func (s *ChatWorkspaceService) CreateConversation(ctx context.Context, userID int64, input WorkspaceCreateConversationInput) (*WorkspaceConversation, error) {
	if s == nil || s.repo == nil || userID <= 0 {
		return nil, ErrWorkspaceConversationNotFound
	}
	return s.repo.CreateConversation(ctx, userID, sanitizeWorkspaceTitle(input.Title))
}

func (s *ChatWorkspaceService) GetConversation(ctx context.Context, userID, conversationID int64) (*WorkspaceConversation, error) {
	if s == nil || s.repo == nil || userID <= 0 || conversationID <= 0 {
		return nil, ErrWorkspaceConversationNotFound
	}
	return s.repo.GetConversation(ctx, userID, conversationID)
}

func (s *ChatWorkspaceService) ListMessages(ctx context.Context, userID, conversationID int64) ([]WorkspaceMessage, error) {
	if _, err := s.GetConversation(ctx, userID, conversationID); err != nil {
		return nil, err
	}
	return s.repo.ListMessages(ctx, userID, conversationID)
}

func (s *ChatWorkspaceService) AppendMessageWithAssistantResponse(ctx context.Context, userID int64, input WorkspaceAppendMessageInput) (*WorkspaceMessage, *WorkspaceMessage, error) {
	userMessage, err := s.appendUserMessage(ctx, userID, input)
	if err != nil {
		return nil, nil, err
	}
	if s.responder == nil {
		return userMessage, nil, errors.New("workspace assistant responder is unavailable")
	}
	result, err := s.responder.GenerateAssistantResponse(ctx, WorkspaceAssistantResponseInput{
		UserID:          userID,
		AllowedGroupIDs: append([]int64(nil), input.AllowedGroupIDs...),
		ConversationID:  input.ConversationID,
		UserMessage:     *userMessage,
		Content:         userMessage.Content,
		Model:           userMessage.Model,
		Intent:          userMessage.Intent,
		Metadata:        userMessage.Metadata,
	})
	if err != nil {
		return userMessage, nil, err
	}
	content := strings.TrimSpace(result.Content)
	if content == "" || containsUnsafeInlinePayload(content) {
		return userMessage, nil, ErrWorkspaceInvalidMessage
	}
	status := strings.TrimSpace(result.Status)
	if status == "" {
		status = WorkspaceMessageStatusCompleted
	}
	if status != WorkspaceMessageStatusCompleted && status != WorkspaceMessageStatusFailed {
		return userMessage, nil, ErrWorkspaceInvalidMessage
	}
	model := strings.TrimSpace(result.Model)
	if model == "" {
		model = userMessage.Model
	}
	assistant, err := s.repo.AppendMessage(ctx, userID, WorkspaceAppendMessageInput{
		ConversationID: input.ConversationID,
		MessageType:    WorkspaceMessageTypeText,
		Role:           WorkspaceRoleAssistant,
		Content:        truncateWorkspaceContent(content),
		Model:          model,
		Intent:         WorkspaceIntentChat,
		Status:         status,
		Metadata:       sanitizeWorkspaceMetadata(result.Metadata),
	}, "")
	if err != nil {
		return userMessage, nil, err
	}
	return userMessage, assistant, nil
}

func (s *ChatWorkspaceService) appendUserMessage(ctx context.Context, userID int64, input WorkspaceAppendMessageInput) (*WorkspaceMessage, error) {
	if _, err := s.GetConversation(ctx, userID, input.ConversationID); err != nil {
		return nil, err
	}
	input.MessageType = defaultWorkspaceValue(input.MessageType, WorkspaceMessageTypeText)
	input.Role = defaultWorkspaceValue(input.Role, WorkspaceRoleUser)
	input.Intent = defaultWorkspaceValue(input.Intent, WorkspaceIntentChat)
	input.Content = strings.TrimSpace(input.Content)
	input.Model = strings.TrimSpace(input.Model)
	if input.MessageType != WorkspaceMessageTypeText || input.Role != WorkspaceRoleUser || input.Content == "" {
		return nil, ErrWorkspaceInvalidMessage
	}
	if input.Intent != WorkspaceIntentChat {
		return nil, ErrWorkspaceCapabilityDisabled
	}
	if !isAllowedWorkspaceModel(input.Model) {
		return nil, ErrWorkspaceInvalidModel
	}
	if containsUnsafeInlinePayload(input.Content) || metadataContainsUnsafeInlinePayload(input.Metadata) {
		return nil, ErrWorkspaceInvalidMessage
	}
	if metadataContainsAttachment(input.Metadata) {
		return nil, ErrWorkspaceAttachmentsDisabled
	}
	input.Content = truncateWorkspaceContent(input.Content)
	input.Status = WorkspaceMessageStatusCompleted
	input.Metadata = sanitizeWorkspaceMetadata(input.Metadata)
	return s.repo.AppendMessage(ctx, userID, input, deriveWorkspaceTitle(input.Content))
}

func sanitizeWorkspaceTitle(value string) string {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) > workspaceMaxTitleLength {
		runes = runes[:workspaceMaxTitleLength]
	}
	return string(runes)
}

func deriveWorkspaceTitle(value string) string {
	runes := []rune(sanitizeWorkspaceTitle(value))
	if len(runes) > 40 {
		runes = runes[:40]
	}
	return string(runes)
}

func truncateWorkspaceContent(value string) string {
	if utf8.RuneCountInString(value) <= workspaceMaxContentLength {
		return value
	}
	return string([]rune(value)[:workspaceMaxContentLength])
}

func defaultWorkspaceValue(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func isAllowedWorkspaceModel(model string) bool {
	if model == "" || utf8.RuneCountInString(model) > 128 {
		return false
	}
	for _, r := range model {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			continue
		}
		switch r {
		case '.', '-', '_', '/', ':':
			continue
		default:
			return false
		}
	}
	return true
}

func containsUnsafeInlinePayload(value string) bool {
	lower := strings.ToLower(value)
	return strings.Contains(lower, ";base64,") || strings.Contains(lower, "data:image/") || strings.Contains(lower, "data:application/")
}

func metadataContainsUnsafeInlinePayload(metadata map[string]any) bool {
	for _, value := range metadata {
		switch typed := value.(type) {
		case string:
			if containsUnsafeInlinePayload(typed) {
				return true
			}
		case map[string]any:
			if metadataContainsUnsafeInlinePayload(typed) {
				return true
			}
		case []any:
			for _, item := range typed {
				if nested, ok := item.(map[string]any); ok && metadataContainsUnsafeInlinePayload(nested) {
					return true
				}
			}
		}
	}
	return false
}

func metadataContainsAttachment(metadata map[string]any) bool {
	for key, value := range metadata {
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "asset", "assets", "asset_id", "asset_ids", "attachment", "attachments", "file", "files", "image", "images":
			return true
		}
		if nested, ok := value.(map[string]any); ok && metadataContainsAttachment(nested) {
			return true
		}
	}
	return false
}

func sanitizeWorkspaceMetadata(metadata map[string]any) map[string]any {
	if len(metadata) == 0 {
		return nil
	}
	out := make(map[string]any)
	for _, key := range []string{"provider", "request_id", "finish_reason"} {
		if value, ok := metadata[key]; ok {
			out[key] = value
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
