package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func newGatewayRoutesTestRouter(redisClient *redis.Client) *gin.Engine {
	return newGatewayRoutesTestRouterWithPlatform(redisClient, service.PlatformOpenAI)
}

func newGatewayRoutesTestRouterWithPlatform(redisClient *redis.Client, platform string) *gin.Engine {
	return newGatewayRoutesTestRouterWithGroup(redisClient, &service.Group{Platform: platform})
}

func newGatewayRoutesTestRouterWithGroup(redisClient *redis.Client, group *service.Group) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	RegisterGatewayRoutes(
		router,
		&handler.Handlers{
			Gateway:       &handler.GatewayHandler{},
			OpenAIGateway: &handler.OpenAIGatewayHandler{},
			SoraGateway:   &handler.SoraGatewayHandler{},
		},
		servermiddleware.APIKeyAuthMiddleware(func(c *gin.Context) {
			groupID := int64(1)
			c.Set(string(servermiddleware.ContextKeyAPIKey), &service.APIKey{
				GroupID: &groupID,
				Group:   group,
			})
			c.Next()
		}),
		nil,
		nil,
		nil,
		nil,
		&config.Config{},
		redisClient,
	)

	return router
}

func TestGatewayRoutesCountTokensHonorsExplicitMessagesDispatchOptIn(t *testing.T) {
	router := newGatewayRoutesTestRouterWithGroup(nil, &service.Group{
		Platform:              service.PlatformAnthropic,
		AllowMessagesDispatch: true,
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/messages/count_tokens", strings.NewReader(`{"model":"claude-opus-4-8"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
	require.Contains(t, w.Body.String(), "Token counting is not supported")
}

func TestGatewayRoutesOpenAIResponsesCompactPathIsRegistered(t *testing.T) {
	router := newGatewayRoutesTestRouter(nil)

	for _, path := range []string{"/v1/responses/compact", "/responses/compact"} {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"model":"gpt-5"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		require.NotEqual(t, http.StatusNotFound, w.Code, "path=%s should hit OpenAI responses handler", path)
	}
}

func TestGatewayRoutesOpenAIImagesPathsAreRegistered(t *testing.T) {
	router := newGatewayRoutesTestRouter(nil)

	for _, path := range []string{
		"/v1/images/generations",
		"/v1/images/edits",
		"/images/generations",
		"/images/edits",
	} {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"model":"gpt-image-1","prompt":"draw a cat"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)
		require.NotEqual(t, http.StatusNotFound, w.Code, "path=%s should hit OpenAI images handler", path)
	}
}

func TestGatewayRoutesApplyConsumptionRateLimit(t *testing.T) {
	redisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() {
		_ = redisClient.Close()
	})
	router := newGatewayRoutesTestRouterWithPlatform(redisClient, service.PlatformAnthropic)

	req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(`{"model":"gpt-image-1","prompt":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "192.0.2.10:1234"
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	count, err := redisServer.Get("rate_limit:" + GatewayConsumptionRateLimitKey + ":192.0.2.10")
	require.NoError(t, err)
	require.Equal(t, "1", count)
}

func TestExecutableGatewayRoutesAttachConsumptionRateLimitToAPIKeyRoutes(t *testing.T) {
	routes := ExecutableGatewayRoutes(&handler.Handlers{
		Gateway:       &handler.GatewayHandler{},
		OpenAIGateway: &handler.OpenAIGatewayHandler{},
		SoraGateway:   &handler.SoraGatewayHandler{},
	})

	for _, route := range routes {
		if !routeRequiresGatewayAPIKey(route) {
			require.NotContains(t, route.Middleware, gatewayConsumptionRateLimitTag, "path=%s should not be rate limited as API-key gateway route", route.Path)
			continue
		}
		require.Contains(t, route.Middleware, gatewayConsumptionRateLimitTag, "path=%s should include gateway consumption rate limit", route.Path)
		require.Equal(t, gatewayConsumptionRateLimitTag, route.Middleware[0], "path=%s should rate limit before auth/provider work", route.Path)
	}
}

func TestExecutableGatewayRoutesOpenAIImagesPathsAreRegistered(t *testing.T) {
	routes := ExecutableGatewayRoutes(&handler.Handlers{
		Gateway:       &handler.GatewayHandler{},
		OpenAIGateway: &handler.OpenAIGatewayHandler{},
	})

	expected := map[string]bool{
		"/v1/images/generations": false,
		"/v1/images/edits":       false,
		"/images/generations":    false,
		"/images/edits":          false,
	}

	for _, route := range routes {
		if route.Method != http.MethodPost {
			continue
		}
		if _, ok := expected[route.Path]; ok {
			expected[route.Path] = true
		}
	}

	for path, found := range expected {
		require.True(t, found, "path=%s should be present in executable gateway routes", path)
	}
}

func TestExecutableGatewayRoutesSoraMediaPathsAreRegistered(t *testing.T) {
	routes := ExecutableGatewayRoutes(&handler.Handlers{
		Gateway:       &handler.GatewayHandler{},
		OpenAIGateway: &handler.OpenAIGatewayHandler{},
		SoraGateway:   &handler.SoraGatewayHandler{},
	})

	expected := map[string]bool{
		"/sora/media/*filepath":        false,
		"/sora/media-signed/*filepath": false,
	}

	for _, route := range routes {
		if route.Method != http.MethodGet {
			continue
		}
		if _, ok := expected[route.Path]; ok {
			expected[route.Path] = true
		}
	}

	for path, found := range expected {
		require.True(t, found, "path=%s should be present in executable gateway routes", path)
	}
}

func TestOpenAIMessagesDispatchHonorsExplicitOptInForAnthropicGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"model":"claude-opus-4-8"}`))
	ctx.Set(string(servermiddleware.ContextKeyAPIKey), &service.APIKey{
		Group: &service.Group{
			Platform:              service.PlatformAnthropic,
			AllowMessagesDispatch: true,
		},
	})

	openAIMessagesDispatchGateway(&handler.Handlers{
		OpenAIGateway: &handler.OpenAIGatewayHandler{},
	})(gatewayctx.FromGin(ctx))

	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	require.Contains(t, recorder.Body.String(), "User context not found")
}

func TestOpenAICountTokensDispatchHonorsExplicitOptInForAnthropicGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/messages/count_tokens", strings.NewReader(`{"model":"claude-opus-4-8"}`))
	ctx.Set(string(servermiddleware.ContextKeyAPIKey), &service.APIKey{
		Group: &service.Group{
			Platform:              service.PlatformAnthropic,
			AllowMessagesDispatch: true,
		},
	})

	openAICountTokensDispatchGateway(&handler.Handlers{})(gatewayctx.FromGin(ctx))

	require.Equal(t, http.StatusNotFound, recorder.Code)
	require.Contains(t, recorder.Body.String(), "Token counting is not supported")
}

func TestShouldUseOpenAIMessagesDispatch(t *testing.T) {
	tests := []struct {
		name  string
		group *service.Group
		want  bool
	}{
		{name: "missing group", group: nil, want: false},
		{name: "openai preserves existing dispatch", group: &service.Group{Platform: service.PlatformOpenAI}, want: true},
		{name: "anthropic remains native by default", group: &service.Group{Platform: service.PlatformAnthropic}, want: false},
		{name: "anthropic explicit opt in", group: &service.Group{Platform: service.PlatformAnthropic, AllowMessagesDispatch: true}, want: true},
		{name: "other platforms cannot opt in", group: &service.Group{Platform: service.PlatformGemini, AllowMessagesDispatch: true}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, shouldUseOpenAIMessagesDispatch(&service.APIKey{Group: tt.group}))
		})
	}
}
