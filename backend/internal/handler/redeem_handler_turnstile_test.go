package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRedeemHandlerBlocksBeforeRedeemWhenTurnstileIsRequired(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &RedeemHandler{
		authService: service.NewAuthService(nil, nil, nil, nil, &config.Config{
			Server:    config.ServerConfig{Mode: "release"},
			Turnstile: config.TurnstileConfig{Required: true},
		}, nil, nil, nil, nil, nil, nil),
	}

	router := gin.New()
	router.Use(withUserSubject(42))
	router.POST("/api/v1/redeem", func(c *gin.Context) {
		h.RedeemGateway(gatewayctx.FromGin(c))
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/redeem", bytes.NewBufferString(`{"code":"TEST-CODE"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.NotEqual(t, http.StatusOK, rec.Code)
}
