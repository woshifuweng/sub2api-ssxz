package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type WorkspaceSub2APITextBridge struct {
	apiKeyService       *service.APIKeyService
	subscriptionService *service.SubscriptionService
	openAIGateway       *OpenAIGatewayHandler
	cfg                 *config.Config
}

func NewWorkspaceSub2APITextBridge(
	apiKeyService *service.APIKeyService,
	subscriptionService *service.SubscriptionService,
	openAIGateway *OpenAIGatewayHandler,
	cfg *config.Config,
) *WorkspaceSub2APITextBridge {
	return &WorkspaceSub2APITextBridge{
		apiKeyService:       apiKeyService,
		subscriptionService: subscriptionService,
		openAIGateway:       openAIGateway,
		cfg:                 cfg,
	}
}

func (b *WorkspaceSub2APITextBridge) CompleteWorkspaceText(ctx context.Context, input service.WorkspaceSub2APITextBridgeInput) (service.WorkspaceSub2APITextBridgeResult, error) {
	if b == nil || b.apiKeyService == nil || b.openAIGateway == nil {
		return service.WorkspaceSub2APITextBridgeResult{}, fmt.Errorf("sub2api chat bridge is not available")
	}
	model := strings.TrimSpace(input.Model)
	if model == "" {
		return service.WorkspaceSub2APITextBridgeResult{}, fmt.Errorf("model is required")
	}
	apiKey, err := b.selectWorkspaceAPIKey(ctx, input.UserID, model)
	if err != nil {
		return service.WorkspaceSub2APITextBridgeResult{}, err
	}
	body, err := buildWorkspaceSub2APITextBridgeBody(model, input.Content)
	if err != nil {
		return service.WorkspaceSub2APITextBridgeResult{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, EndpointChatCompletions, bytes.NewReader(body))
	if err != nil {
		return service.WorkspaceSub2APITextBridgeResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey.Key)
	req.Header.Set("User-Agent", "sub2api-workspace-internal")

	recorder := newWorkspaceSub2APIResponseRecorder()
	transportCtx := gatewayctx.NewNative(req, recorder, nil, "")
	if !middleware2.ApplyAPIKeyAuthWithSubscriptionContext(b.apiKeyService, b.subscriptionService, b.cfg, transportCtx) {
		return service.WorkspaceSub2APITextBridgeResult{}, fmt.Errorf("sub2api api key auth failed with status %d", recorder.statusCode())
	}

	b.openAIGateway.ChatCompletionsGateway(transportCtx)
	if status := recorder.statusCode(); status < 200 || status >= 300 {
		return service.WorkspaceSub2APITextBridgeResult{}, fmt.Errorf("sub2api chat completion failed with status %d", status)
	}
	return parseWorkspaceSub2APITextBridgeResult(recorder.bodyBytes(), model)
}

func (b *WorkspaceSub2APITextBridge) selectWorkspaceAPIKey(ctx context.Context, userID int64, model string) (*service.APIKey, error) {
	keys, _, err := b.apiKeyService.List(ctx, userID, pagination.PaginationParams{Page: 1, PageSize: 100}, service.APIKeyListFilters{Status: service.StatusAPIKeyActive})
	if err != nil {
		return nil, fmt.Errorf("failed to load API keys")
	}
	usableKeys := 0
	for i := range keys {
		key := &keys[i]
		if key == nil || !key.IsActive() || strings.TrimSpace(key.Key) == "" {
			continue
		}
		if key.Group == nil || key.Group.Platform != service.PlatformOpenAI {
			continue
		}
		usableKeys++
		if !key.AllowsModel(model) {
			continue
		}
		return key, nil
	}
	if usableKeys == 0 {
		return nil, service.ErrWorkspaceSub2APITextBridgeMissingAPIKey
	}
	return nil, service.ErrWorkspaceSub2APITextBridgeModelNotAllowed
}

func buildWorkspaceSub2APITextBridgeBody(model, content string) ([]byte, error) {
	payload := map[string]any{
		"model": strings.TrimSpace(model),
		"messages": []map[string]string{{
			"role":    "user",
			"content": strings.TrimSpace(content),
		}},
		"stream": false,
	}
	return json.Marshal(payload)
}

func parseWorkspaceSub2APITextBridgeResult(body []byte, fallbackModel string) (service.WorkspaceSub2APITextBridgeResult, error) {
	var payload struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			InputTokens        int `json:"input_tokens"`
			OutputTokens       int `json:"output_tokens"`
			PromptTokens       int `json:"prompt_tokens"`
			CompletionTokens   int `json:"completion_tokens"`
			InputTokensDetails struct {
				CachedTokens int `json:"cached_tokens"`
			} `json:"input_tokens_details"`
			PromptTokensDetails struct {
				CachedTokens int `json:"cached_tokens"`
			} `json:"prompt_tokens_details"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return service.WorkspaceSub2APITextBridgeResult{}, fmt.Errorf("parse sub2api chat completion response: %w", err)
	}
	content := ""
	if len(payload.Choices) > 0 {
		content = strings.TrimSpace(payload.Choices[0].Message.Content)
	}
	if content == "" {
		return service.WorkspaceSub2APITextBridgeResult{}, fmt.Errorf("sub2api chat completion returned empty content")
	}
	billableUsagePresent := workspaceSub2APITextBridgeBillableUsagePresent(
		payload.Usage.InputTokens,
		payload.Usage.OutputTokens,
		payload.Usage.PromptTokens,
		payload.Usage.CompletionTokens,
	)
	additionalFields := map[string]any{}
	if billableUsagePresent {
		additionalFields["usage_status"] = "usage_payload_present"
		additionalFields["billing_status"] = "sub2api_gateway_usage_path"
	} else {
		additionalFields["usage_status"] = "usage_missing"
		additionalFields["billing_status"] = "billing_not_recorded"
	}
	return service.WorkspaceSub2APITextBridgeResult{
		Content:          content,
		Model:            firstNonEmptyHandlerValue(payload.Model, fallbackModel),
		UpstreamModel:    firstNonEmptyHandlerValue(payload.Model, fallbackModel),
		ProviderName:     service.WorkspaceSub2APITextBridgeName,
		RequestID:        payload.ID,
		UsageRecorded:    billableUsagePresent,
		BillingManaged:   billableUsagePresent,
		ProviderCalled:   true,
		AdditionalFields: additionalFields,
	}, nil
}

func workspaceSub2APITextBridgeBillableUsagePresent(values ...int) bool {
	for _, value := range values {
		if value > 0 {
			return true
		}
	}
	return false
}

type workspaceSub2APIResponseRecorder struct {
	mu          sync.Mutex
	header      http.Header
	body        bytes.Buffer
	status      int
	wroteHeader bool
}

func newWorkspaceSub2APIResponseRecorder() *workspaceSub2APIResponseRecorder {
	return &workspaceSub2APIResponseRecorder{header: make(http.Header)}
}

func (r *workspaceSub2APIResponseRecorder) Header() http.Header {
	return r.header
}

func (r *workspaceSub2APIResponseRecorder) WriteHeader(statusCode int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.wroteHeader {
		return
	}
	r.status = statusCode
	r.wroteHeader = true
}

func (r *workspaceSub2APIResponseRecorder) Write(payload []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.wroteHeader {
		r.status = http.StatusOK
		r.wroteHeader = true
	}
	return r.body.Write(payload)
}

func (r *workspaceSub2APIResponseRecorder) Flush() {}

func (r *workspaceSub2APIResponseRecorder) Written() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.wroteHeader
}

func (r *workspaceSub2APIResponseRecorder) Size() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.body.Len()
}

func (r *workspaceSub2APIResponseRecorder) statusCode() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.status == 0 {
		return http.StatusOK
	}
	return r.status
}

func (r *workspaceSub2APIResponseRecorder) bodyBytes() []byte {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]byte(nil), r.body.Bytes()...)
}

func firstNonEmptyHandlerValue(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
