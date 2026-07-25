package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSynthesizePricingFromLiteLLMIncludesContextWindow(t *testing.T) {
	maxInput := 200000
	maxOutput := 128000

	pricing := synthesizePricingFromLiteLLM(&LiteLLMModelPricing{
		InputCostPerToken: 5e-6,
		MaxInputTokens:    &maxInput,
		MaxOutputTokens:   &maxOutput,
	})

	require.NotNil(t, pricing)
	require.Equal(t, &maxInput, pricing.ContextLength)
	require.Equal(t, &maxOutput, pricing.MaxOutputTokens)
}

func TestFillGlobalPricingFallbackEnrichesExistingChannelPricing(t *testing.T) {
	maxInput := 200000
	maxOutput := 128000
	channelInputPrice := 6e-6
	pricingService := &PricingService{pricingData: map[string]*LiteLLMModelPricing{
		"claude-opus-4-8": {
			InputCostPerToken: 5e-6,
			MaxInputTokens:    &maxInput,
			MaxOutputTokens:   &maxOutput,
		},
	}}
	channelService := &ChannelService{pricingService: pricingService}
	models := []SupportedModel{{
		Name:     "claude-opus-4-8",
		Platform: "anthropic",
		Pricing: &ChannelModelPricing{
			BillingMode: BillingModeToken,
			InputPrice:  &channelInputPrice,
		},
	}}

	channelService.fillGlobalPricingFallback(models)

	require.Equal(t, &channelInputPrice, models[0].Pricing.InputPrice)
	require.Equal(t, &maxInput, models[0].Pricing.ContextLength)
	require.Equal(t, &maxOutput, models[0].Pricing.MaxOutputTokens)
}
