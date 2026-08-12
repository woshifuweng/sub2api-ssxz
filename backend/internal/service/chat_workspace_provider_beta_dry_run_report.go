package service

import "strings"

type WorkspaceProviderBetaDryRunFinding string

const (
	WorkspaceProviderBetaDryRunFindingAmbiguousAssistantMessage WorkspaceProviderBetaDryRunFinding = "ambiguous_assistant_message"
	WorkspaceProviderBetaDryRunFindingAssistantMessageMissing   WorkspaceProviderBetaDryRunFinding = "assistant_message_missing"
)

type WorkspaceProviderBetaDryRunSecurityFlags struct {
	KeyTokenAuthorizationCookieLeakageSignal bool `json:"key_token_authorization_cookie_leakage_signal"`
	BrowserDirectProviderCallSignal          bool `json:"browser_direct_provider_call_signal"`
	ImageAssetTaskSignal                     bool `json:"image_asset_task_signal"`
	BillingLedgerPaymentAnomalySignal        bool `json:"billing_ledger_payment_anomaly_signal"`
}

type WorkspaceProviderBetaDryRunCounterMetadata struct {
	Allowed       *bool    `json:"allowed,omitempty"`
	BlockReasons  []string `json:"block_reasons,omitempty"`
	UserUsed      int64    `json:"user_used,omitempty"`
	UserLimit     int64    `json:"user_limit,omitempty"`
	ProviderUsed  int64    `json:"provider_used,omitempty"`
	ProviderLimit int64    `json:"provider_limit,omitempty"`
	ModelUsed     int64    `json:"model_used,omitempty"`
	ModelLimit    int64    `json:"model_limit,omitempty"`
	TestRunUsed   int64    `json:"test_run_used,omitempty"`
	TestRunLimit  int64    `json:"test_run_limit,omitempty"`
}

type WorkspaceProviderBetaDryRunReportInput struct {
	RequestID            string
	ConversationID       int64
	AssistantMessageID   int64
	Messages             []WorkspaceMessage
	MonitoringSignals    []WorkspaceProviderMonitoringSignal
	MonitoringThresholds WorkspaceProviderMonitoringThresholds
}

type WorkspaceProviderBetaDryRunReport struct {
	RequestID              string                                     `json:"request_id,omitempty"`
	ConversationID         int64                                      `json:"conversation_id,omitempty"`
	AssistantMessageID     int64                                      `json:"assistant_message_id,omitempty"`
	ProviderCalled         bool                                       `json:"provider_called"`
	UsagePresent           bool                                       `json:"usage_present"`
	UsageTotalTokens       int64                                      `json:"usage_total_tokens,omitempty"`
	AuditPresent           bool                                       `json:"audit_present"`
	RequestedModel         string                                     `json:"requested_model,omitempty"`
	MappedModel            string                                     `json:"mapped_model,omitempty"`
	UpstreamModel          string                                     `json:"upstream_model,omitempty"`
	ProviderLabel          string                                     `json:"provider_label,omitempty"`
	LatencyMS              int64                                      `json:"latency_ms,omitempty"`
	ErrorCode              string                                     `json:"error_code,omitempty"`
	BetaAllowlistAllowed   *bool                                      `json:"beta_allowlist_allowed,omitempty"`
	BetaCounterAllowed     *bool                                      `json:"beta_counter_allowed,omitempty"`
	BetaCounterMetadata    WorkspaceProviderBetaDryRunCounterMetadata `json:"beta_counter_metadata"`
	ReconciliationStatus   WorkspaceProviderReconciliationStatus      `json:"reconciliation_status"`
	ReconciliationFindings []WorkspaceProviderReconciliationFinding   `json:"reconciliation_findings,omitempty"`
	MonitoringAlerts       []WorkspaceProviderMonitoringAlert         `json:"monitoring_alerts,omitempty"`
	SecurityFlags          WorkspaceProviderBetaDryRunSecurityFlags   `json:"security_flags"`
	ExtractionFindings     []WorkspaceProviderBetaDryRunFinding       `json:"extraction_findings,omitempty"`
}

func BuildWorkspaceProviderBetaDryRunReport(input WorkspaceProviderBetaDryRunReportInput) WorkspaceProviderBetaDryRunReport {
	requestID := strings.TrimSpace(input.RequestID)
	message, findings := selectWorkspaceProviderBetaDryRunAssistantMessage(input, requestID)
	reconciliationInput := WorkspaceProviderReconciliationInput{}
	if message != nil {
		reconciliationInput.AssistantMessage = message
	} else {
		reconciliationInput.Metadata = map[string]any{
			"request_id":      requestID,
			"provider_called": false,
			"audit_status":    "",
		}
	}

	reconciliation := BuildWorkspaceProviderReconciliationReport(reconciliationInput)
	if reconciliation.RequestID == "" {
		reconciliation.RequestID = requestID
	}
	if reconciliation.ConversationID == 0 {
		reconciliation.ConversationID = input.ConversationID
	}
	if len(findings) > 0 && reconciliation.ReconciliationStatus == WorkspaceProviderReconciliationStatusOK {
		reconciliation.ReconciliationStatus = WorkspaceProviderReconciliationStatusFinding
	}
	monitoring := BuildWorkspaceProviderMonitoringSummary(WorkspaceProviderMonitoringInput{
		Reports:    []WorkspaceProviderReconciliationReport{reconciliation},
		Signals:    input.MonitoringSignals,
		Thresholds: input.MonitoringThresholds,
	})
	metadata := map[string]any{}
	if message != nil {
		metadata = message.Metadata
	}

	return WorkspaceProviderBetaDryRunReport{
		RequestID:              reconciliation.RequestID,
		ConversationID:         reconciliation.ConversationID,
		AssistantMessageID:     reconciliation.AssistantMessageID,
		ProviderCalled:         reconciliation.ProviderCalled,
		UsagePresent:           reconciliation.UsagePresent,
		UsageTotalTokens:       reconciliation.UsageTotalTokens,
		AuditPresent:           reconciliation.AuditStatus != "",
		RequestedModel:         reconciliation.RequestedModel,
		MappedModel:            reconciliation.MappedModel,
		UpstreamModel:          reconciliation.UpstreamModel,
		ProviderLabel:          reconciliation.ProviderLabel,
		LatencyMS:              reconciliation.LatencyMS,
		ErrorCode:              reconciliation.ErrorCode,
		BetaAllowlistAllowed:   reconciliation.BetaAllowlistAllowed,
		BetaCounterAllowed:     reconciliation.BetaCounterAllowed,
		BetaCounterMetadata:    workspaceProviderBetaDryRunCounterMetadata(reconciliation, metadata),
		ReconciliationStatus:   reconciliation.ReconciliationStatus,
		ReconciliationFindings: reconciliation.Findings,
		MonitoringAlerts:       monitoring.Alerts,
		SecurityFlags:          workspaceProviderBetaDryRunSecurityFlags(monitoring),
		ExtractionFindings:     findings,
	}
}

func selectWorkspaceProviderBetaDryRunAssistantMessage(input WorkspaceProviderBetaDryRunReportInput, requestID string) (*WorkspaceMessage, []WorkspaceProviderBetaDryRunFinding) {
	messages := workspaceProviderBetaDryRunCandidateMessages(input.Messages, input.ConversationID)
	if input.AssistantMessageID > 0 {
		for i := range messages {
			if messages[i].ID == input.AssistantMessageID {
				return &messages[i], nil
			}
		}
		return nil, []WorkspaceProviderBetaDryRunFinding{WorkspaceProviderBetaDryRunFindingAssistantMessageMissing}
	}

	if requestID != "" {
		matches := make([]WorkspaceMessage, 0, 1)
		for _, message := range messages {
			if workspaceMetadataString(message.Metadata, "request_id") == requestID {
				matches = append(matches, message)
			}
		}
		return workspaceProviderBetaDryRunSingleMatch(matches)
	}

	providerMatches := make([]WorkspaceMessage, 0, 1)
	for _, message := range messages {
		if workspaceProviderBetaDryRunHasProviderMetadata(message.Metadata) {
			providerMatches = append(providerMatches, message)
		}
	}
	return workspaceProviderBetaDryRunSingleMatch(providerMatches)
}

func workspaceProviderBetaDryRunCandidateMessages(messages []WorkspaceMessage, conversationID int64) []WorkspaceMessage {
	candidates := make([]WorkspaceMessage, 0, len(messages))
	for _, message := range messages {
		if message.Role != WorkspaceRoleAssistant {
			continue
		}
		if conversationID > 0 && message.ConversationID != conversationID {
			continue
		}
		candidates = append(candidates, message)
	}
	return candidates
}

func workspaceProviderBetaDryRunSingleMatch(matches []WorkspaceMessage) (*WorkspaceMessage, []WorkspaceProviderBetaDryRunFinding) {
	switch len(matches) {
	case 0:
		return nil, []WorkspaceProviderBetaDryRunFinding{WorkspaceProviderBetaDryRunFindingAssistantMessageMissing}
	case 1:
		return &matches[0], nil
	default:
		return nil, []WorkspaceProviderBetaDryRunFinding{WorkspaceProviderBetaDryRunFindingAmbiguousAssistantMessage}
	}
}

func workspaceProviderBetaDryRunHasProviderMetadata(metadata map[string]any) bool {
	if metadata == nil {
		return false
	}
	if workspaceMetadataBoolDefault(metadata, "provider_called", false) {
		return true
	}
	return workspaceMetadataString(metadata, "provider_name", "provider_label", "requested_model", "mapped_model", "upstream_model", "audit_status") != ""
}

func workspaceProviderBetaDryRunCounterMetadata(report WorkspaceProviderReconciliationReport, metadata map[string]any) WorkspaceProviderBetaDryRunCounterMetadata {
	return WorkspaceProviderBetaDryRunCounterMetadata{
		Allowed:       report.BetaCounterAllowed,
		BlockReasons:  append([]string(nil), report.BetaCounterBlockReasons...),
		UserUsed:      workspaceMetadataInt64(metadata, "beta_user_used"),
		UserLimit:     workspaceMetadataInt64(metadata, "beta_user_limit"),
		ProviderUsed:  workspaceMetadataInt64(metadata, "beta_provider_used"),
		ProviderLimit: workspaceMetadataInt64(metadata, "beta_provider_limit"),
		ModelUsed:     workspaceMetadataInt64(metadata, "beta_model_used"),
		ModelLimit:    workspaceMetadataInt64(metadata, "beta_model_limit"),
		TestRunUsed:   workspaceMetadataInt64(metadata, "beta_test_run_used"),
		TestRunLimit:  workspaceMetadataInt64(metadata, "beta_test_run_limit"),
	}
}

func workspaceProviderBetaDryRunSecurityFlags(summary WorkspaceProviderMonitoringSummary) WorkspaceProviderBetaDryRunSecurityFlags {
	return WorkspaceProviderBetaDryRunSecurityFlags{
		KeyTokenAuthorizationCookieLeakageSignal: summary.KeyLeakageSignalCount > 0,
		BrowserDirectProviderCallSignal:          summary.BrowserDirectProviderCallCount > 0,
		ImageAssetTaskSignal:                     summary.ImageAssetTaskUnexpectedCount > 0,
		BillingLedgerPaymentAnomalySignal:        summary.BillingLedgerPaymentAnomalyCount > 0,
	}
}
