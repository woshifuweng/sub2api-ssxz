package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/stretchr/testify/require"
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

func TestEstimateOpenAITokenRequestCostUsesSafetyBudgetWithoutCap(t *testing.T) {
	cfg := &config.Config{}
	cfg.Default.RateMultiplier = 1
	svc := &OpenAIGatewayService{billingService: NewBillingService(cfg, nil), cfg: cfg}

	cost, err := svc.EstimateOpenAITokenRequestCost(
		context.Background(),
		"gpt-5.4",
		[]byte(`{"model":"gpt-5.4","messages":[{"role":"user","content":"hello"}]}`),
		nil,
		nil,
	)
	require.NoError(t, err)
	require.NotNil(t, cost)
	require.GreaterOrEqual(t, cost.ActualCost, unboundedTokenRequestMinimumSafetyCost)
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
