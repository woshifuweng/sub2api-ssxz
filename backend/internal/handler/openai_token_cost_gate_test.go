package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestOpenAIChatCompletionsGateway_EstimatedCostOverBalanceDoesNotCallUpstream(t *testing.T) {
	gin.SetMode(gin.TestMode)

	groupID := int64(303)
	user := &service.User{
		ID:          703,
		Status:      service.StatusActive,
		Role:        "user",
		Balance:     1,
		Concurrency: 0,
	}
	group := &service.Group{
		ID:             groupID,
		Name:           "token-cost-gate",
		Platform:       service.PlatformOpenAI,
		Status:         service.StatusActive,
		RateMultiplier: 1,
	}
	apiKey := &service.APIKey{
		ID:      803,
		UserID:  user.ID,
		Key:     "test-token-cost-gate-key",
		Status:  service.StatusAPIKeyActive,
		User:    user,
		GroupID: &groupID,
		Group:   group,
	}
	account := service.Account{
		ID:          903,
		Name:        "chat-upstream",
		Platform:    service.PlatformOpenAI,
		Type:        service.AccountTypeAPIKey,
		Status:      service.StatusActive,
		Schedulable: true,
		Concurrency: 0,
		Credentials: map[string]any{
			"api_key":  "test-upstream-key",
			"base_url": "https://api.example.test",
		},
	}

	cfg := &config.Config{}
	userRepo := &imageStudioGatewayUserRepo{user: user}
	usageRepo := &imageStudioGatewayUsageRepo{}
	billingRepo := &imageStudioGatewayBillingRepo{}
	upstream := &imageStudioGatewayUpstream{
		status: http.StatusOK,
		body:   `{"id":"chatcmpl-test","object":"chat.completion","model":"gpt-5.1","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`,
	}
	billingCacheService := service.NewBillingCacheService(nil, userRepo, nil, nil, cfg)
	gatewayService := service.NewOpenAIGatewayService(
		&imageStudioGatewayAccountRepo{accounts: []service.Account{account}},
		nil,
		usageRepo,
		billingRepo,
		userRepo,
		nil,
		nil,
		nil,
		cfg,
		nil,
		nil,
		service.NewBillingService(cfg, nil),
		nil,
		nil,
		billingCacheService,
		upstream,
		nil,
		nil,
	)
	t.Cleanup(gatewayService.CloseOpenAIWSPool)
	handler := NewOpenAIGatewayHandler(
		gatewayService,
		service.NewConcurrencyService(nil),
		billingCacheService,
		&service.APIKeyService{},
		nil,
		nil,
		cfg,
	)

	body := []byte(`{"model":"gpt-5.1","messages":[{"role":"user","content":"hello"}],"max_completion_tokens":200000}`)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set(string(middleware.ContextKeyAPIKey), apiKey)
	ctx.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: user.ID, Concurrency: user.Concurrency})
	ctx.Set(string(middleware.ContextKeyUserRole), user.Role)

	handler.ChatCompletionsGateway(gatewayctx.FromGin(ctx))

	require.Equal(t, 0, upstream.calls)
	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Contains(t, rec.Body.String(), "insufficient balance")
	require.Equal(t, 0, usageRepo.calls)
	require.Equal(t, 0, billingRepo.calls)
	require.Equal(t, 0, userRepo.deductCalls)
}

func TestOpenAITokenBillingEligibility_UnboundedRequestUsesSafetyBudget(t *testing.T) {
	gin.SetMode(gin.TestMode)

	groupID := int64(304)
	user := &service.User{
		ID:          704,
		Status:      service.StatusActive,
		Role:        "user",
		Balance:     1,
		Concurrency: 0,
	}
	group := &service.Group{
		ID:             groupID,
		Name:           "token-cost-gate",
		Platform:       service.PlatformOpenAI,
		Status:         service.StatusActive,
		RateMultiplier: 1,
	}
	apiKey := &service.APIKey{
		ID:      804,
		UserID:  user.ID,
		Key:     "test-token-cost-gate-unbounded-key",
		Status:  service.StatusAPIKeyActive,
		User:    user,
		GroupID: &groupID,
		Group:   group,
	}

	cfg := &config.Config{}
	userRepo := &imageStudioGatewayUserRepo{user: user}
	billingCacheService := service.NewBillingCacheService(nil, userRepo, nil, nil, cfg)
	gatewayService := service.NewOpenAIGatewayService(
		nil,
		nil,
		nil,
		nil,
		userRepo,
		nil,
		nil,
		nil,
		cfg,
		nil,
		nil,
		service.NewBillingService(cfg, nil),
		nil,
		nil,
		billingCacheService,
		nil,
		nil,
		nil,
	)
	t.Cleanup(gatewayService.CloseOpenAIWSPool)
	t.Cleanup(billingCacheService.Stop)
	handler := &OpenAIGatewayHandler{
		gatewayService:      gatewayService,
		billingCacheService: billingCacheService,
	}

	body := []byte(`{"model":"gpt-5.1","messages":[{"role":"user","content":"hello"}]}`)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))

	ok := handler.checkOpenAITokenBillingEligibilityContext(
		gatewayctx.FromGin(ctx),
		nil,
		apiKey,
		nil,
		"gpt-5.1",
		body,
		false,
		"openai.test",
		handler.handleStreamingAwareErrorContext,
	)

	require.False(t, ok)
	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Contains(t, rec.Body.String(), "insufficient balance")
}
