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
	router.POST("/v1/chat/completions", ok)
	router.POST("/v1/messages", ok)
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

	for _, path := range []string{"/v1/models", "/models", "/v1/usage"} {
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

	for _, path := range []string{"/v1/chat/completions", "/v1/messages"} {
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

	// 非 GET 或非元数据路径一律不豁免。
	require.False(t, isReadOnlyGatewayMetadata(http.MethodPost, "/v1/models"))
	require.False(t, isReadOnlyGatewayMetadata(http.MethodGet, "/v1/chat/completions"))
	require.False(t, isReadOnlyGatewayMetadata(http.MethodGet, "/v1/models/extra"))
	require.False(t, isReadOnlyGatewayMetadata(http.MethodGet, "/v1beta/models"))
}
