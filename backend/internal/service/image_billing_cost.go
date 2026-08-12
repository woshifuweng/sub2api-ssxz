package service

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"go.uber.org/zap"
)

const (
	imageBillingSourceGroupExplicit = "group_explicit"
	imageBillingSourceChannel       = "channel"
	imageBillingSourceFallback      = "fallback"
)

// calculateImageCostForAPIKey keeps explicit group image prices independent
// from the text-model multiplier while preserving multiplier behavior for fallback prices.
func calculateImageCostForAPIKey(
	billingService *BillingService,
	model string,
	imageSize string,
	imageCount int,
	apiKey *APIKey,
	fallbackMultiplier float64,
	quality string,
) *CostBreakdown {
	sizeTier := normalizeImageBillingTierForModel(model, imageSize)
	groupConfig := imagePriceConfigFromAPIKey(apiKey)
	multiplier := fallbackMultiplier
	billingSource := imageBillingSourceFallback
	if apiKeyHasConfiguredImagePrice(apiKey, sizeTier) {
		multiplier = 1
		billingSource = imageBillingSourceGroupExplicit
	}

	cost := billingService.CalculateImageCost(model, sizeTier, imageCount, groupConfig, multiplier, quality)
	logResolvedImageBillingCost(cost, model, sizeTier, imageCount, quality, apiKey, billingSource)
	return cost
}

func logResolvedImageBillingCost(
	cost *CostBreakdown,
	model string,
	sizeTier string,
	imageCount int,
	quality string,
	apiKey *APIKey,
	billingSource string,
) {
	unitPrice := 0.0
	qualityMultiplier := 1.0
	effectiveMultiplier := 0.0
	total := 0.0
	if cost != nil {
		unitPrice = cost.ImageUnitPrice
		qualityMultiplier = cost.ImageQualityMultiplier
		effectiveMultiplier = cost.ImageEffectiveMultiplier
		total = cost.ActualCost
	}

	fields := []zap.Field{
		zap.String("model", model),
		zap.String("size_tier", sizeTier),
		zap.String("quality", quality),
		zap.Int("image_count", imageCount),
		zap.Float64("unit_price", unitPrice),
		zap.Float64("quality_multiplier", qualityMultiplier),
		zap.Float64("effective_multiplier", effectiveMultiplier),
		zap.String("billing_source", billingSource),
		zap.Float64("total", total),
	}
	if apiKey != nil && apiKey.Group != nil {
		fields = append(fields, zap.Int64("group_id", apiKey.Group.ID))
	}
	logger.L().Info("image_billing.cost_resolved", fields...)

	if floor, ok := knownGPTImage2UpstreamFloor(model, sizeTier); ok && unitPrice*qualityMultiplier < floor {
		logger.L().Warn("image_billing.explicit_price_below_upstream_floor",
			append(fields, zap.Float64("upstream_floor", floor))...,
		)
	}
}

func normalizeImageBillingTierForModel(model string, imageSize string) string {
	if strings.EqualFold(strings.TrimSpace(model), "gpt-image-2") &&
		strings.EqualFold(strings.TrimSpace(imageSize), "auto") {
		return ImageBillingSize4K
	}
	return NormalizeImageBillingTierOrDefault(imageSize)
}

func imageQualityMultiplier(quality string) float64 {
	switch strings.ToLower(strings.TrimSpace(quality)) {
	case "high":
		return 2.0
	case "auto", "medium":
		return 1.3
	case "low":
		return 1.0
	default:
		return 1.0
	}
}

func knownGPTImage2UpstreamFloor(model string, sizeTier string) (float64, bool) {
	if !strings.EqualFold(strings.TrimSpace(model), "gpt-image-2") {
		return 0, false
	}
	switch NormalizeImageBillingTierOrDefault(sizeTier) {
	case ImageBillingSize1K:
		return 0.03, true
	case ImageBillingSize2K, ImageBillingSize4K:
		return 0.10, true
	default:
		return 0, false
	}
}
