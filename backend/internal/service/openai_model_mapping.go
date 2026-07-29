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

// openAIOAuthForeignModelPrefixes lists model families that Codex upstreams
// cannot serve without an explicit account-level model mapping.
var openAIOAuthForeignModelPrefixes = []string{
	"deepseek-", "glm-", "kimi-", "moonshot-", "qwen-", "qwen2-", "qwen3-", "qwen4-", "qwq-",
	"minimax-", "gemini-", "gemma-", "grok-", "doubao-", "hunyuan-", "llama-", "llama2-", "llama3-",
	"meta-llama", "mistral-", "mixtral-", "baichuan-", "ernie-", "step-", "seed-", "yi-",
}

// isOpenAIOAuthServableModel 判断「空 model_mapping 的 OpenAI OAuth 账号」能否
// 服务请求模型。空映射默认仍是「允许」，仅排除明确属于其他厂商家族的模型
// （deepseek-*/glm-*、以及 Kimi Code 官方 bare ID k3 / k3-256k 等）——这类
// 请求原样透传必然被 Codex 上游以不可重试的 400 拒绝，且不触发 failover，
// 应在调度阶段就跳过该账号，把请求让给显式声明支持该模型的账号（#3662）。
// bare k3 仅精确匹配（取 last segment 后），不使用宽泛 k3- 前缀，以免误伤
// 自定义别名；显式 model_mapping 命中路径不经过本函数，语义不变。
func isOpenAIOAuthServableModel(requestedModel string) bool {
	model := strings.ToLower(lastOpenAIModelSegment(requestedModel))
	if model == "" {
		return true
	}
	// Kimi Code 官方 bare model ID：无厂商前缀，prefix 黑名单挡不住。
	if model == "k3" || model == "k3-256k" {
		return false
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
