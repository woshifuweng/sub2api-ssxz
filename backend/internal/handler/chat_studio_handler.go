package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const (
	chatStudioDefaultModel = "gpt-5.5"
	chatStudioMaxMessages  = 40
	chatStudioMaxContent   = 12000
)

type ChatStudioHandler struct {
	apiKeyService       *service.APIKeyService
	subscriptionService *service.SubscriptionService
	openAIGateway       *OpenAIGatewayHandler
	cfg                 *config.Config
}

type chatStudioMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatStudioRequest struct {
	Model       string              `json:"model"`
	Messages    []chatStudioMessage `json:"messages"`
	Temperature *float64            `json:"temperature,omitempty"`
	MaxTokens   *int                `json:"max_tokens,omitempty"`
}

func NewChatStudioHandler(
	apiKeyService *service.APIKeyService,
	subscriptionService *service.SubscriptionService,
	openAIGateway *OpenAIGatewayHandler,
	cfg *config.Config,
) *ChatStudioHandler {
	return &ChatStudioHandler{
		apiKeyService:       apiKeyService,
		subscriptionService: subscriptionService,
		openAIGateway:       openAIGateway,
		cfg:                 cfg,
	}
}

func (h *ChatStudioHandler) Complete(c *gin.Context) {
	h.CompleteGateway(gatewayctx.FromGin(c))
}

func (h *ChatStudioHandler) CompleteGateway(c gatewayctx.GatewayContext) {
	if h == nil || h.openAIGateway == nil || h.apiKeyService == nil {
		response.ErrorContext(imageStudioResponder{ctx: c}, http.StatusServiceUnavailable, "AI chat is not available")
		return
	}

	subject, ok := middleware2.GetAuthSubjectFromGatewayContext(c)
	if !ok {
		response.ErrorContext(imageStudioResponder{ctx: c}, http.StatusUnauthorized, "User not authenticated")
		return
	}

	req, err := parseChatStudioRequest(c)
	if err != nil {
		response.ErrorContext(imageStudioResponder{ctx: c}, http.StatusBadRequest, err.Error())
		return
	}

	apiKey, err := h.selectChatStudioAPIKey(c.Request().Context(), subject.UserID, req.Model)
	if err != nil {
		response.ErrorContext(imageStudioResponder{ctx: c}, http.StatusBadRequest, err.Error())
		return
	}

	body, err := buildChatStudioGatewayBody(req)
	if err != nil {
		response.ErrorContext(imageStudioResponder{ctx: c}, http.StatusBadRequest, err.Error())
		return
	}

	upstreamReq := cloneRequestForChatStudioGateway(c.Request(), EndpointChatCompletions, body, apiKey.Key)
	c.SetRequest(upstreamReq)
	ApplyInboundEndpointContext(c)

	if !middleware2.ApplyAPIKeyAuthWithSubscriptionContext(h.apiKeyService, h.subscriptionService, h.cfg, c) {
		return
	}

	h.openAIGateway.ChatCompletionsGateway(c)
}

func parseChatStudioRequest(c gatewayctx.GatewayContext) (*chatStudioRequest, error) {
	if c == nil || c.Request() == nil {
		return nil, fmt.Errorf("missing request")
	}
	defer c.Request().Body.Close()

	var req chatStudioRequest
	decoder := json.NewDecoder(io.LimitReader(c.Request().Body, 1<<20))
	if err := decoder.Decode(&req); err != nil {
		return nil, fmt.Errorf("invalid request body")
	}

	req.Model = normalizeChatStudioModel(req.Model)
	if len(req.Messages) == 0 {
		return nil, fmt.Errorf("message is required")
	}
	if len(req.Messages) > chatStudioMaxMessages {
		req.Messages = req.Messages[len(req.Messages)-chatStudioMaxMessages:]
	}

	cleanMessages := make([]chatStudioMessage, 0, len(req.Messages)+1)
	for _, msg := range req.Messages {
		role := normalizeChatStudioRole(msg.Role)
		content := truncateStudioText(msg.Content, chatStudioMaxContent)
		if strings.TrimSpace(content) == "" {
			continue
		}
		cleanMessages = append(cleanMessages, chatStudioMessage{
			Role:    role,
			Content: content,
		})
	}
	if len(cleanMessages) == 0 {
		return nil, fmt.Errorf("message is required")
	}
	req.Messages = prependChatStudioSystemPrompt(cleanMessages)
	return &req, nil
}

func (h *ChatStudioHandler) selectChatStudioAPIKey(ctx context.Context, userID int64, model string) (*service.APIKey, error) {
	keys, _, err := h.apiKeyService.List(ctx, userID, pagination.PaginationParams{Page: 1, PageSize: 100}, service.APIKeyListFilters{Status: service.StatusAPIKeyActive})
	if err != nil {
		return nil, fmt.Errorf("failed to load API keys")
	}
	for i := range keys {
		key := &keys[i]
		if key == nil || !key.IsActive() || strings.TrimSpace(key.Key) == "" {
			continue
		}
		if !key.AllowsModel(model) {
			continue
		}
		if key.Group != nil && key.Group.Platform == service.PlatformOpenAI {
			return key, nil
		}
	}
	return nil, fmt.Errorf("please create an active OpenAI API key that allows %s", model)
}

func buildChatStudioGatewayBody(req *chatStudioRequest) ([]byte, error) {
	if req == nil {
		return nil, fmt.Errorf("missing request")
	}
	payload := map[string]any{
		"model":    req.Model,
		"messages": req.Messages,
		"stream":   false,
	}
	if req.Temperature != nil {
		payload["temperature"] = clampChatStudioTemperature(*req.Temperature)
	}
	if req.MaxTokens != nil && *req.MaxTokens > 0 {
		payload["max_tokens"] = *req.MaxTokens
	}
	return json.Marshal(payload)
}

func cloneRequestForChatStudioGateway(req *http.Request, endpoint string, body []byte, apiKey string) *http.Request {
	next := req.Clone(req.Context())
	next.Method = http.MethodPost
	next.Body = io.NopCloser(bytes.NewReader(body))
	next.ContentLength = int64(len(body))
	next.Header = req.Header.Clone()
	next.Header.Set("Authorization", "Bearer "+apiKey)
	next.Header.Set("Content-Type", "application/json")
	next.Header.Set("Content-Length", fmt.Sprintf("%d", len(body)))
	if next.URL != nil {
		copiedURL := *next.URL
		copiedURL.Path = endpoint
		copiedURL.RawPath = ""
		copiedURL.RawQuery = ""
		next.URL = &copiedURL
	}
	next.RequestURI = endpoint
	return next
}

func normalizeChatStudioModel(value string) string {
	value = strings.TrimSpace(value)
	switch value {
	case "gpt-5.5", "gpt-5.4", "gpt-5.2", "gpt-5.4-mini":
		return value
	default:
		return chatStudioDefaultModel
	}
}

func normalizeChatStudioRole(value string) string {
	switch strings.TrimSpace(value) {
	case "system", "assistant", "user":
		return value
	default:
		return "user"
	}
}

func prependChatStudioSystemPrompt(messages []chatStudioMessage) []chatStudioMessage {
	system := chatStudioMessage{
		Role:    "system",
		Content: "You are a helpful AI assistant inside SSXZ AI. Answer clearly, solve the user's task directly, and keep replies practical.",
	}
	out := make([]chatStudioMessage, 0, len(messages)+1)
	out = append(out, system)
	for _, msg := range messages {
		if msg.Role == "system" {
			continue
		}
		out = append(out, msg)
	}
	return out
}

func clampChatStudioTemperature(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 2 {
		return 2
	}
	return value
}
