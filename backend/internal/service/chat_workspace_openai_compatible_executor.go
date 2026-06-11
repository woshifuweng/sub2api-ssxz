package service

import (
	"context"
	"errors"
	"strings"
	"time"
)

const (
	WorkspaceOpenAICompatibleTextExecutorProviderName = "workspace_openai_compatible"

	workspaceOpenAICompatibleTextExecutorEndpointLabel = "workspace-openai-compatible-text"
	workspaceOpenAICompatibleTextExecutorStatusOK      = "succeeded"

	workspaceOpenAICompatibleTextExecutorErrorBlocked     = "workspace_openai_compatible_execution_blocked"
	workspaceOpenAICompatibleTextExecutorErrorUnavailable = "workspace_openai_compatible_executor_unavailable"
	workspaceOpenAICompatibleTextExecutorErrorProvider    = "workspace_openai_compatible_provider_error"
	workspaceOpenAICompatibleTextExecutorErrorEmpty       = "workspace_openai_compatible_empty_response"
)

var (
	errWorkspaceOpenAICompatibleExecutionBlocked = errors.New("workspace openai-compatible execution blocked by safety contract")
	errWorkspaceOpenAICompatibleUnavailable      = errors.New("workspace openai-compatible executor unavailable")
	errWorkspaceOpenAICompatibleProviderFailed   = errors.New("workspace openai-compatible provider failed safely")
	errWorkspaceOpenAICompatibleEmptyResponse    = errors.New("workspace openai-compatible provider returned empty response")
)

type WorkspaceOpenAICompatibleTextMessage struct {
	Role    string
	Content string
}

type WorkspaceOpenAICompatibleExecutionRequest struct {
	RequestID         string
	UserID            int64
	ConversationID    int64
	UserMessageID     int64
	RequestedModel    string
	MappedModel       string
	UpstreamModel     string
	ProviderLabel     string
	Platform          string
	EndpointLabel     string
	ServiceTier       string
	Messages          []WorkspaceOpenAICompatibleTextMessage
	BillingPolicy     WorkspaceProviderBillingPolicy
	UsagePolicy       WorkspaceProviderUsagePolicy
	FailurePolicy     WorkspaceProviderFailurePolicy
	ExecutionContract WorkspaceProviderExecutionContract
	AuditContext      WorkspaceOpenAICompatibleAuditContext
	StagingContext    WorkspaceOpenAICompatibleStagingContext
}

type WorkspaceOpenAICompatibleAuditContext struct {
	PromptHash            string
	PromptPreviewRedacted string
	PromptEnhancerUsed    bool
}

type WorkspaceOpenAICompatibleStagingContext struct {
	Enabled           bool
	Environment       string
	TestProviderLabel string
}

type WorkspaceOpenAICompatibleExecutionResponse struct {
	Content        string
	Status         string
	RequestedModel string
	MappedModel    string
	UpstreamModel  string
	ProviderLabel  string
	ProviderName   string
	EndpointLabel  string
	ServiceTier    string
	FallbackUsed   bool
	FallbackReason string
	LatencyMs      int64
	Usage          UpstreamQualityUsage
	ErrorCode      string
}

type WorkspaceOpenAICompatibleTextUpstream interface {
	ExecuteWorkspaceOpenAICompatibleText(ctx context.Context, req WorkspaceOpenAICompatibleExecutionRequest) (WorkspaceOpenAICompatibleExecutionResponse, error)
}

type WorkspaceOpenAICompatibleTextExecutor struct {
	Upstream      WorkspaceOpenAICompatibleTextUpstream
	ProviderLabel string
	Platform      string
	EndpointLabel string
	ServiceTier   string
	Now           func() time.Time
}

func (e WorkspaceOpenAICompatibleTextExecutor) ExecuteWorkspaceTextProvider(ctx context.Context, input WorkspaceTextProviderExecutionInput) (WorkspaceTextProviderExecutionResult, error) {
	req, err := e.BuildExecutionRequest(input)
	if err != nil {
		return workspaceOpenAICompatibleTextErrorResult(input, e, workspaceOpenAICompatibleTextExecutorErrorCode(err), false), err
	}

	startedAt := e.now()
	resp, err := e.Upstream.ExecuteWorkspaceOpenAICompatibleText(ctx, req)
	if err != nil {
		return workspaceOpenAICompatibleTextErrorResult(input, e, workspaceOpenAICompatibleTextExecutorErrorProvider, true), errWorkspaceOpenAICompatibleProviderFailed
	}
	if strings.TrimSpace(resp.Content) == "" {
		return workspaceOpenAICompatibleTextErrorResult(input, e, workspaceOpenAICompatibleTextExecutorErrorEmpty, true), errWorkspaceOpenAICompatibleEmptyResponse
	}

	latencyMs := resp.LatencyMs
	if latencyMs <= 0 {
		latencyMs = time.Since(startedAt).Milliseconds()
	}

	providerName := firstNonEmptyWorkspaceValue(resp.ProviderName, resp.ProviderLabel, req.ProviderLabel, WorkspaceOpenAICompatibleTextExecutorProviderName)
	endpointLabel := firstNonEmptyWorkspaceValue(resp.EndpointLabel, req.EndpointLabel, workspaceOpenAICompatibleTextExecutorEndpointLabel)
	status := firstNonEmptyWorkspaceValue(resp.Status, workspaceOpenAICompatibleTextExecutorStatusOK)
	auditRecord := BuildUpstreamQualityAuditRecord(UpstreamQualityAuditInput{
		RequestID:      req.RequestID,
		Route:          workspaceProviderExecutionRoute,
		Operation:      UpstreamQualityOperationTextCompletion,
		RequestedModel: firstNonEmptyWorkspaceValue(resp.RequestedModel, req.RequestedModel),
		MappedModel:    firstNonEmptyWorkspaceValue(resp.MappedModel, req.MappedModel),
		UpstreamModel:  firstNonEmptyWorkspaceValue(resp.UpstreamModel, req.UpstreamModel),
		ProviderName:   providerName,
		Endpoint:       endpointLabel,
		ServiceTier:    firstNonEmptyWorkspaceValue(resp.ServiceTier, req.ServiceTier),
		FallbackUsed:   resp.FallbackUsed,
		FallbackReason: resp.FallbackReason,
		LatencyMs:      latencyMs,
		Status:         status,
		ErrorCode:      resp.ErrorCode,
		TokenUsage:     resp.Usage,
		Prompt:         input.Content,
		PromptEnhanced: req.AuditContext.PromptEnhancerUsed,
		CreatedAt:      e.now(),
	})

	return WorkspaceTextProviderExecutionResult{
		Content:        strings.TrimSpace(resp.Content),
		Model:          req.RequestedModel,
		MappedModel:    firstNonEmptyWorkspaceValue(resp.MappedModel, req.MappedModel),
		UpstreamModel:  firstNonEmptyWorkspaceValue(resp.UpstreamModel, req.UpstreamModel),
		ProviderName:   providerName,
		EndpointLabel:  endpointLabel,
		ServiceTier:    firstNonEmptyWorkspaceValue(resp.ServiceTier, req.ServiceTier),
		FallbackUsed:   resp.FallbackUsed,
		FallbackReason: resp.FallbackReason,
		LatencyMs:      latencyMs,
		TokenUsage:     resp.Usage,
		ErrorCode:      resp.ErrorCode,
		Metadata:       workspaceOpenAICompatibleTextMetadata(auditRecord, req, resp),
	}, nil
}

func (e WorkspaceOpenAICompatibleTextExecutor) BuildExecutionRequest(input WorkspaceTextProviderExecutionInput) (WorkspaceOpenAICompatibleExecutionRequest, error) {
	if err := validateWorkspaceOpenAICompatibleExecutionInput(input); err != nil {
		return WorkspaceOpenAICompatibleExecutionRequest{}, err
	}
	if e.Upstream == nil {
		return WorkspaceOpenAICompatibleExecutionRequest{}, errWorkspaceOpenAICompatibleUnavailable
	}

	providerLabel := firstNonEmptyWorkspaceValue(e.ProviderLabel, input.ProviderName, WorkspaceOpenAICompatibleTextExecutorProviderName)
	endpointLabel := firstNonEmptyWorkspaceValue(e.EndpointLabel, input.EndpointLabel, workspaceOpenAICompatibleTextExecutorEndpointLabel)
	serviceTier := firstNonEmptyWorkspaceValue(e.ServiceTier, input.ServiceTier)
	auditRecord := BuildUpstreamQualityAuditRecord(UpstreamQualityAuditInput{
		RequestID:      input.RequestID,
		Route:          workspaceProviderExecutionRoute,
		Operation:      UpstreamQualityOperationTextCompletion,
		RequestedModel: input.Model,
		MappedModel:    input.MappedModel,
		UpstreamModel:  input.UpstreamModel,
		ProviderName:   providerLabel,
		Endpoint:       endpointLabel,
		ServiceTier:    serviceTier,
		Status:         "planned",
		Prompt:         input.Content,
		PromptEnhanced: input.PromptEnhancement != nil,
		CreatedAt:      e.now(),
	})

	return WorkspaceOpenAICompatibleExecutionRequest{
		RequestID:      strings.TrimSpace(input.RequestID),
		UserID:         input.UserID,
		ConversationID: input.ConversationID,
		UserMessageID:  input.UserMessageID,
		RequestedModel: strings.TrimSpace(input.Model),
		MappedModel:    firstNonEmptyWorkspaceValue(input.MappedModel, input.Model),
		UpstreamModel:  firstNonEmptyWorkspaceValue(input.UpstreamModel, input.MappedModel, input.Model),
		ProviderLabel:  providerLabel,
		Platform:       firstNonEmptyWorkspaceValue(e.Platform, "openai_compatible"),
		EndpointLabel:  endpointLabel,
		ServiceTier:    serviceTier,
		Messages: []WorkspaceOpenAICompatibleTextMessage{
			{Role: WorkspaceRoleUser, Content: strings.TrimSpace(input.Content)},
		},
		BillingPolicy:     input.ExecutionContract.BillingPolicy,
		UsagePolicy:       input.ExecutionContract.UsagePolicy,
		FailurePolicy:     input.ExecutionContract.FailurePolicy,
		ExecutionContract: input.ExecutionContract,
		AuditContext: WorkspaceOpenAICompatibleAuditContext{
			PromptHash:            auditRecord.PromptHash,
			PromptPreviewRedacted: auditRecord.PromptPreview,
			PromptEnhancerUsed:    input.PromptEnhancement != nil,
		},
		StagingContext: WorkspaceOpenAICompatibleStagingContext{
			Enabled:           true,
			Environment:       serviceTier,
			TestProviderLabel: providerLabel,
		},
	}, nil
}

func validateWorkspaceOpenAICompatibleExecutionInput(input WorkspaceTextProviderExecutionInput) error {
	if !input.ExecutionContract.CanCallProvider || input.ExecutionContract.Decision != WorkspaceProviderExecutionDecisionAllow {
		return errWorkspaceOpenAICompatibleExecutionBlocked
	}
	if !isAllowedWorkspaceModel(input.Model) || normalizeWorkspaceIntent(input.Intent) != WorkspaceIntentChat {
		return errWorkspaceOpenAICompatibleExecutionBlocked
	}
	if input.UserID <= 0 || input.ConversationID <= 0 || input.UserMessageID <= 0 {
		return errWorkspaceOpenAICompatibleExecutionBlocked
	}
	if strings.TrimSpace(input.Content) == "" || containsUnsafeInlinePayload(input.Content) {
		return errWorkspaceOpenAICompatibleExecutionBlocked
	}
	switch input.ExecutionContract.BillingPolicy {
	case "", WorkspaceProviderBillingPolicyNoBilling:
		return errWorkspaceOpenAICompatibleExecutionBlocked
	}
	switch input.ExecutionContract.UsagePolicy {
	case "", WorkspaceProviderUsagePolicyAuditOnly:
		return errWorkspaceOpenAICompatibleExecutionBlocked
	}
	return nil
}

func workspaceOpenAICompatibleTextErrorResult(input WorkspaceTextProviderExecutionInput, executor WorkspaceOpenAICompatibleTextExecutor, errorCode string, providerCalled bool) WorkspaceTextProviderExecutionResult {
	providerName := firstNonEmptyWorkspaceValue(executor.ProviderLabel, input.ProviderName, WorkspaceOpenAICompatibleTextExecutorProviderName)
	endpointLabel := firstNonEmptyWorkspaceValue(executor.EndpointLabel, input.EndpointLabel, workspaceOpenAICompatibleTextExecutorEndpointLabel)
	auditRecord := BuildUpstreamQualityAuditRecord(UpstreamQualityAuditInput{
		RequestID:      input.RequestID,
		Route:          workspaceProviderExecutionRoute,
		Operation:      UpstreamQualityOperationTextCompletion,
		RequestedModel: input.Model,
		MappedModel:    input.MappedModel,
		UpstreamModel:  input.UpstreamModel,
		ProviderName:   providerName,
		Endpoint:       endpointLabel,
		ServiceTier:    firstNonEmptyWorkspaceValue(executor.ServiceTier, input.ServiceTier),
		Status:         "failed",
		ErrorCode:      errorCode,
		Prompt:         input.Content,
		PromptEnhanced: input.PromptEnhancement != nil,
		CreatedAt:      executor.now(),
	})
	return WorkspaceTextProviderExecutionResult{
		Model:         strings.TrimSpace(input.Model),
		MappedModel:   firstNonEmptyWorkspaceValue(input.MappedModel, input.Model),
		UpstreamModel: firstNonEmptyWorkspaceValue(input.UpstreamModel, input.MappedModel, input.Model),
		ProviderName:  providerName,
		EndpointLabel: endpointLabel,
		ServiceTier:   firstNonEmptyWorkspaceValue(executor.ServiceTier, input.ServiceTier),
		ErrorCode:     errorCode,
		Metadata: map[string]any{
			"status":               "failed",
			"provider_adapter":     "openai_compatible",
			"provider_called":      providerCalled,
			"billing_touched":      false,
			"asset_touched":        false,
			"safe_error":           "workspace openai-compatible execution did not complete",
			"audit_status":         auditRecord.Status,
			"audit_error_code":     auditRecord.ErrorCode,
			"audit_prompt_hash":    auditRecord.PromptHash,
			"audit_endpoint_label": auditRecord.EndpointLabel,
		},
	}
}

func workspaceOpenAICompatibleTextMetadata(auditRecord UpstreamQualityAuditRecord, req WorkspaceOpenAICompatibleExecutionRequest, resp WorkspaceOpenAICompatibleExecutionResponse) map[string]any {
	return map[string]any{
		"provider_adapter":     "openai_compatible",
		"provider_label":       req.ProviderLabel,
		"platform":             req.Platform,
		"status":               firstNonEmptyWorkspaceValue(resp.Status, workspaceOpenAICompatibleTextExecutorStatusOK),
		"provider_called":      true,
		"billing_touched":      false,
		"asset_touched":        false,
		"requested_model":      auditRecord.RequestedModel,
		"mapped_model":         auditRecord.MappedModel,
		"upstream_model":       auditRecord.UpstreamModel,
		"endpoint_label":       auditRecord.EndpointLabel,
		"fallback_used":        auditRecord.FallbackUsed,
		"fallback_reason":      auditRecord.FallbackReason,
		"latency_ms":           auditRecord.LatencyMs,
		"usage_input_tokens":   auditRecord.TokenUsage.InputTokens,
		"usage_output_tokens":  auditRecord.TokenUsage.OutputTokens,
		"usage_total_tokens":   auditRecord.TokenUsage.TotalTokens,
		"billing_policy":       string(req.BillingPolicy),
		"usage_policy":         string(req.UsagePolicy),
		"failure_policy":       string(req.FailurePolicy),
		"audit_status":         auditRecord.Status,
		"audit_error_code":     auditRecord.ErrorCode,
		"audit_prompt_hash":    auditRecord.PromptHash,
		"audit_endpoint_label": auditRecord.EndpointLabel,
		"prompt_enhancer_used": req.AuditContext.PromptEnhancerUsed,
	}
}

func workspaceOpenAICompatibleTextExecutorErrorCode(err error) string {
	switch {
	case errors.Is(err, errWorkspaceOpenAICompatibleUnavailable):
		return workspaceOpenAICompatibleTextExecutorErrorUnavailable
	case errors.Is(err, errWorkspaceOpenAICompatibleExecutionBlocked):
		return workspaceOpenAICompatibleTextExecutorErrorBlocked
	default:
		return workspaceOpenAICompatibleTextExecutorErrorProvider
	}
}

func (e WorkspaceOpenAICompatibleTextExecutor) now() time.Time {
	if e.Now != nil {
		return e.Now().UTC()
	}
	return time.Now().UTC()
}
