package service

import (
	"context"
	"strings"
	"time"
)

const (
	WorkspaceProviderNameTextAdapter = "workspace_text_provider"
	workspaceTextProviderEndpoint    = "/workspace-provider-text"
)

type WorkspaceTextProviderExecutor interface {
	ExecuteWorkspaceTextProvider(ctx context.Context, input WorkspaceTextProviderExecutionInput) (WorkspaceTextProviderExecutionResult, error)
}

type WorkspaceTextProviderExecutionInput struct {
	RequestID         string
	UserID            int64
	ConversationID    int64
	UserMessageID     int64
	Content           string
	Model             string
	Intent            string
	MappedModel       string
	UpstreamModel     string
	ProviderName      string
	EndpointLabel     string
	ServiceTier       string
	PromptEnhancement *UpstreamPromptEnhancementResult
	ExecutionContract WorkspaceProviderExecutionContract
	OriginalMetadata  map[string]any
}

type WorkspaceTextProviderExecutionResult struct {
	Content        string
	Model          string
	MappedModel    string
	UpstreamModel  string
	ProviderName   string
	EndpointLabel  string
	ServiceTier    string
	FallbackUsed   bool
	FallbackReason string
	LatencyMs      int64
	TokenUsage     UpstreamQualityUsage
	ErrorCode      string
	Metadata       map[string]any
}

type WorkspaceTextProviderAdapter struct {
	FeatureGateEnabled      bool
	Executor                WorkspaceTextProviderExecutor
	RequestID               string
	ProviderName            string
	EndpointLabel           string
	EndpointBaseURL         string
	ServiceTier             string
	MappedModel             string
	UpstreamModel           string
	BillingEligibilityKnown bool
	BillingEligible         bool
	BillingPolicy           WorkspaceProviderBillingPolicy
	UsagePolicy             WorkspaceProviderUsagePolicy
	FailurePolicy           WorkspaceProviderFailurePolicy
}

func (a WorkspaceTextProviderAdapter) GenerateWorkspaceResponse(ctx context.Context, input WorkspaceProviderRequest) (WorkspaceProviderResponse, error) {
	model := strings.TrimSpace(input.Model)
	intent := normalizeWorkspaceIntent(input.Intent)
	providerName := firstNonEmptyWorkspaceValue(a.ProviderName, WorkspaceProviderNameTextAdapter)
	endpointLabel := firstNonEmptyWorkspaceValue(a.EndpointLabel, workspaceTextProviderEndpoint)
	mappedModel := firstNonEmptyWorkspaceValue(a.MappedModel, model)
	upstreamModel := firstNonEmptyWorkspaceValue(a.UpstreamModel, mappedModel)
	promptEnhanced := input.PromptEnhancement != nil
	diagnostics := WorkspaceProviderDiagnostics{
		RequestedModel:        model,
		MappedModel:           mappedModel,
		UpstreamModel:         upstreamModel,
		ProviderName:          providerName,
		ServiceTier:           a.ServiceTier,
		SupportedCapabilities: []WorkspaceProviderCapability{WorkspaceProviderCapabilityText},
		PromptEnhancerUsed:    promptEnhanced,
		PromptEnhancement:     input.PromptEnhancement,
	}
	contract := ValidateWorkspaceProviderExecutionPlan(WorkspaceProviderExecutionRequest{
		RequestID:               a.RequestID,
		FeatureGateEnabled:      a.FeatureGateEnabled,
		UserID:                  input.UserID,
		ConversationID:          input.ConversationID,
		UserMessageID:           input.UserMessageID,
		Content:                 input.Content,
		Model:                   model,
		Intent:                  intent,
		Capability:              input.Capability,
		ProviderAvailable:       a.Executor != nil,
		BillingEligibilityKnown: a.BillingEligibilityKnown,
		BillingEligible:         a.BillingEligible,
		BillingPolicy:           a.BillingPolicy,
		UsagePolicy:             a.UsagePolicy,
		FailurePolicy:           a.FailurePolicy,
		Diagnostics:             diagnostics,
		EndpointLabel:           endpointLabel,
		EndpointBaseURL:         a.EndpointBaseURL,
		PromptEnhancement:       input.PromptEnhancement,
		CreatedAt:               time.Now().UTC(),
	})
	if !contract.CanCallProvider {
		return workspaceTextProviderBlockedResponse(model, intent, diagnostics, contract), nil
	}

	result, err := a.Executor.ExecuteWorkspaceTextProvider(ctx, WorkspaceTextProviderExecutionInput{
		RequestID:         a.RequestID,
		UserID:            input.UserID,
		ConversationID:    input.ConversationID,
		UserMessageID:     input.UserMessageID,
		Content:           input.Content,
		Model:             model,
		Intent:            intent,
		MappedModel:       mappedModel,
		UpstreamModel:     upstreamModel,
		ProviderName:      providerName,
		EndpointLabel:     endpointLabel,
		ServiceTier:       a.ServiceTier,
		PromptEnhancement: input.PromptEnhancement,
		ExecutionContract: contract,
		OriginalMetadata:  input.Metadata,
	})
	if err != nil {
		return workspaceTextProviderFailureResponse(input, model, intent, diagnostics, contract, "workspace_provider_error"), nil
	}
	if strings.TrimSpace(result.Content) == "" {
		return workspaceTextProviderFailureResponse(input, model, intent, diagnostics, contract, "workspace_provider_empty_response"), nil
	}

	successDiagnostics := diagnostics
	successDiagnostics.MappedModel = firstNonEmptyWorkspaceValue(result.MappedModel, mappedModel)
	successDiagnostics.UpstreamModel = firstNonEmptyWorkspaceValue(result.UpstreamModel, upstreamModel)
	successDiagnostics.ProviderName = firstNonEmptyWorkspaceValue(result.ProviderName, providerName)
	successDiagnostics.ServiceTier = firstNonEmptyWorkspaceValue(result.ServiceTier, a.ServiceTier)
	endpointLabel = firstNonEmptyWorkspaceValue(result.EndpointLabel, endpointLabel)
	auditRecord := BuildUpstreamQualityAuditRecord(UpstreamQualityAuditInput{
		RequestID:      a.RequestID,
		Route:          workspaceProviderExecutionRoute,
		Operation:      UpstreamQualityOperationTextCompletion,
		RequestedModel: model,
		MappedModel:    successDiagnostics.MappedModel,
		UpstreamModel:  successDiagnostics.UpstreamModel,
		ProviderName:   successDiagnostics.ProviderName,
		Endpoint:       endpointLabel,
		ServiceTier:    successDiagnostics.ServiceTier,
		FallbackUsed:   result.FallbackUsed,
		FallbackReason: result.FallbackReason,
		LatencyMs:      result.LatencyMs,
		Status:         "succeeded",
		TokenUsage:     result.TokenUsage,
		Prompt:         input.Content,
		PromptEnhanced: promptEnhanced,
		CreatedAt:      time.Now().UTC(),
	})
	successDiagnostics.AuditRecord = &auditRecord

	metadata := workspaceTextProviderSuccessMetadata(result.Metadata, successDiagnostics, contract, endpointLabel, result)
	return WorkspaceProviderResponse{
		Content:     strings.TrimSpace(result.Content),
		Model:       firstNonEmptyWorkspaceValue(result.Model, model),
		Intent:      intent,
		Metadata:    metadata,
		Diagnostics: successDiagnostics,
	}, nil
}

func workspaceTextProviderBlockedResponse(model, intent string, diagnostics WorkspaceProviderDiagnostics, contract WorkspaceProviderExecutionContract) WorkspaceProviderResponse {
	diagnostics.DisabledCapabilityReason = "workspace text provider execution is blocked by safety contract"
	auditRecord := contract.Audit.Record
	diagnostics.AuditRecord = &auditRecord
	metadata := workspaceProviderUnavailableMetadata(diagnostics)
	metadata["provider_adapter"] = "text"
	metadata["execution_decision"] = string(contract.Decision)
	metadata["execution_block_reasons"] = workspaceExecutionReasonsAsStrings(contract.BlockReasons)
	metadata["billing_policy"] = string(contract.BillingPolicy)
	metadata["usage_policy"] = string(contract.UsagePolicy)
	metadata["failure_policy"] = string(contract.FailurePolicy)
	metadata["reconcile_needed"] = contract.Diagnostics.ReconcileNeeded
	return WorkspaceProviderResponse{
		Content:     WorkspaceAssistantUnavailableContent,
		Model:       model,
		Intent:      intent,
		Metadata:    metadata,
		Diagnostics: diagnostics,
	}
}

func workspaceTextProviderFailureResponse(input WorkspaceProviderRequest, model, intent string, diagnostics WorkspaceProviderDiagnostics, contract WorkspaceProviderExecutionContract, errorCode string) WorkspaceProviderResponse {
	auditRecord := BuildUpstreamQualityAuditRecord(UpstreamQualityAuditInput{
		RequestID:      contract.Diagnostics.RequestID,
		Route:          workspaceProviderExecutionRoute,
		Operation:      UpstreamQualityOperationTextCompletion,
		RequestedModel: model,
		MappedModel:    diagnostics.MappedModel,
		UpstreamModel:  diagnostics.UpstreamModel,
		ProviderName:   diagnostics.ProviderName,
		Endpoint:       contract.Diagnostics.EndpointLabel,
		ServiceTier:    diagnostics.ServiceTier,
		Status:         "failed",
		ErrorCode:      errorCode,
		Prompt:         input.Content,
		PromptEnhanced: diagnostics.PromptEnhancerUsed,
		CreatedAt:      time.Now().UTC(),
	})
	diagnostics.AuditRecord = &auditRecord
	metadata := map[string]any{
		"status":               "failed",
		"placeholder":          true,
		"provider_connected":   false,
		"provider_called":      true,
		"billing_touched":      false,
		"asset_touched":        false,
		"provider_adapter":     "text",
		"provider_name":        diagnostics.ProviderName,
		"requested_model":      diagnostics.RequestedModel,
		"mapped_model":         diagnostics.MappedModel,
		"upstream_model":       diagnostics.UpstreamModel,
		"execution_decision":   string(contract.Decision),
		"billing_policy":       string(contract.BillingPolicy),
		"usage_policy":         string(contract.UsagePolicy),
		"failure_policy":       string(contract.FailurePolicy),
		"reconcile_needed":     contract.Diagnostics.ReconcileNeeded,
		"audit_status":         auditRecord.Status,
		"audit_error_code":     auditRecord.ErrorCode,
		"audit_prompt_hash":    auditRecord.PromptHash,
		"audit_endpoint_label": auditRecord.EndpointLabel,
		"prompt_enhancer_used": diagnostics.PromptEnhancerUsed,
		"safe_error":           "workspace text provider failed before a user-visible AI response was completed",
	}
	return WorkspaceProviderResponse{
		Content:     WorkspaceAssistantUnavailableContent,
		Model:       model,
		Intent:      intent,
		Metadata:    metadata,
		Diagnostics: diagnostics,
	}
}

func workspaceTextProviderSuccessMetadata(base map[string]any, diagnostics WorkspaceProviderDiagnostics, contract WorkspaceProviderExecutionContract, endpointLabel string, result WorkspaceTextProviderExecutionResult) map[string]any {
	metadata := make(map[string]any, len(base)+24)
	for key, value := range base {
		metadata[key] = value
	}
	metadata["status"] = "completed"
	metadata["placeholder"] = false
	metadata["provider_connected"] = true
	metadata["provider_called"] = true
	metadata["billing_touched"] = false
	metadata["asset_touched"] = false
	metadata["provider_adapter"] = "text"
	metadata["provider_name"] = diagnostics.ProviderName
	metadata["requested_model"] = diagnostics.RequestedModel
	metadata["mapped_model"] = diagnostics.MappedModel
	metadata["upstream_model"] = diagnostics.UpstreamModel
	metadata["service_tier"] = diagnostics.ServiceTier
	metadata["endpoint_label"] = endpointLabel
	metadata["fallback_used"] = result.FallbackUsed
	metadata["fallback_reason"] = result.FallbackReason
	metadata["latency_ms"] = result.LatencyMs
	metadata["usage_input_tokens"] = result.TokenUsage.InputTokens
	metadata["usage_output_tokens"] = result.TokenUsage.OutputTokens
	metadata["usage_total_tokens"] = result.TokenUsage.TotalTokens
	metadata["execution_decision"] = string(contract.Decision)
	metadata["billing_policy"] = string(contract.BillingPolicy)
	metadata["usage_policy"] = string(contract.UsagePolicy)
	metadata["failure_policy"] = string(contract.FailurePolicy)
	metadata["reconcile_needed"] = contract.Diagnostics.ReconcileNeeded
	metadata["prompt_enhancer_used"] = diagnostics.PromptEnhancerUsed
	if diagnostics.AuditRecord != nil {
		metadata["audit_status"] = diagnostics.AuditRecord.Status
		metadata["audit_error_code"] = diagnostics.AuditRecord.ErrorCode
		metadata["audit_prompt_hash"] = diagnostics.AuditRecord.PromptHash
		metadata["audit_endpoint_label"] = diagnostics.AuditRecord.EndpointLabel
	}
	return metadata
}

func workspaceExecutionReasonsAsStrings(reasons []WorkspaceProviderExecutionBlockReason) []string {
	values := make([]string, 0, len(reasons))
	for _, reason := range reasons {
		values = append(values, string(reason))
	}
	return values
}
