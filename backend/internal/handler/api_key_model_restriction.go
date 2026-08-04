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

func canonicalCustomerGatewayModelForAPIKey(apiKey *service.APIKey, model string) (canonical string, enforced bool, available bool) {
	if apiKey == nil || apiKey.Group == nil || !service.IsCustomerGatewayPlatform(apiKey.Group.Platform) {
		return strings.TrimSpace(model), false, true
	}
	canonical, available = service.CanonicalCustomerGatewayModel(apiKey.Group.Platform, model)
	return canonical, true, available
}

// apiKeyBlocksModel determines whether a model is disabled for any platform
// reachable by the API key. It covers both the plural and legacy singular
// group fields; keys without group information fail closed across customer
// gateway platforms.
func apiKeyBlocksModel(apiKey *service.APIKey, model string) bool {
	if apiKey == nil {
		return false
	}
	platforms := make(map[string]struct{}, 2)
	for _, group := range apiKey.Groups {
		if group != nil && strings.TrimSpace(group.Platform) != "" {
			platforms[group.Platform] = struct{}{}
		}
	}
	if apiKey.Group != nil && strings.TrimSpace(apiKey.Group.Platform) != "" {
		platforms[apiKey.Group.Platform] = struct{}{}
	}
	if len(platforms) == 0 {
		platforms[service.PlatformOpenAI] = struct{}{}
		platforms[service.PlatformAnthropic] = struct{}{}
	}
	for platform := range platforms {
		if service.IsBlockedCustomerGatewayModel(platform, model) {
			return true
		}
	}
	return false
}

func apiKeyAllowsRequestedModel(apiKey *service.APIKey, model string) bool {
	if apiKeyBlocksModel(apiKey, model) {
		return false
	}
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

func customerGatewayModelUnavailableMessage(model string) string {
	model = strings.TrimSpace(model)
	if model == "" {
		return "Requested model is not available"
	}
	return fmt.Sprintf("Model %q is not available", model)
}

func apiKeyModelNotAllowedMessage(model string) string {
	model = strings.TrimSpace(model)
	if model == "" {
		return "Requested model is not allowed for this API key"
	}
	return fmt.Sprintf("Model %q is not allowed for this API key", model)
}

func apiKeyModelRestrictionMessage(apiKey *service.APIKey, model string) string {
	if apiKeyBlocksModel(apiKey, model) {
		return customerGatewayModelUnavailableMessage(model)
	}
	return apiKeyModelNotAllowedMessage(model)
}

func filterClaudeModelsForAPIKey(apiKey *service.APIKey, models []claude.Model) []claude.Model {
	if apiKey == nil {
		return models
	}
	out := make([]claude.Model, 0, len(models))
	for _, model := range models {
		if apiKeyAllowsRequestedModel(apiKey, model.ID) {
			out = append(out, model)
		}
	}
	return out
}

func filterOpenAIModelsForAPIKey(apiKey *service.APIKey, models []openai.Model) []openai.Model {
	if apiKey == nil {
		return models
	}
	out := make([]openai.Model, 0, len(models))
	for _, model := range models {
		if apiKeyAllowsRequestedModel(apiKey, model.ID) {
			out = append(out, model)
		}
	}
	return out
}

func filterGeminiModelsForAPIKey(apiKey *service.APIKey, models []gemini.Model) []gemini.Model {
	if apiKey == nil {
		return models
	}
	out := make([]gemini.Model, 0, len(models))
	for _, model := range models {
		if apiKeyAllowsRequestedModel(apiKey, model.Name) {
			out = append(out, model)
		}
	}
	return out
}

func filterAntigravityClaudeModelsForAPIKey(apiKey *service.APIKey, models []antigravity.ClaudeModel) []antigravity.ClaudeModel {
	if apiKey == nil {
		return models
	}
	out := make([]antigravity.ClaudeModel, 0, len(models))
	for _, model := range models {
		if apiKeyAllowsRequestedModel(apiKey, model.ID) {
			out = append(out, model)
		}
	}
	return out
}

func filterAntigravityGeminiModelsForAPIKey(apiKey *service.APIKey, models []antigravity.GeminiModel) []antigravity.GeminiModel {
	if apiKey == nil {
		return models
	}
	out := make([]antigravity.GeminiModel, 0, len(models))
	for _, model := range models {
		if apiKeyAllowsRequestedModel(apiKey, model.Name) {
			out = append(out, model)
		}
	}
	return out
}
