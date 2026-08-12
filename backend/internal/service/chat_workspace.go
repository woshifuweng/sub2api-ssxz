package service

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	WorkspaceMessageTypeText  = "text"
	WorkspaceMessageTypeImage = "image"

	WorkspaceRoleUser      = "user"
	WorkspaceRoleAssistant = "assistant"
	WorkspaceRoleSystem    = "system"

	WorkspaceIntentChat            = "chat"
	WorkspaceIntentImageGeneration = "image_generation"

	WorkspaceMessageStatusPending   = "pending"
	WorkspaceMessageStatusCompleted = "completed"
	WorkspaceMessageStatusFailed    = "failed"

	WorkspaceAssistantUnavailableContent = WorkspaceSub2APITextBridgeTemporarilyUnavailableContent

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

type WorkspaceAppendAssistantMessageInput struct {
	ConversationID int64
	MessageType    string
	Content        string
	Model          string
	Intent         string
	Status         string
	Metadata       map[string]any
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
	Content     string
	MessageType string
	Model       string
	Intent      string
	Status      string
	Metadata    map[string]any
}

type WorkspaceAssistantResponder interface {
	GenerateAssistantResponse(ctx context.Context, input WorkspaceAssistantResponseInput) (WorkspaceAssistantResponse, error)
}

type ChatWorkspaceRepository interface {
	ListConversations(ctx context.Context, userID int64) ([]WorkspaceConversation, error)
	CreateConversation(ctx context.Context, userID int64, title string) (*WorkspaceConversation, error)
	GetConversation(ctx context.Context, userID, conversationID int64) (*WorkspaceConversation, error)
	ListMessages(ctx context.Context, userID, conversationID int64) ([]WorkspaceMessage, error)
	AppendMessage(ctx context.Context, userID int64, input WorkspaceAppendMessageInput, titleIfEmpty string) (*WorkspaceMessage, error)
}

type ChatWorkspaceService struct {
	repo                         ChatWorkspaceRepository
	responder                    WorkspaceAssistantResponder
	selectedModelCatalogResolver WorkspaceSelectedModelCatalogResolver
}

func NewChatWorkspaceService(repo ChatWorkspaceRepository) *ChatWorkspaceService {
	return NewChatWorkspaceServiceWithResponder(repo, nil)
}

func NewChatWorkspaceServiceWithResponder(repo ChatWorkspaceRepository, responder WorkspaceAssistantResponder) *ChatWorkspaceService {
	return NewChatWorkspaceServiceWithResponderAndModelCatalogResolver(repo, responder, nil)
}

func NewChatWorkspaceServiceWithResponderAndModelCatalogResolver(repo ChatWorkspaceRepository, responder WorkspaceAssistantResponder, resolver WorkspaceSelectedModelCatalogResolver) *ChatWorkspaceService {
	if responder == nil {
		responder = WorkspaceUnavailableAssistantResponder{}
	}
	return &ChatWorkspaceService{repo: repo, responder: responder, selectedModelCatalogResolver: resolver}
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
	if metadataContainsUserAssetPayload(input.Metadata) {
		return nil, ErrWorkspaceAttachmentsDisabled
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
	capabilityPlan := NewWorkspaceCapabilityPlanner().Plan(WorkspaceCapabilityPlannerInput{
		Text:          input.Content,
		SelectedModel: input.Model,
	})
	modelCapabilityMetadata := ResolveWorkspaceModelCapabilities(input.Model, WorkspaceModelCapabilityHints{})
	modelCatalogResolution := s.resolveSelectedModelFromCatalog(ctx, userID, input, capabilityPlan.PlannedCapability)
	if modelCatalogResolution.Model != "" {
		if workspaceSelectedModelResolutionBlocksMessage(modelCatalogResolution, capabilityPlan.PlannedCapability) {
			return nil, ErrWorkspaceInvalidModel
		}
		modelCapabilityMetadata = modelCatalogResolution.ModelCapabilityMetadata()
	}
	capabilityPlan = ApplyWorkspaceModelCapabilityMatch(capabilityPlan, modelCapabilityMetadata)
	imageExperiencePlan := BuildWorkspaceImageExperiencePlan(WorkspaceImageExperienceEnhancerInput{
		Text:                    input.Content,
		PlannedCapability:       capabilityPlan.PlannedCapability,
		SelectedModel:           input.Model,
		ModelCapabilityMetadata: modelCapabilityMetadata,
	})
	input.Metadata = mergeWorkspaceCapabilityPlanMetadata(input.Metadata, capabilityPlan)
	input.Metadata = mergeWorkspaceModelCapabilityMetadata(input.Metadata, modelCapabilityMetadata, capabilityPlan)
	if modelCatalogResolution.Model != "" {
		input.Metadata = mergeWorkspaceSelectedModelCatalogMetadata(input.Metadata, modelCatalogResolution)
	}
	input.Metadata = mergeWorkspaceImageExperiencePlanMetadata(input.Metadata, imageExperiencePlan)

	return s.repo.AppendMessage(ctx, userID, input, deriveWorkspaceTitle(input.Content))
}

func (s *ChatWorkspaceService) resolveSelectedModelFromCatalog(ctx context.Context, userID int64, input WorkspaceAppendMessageInput, planned WorkspacePlannedCapability) WorkspaceSelectedModelChannelCatalogResolution {
	if s == nil || s.selectedModelCatalogResolver == nil {
		return WorkspaceSelectedModelChannelCatalogResolution{}
	}
	resolution, err := s.selectedModelCatalogResolver.ResolveSelectedModel(ctx, WorkspaceSelectedModelChannelCatalogResolverInput{
		UserID:            userID,
		AllowedGroupIDs:   cloneWorkspaceInt64Slice(input.AllowedGroupIDs),
		SelectedModel:     input.Model,
		PlannedCapability: planned,
	})
	if err != nil {
		return WorkspaceSelectedModelChannelCatalogResolution{
			Model:              strings.TrimSpace(input.Model),
			ModelCatalogSource: WorkspaceModelCatalogSourceUnknown,
			BlockReason:        WorkspaceSelectedModelBlockReasonChannelCatalogMissing,
		}
	}
	return resolution
}

func (s *ChatWorkspaceService) AppendMessageWithAssistantResponse(ctx context.Context, userID int64, input WorkspaceAppendMessageInput) (*WorkspaceMessage, *WorkspaceMessage, error) {
	userMessage, err := s.AppendMessage(ctx, userID, input)
	if err != nil {
		return nil, nil, err
	}

	responder := s.responder
	if responder == nil {
		responder = WorkspaceUnavailableAssistantResponder{}
	}
	assistantResponse, err := responder.GenerateAssistantResponse(ctx, WorkspaceAssistantResponseInput{
		UserID:          userID,
		AllowedGroupIDs: cloneWorkspaceInt64Slice(input.AllowedGroupIDs),
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

	assistantMessage, err := s.AppendAssistantMessage(ctx, userID, WorkspaceAppendAssistantMessageInput{
		ConversationID: input.ConversationID,
		MessageType:    assistantResponse.MessageType,
		Content:        assistantResponse.Content,
		Model:          firstNonEmptyWorkspaceValue(assistantResponse.Model, userMessage.Model),
		Intent:         firstNonEmptyWorkspaceValue(assistantResponse.Intent, userMessage.Intent),
		Status:         normalizeWorkspaceAssistantResponseStatus(assistantResponse.Status, assistantResponse.Metadata),
		Metadata:       assistantResponse.Metadata,
	})
	if err != nil {
		return userMessage, nil, err
	}
	return userMessage, assistantMessage, nil
}

func (s *ChatWorkspaceService) AppendAssistantMessage(ctx context.Context, userID int64, input WorkspaceAppendAssistantMessageInput) (*WorkspaceMessage, error) {
	if s == nil || s.repo == nil || userID <= 0 || input.ConversationID <= 0 {
		return nil, ErrWorkspaceConversationNotFound
	}
	if _, err := s.repo.GetConversation(ctx, userID, input.ConversationID); err != nil {
		return nil, err
	}

	content := strings.TrimSpace(input.Content)
	messageType := normalizeWorkspaceMessageType(input.MessageType)
	model := strings.TrimSpace(input.Model)
	intent := normalizeWorkspaceIntent(input.Intent)
	status := normalizeWorkspaceMessageStatus(input.Status)

	if containsUnsafeInlinePayload(content) || metadataContainsUnsafeInlinePayload(input.Metadata) {
		return nil, ErrWorkspaceInvalidMessage
	}
	if !isAllowedWorkspaceModel(model) {
		return nil, ErrWorkspaceInvalidModel
	}
	if !isAllowedWorkspaceMessageStatus(status) {
		return nil, ErrWorkspaceInvalidMessage
	}
	if messageType == WorkspaceMessageTypeText && intent != WorkspaceIntentChat {
		if isDisabledWorkspaceIntent(intent) {
			return nil, ErrWorkspaceCapabilityDisabled
		}
		return nil, ErrWorkspaceInvalidIntent
	}
	if !isAllowedAssistantWorkspaceMessage(messageType, intent, status, content, input.Metadata) {
		return nil, ErrWorkspaceInvalidMessage
	}
	if utf8.RuneCountInString(content) > workspaceMaxContentLength {
		content = string([]rune(content)[:workspaceMaxContentLength])
	}

	return s.repo.AppendMessage(ctx, userID, WorkspaceAppendMessageInput{
		ConversationID: input.ConversationID,
		MessageType:    messageType,
		Role:           WorkspaceRoleAssistant,
		Content:        content,
		Model:          model,
		Intent:         intent,
		Status:         status,
		Metadata:       input.Metadata,
	}, "")
}

type WorkspaceUnavailableAssistantResponder struct{}

func (WorkspaceUnavailableAssistantResponder) GenerateAssistantResponse(ctx context.Context, input WorkspaceAssistantResponseInput) (WorkspaceAssistantResponse, error) {
	return WorkspaceProviderAssistantResponder{Adapter: WorkspaceProviderUnavailableAdapter{}}.GenerateAssistantResponse(ctx, input)
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

func normalizeWorkspaceMessageStatus(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return WorkspaceMessageStatusCompleted
	}
	return value
}

func normalizeWorkspaceAssistantResponseStatus(value string, metadata map[string]any) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	switch strings.TrimSpace(workspaceMetadataString(metadata, "status")) {
	case WorkspaceMessageStatusPending:
		return WorkspaceMessageStatusPending
	case WorkspaceMessageStatusCompleted:
		return WorkspaceMessageStatusCompleted
	case WorkspaceMessageStatusFailed, "unavailable":
		return WorkspaceMessageStatusFailed
	default:
		return ""
	}
}

func isAllowedWorkspaceMessageStatus(value string) bool {
	switch strings.TrimSpace(value) {
	case WorkspaceMessageStatusPending, WorkspaceMessageStatusCompleted, WorkspaceMessageStatusFailed:
		return true
	default:
		return false
	}
}

func firstNonEmptyWorkspaceValue(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func isAllowedWorkspaceModel(model string) bool {
	normalized := strings.TrimSpace(model)
	if normalized == "" || utf8.RuneCountInString(normalized) > 128 {
		return false
	}
	for _, r := range normalized {
		if r >= 'a' && r <= 'z' {
			continue
		}
		if r >= 'A' && r <= 'Z' {
			continue
		}
		if r >= '0' && r <= '9' {
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

func workspaceSelectedModelResolutionBlocksMessage(resolution WorkspaceSelectedModelChannelCatalogResolution, planned WorkspacePlannedCapability) bool {
	if resolution.ModelCatalogSource == "" {
		return false
	}
	if resolution.ModelCatalogSource == WorkspaceModelCatalogSourceFakeGate &&
		resolution.Fake && resolution.TestOnly &&
		planned == WorkspacePlannedCapabilityImageGeneration {
		return false
	}
	if resolution.ModelCatalogSource != WorkspaceModelCatalogSourceRealChannel {
		return true
	}
	if resolution.BlockReason != "" {
		return true
	}
	if !resolution.UserAllowed || !resolution.GroupAllowed {
		return true
	}
	if resolution.PricingStatus != "" && resolution.PricingStatus != WorkspaceSelectedModelPricingConfigured {
		return true
	}
	return !resolution.CapabilityMatched
}

func isDisabledWorkspaceIntent(intent string) bool {
	switch strings.TrimSpace(intent) {
	case "image_generation", "image", "image_edit", "vision", "file_analysis", "web", "memory", "toolbox", "tools", "document_analysis":
		return true
	default:
		return false
	}
}

func isAllowedAssistantWorkspaceMessage(messageType, intent, status, content string, metadata map[string]any) bool {
	switch messageType {
	case WorkspaceMessageTypeText:
		return content != "" && intent == WorkspaceIntentChat
	case WorkspaceMessageTypeImage:
		return intent == WorkspaceIntentImageGeneration && workspaceImageMetadataMatchesStatus(status, content, metadata)
	default:
		return false
	}
}

func workspaceImageMetadataMatchesStatus(status, content string, metadata map[string]any) bool {
	if metadata == nil {
		return false
	}
	if strings.TrimSpace(workspaceMetadataString(metadata, "result_type")) != "image" {
		return false
	}
	switch status {
	case WorkspaceMessageStatusPending:
		return true
	case WorkspaceMessageStatusFailed:
		return content != "" ||
			strings.TrimSpace(workspaceMetadataString(metadata, "error_code")) != "" ||
			strings.TrimSpace(workspaceMetadataString(metadata, "error_message")) != ""
	default:
		return workspaceImageMetadataHasAsset(metadata)
	}
}

func workspaceImageMetadataHasAsset(metadata map[string]any) bool {
	assets, ok := metadata["assets"].([]any)
	if !ok || len(assets) == 0 {
		return false
	}
	for _, asset := range assets {
		item, ok := asset.(map[string]any)
		if !ok {
			continue
		}
		if strings.TrimSpace(workspaceMetadataString(item, "url")) != "" {
			return true
		}
	}
	return false
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

func metadataContainsUserAssetPayload(metadata map[string]any) bool {
	for key, value := range metadata {
		if isWorkspaceUserAssetMetadataKey(key) {
			return true
		}
		if nested, ok := value.(map[string]any); ok && metadataContainsUserAssetPayload(nested) {
			return true
		}
		if items, ok := value.([]any); ok {
			for _, item := range items {
				if nested, ok := item.(map[string]any); ok && metadataContainsUserAssetPayload(nested) {
					return true
				}
			}
		}
	}
	return false
}

func isWorkspaceUserAssetMetadataKey(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "asset", "assets", "asset_id", "asset_ids", "assetid", "assetids",
		"attachment", "attachments", "attachment_id", "attachment_ids",
		"file", "files", "file_id", "file_ids",
		"image", "images", "image_url", "image_urls", "imageurl", "imageurls",
		"preview_url", "preview_urls", "url", "urls":
		return true
	default:
		return false
	}
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
