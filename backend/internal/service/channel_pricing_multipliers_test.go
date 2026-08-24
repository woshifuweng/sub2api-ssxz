package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func pricingMultiplier(value float64) *float64 { return &value }

func TestChannelOverridePreservesCatalogFastRatioByDefault(t *testing.T) {
	pricing := &ModelPricing{
		InputPricePerToken:          2,
		InputPricePerTokenPriority:  4,
		OutputPricePerToken:         6,
		OutputPricePerTokenPriority: 12,
	}
	applyChannelTokenPriceOverrides(pricing, &ChannelModelPricing{
		InputPrice:  pricingMultiplier(3),
		OutputPrice: pricingMultiplier(9),
	})

	require.InDelta(t, 3, pricing.InputPricePerToken, 1e-12)
	require.InDelta(t, 6, pricing.InputPricePerTokenPriority, 1e-12)
	require.InDelta(t, 9, pricing.OutputPricePerToken, 1e-12)
	require.InDelta(t, 18, pricing.OutputPricePerTokenPriority, 1e-12)
}

func TestAnthropicFastUsesDefaultMultiplierWithoutCatalogTier(t *testing.T) {
	pricing := &ModelPricing{InputPricePerToken: 5e-6, OutputPricePerToken: 25e-6}
	cost := (&BillingService{}).computeTokenBreakdown(pricing, UsageTokens{
		InputTokens: 1_000_000, OutputTokens: 1_000_000,
	}, 1, "fast", false)

	require.InDelta(t, 10, cost.InputCost, 1e-12)
	require.InDelta(t, 50, cost.OutputCost, 1e-12)
}

func TestBuiltInModelFastDefaults(t *testing.T) {
	service := &BillingService{fallbackPrices: make(map[string]*ModelPricing)}
	service.initFallbackPricing()

	for _, tt := range []struct {
		model string
		want  float64
	}{
		{model: "gpt-5.5", want: 2.5},
		{model: "claude-opus-4.8", want: 2},
		{model: "claude-opus-5", want: 2},
	} {
		pricing := service.fallbackPrices[tt.model]
		require.NotNil(t, pricing)
		require.InDelta(t, tt.want, pricing.InputPricePerTokenPriority/pricing.InputPricePerToken, 1e-12)
		require.InDelta(t, tt.want, pricing.OutputPricePerTokenPriority/pricing.OutputPricePerToken, 1e-12)
	}
}

func TestIntervalPricePreservesDefaultFastRatio(t *testing.T) {
	pricing := intervalToModelPricing(&PricingInterval{
		InputPrice: pricingMultiplier(7),
	}, &ModelPricing{
		InputPricePerToken:         5,
		InputPricePerTokenPriority: 10,
	}, nil)

	require.InDelta(t, 7, pricing.InputPricePerToken, 1e-12)
	require.InDelta(t, 14, pricing.InputPricePerTokenPriority, 1e-12)
}

func TestAnthropicSpeedServiceTier(t *testing.T) {
	account := &Account{Platform: PlatformAnthropic}

	for _, model := range []string{"claude-opus-5", "claude-opus-4-8", "claude-opus-4.8"} {
		tier := anthropicSpeedServiceTier(account, "fast", model)
		require.NotNil(t, tier, "model %s should bill as fast", model)
		require.Equal(t, "fast", *tier)
	}

	require.Nil(t, anthropicSpeedServiceTier(&Account{Platform: PlatformOpenAI}, "fast", "claude-opus-5"))
	require.Nil(t, anthropicSpeedServiceTier(account, "standard", "claude-opus-5"))
}

// fast mode 不存在于这些模型/承载上，即便客户端传了 speed=fast 也不能计 2x。
func TestAnthropicSpeedServiceTierRejectsUnsupportedTargets(t *testing.T) {
	account := &Account{Platform: PlatformAnthropic}

	for _, model := range []string{
		"claude-opus-4-7",  // fast mode 已被移除
		"claude-opus-4-6",  //
		"claude-opus-4-5",  // 不能被 "opus-5" 规则误判
		"claude-sonnet-5",  // 非 Opus
		"claude-haiku-4-5", //
		"",                 //
	} {
		require.Nil(t, anthropicSpeedServiceTier(account, "fast", model),
			"model %q must not bill as fast", model)
	}

	bedrock := &Account{Platform: PlatformAnthropic, Type: AccountTypeBedrock}
	require.Nil(t, anthropicSpeedServiceTier(bedrock, "fast", "claude-opus-5"))
}

func TestAnthropicSpeedModelPrefersMappedUpstreamModel(t *testing.T) {
	parsed := &ParsedRequest{Model: "claude-opus-5"}
	require.Equal(t, "claude-opus-4-7", anthropicSpeedModel(parsed, &ForwardResult{
		UpstreamModel: "claude-opus-4-7",
	}))
	require.Equal(t, "claude-opus-5", anthropicSpeedModel(parsed, &ForwardResult{}))
}

func TestCalculateTokenCostContextTierEnablement(t *testing.T) {
	base := &ModelPricing{InputPricePerToken: 1e-6}
	resolved := &ResolvedPricing{
		BasePricing: base,
		Intervals: []PricingInterval{{
			MinTokens:  100,
			InputPrice: pricingMultiplier(2e-6),
		}},
	}
	resolver := &ModelPricingResolver{}
	service := &BillingService{}
	tokens := UsageTokens{InputTokens: 200}

	t.Run("group disabled uses base tier", func(t *testing.T) {
		resolved.longContextPricingEnabled = false
		cost, err := service.calculateTokenCost(resolved, CostInput{
			Model: "custom", Tokens: tokens, RateMultiplier: 1, Resolver: resolver,
		})
		require.NoError(t, err)
		require.InDelta(t, 200e-6, cost.TotalCost, 1e-12)
	})

	t.Run("group enabled uses interval", func(t *testing.T) {
		resolved.longContextPricingEnabled = true
		accountDisabled := false
		cost, err := service.calculateTokenCost(resolved, CostInput{
			Model: "custom", Tokens: tokens, RateMultiplier: 1, Resolver: resolver,
			LongContextBillingEnabled: &accountDisabled,
		})
		require.NoError(t, err)
		require.InDelta(t, 400e-6, cost.TotalCost, 1e-12)
	})

	t.Run("account enabled overrides disabled group", func(t *testing.T) {
		resolved.longContextPricingEnabled = false
		accountEnabled := true
		cost, err := service.calculateTokenCost(resolved, CostInput{
			Model: "custom", Tokens: tokens, RateMultiplier: 1, Resolver: resolver,
			LongContextBillingEnabled: &accountEnabled,
		})
		require.NoError(t, err)
		require.InDelta(t, 400e-6, cost.TotalCost, 1e-12)
	})
}
