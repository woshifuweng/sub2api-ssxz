package service

import (
	"fmt"
	"strings"
)

// SoraPriceConfig contains the optional per-request Sora prices configured by a group.
type SoraPriceConfig struct {
	ImagePrice360          *float64
	ImagePrice540          *float64
	VideoPricePerRequest   *float64
	VideoPricePerRequestHD *float64
}

func (s *BillingService) CalculateSoraImageCost(imageSize string, imageCount int, groupConfig *SoraPriceConfig, rateMultiplier float64) *CostBreakdown {
	if imageCount <= 0 {
		return &CostBreakdown{}
	}

	unitPrice := 0.0
	if groupConfig != nil {
		switch imageSize {
		case "540":
			if groupConfig.ImagePrice540 != nil {
				unitPrice = *groupConfig.ImagePrice540
			}
		default:
			if groupConfig.ImagePrice360 != nil {
				unitPrice = *groupConfig.ImagePrice360
			}
		}
	}

	totalCost := unitPrice * float64(imageCount)
	if rateMultiplier <= 0 {
		rateMultiplier = 1.0
	}
	return &CostBreakdown{TotalCost: totalCost, ActualCost: totalCost * rateMultiplier}
}

func (s *BillingService) CalculateSoraVideoCost(model string, groupConfig *SoraPriceConfig, rateMultiplier float64) *CostBreakdown {
	return s.CalculateSoraVideoCostForCount(model, 1, groupConfig, rateMultiplier)
}

func (s *BillingService) CalculateSoraVideoCostForCount(model string, videoCount int, groupConfig *SoraPriceConfig, rateMultiplier float64) *CostBreakdown {
	if videoCount <= 0 {
		videoCount = 1
	}
	if videoCount > 3 {
		videoCount = 3
	}

	unitPrice := 0.0
	if groupConfig != nil {
		if strings.Contains(strings.ToLower(model), "sora2pro-hd") && groupConfig.VideoPricePerRequestHD != nil {
			unitPrice = *groupConfig.VideoPricePerRequestHD
		}
		if unitPrice <= 0 && groupConfig.VideoPricePerRequest != nil {
			unitPrice = *groupConfig.VideoPricePerRequest
		}
	}

	totalCost := unitPrice * float64(videoCount)
	if rateMultiplier <= 0 {
		rateMultiplier = 1.0
	}
	return &CostBreakdown{TotalCost: totalCost, ActualCost: totalCost * rateMultiplier}
}

// CalculateCostFromModelPricing calculates token costs from an already resolved pricing record.
func CalculateCostFromModelPricing(pricing *ModelPricing, tokens UsageTokens, rateMultiplier float64, serviceTier string) (*CostBreakdown, error) {
	if pricing == nil {
		return nil, fmt.Errorf("pricing is nil")
	}

	breakdown := &CostBreakdown{}
	inputPricePerToken := pricing.InputPricePerToken
	outputPricePerToken := pricing.OutputPricePerToken
	cacheReadPricePerToken := pricing.CacheReadPricePerToken
	cacheCreationPricePerToken := pricing.CacheCreationPricePerToken
	cacheCreation5mPrice := pricing.CacheCreation5mPrice
	cacheCreation1hPrice := pricing.CacheCreation1hPrice
	tierMultiplier := 1.0
	if usePriorityServiceTierPricing(serviceTier, pricing) {
		if pricing.InputPricePerTokenPriority > 0 {
			inputPricePerToken = pricing.InputPricePerTokenPriority
		}
		if pricing.OutputPricePerTokenPriority > 0 {
			outputPricePerToken = pricing.OutputPricePerTokenPriority
		}
		if pricing.CacheReadPricePerTokenPriority > 0 {
			cacheReadPricePerToken = pricing.CacheReadPricePerTokenPriority
		}
	} else {
		tierMultiplier = serviceTierCostMultiplier(serviceTier)
	}
	if pricing.LongContextInputThreshold > 0 &&
		(pricing.LongContextInputMultiplier > 1 || pricing.LongContextOutputMultiplier > 1) &&
		tokens.InputTokens+tokens.CacheCreationTokens+tokens.CacheReadTokens > pricing.LongContextInputThreshold {
		inputPricePerToken *= pricing.LongContextInputMultiplier
		outputPricePerToken *= pricing.LongContextOutputMultiplier
		cacheReadPricePerToken *= pricing.LongContextInputMultiplier
		cacheCreationPricePerToken *= pricing.LongContextInputMultiplier
		cacheCreation5mPrice *= pricing.LongContextInputMultiplier
		cacheCreation1hPrice *= pricing.LongContextInputMultiplier
	}

	breakdown.InputCost = float64(tokens.InputTokens) * inputPricePerToken
	breakdown.OutputCost = float64(tokens.OutputTokens) * outputPricePerToken
	if pricing.SupportsCacheBreakdown && (pricing.CacheCreation5mPrice > 0 || pricing.CacheCreation1hPrice > 0) {
		if tokens.CacheCreation5mTokens == 0 && tokens.CacheCreation1hTokens == 0 && tokens.CacheCreationTokens > 0 {
			breakdown.CacheCreationCost = float64(tokens.CacheCreationTokens) * cacheCreation5mPrice
		} else {
			breakdown.CacheCreationCost = float64(tokens.CacheCreation5mTokens)*cacheCreation5mPrice +
				float64(tokens.CacheCreation1hTokens)*cacheCreation1hPrice
		}
	} else {
		breakdown.CacheCreationCost = float64(tokens.CacheCreationTokens) * cacheCreationPricePerToken
	}
	breakdown.CacheReadCost = float64(tokens.CacheReadTokens) * cacheReadPricePerToken

	if tierMultiplier != 1.0 {
		breakdown.InputCost *= tierMultiplier
		breakdown.OutputCost *= tierMultiplier
		breakdown.CacheCreationCost *= tierMultiplier
		breakdown.CacheReadCost *= tierMultiplier
	}
	breakdown.TotalCost = breakdown.InputCost + breakdown.OutputCost + breakdown.CacheCreationCost + breakdown.CacheReadCost
	if rateMultiplier <= 0 {
		rateMultiplier = 1.0
	}
	breakdown.ActualCost = breakdown.TotalCost * rateMultiplier
	return breakdown, nil
}
