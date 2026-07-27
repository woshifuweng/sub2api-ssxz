package service

import (
	"encoding/json"
	"math"
	"sort"
)

const pricingDriftThreshold = 0.0001

type pricingDriftChange struct {
	Model     string  `json:"model"`
	OldInput  float64 `json:"old_input"`
	NewInput  float64 `json:"new_input"`
	OldOutput float64 `json:"old_output"`
	NewOutput float64 `json:"new_output"`
}

func clonePricingSnapshot(snapshot map[string]*LiteLLMModelPricing) map[string]*LiteLLMModelPricing {
	if len(snapshot) == 0 {
		return nil
	}

	clone := make(map[string]*LiteLLMModelPricing, len(snapshot))
	for model, pricing := range snapshot {
		if pricing == nil {
			continue
		}
		pricingCopy := *pricing
		clone[model] = &pricingCopy
	}
	return clone
}

func detectPricingDrift(previous, current map[string]*LiteLLMModelPricing) []pricingDriftChange {
	if len(previous) == 0 || len(current) == 0 {
		return nil
	}

	models := make([]string, 0, len(previous))
	for model := range previous {
		if _, ok := current[model]; ok {
			models = append(models, model)
		}
	}
	sort.Strings(models)

	changes := make([]pricingDriftChange, 0, len(models))
	for _, model := range models {
		oldPricing := previous[model]
		newPricing := current[model]
		if oldPricing == nil || newPricing == nil {
			continue
		}
		if math.Abs(oldPricing.InputCostPerToken-newPricing.InputCostPerToken) <= pricingDriftThreshold &&
			math.Abs(oldPricing.OutputCostPerToken-newPricing.OutputCostPerToken) <= pricingDriftThreshold {
			continue
		}

		changes = append(changes, pricingDriftChange{
			Model:     model,
			OldInput:  oldPricing.InputCostPerToken,
			NewInput:  newPricing.InputCostPerToken,
			OldOutput: oldPricing.OutputCostPerToken,
			NewOutput: newPricing.OutputCostPerToken,
		})
	}
	return changes
}

func pricingDriftAlertBody(changes []pricingDriftChange) string {
	payload := struct {
		Event   string               `json:"event"`
		Changed []pricingDriftChange `json:"changed"`
	}{
		Event:   "pricing_drift",
		Changed: changes,
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return `{"event":"pricing_drift","changed":[]}`
	}
	return string(raw)
}
