package service

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGetModelPricing_OfficialExactPricesOverrideRemoteData(t *testing.T) {
	maxInput := 400000
	maxOutput := 128000
	svc := &PricingService{
		pricingData: map[string]*LiteLLMModelPricing{
			"claude-opus-4-8": {
				InputCostPerToken:  99,
				OutputCostPerToken: 99,
				MaxInputTokens:     &maxInput,
				MaxOutputTokens:    &maxOutput,
			},
			"gpt-5.5": {InputCostPerToken: 99, OutputCostPerToken: 99},
			"gpt-5.6": {InputCostPerToken: 99, OutputCostPerToken: 99},
		},
	}

	tests := []struct {
		model       string
		inputPrice  float64
		outputPrice float64
	}{
		{model: "claude-fable-5", inputPrice: 10e-6, outputPrice: 50e-6},
		{model: "claude-opus-4-5", inputPrice: 5e-6, outputPrice: 25e-6},
		{model: "claude-opus-4-6", inputPrice: 5e-6, outputPrice: 25e-6},
		{model: "claude-opus-4-7", inputPrice: 5e-6, outputPrice: 25e-6},
		{model: "claude-opus-4-8", inputPrice: 5e-6, outputPrice: 25e-6},
		{model: "claude-sonnet-4-6", inputPrice: 3e-6, outputPrice: 15e-6},
		{model: "claude-sonnet-4-5", inputPrice: 3e-6, outputPrice: 15e-6},
		{model: "claude-sonnet-5", inputPrice: 2e-6, outputPrice: 10e-6},
		{model: "gpt-5.4", inputPrice: 2.5e-6, outputPrice: 15e-6},
		{model: "gpt-5.4-mini", inputPrice: 0.75e-6, outputPrice: 4.5e-6},
		{model: "gpt-5.5", inputPrice: 5e-6, outputPrice: 30e-6},
		{model: "gpt-5.6-sol", inputPrice: 5e-6, outputPrice: 30e-6},
		{model: "gpt-5.6-terra", inputPrice: 2.5e-6, outputPrice: 15e-6},
		{model: "gpt-5.6-luna", inputPrice: 1e-6, outputPrice: 6e-6},
		{model: "gpt-5.6", inputPrice: 5e-6, outputPrice: 30e-6},
		{model: "gpt-image-2", inputPrice: 5e-6, outputPrice: 0},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			pricing := svc.GetModelPricing(tt.model)
			require.NotNil(t, pricing)
			require.InDelta(t, tt.inputPrice, pricing.InputCostPerToken, 1e-12)
			require.InDelta(t, tt.outputPrice, pricing.OutputCostPerToken, 1e-12)
		})
	}

	pricing := svc.GetModelPricing("claude-opus-4-8")
	require.Equal(t, &maxInput, pricing.MaxInputTokens)
	require.Equal(t, &maxOutput, pricing.MaxOutputTokens)
}

func TestGetModelPricing_OfficialExactContextAndImagePrice(t *testing.T) {
	svc := &PricingService{pricingData: map[string]*LiteLLMModelPricing{}}

	for _, model := range []string{"gpt-5.4", "gpt-5.5", "gpt-5.6-sol", "gpt-5.6-terra", "gpt-5.6-luna"} {
		pricing := svc.GetModelPricing(model)
		require.NotNil(t, pricing, model)
		require.NotNil(t, pricing.MaxInputTokens, model)
		require.Equal(t, 1050000, *pricing.MaxInputTokens, model)
		require.NotNil(t, pricing.MaxOutputTokens, model)
		require.Equal(t, 128000, *pricing.MaxOutputTokens, model)
	}

	imagePricing := svc.GetModelPricing("gpt-image-2")
	require.NotNil(t, imagePricing)
	require.InDelta(t, 30e-6, imagePricing.OutputCostPerImageToken, 1e-12)
	require.Equal(t, "image_generation", imagePricing.Mode)
}

func TestGetOfficialExactModelPricing_ClaudeSonnet5SwitchesAtOfficialBoundary(t *testing.T) {
	before := getOfficialExactModelPricingAt("claude-sonnet-5", time.Date(2026, time.August, 31, 23, 59, 59, 0, time.UTC))
	require.NotNil(t, before)
	require.InDelta(t, 2e-6, before.InputCostPerToken, 1e-12)
	require.InDelta(t, 10e-6, before.OutputCostPerToken, 1e-12)
	require.InDelta(t, 2.5e-6, before.CacheCreationInputTokenCost, 1e-12)
	require.InDelta(t, 4e-6, before.CacheCreationInputTokenCostAbove1hr, 1e-12)
	require.InDelta(t, 0.2e-6, before.CacheReadInputTokenCost, 1e-12)

	after := getOfficialExactModelPricingAt("claude-sonnet-5", time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC))
	require.NotNil(t, after)
	require.InDelta(t, 3e-6, after.InputCostPerToken, 1e-12)
	require.InDelta(t, 15e-6, after.OutputCostPerToken, 1e-12)
	require.InDelta(t, 3.75e-6, after.CacheCreationInputTokenCost, 1e-12)
	require.InDelta(t, 6e-6, after.CacheCreationInputTokenCostAbove1hr, 1e-12)
	require.InDelta(t, 0.3e-6, after.CacheReadInputTokenCost, 1e-12)
}

func TestGetModelPricing_UnknownVersionedModelsRefuseGuessedPricingAndWarn(t *testing.T) {
	logSink, restore := captureStructuredLog(t)
	defer restore()

	svc := &PricingService{
		pricingData: map[string]*LiteLLMModelPricing{
			"claude-opus-4-5": {InputCostPerToken: 5e-6, OutputCostPerToken: 25e-6},
			"claude-opus-4-6": {InputCostPerToken: 5e-6, OutputCostPerToken: 25e-6},
			"gpt-5.1-codex":   {InputCostPerToken: 1.5e-6, OutputCostPerToken: 12e-6},
		},
	}

	require.Nil(t, svc.GetModelPricing("claude-opus-4-9"))
	require.Nil(t, svc.GetModelPricing("gpt-5.7"))
	require.True(t, logSink.ContainsMessageAtLevel("pricing unavailable for exact model claude-opus-4-9", "warn"))
	require.True(t, logSink.ContainsMessageAtLevel("pricing unavailable for exact model gpt-5.7", "warn"))
}

func TestMatchByModelFamily_IsDeterministicForLegacyAliases(t *testing.T) {
	newest := &LiteLLMModelPricing{InputCostPerToken: 6}
	svc := &PricingService{
		pricingData: map[string]*LiteLLMModelPricing{
			"claude-opus-4-1": {InputCostPerToken: 1},
			"claude-opus-4-5": {InputCostPerToken: 5},
			"claude-opus-4-6": newest,
		},
	}

	for range 100 {
		require.Same(t, newest, svc.matchByModelFamily("claude-opus"))
	}
}

func TestParsePricingData_ParsesPriorityAndServiceTierFields(t *testing.T) {
	svc := &PricingService{}
	body := []byte(`{
		"gpt-5.4": {
			"input_cost_per_token": 0.0000025,
			"input_cost_per_token_priority": 0.000005,
			"output_cost_per_token": 0.000015,
			"output_cost_per_token_priority": 0.00003,
			"cache_creation_input_token_cost": 0.0000025,
			"cache_read_input_token_cost": 0.00000025,
			"cache_read_input_token_cost_priority": 0.0000005,
			"supports_service_tier": true,
			"supports_prompt_caching": true,
			"litellm_provider": "openai",
			"mode": "chat"
		}
	}`)

	data, err := svc.parsePricingData(body)
	require.NoError(t, err)
	pricing := data["gpt-5.4"]
	require.NotNil(t, pricing)
	require.InDelta(t, 5e-6, pricing.InputCostPerTokenPriority, 1e-12)
	require.InDelta(t, 3e-5, pricing.OutputCostPerTokenPriority, 1e-12)
	require.InDelta(t, 5e-7, pricing.CacheReadInputTokenCostPriority, 1e-12)
	require.True(t, pricing.SupportsServiceTier)
}

func TestGetModelPricing_Gpt53CodexSparkRefusesAllProxyPricing(t *testing.T) {
	svc := &PricingService{
		pricingData: map[string]*LiteLLMModelPricing{
			"gpt-5.3-codex-spark": {InputCostPerToken: 99, OutputCostPerToken: 99},
			"gpt-5.3-codex":       {InputCostPerToken: 1.75e-6, OutputCostPerToken: 14e-6},
			"gpt-5.1-codex":       {InputCostPerToken: 1.5e-6, OutputCostPerToken: 12e-6},
		},
	}

	require.Nil(t, svc.GetModelPricing("gpt-5.3-codex-spark"))
	require.Nil(t, svc.GetModelPricing("gpt-5.3-codex-spark-high"))
}

func TestGetModelPricing_Gpt53CodexFallbackStillUsesGpt52Codex(t *testing.T) {
	gpt52CodexPricing := &LiteLLMModelPricing{InputCostPerToken: 2}

	svc := &PricingService{
		pricingData: map[string]*LiteLLMModelPricing{
			"gpt-5.2-codex": gpt52CodexPricing,
		},
	}

	got := svc.GetModelPricing("gpt-5.3-codex")
	require.Same(t, gpt52CodexPricing, got)
}

func TestGetModelPricing_OpenAIFallbackMatchedLoggedAsInfo(t *testing.T) {
	logSink, restore := captureStructuredLog(t)
	defer restore()

	gpt52CodexPricing := &LiteLLMModelPricing{InputCostPerToken: 2}
	svc := &PricingService{
		pricingData: map[string]*LiteLLMModelPricing{
			"gpt-5.2-codex": gpt52CodexPricing,
		},
	}

	got := svc.GetModelPricing("gpt-5.3-codex")
	require.Same(t, gpt52CodexPricing, got)

	require.True(t, logSink.ContainsMessageAtLevel("[Pricing] OpenAI fallback matched gpt-5.3-codex -> gpt-5.2-codex", "info"))
	require.False(t, logSink.ContainsMessageAtLevel("[Pricing] OpenAI fallback matched gpt-5.3-codex -> gpt-5.2-codex", "warn"))
}

func TestGetModelPricing_Gpt54UsesStaticFallbackWhenRemoteMissing(t *testing.T) {
	svc := &PricingService{
		pricingData: map[string]*LiteLLMModelPricing{
			"gpt-5.1-codex": &LiteLLMModelPricing{InputCostPerToken: 1.25e-6},
		},
	}

	got := svc.GetModelPricing("gpt-5.4")
	require.NotNil(t, got)
	require.InDelta(t, 2.5e-6, got.InputCostPerToken, 1e-12)
	require.InDelta(t, 1.5e-5, got.OutputCostPerToken, 1e-12)
	require.InDelta(t, 2.5e-7, got.CacheReadInputTokenCost, 1e-12)
	require.Equal(t, 272000, got.LongContextInputTokenThreshold)
	require.InDelta(t, 2.0, got.LongContextInputCostMultiplier, 1e-12)
	require.InDelta(t, 1.5, got.LongContextOutputCostMultiplier, 1e-12)
}

func TestGetModelPricing_Gpt54MiniUsesDedicatedStaticFallbackWhenRemoteMissing(t *testing.T) {
	svc := &PricingService{
		pricingData: map[string]*LiteLLMModelPricing{
			"gpt-5.1-codex": {InputCostPerToken: 1.25e-6},
		},
	}

	got := svc.GetModelPricing("gpt-5.4-mini")
	require.NotNil(t, got)
	require.InDelta(t, 7.5e-7, got.InputCostPerToken, 1e-12)
	require.InDelta(t, 4.5e-6, got.OutputCostPerToken, 1e-12)
	require.InDelta(t, 7.5e-8, got.CacheReadInputTokenCost, 1e-12)
	require.Zero(t, got.LongContextInputTokenThreshold)
}

func TestGetModelPricing_Gpt54NanoUsesDedicatedStaticFallbackWhenRemoteMissing(t *testing.T) {
	svc := &PricingService{
		pricingData: map[string]*LiteLLMModelPricing{
			"gpt-5.1-codex": {InputCostPerToken: 1.25e-6},
		},
	}

	got := svc.GetModelPricing("gpt-5.4-nano")
	require.NotNil(t, got)
	require.InDelta(t, 2e-7, got.InputCostPerToken, 1e-12)
	require.InDelta(t, 1.25e-6, got.OutputCostPerToken, 1e-12)
	require.InDelta(t, 2e-8, got.CacheReadInputTokenCost, 1e-12)
	require.Zero(t, got.LongContextInputTokenThreshold)
}

func TestParsePricingData_PreservesPriorityAndServiceTierFields(t *testing.T) {
	raw := map[string]any{
		"gpt-5.4": map[string]any{
			"input_cost_per_token":                 2.5e-6,
			"input_cost_per_token_priority":        5e-6,
			"output_cost_per_token":                15e-6,
			"output_cost_per_token_priority":       30e-6,
			"cache_read_input_token_cost":          0.25e-6,
			"cache_read_input_token_cost_priority": 0.5e-6,
			"supports_service_tier":                true,
			"supports_prompt_caching":              true,
			"litellm_provider":                     "openai",
			"mode":                                 "chat",
		},
	}
	body, err := json.Marshal(raw)
	require.NoError(t, err)

	svc := &PricingService{}
	pricingMap, err := svc.parsePricingData(body)
	require.NoError(t, err)

	pricing := pricingMap["gpt-5.4"]
	require.NotNil(t, pricing)
	require.InDelta(t, 2.5e-6, pricing.InputCostPerToken, 1e-12)
	require.InDelta(t, 5e-6, pricing.InputCostPerTokenPriority, 1e-12)
	require.InDelta(t, 15e-6, pricing.OutputCostPerToken, 1e-12)
	require.InDelta(t, 30e-6, pricing.OutputCostPerTokenPriority, 1e-12)
	require.InDelta(t, 0.25e-6, pricing.CacheReadInputTokenCost, 1e-12)
	require.InDelta(t, 0.5e-6, pricing.CacheReadInputTokenCostPriority, 1e-12)
	require.True(t, pricing.SupportsServiceTier)
}

func TestParsePricingData_PreservesServiceTierPriorityFields(t *testing.T) {
	svc := &PricingService{}
	pricingData, err := svc.parsePricingData([]byte(`{
		"gpt-5.4": {
			"input_cost_per_token": 0.0000025,
			"input_cost_per_token_priority": 0.000005,
			"output_cost_per_token": 0.000015,
			"output_cost_per_token_priority": 0.00003,
			"cache_read_input_token_cost": 0.00000025,
			"cache_read_input_token_cost_priority": 0.0000005,
			"supports_service_tier": true,
			"litellm_provider": "openai",
			"mode": "chat"
		}
	}`))
	require.NoError(t, err)

	pricing := pricingData["gpt-5.4"]
	require.NotNil(t, pricing)
	require.InDelta(t, 0.0000025, pricing.InputCostPerToken, 1e-12)
	require.InDelta(t, 0.000005, pricing.InputCostPerTokenPriority, 1e-12)
	require.InDelta(t, 0.000015, pricing.OutputCostPerToken, 1e-12)
	require.InDelta(t, 0.00003, pricing.OutputCostPerTokenPriority, 1e-12)
	require.InDelta(t, 0.00000025, pricing.CacheReadInputTokenCost, 1e-12)
	require.InDelta(t, 0.0000005, pricing.CacheReadInputTokenCostPriority, 1e-12)
	require.True(t, pricing.SupportsServiceTier)
}

func TestParsePricingData_PreservesContextWindowFields(t *testing.T) {
	svc := &PricingService{}
	pricingData, err := svc.parsePricingData([]byte(`{
		"claude-opus-4-8": {
			"input_cost_per_token": 0.000005,
			"output_cost_per_token": 0.000025,
			"max_input_tokens": 200000,
			"max_output_tokens": 128000,
			"litellm_provider": "anthropic",
			"mode": "chat"
		}
	}`))
	require.NoError(t, err)

	pricing := pricingData["claude-opus-4-8"]
	require.NotNil(t, pricing)
	require.NotNil(t, pricing.MaxInputTokens)
	require.NotNil(t, pricing.MaxOutputTokens)
	require.Equal(t, 200000, *pricing.MaxInputTokens)
	require.Equal(t, 128000, *pricing.MaxOutputTokens)
}
