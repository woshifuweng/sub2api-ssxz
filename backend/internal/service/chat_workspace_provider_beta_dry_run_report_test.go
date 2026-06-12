package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWorkspaceBetaDryRunReportSuccessFromRequestID(t *testing.T) {
	report := BuildWorkspaceProviderBetaDryRunReport(WorkspaceProviderBetaDryRunReportInput{
		RequestID:            "req-dry-run",
		ConversationID:       22,
		Messages:             []WorkspaceMessage{workspaceBetaDryRunUserMessage("prompt must stay private"), workspaceBetaDryRunAssistantMessage(11, "req-dry-run", workspaceBetaDryRunSuccessMetadata())},
		MonitoringThresholds: workspaceMonitoringTestThresholds(),
	})

	require.Equal(t, "req-dry-run", report.RequestID)
	require.Equal(t, int64(22), report.ConversationID)
	require.Equal(t, int64(11), report.AssistantMessageID)
	require.True(t, report.ProviderCalled)
	require.True(t, report.UsagePresent)
	require.Equal(t, int64(9), report.UsageTotalTokens)
	require.True(t, report.AuditPresent)
	require.Equal(t, "deepseek-v4-flash", report.RequestedModel)
	require.Equal(t, "deepseek-v4-flash", report.MappedModel)
	require.Equal(t, "deepseek-v4-flash", report.UpstreamModel)
	require.Equal(t, "deepseek-staging", report.ProviderLabel)
	require.Equal(t, int64(1078), report.LatencyMS)
	require.Empty(t, report.ErrorCode)
	require.NotNil(t, report.BetaAllowlistAllowed)
	require.True(t, *report.BetaAllowlistAllowed)
	require.NotNil(t, report.BetaCounterAllowed)
	require.True(t, *report.BetaCounterAllowed)
	require.Equal(t, int64(1), report.BetaCounterMetadata.UserUsed)
	require.Equal(t, int64(3), report.BetaCounterMetadata.UserLimit)
	require.Equal(t, int64(1), report.BetaCounterMetadata.TestRunUsed)
	require.Equal(t, int64(1), report.BetaCounterMetadata.TestRunLimit)
	require.Equal(t, WorkspaceProviderReconciliationStatusOK, report.ReconciliationStatus)
	require.Empty(t, report.ReconciliationFindings)
	require.Empty(t, report.MonitoringAlerts)
	require.Empty(t, report.ExtractionFindings)
}

func TestWorkspaceBetaDryRunReportFlagsMissingUsage(t *testing.T) {
	metadata := workspaceBetaDryRunSuccessMetadata()
	delete(metadata, "usage_total_tokens")

	report := BuildWorkspaceProviderBetaDryRunReport(WorkspaceProviderBetaDryRunReportInput{
		RequestID:            "req-dry-run",
		ConversationID:       22,
		Messages:             []WorkspaceMessage{workspaceBetaDryRunAssistantMessage(11, "req-dry-run", metadata)},
		MonitoringThresholds: workspaceMonitoringTestThresholds(),
	})

	require.True(t, report.ProviderCalled)
	require.False(t, report.UsagePresent)
	require.Contains(t, report.ReconciliationFindings, WorkspaceProviderFindingProviderCalledWithoutUsage)
	require.Contains(t, report.ReconciliationFindings, WorkspaceProviderFindingCompletedMessageWithoutUsage)
	require.True(t, workspaceBetaDryRunHasAlert(report, WorkspaceProviderAlertCalledWithoutUsage))
	require.True(t, workspaceBetaDryRunHasAlert(report, WorkspaceProviderAlertUsageMissing))
}

func TestWorkspaceBetaDryRunReportFlagsMissingAudit(t *testing.T) {
	metadata := workspaceBetaDryRunSuccessMetadata()
	delete(metadata, "audit_status")

	report := BuildWorkspaceProviderBetaDryRunReport(WorkspaceProviderBetaDryRunReportInput{
		RequestID:            "req-dry-run",
		ConversationID:       22,
		Messages:             []WorkspaceMessage{workspaceBetaDryRunAssistantMessage(11, "req-dry-run", metadata)},
		MonitoringThresholds: workspaceMonitoringTestThresholds(),
	})

	require.True(t, report.ProviderCalled)
	require.False(t, report.AuditPresent)
	require.Contains(t, report.ReconciliationFindings, WorkspaceProviderFindingProviderCalledWithoutAudit)
	require.Contains(t, report.ReconciliationFindings, WorkspaceProviderFindingCompletedMessageWithoutAudit)
	require.True(t, workspaceBetaDryRunHasAlert(report, WorkspaceProviderAlertCalledWithoutAudit))
	require.True(t, workspaceBetaDryRunHasAlert(report, WorkspaceProviderAlertAuditMissing))
}

func TestWorkspaceBetaDryRunReportSurfacesBetaBlockMetadata(t *testing.T) {
	metadata := workspaceBetaDryRunSuccessMetadata()
	metadata["provider_called"] = false
	metadata["status"] = "unavailable"
	metadata["audit_status"] = "blocked"
	metadata["audit_error_code"] = "beta_counter_exceeded"
	metadata["beta_counter_allowed"] = false
	metadata["beta_counter_block_reasons"] = []string{WorkspaceTextProviderBetaCounterReasonTestRunExceeded}
	delete(metadata, "usage_total_tokens")

	report := BuildWorkspaceProviderBetaDryRunReport(WorkspaceProviderBetaDryRunReportInput{
		RequestID:            "req-blocked",
		ConversationID:       22,
		Messages:             []WorkspaceMessage{workspaceBetaDryRunAssistantMessage(11, "req-blocked", metadata)},
		MonitoringThresholds: workspaceMonitoringTestThresholds(),
	})

	require.False(t, report.ProviderCalled)
	require.NotNil(t, report.BetaAllowlistAllowed)
	require.True(t, *report.BetaAllowlistAllowed)
	require.NotNil(t, report.BetaCounterAllowed)
	require.False(t, *report.BetaCounterAllowed)
	require.Equal(t, []string{WorkspaceTextProviderBetaCounterReasonTestRunExceeded}, report.BetaCounterMetadata.BlockReasons)
	require.Equal(t, WorkspaceProviderReconciliationStatusBlocked, report.ReconciliationStatus)
	require.Empty(t, report.ReconciliationFindings)
}

func TestWorkspaceBetaDryRunReportMatchesAssistantMessageIDBeforeRequestID(t *testing.T) {
	first := workspaceBetaDryRunAssistantMessage(11, "req-dry-run", workspaceBetaDryRunSuccessMetadata())
	secondMetadata := workspaceBetaDryRunSuccessMetadata()
	secondMetadata["request_id"] = "req-other"
	secondMetadata["latency_ms"] = int64(222)
	second := workspaceBetaDryRunAssistantMessage(12, "req-other", secondMetadata)

	report := BuildWorkspaceProviderBetaDryRunReport(WorkspaceProviderBetaDryRunReportInput{
		RequestID:          "req-dry-run",
		ConversationID:     22,
		AssistantMessageID: 12,
		Messages:           []WorkspaceMessage{first, second},
	})

	require.Equal(t, int64(12), report.AssistantMessageID)
	require.Equal(t, "req-other", report.RequestID)
	require.Equal(t, int64(222), report.LatencyMS)
}

func TestWorkspaceBetaDryRunReportFlagsAmbiguousAssistantMessage(t *testing.T) {
	report := BuildWorkspaceProviderBetaDryRunReport(WorkspaceProviderBetaDryRunReportInput{
		RequestID: "req-dry-run",
		Messages: []WorkspaceMessage{
			workspaceBetaDryRunAssistantMessage(11, "req-dry-run", workspaceBetaDryRunSuccessMetadata()),
			workspaceBetaDryRunAssistantMessage(12, "req-dry-run", workspaceBetaDryRunSuccessMetadata()),
		},
	})

	require.Zero(t, report.AssistantMessageID)
	require.Contains(t, report.ExtractionFindings, WorkspaceProviderBetaDryRunFindingAmbiguousAssistantMessage)
	require.Equal(t, WorkspaceProviderReconciliationStatusFinding, report.ReconciliationStatus)
}

func TestWorkspaceBetaDryRunReportFlagsMissingAssistantMessage(t *testing.T) {
	report := BuildWorkspaceProviderBetaDryRunReport(WorkspaceProviderBetaDryRunReportInput{
		RequestID:          "req-missing",
		ConversationID:     22,
		Messages:           []WorkspaceMessage{workspaceBetaDryRunUserMessage("prompt must stay private")},
		AssistantMessageID: 99,
	})

	require.Equal(t, "req-missing", report.RequestID)
	require.Equal(t, int64(22), report.ConversationID)
	require.Zero(t, report.AssistantMessageID)
	require.False(t, report.ProviderCalled)
	require.Contains(t, report.ExtractionFindings, WorkspaceProviderBetaDryRunFindingAssistantMessageMissing)
	require.Equal(t, WorkspaceProviderReconciliationStatusFinding, report.ReconciliationStatus)
}

func TestWorkspaceBetaDryRunReportIgnoresUserMessages(t *testing.T) {
	userMessage := workspaceBetaDryRunUserMessage("prompt must stay private")
	userMessage.Metadata = workspaceBetaDryRunSuccessMetadata()

	report := BuildWorkspaceProviderBetaDryRunReport(WorkspaceProviderBetaDryRunReportInput{
		RequestID:      "req-dry-run",
		ConversationID: 22,
		Messages:       []WorkspaceMessage{userMessage},
	})

	require.Zero(t, report.AssistantMessageID)
	require.False(t, report.ProviderCalled)
	require.Contains(t, report.ExtractionFindings, WorkspaceProviderBetaDryRunFindingAssistantMessageMissing)
}

func TestWorkspaceBetaDryRunReportSecuritySignalsAndSanitization(t *testing.T) {
	metadata := workspaceBetaDryRunSuccessMetadata()
	metadata["prompt"] = "full sensitive prompt must not appear"
	metadata["authorization"] = "Bearer provider-token"
	metadata["token"] = "provider-token"
	metadata["cookie"] = "session-cookie"
	metadata["provider_key"] = "sk-provider-key"
	message := workspaceBetaDryRunAssistantMessage(11, "req-dry-run", metadata)
	message.Content = "assistant content only"

	report := BuildWorkspaceProviderBetaDryRunReport(WorkspaceProviderBetaDryRunReportInput{
		RequestID:      "req-dry-run",
		ConversationID: 22,
		Messages:       []WorkspaceMessage{message},
		MonitoringSignals: []WorkspaceProviderMonitoringSignal{
			WorkspaceProviderMonitoringSignalBrowserDirectProviderCall,
			WorkspaceProviderMonitoringSignalKeyOrTokenLeakage,
			WorkspaceProviderMonitoringSignalBillingLedgerPayment,
			WorkspaceProviderMonitoringSignalImageAssetTask,
		},
	})

	require.True(t, report.SecurityFlags.BrowserDirectProviderCallSignal)
	require.True(t, report.SecurityFlags.KeyTokenAuthorizationCookieLeakageSignal)
	require.True(t, report.SecurityFlags.BillingLedgerPaymentAnomalySignal)
	require.True(t, report.SecurityFlags.ImageAssetTaskSignal)
	require.True(t, workspaceBetaDryRunHasAlert(report, WorkspaceProviderAlertBrowserDirectCall))
	require.True(t, workspaceBetaDryRunHasAlert(report, WorkspaceProviderAlertKeyOrTokenLeakage))

	payload, err := json.Marshal(report)
	require.NoError(t, err)
	text := string(payload)
	require.NotContains(t, text, "full sensitive prompt")
	require.NotContains(t, text, "Bearer provider-token")
	require.NotContains(t, text, "provider-token")
	require.NotContains(t, text, "session-cookie")
	require.NotContains(t, text, "sk-provider-key")
	require.NotContains(t, text, "prompt must stay private")
}

func workspaceBetaDryRunSuccessMetadata() map[string]any {
	return map[string]any{
		"request_id":             "req-dry-run",
		"provider_called":        true,
		"requested_model":        "deepseek-v4-flash",
		"mapped_model":           "deepseek-v4-flash",
		"upstream_model":         "deepseek-v4-flash",
		"provider_name":          "deepseek-staging",
		"endpoint_label":         "deepseek-staging",
		"latency_ms":             int64(1078),
		"usage_total_tokens":     9,
		"audit_status":           "succeeded",
		"audit_error_code":       "",
		"beta_allowlist_allowed": true,
		"beta_counter_allowed":   true,
		"beta_user_used":         int64(1),
		"beta_user_limit":        int64(3),
		"beta_provider_used":     int64(1),
		"beta_provider_limit":    int64(3),
		"beta_model_used":        int64(1),
		"beta_model_limit":       int64(3),
		"beta_test_run_used":     int64(1),
		"beta_test_run_limit":    int64(1),
	}
}

func workspaceBetaDryRunAssistantMessage(id int64, requestID string, metadata map[string]any) WorkspaceMessage {
	cloned := cloneWorkspaceProviderReconciliationMetadata(metadata)
	cloned["request_id"] = requestID
	return WorkspaceMessage{
		ID:             id,
		ConversationID: 22,
		UserID:         33,
		MessageType:    WorkspaceMessageTypeText,
		Role:           WorkspaceRoleAssistant,
		Content:        "assistant response",
		Model:          "deepseek-v4-flash",
		Intent:         WorkspaceIntentChat,
		Status:         WorkspaceMessageStatusCompleted,
		Metadata:       cloned,
	}
}

func workspaceBetaDryRunUserMessage(content string) WorkspaceMessage {
	return WorkspaceMessage{
		ID:             10,
		ConversationID: 22,
		UserID:         33,
		MessageType:    WorkspaceMessageTypeText,
		Role:           WorkspaceRoleUser,
		Content:        content,
		Model:          "deepseek-v4-flash",
		Intent:         WorkspaceIntentChat,
		Status:         WorkspaceMessageStatusCompleted,
		Metadata:       map[string]any{"request_id": "req-dry-run"},
	}
}

func workspaceBetaDryRunHasAlert(report WorkspaceProviderBetaDryRunReport, code WorkspaceProviderMonitoringAlertCode) bool {
	for _, alert := range report.MonitoringAlerts {
		if alert.Code == code {
			return true
		}
	}
	return false
}
