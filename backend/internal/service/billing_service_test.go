//go:build unit

package service

import (
	"math"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func newTestBillingService() *BillingService {
	return NewBillingService(&config.Config{}, nil)
}

func TestCalculateCost_BasicComputation(t *testing.T) {
	svc := newTestBillingService()

	// 使用 claude-sonnet-4 的回退价格：Input $3/MTok, Output $15/MTok
	tokens := UsageTokens{
		InputTokens:  1000,
		OutputTokens: 500,
	}
	cost, err := svc.CalculateCost("claude-sonnet-4", tokens, 1.0)
	require.NoError(t, err)

	// 1000 * 3e-6 = 0.003, 500 * 15e-6 = 0.0075
	expectedInput := 1000 * 3e-6
	expectedOutput := 500 * 15e-6
	require.InDelta(t, expectedInput, cost.InputCost, 1e-10)
	require.InDelta(t, expectedOutput, cost.OutputCost, 1e-10)
	require.InDelta(t, expectedInput+expectedOutput, cost.TotalCost, 1e-10)
	require.InDelta(t, expectedInput+expectedOutput, cost.ActualCost, 1e-10)
}

func TestCalculateCost_WithCacheTokens(t *testing.T) {
	svc := newTestBillingService()

	tokens := UsageTokens{
		InputTokens:         1000,
		OutputTokens:        500,
		CacheCreationTokens: 2000,
		CacheReadTokens:     3000,
	}
	cost, err := svc.CalculateCost("claude-sonnet-4", tokens, 1.0)
	require.NoError(t, err)

	expectedCacheCreation := 2000 * 3.75e-6
	expectedCacheRead := 3000 * 0.3e-6
	require.InDelta(t, expectedCacheCreation, cost.CacheCreationCost, 1e-10)
	require.InDelta(t, expectedCacheRead, cost.CacheReadCost, 1e-10)

	expectedTotal := cost.InputCost + cost.OutputCost + expectedCacheCreation + expectedCacheRead
	require.InDelta(t, expectedTotal, cost.TotalCost, 1e-10)
}

func TestCalculateCost_RateMultiplier(t *testing.T) {
	svc := newTestBillingService()

	tokens := UsageTokens{InputTokens: 1000, OutputTokens: 500}

	cost1x, err := svc.CalculateCost("claude-sonnet-4", tokens, 1.0)
	require.NoError(t, err)

	cost2x, err := svc.CalculateCost("claude-sonnet-4", tokens, 2.0)
	require.NoError(t, err)

	// TotalCost 不受倍率影响，ActualCost 翻倍
	require.InDelta(t, cost1x.TotalCost, cost2x.TotalCost, 1e-10)
	require.InDelta(t, cost1x.ActualCost*2, cost2x.ActualCost, 1e-10)
}

func TestCalculateCost_ZeroMultiplierDefaultsToOne(t *testing.T) {
	svc := newTestBillingService()

	tokens := UsageTokens{InputTokens: 1000}

	costZero, err := svc.CalculateCost("claude-sonnet-4", tokens, 0)
	require.NoError(t, err)

	costOne, err := svc.CalculateCost("claude-sonnet-4", tokens, 1.0)
	require.NoError(t, err)

	require.InDelta(t, costOne.ActualCost, costZero.ActualCost, 1e-10)
}

func TestCalculateCost_NegativeMultiplierDefaultsToOne(t *testing.T) {
	svc := newTestBillingService()

	tokens := UsageTokens{InputTokens: 1000}

	costNeg, err := svc.CalculateCost("claude-sonnet-4", tokens, -1.0)
	require.NoError(t, err)

	costOne, err := svc.CalculateCost("claude-sonnet-4", tokens, 1.0)
	require.NoError(t, err)

	require.InDelta(t, costOne.ActualCost, costNeg.ActualCost, 1e-10)
}

func TestGetModelPricing_FallbackMatchesByFamily(t *testing.T) {
	svc := newTestBillingService()

	tests := []struct {
		model         string
		expectedInput float64
	}{
		{"claude-opus-4.5-20250101", 5e-6},
		{"claude-3-opus-20240229", 15e-6},
		{"claude-sonnet-4-20250514", 3e-6},
		{"claude-3-5-sonnet-20241022", 3e-6},
		{"claude-3-5-haiku-20241022", 1e-6},
		{"claude-3-haiku-20240307", 0.25e-6},
	}

	for _, tt := range tests {
		pricing, err := svc.GetModelPricing(tt.model)
		require.NoError(t, err, "模型 %s", tt.model)
		require.InDelta(t, tt.expectedInput, pricing.InputPricePerToken, 1e-12, "模型 %s 输入价格", tt.model)
	}
}

func TestGetModelPricing_CaseInsensitive(t *testing.T) {
	svc := newTestBillingService()

	p1, err := svc.GetModelPricing("Claude-Sonnet-4")
	require.NoError(t, err)

	p2, err := svc.GetModelPricing("claude-sonnet-4")
	require.NoError(t, err)

	require.Equal(t, p1.InputPricePerToken, p2.InputPricePerToken)
}

func TestGetModelPricing_GLM52UsesOwnPrice(t *testing.T) {
	svc := newTestBillingService()

	pricing, err := svc.GetModelPricing("glm-5.2")
	require.NoError(t, err)
	require.NotNil(t, pricing)
	require.InDelta(t, 1.4e-6, pricing.InputPricePerToken, 1e-12)
	require.InDelta(t, 4.4e-6, pricing.OutputPricePerToken, 1e-12)
	require.InDelta(t, 0.26e-6, pricing.CacheReadPricePerToken, 1e-12)
}

func TestGetModelPricing_UnknownClaudeModelReturnsError(t *testing.T) {
	logSink, restore := captureStructuredLog(t)
	defer restore()

	svc := newTestBillingService()

	pricing, err := svc.GetModelPricing("claude-unknown-model")
	require.Error(t, err)
	require.Nil(t, pricing)
	require.Contains(t, err.Error(), "pricing not found")
	require.True(t, logSink.ContainsMessageAtLevel("pricing unavailable for exact model claude-unknown-model", "warn"))
}

func TestGetModelPricing_UnknownOpenAIModelReturnsError(t *testing.T) {
	svc := newTestBillingService()

	pricing, err := svc.GetModelPricing("gpt-unknown-model")
	require.Error(t, err)
	require.Nil(t, pricing)
	require.Contains(t, err.Error(), "pricing not found")
}

func TestGetModelPricing_OpenAIGPT51Fallback(t *testing.T) {
	svc := newTestBillingService()

	pricing, err := svc.GetModelPricing("gpt-5.1")
	require.NoError(t, err)
	require.NotNil(t, pricing)
	require.InDelta(t, 1.25e-6, pricing.InputPricePerToken, 1e-12)
}

func TestGetModelPricing_OpenAIGPT54Fallback(t *testing.T) {
	svc := newTestBillingService()

	pricing, err := svc.GetModelPricing("gpt-5.4")
	require.NoError(t, err)
	require.NotNil(t, pricing)
	require.InDelta(t, 2.5e-6, pricing.InputPricePerToken, 1e-12)
	require.InDelta(t, 15e-6, pricing.OutputPricePerToken, 1e-12)
	require.InDelta(t, 0.25e-6, pricing.CacheReadPricePerToken, 1e-12)
	require.Equal(t, 272000, pricing.LongContextInputThreshold)
	require.InDelta(t, 2.0, pricing.LongContextInputMultiplier, 1e-12)
	require.InDelta(t, 1.5, pricing.LongContextOutputMultiplier, 1e-12)
}

func TestGetModelPricing_OpenAIGPT54MiniFallback(t *testing.T) {
	svc := newTestBillingService()

	pricing, err := svc.GetModelPricing("gpt-5.4-mini")
	require.NoError(t, err)
	require.NotNil(t, pricing)
	require.InDelta(t, 7.5e-7, pricing.InputPricePerToken, 1e-12)
	require.InDelta(t, 4.5e-6, pricing.OutputPricePerToken, 1e-12)
	require.InDelta(t, 7.5e-8, pricing.CacheReadPricePerToken, 1e-12)
	require.Zero(t, pricing.LongContextInputThreshold)
}

func TestGetModelPricing_OpenAIGPT54NanoFallback(t *testing.T) {
	svc := newTestBillingService()

	pricing, err := svc.GetModelPricing("gpt-5.4-nano")
	require.NoError(t, err)
	require.NotNil(t, pricing)
	require.InDelta(t, 2e-7, pricing.InputPricePerToken, 1e-12)
	require.InDelta(t, 1.25e-6, pricing.OutputPricePerToken, 1e-12)
	require.InDelta(t, 2e-8, pricing.CacheReadPricePerToken, 1e-12)
	require.Zero(t, pricing.LongContextInputThreshold)
}

func TestCalculateCost_OpenAIGPT54LongContextAppliesWholeSessionMultipliers(t *testing.T) {
	svc := newTestBillingService()

	tokens := UsageTokens{
		InputTokens:  300000,
		OutputTokens: 4000,
	}

	cost, err := svc.CalculateCost("gpt-5.4-2026-03-05", tokens, 1.0)
	require.NoError(t, err)

	expectedInput := float64(tokens.InputTokens) * 2.5e-6 * 2.0
	expectedOutput := float64(tokens.OutputTokens) * 15e-6 * 1.5
	require.InDelta(t, expectedInput, cost.InputCost, 1e-10)
	require.InDelta(t, expectedOutput, cost.OutputCost, 1e-10)
	require.InDelta(t, expectedInput+expectedOutput, cost.TotalCost, 1e-10)
	require.InDelta(t, expectedInput+expectedOutput, cost.ActualCost, 1e-10)
}

func TestCalculateCost_OpenAIGPT56LongContextMultipliesAllInputCategories(t *testing.T) {
	svc := newTestBillingService()
	tokens := UsageTokens{
		InputTokens:         270000,
		OutputTokens:        4000,
		CacheCreationTokens: 2000,
		CacheReadTokens:     3000,
	}

	cost, err := svc.CalculateCost("gpt-5.6-terra", tokens, 1.0)
	require.NoError(t, err)

	require.InDelta(t, float64(tokens.InputTokens)*2.5e-6*2, cost.InputCost, 1e-10)
	require.InDelta(t, float64(tokens.OutputTokens)*15e-6*1.5, cost.OutputCost, 1e-10)
	require.InDelta(t, float64(tokens.CacheCreationTokens)*3.125e-6*2, cost.CacheCreationCost, 1e-10)
	require.InDelta(t, float64(tokens.CacheReadTokens)*0.25e-6*2, cost.CacheReadCost, 1e-10)
}

func TestGetFallbackPricing_FamilyMatching(t *testing.T) {
	svc := newTestBillingService()

	tests := []struct {
		name             string
		model            string
		expectedInput    float64
		expectNilPricing bool
	}{
		{name: "empty model", model: "   ", expectNilPricing: true},
		{name: "claude opus 4.6", model: "claude-opus-4.6-20260201", expectedInput: 5e-6},
		{name: "claude opus 4.5 alt separator", model: "claude-opus-4-5-20260101", expectedInput: 5e-6},
		{name: "claude generic model has no guessed fallback", model: "claude-foo-bar", expectNilPricing: true},
		{name: "claude sonnet 5 has exact dated fallback", model: "claude-sonnet-5", expectedInput: 2e-6},
		{name: "future claude haiku has no guessed fallback", model: "claude-haiku-5", expectNilPricing: true},
		{name: "gemini explicit fallback", model: "gemini-3-1-pro", expectedInput: 2e-6},
		{name: "gemini unknown no fallback", model: "gemini-2.0-pro", expectNilPricing: true},
		{name: "openai gpt5.1", model: "gpt-5.1", expectedInput: 1.25e-6},
		{name: "openai gpt5.4", model: "gpt-5.4", expectedInput: 2.5e-6},
		{name: "openai gpt5.4 mini", model: "gpt-5.4-mini", expectedInput: 7.5e-7},
		{name: "openai gpt5.4 nano", model: "gpt-5.4-nano", expectedInput: 2e-7},
		{name: "openai gpt5.3 codex", model: "gpt-5.3-codex", expectedInput: 1.5e-6},
		{name: "openai gpt5.1 codex max alias", model: "gpt-5.1-codex-max", expectedInput: 1.5e-6},
		{name: "openai codex mini latest alias", model: "codex-mini-latest", expectedInput: 1.5e-6},
		{name: "openai unknown no fallback", model: "gpt-unknown-model", expectNilPricing: true},
		{
			name:              "deepseek v4 pro",
			model:             "deepseek-v4-pro",
			expectedInput:     4.35e-7,
			expectedOutput:    floatPtr(8.7e-7),
			expectedCacheRead: floatPtr(3.625e-9),
		},
		{
			name:              "deepseek v4 flash",
			model:             "deepseek-v4-flash",
			expectedInput:     1.4e-7,
			expectedOutput:    floatPtr(2.8e-7),
			expectedCacheRead: floatPtr(2.8e-9),
		},
		{
			name:              "deepseek chat alias → flash",
			model:             "deepseek-chat",
			expectedInput:     1.4e-7,
			expectedOutput:    floatPtr(2.8e-7),
			expectedCacheRead: floatPtr(2.8e-9),
		},
		{
			name:              "deepseek reasoner alias → flash",
			model:             "deepseek-reasoner",
			expectedInput:     1.4e-7,
			expectedOutput:    floatPtr(2.8e-7),
			expectedCacheRead: floatPtr(2.8e-9),
		},

		// ---- 智谱 GLM（z.ai USD 口径）----
		{
			name:              "glm 5.1 flagship",
			model:             "glm-5.1",
			expectedInput:     1.4e-6,
			expectedOutput:    floatPtr(4.4e-6),
			expectedCacheRead: floatPtr(0.26e-6),
		},
		{
			name:              "glm 5 base",
			model:             "glm-5",
			expectedInput:     1e-6,
			expectedOutput:    floatPtr(3.2e-6),
			expectedCacheRead: floatPtr(0.2e-6),
		},
		{
			name:              "glm 5 turbo",
			model:             "glm-5-turbo",
			expectedInput:     1.2e-6,
			expectedOutput:    floatPtr(4e-6),
			expectedCacheRead: floatPtr(0.24e-6),
		},
		{
			name:              "glm 4.7",
			model:             "glm-4.7",
			expectedInput:     0.6e-6,
			expectedOutput:    floatPtr(2.2e-6),
			expectedCacheRead: floatPtr(0.11e-6),
		},
		{
			name:              "glm 4.6",
			model:             "glm-4.6",
			expectedInput:     0.6e-6,
			expectedOutput:    floatPtr(2.2e-6),
			expectedCacheRead: floatPtr(0.11e-6),
		},
		{
			name:              "glm 4.5",
			model:             "glm-4.5",
			expectedInput:     0.6e-6,
			expectedOutput:    floatPtr(2.2e-6),
			expectedCacheRead: floatPtr(0.11e-6),
		},
		{
			name:              "glm 4.5-x premium",
			model:             "glm-4.5-x",
			expectedInput:     2.2e-6,
			expectedOutput:    floatPtr(8.9e-6),
			expectedCacheRead: floatPtr(0.45e-6),
		},
		{
			name:              "glm 4.5-air lightweight",
			model:             "glm-4.5-air",
			expectedInput:     0.2e-6,
			expectedOutput:    floatPtr(1.1e-6),
			expectedCacheRead: floatPtr(0.03e-6),
		},
		{
			name:              "glm 4.7-flashx",
			model:             "glm-4.7-flashx",
			expectedInput:     0.07e-6,
			expectedOutput:    floatPtr(0.4e-6),
			expectedCacheRead: floatPtr(0.01e-6),
		},
		{
			name:              "glm 4.5-flash free tier",
			model:             "glm-4.5-flash",
			expectedInput:     0, // Free tier on z.ai
			expectedOutput:    floatPtr(0),
			expectedCacheRead: floatPtr(0),
		},
		{
			name:              "glm 4.7-flash free tier",
			model:             "glm-4.7-flash",
			expectedInput:     0,
			expectedOutput:    floatPtr(0),
			expectedCacheRead: floatPtr(0),
		},
		{
			name:           "glm 4-32b legacy",
			model:          "glm-4-32b-0414-128k",
			expectedInput:  0.1e-6,
			expectedOutput: floatPtr(0.1e-6),
		},
		// 关键：5.1 必须先于 5 匹配（避免被 glm-5 抢走）
		{
			name:              "glm 5.1 vs glm 5 ordering (verbatim 5.1)",
			model:             "glm-5.1",
			expectedInput:     1.4e-6, // = glm-5.1 价格
			expectedOutput:    floatPtr(4.4e-6),
			expectedCacheRead: floatPtr(0.26e-6),
		},
		{
			name:              "glm 4.5-air vs glm 4.5 ordering",
			model:             "glm-4.5-air",
			expectedInput:     0.2e-6, // = glm-4.5-air 价格（不是 glm-4.5 的 0.6e-6）
			expectedOutput:    floatPtr(1.1e-6),
			expectedCacheRead: floatPtr(0.03e-6),
		},

		// ---- 月之暗面 Kimi ----
		{
			name:              "kimi k3 flagship",
			model:             "kimi-k3",
			expectedInput:     3e-6,
			expectedOutput:    floatPtr(15e-6),
			expectedCacheRead: floatPtr(0.30e-6),
		},
		{
			name:              "kimi code bare alias k3",
			model:             "k3",
			expectedInput:     3e-6,
			expectedOutput:    floatPtr(15e-6),
			expectedCacheRead: floatPtr(0.30e-6),
		},
		{
			name:              "kimi code bare alias k3-256k",
			model:             "k3-256k",
			expectedInput:     3e-6,
			expectedOutput:    floatPtr(15e-6),
			expectedCacheRead: floatPtr(0.30e-6),
		},
		{
			name:              "kimi k3 path suffix moonshot",
			model:             "moonshot/kimi-k3",
			expectedInput:     3e-6,
			expectedOutput:    floatPtr(15e-6),
			expectedCacheRead: floatPtr(0.30e-6),
		},
		{
			name:              "kimi code bare path suffix",
			model:             "kimi-code/k3",
			expectedInput:     3e-6,
			expectedOutput:    floatPtr(15e-6),
			expectedCacheRead: floatPtr(0.30e-6),
		},
		{
			name:              "kimi k2.6 flagship",
			model:             "kimi-k2.6",
			expectedInput:     0.95e-6,
			expectedOutput:    floatPtr(4e-6),
			expectedCacheRead: floatPtr(0.15e-6),
		},
		{
			name:              "kimi for coding explicit alias",
			model:             "kimi-for-coding",
			expectedInput:     0.95e-6,
			expectedOutput:    floatPtr(4e-6),
			expectedCacheRead: floatPtr(0.15e-6),
		},
		{
			name:              "kimi k2.5",
			model:             "kimi-k2.5",
			expectedInput:     0.60e-6,
			expectedOutput:    floatPtr(3e-6),
			expectedCacheRead: floatPtr(0.098e-6),
		},
		{
			name:              "kimi k2-thinking",
			model:             "kimi-k2-thinking",
			expectedInput:     0.56e-6,
			expectedOutput:    floatPtr(2.24e-6),
			expectedCacheRead: floatPtr(0.14e-6),
		},
		{
			name:              "kimi k2 base",
			model:             "kimi-k2",
			expectedInput:     0.56e-6,
			expectedOutput:    floatPtr(2.24e-6),
			expectedCacheRead: floatPtr(0.14e-6),
		},
		// 关键：k2.6 / k2.5 / k2-thinking 必须先于 k2 匹配
		{
			name:              "kimi k2.6 vs k2 ordering",
			model:             "kimi-k2.6",
			expectedInput:     0.95e-6, // = k2.6 不是 k2 的 0.56e-6
			expectedOutput:    floatPtr(4e-6),
			expectedCacheRead: floatPtr(0.15e-6),
		},
		{
			name:              "kimi k2 thinking hyphenated variant",
			model:             "kimi-k2-thinking-preview",
			expectedInput:     0.56e-6,
			expectedOutput:    floatPtr(2.24e-6),
			expectedCacheRead: floatPtr(0.14e-6),
		},

		// ---- MiniMax M 系列 ----
		{
			name:              "minimax m3",
			model:             "minimax-m3",
			expectedInput:     0.60e-6,
			expectedOutput:    floatPtr(2.40e-6),
			expectedCacheRead: floatPtr(0.12e-6),
		},
		{
			name:              "minimax m3 long ctx boundary keep standard tier",
			model:             "minimax-m3-long", // 仍按 standard tier (≤512K)
			expectedInput:     0.60e-6,
			expectedOutput:    floatPtr(2.40e-6),
			expectedCacheRead: floatPtr(0.12e-6),
		},
		{
			name:              "minimax m2.7",
			model:             "minimax-m2.7",
			expectedInput:     0.30e-6,
			expectedOutput:    floatPtr(1.20e-6),
			expectedCacheRead: floatPtr(0.06e-6),
		},
		{
			name:              "minimax m2.7 highspeed",
			model:             "minimax-m2.7-highspeed",
			expectedInput:     0.60e-6,
			expectedOutput:    floatPtr(2.40e-6),
			expectedCacheRead: floatPtr(0.06e-6),
		},
		{
			name:              "minimax m2.5",
			model:             "minimax-m2.5",
			expectedInput:     0.30e-6,
			expectedOutput:    floatPtr(1.20e-6),
			expectedCacheRead: floatPtr(0.03e-6),
		},
		{
			name:              "minimax m2 legacy",
			model:             "minimax-m2",
			expectedInput:     0.30e-6,
			expectedOutput:    floatPtr(1.20e-6),
			expectedCacheRead: floatPtr(0.03e-6),
		},

		// ---- 火山方舟 豆包 Embedding（多模态向量化）----
		{
			name:           "doubao embedding vision text rate",
			model:          "doubao-embedding-vision",
			expectedInput:  0.098e-6,
			expectedOutput: floatPtr(0),
		},
		{
			name:          "doubao embedding vision versioned alias",
			model:         "doubao-embedding-vision-251215",
			expectedInput: 0.098e-6,
		},

		// ---- 负向用例 ----
		{name: "qwen unknown no fallback", model: "qwen-max", expectNilPricing: true},
		// doubao-pro / doubao-embedding（纯文本）不在白名单，不回退；仅 doubao-embedding-vision 显式命中。
		{name: "doubao unknown no fallback", model: "doubao-pro", expectNilPricing: true},
		{name: "doubao text embedding no fallback", model: "doubao-embedding-text-240515", expectNilPricing: true},
		{name: "hunyuan unknown no fallback", model: "hunyuan-t1", expectNilPricing: true},
		{name: "moonshot v1 not covered", model: "moonshot-v1-8k", expectNilPricing: true},
		// bare k3 仅精确/后缀匹配：相似未知型号不得因含 "k3" 误命中。
		{name: "k3-like unknown no fallback", model: "foo-k3-bar", expectNilPricing: true},
		// 路径最后一段不是 /k3：foo-k3 不得因 HasSuffix("/k3") 或 Contains 误命中。
		{name: "path segment not bare k3 no fallback", model: "vendor/foo-k3", expectNilPricing: true},
		// kimi-k3 非 Contains：kimi-k30 / 内嵌 foo-kimi-k3-bar 不得误命中。
		{name: "kimi-k30 unknown no fallback", model: "kimi-k30", expectNilPricing: true},
		{name: "embedded kimi-k3 unknown no fallback", model: "foo-kimi-k3-bar", expectNilPricing: true},
		// kimi-k3[1m] 是 Claude Code 上下文选择语法，不是 Kimi API 模型 ID，不命中 fallback。
		{name: "kimi-k3[1m] not an API model id no fallback", model: "kimi-k3[1m]", expectNilPricing: true},
		{name: "path kimi-k3[1m] not an API model id no fallback", model: "moonshot/kimi-k3[1m]", expectNilPricing: true},
		// kimi-k2-0905 / kimi-k2-0711 官方未公布独立价，走 kimi-k2 隐性回退（接受）——
		// 如未来官方公布独立价，需在 getFallbackPricing 加显式分支。
		{
			name:              "kimi k2-0905-preview implicit fallback to k2",
			model:             "kimi-k2-0905-preview",
			expectedInput:     0.56e-6,
			expectedOutput:    floatPtr(2.24e-6),
			expectedCacheRead: floatPtr(0.14e-6),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pricing := svc.getFallbackPricing(tt.model)
			if tt.expectNilPricing {
				require.Nil(t, pricing)
				return
			}
			require.NotNil(t, pricing)
			require.InDelta(t, tt.expectedInput, pricing.InputPricePerToken, 1e-12)
		})
	}
}
func TestCalculateCostWithLongContext_BelowThreshold(t *testing.T) {
	svc := newTestBillingService()

	tokens := UsageTokens{
		InputTokens:     50000,
		OutputTokens:    1000,
		CacheReadTokens: 100000,
	}
	// 总输入 150k < 200k 阈值，应走正常计费
	cost, err := svc.CalculateCostWithLongContext("claude-sonnet-4", tokens, 1.0, 200000, 2.0)
	require.NoError(t, err)

	normalCost, err := svc.CalculateCost("claude-sonnet-4", tokens, 1.0)
	require.NoError(t, err)

	require.InDelta(t, normalCost.ActualCost, cost.ActualCost, 1e-10)
}

func TestCalculateCostWithLongContext_AboveThreshold_CacheExceedsThreshold(t *testing.T) {
	svc := newTestBillingService()

	// 缓存 210k + 输入 10k = 220k > 200k 阈值
	// 缓存已超阈值：范围内 200k 缓存，范围外 10k 缓存 + 10k 输入
	tokens := UsageTokens{
		InputTokens:     10000,
		OutputTokens:    1000,
		CacheReadTokens: 210000,
	}
	cost, err := svc.CalculateCostWithLongContext("claude-sonnet-4", tokens, 1.0, 200000, 2.0)
	require.NoError(t, err)

	// 范围内：200k cache + 0 input + 1k output
	inRange, _ := svc.CalculateCost("claude-sonnet-4", UsageTokens{
		InputTokens:     0,
		OutputTokens:    1000,
		CacheReadTokens: 200000,
	}, 1.0)

	// 范围外：10k cache + 10k input，倍率 2.0
	outRange, _ := svc.CalculateCost("claude-sonnet-4", UsageTokens{
		InputTokens:     10000,
		CacheReadTokens: 10000,
	}, 2.0)

	require.InDelta(t, inRange.ActualCost+outRange.ActualCost, cost.ActualCost, 1e-10)
}

func TestCalculateCostWithLongContext_AboveThreshold_CacheBelowThreshold(t *testing.T) {
	svc := newTestBillingService()

	// 缓存 100k + 输入 150k = 250k > 200k 阈值
	// 缓存未超阈值：范围内 100k 缓存 + 100k 输入，范围外 50k 输入
	tokens := UsageTokens{
		InputTokens:     150000,
		OutputTokens:    1000,
		CacheReadTokens: 100000,
	}
	cost, err := svc.CalculateCostWithLongContext("claude-sonnet-4", tokens, 1.0, 200000, 2.0)
	require.NoError(t, err)

	require.True(t, cost.ActualCost > 0, "费用应大于 0")

	// 正常费用不含长上下文
	normalCost, _ := svc.CalculateCost("claude-sonnet-4", tokens, 1.0)
	require.True(t, cost.ActualCost > normalCost.ActualCost, "长上下文费用应高于正常费用")
}

func TestCalculateCostWithLongContext_DisabledThreshold(t *testing.T) {
	svc := newTestBillingService()

	tokens := UsageTokens{InputTokens: 300000, CacheReadTokens: 0}

	// threshold <= 0 应禁用长上下文计费
	cost1, err := svc.CalculateCostWithLongContext("claude-sonnet-4", tokens, 1.0, 0, 2.0)
	require.NoError(t, err)

	cost2, err := svc.CalculateCost("claude-sonnet-4", tokens, 1.0)
	require.NoError(t, err)

	require.InDelta(t, cost2.ActualCost, cost1.ActualCost, 1e-10)
}

func TestCalculateCostWithLongContext_ExtraMultiplierLessEqualOne(t *testing.T) {
	svc := newTestBillingService()

	tokens := UsageTokens{InputTokens: 300000}

	// extraMultiplier <= 1 应禁用长上下文计费
	cost, err := svc.CalculateCostWithLongContext("claude-sonnet-4", tokens, 1.0, 200000, 1.0)
	require.NoError(t, err)

	normalCost, err := svc.CalculateCost("claude-sonnet-4", tokens, 1.0)
	require.NoError(t, err)

	require.InDelta(t, normalCost.ActualCost, cost.ActualCost, 1e-10)
}

func TestCalculateImageCost(t *testing.T) {
	svc := newTestBillingService()

	price := 0.134
	cfg := &ImagePriceConfig{Price1K: &price}
	cost := svc.CalculateImageCost("gpt-image-1", "1K", 3, cfg, 1.0, "")

	require.InDelta(t, 0.134*3, cost.TotalCost, 1e-10)
	require.InDelta(t, 0.134*3, cost.ActualCost, 1e-10)
}

func TestCalculateSoraVideoCost(t *testing.T) {
	svc := newTestBillingService()

	price := 0.5
	cfg := &SoraPriceConfig{VideoPricePerRequest: &price}
	cost := svc.CalculateSoraVideoCost("sora-video", cfg, 1.0)

	require.InDelta(t, 0.5, cost.TotalCost, 1e-10)
}

func TestCalculateSoraVideoCostForCount(t *testing.T) {
	svc := newTestBillingService()

	price := 0.5
	cfg := &SoraPriceConfig{VideoPricePerRequest: &price}
	cost := svc.CalculateSoraVideoCostForCount("sora-video", 3, cfg, 2.0)

	require.InDelta(t, 1.5, cost.TotalCost, 1e-10)
	require.InDelta(t, 3.0, cost.ActualCost, 1e-10)
}

func TestCalculateSoraVideoCost_HDModel(t *testing.T) {
	svc := newTestBillingService()

	hdPrice := 1.0
	normalPrice := 0.5
	cfg := &SoraPriceConfig{
		VideoPricePerRequest:   &normalPrice,
		VideoPricePerRequestHD: &hdPrice,
	}
	cost := svc.CalculateSoraVideoCost("sora2pro-hd", cfg, 1.0)
	require.InDelta(t, 1.0, cost.TotalCost, 1e-10)
}

func TestIsModelSupported(t *testing.T) {
	svc := newTestBillingService()

	require.True(t, svc.IsModelSupported("claude-sonnet-4"))
	require.True(t, svc.IsModelSupported("Claude-Opus-4.5"))
	require.True(t, svc.IsModelSupported("claude-3-haiku"))
	require.False(t, svc.IsModelSupported("gpt-4o"))
	require.False(t, svc.IsModelSupported("gemini-pro"))
}

func TestCalculateCost_ZeroTokens(t *testing.T) {
	svc := newTestBillingService()

	cost, err := svc.CalculateCost("claude-sonnet-4", UsageTokens{}, 1.0)
	require.NoError(t, err)
	require.Equal(t, 0.0, cost.TotalCost)
	require.Equal(t, 0.0, cost.ActualCost)
}

func TestCalculateCostWithConfig(t *testing.T) {
	cfg := &config.Config{}
	cfg.Default.RateMultiplier = 1.5
	svc := NewBillingService(cfg, nil)

	tokens := UsageTokens{InputTokens: 1000, OutputTokens: 500}
	cost, err := svc.CalculateCostWithConfig("claude-sonnet-4", tokens)
	require.NoError(t, err)

	expected, _ := svc.CalculateCost("claude-sonnet-4", tokens, 1.5)
	require.InDelta(t, expected.ActualCost, cost.ActualCost, 1e-10)
}

func TestCalculateCostWithConfig_ZeroMultiplier(t *testing.T) {
	cfg := &config.Config{}
	cfg.Default.RateMultiplier = 0
	svc := NewBillingService(cfg, nil)

	tokens := UsageTokens{InputTokens: 1000}
	cost, err := svc.CalculateCostWithConfig("claude-sonnet-4", tokens)
	require.NoError(t, err)

	// 倍率 <=0 时默认 1.0
	expected, _ := svc.CalculateCost("claude-sonnet-4", tokens, 1.0)
	require.InDelta(t, expected.ActualCost, cost.ActualCost, 1e-10)
}

func TestGetEstimatedCost(t *testing.T) {
	svc := newTestBillingService()

	est, err := svc.GetEstimatedCost("claude-sonnet-4", 1000, 500)
	require.NoError(t, err)
	require.True(t, est > 0)
}

func TestListSupportedModels(t *testing.T) {
	svc := newTestBillingService()

	models := svc.ListSupportedModels()
	require.NotEmpty(t, models)
	require.GreaterOrEqual(t, len(models), 6)
}

func TestGetPricingServiceStatus_NilService(t *testing.T) {
	svc := newTestBillingService()

	status := svc.GetPricingServiceStatus()
	require.NotNil(t, status)
	require.Equal(t, "using fallback", status["last_updated"])
}

func TestForceUpdatePricing_NilService(t *testing.T) {
	svc := newTestBillingService()

	err := svc.ForceUpdatePricing()
	require.Error(t, err)
	require.Contains(t, err.Error(), "not initialized")
}

func TestCalculateSoraImageCost(t *testing.T) {
	svc := newTestBillingService()

	price360 := 0.05
	price540 := 0.08
	cfg := &SoraPriceConfig{ImagePrice360: &price360, ImagePrice540: &price540}

	cost := svc.CalculateSoraImageCost("360", 2, cfg, 1.0)
	require.InDelta(t, 0.10, cost.TotalCost, 1e-10)

	cost540 := svc.CalculateSoraImageCost("540", 1, cfg, 2.0)
	require.InDelta(t, 0.08, cost540.TotalCost, 1e-10)
	require.InDelta(t, 0.16, cost540.ActualCost, 1e-10)
}

func TestCalculateSoraImageCost_ZeroCount(t *testing.T) {
	svc := newTestBillingService()
	cost := svc.CalculateSoraImageCost("360", 0, nil, 1.0)
	require.Equal(t, 0.0, cost.TotalCost)
}

func TestCalculateSoraVideoCost_NilConfig(t *testing.T) {
	svc := newTestBillingService()
	cost := svc.CalculateSoraVideoCost("sora-video", nil, 1.0)
	require.Equal(t, 0.0, cost.TotalCost)
}

func TestCalculateCostWithLongContext_PropagatesError(t *testing.T) {
	// 使用空的 fallback prices 让 GetModelPricing 失败
	svc := &BillingService{
		cfg:            &config.Config{},
		fallbackPrices: make(map[string]*ModelPricing),
	}

	tokens := UsageTokens{InputTokens: 300000, CacheReadTokens: 0}
	_, err := svc.CalculateCostWithLongContext("unknown-model", tokens, 1.0, 200000, 2.0)
	require.Error(t, err)
	require.Contains(t, err.Error(), "pricing not found")
}

func TestCalculateCost_SupportsCacheBreakdown(t *testing.T) {
	svc := &BillingService{
		cfg: &config.Config{},
		fallbackPrices: map[string]*ModelPricing{
			"claude-sonnet-4": {
				InputPricePerToken:     3e-6,
				OutputPricePerToken:    15e-6,
				SupportsCacheBreakdown: true,
				CacheCreation5mPrice:   4e-6, // per token
				CacheCreation1hPrice:   5e-6, // per token
			},
		},
	}

	tokens := UsageTokens{
		InputTokens:           1000,
		OutputTokens:          500,
		CacheCreation5mTokens: 100000,
		CacheCreation1hTokens: 50000,
	}
	cost, err := svc.CalculateCost("claude-sonnet-4", tokens, 1.0)
	require.NoError(t, err)

	expected5m := float64(tokens.CacheCreation5mTokens) * 4e-6
	expected1h := float64(tokens.CacheCreation1hTokens) * 5e-6
	require.InDelta(t, expected5m+expected1h, cost.CacheCreationCost, 1e-10)
}

func TestCalculateCost_LargeTokenCount(t *testing.T) {
	svc := newTestBillingService()

	tokens := UsageTokens{
		InputTokens:  1_000_000,
		OutputTokens: 1_000_000,
	}
	cost, err := svc.CalculateCost("claude-sonnet-4", tokens, 1.0)
	require.NoError(t, err)

	// Input: 1M * 3e-6 = $3, Output: 1M * 15e-6 = $15
	require.InDelta(t, 3.0, cost.InputCost, 1e-6)
	require.InDelta(t, 15.0, cost.OutputCost, 1e-6)
	require.False(t, math.IsNaN(cost.TotalCost))
	require.False(t, math.IsInf(cost.TotalCost, 0))
}

func TestServiceTierCostMultiplier(t *testing.T) {
	require.InDelta(t, 2.0, serviceTierCostMultiplier("priority"), 1e-12)
	require.InDelta(t, 2.0, serviceTierCostMultiplier(" Priority "), 1e-12)
	require.InDelta(t, 0.5, serviceTierCostMultiplier("flex"), 1e-12)
	require.InDelta(t, 1.0, serviceTierCostMultiplier(""), 1e-12)
	require.InDelta(t, 1.0, serviceTierCostMultiplier("default"), 1e-12)
}

func TestCalculateCostWithServiceTier_OpenAIPriorityUsesPriorityPricing(t *testing.T) {
	svc := newTestBillingService()
	tokens := UsageTokens{InputTokens: 100, OutputTokens: 50, CacheReadTokens: 20}

	baseCost, err := svc.CalculateCost("gpt-5.1-codex", tokens, 1.0)
	require.NoError(t, err)

	priorityCost, err := svc.CalculateCostWithServiceTier("gpt-5.1-codex", tokens, 1.0, "priority")
	require.NoError(t, err)

	require.InDelta(t, baseCost.InputCost*2, priorityCost.InputCost, 1e-10)
	require.InDelta(t, baseCost.OutputCost*2, priorityCost.OutputCost, 1e-10)
	require.InDelta(t, baseCost.CacheReadCost*2, priorityCost.CacheReadCost, 1e-10)
	require.InDelta(t, baseCost.TotalCost*2, priorityCost.TotalCost, 1e-10)
}

func TestCalculateCostWithServiceTier_FlexAppliesHalfMultiplier(t *testing.T) {
	svc := newTestBillingService()
	tokens := UsageTokens{InputTokens: 100, OutputTokens: 50, CacheCreationTokens: 40, CacheReadTokens: 20}

	baseCost, err := svc.CalculateCost("gpt-5.4", tokens, 1.0)
	require.NoError(t, err)

	flexCost, err := svc.CalculateCostWithServiceTier("gpt-5.4", tokens, 1.0, "flex")
	require.NoError(t, err)

	require.InDelta(t, baseCost.InputCost*0.5, flexCost.InputCost, 1e-10)
	require.InDelta(t, baseCost.OutputCost*0.5, flexCost.OutputCost, 1e-10)
	require.InDelta(t, baseCost.CacheCreationCost*0.5, flexCost.CacheCreationCost, 1e-10)
	require.InDelta(t, baseCost.CacheReadCost*0.5, flexCost.CacheReadCost, 1e-10)
	require.InDelta(t, baseCost.TotalCost*0.5, flexCost.TotalCost, 1e-10)
}

func TestCalculateCostWithServiceTier_Gpt54MiniPriorityFallsBackToTierMultiplier(t *testing.T) {
	svc := newTestBillingService()
	tokens := UsageTokens{InputTokens: 120, OutputTokens: 30, CacheCreationTokens: 12, CacheReadTokens: 8}

	baseCost, err := svc.CalculateCost("gpt-5.4-mini", tokens, 1.0)
	require.NoError(t, err)

	priorityCost, err := svc.CalculateCostWithServiceTier("gpt-5.4-mini", tokens, 1.0, "priority")
	require.NoError(t, err)

	require.InDelta(t, baseCost.InputCost*2, priorityCost.InputCost, 1e-10)
	require.InDelta(t, baseCost.OutputCost*2, priorityCost.OutputCost, 1e-10)
	require.InDelta(t, baseCost.CacheCreationCost*2, priorityCost.CacheCreationCost, 1e-10)
	require.InDelta(t, baseCost.CacheReadCost*2, priorityCost.CacheReadCost, 1e-10)
	require.InDelta(t, baseCost.TotalCost*2, priorityCost.TotalCost, 1e-10)
}

func TestCalculateCostWithServiceTier_Gpt54NanoFlexAppliesHalfMultiplier(t *testing.T) {
	svc := newTestBillingService()
	tokens := UsageTokens{InputTokens: 100, OutputTokens: 50, CacheCreationTokens: 40, CacheReadTokens: 20}

	baseCost, err := svc.CalculateCost("gpt-5.4-nano", tokens, 1.0)
	require.NoError(t, err)

	flexCost, err := svc.CalculateCostWithServiceTier("gpt-5.4-nano", tokens, 1.0, "flex")
	require.NoError(t, err)

	require.InDelta(t, baseCost.InputCost*0.5, flexCost.InputCost, 1e-10)
	require.InDelta(t, baseCost.OutputCost*0.5, flexCost.OutputCost, 1e-10)
	require.InDelta(t, baseCost.CacheCreationCost*0.5, flexCost.CacheCreationCost, 1e-10)
	require.InDelta(t, baseCost.CacheReadCost*0.5, flexCost.CacheReadCost, 1e-10)
	require.InDelta(t, baseCost.TotalCost*0.5, flexCost.TotalCost, 1e-10)
}

func TestCalculateCostWithServiceTier_PriorityFallsBackToTierMultiplierWithoutExplicitPriorityPrice(t *testing.T) {
	svc := newTestBillingService()
	tokens := UsageTokens{InputTokens: 120, OutputTokens: 30, CacheCreationTokens: 12, CacheReadTokens: 8}

	baseCost, err := svc.CalculateCost("claude-sonnet-4", tokens, 1.0)
	require.NoError(t, err)

	priorityCost, err := svc.CalculateCostWithServiceTier("claude-sonnet-4", tokens, 1.0, "priority")
	require.NoError(t, err)

	require.InDelta(t, baseCost.InputCost*2, priorityCost.InputCost, 1e-10)
	require.InDelta(t, baseCost.OutputCost*2, priorityCost.OutputCost, 1e-10)
	require.InDelta(t, baseCost.CacheCreationCost*2, priorityCost.CacheCreationCost, 1e-10)
	require.InDelta(t, baseCost.CacheReadCost*2, priorityCost.CacheReadCost, 1e-10)
	require.InDelta(t, baseCost.TotalCost*2, priorityCost.TotalCost, 1e-10)
}

func TestBillingServiceGetModelPricing_UsesDynamicPriorityFields(t *testing.T) {
	pricingSvc := &PricingService{
		pricingData: map[string]*LiteLLMModelPricing{
			"gpt-5.4": {
				InputCostPerToken:               2.5e-6,
				InputCostPerTokenPriority:       5e-6,
				OutputCostPerToken:              15e-6,
				OutputCostPerTokenPriority:      30e-6,
				CacheCreationInputTokenCost:     2.5e-6,
				CacheReadInputTokenCost:         0.25e-6,
				CacheReadInputTokenCostPriority: 0.5e-6,
				LongContextInputTokenThreshold:  272000,
				LongContextInputCostMultiplier:  2.0,
				LongContextOutputCostMultiplier: 1.5,
			},
		},
	}
	svc := NewBillingService(&config.Config{}, pricingSvc)

	pricing, err := svc.GetModelPricing("gpt-5.4")
	require.NoError(t, err)
	require.InDelta(t, 2.5e-6, pricing.InputPricePerToken, 1e-12)
	require.InDelta(t, 5e-6, pricing.InputPricePerTokenPriority, 1e-12)
	require.InDelta(t, 15e-6, pricing.OutputPricePerToken, 1e-12)
	require.InDelta(t, 30e-6, pricing.OutputPricePerTokenPriority, 1e-12)
	require.InDelta(t, 0.25e-6, pricing.CacheReadPricePerToken, 1e-12)
	require.InDelta(t, 0.5e-6, pricing.CacheReadPricePerTokenPriority, 1e-12)
	require.Equal(t, 272000, pricing.LongContextInputThreshold)
	require.InDelta(t, 2.0, pricing.LongContextInputMultiplier, 1e-12)
	require.InDelta(t, 1.5, pricing.LongContextOutputMultiplier, 1e-12)
}

func TestBillingServiceGetModelPricing_UsesOfficialExactPricesWithoutDynamicData(t *testing.T) {
	svc := NewBillingService(&config.Config{}, nil)

	tests := []struct {
		model       string
		inputPrice  float64
		outputPrice float64
	}{
		{model: "claude-fable-5", inputPrice: 10e-6, outputPrice: 50e-6},
		{model: "claude-opus-4-6", inputPrice: 5e-6, outputPrice: 25e-6},
		{model: "claude-opus-4-7", inputPrice: 5e-6, outputPrice: 25e-6},
		{model: "claude-opus-4-8", inputPrice: 5e-6, outputPrice: 25e-6},
		{model: "claude-sonnet-4-6", inputPrice: 3e-6, outputPrice: 15e-6},
		{model: "claude-sonnet-5", inputPrice: 2e-6, outputPrice: 10e-6},
		{model: "gpt-5.4", inputPrice: 2.5e-6, outputPrice: 15e-6},
		{model: "gpt-5.4-mini", inputPrice: 0.75e-6, outputPrice: 4.5e-6},
		{model: "gpt-5.5", inputPrice: 5e-6, outputPrice: 30e-6},
		{model: "gpt-5.6-sol", inputPrice: 5e-6, outputPrice: 30e-6},
		{model: "gpt-5.6-terra", inputPrice: 2.5e-6, outputPrice: 15e-6},
		{model: "gpt-5.6-luna", inputPrice: 1e-6, outputPrice: 6e-6},
		{model: "gpt-5.6", inputPrice: 5e-6, outputPrice: 30e-6},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			pricing, err := svc.GetModelPricing(tt.model)
			require.NoError(t, err)
			require.InDelta(t, tt.inputPrice, pricing.InputPricePerToken, 1e-12)
			require.InDelta(t, tt.outputPrice, pricing.OutputPricePerToken, 1e-12)
		})
	}
}

func TestBillingFallbackPricing_ClaudeSonnet5SwitchesAtOfficialBoundary(t *testing.T) {
	svc := NewBillingService(&config.Config{}, nil)

	before := svc.getFallbackPricingAt("claude-sonnet-5", time.Date(2026, time.August, 31, 23, 59, 59, 0, time.UTC))
	require.NotNil(t, before)
	require.InDelta(t, 2e-6, before.InputPricePerToken, 1e-12)
	require.InDelta(t, 10e-6, before.OutputPricePerToken, 1e-12)

	after := svc.getFallbackPricingAt("claude-sonnet-5", time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC))
	require.NotNil(t, after)
	require.InDelta(t, 3e-6, after.InputPricePerToken, 1e-12)
	require.InDelta(t, 15e-6, after.OutputPricePerToken, 1e-12)
}

func TestBillingServiceGetModelPricing_Gpt53CodexSparkHasNoProxyPrice(t *testing.T) {
	svc := NewBillingService(&config.Config{}, &PricingService{pricingData: map[string]*LiteLLMModelPricing{
		"gpt-5.3-codex-spark": {InputCostPerToken: 99, OutputCostPerToken: 99},
	}})

	pricing, err := svc.GetModelPricing("gpt-5.3-codex-spark")
	require.Error(t, err)
	require.Nil(t, pricing)
	require.Contains(t, err.Error(), "pricing not found")
}

func TestBillingServiceGetModelPricing_UnknownClaudeVersionDoesNotFallBackToOlderOpus(t *testing.T) {
	svc := NewBillingService(&config.Config{}, nil)

	pricing, err := svc.GetModelPricing("claude-opus-4-9")
	require.Error(t, err)
	require.Nil(t, pricing)
	require.Contains(t, err.Error(), "pricing not found")
}

func TestBillingServiceGetModelPricing_OpenAIFallbackGpt52Variants(t *testing.T) {
	svc := newTestBillingService()

	gpt52, err := svc.GetModelPricing("gpt-5.2")
	require.NoError(t, err)
	require.NotNil(t, gpt52)
	require.InDelta(t, 1.75e-6, gpt52.InputPricePerToken, 1e-12)
	require.InDelta(t, 3.5e-6, gpt52.InputPricePerTokenPriority, 1e-12)

	gpt52Codex, err := svc.GetModelPricing("gpt-5.2-codex")
	require.NoError(t, err)
	require.NotNil(t, gpt52Codex)
	require.InDelta(t, 1.75e-6, gpt52Codex.InputPricePerToken, 1e-12)
	require.InDelta(t, 3.5e-6, gpt52Codex.InputPricePerTokenPriority, 1e-12)
	require.InDelta(t, 28e-6, gpt52Codex.OutputPricePerTokenPriority, 1e-12)
}

func TestCalculateCostWithServiceTier_PriorityFallsBackToTierMultiplierWhenExplicitPriceMissing(t *testing.T) {
	svc := NewBillingService(&config.Config{}, &PricingService{
		pricingData: map[string]*LiteLLMModelPricing{
			"custom-no-priority": {
				InputCostPerToken:           1e-6,
				OutputCostPerToken:          2e-6,
				CacheCreationInputTokenCost: 0.5e-6,
				CacheReadInputTokenCost:     0.25e-6,
			},
		},
	})
	tokens := UsageTokens{InputTokens: 100, OutputTokens: 50, CacheCreationTokens: 40, CacheReadTokens: 20}

	baseCost, err := svc.CalculateCost("custom-no-priority", tokens, 1.0)
	require.NoError(t, err)

	priorityCost, err := svc.CalculateCostWithServiceTier("custom-no-priority", tokens, 1.0, "priority")
	require.NoError(t, err)

	require.InDelta(t, baseCost.InputCost*2, priorityCost.InputCost, 1e-10)
	require.InDelta(t, baseCost.OutputCost*2, priorityCost.OutputCost, 1e-10)
	require.InDelta(t, baseCost.CacheCreationCost*2, priorityCost.CacheCreationCost, 1e-10)
	require.InDelta(t, baseCost.CacheReadCost*2, priorityCost.CacheReadCost, 1e-10)
	require.InDelta(t, baseCost.TotalCost*2, priorityCost.TotalCost, 1e-10)
}

func TestGetModelPricing_OpenAIGpt52FallbacksExposePriorityPrices(t *testing.T) {
	svc := newTestBillingService()

	gpt52, err := svc.GetModelPricing("gpt-5.2")
	require.NoError(t, err)
	require.InDelta(t, 1.75e-6, gpt52.InputPricePerToken, 1e-12)
	require.InDelta(t, 3.5e-6, gpt52.InputPricePerTokenPriority, 1e-12)
	require.InDelta(t, 14e-6, gpt52.OutputPricePerToken, 1e-12)
	require.InDelta(t, 28e-6, gpt52.OutputPricePerTokenPriority, 1e-12)

	gpt52Codex, err := svc.GetModelPricing("gpt-5.2-codex")
	require.NoError(t, err)
	require.InDelta(t, 1.75e-6, gpt52Codex.InputPricePerToken, 1e-12)
	require.InDelta(t, 3.5e-6, gpt52Codex.InputPricePerTokenPriority, 1e-12)
	require.InDelta(t, 14e-6, gpt52Codex.OutputPricePerToken, 1e-12)
	require.InDelta(t, 28e-6, gpt52Codex.OutputPricePerTokenPriority, 1e-12)
}

func TestGetModelPricing_MapsDynamicPriorityFieldsIntoBillingPricing(t *testing.T) {
	svc := NewBillingService(&config.Config{}, &PricingService{
		pricingData: map[string]*LiteLLMModelPricing{
			"dynamic-tier-model": {
				InputCostPerToken:                   1e-6,
				InputCostPerTokenPriority:           2e-6,
				OutputCostPerToken:                  3e-6,
				OutputCostPerTokenPriority:          6e-6,
				CacheCreationInputTokenCost:         4e-6,
				CacheCreationInputTokenCostAbove1hr: 5e-6,
				CacheReadInputTokenCost:             7e-7,
				CacheReadInputTokenCostPriority:     8e-7,
				LongContextInputTokenThreshold:      999,
				LongContextInputCostMultiplier:      1.5,
				LongContextOutputCostMultiplier:     1.25,
			},
		},
	})

	pricing, err := svc.GetModelPricing("dynamic-tier-model")
	require.NoError(t, err)
	require.InDelta(t, 1e-6, pricing.InputPricePerToken, 1e-12)
	require.InDelta(t, 2e-6, pricing.InputPricePerTokenPriority, 1e-12)
	require.InDelta(t, 3e-6, pricing.OutputPricePerToken, 1e-12)
	require.InDelta(t, 6e-6, pricing.OutputPricePerTokenPriority, 1e-12)
	require.InDelta(t, 4e-6, pricing.CacheCreation5mPrice, 1e-12)
	require.InDelta(t, 5e-6, pricing.CacheCreation1hPrice, 1e-12)
	require.True(t, pricing.SupportsCacheBreakdown)
	require.InDelta(t, 7e-7, pricing.CacheReadPricePerToken, 1e-12)
	require.InDelta(t, 8e-7, pricing.CacheReadPricePerTokenPriority, 1e-12)
	require.Equal(t, 999, pricing.LongContextInputThreshold)
	require.InDelta(t, 1.5, pricing.LongContextInputMultiplier, 1e-12)
	require.InDelta(t, 1.25, pricing.LongContextOutputMultiplier, 1e-12)
}
