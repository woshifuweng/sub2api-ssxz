package service

import (
	"strings"
	"time"
)

const workspaceProviderExecutionRoute = "/api/v1/chat-workspace/conversations/:id/messages"

type WorkspaceProviderBillingPolicy string

const (
	WorkspaceProviderBillingPolicyNoBilling                  WorkspaceProviderBillingPolicy = "no_billing"
	WorkspaceProviderBillingPolicyPrecheckOnly               WorkspaceProviderBillingPolicy = "precheck_only"
	WorkspaceProviderBillingPolicyRecordUsageAfterSuccess    WorkspaceProviderBillingPolicy = "record_usage_after_success"
	WorkspaceProviderBillingPolicyRecordUsageOnProviderUsage WorkspaceProviderBillingPolicy = "record_usage_on_provider_reported_usage"
)

type WorkspaceProviderUsagePolicy string

const (
	WorkspaceProviderUsagePolicyAuditOnly              WorkspaceProviderUsagePolicy = "audit_only"
	WorkspaceProviderUsagePolicyRecordAfterSuccess     WorkspaceProviderUsagePolicy = "record_after_success"
	WorkspaceProviderUsagePolicyRecordProviderReported WorkspaceProviderUsagePolicy = "record_provider_reported"
)

type WorkspaceProviderFailurePolicy string

const (
	WorkspaceProviderFailurePolicyFailClosed        WorkspaceProviderFailurePolicy = "fail_closed"
	WorkspaceProviderFailurePolicyNoChargeOnFailure WorkspaceProviderFailurePolicy = "provider_failure_no_charge"
	WorkspaceProviderFailurePolicyReconcileRequired WorkspaceProviderFailurePolicy = "reconcile_required"
)

type WorkspaceProviderExecutionDecision string

const (
	WorkspaceProviderExecutionDecisionAllow WorkspaceProviderExecutionDecision = "allow"
	WorkspaceProviderExecutionDecisionDeny  WorkspaceProviderExecutionDecision = "deny"
)

type WorkspaceProviderExecutionBlockReason string

const (
	WorkspaceProviderExecutionBlockFeatureGateDisabled WorkspaceProviderExecutionBlockReason = "feature_gate_disabled"
	WorkspaceProviderExecutionBlockInvalidContext      WorkspaceProviderExecutionBlockReason = "invalid_workspace_context"
	WorkspaceProviderExecutionBlockInvalidModel        WorkspaceProviderExecutionBlockReason = "invalid_model"
	WorkspaceProviderExecutionBlockInvalidIntent       WorkspaceProviderExecutionBlockReason = "invalid_intent"
	WorkspaceProviderExecutionBlockCapabilityDisabled  WorkspaceProviderExecutionBlockReason = "capability_disabled"
	WorkspaceProviderExecutionBlockUnsafePrompt        WorkspaceProviderExecutionBlockReason = "unsafe_prompt"
	WorkspaceProviderExecutionBlockArbitraryBaseURL    WorkspaceProviderExecutionBlockReason = "arbitrary_base_url"
	WorkspaceProviderExecutionBlockBillingPolicy       WorkspaceProviderExecutionBlockReason = "billing_policy_missing"
	WorkspaceProviderExecutionBlockUsagePolicy         WorkspaceProviderExecutionBlockReason = "usage_policy_missing"
	WorkspaceProviderExecutionBlockBillingUnknown      WorkspaceProviderExecutionBlockReason = "billing_unknown"
	WorkspaceProviderExecutionBlockBillingIneligible   WorkspaceProviderExecutionBlockReason = "billing_not_eligible"
	WorkspaceProviderExecutionBlockNoFreeProviderCall  WorkspaceProviderExecutionBlockReason = "no_free_provider_call"
	WorkspaceProviderExecutionBlockProviderUnavailable WorkspaceProviderExecutionBlockReason = "provider_unavailable"
)

type WorkspaceProviderExecutionRequest struct {
	RequestID               string
	FeatureGateEnabled      bool
	UserID                  int64
	ConversationID          int64
	UserMessageID           int64
	Content                 string
	Model                   string
	Intent                  string
	Capability              WorkspaceProviderCapability
	ProviderAvailable       bool
	BillingEligibilityKnown bool
	BillingEligible         bool
	BillingPolicy           WorkspaceProviderBillingPolicy
	UsagePolicy             WorkspaceProviderUsagePolicy
	FailurePolicy           WorkspaceProviderFailurePolicy
	Diagnostics             WorkspaceProviderDiagnostics
	EndpointLabel           string
	EndpointBaseURL         string
	PromptEnhancement       *UpstreamPromptEnhancementResult
	CreatedAt               time.Time
}

type WorkspaceProviderExecutionContract struct {
	Decision         WorkspaceProviderExecutionDecision      `json:"decision"`
	CanCallProvider  bool                                    `json:"can_call_provider"`
	BlockReasons     []WorkspaceProviderExecutionBlockReason `json:"block_reasons,omitempty"`
	BillingPolicy    WorkspaceProviderBillingPolicy          `json:"billing_policy,omitempty"`
	UsagePolicy      WorkspaceProviderUsagePolicy            `json:"usage_policy,omitempty"`
	FailurePolicy    WorkspaceProviderFailurePolicy          `json:"failure_policy,omitempty"`
	RequiredSteps    []string                                `json:"required_steps,omitempty"`
	FailureSemantics []string                                `json:"failure_semantics,omitempty"`
	Diagnostics      WorkspaceProviderExecutionDiagnostics   `json:"diagnostics"`
	Audit            WorkspaceProviderExecutionAudit         `json:"audit"`
}

type WorkspaceProviderExecutionDiagnostics struct {
	RequestID               string                                  `json:"request_id,omitempty"`
	FeatureGateEnabled      bool                                    `json:"feature_gate_enabled"`
	BillingEligibilityKnown bool                                    `json:"billing_eligibility_known"`
	BillingEligible         bool                                    `json:"billing_eligible"`
	ProviderAvailable       bool                                    `json:"provider_available"`
	RequestedModel          string                                  `json:"requested_model,omitempty"`
	MappedModel             string                                  `json:"mapped_model,omitempty"`
	UpstreamModel           string                                  `json:"upstream_model,omitempty"`
	ProviderName            string                                  `json:"provider_name,omitempty"`
	EndpointLabel           string                                  `json:"endpoint_label,omitempty"`
	Capability              WorkspaceProviderCapability             `json:"capability,omitempty"`
	PromptEnhancerUsed      bool                                    `json:"prompt_enhancer_used"`
	ReconcileNeeded         bool                                    `json:"reconcile_needed"`
	BlockReasons            []WorkspaceProviderExecutionBlockReason `json:"block_reasons,omitempty"`
}

type WorkspaceProviderExecutionAudit struct {
	Record                 UpstreamQualityAuditRecord              `json:"record"`
	BillingPolicy          WorkspaceProviderBillingPolicy          `json:"billing_policy,omitempty"`
	UsagePolicy            WorkspaceProviderUsagePolicy            `json:"usage_policy,omitempty"`
	FailurePolicy          WorkspaceProviderFailurePolicy          `json:"failure_policy,omitempty"`
	BlockReasons           []WorkspaceProviderExecutionBlockReason `json:"block_reasons,omitempty"`
	ProviderCalled         bool                                    `json:"provider_called"`
	BillingTouched         bool                                    `json:"billing_touched"`
	RequiresReconciliation bool                                    `json:"requires_reconciliation"`
}

func ValidateWorkspaceProviderExecutionPlan(input WorkspaceProviderExecutionRequest) WorkspaceProviderExecutionContract {
	billingPolicy := input.BillingPolicy
	usagePolicy := input.UsagePolicy
	failurePolicy := normalizeWorkspaceProviderFailurePolicy(input.FailurePolicy)
	reasons := workspaceProviderExecutionBlockReasons(input, billingPolicy, usagePolicy)
	decision := WorkspaceProviderExecutionDecisionDeny
	canCallProvider := false
	if len(reasons) == 0 {
		decision = WorkspaceProviderExecutionDecisionAllow
		canCallProvider = true
	}

	diagnostics := BuildWorkspaceProviderExecutionDiagnostics(input, reasons, failurePolicy)
	audit := BuildWorkspaceProviderExecutionAudit(input, decision, canCallProvider, reasons, billingPolicy, usagePolicy, failurePolicy, diagnostics)
	return WorkspaceProviderExecutionContract{
		Decision:         decision,
		CanCallProvider:  canCallProvider,
		BlockReasons:     reasons,
		BillingPolicy:    billingPolicy,
		UsagePolicy:      usagePolicy,
		FailurePolicy:    failurePolicy,
		RequiredSteps:    workspaceProviderExecutionRequiredSteps(),
		FailureSemantics: workspaceProviderExecutionFailureSemantics(failurePolicy),
		Diagnostics:      diagnostics,
		Audit:            audit,
	}
}

func BuildWorkspaceProviderExecutionDiagnostics(
	input WorkspaceProviderExecutionRequest,
	reasons []WorkspaceProviderExecutionBlockReason,
	failurePolicy WorkspaceProviderFailurePolicy,
) WorkspaceProviderExecutionDiagnostics {
	diagnostics := input.Diagnostics
	endpointLabel := firstNonEmptyWorkspaceValue(input.EndpointLabel, diagnostics.ProviderName)
	out := WorkspaceProviderExecutionDiagnostics{
		RequestID:               strings.TrimSpace(input.RequestID),
		FeatureGateEnabled:      input.FeatureGateEnabled,
		BillingEligibilityKnown: input.BillingEligibilityKnown,
		BillingEligible:         input.BillingEligible,
		ProviderAvailable:       input.ProviderAvailable,
		RequestedModel:          firstNonEmptyWorkspaceValue(diagnostics.RequestedModel, input.Model),
		MappedModel:             firstNonEmptyWorkspaceValue(diagnostics.MappedModel, input.Model),
		UpstreamModel:           diagnostics.UpstreamModel,
		ProviderName:            diagnostics.ProviderName,
		EndpointLabel:           endpointLabel,
		Capability:              normalizeWorkspaceProviderCapability(input.Capability),
		PromptEnhancerUsed:      input.PromptEnhancement != nil || diagnostics.PromptEnhancerUsed,
		ReconcileNeeded:         failurePolicy == WorkspaceProviderFailurePolicyReconcileRequired,
		BlockReasons:            reasons,
	}
	return out
}

func BuildWorkspaceProviderExecutionAudit(
	input WorkspaceProviderExecutionRequest,
	decision WorkspaceProviderExecutionDecision,
	providerCalled bool,
	reasons []WorkspaceProviderExecutionBlockReason,
	billingPolicy WorkspaceProviderBillingPolicy,
	usagePolicy WorkspaceProviderUsagePolicy,
	failurePolicy WorkspaceProviderFailurePolicy,
	diagnostics WorkspaceProviderExecutionDiagnostics,
) WorkspaceProviderExecutionAudit {
	status := string(decision)
	errorCode := workspaceProviderExecutionErrorCode(reasons)
	if providerCalled {
		status = "planned"
	}
	endpoint := diagnostics.EndpointLabel
	if endpoint == "" {
		endpoint = "/workspace-provider-execution-contract"
	}
	record := BuildUpstreamQualityAuditRecord(UpstreamQualityAuditInput{
		RequestID:      input.RequestID,
		Route:          workspaceProviderExecutionRoute,
		Operation:      UpstreamQualityOperationTextCompletion,
		RequestedModel: diagnostics.RequestedModel,
		MappedModel:    diagnostics.MappedModel,
		UpstreamModel:  diagnostics.UpstreamModel,
		ProviderName:   diagnostics.ProviderName,
		Endpoint:       endpoint,
		ServiceTier:    diagnostics.EndpointLabel,
		Status:         status,
		ErrorCode:      errorCode,
		Prompt:         input.Content,
		PromptEnhanced: diagnostics.PromptEnhancerUsed,
		CreatedAt:      input.CreatedAt,
	})
	return WorkspaceProviderExecutionAudit{
		Record:                 record,
		BillingPolicy:          billingPolicy,
		UsagePolicy:            usagePolicy,
		FailurePolicy:          failurePolicy,
		BlockReasons:           reasons,
		ProviderCalled:         providerCalled,
		BillingTouched:         false,
		RequiresReconciliation: failurePolicy == WorkspaceProviderFailurePolicyReconcileRequired,
	}
}

func workspaceProviderExecutionBlockReasons(
	input WorkspaceProviderExecutionRequest,
	billingPolicy WorkspaceProviderBillingPolicy,
	usagePolicy WorkspaceProviderUsagePolicy,
) []WorkspaceProviderExecutionBlockReason {
	reasons := make([]WorkspaceProviderExecutionBlockReason, 0)
	if !input.FeatureGateEnabled {
		reasons = append(reasons, WorkspaceProviderExecutionBlockFeatureGateDisabled)
	}
	if input.UserID <= 0 || input.ConversationID <= 0 || input.UserMessageID <= 0 {
		reasons = append(reasons, WorkspaceProviderExecutionBlockInvalidContext)
	}
	if !isAllowedWorkspaceModel(input.Model) {
		reasons = append(reasons, WorkspaceProviderExecutionBlockInvalidModel)
	}
	intent := normalizeWorkspaceIntent(input.Intent)
	if intent != WorkspaceIntentChat {
		if isDisabledWorkspaceIntent(intent) {
			reasons = append(reasons, WorkspaceProviderExecutionBlockCapabilityDisabled)
		} else {
			reasons = append(reasons, WorkspaceProviderExecutionBlockInvalidIntent)
		}
	}
	if capability := normalizeWorkspaceProviderCapability(input.Capability); capability != WorkspaceProviderCapabilityText {
		reasons = append(reasons, WorkspaceProviderExecutionBlockCapabilityDisabled)
	}
	if containsUnsafeInlinePayload(input.Content) {
		reasons = append(reasons, WorkspaceProviderExecutionBlockUnsafePrompt)
	}
	if strings.TrimSpace(input.EndpointBaseURL) != "" {
		reasons = append(reasons, WorkspaceProviderExecutionBlockArbitraryBaseURL)
	}
	if !input.ProviderAvailable {
		reasons = append(reasons, WorkspaceProviderExecutionBlockProviderUnavailable)
	}
	if !input.BillingEligibilityKnown {
		reasons = append(reasons, WorkspaceProviderExecutionBlockBillingUnknown)
	} else if !input.BillingEligible {
		reasons = append(reasons, WorkspaceProviderExecutionBlockBillingIneligible)
	}
	if billingPolicy == "" {
		reasons = append(reasons, WorkspaceProviderExecutionBlockBillingPolicy)
	} else if billingPolicy == WorkspaceProviderBillingPolicyNoBilling {
		reasons = append(reasons, WorkspaceProviderExecutionBlockNoFreeProviderCall)
	}
	if usagePolicy == "" || usagePolicy == WorkspaceProviderUsagePolicyAuditOnly {
		reasons = append(reasons, WorkspaceProviderExecutionBlockUsagePolicy)
	}
	return reasons
}

func normalizeWorkspaceProviderCapability(capability WorkspaceProviderCapability) WorkspaceProviderCapability {
	if strings.TrimSpace(string(capability)) == "" {
		return WorkspaceProviderCapabilityText
	}
	return capability
}

func normalizeWorkspaceProviderFailurePolicy(policy WorkspaceProviderFailurePolicy) WorkspaceProviderFailurePolicy {
	if strings.TrimSpace(string(policy)) == "" {
		return WorkspaceProviderFailurePolicyFailClosed
	}
	return policy
}

func workspaceProviderExecutionRequiredSteps() []string {
	return []string{
		"feature_gate_enabled",
		"server_side_model_intent_capability_validation",
		"billing_eligibility_precheck",
		"provider_call",
		"assistant_message_persistence",
		"usage_accounting",
		"redacted_audit_record",
	}
}

func workspaceProviderExecutionFailureSemantics(policy WorkspaceProviderFailurePolicy) []string {
	semantics := []string{
		"provider_not_called_means_no_usage_record",
		"provider_failure_returns_safe_workspace_error_without_internal_details",
		"billing_or_usage_unknown_fails_closed_before_provider_call",
	}
	switch policy {
	case WorkspaceProviderFailurePolicyNoChargeOnFailure:
		semantics = append(semantics, "provider_failure_records_no_charge_or_refundable_state")
	case WorkspaceProviderFailurePolicyReconcileRequired:
		semantics = append(semantics, "message_or_usage_mismatch_requires_reconciliation_audit")
	default:
		semantics = append(semantics, "message_saved_but_usage_failed_must_not_silently_succeed")
	}
	return semantics
}

func workspaceProviderExecutionErrorCode(reasons []WorkspaceProviderExecutionBlockReason) string {
	if len(reasons) == 0 {
		return ""
	}
	values := make([]string, 0, len(reasons))
	for _, reason := range reasons {
		values = append(values, string(reason))
	}
	return strings.Join(values, ",")
}
