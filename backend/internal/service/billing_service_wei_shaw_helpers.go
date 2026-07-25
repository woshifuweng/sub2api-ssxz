package service

import "strings"

const (
	defaultWebSearchPricePerCall        = 0.01
	defaultGrokImagineVideoPrice480P    = 0.05
	defaultGrokImagineVideoPrice720P    = 0.07
	defaultGrokImagineVideo15Price480P  = 0.08
	defaultGrokImagineVideo15Price720P  = 0.14
	defaultGrokImagineVideo15Price1080P = 0.25
)

// CalculateWebSearchCost calculates Codex alpha/search per-request billing.
func (s *BillingService) CalculateWebSearchCost(callCount int, groupPrice *float64, rateMultiplier float64) *CostBreakdown {
	if callCount <= 0 {
		return &CostBreakdown{}
	}
	unitPrice := defaultWebSearchPricePerCall
	if groupPrice != nil && *groupPrice >= 0 {
		unitPrice = *groupPrice
	}
	totalCost := unitPrice * float64(callCount)
	if rateMultiplier < 0 {
		rateMultiplier = 0
	}
	return &CostBreakdown{
		TotalCost:   totalCost,
		ActualCost:  totalCost * rateMultiplier,
		BillingMode: string(BillingModePerRequest),
	}
}

func (s *BillingService) calculateCostWithServiceTierPolicy(model string, tokens UsageTokens, rateMultiplier float64, serviceTier string, longContextBillingEnabled bool) (*CostBreakdown, error) {
	pricing, err := s.GetModelPricing(model)
	if err != nil {
		return nil, err
	}
	if !longContextBillingEnabled && pricing != nil {
		cloned := *pricing
		cloned.LongContextInputThreshold = 0
		pricing = &cloned
	}
	return CalculateCostFromModelPricing(pricing, tokens, rateMultiplier, serviceTier)
}

func (s *BillingService) CalculateVideoCost(model, resolution string, videoCount, durationSeconds int, groupConfig *VideoPriceConfig, rateMultiplier float64) *CostBreakdown {
	if videoCount <= 0 {
		return &CostBreakdown{}
	}
	resolution = NormalizeVideoBillingResolutionOrDefault(resolution)
	durationSeconds = NormalizeVideoBillingDurationSecondsOrDefault(durationSeconds)
	perSecondPrice := s.getVideoUnitPrice(model, resolution, groupConfig)
	totalCost := perSecondPrice * float64(durationSeconds) * float64(videoCount)
	if rateMultiplier < 0 {
		rateMultiplier = 0
	}
	return &CostBreakdown{
		TotalCost:   totalCost,
		ActualCost:  totalCost * rateMultiplier,
		BillingMode: string(BillingModeVideo),
	}
}

func (s *BillingService) getVideoUnitPrice(model, resolution string, groupConfig *VideoPriceConfig) float64 {
	if groupConfig != nil {
		switch resolution {
		case VideoBillingResolution480P:
			if groupConfig.Price480P != nil {
				return *groupConfig.Price480P
			}
		case VideoBillingResolution720P:
			if groupConfig.Price720P != nil {
				return *groupConfig.Price720P
			}
		case VideoBillingResolution1080P:
			if groupConfig.Price1080P != nil {
				return *groupConfig.Price1080P
			}
		}
	}
	if price, ok := getDefaultGrokImagineVideoPrice(model, resolution); ok {
		return price
	}
	return s.getDefaultImagePrice(model, ImageBillingSize2K)
}

func getDefaultGrokImagineVideoPrice(model, resolution string) (float64, bool) {
	model = strings.ToLower(strings.TrimSpace(model))
	switch {
	case strings.HasPrefix(model, "grok-imagine-video-1.5"):
		switch NormalizeVideoBillingResolutionOrDefault(resolution) {
		case VideoBillingResolution480P:
			return defaultGrokImagineVideo15Price480P, true
		case VideoBillingResolution720P:
			return defaultGrokImagineVideo15Price720P, true
		case VideoBillingResolution1080P:
			return defaultGrokImagineVideo15Price1080P, true
		}
	case strings.HasPrefix(model, "grok-imagine-video"):
		switch NormalizeVideoBillingResolutionOrDefault(resolution) {
		case VideoBillingResolution480P:
			return defaultGrokImagineVideoPrice480P, true
		case VideoBillingResolution720P, VideoBillingResolution1080P:
			return defaultGrokImagineVideoPrice720P, true
		}
	}
	return 0, false
}
