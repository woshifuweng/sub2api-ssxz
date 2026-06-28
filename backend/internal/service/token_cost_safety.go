package service

const (
	// Unbounded requests need a backend safety budget so low-balance users must supply an explicit token cap.
	unboundedTokenRequestSafetyOutputTokens = 500000
	unboundedTokenRequestMinimumSafetyCost  = 10.0
)

func applyUnboundedTokenRequestSafetyFloor(cost *CostBreakdown) *CostBreakdown {
	if cost == nil {
		return nil
	}
	if cost.TotalCost < unboundedTokenRequestMinimumSafetyCost {
		cost.TotalCost = unboundedTokenRequestMinimumSafetyCost
	}
	if cost.ActualCost < unboundedTokenRequestMinimumSafetyCost {
		cost.ActualCost = unboundedTokenRequestMinimumSafetyCost
	}
	return cost
}
