package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	"github.com/Wei-Shaw/sub2api/internal/pkg/gemini"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func TestAPIKeyAllowsRequestedModel(t *testing.T) {
	apiKey := &service.APIKey{AllowedModels: []string{"gpt-5.4", "gemini-2.5-pro"}}

	if !apiKeyAllowsRequestedModel(apiKey, "gpt-5.4") {
		t.Fatalf("expected exact model to be allowed")
	}
	if !apiKeyAllowsRequestedModel(apiKey, "models/gemini-2.5-pro") {
		t.Fatalf("expected models/ prefixed gemini model to be allowed")
	}
	if apiKeyAllowsRequestedModel(apiKey, "gpt-5.5") {
		t.Fatalf("expected non-allowlisted model to be rejected")
	}
}

func TestFilterOpenAIModelsForAPIKey(t *testing.T) {
	apiKey := &service.APIKey{AllowedModels: []string{"gpt-5.4"}}
	models := []openai.Model{
		{ID: "gpt-5.4"},
		{ID: "gpt-5.5"},
	}

	filtered := filterOpenAIModelsForAPIKey(apiKey, models)
	if len(filtered) != 1 || filtered[0].ID != "gpt-5.4" {
		t.Fatalf("unexpected filtered models: %#v", filtered)
	}
}

func TestFilterGeminiModelsForAPIKey(t *testing.T) {
	apiKey := &service.APIKey{AllowedModels: []string{"gemini-2.5-pro"}}
	models := []gemini.Model{
		{Name: "models/gemini-2.5-pro"},
		{Name: "models/gemini-2.5-flash"},
	}

	filtered := filterGeminiModelsForAPIKey(apiKey, models)
	if len(filtered) != 1 || filtered[0].Name != "models/gemini-2.5-pro" {
		t.Fatalf("unexpected filtered models: %#v", filtered)
	}
}

func TestModelsGatewayFiltersKiroModelsForAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	apiKey := &service.APIKey{
		AllowedModels: []string{"claude-sonnet-4-6"},
		Group:         &service.Group{ID: 10, Platform: service.PlatformKiro},
	}

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	ctx.Set(string(middleware.ContextKeyAPIKey), apiKey)

	handler := &GatewayHandler{}
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
	if payload.Object != "list" {
		t.Fatalf("unexpected object: %q", payload.Object)
	}
	if len(payload.Data) != 1 || payload.Data[0].ID != "claude-sonnet-4-6" {
		t.Fatalf("unexpected filtered kiro models: %#v", payload.Data)
	}
}
