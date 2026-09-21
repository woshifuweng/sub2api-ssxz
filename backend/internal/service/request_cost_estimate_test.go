package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
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

type requestEstimateBalanceCache struct {
	BillingCache
	balance float64
}

func (c *requestEstimateBalanceCache) GetUserBalance(context.Context, int64) (float64, error) {
	return c.balance, nil
}

func TestEstimateOpenAITokenRequestCostFastFlagsBalanceBoundaries(t *testing.T) {
	for _, withResolver := range []bool{false, true} {
		for _, force := range []bool{false, true} {
			for _, free := range []bool{false, true} {
				for _, tier := range []string{"", "default", "priority", " FAST ", "flex"} {
					t.Run(fmt.Sprintf("resolver=%t/force=%t/free=%t/tier=%s", withResolver, force, free, tier), func(t *testing.T) {
						cfg := &config.Config{}
						cfg.Default.RateMultiplier = 1
						cfg.Billing.MinimumBalanceReserve = 0.01
						billing := NewBillingService(cfg, nil)
						svc := &OpenAIGatewayService{billingService: billing, cfg: cfg}
						if withResolver {
							svc.resolver = NewModelPricingResolver(nil, billing)
						}
						group := &Group{ID: 305, Platform: PlatformOpenAI, Status: StatusActive, Hydrated: true,
							RateMultiplier: 1, ForceOpenAIFast: force, FreeOpenAIFast: free}
						user := &User{ID: 1}
						key := &APIKey{GroupID: &group.ID, Group: group, User: user}
						body := []byte(`{"model":"gpt-6","input":"hello","max_output_tokens":16384,"reasoning":{"effort":"max"}}`)
						if tier != "" {
							var err error
							body, err = sjson.SetBytes(body, "service_tier", tier)
							require.NoError(t, err)
						}
						original := string(body)
						got, err := svc.EstimateOpenAITokenRequestCost(context.Background(), "gpt-6", body, key, user)
						require.NoError(t, err)
						require.Equal(t, original, string(body), "preflight must not mutate the forwarded payload")

						// Independently model the final customer charge: force changes the
						// forwarded tier; Free Fast then discounts only priority/fast.
						chargedTier := tier
						priority := force || tier == "priority" || tier == " FAST "
						if priority {
							chargedTier = "priority"
							if free {
								chargedTier = ""
							}
						}
						effort := "max"
						want, err := svc.calculateOpenAIRecordUsageCost(context.Background(),
							&OpenAIForwardResult{ReasoningEffort: &effort}, key, []string{"gpt-6"}, 1, 1, 1, 1,
							UsageTokens{InputTokens: estimateRequestInputTokens(body), OutputTokens: 16384},
							chargedTier, nil, time.Now())
						require.NoError(t, err)
						require.InDelta(t, want.ActualCost, got.ActualCost, 1e-12)

						cache := &requestEstimateBalanceCache{}
						eligibility := &BillingCacheService{cache: cache, cfg: cfg}
						for _, balance := range []float64{want.ActualCost - 1e-8, want.ActualCost, want.ActualCost + 1e-8, 1} {
							cache.balance = balance
							err := eligibility.CheckBillingEligibilityForCost(context.Background(), user, key, group, nil, PlatformOpenAI, got.ActualCost)
							if balance < want.ActualCost {
								require.ErrorIs(t, err, ErrInsufficientBalance, "balance=%g", balance)
							} else {
								require.NoError(t, err, "balance=%g", balance)
							}
						}
					})
				}
			}
		}
	}
}

func TestEstimateOpenAITokenRequestCostFastFlagsPlatformScope(t *testing.T) {
	for _, platform := range []string{PlatformOpenAI, PlatformGrok, PlatformComposite} {
		for _, target := range []string{"", PlatformOpenAI, PlatformGrok} {
			for _, free := range []bool{false, true} {
				t.Run(fmt.Sprintf("group=%s/target=%s/free=%t", platform, target, free), func(t *testing.T) {
					cfg := &config.Config{}
					cfg.Default.RateMultiplier = 1
					billing := NewBillingService(cfg, nil)
					svc := &OpenAIGatewayService{billingService: billing, cfg: cfg, resolver: NewModelPricingResolver(nil, billing)}
					inputPrice, outputPrice, fastMultiplier, effortMultiplier := 0.001, 0.002, 3.0, 2.0
					group := &Group{ID: 305, Platform: platform, Status: StatusActive, Hydrated: true,
						RateMultiplier: 0.35, ForceOpenAIFast: true, FreeOpenAIFast: free,
						ModelPricing: []ChannelModelPricing{{Models: []string{"gpt-6-astra"}, BillingMode: BillingModeToken,
							InputPrice: &inputPrice, OutputPrice: &outputPrice, FastMultiplier: &fastMultiplier,
							MaxReasoningEffortMultiplier: &effortMultiplier}}}
					user := &User{ID: 1}
					key := &APIKey{GroupID: &group.ID, Group: group, User: user}
					ctx := WithResolvedTargetPlatform(context.Background(), target)
					body := []byte(`{"model":"gpt-6-astra","input":"hi","max_output_tokens":10,"reasoning":{"effort":"max"}}`)
					got, err := svc.EstimateOpenAITokenRequestCost(ctx, "gpt-6-astra", body, key, user)
					require.NoError(t, err)
					want := (float64(estimateRequestInputTokens(body))*inputPrice + 10*outputPrice) * 0.35 * effortMultiplier
					effectivePlatform := platform
					if target != "" {
						effectivePlatform = target
					}
					if (platform == PlatformOpenAI || platform == PlatformComposite) && effectivePlatform == PlatformOpenAI && !free {
						want *= fastMultiplier
					}
					require.InDelta(t, want, got.ActualCost, 1e-12)
				})
			}
		}
	}
}
