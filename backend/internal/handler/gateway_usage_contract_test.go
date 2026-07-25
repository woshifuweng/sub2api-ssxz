package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type gatewayUsageAPIKeyRepoStub struct {
	service.APIKeyRepository
	apiKey *service.APIKey
	err    error
	calls  int
}

func (s *gatewayUsageAPIKeyRepoStub) GetByID(_ context.Context, _ int64) (*service.APIKey, error) {
	s.calls++
	return s.apiKey, s.err
}

func TestGatewayUsageSubscriptionKeyWithoutActiveSubscriptionIsExplicitlyInvalid(t *testing.T) {
	c, rec := newGatewayUsageTestContext(subscriptionAPIKeyForUsageTest())

	h := &GatewayHandler{}
	h.Usage(c)

	require.Equal(t, http.StatusOK, rec.Code)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	require.Equal(t, "unrestricted", payload["mode"])
	require.Equal(t, false, payload["isValid"])
	require.Equal(t, false, payload["is_active"])
	require.Equal(t, "subscription_not_found", payload["status"])
	require.Equal(t, float64(0), payload["remaining"])
	require.Equal(t, float64(0), payload["balance"])
	require.Equal(t, "USD", payload["unit"])
	requireUsageQuotaRemaining(t, payload, 0)
	require.Equal(t, "No active subscription found for this group", payload["message"])
	require.NotContains(t, payload, "subscription")
}

func TestGatewayUsageSubscriptionKeyWithActiveSubscriptionReturnsRemainingAndLimits(t *testing.T) {
	c, rec := newGatewayUsageTestContext(subscriptionAPIKeyForUsageTest())
	c.Set(string(middleware.ContextKeySubscription), &service.UserSubscription{
		UserID:          42,
		GroupID:         7,
		Status:          service.SubscriptionStatusActive,
		ExpiresAt:       time.Now().Add(24 * time.Hour),
		DailyUsageUSD:   2,
		WeeklyUsageUSD:  30,
		MonthlyUsageUSD: 90,
	})

	h := &GatewayHandler{}
	h.Usage(c)

	require.Equal(t, http.StatusOK, rec.Code)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	require.Equal(t, "unrestricted", payload["mode"])
	require.Equal(t, true, payload["isValid"])
	require.Equal(t, true, payload["is_active"])
	require.Equal(t, float64(8), payload["remaining"])
	require.Equal(t, float64(8), payload["balance"])
	require.Equal(t, "USD", payload["unit"])
	requireUsageQuotaRemaining(t, payload, 8)
	require.NotContains(t, payload, "status")
	require.NotContains(t, payload, "message")

	subscription, ok := payload["subscription"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(2), subscription["daily_usage_usd"])
	require.Equal(t, float64(10), subscription["daily_limit_usd"])
	require.Equal(t, float64(30), subscription["weekly_usage_usd"])
	require.Equal(t, float64(50), subscription["weekly_limit_usd"])
	require.Equal(t, float64(90), subscription["monthly_usage_usd"])
	require.Equal(t, float64(200), subscription["monthly_limit_usd"])
}

func TestGatewayUsageQuotaLimitedReturnsCCSwitchBalanceAliases(t *testing.T) {
	apiKey := quotaLimitedAPIKeyForUsageTest()
	c, rec := newGatewayUsageTestContext(apiKey)

	(&GatewayHandler{}).Usage(c)

	require.Equal(t, http.StatusOK, rec.Code)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	require.Equal(t, true, payload["is_active"])
	require.Equal(t, float64(7.5), payload["remaining"])
	require.Equal(t, float64(7.5), payload["balance"])
	require.Equal(t, "USD", payload["unit"])
	requireUsageQuotaRemaining(t, payload, 7.5)
}

func TestGatewayUsageQuotaLimitedRefreshesQuotaAfterSettlement(t *testing.T) {
	cached := quotaLimitedAPIKeyForUsageTest()
	fresh := *cached
	fresh.QuotaUsed = 4.25
	repo := &gatewayUsageAPIKeyRepoStub{apiKey: &fresh}
	c, rec := newGatewayUsageTestContext(cached)

	h := &GatewayHandler{
		apiKeyService: service.NewAPIKeyService(repo, nil, nil, nil, nil, nil, nil),
	}
	h.Usage(c)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 1, repo.calls)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	require.Equal(t, float64(5.75), payload["remaining"])
	require.Equal(t, float64(5.75), payload["balance"])
	requireUsageQuotaRemaining(t, payload, 5.75)
}

func TestGatewayUsageReturnsSafeErrorWhenQuotaRefreshFails(t *testing.T) {
	cached := quotaLimitedAPIKeyForUsageTest()
	repo := &gatewayUsageAPIKeyRepoStub{err: errors.New("database unavailable")}
	c, rec := newGatewayUsageTestContext(cached)

	h := &GatewayHandler{
		apiKeyService: service.NewAPIKeyService(repo, nil, nil, nil, nil, nil, nil),
	}
	h.Usage(c)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.NotContains(t, rec.Body.String(), "database unavailable")
}

func TestGatewayUsageRejectsQuotaRefreshForDifferentOwner(t *testing.T) {
	cached := quotaLimitedAPIKeyForUsageTest()
	fresh := *cached
	fresh.UserID++
	repo := &gatewayUsageAPIKeyRepoStub{apiKey: &fresh}
	c, rec := newGatewayUsageTestContext(cached)

	h := &GatewayHandler{
		apiKeyService: service.NewAPIKeyService(repo, nil, nil, nil, nil, nil, nil),
	}
	h.Usage(c)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func requireUsageQuotaRemaining(t *testing.T, payload map[string]any, expected float64) {
	t.Helper()

	quota, ok := payload["quota"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, expected, quota["remaining"])
}

func newGatewayUsageTestContext(apiKey *service.APIKey) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/usage", nil)
	c.Set(string(middleware.ContextKeyAPIKey), apiKey)
	c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: apiKey.UserID})
	return c, rec
}

func subscriptionAPIKeyForUsageTest() *service.APIKey {
	dailyLimit := 10.0
	weeklyLimit := 50.0
	monthlyLimit := 200.0
	return &service.APIKey{
		ID:     101,
		UserID: 42,
		Status: service.StatusAPIKeyActive,
		Group: &service.Group{
			ID:               7,
			Name:             "Pro Subscription",
			SubscriptionType: service.SubscriptionTypeSubscription,
			DailyLimitUSD:    &dailyLimit,
			WeeklyLimitUSD:   &weeklyLimit,
			MonthlyLimitUSD:  &monthlyLimit,
		},
	}
}

func quotaLimitedAPIKeyForUsageTest() *service.APIKey {
	return &service.APIKey{
		ID:        102,
		UserID:    42,
		Status:    service.StatusAPIKeyActive,
		Quota:     10,
		QuotaUsed: 2.5,
	}
}
