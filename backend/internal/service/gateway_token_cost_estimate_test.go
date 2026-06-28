package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestGatewayServiceEstimateGatewayTokenRequestCost(t *testing.T) {
	svc := &GatewayService{
		cfg:            &config.Config{},
		billingService: NewBillingService(&config.Config{}, nil),
	}

	parsed, err := ParseGatewayRequest(
		[]byte(`{"model":"claude-sonnet-4","messages":[{"role":"user","content":"hello"}],"max_tokens":200000}`),
		domain.PlatformAnthropic,
	)
	require.NoError(t, err)

	cost, err := svc.EstimateGatewayTokenRequestCost(context.Background(), parsed, &APIKey{}, &User{ID: 1})
	require.NoError(t, err)
	require.NotNil(t, cost)
	require.Greater(t, cost.ActualCost, 2.0)

	parsedWithoutLimit, err := ParseGatewayRequest(
		[]byte(`{"model":"claude-sonnet-4","messages":[{"role":"user","content":"hello"}]}`),
		domain.PlatformAnthropic,
	)
	require.NoError(t, err)

	noLimitCost, err := svc.EstimateGatewayTokenRequestCost(context.Background(), parsedWithoutLimit, &APIKey{}, &User{ID: 1})
	require.NoError(t, err)
	require.Nil(t, noLimitCost)

	geminiParsed, err := ParseGatewayRequest(
		[]byte(`{"contents":[{"role":"user","parts":[{"text":"hello"}]}],"generationConfig":{"maxOutputTokens":200000}}`),
		domain.PlatformGemini,
	)
	require.NoError(t, err)
	geminiParsed.Model = "gemini-3.1-pro"

	geminiCost, err := svc.EstimateGatewayTokenRequestCost(context.Background(), geminiParsed, &APIKey{}, &User{ID: 1})
	require.NoError(t, err)
	require.NotNil(t, geminiCost)
	require.Greater(t, geminiCost.ActualCost, 2.0)
}
