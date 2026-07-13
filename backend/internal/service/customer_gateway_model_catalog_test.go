package service

import (
	"context"
	"testing"
	"time"

	gocache "github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/require"
)

func TestCustomerGatewayModelCatalog_ContainsOnlyApprovedModels(t *testing.T) {
	require.Equal(t, []string{
		"gpt-5.6-sol",
		"gpt-5.6-terra",
		"gpt-5.6-luna",
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

func TestCustomerGatewayModelCatalog_AllVisibleModelsHaveExactPricing(t *testing.T) {
	checkAt := []time.Time{
		time.Date(2026, time.August, 31, 23, 59, 59, 0, time.UTC),
		time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC),
	}
	for _, platform := range []string{PlatformOpenAI, PlatformAnthropic} {
		for _, model := range CustomerGatewayModelIDs(platform) {
			for _, at := range checkAt {
				require.NotNil(t, getOfficialExactModelPricingAt(model, at), "%s has no exact price at %s", model, at)
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
		"gpt-5.6-luna",
		"gpt-5.6-sol",
		"gpt-5.6-terra",
	}, svc.GetAvailableModels(context.Background(), &groupID, PlatformOpenAI))
}
