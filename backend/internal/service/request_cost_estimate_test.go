package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestEstimateGatewayTokenRequestCostUsesForwardedServerCapWhenClientOmitsLimit(t *testing.T) {
	cfg := &config.Config{}
	cfg.Default.RateMultiplier = 1
	svc := &GatewayService{billingService: NewBillingService(cfg, nil), cfg: cfg}
	parsedWithoutLimit, err := ParseGatewayRequest(NewRequestBodyRef([]byte(`{"model":"claude-sonnet-4","messages":[{"role":"user","content":"hello"}]}`)), domain.PlatformAnthropic)
	require.NoError(t, err)
	parsedWithServerCap, err := ParseGatewayRequest(NewRequestBodyRef([]byte(`{"model":"claude-sonnet-4","messages":[{"role":"user","content":"hello"}],"max_tokens":16384}`)), domain.PlatformAnthropic)
	require.NoError(t, err)

	withoutLimitCost, err := svc.EstimateGatewayTokenRequestCost(context.Background(), parsedWithoutLimit, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, withoutLimitCost)
	serverCapCost, err := svc.EstimateGatewayTokenRequestCost(context.Background(), parsedWithServerCap, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, serverCapCost)
	// The explicit JSON is slightly longer, so input-token estimation may differ by
	// a few bytes; the output-token budget must still be the same 16,384 cap.
	require.InDelta(t, serverCapCost.ActualCost, withoutLimitCost.ActualCost, 0.001)
	require.Less(t, withoutLimitCost.ActualCost, 10.0)
}

func TestEstimateGatewayTokenRequestCostHonorsChatCompletionsLimit(t *testing.T) {
	cfg := &config.Config{}
	cfg.Default.RateMultiplier = 1
	svc := &GatewayService{billingService: NewBillingService(cfg, nil), cfg: cfg}
	explicit, err := ParseGatewayRequest(NewRequestBodyRef([]byte(`{"model":"claude-sonnet-4","messages":[],"max_completion_tokens":2048}`)), "chat_completions")
	require.NoError(t, err)
	serverCap, err := ParseGatewayRequest(NewRequestBodyRef([]byte(`{"model":"claude-sonnet-4","messages":[],"max_completion_tokens":16384}`)), "chat_completions")
	require.NoError(t, err)

	explicitCost, err := svc.EstimateGatewayTokenRequestCost(context.Background(), explicit, nil, nil)
	require.NoError(t, err)
	serverCapCost, err := svc.EstimateGatewayTokenRequestCost(context.Background(), serverCap, nil, nil)
	require.NoError(t, err)
	require.Less(t, explicitCost.ActualCost, serverCapCost.ActualCost)
}

func TestEnforceUnboundedTokenRequestLimit(t *testing.T) {
	tests := []struct {
		name            string
		body            string
		targetPath      string
		recognizedPaths []string
		wantChanged     bool
		wantLimit       int64
	}{
		{
			name:            "injects responses cap when omitted",
			body:            `{"model":"gpt-5.5","input":"hello"}`,
			targetPath:      "max_output_tokens",
			recognizedPaths: []string{"max_output_tokens"},
			wantChanged:     true,
			wantLimit:       unboundedTokenRequestMaxOutputTokens,
		},
		{
			name:            "preserves explicit responses cap",
			body:            `{"model":"gpt-5.5","input":"hello","max_output_tokens":2048}`,
			targetPath:      "max_output_tokens",
			recognizedPaths: []string{"max_output_tokens"},
			wantChanged:     false,
			wantLimit:       2048,
		},
		{
			name:            "injects chat completions cap when omitted",
			body:            `{"model":"gpt-5.5","messages":[{"role":"user","content":"hello"}]}`,
			targetPath:      "max_completion_tokens",
			recognizedPaths: []string{"max_completion_tokens", "max_tokens"},
			wantChanged:     true,
			wantLimit:       unboundedTokenRequestMaxOutputTokens,
		},
		{
			name:            "preserves legacy chat completions cap",
			body:            `{"model":"gpt-5.5","messages":[],"max_tokens":1024}`,
			targetPath:      "max_completion_tokens",
			recognizedPaths: []string{"max_completion_tokens", "max_tokens"},
			wantChanged:     false,
			wantLimit:       1024,
		},
		{
			name:            "injects anthropic cap when omitted",
			body:            `{"model":"claude-sonnet-4","messages":[{"role":"user","content":"hello"}]}`,
			targetPath:      "max_tokens",
			recognizedPaths: []string{"max_tokens"},
			wantChanged:     true,
			wantLimit:       unboundedTokenRequestMaxOutputTokens,
		},
		{
			name:       "injects gemini cap when omitted",
			body:       `{"contents":[{"role":"user","parts":[{"text":"hello"}]}]}`,
			targetPath: "generationConfig.maxOutputTokens",
			recognizedPaths: []string{
				"generationConfig.maxOutputTokens",
				"generation_config.max_output_tokens",
				"maxOutputTokens",
			},
			wantChanged: true,
			wantLimit:   unboundedTokenRequestMaxOutputTokens,
		},
		{
			name:       "preserves alternate gemini cap",
			body:       `{"generation_config":{"max_output_tokens":4096}}`,
			targetPath: "generationConfig.maxOutputTokens",
			recognizedPaths: []string{
				"generationConfig.maxOutputTokens",
				"generation_config.max_output_tokens",
				"maxOutputTokens",
			},
			wantChanged: false,
			wantLimit:   4096,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, changed, err := EnforceUnboundedTokenRequestLimit(
				[]byte(tt.body),
				tt.targetPath,
				tt.recognizedPaths...,
			)
			require.NoError(t, err)
			require.Equal(t, tt.wantChanged, changed)
			if changed {
				require.Equal(t, tt.wantLimit, gjson.GetBytes(got, tt.targetPath).Int())
			} else {
				found := int64(0)
				for _, path := range tt.recognizedPaths {
					if value := gjson.GetBytes(got, path); value.Exists() {
						found = value.Int()
						break
					}
				}
				require.Equal(t, tt.wantLimit, found)
			}
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
