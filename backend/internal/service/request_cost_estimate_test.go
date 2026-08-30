package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestEstimateGatewayTokenRequestCostUsesSafetyBudgetWithoutCap(t *testing.T) {
	cfg := &config.Config{}
	cfg.Default.RateMultiplier = 1
	svc := &GatewayService{billingService: NewBillingService(cfg, nil), cfg: cfg}
	parsed, err := ParseGatewayRequest(NewRequestBodyRef([]byte(`{"model":"claude-sonnet-4","messages":[{"role":"user","content":"hello"}]}`)), domain.PlatformAnthropic)
	require.NoError(t, err)

	cost, err := svc.EstimateGatewayTokenRequestCost(context.Background(), parsed, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, cost)
	require.GreaterOrEqual(t, cost.ActualCost, unboundedTokenRequestMinimumSafetyCost)
}

func TestEnforceUnboundedTokenRequestLimit(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		wantChanged bool
		wantLimit   int64
	}{
		{
			name:        "injects server cap when omitted",
			body:        `{"model":"gpt-5.5","input":"hello"}`,
			wantChanged: true,
			wantLimit:   unboundedTokenRequestMaxOutputTokens,
		},
		{
			name:        "preserves explicit client cap",
			body:        `{"model":"gpt-5.5","input":"hello","max_output_tokens":2048}`,
			wantChanged: false,
			wantLimit:   2048,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, changed, err := EnforceUnboundedTokenRequestLimit(
				[]byte(tt.body),
				"max_output_tokens",
				"max_output_tokens",
			)
			require.NoError(t, err)
			require.Equal(t, tt.wantChanged, changed)
			require.Equal(t, tt.wantLimit, gjson.GetBytes(got, "max_output_tokens").Int())
		})
	}
}

func TestEstimateOpenAITokenRequestCostUsesServerCapPricingWithoutCap(t *testing.T) {
	cfg := &config.Config{}
	cfg.Default.RateMultiplier = 1
	svc := &OpenAIGatewayService{billingService: NewBillingService(cfg, nil), cfg: cfg}
	groupID := int64(305)
	apiKey := &APIKey{
		UserID:  1,
		GroupID: &groupID,
		Group: &Group{
			ID:             groupID,
			RateMultiplier: 0.35,
		},
	}
	user := &User{ID: 1}

	noLimitCost, err := svc.EstimateOpenAITokenRequestCost(
		context.Background(),
		"gpt-5.5",
		[]byte(`{"model":"gpt-5.5","input":"hello"}`),
		apiKey,
		user,
	)
	require.NoError(t, err)
	require.NotNil(t, noLimitCost)

	explicitServerCapCost, err := svc.EstimateOpenAITokenRequestCost(
		context.Background(),
		"gpt-5.5",
		[]byte(`{"model":"gpt-5.5","input":"hello","max_output_tokens":16384}`),
		apiKey,
		user,
	)
	require.NoError(t, err)
	require.NotNil(t, explicitServerCapCost)
	require.InDelta(t, explicitServerCapCost.ActualCost, noLimitCost.ActualCost, 0.001)
	require.Less(t, noLimitCost.ActualCost, 3.0)

	oversizedCost, err := svc.EstimateOpenAITokenRequestCost(
		context.Background(),
		"gpt-5.5",
		[]byte(`{"model":"gpt-5.5","input":"hello","max_output_tokens":500000}`),
		apiKey,
		user,
	)
	require.NoError(t, err)
	require.NotNil(t, oversizedCost)
	require.Greater(t, oversizedCost.ActualCost, 3.0)
}

func TestEstimateOpenAIImageCostUsesResolvedGroupPrice(t *testing.T) {
	cfg := &config.Config{}
	cfg.Default.RateMultiplier = 1
	price := 2.0
	groupID := int64(7)
	group := &Group{ID: groupID, RateMultiplier: 1, ImagePrice1K: &price}
	apiKey := &APIKey{GroupID: &groupID, Group: group}
	user := &User{ID: 9}
	svc := &OpenAIGatewayService{billingService: NewBillingService(cfg, nil), cfg: cfg}

	cost := svc.EstimateOpenAIImageCost(context.Background(), "gpt-image-2", "1K", 1, apiKey, user)

	require.NotNil(t, cost)
	require.InDelta(t, 2, cost.ActualCost, 0.000001)
}
