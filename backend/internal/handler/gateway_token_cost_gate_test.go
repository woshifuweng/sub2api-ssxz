package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGatewayMessagesGateway_EstimatedCostOverBalanceReturnsBeforeAccountSelection(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler, apiKey, cleanup := newGatewayTokenCostGateHandlerForTest(t)
	t.Cleanup(cleanup)
	body := []byte(`{"model":"claude-sonnet-4","messages":[{"role":"user","content":"hello"}],"max_tokens":200000}`)

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set(string(middleware.ContextKeyAPIKey), apiKey)
	ctx.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: apiKey.User.ID, Concurrency: apiKey.User.Concurrency})
	ctx.Set(string(middleware.ContextKeyUserRole), apiKey.User.Role)

	handler.MessagesGateway(gatewayctx.FromGin(ctx))

	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Contains(t, rec.Body.String(), "insufficient balance")
}

func TestGeminiV1BetaModelsGateway_EstimatedCostOverBalanceReturnsBeforeAccountSelection(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler, apiKey, cleanup := newGatewayTokenCostGateHandlerForTest(t)
	t.Cleanup(cleanup)
	apiKey.Group.Platform = service.PlatformGemini
	body := []byte(`{"contents":[{"role":"user","parts":[{"text":"hello"}]}],"generationConfig":{"maxOutputTokens":200000}}`)

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-3.1-pro:generateContent", bytes.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Params = gin.Params{{Key: "modelAction", Value: "/gemini-3.1-pro:generateContent"}}
	ctx.Set(string(middleware.ContextKeyAPIKey), apiKey)
	ctx.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: apiKey.User.ID, Concurrency: apiKey.User.Concurrency})
	ctx.Set(string(middleware.ContextKeyUserRole), apiKey.User.Role)

	handler.GeminiV1BetaModelsGateway(gatewayctx.FromGin(ctx))

	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Contains(t, rec.Body.String(), "insufficient balance")
}

func TestGatewayTokenBillingEligibility_EstimatedCostOverBalanceBlocksGenericRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler, apiKey, cleanup := newGatewayTokenCostGateHandlerForTest(t)
	t.Cleanup(cleanup)
	body := []byte(`{"model":"claude-sonnet-4","messages":[{"role":"user","content":"hello"}],"max_tokens":200000}`)
	parsedReq, err := service.ParseGatewayRequest(body, domain.PlatformAnthropic)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))

	ok := handler.checkGatewayTokenBillingEligibilityContext(
		gatewayctx.FromGin(ctx),
		nil,
		apiKey,
		nil,
		parsedReq,
		false,
		"gateway.test",
		0,
		0,
		handler.handleStreamingAwareErrorContext,
	)

	require.False(t, ok)
	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Contains(t, rec.Body.String(), "insufficient balance")
}

func TestGatewayTokenBillingEligibility_EstimatedCostOverBalanceBlocksGeminiRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler, apiKey, cleanup := newGatewayTokenCostGateHandlerForTest(t)
	t.Cleanup(cleanup)
	body := []byte(`{"contents":[{"role":"user","parts":[{"text":"hello"}]}],"generationConfig":{"maxOutputTokens":200000}}`)
	parsedReq, err := service.ParseGatewayRequest(body, domain.PlatformGemini)
	require.NoError(t, err)
	parsedReq.Model = "gemini-3.1-pro"

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-3.1-pro:generateContent", bytes.NewReader(body))

	ok := handler.checkGatewayTokenBillingEligibilityContext(
		gatewayctx.FromGin(ctx),
		nil,
		apiKey,
		nil,
		parsedReq,
		false,
		"gemini.test",
		200000,
		2.0,
		func(c gatewayctx.GatewayContext, status int, _ string, message string, _ bool) {
			googleErrorContext(c, status, message)
		},
	)

	require.False(t, ok)
	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Contains(t, rec.Body.String(), "insufficient balance")
}

func newGatewayTokenCostGateHandlerForTest(t *testing.T) (*GatewayHandler, *service.APIKey, func()) {
	t.Helper()

	cfg := &config.Config{}
	user := &service.User{
		ID:          715,
		Status:      service.StatusActive,
		Role:        "user",
		Balance:     1,
		Concurrency: 0,
	}
	group := &service.Group{
		ID:             315,
		Name:           "token-cost-gate",
		Platform:       service.PlatformAnthropic,
		Status:         service.StatusActive,
		RateMultiplier: 1,
	}
	apiKey := &service.APIKey{
		ID:     815,
		UserID: user.ID,
		Key:    "test-generic-token-cost-gate-key",
		Status: service.StatusAPIKeyActive,
		User:   user,
		Group:  group,
	}
	userRepo := &imageStudioGatewayUserRepo{user: user}
	billingCacheService := service.NewBillingCacheService(nil, userRepo, nil, nil, cfg)
	gatewayService := service.NewGatewayService(
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
		service.NewConcurrencyService(nil),
		service.NewBillingService(cfg, nil),
		nil,
		billingCacheService,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)
	handler := &GatewayHandler{
		gatewayService:      gatewayService,
		billingCacheService: billingCacheService,
		concurrencyHelper:   NewConcurrencyHelper(service.NewConcurrencyService(nil), SSEPingFormatClaude, 0),
		cfg:                 cfg,
	}

	return handler, apiKey, func() {
		gatewayService.CloseBackgroundWorkers()
		billingCacheService.Stop()
	}
}
