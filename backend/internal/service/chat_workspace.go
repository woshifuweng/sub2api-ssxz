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
	WorkspaceRoleSystem    = "system"

	WorkspaceIntentChat = "chat"

	WorkspaceMessageStatusCompleted = "completed"

	workspaceMaxTitleLength   = 255
	workspaceMaxContentLength = 12000
)

var (
	ErrWorkspaceConversationNotFound = errors.New("workspace conversation not found")
	ErrWorkspaceInvalidModel         = errors.New("workspace model is not available")
	ErrWorkspaceInvalidIntent        = errors.New("workspace intent is not available")
	ErrWorkspaceInvalidMessage       = errors.New("workspace message is invalid")
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
	ConversationID int64
	MessageType    string
	Role           string
	Content        string
	Model          string
	Intent         string
	Status         string
	Metadata       map[string]any
}

type ChatWorkspaceRepository interface {
	ListConversations(ctx context.Context, userID int64) ([]WorkspaceConversation, error)
	CreateConversation(ctx context.Context, userID int64, title string) (*WorkspaceConversation, error)
	GetConversation(ctx context.Context, userID, conversationID int64) (*WorkspaceConversation, error)
	ListMessages(ctx context.Context, userID, conversationID int64) ([]WorkspaceMessage, error)
	AppendMessage(ctx context.Context, userID int64, input WorkspaceAppendMessageInput, titleIfEmpty string) (*WorkspaceMessage, error)
}

type ChatWorkspaceService struct {
	repo ChatWorkspaceRepository
}

func NewChatWorkspaceService(repo ChatWorkspaceRepository) *ChatWorkspaceService {
	return &ChatWorkspaceService{repo: repo}
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
	if s == nil || s.repo == nil || userID <= 0 || conversationID <= 0 {
		return nil, ErrWorkspaceConversationNotFound
	}
	if _, err := s.repo.GetConversation(ctx, userID, conversationID); err != nil {
		return nil, err
	}
	return s.repo.ListMessages(ctx, userID, conversationID)
}

func (s *ChatWorkspaceService) AppendMessage(ctx context.Context, userID int64, input WorkspaceAppendMessageInput) (*WorkspaceMessage, error) {
	if s == nil || s.repo == nil || userID <= 0 || input.ConversationID <= 0 {
		return nil, ErrWorkspaceConversationNotFound
	}
	if _, err := s.repo.GetConversation(ctx, userID, input.ConversationID); err != nil {
		return nil, err
	}

	input.MessageType = normalizeWorkspaceMessageType(input.MessageType)
	input.Role = normalizeWorkspaceRole(input.Role)
	input.Content = strings.TrimSpace(input.Content)
	input.Model = strings.TrimSpace(input.Model)
	input.Intent = normalizeWorkspaceIntent(input.Intent)
	input.Status = WorkspaceMessageStatusCompleted

	if input.MessageType != WorkspaceMessageTypeText || input.Role != WorkspaceRoleUser || input.Content == "" {
		return nil, ErrWorkspaceInvalidMessage
	}
	if containsUnsafeInlinePayload(input.Content) || metadataContainsUnsafeInlinePayload(input.Metadata) {
		return nil, ErrWorkspaceInvalidMessage
	}
	if !isAllowedWorkspaceModel(input.Model) {
		return nil, ErrWorkspaceInvalidModel
	}
	if input.Intent != WorkspaceIntentChat {
		if isDisabledWorkspaceIntent(input.Intent) {
			return nil, ErrWorkspaceCapabilityDisabled
		}
		return nil, ErrWorkspaceInvalidIntent
	}
	if utf8.RuneCountInString(input.Content) > workspaceMaxContentLength {
		input.Content = string([]rune(input.Content)[:workspaceMaxContentLength])
	}

	return s.repo.AppendMessage(ctx, userID, input, deriveWorkspaceTitle(input.Content))
}

func sanitizeWorkspaceTitle(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	runes := []rune(value)
	if len(runes) > workspaceMaxTitleLength {
		runes = runes[:workspaceMaxTitleLength]
	}
	return string(runes)
}

func deriveWorkspaceTitle(content string) string {
	content = sanitizeWorkspaceTitle(content)
	if content == "" {
		return ""
	}
	runes := []rune(content)
	if len(runes) > 40 {
		return string(runes[:40])
	}
	return content
}

func normalizeWorkspaceMessageType(value string) string {
	if strings.TrimSpace(value) == "" {
		return WorkspaceMessageTypeText
	}
	return strings.TrimSpace(value)
}

func normalizeWorkspaceRole(value string) string {
	if strings.TrimSpace(value) == "" {
		return WorkspaceRoleUser
	}
	return strings.TrimSpace(value)
}

func normalizeWorkspaceIntent(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return WorkspaceIntentChat
	}
	return value
}

func isAllowedWorkspaceModel(model string) bool {
	switch strings.TrimSpace(model) {
	case "gpt-5.5", "gpt-5.4", "gpt-5.2", "gpt-5.4-mini":
		return true
	default:
		return false
	}
}

func isDisabledWorkspaceIntent(intent string) bool {
	switch strings.TrimSpace(intent) {
	case "image_generation", "image", "image_edit", "vision", "file_analysis", "web", "memory", "toolbox", "tools", "document_analysis":
		return true
	default:
		return false
	}
}

func containsUnsafeInlinePayload(value string) bool {
	lower := strings.ToLower(value)
	return strings.Contains(lower, "data:image/") ||
		strings.Contains(lower, "data:application/") ||
		strings.Contains(lower, ";base64,")
}

func metadataContainsUnsafeInlinePayload(metadata map[string]any) bool {
	for _, value := range metadata {
		if anyContainsUnsafeInlinePayload(value) {
			return true
		}
	}
	return false
}

func anyContainsUnsafeInlinePayload(value any) bool {
	switch v := value.(type) {
	case string:
		return containsUnsafeInlinePayload(v)
	case []any:
		for _, item := range v {
			if anyContainsUnsafeInlinePayload(item) {
				return true
			}
		}
	case map[string]any:
		return metadataContainsUnsafeInlinePayload(v)
	case map[string]string:
		for _, item := range v {
			if containsUnsafeInlinePayload(item) {
				return true
			}
		}
	}
	return false
}
