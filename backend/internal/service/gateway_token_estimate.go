package service

import (
	"context"
	"strings"

	"github.com/tidwall/gjson"
)

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
	usesUnboundedSafetyBudget := outputTokens <= 0
	if usesUnboundedSafetyBudget {
		outputTokens = unboundedTokenRequestSafetyOutputTokens
	}

	multiplier := 0.0
	if s.cfg != nil {
		multiplier = s.cfg.Default.RateMultiplier
	}
	if apiKey != nil && apiKey.GroupID != nil && apiKey.Group != nil && user != nil {
		multiplier = s.getUserGroupRateMultiplier(ctx, user.ID, *apiKey.GroupID, apiKey.Group.RateMultiplier)
	}

	cost, err := s.billingService.CalculateCostWithLongContext(model, UsageTokens{
		InputTokens:  estimateGatewayRequestInputTokens(parsed.Body.Bytes()),
		OutputTokens: outputTokens,
	}, multiplier, longContextThreshold, longContextMultiplier)
	if err != nil {
		return nil, err
	}
	if usesUnboundedSafetyBudget {
		return applyUnboundedTokenRequestSafetyFloor(cost), nil
	}
	return cost, nil
}

func estimateGatewayRequestOutputTokens(parsed *ParsedRequest) int {
	if parsed == nil {
		return 0
	}
	if parsed.MaxTokens > 0 {
		return parsed.MaxTokens
	}
	for _, path := range []string{"generationConfig.maxOutputTokens", "generation_config.max_output_tokens", "maxOutputTokens"} {
		result := gjson.GetBytes(parsed.Body.Bytes(), path)
		if !result.Exists() || result.Type != gjson.Number {
			continue
		}
		value := result.Int()
		if value <= 0 {
			continue
		}
		maxInt := int64(int(^uint(0) >> 1))
		if value > maxInt {
			return int(maxInt)
		}
		return int(value)
	}
	return 0
}

func estimateGatewayRequestInputTokens(body []byte) int {
	if len(body) == 0 {
		return 0
	}
	tokens := (len(body) + 3) / 4
	if tokens < 1 {
		return 1
	}
	return tokens
}
