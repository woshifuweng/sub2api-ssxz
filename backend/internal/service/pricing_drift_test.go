//go:build unit

package service

import (
	"encoding/json"
	"testing"
)

func pricingForTest(input, output float64) *LiteLLMModelPricing {
	return &LiteLLMModelPricing{
		InputCostPerToken:  input,
		OutputCostPerToken: output,
	}
}

func TestPricingDrift_NoAlert_WhenPriceUnchanged(t *testing.T) {
	previous := map[string]*LiteLLMModelPricing{"gpt-test": pricingForTest(0.001, 0.002)}
	current := map[string]*LiteLLMModelPricing{"gpt-test": pricingForTest(0.00105, 0.00209)}

	if changes := detectPricingDrift(previous, current); len(changes) != 0 {
		t.Fatalf("expected no pricing drift, got %#v", changes)
	}
}

func TestPricingDrift_Alert_WhenInputPriceChanges(t *testing.T) {
	previous := map[string]*LiteLLMModelPricing{"gpt-test": pricingForTest(0.001, 0.002)}
	current := map[string]*LiteLLMModelPricing{"gpt-test": pricingForTest(0.0012, 0.002)}

	changes := detectPricingDrift(previous, current)
	if len(changes) != 1 || changes[0].OldInput != 0.001 || changes[0].NewInput != 0.0012 {
		t.Fatalf("unexpected input drift: %#v", changes)
	}

	assertPricingDriftBody(t, pricingDriftAlertBody(changes), 0.0012, 0.002)
}

func TestPricingDrift_Alert_WhenOutputPriceChanges(t *testing.T) {
	previous := map[string]*LiteLLMModelPricing{"gpt-test": pricingForTest(0.001, 0.002)}
	current := map[string]*LiteLLMModelPricing{"gpt-test": pricingForTest(0.001, 0.0023)}

	changes := detectPricingDrift(previous, current)
	if len(changes) != 1 || changes[0].OldOutput != 0.002 || changes[0].NewOutput != 0.0023 {
		t.Fatalf("unexpected output drift: %#v", changes)
	}

	assertPricingDriftBody(t, pricingDriftAlertBody(changes), 0.001, 0.0023)
}

func assertPricingDriftBody(t *testing.T, body string, expectedInput, expectedOutput float64) {
	t.Helper()
	var payload struct {
		Event   string `json:"event"`
		Changed []struct {
			NewInput  float64 `json:"new_input"`
			NewOutput float64 `json:"new_output"`
		} `json:"changed"`
	}
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		t.Fatalf("invalid alert body: %v", err)
	}
	if payload.Event != "pricing_drift" || len(payload.Changed) != 1 {
		t.Fatalf("unexpected alert payload: %s", body)
	}
	if payload.Changed[0].NewInput != expectedInput || payload.Changed[0].NewOutput != expectedOutput {
		t.Fatalf("unexpected changed prices: %#v", payload.Changed[0])
	}
}
