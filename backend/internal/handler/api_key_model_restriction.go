package handler

import (
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/antigravity"
	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	"github.com/Wei-Shaw/sub2api/internal/pkg/gemini"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func normalizeAPIKeyAllowedModelID(model string) string {
	model = strings.TrimSpace(model)
	model = strings.TrimPrefix(model, "models/")
	if strings.EqualFold(model, "gpt-5.6") {
		return "gpt-5.6-sol"
	}
	return model
}

func apiKeyAllowsRequestedModel(apiKey *service.APIKey, model string) bool {
	if apiKey == nil || len(apiKey.AllowedModels) == 0 {
		return true
	}
	requested := normalizeAPIKeyAllowedModelID(model)
	if requested == "" {
		return true
	}
	for _, allowed := range apiKey.AllowedModels {
		if normalizeAPIKeyAllowedModelID(allowed) == requested {
			return true
		}
	}
	return false
}

func apiKeyModelNotAllowedMessage(model string) string {
	if model = strings.TrimSpace(model); model != "" {
		return fmt.Sprintf("Model %q is not allowed for this API key", model)
	}
	return "Requested model is not allowed for this API key"
}

func filterModelIDsForAPIKey(apiKey *service.APIKey, models []string) []string {
	out := make([]string, 0, len(models))
	for _, model := range models {
		if apiKeyAllowsRequestedModel(apiKey, model) {
			out = append(out, model)
		}
	}
	return out
}

func filterClaudeModelsForAPIKey(apiKey *service.APIKey, models []claude.Model) []claude.Model {
	out := make([]claude.Model, 0, len(models))
	for _, model := range models {
		if apiKeyAllowsRequestedModel(apiKey, model.ID) {
			out = append(out, model)
		}
	}
	return out
}

func filterOpenAIModelsForAPIKey(apiKey *service.APIKey, models []openai.Model) []openai.Model {
	out := make([]openai.Model, 0, len(models))
	for _, model := range models {
		if apiKeyAllowsRequestedModel(apiKey, model.ID) {
			out = append(out, model)
		}
	}
	return out
}

func filterGeminiModelsForAPIKey(apiKey *service.APIKey, models []gemini.Model) []gemini.Model {
	out := make([]gemini.Model, 0, len(models))
	for _, model := range models {
		if apiKeyAllowsRequestedModel(apiKey, model.Name) {
			out = append(out, model)
		}
	}
	return out
}

func filterAntigravityGeminiModelsForAPIKey(apiKey *service.APIKey, models []antigravity.GeminiModel) []antigravity.GeminiModel {
	out := make([]antigravity.GeminiModel, 0, len(models))
	for _, model := range models {
		if apiKeyAllowsRequestedModel(apiKey, model.Name) {
			out = append(out, model)
		}
	}
	return out
}
