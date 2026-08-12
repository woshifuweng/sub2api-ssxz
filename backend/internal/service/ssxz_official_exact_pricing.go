package service

import (
	"strings"
	"time"
)

// officialExactModelPricing pins SSXZ-only models that are not guaranteed to
// exist in the upstream LiteLLM price feed.
var officialExactModelPricing = map[string]LiteLLMModelPricing{
	"claude-fable-5": {
		InputCostPerToken: 10e-6, OutputCostPerToken: 50e-6,
		CacheCreationInputTokenCost: 12.5e-6, CacheCreationInputTokenCostAbove1hr: 20e-6,
		CacheReadInputTokenCost: 1e-6, LiteLLMProvider: "anthropic", Mode: "chat", SupportsPromptCaching: true,
	},
	"claude-opus-4-6": {
		InputCostPerToken: 5e-6, OutputCostPerToken: 25e-6,
		CacheCreationInputTokenCost: 6.25e-6, CacheCreationInputTokenCostAbove1hr: 10e-6,
		CacheReadInputTokenCost: 0.5e-6, LiteLLMProvider: "anthropic", Mode: "chat", SupportsPromptCaching: true,
	},
	"claude-opus-4-7": {
		InputCostPerToken: 5e-6, OutputCostPerToken: 25e-6,
		CacheCreationInputTokenCost: 6.25e-6, CacheCreationInputTokenCostAbove1hr: 10e-6,
		CacheReadInputTokenCost: 0.5e-6, LiteLLMProvider: "anthropic", Mode: "chat", SupportsPromptCaching: true,
	},
	"claude-opus-4-8": {
		InputCostPerToken: 5e-6, OutputCostPerToken: 25e-6,
		CacheCreationInputTokenCost: 6.25e-6, CacheCreationInputTokenCostAbove1hr: 10e-6,
		CacheReadInputTokenCost: 0.5e-6, LiteLLMProvider: "anthropic", Mode: "chat", SupportsPromptCaching: true,
	},
	"claude-sonnet-4-6": {
		InputCostPerToken: 3e-6, OutputCostPerToken: 15e-6,
		CacheCreationInputTokenCost: 3.75e-6, CacheCreationInputTokenCostAbove1hr: 6e-6,
		CacheReadInputTokenCost: 0.3e-6, LiteLLMProvider: "anthropic", Mode: "chat", SupportsPromptCaching: true,
	},
	"claude-sonnet-5": {
		InputCostPerToken: 3e-6, OutputCostPerToken: 15e-6,
		CacheCreationInputTokenCost: 3.75e-6, CacheCreationInputTokenCostAbove1hr: 6e-6,
		CacheReadInputTokenCost: 0.3e-6, LiteLLMProvider: "anthropic", Mode: "chat", SupportsPromptCaching: true,
	},
	"gpt-5.4": {
		InputCostPerToken: 2.5e-6, InputCostPerTokenPriority: 5e-6,
		OutputCostPerToken: 15e-6, OutputCostPerTokenPriority: 30e-6,
		CacheReadInputTokenCost: 0.25e-6, CacheReadInputTokenCostPriority: 0.5e-6,
		LongContextInputTokenThreshold: 272000, LongContextInputCostMultiplier: 2, LongContextOutputCostMultiplier: 1.5,
		SupportsServiceTier: true, LiteLLMProvider: "openai", Mode: "chat", SupportsPromptCaching: true,
	},
	"gpt-5.4-mini": {
		InputCostPerToken: 0.75e-6, OutputCostPerToken: 4.5e-6,
		CacheReadInputTokenCost: 0.075e-6, LiteLLMProvider: "openai", Mode: "chat", SupportsPromptCaching: true,
	},
	"gpt-5.5": {
		InputCostPerToken: 5e-6, InputCostPerTokenPriority: 12.5e-6,
		OutputCostPerToken: 30e-6, OutputCostPerTokenPriority: 75e-6,
		CacheReadInputTokenCost: 0.5e-6, CacheReadInputTokenCostPriority: 1.25e-6,
		LongContextInputTokenThreshold: 272000, LongContextInputCostMultiplier: 2, LongContextOutputCostMultiplier: 1.5,
		SupportsServiceTier: true, LiteLLMProvider: "openai", Mode: "chat", SupportsPromptCaching: true,
	},
	"gpt-5.6-sol": {
		InputCostPerToken: 5e-6, OutputCostPerToken: 30e-6,
		CacheCreationInputTokenCost: 6.25e-6, CacheReadInputTokenCost: 0.5e-6,
		LongContextInputTokenThreshold: 272000, LongContextInputCostMultiplier: 2, LongContextOutputCostMultiplier: 1.5,
		LiteLLMProvider: "openai", Mode: "chat", SupportsPromptCaching: true,
	},
	"gpt-5.6-terra": {
		InputCostPerToken: 2.5e-6, OutputCostPerToken: 15e-6,
		CacheCreationInputTokenCost: 3.125e-6, CacheReadInputTokenCost: 0.25e-6,
		LongContextInputTokenThreshold: 272000, LongContextInputCostMultiplier: 2, LongContextOutputCostMultiplier: 1.5,
		LiteLLMProvider: "openai", Mode: "chat", SupportsPromptCaching: true,
	},
	"gpt-5.6-luna": {
		InputCostPerToken: 1e-6, OutputCostPerToken: 6e-6,
		CacheCreationInputTokenCost: 1.25e-6, CacheReadInputTokenCost: 0.1e-6,
		LongContextInputTokenThreshold: 272000, LongContextInputCostMultiplier: 2, LongContextOutputCostMultiplier: 1.5,
		LiteLLMProvider: "openai", Mode: "chat", SupportsPromptCaching: true,
	},
}

var claudeSonnet5IntroPricing = LiteLLMModelPricing{
	InputCostPerToken: 2e-6, OutputCostPerToken: 10e-6,
	CacheCreationInputTokenCost: 2.5e-6, CacheCreationInputTokenCostAbove1hr: 4e-6,
	CacheReadInputTokenCost: 0.2e-6, LiteLLMProvider: "anthropic", Mode: "chat", SupportsPromptCaching: true,
}

func getOfficialExactModelPricing(model string) *LiteLLMModelPricing {
	return getOfficialExactModelPricingAt(model, time.Now().UTC())
}

func getOfficialExactModelPricingAt(model string, now time.Time) *LiteLLMModelPricing {
	model = strings.ToLower(strings.TrimSpace(model))
	if model == "gpt-5.6" {
		model = "gpt-5.6-sol"
	}
	if model == "claude-sonnet-5" && now.Before(claudeSonnet5StandardPricingStartsAt) {
		pricing := claudeSonnet5IntroPricing
		return &pricing
	}
	pricing, ok := officialExactModelPricing[model]
	if !ok {
		return nil
	}
	copy := pricing
	return &copy
}

func isUnpricedBlockedModel(model string) bool {
	model = strings.ToLower(strings.TrimSpace(model))
	model = strings.TrimPrefix(model, "models/")
	return strings.HasPrefix(model, "gpt-5.3-codex-spark")
}
