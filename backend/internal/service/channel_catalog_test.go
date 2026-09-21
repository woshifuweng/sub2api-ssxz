//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func catalogAccount(id, group int64, platform string, models ...string) Account {
	mapping := map[string]any{}
	for _, model := range models {
		mapping[model] = model
	}
	return Account{ID: id, GroupIDs: []int64{group}, Status: StatusActive, Schedulable: true,
		Platform: platform, Type: "apikey", Credentials: map[string]any{"model_mapping": mapping}}
}

func TestGroupCatalog_GroupOverridesAndOrphanGroups(t *testing.T) {
	price := func(model, platform string, value float64) ChannelModelPricing {
		return ChannelModelPricing{Platform: platform, Models: []string{model}, BillingMode: BillingModeToken,
			InputPrice: &value, OutputPrice: &value}
	}
	channels := []Channel{{ID: 1, Status: StatusActive, GroupIDs: []int64{1, 2}, RestrictModels: true,
		ModelPricing: []ChannelModelPricing{price("gpt-6-astra", PlatformOpenAI, 10e-6)}}}
	groups := []Group{
		{ID: 1, Name: "A", Status: StatusActive, Platform: PlatformOpenAI, RateMultiplier: .35,
			ModelPricing: []ChannelModelPricing{price("gpt-6-astra", PlatformOpenAI, 2e-6)}},
		{ID: 2, Name: "B", Status: StatusActive, Platform: PlatformOpenAI, RateMultiplier: .25},
		{ID: 3, Name: "C", Status: StatusActive, Platform: PlatformZhipu,
			ModelPricing: []ChannelModelPricing{price("glm-5.3-flash", PlatformZhipu, .8e-6)}},
	}
	accounts := []Account{catalogAccount(1, 1, PlatformOpenAI, "gpt-6-astra"),
		catalogAccount(2, 2, PlatformOpenAI, "gpt-6-astra"), catalogAccount(3, 3, PlatformZhipu, "glm-5.3-flash")}
	out := buildGroupCatalog(context.Background(), channels, groups, accounts, NewBillingService(nil, nil))
	require.Len(t, out, 3)
	require.InDelta(t, 2e-6, *out[0].SupportedModels[0].Pricing.InputPrice, 1e-12)
	require.InDelta(t, 10e-6, *out[1].SupportedModels[0].Pricing.InputPrice, 1e-12)
	require.InDelta(t, .8e-6, *out[2].SupportedModels[0].Pricing.InputPrice, 1e-12)
	require.Equal(t, .35, out[0].Groups[0].RateMultiplier)
	require.InDelta(t, 10e-6, *channels[0].ModelPricing[0].InputPrice, 1e-12, "catalog must not mutate source pricing")
}

func TestGroupCatalog_AccountAndChannelRestrictionsAndPermissionBoundary(t *testing.T) {
	p := 1e-6
	channels := []Channel{{ID: 1, Status: StatusActive, GroupIDs: []int64{1, 2}, RestrictModels: true,
		ModelPricing: []ChannelModelPricing{{Platform: PlatformAnthropic, Models: []string{"claude-fable-5-1", "claude-opus-5"}, InputPrice: &p, OutputPrice: &p}}}}
	groups := []Group{{ID: 1, Name: "allowed", Status: StatusActive, Platform: PlatformAnthropic}}
	accounts := []Account{catalogAccount(1, 1, PlatformAnthropic, "claude-opus-5", "not-priced"),
		catalogAccount(2, 2, PlatformAnthropic, "claude-fable-5-1"),
		catalogAccount(3, 1, PlatformOpenAI, "claude-fable-5-1")}
	out := buildGroupCatalog(context.Background(), channels, groups, accounts, NewBillingService(nil, nil))
	require.Len(t, out, 1)
	require.Len(t, out[0].SupportedModels, 1)
	require.Equal(t, "claude-opus-5", out[0].SupportedModels[0].Name)
	accounts[0].Schedulable = false
	require.Empty(t, buildGroupCatalog(context.Background(), channels, groups, accounts, NewBillingService(nil, nil)))
}

func TestGroupCatalog_UnknownPricingRetainedNotFree(t *testing.T) {
	groups := []Group{{ID: 1, Name: "allowed", Status: StatusActive, Platform: PlatformOpenAI}}
	accounts := []Account{catalogAccount(1, 1, PlatformOpenAI, "custom-unpriced-unique")}
	out := buildGroupCatalog(context.Background(), nil, groups, accounts, NewBillingService(nil, nil))
	require.Len(t, out, 1)
	require.Equal(t, "custom-unpriced-unique", out[0].SupportedModels[0].Name)
	require.Nil(t, out[0].SupportedModels[0].Pricing)
}

func TestGroupCatalog_ImageGroupPricesAndIndependentMultiplier(t *testing.T) {
	one, four := .15, .30
	groups := []Group{{ID: 1, Name: "image", Status: StatusActive, Platform: PlatformOpenAI,
		ImagePrice1K: &one, ImagePrice2K: &one, ImagePrice4K: &four, ImageRateIndependent: true, ImageRateMultiplier: 1}}
	accounts := []Account{catalogAccount(1, 1, PlatformOpenAI, "gpt-image-2")}
	out := buildGroupCatalog(context.Background(), nil, groups, accounts, NewBillingService(nil, nil))
	require.Len(t, out, 1)
	require.True(t, out[0].Groups[0].ImageRateIndependent)
	p := out[0].SupportedModels[0].Pricing
	require.Equal(t, BillingModeImage, p.BillingMode)
	require.Len(t, p.Intervals, 3)
	require.InDelta(t, .15, *p.Intervals[0].PerRequestPrice, 1e-12)
	require.InDelta(t, .30, *p.Intervals[2].PerRequestPrice, 1e-12)
}

func TestGroupCatalog_EmptyPriceCardDoesNotImplyFree(t *testing.T) {
	groups := []Group{{ID: 1, Name: "empty", Status: StatusActive, Platform: PlatformOpenAI}}
	channels := []Channel{{ID: 1, Status: StatusActive, GroupIDs: []int64{1}, ModelPricing: []ChannelModelPricing{
		{Platform: PlatformOpenAI, Models: []string{"unknown-unpriced-special"}, BillingMode: BillingModeToken},
	}}}
	accounts := []Account{catalogAccount(1, 1, PlatformOpenAI, "unknown-unpriced-special")}
	out := buildGroupCatalog(context.Background(), channels, groups, accounts, NewBillingService(nil, nil))
	require.Len(t, out, 1)
	require.Nil(t, out[0].SupportedModels[0].Pricing)
}

func TestGroupCatalog_RequestedAliasUsesRequestedPrice(t *testing.T) {
	p, q := 2e-6, 10e-6
	channels := []Channel{{ID: 1, Status: StatusActive, GroupIDs: []int64{1}, RestrictModels: true,
		BillingModelSource: BillingModelSourceRequested,
		ModelMapping:       map[string]map[string]string{PlatformOpenAI: {"public-alias": "gpt-6-astra"}},
		ModelPricing: []ChannelModelPricing{
			{Platform: PlatformOpenAI, Models: []string{"public-alias"}, InputPrice: &p, OutputPrice: &p},
			{Platform: PlatformOpenAI, Models: []string{"gpt-6-astra"}, InputPrice: &q, OutputPrice: &q}}}}
	groups := []Group{{ID: 1, Name: "allowed", Status: StatusActive, Platform: PlatformOpenAI}}
	accounts := []Account{catalogAccount(1, 1, PlatformOpenAI, "gpt-6-astra")}
	out := buildGroupCatalog(context.Background(), channels, groups, accounts, NewBillingService(nil, nil))
	require.Len(t, out, 1)
	for _, model := range out[0].SupportedModels {
		if model.Name == "public-alias" {
			require.InDelta(t, p, *model.Pricing.InputPrice, 1e-12)
			return
		}
	}
	t.Fatal("alias missing")
}
