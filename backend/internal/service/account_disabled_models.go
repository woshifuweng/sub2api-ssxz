package service

import (
	"fmt"
	"strings"
)

const openAIDisabledModelsKey = "openai_disabled_models"

// Explicit operational quarantine is independent of discovery mappings and
// temporary rate limits. Exact model IDs only: a blocked GPT-5.5 must not also
// disable GPT-5.6 or future models, even on authentication-only passthrough.
func (a *Account) IsOpenAIModelDisabled(model string) bool {
	if a == nil || !a.IsOpenAI() || a.Extra == nil {
		return false
	}
	model = strings.TrimSpace(model)
	if model == "" {
		return false
	}
	switch values := a.Extra[openAIDisabledModelsKey].(type) {
	case []string:
		for _, value := range values {
			if strings.TrimSpace(value) == model {
				return true
			}
		}
	case []any:
		for _, value := range values {
			if id, ok := value.(string); ok && strings.TrimSpace(id) == model {
				return true
			}
		}
	}
	return false
}

func ValidateOpenAIDisabledModelsExtra(extra map[string]any) error {
	raw, exists := extra[openAIDisabledModelsKey]
	if !exists {
		return nil
	}
	var models []string
	switch values := raw.(type) {
	case []string:
		models = values
	case []any:
		for _, value := range values {
			id, ok := value.(string)
			if !ok {
				return fmt.Errorf("%s must be an array of model IDs", openAIDisabledModelsKey)
			}
			models = append(models, id)
		}
	default:
		return fmt.Errorf("%s must be an array of model IDs", openAIDisabledModelsKey)
	}
	if len(models) > 200 {
		return fmt.Errorf("%s supports at most 200 model IDs", openAIDisabledModelsKey)
	}
	for _, model := range models {
		if model == "" || model != strings.TrimSpace(model) || len(model) > 200 || strings.ContainsAny(model, "*\r\n\t ") {
			return fmt.Errorf("%s requires exact, non-empty model IDs without wildcards or whitespace", openAIDisabledModelsKey)
		}
	}
	return nil
}
