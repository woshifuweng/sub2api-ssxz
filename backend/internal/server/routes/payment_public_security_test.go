//go:build unit

package routes

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func newPublicPaymentSecurityRouter(t *testing.T, publicRPM int) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	settings := service.NewSettingService(&channelMonitorRouteSettingRepoStub{
		values: map[string]string{
			service.SettingKeyPanelRateLimitSettings: `{"enabled":true,"user_rpm":0,"heavy_rpm":0,"exempt_admin":true,"public_ip_rpm":` + strconv.Itoa(publicRPM) + `}`,
		},
	}, &config.Config{})
	redisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })
	panelLimiter := middleware.NewPanelRateLimiter(redisClient, settings)

	pass := func(c *gin.Context) { c.Next() }
	router := gin.New()
	router.Use(gin.Recovery())
	v1 := router.Group("/api/v1")
	RegisterPaymentRoutes(
		v1,
		handler.NewPaymentHandler(nil, nil),
		handler.NewPaymentWebhookHandler(nil, nil),
		admin.NewPaymentHandler(nil, nil),
		pass,
		pass,
		pass,
		pass,
		settings,
		panelLimiter,
	)
	return router
}

func performPublicPaymentRequest(router *gin.Engine, path, body string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	request.RemoteAddr = "203.0.113.9:12345"
	router.ServeHTTP(recorder, request)
	return recorder
}

func TestPaymentPublicRoutesRejectOversizedJSON(t *testing.T) {
	router := newPublicPaymentSecurityRouter(t, 100)
	oversized := `{"out_trade_no":"` + strings.Repeat("x", publicPaymentMaxBodySize) + `"}`

	response := performPublicPaymentRequest(router, "/api/v1/payment/public/orders/verify", oversized)

	require.Equal(t, http.StatusRequestEntityTooLarge, response.Code)
	require.Contains(t, response.Body.String(), "REQUEST_BODY_TOO_LARGE")
}

func TestPaymentPublicRoutesUsePublicIPRateLimit(t *testing.T) {
	router := newPublicPaymentSecurityRouter(t, 1)

	first := performPublicPaymentRequest(router, "/api/v1/payment/public/orders/verify", `{`)
	second := performPublicPaymentRequest(router, "/api/v1/payment/public/orders/resolve", `{`)

	require.Equal(t, http.StatusBadRequest, first.Code)
	require.Equal(t, http.StatusTooManyRequests, second.Code)
	require.NotEmpty(t, second.Header().Get("Retry-After"))
	require.Contains(t, second.Body.String(), "RATE_LIMITED")
}
