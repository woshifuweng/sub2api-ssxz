package service

import "strings"

type WorkspaceProviderReconciliationStatus string

const (
	WorkspaceProviderReconciliationStatusOK      WorkspaceProviderReconciliationStatus = "ok"
	WorkspaceProviderReconciliationStatusBlocked WorkspaceProviderReconciliationStatus = "blocked"
	WorkspaceProviderReconciliationStatusFinding WorkspaceProviderReconciliationStatus = "finding"
)

type WorkspaceProviderReconciliationFinding string

const (
	WorkspaceProviderFindingProviderCalledWithoutUsage       WorkspaceProviderReconciliationFinding = "provider_called_without_usage"
	WorkspaceProviderFindingProviderCalledWithoutAudit       WorkspaceProviderReconciliationFinding = "provider_called_without_audit"
	WorkspaceProviderFindingCompletedMessageWithoutProvider  WorkspaceProviderReconciliationFinding = "completed_message_without_provider_called"
	WorkspaceProviderFindingCompletedMessageWithoutUsage     WorkspaceProviderReconciliationFinding = "completed_message_without_usage"
	WorkspaceProviderFindingCompletedMessageWithoutAudit     WorkspaceProviderReconciliationFinding = "completed_message_without_audit"
	WorkspaceProviderFindingUsageWithoutProviderCalled       WorkspaceProviderReconciliationFinding = "usage_without_provider_called"
	WorkspaceProviderFindingAuditWithoutMessage              WorkspaceProviderReconciliationFinding = "audit_without_message"
	WorkspaceProviderFindingMissingRequestedModel            WorkspaceProviderReconciliationFinding = "missing_requested_model"
	WorkspaceProviderFindingMissingMappedModel               WorkspaceProviderReconciliationFinding = "missing_mapped_model"
	WorkspaceProviderFindingMissingUpstreamModel             WorkspaceProviderReconciliationFinding = "missing_upstream_model"
	WorkspaceProviderFindingMissingProviderLabel             WorkspaceProviderReconciliationFinding = "missing_provider_label"
	WorkspaceProviderFindingMissingErrorCodeOnFailure        WorkspaceProviderReconciliationFinding = "missing_error_code_on_failure"
	WorkspaceProviderFindingBlockedButProviderCalled         WorkspaceProviderReconciliationFinding = "blocked_but_provider_called"
	WorkspaceProviderFindingCounterAllowedButProviderNotCall WorkspaceProviderReconciliationFinding = "counter_allowed_but_provider_not_called"
	WorkspaceProviderFindingCounterBlockedButProviderCalled  WorkspaceProviderReconciliationFinding = "counter_blocked_but_provider_called"
	WorkspaceProviderFindingBlockedWithoutReason             WorkspaceProviderReconciliationFinding = "blocked_without_reason"
)

type WorkspaceProviderReconciliationInput struct {
	AssistantMessage *WorkspaceMessage
	Metadata         map[string]any
}

type WorkspaceProviderReconciliationReport struct {
	RequestID               string                                   `json:"request_id,omitempty"`
	ConversationID          int64                                    `json:"conversation_id,omitempty"`
	AssistantMessageID      int64                                    `json:"assistant_message_id,omitempty"`
	UserID                  int64                                    `json:"user_id,omitempty"`
	RequestedModel          string                                   `json:"requested_model,omitempty"`
	MappedModel             string                                   `json:"mapped_model,omitempty"`
	UpstreamModel           string                                   `json:"upstream_model,omitempty"`
	ProviderLabel           string                                   `json:"provider_label,omitempty"`
	ProviderCalled          bool                                     `json:"provider_called"`
	AssistantMessageStatus  string                                   `json:"assistant_message_status,omitempty"`
	AuditStatus             string                                   `json:"audit_status,omitempty"`
	UsagePresent            bool                                     `json:"usage_present"`
	UsageTotalTokens        int64                                    `json:"usage_total_tokens,omitempty"`
	LatencyMS               int64                                    `json:"latency_ms,omitempty"`
	ErrorCode               string                                   `json:"error_code,omitempty"`
	FallbackUsed            bool                                     `json:"fallback_used"`
	BetaAllowlistAllowed    *bool                                    `json:"beta_allowlist_allowed,omitempty"`
	BetaCounterAllowed      *bool                                    `json:"beta_counter_allowed,omitempty"`
	BetaCounterBlockReasons []string                                 `json:"beta_counter_block_reasons,omitempty"`
	StagingQAUsed           int64                                    `json:"staging_qa_used,omitempty"`
	KillSwitchBlocked       bool                                     `json:"kill_switch_blocked"`
	ReconciliationStatus    WorkspaceProviderReconciliationStatus    `json:"reconciliation_status"`
	Findings                []WorkspaceProviderReconciliationFinding `json:"findings,omitempty"`
}

func BuildWorkspaceProviderReconciliationReportForMessage(message WorkspaceMessage) WorkspaceProviderReconciliationReport {
	return BuildWorkspaceProviderReconciliationReport(WorkspaceProviderReconciliationInput{AssistantMessage: &message})
}

func BuildWorkspaceProviderReconciliationReport(input WorkspaceProviderReconciliationInput) WorkspaceProviderReconciliationReport {
	metadata := cloneWorkspaceProviderReconciliationMetadata(input.Metadata)
	if input.AssistantMessage != nil {
		messageMetadata := cloneWorkspaceProviderReconciliationMetadata(input.AssistantMessage.Metadata)
		mergeWorkspaceMetadata(messageMetadata, metadata)
		metadata = messageMetadata
	}

	report := WorkspaceProviderReconciliationReport{
		RequestID:               workspaceMetadataString(metadata, "request_id"),
		RequestedModel:          workspaceMetadataString(metadata, "requested_model"),
		MappedModel:             workspaceMetadataString(metadata, "mapped_model"),
		UpstreamModel:           workspaceMetadataString(metadata, "upstream_model"),
		ProviderLabel:           workspaceMetadataString(metadata, "provider_label", "provider_name", "staging_qa_provider", "beta_allowlist_provider", "beta_counter_provider"),
		ProviderCalled:          workspaceMetadataBoolDefault(metadata, "provider_called", false),
		AuditStatus:             workspaceMetadataString(metadata, "audit_status"),
		LatencyMS:               workspaceMetadataInt64(metadata, "latency_ms"),
		ErrorCode:               workspaceMetadataString(metadata, "error_code", "audit_error_code"),
		FallbackUsed:            workspaceMetadataBoolDefault(metadata, "fallback_used", false),
		BetaAllowlistAllowed:    workspaceMetadataBoolPointer(metadata, "beta_allowlist_allowed"),
		BetaCounterAllowed:      workspaceMetadataBoolPointer(metadata, "beta_counter_allowed"),
		BetaCounterBlockReasons: workspaceMetadataStringSlice(metadata, "beta_counter_block_reasons"),
		StagingQAUsed:           workspaceMetadataInt64(metadata, "staging_qa_used"),
	}
	report.UsagePresent, report.UsageTotalTokens = workspaceMetadataUsage(metadata)
	report.KillSwitchBlocked = workspaceMetadataHasBlockReason(metadata, WorkspaceTextProviderGateReasonKillSwitchActive)

	if input.AssistantMessage != nil {
		report.AssistantMessageID = input.AssistantMessage.ID
		report.ConversationID = input.AssistantMessage.ConversationID
		report.UserID = input.AssistantMessage.UserID
		report.AssistantMessageStatus = input.AssistantMessage.Status
	} else {
		report.AssistantMessageStatus = workspaceMetadataString(metadata, "status")
	}

	blocked := workspaceReconciliationBlocked(report, metadata)
	report.Findings = workspaceReconciliationFindings(report, input.AssistantMessage == nil, blocked)
	report.ReconciliationStatus = workspaceReconciliationStatus(report.Findings, blocked)
	return report
}

func workspaceReconciliationFindings(report WorkspaceProviderReconciliationReport, missingMessage, blocked bool) []WorkspaceProviderReconciliationFinding {
	findings := make([]WorkspaceProviderReconciliationFinding, 0)
	add := func(finding WorkspaceProviderReconciliationFinding) {
		findings = append(findings, finding)
	}

	if report.ProviderCalled && !report.UsagePresent {
		add(WorkspaceProviderFindingProviderCalledWithoutUsage)
	}
	if report.ProviderCalled && report.AuditStatus == "" {
		add(WorkspaceProviderFindingProviderCalledWithoutAudit)
	}
	completed := report.AssistantMessageStatus == WorkspaceMessageStatusCompleted
	if completed && !blocked && !report.ProviderCalled {
		add(WorkspaceProviderFindingCompletedMessageWithoutProvider)
	}
	if completed && !blocked && !report.UsagePresent {
		add(WorkspaceProviderFindingCompletedMessageWithoutUsage)
	}
	if completed && !blocked && report.AuditStatus == "" {
		add(WorkspaceProviderFindingCompletedMessageWithoutAudit)
	}
	if !report.ProviderCalled && report.UsagePresent {
		add(WorkspaceProviderFindingUsageWithoutProviderCalled)
	}
	if missingMessage && report.AuditStatus != "" {
		add(WorkspaceProviderFindingAuditWithoutMessage)
	}
	if report.RequestedModel == "" {
		add(WorkspaceProviderFindingMissingRequestedModel)
	}
	if report.MappedModel == "" {
		add(WorkspaceProviderFindingMissingMappedModel)
	}
	if report.UpstreamModel == "" {
		add(WorkspaceProviderFindingMissingUpstreamModel)
	}
	if report.ProviderLabel == "" {
		add(WorkspaceProviderFindingMissingProviderLabel)
	}
	if workspaceReconciliationFailureStatus(report.AssistantMessageStatus, report.AuditStatus) && report.ErrorCode == "" {
		add(WorkspaceProviderFindingMissingErrorCodeOnFailure)
	}
	if blocked && report.ProviderCalled {
		add(WorkspaceProviderFindingBlockedButProviderCalled)
	}
	if report.BetaCounterAllowed != nil && *report.BetaCounterAllowed && !report.ProviderCalled && !blocked {
		add(WorkspaceProviderFindingCounterAllowedButProviderNotCall)
	}
	if report.BetaCounterAllowed != nil && !*report.BetaCounterAllowed && report.ProviderCalled {
		add(WorkspaceProviderFindingCounterBlockedButProviderCalled)
	}
	if blocked && !workspaceReconciliationHasBlockReason(report) {
		add(WorkspaceProviderFindingBlockedWithoutReason)
	}
	return findings
}

func workspaceReconciliationStatus(findings []WorkspaceProviderReconciliationFinding, blocked bool) WorkspaceProviderReconciliationStatus {
	if len(findings) > 0 {
		return WorkspaceProviderReconciliationStatusFinding
	}
	if blocked {
		return WorkspaceProviderReconciliationStatusBlocked
	}
	return WorkspaceProviderReconciliationStatusOK
}

func workspaceReconciliationBlocked(report WorkspaceProviderReconciliationReport, metadata map[string]any) bool {
	if report.ProviderCalled {
		return workspaceMetadataHasAnyBlockReason(metadata)
	}
	if workspaceMetadataHasAnyBlockReason(metadata) || report.KillSwitchBlocked {
		return true
	}
	if report.BetaAllowlistAllowed != nil && !*report.BetaAllowlistAllowed {
		return true
	}
	if report.BetaCounterAllowed != nil && !*report.BetaCounterAllowed {
		return true
	}
	if allowed := workspaceMetadataBoolPointer(metadata, "staging_qa_allowed"); allowed != nil && !*allowed {
		return true
	}
	return strings.EqualFold(workspaceMetadataString(metadata, "status"), "unavailable")
}

func workspaceReconciliationHasBlockReason(report WorkspaceProviderReconciliationReport) bool {
	if report.KillSwitchBlocked || len(report.BetaCounterBlockReasons) > 0 {
		return true
	}
	if report.BetaAllowlistAllowed != nil && !*report.BetaAllowlistAllowed {
		return true
	}
	if report.BetaCounterAllowed != nil && !*report.BetaCounterAllowed {
		return true
	}
	return false
}

func workspaceReconciliationFailureStatus(messageStatus, auditStatus string) bool {
	return strings.EqualFold(messageStatus, "failed") || strings.EqualFold(auditStatus, "failed")
}

func workspaceMetadataString(metadata map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := metadata[key]
		if !ok {
			continue
		}
		if typed, ok := value.(string); ok {
			if trimmed := strings.TrimSpace(typed); trimmed != "" {
				return trimmed
			}
		}
	}
	return ""
}

func cloneWorkspaceProviderReconciliationMetadata(metadata map[string]any) map[string]any {
	if metadata == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(metadata))
	for key, value := range metadata {
		out[key] = value
	}
	return out
}

func workspaceMetadataBoolDefault(metadata map[string]any, key string, fallback bool) bool {
	value := workspaceMetadataBoolPointer(metadata, key)
	if value == nil {
		return fallback
	}
	return *value
}

func workspaceMetadataBoolPointer(metadata map[string]any, key string) *bool {
	value, ok := metadata[key]
	if !ok {
		return nil
	}
	switch typed := value.(type) {
	case bool:
		return &typed
	case string:
		parsed := strings.EqualFold(strings.TrimSpace(typed), "true")
		if parsed || strings.EqualFold(strings.TrimSpace(typed), "false") {
			return &parsed
		}
	}
	return nil
}

func workspaceMetadataInt64(metadata map[string]any, key string) int64 {
	value, ok := metadata[key]
	if !ok {
		return 0
	}
	switch typed := value.(type) {
	case int:
		return int64(typed)
	case int64:
		return typed
	case int32:
		return int64(typed)
	case float64:
		return int64(typed)
	case float32:
		return int64(typed)
	case uint:
		return int64(typed)
	case uint64:
		return int64(typed)
	case uint32:
		return int64(typed)
	}
	return 0
}

func workspaceMetadataStringSlice(metadata map[string]any, key string) []string {
	value, ok := metadata[key]
	if !ok {
		return nil
	}
	switch typed := value.(type) {
	case []string:
		return cloneWorkspaceStringSlice(typed)
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if value, ok := item.(string); ok && strings.TrimSpace(value) != "" {
				out = append(out, strings.TrimSpace(value))
			}
		}
		return out
	case string:
		if strings.TrimSpace(typed) == "" {
			return nil
		}
		return []string{strings.TrimSpace(typed)}
	default:
		return nil
	}
}

func workspaceMetadataUsage(metadata map[string]any) (bool, int64) {
	if _, ok := metadata["usage_total_tokens"]; ok {
		return true, workspaceMetadataInt64(metadata, "usage_total_tokens")
	}
	if _, ok := metadata["usage_input_tokens"]; ok {
		return true, 0
	}
	if _, ok := metadata["usage_output_tokens"]; ok {
		return true, 0
	}
	if usage, ok := metadata["usage"].(map[string]any); ok {
		if _, ok := usage["total_tokens"]; ok {
			return true, workspaceMetadataInt64(usage, "total_tokens")
		}
		return len(usage) > 0, 0
	}
	return false, 0
}

func workspaceMetadataHasAnyBlockReason(metadata map[string]any) bool {
	for _, key := range []string{"execution_block_reasons", "beta_allowlist_block_reasons", "beta_counter_block_reasons", "staging_qa_block_reasons"} {
		if len(workspaceMetadataStringSlice(metadata, key)) > 0 {
			return true
		}
	}
	return false
}

func workspaceMetadataHasBlockReason(metadata map[string]any, reason string) bool {
	for _, key := range []string{"execution_block_reasons", "beta_allowlist_block_reasons", "beta_counter_block_reasons", "staging_qa_block_reasons"} {
		for _, value := range workspaceMetadataStringSlice(metadata, key) {
			if value == reason {
				return true
			}
		}
	}
	return false
}
