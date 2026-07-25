package service

import (
	"fmt"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const (
	// Unbounded requests need a backend safety budget so low-balance users must supply an explicit token cap.
	unboundedTokenRequestSafetyOutputTokens = 500000
	unboundedTokenRequestMinimumSafetyCost  = 10.0
	// Requests that omit an output limit receive a conservative platform cap before forwarding.
	// Explicit client limits remain untouched.
	unboundedTokenRequestMaxOutputTokens = 16384
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
