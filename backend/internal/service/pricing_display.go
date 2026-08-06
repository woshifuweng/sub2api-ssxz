package service

// SynthesizePricingFromBilling adapts the billing service's canonical model
// pricing to the channel display DTO. It intentionally reuses the existing
// LiteLLM-to-channel conversion so display pricing keeps the same shape as
// the normal available-channel fallback without writing channel overrides.
func SynthesizePricingFromBilling(pricing *ModelPricing) *ChannelModelPricing {
	if pricing == nil {
		return nil
	}
	return synthesizePricingFromLiteLLM(&LiteLLMModelPricing{
		InputCostPerToken:           pricing.InputPricePerToken,
		OutputCostPerToken:          pricing.OutputPricePerToken,
		CacheCreationInputTokenCost: pricing.CacheCreationPricePerToken,
		CacheReadInputTokenCost:     pricing.CacheReadPricePerToken,
		OutputCostPerImageToken:     pricing.ImageOutputPricePerToken,
	})
}
