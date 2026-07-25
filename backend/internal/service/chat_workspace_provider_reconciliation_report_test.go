package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWorkspaceReconciliationSuccessIsOK(t *testing.T) {
	report := BuildWorkspaceProviderReconciliationReportForMessage(workspaceReconciliationMessage(map[string]any{
		"request_id":             "req-ok",
		"provider_called":        true,
		"requested_model":        "deepseek-v4-flash",
		"mapped_model":           "deepseek-v4-flash",
		"upstream_model":         "deepseek-v4-flash",
		"provider_name":          "deepseek-staging",
		"endpoint_label":         "deepseek-staging",
		"latency_ms":             int64(120),
		"usage_total_tokens":     6,
		"audit_status":           "succeeded",
		"audit_error_code":       "",
		"beta_allowlist_allowed": true,
		"beta_counter_allowed":   true,
		"staging_qa_allowed":     true,
		"staging_qa_used":        int64(1),
	}))

	require.Equal(t, WorkspaceProviderReconciliationStatusOK, report.ReconciliationStatus)
	require.Empty(t, report.Findings)
	require.Equal(t, "req-ok", report.RequestID)
	require.Equal(t, int64(11), report.AssistantMessageID)
	require.Equal(t, int64(22), report.ConversationID)
	require.Equal(t, int64(33), report.UserID)
	require.True(t, report.ProviderCalled)
	require.True(t, report.UsagePresent)
	require.Equal(t, int64(6), report.UsageTotalTokens)
	require.Equal(t, int64(120), report.LatencyMS)
	require.Equal(t, "deepseek-staging", report.ProviderLabel)
	require.NotNil(t, report.BetaAllowlistAllowed)
	require.True(t, *report.BetaAllowlistAllowed)
	require.NotNil(t, report.BetaCounterAllowed)
	require.True(t, *report.BetaCounterAllowed)
}

func TestWorkspaceReconciliationFindsProviderCalledWithoutUsage(t *testing.T) {
	report := BuildWorkspaceProviderReconciliationReportForMessage(workspaceReconciliationMessage(map[string]any{
		"provider_called": true,
		"requested_model": "deepseek-v4-flash",
		"mapped_model":    "deepseek-v4-flash",
		"upstream_model":  "deepseek-v4-flash",
		"provider_name":   "deepseek-staging",
		"audit_status":    "succeeded",
	}))

	require.Contains(t, report.Findings, WorkspaceProviderFindingProviderCalledWithoutUsage)
	require.Contains(t, report.Findings, WorkspaceProviderFindingCompletedMessageWithoutUsage)
	require.Equal(t, WorkspaceProviderReconciliationStatusFinding, report.ReconciliationStatus)
}

func TestWorkspaceReconciliationFindsProviderCalledWithoutAudit(t *testing.T) {
	report := BuildWorkspaceProviderReconciliationReportForMessage(workspaceReconciliationMessage(map[string]any{
		"provider_called":    true,
		"requested_model":    "deepseek-v4-flash",
		"mapped_model":       "deepseek-v4-flash",
		"upstream_model":     "deepseek-v4-flash",
		"provider_name":      "deepseek-staging",
		"usage_total_tokens": 4,
	}))

	require.Contains(t, report.Findings, WorkspaceProviderFindingProviderCalledWithoutAudit)
	require.Contains(t, report.Findings, WorkspaceProviderFindingCompletedMessageWithoutAudit)
	require.Equal(t, WorkspaceProviderReconciliationStatusFinding, report.ReconciliationStatus)
}

func TestWorkspaceReconciliationFindsCompletedMessageWithoutProviderCalled(t *testing.T) {
	report := BuildWorkspaceProviderReconciliationReportForMessage(workspaceReconciliationMessage(map[string]any{
		"provider_called":    false,
		"requested_model":    "deepseek-v4-flash",
		"mapped_model":       "deepseek-v4-flash",
		"upstream_model":     "deepseek-v4-flash",
		"provider_name":      "deepseek-staging",
		"usage_total_tokens": 3,
		"audit_status":       "succeeded",
	}))

	require.Contains(t, report.Findings, WorkspaceProviderFindingCompletedMessageWithoutProvider)
	require.Contains(t, report.Findings, WorkspaceProviderFindingUsageWithoutProviderCalled)
}

func TestWorkspaceReconciliationBlockedPlaceholderAllowsNoUsage(t *testing.T) {
	report := BuildWorkspaceProviderReconciliationReportForMessage(workspaceReconciliationMessage(map[string]any{
		"status":                  "unavailable",
		"provider_called":         false,
		"requested_model":         "deepseek-v4-flash",
		"mapped_model":            "deepseek-v4-flash",
		"upstream_model":          "deepseek-v4-flash",
		"provider_name":           "deepseek-staging",
		"audit_status":            "blocked",
		"audit_error_code":        "kill_switch_active",
		"execution_block_reasons": []string{WorkspaceTextProviderGateReasonKillSwitchActive},
	}))

	require.Equal(t, WorkspaceProviderReconciliationStatusBlocked, report.ReconciliationStatus)
	require.Empty(t, report.Findings)
	require.True(t, report.KillSwitchBlocked)
	require.False(t, report.ProviderCalled)
	require.False(t, report.UsagePresent)
}

func TestWorkspaceReconciliationBlockedPathsDoNotCallProvider(t *testing.T) {
	tests := []struct {
		name     string
		metadata map[string]any
	}{
		{
			name: "kill switch",
			metadata: map[string]any{
				"execution_block_reasons": []any{WorkspaceTextProviderGateReasonKillSwitchActive},
			},
		},
		{
			name: "beta allowlist",
			metadata: map[string]any{
				"beta_allowlist_allowed":       false,
				"beta_allowlist_block_reasons": []string{WorkspaceTextProviderBetaReasonSubjectNotAllowed},
			},
		},
		{
			name: "beta counter",
			metadata: map[string]any{
				"beta_counter_allowed":       false,
				"beta_counter_block_reasons": []string{WorkspaceTextProviderBetaCounterReasonUserLimitExceeded},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata := map[string]any{
				"status":           "unavailable",
				"provider_called":  false,
				"requested_model":  "deepseek-v4-flash",
				"mapped_model":     "deepseek-v4-flash",
				"upstream_model":   "deepseek-v4-flash",
				"provider_name":    "deepseek-staging",
				"audit_status":     "blocked",
				"audit_error_code": tt.name,
			}
			mergeWorkspaceMetadata(metadata, tt.metadata)

			report := BuildWorkspaceProviderReconciliationReportForMessage(workspaceReconciliationMessage(metadata))

			require.False(t, report.ProviderCalled)
			require.Equal(t, WorkspaceProviderReconciliationStatusBlocked, report.ReconciliationStatus)
			require.Empty(t, report.Findings)
		})
	}
}

func TestWorkspaceReconciliationFindsMissingMetadata(t *testing.T) {
	report := BuildWorkspaceProviderReconciliationReportForMessage(workspaceReconciliationMessage(map[string]any{
		"provider_called":    true,
		"usage_total_tokens": 1,
		"audit_status":       "succeeded",
	}))

	require.Contains(t, report.Findings, WorkspaceProviderFindingMissingRequestedModel)
	require.Contains(t, report.Findings, WorkspaceProviderFindingMissingMappedModel)
	require.Contains(t, report.Findings, WorkspaceProviderFindingMissingUpstreamModel)
	require.Contains(t, report.Findings, WorkspaceProviderFindingMissingProviderLabel)
}

func TestWorkspaceReconciliationFindsFailureWithoutErrorCode(t *testing.T) {
	message := workspaceReconciliationMessage(map[string]any{
		"provider_called":    true,
		"requested_model":    "deepseek-v4-flash",
		"mapped_model":       "deepseek-v4-flash",
		"upstream_model":     "deepseek-v4-flash",
		"provider_name":      "deepseek-staging",
		"usage_total_tokens": 1,
		"audit_status":       "failed",
	})
	message.Status = "failed"

	report := BuildWorkspaceProviderReconciliationReportForMessage(message)

	require.Contains(t, report.Findings, WorkspaceProviderFindingMissingErrorCodeOnFailure)
}

func TestWorkspaceReconciliationFindsCounterInconsistencies(t *testing.T) {
	t.Run("counter allowed but provider not called", func(t *testing.T) {
		report := BuildWorkspaceProviderReconciliationReportForMessage(workspaceReconciliationMessage(map[string]any{
			"provider_called":      false,
			"requested_model":      "deepseek-v4-flash",
			"mapped_model":         "deepseek-v4-flash",
			"upstream_model":       "deepseek-v4-flash",
			"provider_name":        "deepseek-staging",
			"audit_status":         "blocked",
			"beta_counter_allowed": true,
		}))

		require.Contains(t, report.Findings, WorkspaceProviderFindingCounterAllowedButProviderNotCall)
	})

	t.Run("counter blocked but provider called", func(t *testing.T) {
		report := BuildWorkspaceProviderReconciliationReportForMessage(workspaceReconciliationMessage(map[string]any{
			"provider_called":            true,
			"requested_model":            "deepseek-v4-flash",
			"mapped_model":               "deepseek-v4-flash",
			"upstream_model":             "deepseek-v4-flash",
			"provider_name":              "deepseek-staging",
			"usage_total_tokens":         1,
			"audit_status":               "succeeded",
			"beta_counter_allowed":       false,
			"beta_counter_block_reasons": []string{WorkspaceTextProviderBetaCounterReasonModelExceeded},
		}))

		require.Contains(t, report.Findings, WorkspaceProviderFindingCounterBlockedButProviderCalled)
		require.Contains(t, report.Findings, WorkspaceProviderFindingBlockedButProviderCalled)
	})
}

func TestWorkspaceReconciliationFindsAuditWithoutMessage(t *testing.T) {
	report := BuildWorkspaceProviderReconciliationReport(WorkspaceProviderReconciliationInput{
		Metadata: map[string]any{
			"provider_called":    false,
			"requested_model":    "deepseek-v4-flash",
			"mapped_model":       "deepseek-v4-flash",
			"upstream_model":     "deepseek-v4-flash",
			"provider_name":      "deepseek-staging",
			"audit_status":       "succeeded",
			"usage_total_tokens": 2,
		},
	})

	require.Contains(t, report.Findings, WorkspaceProviderFindingAuditWithoutMessage)
}

func TestWorkspaceReconciliationReportExcludesSensitivePromptAndSecretValues(t *testing.T) {
	report := BuildWorkspaceProviderReconciliationReportForMessage(workspaceReconciliationMessage(map[string]any{
		"provider_called":    true,
		"requested_model":    "deepseek-v4-flash",
		"mapped_model":       "deepseek-v4-flash",
		"upstream_model":     "deepseek-v4-flash",
		"provider_name":      "deepseek-staging",
		"usage_total_tokens": 3,
		"audit_status":       "succeeded",
		"prompt":             "full sensitive prompt must not appear",
		"authorization":      "Bearer provider-token",
		"token":              "provider-token",
		"cookie":             "session-cookie",
		"provider_key":       "sk-provider-key",
	}))

	payload, err := json.Marshal(report)
	require.NoError(t, err)
	text := string(payload)

	require.NotContains(t, text, "full sensitive prompt")
	require.NotContains(t, text, "Bearer provider-token")
	require.NotContains(t, text, "session-cookie")
	require.NotContains(t, text, "sk-provider-key")
}

func workspaceReconciliationMessage(metadata map[string]any) WorkspaceMessage {
	return WorkspaceMessage{
		ID:             11,
		ConversationID: 22,
		UserID:         33,
		MessageType:    WorkspaceMessageTypeText,
		Role:           WorkspaceRoleAssistant,
		Content:        "assistant response",
		Model:          "deepseek-v4-flash",
		Intent:         WorkspaceIntentChat,
		Status:         WorkspaceMessageStatusCompleted,
		Metadata:       metadata,
	}
}
