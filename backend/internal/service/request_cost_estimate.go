package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const (
	// Requests that omit an output limit receive this platform cap before forwarding.
	// Explicit client limits remain untouched.
	unboundedTokenRequestMaxOutputTokens = 16384
)

// EnforceUnboundedTokenRequestLimit injects a platform cap only when clients omit
// every recognized output-token field. Explicit values, including malformed ones,
// are left to the normal validation path instead of being silently rewritten.
func EnforceUnboundedTokenRequestLimit(body []byte, targetPath string, recognizedPaths ...string) ([]byte, bool, error) {
	if !gjson.ValidBytes(body) {
		return nil, false, fmt.Errorf("invalid json")
	}
	if len(recognizedPaths) == 0 {
		recognizedPaths = []string{targetPath}
	}
	for _, path := range recognizedPaths {
		result := gjson.GetBytes(body, path)
		if !result.Exists() || result.Type == gjson.Null {
			continue
		}
		return body, false, nil
	}

	limited, err := sjson.SetBytes(body, targetPath, unboundedTokenRequestMaxOutputTokens)
	if err != nil {
		return nil, false, fmt.Errorf("set output token limit: %w", err)
	}
	return limited, true, nil
}

func (s *GatewayService) EstimateGatewayTokenRequestCost(ctx context.Context, parsed *ParsedRequest, apiKey *APIKey, user *User) (*CostBreakdown, error) {
	return s.EstimateGatewayTokenRequestCostWithLongContext(ctx, parsed, apiKey, user, 0, 0)
}

func (s *GatewayService) EstimateGatewayTokenRequestCostWithLongContext(ctx context.Context, parsed *ParsedRequest, apiKey *APIKey, user *User, longContextThreshold int, longContextMultiplier float64) (*CostBreakdown, error) {
	if s == nil || s.billingService == nil || parsed == nil || parsed.Body == nil || parsed.Body.Len() == 0 {
		return nil, nil
	}
	model := strings.TrimSpace(parsed.Model)
	if model == "" {
		return nil, nil
	}

	outputTokens := estimateGatewayRequestOutputTokens(parsed)
	if outputTokens <= 0 {
		outputTokens = unboundedTokenRequestMaxOutputTokens
	}
	defaultMultiplier := 0.0
	if s.cfg != nil {
		defaultMultiplier = s.cfg.Default.RateMultiplier
	}
	multiplier := requestEstimateRateMultiplier(ctx, defaultMultiplier, apiKey, user, s.getUserGroupRateMultiplier)
	cost, err := s.billingService.CalculateCostWithLongContext(model, UsageTokens{
		InputTokens:  estimateRequestInputTokens(parsed.Body.Bytes()),
		OutputTokens: outputTokens,
	}, multiplier, longContextThreshold, longContextMultiplier)
	if err != nil {
		return nil, err
	}
	return cost, nil
}

func (s *OpenAIGatewayService) EstimateOpenAITokenRequestCost(ctx context.Context, model string, body []byte, apiKey *APIKey, user *User) (*CostBreakdown, error) {
	if s == nil || s.billingService == nil || strings.TrimSpace(model) == "" || len(body) == 0 {
		return nil, nil
	}
	outputTokens := estimateOpenAIRequestOutputTokenLimit(body)
	if outputTokens <= 0 {
		outputTokens = unboundedTokenRequestMaxOutputTokens
	}
	defaultMultiplier := 0.0
	if s.cfg != nil {
		defaultMultiplier = s.cfg.Default.RateMultiplier
	}
	multiplier := requestEstimateRateMultiplier(ctx, defaultMultiplier, apiKey, user, func(ctx context.Context, userID, groupID int64, fallback float64) float64 {
		if s.userGroupRateResolver == nil {
			return fallback
		}
		return s.userGroupRateResolver.Resolve(ctx, userID, groupID, fallback)
	})
	cost, err := s.calculateOpenAIRecordUsageTokenCost(
		ctx,
		apiKey,
		strings.TrimSpace(model),
		multiplier,
		time.Now(),
		UsageTokens{InputTokens: estimateRequestInputTokens(body), OutputTokens: outputTokens},
		strings.TrimSpace(gjson.GetBytes(body, "service_tier").String()),
		nil,
	)
	if err != nil {
		return nil, err
	}
	return cost, nil
}

func (s *OpenAIGatewayService) EstimateOpenAIImageCost(ctx context.Context, model, imageSize string, imageCount int, apiKey *APIKey, user *User) *CostBreakdown {
	if s == nil || s.billingService == nil || strings.TrimSpace(model) == "" || imageCount <= 0 {
		return nil
	}
	defaultMultiplier := 0.0
	if s.cfg != nil {
		defaultMultiplier = s.cfg.Default.RateMultiplier
	}
	multiplier := requestEstimateRateMultiplier(ctx, defaultMultiplier, apiKey, user, func(ctx context.Context, userID, groupID int64, fallback float64) float64 {
		if s.userGroupRateResolver == nil {
			return fallback
		}
		return s.userGroupRateResolver.Resolve(ctx, userID, groupID, fallback)
	})
	return s.calculateOpenAIImageCost(ctx, strings.TrimSpace(model), apiKey, &OpenAIForwardResult{
		ImageCount: imageCount,
		ImageSize:  imageSize,
	}, multiplier)
}

func requestEstimateRateMultiplier(ctx context.Context, defaultMultiplier float64, apiKey *APIKey, user *User, resolve func(context.Context, int64, int64, float64) float64) float64 {
	multiplier := defaultMultiplier
	if apiKey != nil && apiKey.GroupID != nil && apiKey.Group != nil && user != nil && resolve != nil {
		multiplier = resolve(ctx, user.ID, *apiKey.GroupID, apiKey.Group.RateMultiplier)
	}
	if multiplier <= 0 {
		return 1
	}
	return multiplier
}

func estimateGatewayRequestOutputTokens(parsed *ParsedRequest) int {
	if parsed == nil {
		return 0
	}
	if parsed.MaxTokens > 0 {
		return parsed.MaxTokens
	}
	for _, path := range []string{
		"max_completion_tokens",
		"max_output_tokens",
		"generationConfig.maxOutputTokens",
		"generation_config.max_output_tokens",
		"maxOutputTokens",
		"output_config.max_tokens",
	} {
		if value := positiveJSONInt(parsed.Body.Bytes(), path); value > 0 {
			return value
		}
	}
	return 0
}

func estimateOpenAIRequestOutputTokenLimit(body []byte) int {
	for _, path := range []string{"max_completion_tokens", "max_output_tokens", "max_tokens", "max_tokens_to_sample", "output_config.max_tokens"} {
		if value := positiveJSONInt(body, path); value > 0 {
			return value
		}
	}
	return 0
}

func positiveJSONInt(body []byte, path string) int {
	result := gjson.GetBytes(body, path)
	if !result.Exists() || result.Type != gjson.Number || result.Int() <= 0 {
		return 0
	}
	value := result.Int()
	maxInt := int64(int(^uint(0) >> 1))
	if value > maxInt {
		return int(maxInt)
	}
	return int(value)
}

func estimateRequestInputTokens(body []byte) int {
	if len(body) == 0 {
		return 0
	}
	tokens := (len(body) + 3) / 4
	if tokens < 1 {
		return 1
	}
	return tokens
}
