package service

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestWorkspaceProviderExecutionContractBlocksMissingBillingPolicy(t *testing.T) {
	contract := ValidateWorkspaceProviderExecutionPlan(validWorkspaceProviderExecutionRequest())

	if contract.CanCallProvider {
		t.Fatalf("expected provider call to be blocked without billing and usage policy")
	}
	if !hasWorkspaceExecutionReason(contract.BlockReasons, WorkspaceProviderExecutionBlockBillingPolicy) {
		t.Fatalf("expected missing billing policy block reason, got %#v", contract.BlockReasons)
	}
	if !hasWorkspaceExecutionReason(contract.BlockReasons, WorkspaceProviderExecutionBlockUsagePolicy) {
		t.Fatalf("expected missing usage policy block reason, got %#v", contract.BlockReasons)
	}
	if contract.Audit.ProviderCalled {
		t.Fatal("blocked execution must not mark provider as called")
	}
	if contract.Audit.BillingTouched {
		t.Fatal("execution contract must not touch billing")
	}
}

func TestWorkspaceProviderExecutionContractFailsClosedWhenBillingUnknown(t *testing.T) {
	input := validWorkspaceProviderExecutionRequest()
	input.BillingPolicy = WorkspaceProviderBillingPolicyRecordUsageAfterSuccess
	input.UsagePolicy = WorkspaceProviderUsagePolicyRecordAfterSuccess
	input.BillingEligibilityKnown = false
	input.BillingEligible = false

	contract := ValidateWorkspaceProviderExecutionPlan(input)

	if contract.CanCallProvider {
		t.Fatal("expected billing unknown to fail closed before provider call")
	}
	if !hasWorkspaceExecutionReason(contract.BlockReasons, WorkspaceProviderExecutionBlockBillingUnknown) {
		t.Fatalf("expected billing unknown block reason, got %#v", contract.BlockReasons)
	}
	if contract.FailurePolicy != WorkspaceProviderFailurePolicyFailClosed {
		t.Fatalf("expected default fail-closed policy, got %q", contract.FailurePolicy)
	}
}

func TestWorkspaceProviderExecutionContractBlocksFeatureGateInvalidInputAndDisabledCapability(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*WorkspaceProviderExecutionRequest)
		reason WorkspaceProviderExecutionBlockReason
	}{
		{
			name: "feature gate disabled",
			mutate: func(input *WorkspaceProviderExecutionRequest) {
				input.FeatureGateEnabled = false
			},
			reason: WorkspaceProviderExecutionBlockFeatureGateDisabled,
		},
		{
			name: "invalid model",
			mutate: func(input *WorkspaceProviderExecutionRequest) {
				input.Model = "unknown model"
			},
			reason: WorkspaceProviderExecutionBlockInvalidModel,
		},
		{
			name: "invalid intent",
			mutate: func(input *WorkspaceProviderExecutionRequest) {
				input.Intent = "unknown_intent"
			},
			reason: WorkspaceProviderExecutionBlockInvalidIntent,
		},
		{
			name: "disabled intent",
			mutate: func(input *WorkspaceProviderExecutionRequest) {
				input.Intent = "image_generation"
			},
			reason: WorkspaceProviderExecutionBlockCapabilityDisabled,
		},
		{
			name: "disabled capability",
			mutate: func(input *WorkspaceProviderExecutionRequest) {
				input.Capability = WorkspaceProviderCapabilityImageGeneration
			},
			reason: WorkspaceProviderExecutionBlockCapabilityDisabled,
		},
		{
			name: "arbitrary base url",
			mutate: func(input *WorkspaceProviderExecutionRequest) {
				input.EndpointBaseURL = "https://provider.example.com/v1"
			},
			reason: WorkspaceProviderExecutionBlockArbitraryBaseURL,
		},
		{
			name: "unsafe inline payload",
			mutate: func(input *WorkspaceProviderExecutionRequest) {
				input.Content = "please store data:image/png;base64,AAAA"
			},
			reason: WorkspaceProviderExecutionBlockUnsafePrompt,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := validWorkspaceProviderExecutionRequest()
			input.BillingPolicy = WorkspaceProviderBillingPolicyRecordUsageAfterSuccess
			input.UsagePolicy = WorkspaceProviderUsagePolicyRecordAfterSuccess
			tc.mutate(&input)

			contract := ValidateWorkspaceProviderExecutionPlan(input)
			if contract.CanCallProvider {
				t.Fatalf("expected provider call to be blocked for %s", tc.name)
			}
			if !hasWorkspaceExecutionReason(contract.BlockReasons, tc.reason) {
				t.Fatalf("expected block reason %q, got %#v", tc.reason, contract.BlockReasons)
			}
		})
	}
}

func TestWorkspaceProviderExecutionContractAllowsUsageRecordedTextPlan(t *testing.T) {
	input := validWorkspaceProviderExecutionRequest()
	input.BillingPolicy = WorkspaceProviderBillingPolicyRecordUsageOnProviderUsage
	input.UsagePolicy = WorkspaceProviderUsagePolicyRecordProviderReported
	input.Diagnostics.UpstreamModel = "gpt-5.5-upstream"
	input.Diagnostics.ProviderName = "openai-compatible"
	input.EndpointLabel = "/workspace-provider-text"

	contract := ValidateWorkspaceProviderExecutionPlan(input)

	if !contract.CanCallProvider {
		t.Fatalf("expected provider call to be allowed, reasons: %#v", contract.BlockReasons)
	}
	if contract.Decision != WorkspaceProviderExecutionDecisionAllow {
		t.Fatalf("expected allow decision, got %q", contract.Decision)
	}
	if contract.Diagnostics.RequestedModel != "gpt-5.5" {
		t.Fatalf("unexpected requested model %q", contract.Diagnostics.RequestedModel)
	}
	if contract.Diagnostics.UpstreamModel != "gpt-5.5-upstream" {
		t.Fatalf("unexpected upstream model %q", contract.Diagnostics.UpstreamModel)
	}
	if !contract.Audit.ProviderCalled {
		t.Fatal("allowed contract should mark provider execution as planned")
	}
	if contract.Audit.BillingTouched {
		t.Fatal("contract planning must not touch billing")
	}
	if contract.Audit.Record.Status != "planned" {
		t.Fatalf("expected planned audit status, got %q", contract.Audit.Record.Status)
	}
}

func TestWorkspaceProviderExecutionContractBlocksNoFreeProviderCall(t *testing.T) {
	input := validWorkspaceProviderExecutionRequest()
	input.BillingPolicy = WorkspaceProviderBillingPolicyNoBilling
	input.UsagePolicy = WorkspaceProviderUsagePolicyAuditOnly

	contract := ValidateWorkspaceProviderExecutionPlan(input)

	if contract.CanCallProvider {
		t.Fatal("expected no-billing plan to block real provider execution")
	}
	if !hasWorkspaceExecutionReason(contract.BlockReasons, WorkspaceProviderExecutionBlockNoFreeProviderCall) {
		t.Fatalf("expected no-free-provider-call block reason, got %#v", contract.BlockReasons)
	}
}

func TestWorkspaceProviderExecutionAuditRedactsSecretsAndPrompt(t *testing.T) {
	input := validWorkspaceProviderExecutionRequest()
	input.BillingPolicy = WorkspaceProviderBillingPolicyRecordUsageAfterSuccess
	input.UsagePolicy = WorkspaceProviderUsagePolicyRecordAfterSuccess
	input.Content = "Create a campaign. Authorization: Bearer sk-secret-token cookie=session-secret access_token=abc123 private customer notes should not be stored fully."
	input.Diagnostics.ProviderName = "provider-with-secret-safe-label"
	input.EndpointLabel = "/workspace-provider-text"

	contract := ValidateWorkspaceProviderExecutionPlan(input)
	payload, err := json.Marshal(contract)
	if err != nil {
		t.Fatalf("marshal contract: %v", err)
	}
	serialized := string(payload)
	for _, forbidden := range []string{"sk-secret-token", "session-secret", "abc123", input.Content} {
		if strings.Contains(serialized, forbidden) {
			t.Fatalf("contract leaked forbidden value %q in %s", forbidden, serialized)
		}
	}
	if contract.Audit.Record.PromptHash == "" {
		t.Fatal("expected prompt hash")
	}
	if !strings.Contains(contract.Audit.Record.PromptPreview, "[REDACTED]") {
		t.Fatalf("expected redacted prompt preview, got %q", contract.Audit.Record.PromptPreview)
	}
}

func TestWorkspaceProviderExecutionContractMarksReconcileFailurePolicy(t *testing.T) {
	input := validWorkspaceProviderExecutionRequest()
	input.BillingPolicy = WorkspaceProviderBillingPolicyRecordUsageAfterSuccess
	input.UsagePolicy = WorkspaceProviderUsagePolicyRecordAfterSuccess
	input.FailurePolicy = WorkspaceProviderFailurePolicyReconcileRequired

	contract := ValidateWorkspaceProviderExecutionPlan(input)

	if !contract.Diagnostics.ReconcileNeeded {
		t.Fatal("expected reconcile-needed diagnostic")
	}
	if !contract.Audit.RequiresReconciliation {
		t.Fatal("expected audit to require reconciliation")
	}
	if !containsWorkspaceExecutionSemantic(contract.FailureSemantics, "reconciliation") {
		t.Fatalf("expected reconciliation failure semantic, got %#v", contract.FailureSemantics)
	}
}

func validWorkspaceProviderExecutionRequest() WorkspaceProviderExecutionRequest {
	return WorkspaceProviderExecutionRequest{
		RequestID:               "req_test_123",
		FeatureGateEnabled:      true,
		UserID:                  1001,
		ConversationID:          2002,
		UserMessageID:           3003,
		Content:                 "Hello, summarize this workspace conversation.",
		Model:                   "gpt-5.5",
		Intent:                  WorkspaceIntentChat,
		Capability:              WorkspaceProviderCapabilityText,
		ProviderAvailable:       true,
		BillingEligibilityKnown: true,
		BillingEligible:         true,
		Diagnostics: WorkspaceProviderDiagnostics{
			RequestedModel: "gpt-5.5",
			MappedModel:    "gpt-5.5",
			ProviderName:   "workspace-text-provider",
		},
		EndpointLabel: "/workspace-provider-text",
	}
}

func hasWorkspaceExecutionReason(reasons []WorkspaceProviderExecutionBlockReason, expected WorkspaceProviderExecutionBlockReason) bool {
	for _, reason := range reasons {
		if reason == expected {
			return true
		}
	}
	return false
}

func containsWorkspaceExecutionSemantic(values []string, fragment string) bool {
	for _, value := range values {
		if strings.Contains(value, fragment) {
			return true
		}
	}
	return false
}
