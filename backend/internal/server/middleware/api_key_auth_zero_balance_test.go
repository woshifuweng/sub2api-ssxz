//go:build unit

package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// zeroBalanceTestRouter 挂载与网关一致的目标路由：只读元数据 + 推理端点。
func zeroBalanceTestRouter(t *testing.T) *gin.Engine {
	t.Helper()

	user := &service.User{
		ID:          7,
		Role:        service.RoleUser,
		Status:      service.StatusActive,
		Balance:     0, // 新注册用户：零余额
		Concurrency: 3,
	}
	apiKey := &service.APIKey{
		ID:     100,
		UserID: user.ID,
		Key:    "zero-balance-key",
		Status: service.StatusActive,
		User:   user,
	}
	apiKeyRepo := &stubApiKeyRepo{
		getByKey: func(ctx context.Context, key string) (*service.APIKey, error) {
			if key != apiKey.Key {
				return nil, service.ErrAPIKeyNotFound
			}
			clone := *apiKey
			return &clone, nil
		},
	}

	cfg := &config.Config{RunMode: config.RunModeStandard}
	apiKeyService := service.NewAPIKeyService(apiKeyRepo, nil, nil, nil, nil, nil, cfg)

	router := gin.New()
	router.Use(gin.HandlerFunc(NewAPIKeyAuthMiddleware(apiKeyService, nil, cfg)))
	ok := func(c *gin.Context) {
		_, hasKey := c.Get(string(ContextKeyAPIKey))
		require.True(t, hasKey)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
	router.GET("/v1/models", ok)
	router.GET("/models", ok)
	router.GET("/v1/usage", ok)
	router.GET("/antigravity/v1/models", ok)
	router.GET("/antigravity/v1/usage", ok)
	router.GET("/antigravity/models", ok)
	router.GET("/sora/v1/models", ok)
	router.POST("/v1/chat/completions", ok)
	router.POST("/v1/messages", ok)
	router.POST("/antigravity/v1/messages", ok)
	return router
}

func zeroBalanceRequest(t *testing.T, router *gin.Engine, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("x-api-key", "zero-balance-key")
	router.ServeHTTP(w, req)
	return w
}

// 余额 0 时只读元数据端点必须放行：余额是消费门槛，不是身份门槛。
func TestZeroBalanceAllowsReadOnlyMetadataEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := zeroBalanceTestRouter(t)

	for _, path := range []string{
		"/v1/models", "/models", "/v1/usage",
		"/antigravity/v1/models", "/antigravity/v1/usage", "/antigravity/models",
		"/sora/v1/models",
	} {
		t.Run(path, func(t *testing.T) {
			w := zeroBalanceRequest(t, router, http.MethodGet, path)
			require.Equal(t, http.StatusOK, w.Code, "GET %s 不应因零余额被拒", path)
			require.Contains(t, w.Body.String(), `"ok":true`)
		})
	}
}

// 真正产生费用的推理端点在余额 0 时仍然必须被拦。
func TestZeroBalanceStillBlocksInferenceEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := zeroBalanceTestRouter(t)

	for _, path := range []string{"/v1/chat/completions", "/v1/messages", "/antigravity/v1/messages"} {
		t.Run(path, func(t *testing.T) {
			w := zeroBalanceRequest(t, router, http.MethodPost, path)
			require.Equal(t, http.StatusForbidden, w.Code, "POST %s 零余额应返回 403", path)
			require.Contains(t, w.Body.String(), "INSUFFICIENT_BALANCE")
		})
	}
}

func TestIsReadOnlyGatewayMetadata(t *testing.T) {
	require.True(t, isReadOnlyGatewayMetadata(http.MethodGet, "/v1/models"))
	require.True(t, isReadOnlyGatewayMetadata(http.MethodGet, "/v1/models/"))
	require.True(t, isReadOnlyGatewayMetadata(http.MethodGet, "/models"))
	require.True(t, isReadOnlyGatewayMetadata(http.MethodGet, "/v1/usage"))
	require.True(t, isReadOnlyGatewayMetadata(http.MethodGet, "/antigravity/v1/models"))
	require.True(t, isReadOnlyGatewayMetadata(http.MethodGet, "/antigravity/v1/usage"))
	require.True(t, isReadOnlyGatewayMetadata(http.MethodGet, "/antigravity/models"))
	require.True(t, isReadOnlyGatewayMetadata(http.MethodGet, "/sora/v1/models"))

	// 非 GET 或非元数据路径一律不豁免。
	require.False(t, isReadOnlyGatewayMetadata(http.MethodPost, "/v1/models"))
	require.False(t, isReadOnlyGatewayMetadata(http.MethodGet, "/v1/chat/completions"))
	require.False(t, isReadOnlyGatewayMetadata(http.MethodGet, "/v1/models/extra"))
	require.False(t, isReadOnlyGatewayMetadata(http.MethodGet, "/v1beta/models"))
	require.False(t, isReadOnlyGatewayMetadata(http.MethodPost, "/antigravity/v1/models"))
	require.False(t, isReadOnlyGatewayMetadata(http.MethodGet, "/antigravity/v1/messages"))
	require.False(t, isReadOnlyGatewayMetadata(http.MethodGet, "/sora/v1/chat/completions"))
	// 平台前缀同样只认精确路径，不吞子路径。
	require.False(t, isReadOnlyGatewayMetadata(http.MethodGet, "/antigravity/v1/models/extra"))
	require.False(t, isReadOnlyGatewayMetadata(http.MethodGet, "/sora/v1/models/extra"))
}

// ---------------------------------------------------------------------------
// Google（Gemini /v1beta）中间件的零余额豁免
// ---------------------------------------------------------------------------

// zeroBalanceGoogleRouter 按 routes/gateway.go 的真实注册形态挂载 Gemini 路由：
// GET /models、GET /models/:model 与推理入口 POST /models/*modelAction 同前缀。
func zeroBalanceGoogleRouter(t *testing.T) *gin.Engine {
	t.Helper()

	user := &service.User{
		ID:          8,
		Role:        service.RoleUser,
		Status:      service.StatusActive,
		Balance:     0, // 零余额
		Concurrency: 3,
	}
	apiKey := &service.APIKey{
		ID:     101,
		UserID: user.ID,
		Key:    "zero-balance-google-key",
		Status: service.StatusActive,
		User:   user,
	}
	apiKeyRepo := &stubApiKeyRepo{
		getByKey: func(ctx context.Context, key string) (*service.APIKey, error) {
			if key != apiKey.Key {
				return nil, service.ErrAPIKeyNotFound
			}
			clone := *apiKey
			return &clone, nil
		},
	}

	cfg := &config.Config{RunMode: config.RunModeStandard}
	apiKeyService := service.NewAPIKeyService(apiKeyRepo, nil, nil, nil, nil, nil, cfg)

	router := gin.New()
	router.Use(APIKeyAuthWithSubscriptionGoogle(apiKeyService, nil, cfg))
	ok := func(c *gin.Context) {
		_, hasKey := c.Get(string(ContextKeyAPIKey))
		require.True(t, hasKey)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
	for _, g := range []*gin.RouterGroup{
		router.Group("/v1beta"),
		router.Group("/antigravity/v1beta"),
	} {
		g.GET("/models", ok)
		g.GET("/models/:model", ok)
		g.POST("/models/*modelAction", ok)
	}
	return router
}

func zeroBalanceGoogleRequest(t *testing.T, router *gin.Engine, method, path, key string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)
	if key != "" {
		req.Header.Set("x-goog-api-key", key)
	}
	router.ServeHTTP(w, req)
	return w
}

// 零余额下 Gemini 模型列表/单模型元数据必须放行。
func TestZeroBalanceGoogleAllowsModelMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := zeroBalanceGoogleRouter(t)

	for _, path := range []string{
		"/v1beta/models",
		"/v1beta/models/gemini-2.0-flash",
		"/antigravity/v1beta/models",
		"/antigravity/v1beta/models/gemini-2.0-flash",
	} {
		t.Run(path, func(t *testing.T) {
			w := zeroBalanceGoogleRequest(t, router, http.MethodGet, path, "zero-balance-google-key")
			require.Equal(t, http.StatusOK, w.Code, "GET %s 不应因零余额被拒", path)
			require.Contains(t, w.Body.String(), `"ok":true`)
		})
	}
}

// 防回归核心：同前缀的推理入口 POST /models/{model}:{action} 零余额必须仍被拦。
func TestZeroBalanceGoogleStillBlocksInference(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := zeroBalanceGoogleRouter(t)

	for _, path := range []string{
		"/v1beta/models/gemini-2.0-flash:generateContent",
		"/v1beta/models/gemini-2.0-flash:streamGenerateContent",
		"/antigravity/v1beta/models/gemini-2.0-flash:generateContent",
	} {
		t.Run(path, func(t *testing.T) {
			w := zeroBalanceGoogleRequest(t, router, http.MethodPost, path, "zero-balance-google-key")
			require.Equal(t, http.StatusForbidden, w.Code, "POST %s 零余额应返回 403", path)
			require.Contains(t, w.Body.String(), "Insufficient account balance")
		})
	}
}

// 身份门槛不许放松：无效 key 读元数据仍是 401。
func TestZeroBalanceGoogleInvalidKeyStillUnauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := zeroBalanceGoogleRouter(t)

	w := zeroBalanceGoogleRequest(t, router, http.MethodGet, "/v1beta/models", "no-such-key")
	require.Equal(t, http.StatusUnauthorized, w.Code)

	w = zeroBalanceGoogleRequest(t, router, http.MethodGet, "/v1beta/models", "")
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestIsGoogleReadOnlyModelMetadata(t *testing.T) {
	require.True(t, isGoogleReadOnlyModelMetadata(http.MethodGet, "/v1beta/models"))
	require.True(t, isGoogleReadOnlyModelMetadata(http.MethodGet, "/v1beta/models/"))
	require.True(t, isGoogleReadOnlyModelMetadata(http.MethodGet, "/v1beta/models/gemini-2.0-flash"))
	require.True(t, isGoogleReadOnlyModelMetadata(http.MethodGet, "/antigravity/v1beta/models"))
	require.True(t, isGoogleReadOnlyModelMetadata(http.MethodGet, "/antigravity/v1beta/models/gemini-2.0-flash"))

	// 非 GET 一律不豁免——推理入口与元数据同前缀。
	require.False(t, isGoogleReadOnlyModelMetadata(http.MethodPost, "/v1beta/models"))
	require.False(t, isGoogleReadOnlyModelMetadata(http.MethodPost, "/v1beta/models/gemini-2.0-flash:generateContent"))
	// GET 带 ':' action 形态也不豁免（%3A 解码后同样落到这里）。
	require.False(t, isGoogleReadOnlyModelMetadata(http.MethodGet, "/v1beta/models/gemini-2.0-flash:generateContent"))
	// 多段路径不豁免。
	require.False(t, isGoogleReadOnlyModelMetadata(http.MethodGet, "/v1beta/models/gemini-2.0-flash/extra"))
	// 其他 /v1beta 端点不豁免。
	require.False(t, isGoogleReadOnlyModelMetadata(http.MethodGet, "/v1beta/cachedContents"))
	// 标准中间件管辖的路径不归这个 helper。
	require.False(t, isGoogleReadOnlyModelMetadata(http.MethodGet, "/v1/models"))
}
