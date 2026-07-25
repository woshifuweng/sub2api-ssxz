//go:build unit

package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func TestModelsGatewayCustomerPlatformDoesNotFallBackToStaticModels(t *testing.T) {
	gin.SetMode(gin.TestMode)

	groupID := int64(11)
	apiKey := &service.APIKey{
		GroupID: &groupID,
		Group:   &service.Group{ID: groupID, Platform: service.PlatformAnthropic},
	}
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	ctx.Set(string(middleware.ContextKeyAPIKey), apiKey)

	handler := &GatewayHandler{gatewayService: newMinimalGatewayService(&stubAccountRepoForHandler{})}
	handler.ModelsGateway(gatewayctx.FromGin(ctx))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var payload struct {
		Object string         `json:"object"`
		Data   []claude.Model `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Object != "list" || len(payload.Data) != 0 {
		t.Fatalf("expected an empty fail-closed model list, got %#v", payload)
	}
}
