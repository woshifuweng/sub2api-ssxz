package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGatewayUsageSubscriptionKeyWithoutActiveSubscriptionIsExplicitlyInvalid(t *testing.T) {
	c, rec := newGatewayUsageTestContext(subscriptionAPIKeyForUsageTest())

	h := &GatewayHandler{}
	h.Usage(c)

	require.Equal(t, http.StatusOK, rec.Code)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	require.Equal(t, "unrestricted", payload["mode"])
	require.Equal(t, false, payload["isValid"])
	require.Equal(t, "subscription_not_found", payload["status"])
	require.Equal(t, float64(0), payload["remaining"])
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
	require.Equal(t, float64(8), payload["remaining"])
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
