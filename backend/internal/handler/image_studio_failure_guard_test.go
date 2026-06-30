package handler

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
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

type imageStudioGatewayUsageRepo struct {
	service.UsageLogRepository

	calls int
}

func (r *imageStudioGatewayUsageRepo) Create(context.Context, *service.UsageLog) (bool, error) {
	r.calls++
	return true, nil
}

type imageStudioGatewayBillingRepo struct {
	service.UsageBillingRepository

	calls int
}

func (r *imageStudioGatewayBillingRepo) Apply(context.Context, *service.UsageBillingCommand) (*service.UsageBillingApplyResult, error) {
	r.calls++
	return &service.UsageBillingApplyResult{Applied: true}, nil
}

type imageStudioGatewayUserRepo struct {
	service.UserRepository

	user        *service.User
	deductCalls int
}

func (r *imageStudioGatewayUserRepo) GetByID(context.Context, int64) (*service.User, error) {
	return r.user, nil
}

func (r *imageStudioGatewayUserRepo) DeductBalance(context.Context, int64, float64) error {
	r.deductCalls++
	return nil
}

type imageStudioGatewayAccountRepo struct {
	service.AccountRepository

	accounts []service.Account
}

func (r *imageStudioGatewayAccountRepo) ListSchedulableByGroupIDAndPlatform(_ context.Context, _ int64, platform string) ([]service.Account, error) {
	return r.byPlatform(platform), nil
}

func (r *imageStudioGatewayAccountRepo) ListSchedulableByPlatform(_ context.Context, platform string) ([]service.Account, error) {
	return r.byPlatform(platform), nil
}

func (r *imageStudioGatewayAccountRepo) ListSchedulableUngroupedByPlatform(_ context.Context, platform string) ([]service.Account, error) {
	return r.byPlatform(platform), nil
}

func (r *imageStudioGatewayAccountRepo) byPlatform(platform string) []service.Account {
	out := make([]service.Account, 0, len(r.accounts))
	for _, account := range r.accounts {
		if account.Platform == platform && account.IsSchedulable() {
			out = append(out, account)
		}
	}
	return out
}

type imageStudioGatewayUpstream struct {
	status int
	body   string
	calls  int
}

func (u *imageStudioGatewayUpstream) Do(*http.Request, string, int64, int) (*http.Response, error) {
	u.calls++
	return &http.Response{
		StatusCode: u.status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewBufferString(u.body)),
	}, nil
}

func (u *imageStudioGatewayUpstream) DoWithTLS(req *http.Request, proxyURL string, accountID int64, accountConcurrency int, _ bool) (*http.Response, error) {
	return u.Do(req, proxyURL, accountID, accountConcurrency)
}

func TestOpenAIImagesGateway_UpstreamFailureDoesNotRecordUsageOrBilling(t *testing.T) {
	gin.SetMode(gin.TestMode)

	groupID := int64(301)
	user := &service.User{
		ID:          701,
		Status:      service.StatusActive,
		Role:        "user",
		Balance:     10,
		Concurrency: 0,
	}
	group := &service.Group{
		ID:       groupID,
		Name:     "image-test",
		Platform: service.PlatformOpenAI,
		Status:   service.StatusActive,
	}
	apiKey := &service.APIKey{
		ID:      801,
		UserID:  user.ID,
		Key:     "test-image-studio-key",
		Status:  service.StatusAPIKeyActive,
		User:    user,
		GroupID: &groupID,
		Group:   group,
	}
	account := service.Account{
		ID:          901,
		Name:        "image-upstream",
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
		status: http.StatusBadRequest,
		body:   `{"error":{"message":"invalid image request"}}`,
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

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewReader([]byte(`{"model":"gpt-image-2","prompt":"draw a cat"}`)))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set(string(middleware.ContextKeyAPIKey), apiKey)
	ctx.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: user.ID, Concurrency: user.Concurrency})
	ctx.Set(string(middleware.ContextKeyUserRole), user.Role)

	handler.ImagesGateway(gatewayctx.FromGin(ctx))

	require.Equal(t, 1, upstream.calls)
	require.Equal(t, http.StatusBadGateway, rec.Code)
	require.Contains(t, rec.Body.String(), "Upstream request failed")
	require.Equal(t, 0, usageRepo.calls)
	require.Equal(t, 0, billingRepo.calls)
	require.Equal(t, 0, userRepo.deductCalls)
}

func TestOpenAIImagesGateway_EstimatedCostOverBalanceDoesNotCallUpstream(t *testing.T) {
	gin.SetMode(gin.TestMode)

	groupID := int64(302)
	imagePrice2K := 2.0
	user := &service.User{
		ID:          702,
		Status:      service.StatusActive,
		Role:        "user",
		Balance:     1,
		Concurrency: 0,
	}
	group := &service.Group{
		ID:             groupID,
		Name:           "image-cost-gate",
		Platform:       service.PlatformOpenAI,
		Status:         service.StatusActive,
		RateMultiplier: 1,
		ImagePrice2K:   &imagePrice2K,
	}
	apiKey := &service.APIKey{
		ID:      802,
		UserID:  user.ID,
		Key:     "test-image-cost-gate-key",
		Status:  service.StatusAPIKeyActive,
		User:    user,
		GroupID: &groupID,
		Group:   group,
	}
	account := service.Account{
		ID:          902,
		Name:        "image-upstream",
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
		body:   `{"data":[{"url":"https://cdn.example.test/image.png"}]}`,
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

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewReader([]byte(`{"model":"gpt-image-2","prompt":"draw a cat"}`)))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set(string(middleware.ContextKeyAPIKey), apiKey)
	ctx.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: user.ID, Concurrency: user.Concurrency})
	ctx.Set(string(middleware.ContextKeyUserRole), user.Role)

	handler.ImagesGateway(gatewayctx.FromGin(ctx))

	require.Equal(t, 0, upstream.calls)
	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Contains(t, rec.Body.String(), "insufficient balance")
	require.Equal(t, 0, usageRepo.calls)
	require.Equal(t, 0, billingRepo.calls)
	require.Equal(t, 0, userRepo.deductCalls)
}

func TestImageStudioGenerateGateway_IdempotencyReplayDoesNotCallUpstreamOrDuplicateHistory(t *testing.T) {
	gin.SetMode(gin.TestMode)

	groupID := int64(303)
	user := &service.User{
		ID:          703,
		Status:      service.StatusActive,
		Role:        "user",
		Balance:     10,
		Concurrency: 0,
	}
	group := &service.Group{
		ID:       groupID,
		Name:     "image-idempotency",
		Platform: service.PlatformOpenAI,
		Status:   service.StatusActive,
	}
	apiKey := &service.APIKey{
		ID:      803,
		UserID:  user.ID,
		Key:     "test-image-idempotency-key",
		Status:  service.StatusAPIKeyActive,
		User:    user,
		GroupID: &groupID,
		Group:   group,
	}
	account := service.Account{
		ID:          903,
		Name:        "image-upstream",
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
	genRepo := &imageStudioTestGenRepo{}
	upstream := &imageStudioGatewayUpstream{
		status: http.StatusOK,
		body:   `{"data":[{"url":"https://cdn.example.test/image.png"}]}`,
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

	apiKeyService := service.NewAPIKeyService(&imageStudioAPIKeyRepo{key: apiKey}, nil, nil, nil, nil, nil, cfg)
	handler := NewImageStudioHandler(
		apiKeyService,
		nil,
		NewOpenAIGatewayHandler(
			gatewayService,
			service.NewConcurrencyService(nil),
			billingCacheService,
			apiKeyService,
			nil,
			nil,
			cfg,
		),
		cfg,
		service.NewSoraGenerationService(genRepo, nil, nil),
		nil,
	)
	idempotencyConfig := service.DefaultIdempotencyConfig()
	idempotencyConfig.ObserveOnly = false
	service.SetDefaultIdempotencyCoordinator(service.NewIdempotencyCoordinator(newUserMemoryIdempotencyRepoStub(), nil, idempotencyConfig))
	t.Cleanup(func() {
		service.SetDefaultIdempotencyCoordinator(nil)
	})

	router := gin.New()
	router.Use(withUserSubject(user.ID))
	router.POST("/api/v1/image-studio/generate", handler.Generate)
	callGenerate := func() *httptest.ResponseRecorder {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		require.NoError(t, writer.WriteField("product_name", "skincare cream"))
		require.NoError(t, writer.WriteField("selling_points", "clean ecommerce product image"))
		require.NoError(t, writer.WriteField("model", imageStudioModel))
		require.NoError(t, writer.Close())
		req := httptest.NewRequest(http.MethodPost, "/api/v1/image-studio/generate", &body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Idempotency-Key", "image-generate-once")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	first := callGenerate()
	second := callGenerate()

	require.Equal(t, http.StatusOK, first.Code)
	require.Equal(t, http.StatusOK, second.Code)
	require.Equal(t, "true", second.Header().Get("X-Idempotency-Replayed"))
	require.Equal(t, first.Body.String(), second.Body.String())
	require.Equal(t, 1, upstream.calls)
	require.Len(t, genRepo.created, 1)
}

func TestPersistImageStudioWork_DoesNotCreateHistoryForFailedOrTruncatedCapture(t *testing.T) {
	tests := []struct {
		name    string
		capture *imageStudioCaptureContext
	}{
		{
			name:    "upstream_4xx",
			capture: &imageStudioCaptureContext{status: http.StatusBadRequest},
		},
		{
			name:    "upstream_5xx",
			capture: &imageStudioCaptureContext{status: http.StatusInternalServerError},
		},
		{
			name:    "truncated_success_body",
			capture: &imageStudioCaptureContext{status: http.StatusOK, truncated: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &imageStudioTestGenRepo{}
			handler := &ImageStudioHandler{
				genService: service.NewSoraGenerationService(repo, nil, nil),
			}
			tt.capture.capture([]byte(`{"data":[{"url":"https://cdn.example.com/work.png"}]}`))

			handler.persistImageStudioWork(context.Background(), tt.capture, 7, 42, imageStudioModel, "product prompt")

			require.Empty(t, repo.created)
		})
	}
}

var _ service.UsageLogRepository = (*imageStudioGatewayUsageRepo)(nil)
var _ service.UsageBillingRepository = (*imageStudioGatewayBillingRepo)(nil)
var _ service.UserRepository = (*imageStudioGatewayUserRepo)(nil)
var _ service.AccountRepository = (*imageStudioGatewayAccountRepo)(nil)
var _ service.HTTPUpstream = (*imageStudioGatewayUpstream)(nil)
