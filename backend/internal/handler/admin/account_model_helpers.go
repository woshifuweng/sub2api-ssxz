package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	"github.com/Wei-Shaw/sub2api/internal/pkg/geminicli"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
)

func mappingKeys(mapping map[string]string) []string {
	if len(mapping) == 0 {
		return nil
	}
	keys := make([]string, 0, len(mapping))
	for key := range mapping {
		keys = append(keys, key)
	}
	return keys
}

func buildOpenAIModelsFromIDs(ids []string) []openai.Model {
	defaults := make(map[string]openai.Model, len(openai.DefaultModels))
	for _, model := range openai.DefaultModels {
		defaults[model.ID] = model
	}
	models := make([]openai.Model, 0, len(ids))
	for _, id := range ids {
		if model, ok := defaults[id]; ok {
			models = append(models, model)
		} else {
			models = append(models, openai.Model{ID: id, Object: "model", Type: "model", DisplayName: id})
		}
	}
	return models
}

func buildGeminiModelsFromIDs(ids []string) []geminicli.Model {
	defaults := make(map[string]geminicli.Model, len(geminicli.DefaultModels))
	for _, model := range geminicli.DefaultModels {
		defaults[model.ID] = model
	}
	models := make([]geminicli.Model, 0, len(ids))
	for _, id := range ids {
		if model, ok := defaults[id]; ok {
			models = append(models, model)
		} else {
			models = append(models, geminicli.Model{ID: id, Type: "model", DisplayName: id})
		}
	}
	return models
}

func buildClaudeModelsFromIDs(ids []string) []claude.Model {
	defaults := make(map[string]claude.Model, len(claude.DefaultModels))
	for _, model := range claude.DefaultModels {
		defaults[model.ID] = model
	}
	models := make([]claude.Model, 0, len(ids))
	for _, id := range ids {
		if model, ok := defaults[id]; ok {
			models = append(models, model)
		} else {
			models = append(models, claude.Model{ID: id, Type: "model", DisplayName: id})
		}
	}
	return models
}
