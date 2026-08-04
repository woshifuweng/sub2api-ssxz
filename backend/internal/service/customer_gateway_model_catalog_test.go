package service

import (
	"context"
	"testing"
	"time"

	gocache "github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/require"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

func TestCustomerGatewayModelCatalog_ContainsOnlyApprovedModels(t *testing.T) {
	require.Equal(t, []string{
		"gpt-5.6-sol",
		"gpt-5.6-terra",
		"gpt-5.5",
		"gpt-5.4",
		"gpt-5.4-mini",
	}, CustomerGatewayModelIDs(PlatformOpenAI))

	require.Equal(t, []string{
		"claude-fable-5",
		"claude-opus-4-6",
		"claude-opus-4-7",
		"claude-opus-4-8",
		"claude-sonnet-4-6",
		"claude-sonnet-5",
	}, CustomerGatewayModelIDs(PlatformAnthropic))
}

func TestCanonicalCustomerGatewayModel_NormalizesOnlyApprovedAliases(t *testing.T) {
	model, ok := CanonicalCustomerGatewayModel(PlatformOpenAI, "gpt-5.6")
	require.True(t, ok)
	require.Equal(t, "gpt-5.6-sol", model)

	for _, blocked := range []struct {
		platform string
		model    string
	}{
		{platform: PlatformOpenAI, model: "gpt-5.3-codex-spark"},
		{platform: PlatformOpenAI, model: "gpt-5.3-codex-spark-high"},
		{platform: PlatformOpenAI, model: "gpt-5.6-luna"},
		{platform: PlatformAnthropic, model: "haiku"},
		{platform: PlatformAnthropic, model: "claude-haiku-4-5"},
		{platform: PlatformAnthropic, model: "claude-3-5-haiku"},
	} {
		t.Run(blocked.model, func(t *testing.T) {
			_, allowed := CanonicalCustomerGatewayModel(blocked.platform, blocked.model)
			require.False(t, allowed)
		})
	}
}

// TestCustomerGatewayModelCatalog_AllVisibleModelsHaveExactPricing 守卫「对客可见模型
// 必须在 fallback 层有精确官方价」。动态 LiteLLM 数据不可用时（首次启动、同步失败、
// 上游删条目），计费会掉到 fallback；此时任何按型号系列瞎猜的价格都是真金白银的误差。
//
// 期望价取自官方价目表，与生产动态数据一致。新增对客模型必须同步补 fallback 条目，
// 否则本测试会以 "no expected price" 失败。
func TestCustomerGatewayModelCatalog_AllVisibleModelsHaveExactPricing(t *testing.T) {
	beforeSonnet5Standard := time.Date(2026, time.August, 31, 23, 59, 59, 0, time.UTC)
	afterSonnet5Standard := time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)

	type expectedPrice struct {
		input  float64
		output float64
	}
	// model -> 时刻 -> 期望价（USD/token）
	expected := map[string]map[time.Time]expectedPrice{
		"claude-fable-5": {
			beforeSonnet5Standard: {input: 10e-6, output: 50e-6},
			afterSonnet5Standard:  {input: 10e-6, output: 50e-6},
		},
		"claude-opus-4-6": {
			beforeSonnet5Standard: {input: 5e-6, output: 25e-6},
			afterSonnet5Standard:  {input: 5e-6, output: 25e-6},
		},
		"claude-opus-4-7": {
			beforeSonnet5Standard: {input: 5e-6, output: 25e-6},
			afterSonnet5Standard:  {input: 5e-6, output: 25e-6},
		},
		"claude-opus-4-8": {
			beforeSonnet5Standard: {input: 5e-6, output: 25e-6},
			afterSonnet5Standard:  {input: 5e-6, output: 25e-6},
		},
		"claude-sonnet-4-6": {
			beforeSonnet5Standard: {input: 3e-6, output: 15e-6},
			afterSonnet5Standard:  {input: 3e-6, output: 15e-6},
		},
		// introductory $2/$10 → 2026-09-01 UTC 起标准 $3/$15
		"claude-sonnet-5": {
			beforeSonnet5Standard: {input: 2e-6, output: 10e-6},
			afterSonnet5Standard:  {input: 3e-6, output: 15e-6},
		},
		"gpt-5.6-sol": {
			beforeSonnet5Standard: {input: 5e-6, output: 30e-6},
			afterSonnet5Standard:  {input: 5e-6, output: 30e-6},
		},
		"gpt-5.6-terra": {
			beforeSonnet5Standard: {input: 2e-6, output: 12e-6},
			afterSonnet5Standard:  {input: 2e-6, output: 12e-6},
		},
		"gpt-5.5": {
			beforeSonnet5Standard: {input: 5e-6, output: 30e-6},
			afterSonnet5Standard:  {input: 5e-6, output: 30e-6},
		},
		"gpt-5.4": {
			beforeSonnet5Standard: {input: 2.5e-6, output: 15e-6},
			afterSonnet5Standard:  {input: 2.5e-6, output: 15e-6},
		},
		"gpt-5.4-mini": {
			beforeSonnet5Standard: {input: 0.75e-6, output: 4.5e-6},
			afterSonnet5Standard:  {input: 0.75e-6, output: 4.5e-6},
		},
	}

	// nil pricing service → 强制走纯 fallback 路径
	svc := NewBillingService(&config.Config{}, nil)

	for _, platform := range []string{PlatformOpenAI, PlatformAnthropic} {
		for _, model := range CustomerGatewayModelIDs(platform) {
			perTime, ok := expected[model]
			require.True(t, ok, "%s (%s) is customer-visible but has no expected fallback price in this test; add its fallback entry in billing_service.go and the expectation here", model, platform)

			for _, at := range []time.Time{beforeSonnet5Standard, afterSonnet5Standard} {
				want := perTime[at]
				got := svc.getFallbackPricingAt(model, at)
				require.NotNil(t, got, "%s has no fallback price at %s", model, at)
				require.InDelta(t, want.input, got.InputPricePerToken, 1e-12, "%s input price at %s", model, at)
				require.InDelta(t, want.output, got.OutputPricePerToken, 1e-12, "%s output price at %s", model, at)
			}
		}
	}
}

func TestGetAvailableModels_GroupCatalogUsesModelIntersectionNotAccountPlatform(t *testing.T) {
	groupID := int64(11)
	repo := &modelsListAccountRepoStub{byGroup: map[int64][]Account{
		groupID: {
			{
				ID:       32,
				Platform: PlatformOpenAI,
				Credentials: map[string]any{"model_mapping": map[string]any{
					"claude-fable-5":      "claude-fable-5",
					"claude-opus-4-6":     "claude-opus-4-6",
					"claude-opus-4-7":     "claude-opus-4-7",
					"claude-opus-4-8":     "claude-opus-4-8",
					"claude-sonnet-4-6":   "claude-sonnet-4-6",
					"claude-sonnet-5":     "claude-sonnet-5",
					"claude-haiku-4-5":    "claude-haiku-4-5",
					"gpt-5.3-codex-spark": "gpt-5.3-codex-spark",
				}},
			},
		},
	}}
	svc := &GatewayService{
		accountRepo:        repo,
		modelsListCache:    gocache.New(time.Minute, time.Minute),
		modelsListCacheTTL: time.Minute,
	}

	require.Equal(t, []string{
		"claude-fable-5",
		"claude-opus-4-6",
		"claude-opus-4-7",
		"claude-opus-4-8",
		"claude-sonnet-4-6",
		"claude-sonnet-5",
	}, svc.GetAvailableModels(context.Background(), &groupID, PlatformAnthropic))
}

func TestGetAvailableModels_OpenAICatalogHidesLegacyAndBlockedModels(t *testing.T) {
	groupID := int64(10)
	repo := &modelsListAccountRepoStub{byGroup: map[int64][]Account{
		groupID: {
			{
				ID:       30,
				Platform: PlatformOpenAI,
				Extra: map[string]any{AccountExtraFetchedModelsKey: []any{
					"gpt-5.6",
					"gpt-5.6-sol",
					"gpt-5.6-terra",
					"gpt-5.6-luna",
					"gpt-5.5",
					"gpt-5.4",
					"gpt-5.4-mini",
					"gpt-5.4-nano",
					"gpt-5.3-codex-spark",
				}},
			},
		},
	}}
	svc := &GatewayService{accountRepo: repo}

	require.Equal(t, []string{
		"gpt-5.4",
		"gpt-5.4-mini",
		"gpt-5.5",
		"gpt-5.6-sol",
		"gpt-5.6-terra",
	}, svc.GetAvailableModels(context.Background(), &groupID, PlatformOpenAI))
}
