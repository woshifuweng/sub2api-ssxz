package service

import "strings"

// resolveOpenAIForwardModel determines the upstream model for OpenAI-compatible
// forwarding. Group-level default mapping only applies when the account itself
// did not match any explicit model_mapping rule.
func resolveOpenAIForwardModel(account *Account, requestedModel, defaultMappedModel string) string {
	mappedModel, _ := resolveOpenAIForwardModelWithMatch(account, requestedModel, defaultMappedModel)
	return mappedModel
}

// resolveOpenAIForwardModelWithMatch also reports whether account model_mapping
// selected the upstream model. Explicit mappings must survive generic Codex
// model normalization, including identity mappings.
func resolveOpenAIForwardModelWithMatch(account *Account, requestedModel, defaultMappedModel string) (string, bool) {
	if account == nil {
		if defaultMappedModel != "" {
			return defaultMappedModel, false
		}
		return requestedModel, false
	}

	mappedModel, matched := account.ResolveMappedModel(requestedModel)
	if !matched && defaultMappedModel != "" {
		return defaultMappedModel, false
	}
	return mappedModel, matched
}

var openAIOAuthForeignModelPrefixes = []string{
	"deepseek-", "glm-", "kimi-", "moonshot-", "qwen-", "qwen2-", "qwen3-", "qwen4-", "qwq-",
	"minimax-", "gemini-", "gemma-", "grok-", "doubao-", "hunyuan-", "llama-", "llama2-", "llama3-",
	"meta-llama", "mistral-", "mixtral-", "baichuan-", "ernie-", "step-", "seed-", "yi-",
}

// isOpenAIOAuthServableModel rejects model families that Codex upstreams
// cannot serve while retaining custom aliases for channel-level mapping.
func isOpenAIOAuthServableModel(requestedModel string) bool {
	model := strings.ToLower(lastOpenAIModelSegment(requestedModel))
	if model == "" {
		return true
	}
	for _, prefix := range openAIOAuthForeignModelPrefixes {
		if strings.HasPrefix(model, prefix) {
			return false
		}
	}
	return true
}

// resolveOpenAICompactForwardModel applies only compact-specific mappings.
func resolveOpenAICompactForwardModel(account *Account, model string) string {
	trimmedModel := strings.TrimSpace(model)
	if trimmedModel == "" || account == nil {
		return trimmedModel
	}

	mappedModel, matched := account.ResolveCompactMappedModel(trimmedModel)
	if !matched {
		return trimmedModel
	}
	if trimmedMapped := strings.TrimSpace(mappedModel); trimmedMapped != "" {
		return trimmedMapped
	}
	return trimmedModel
}
