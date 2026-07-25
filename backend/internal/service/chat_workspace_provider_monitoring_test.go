package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWorkspaceProviderMonitoringHealthyReportsProduceNoAlerts(t *testing.T) {
	summary := BuildWorkspaceProviderMonitoringSummary(WorkspaceProviderMonitoringInput{
		Reports: []WorkspaceProviderReconciliationReport{
			workspaceMonitoringHealthyReport(),
			workspaceMonitoringHealthyReport(),
		},
		Thresholds: workspaceMonitoringTestThresholds(),
	})

	require.Equal(t, 2, summary.ProviderCalledCount)
	require.Equal(t, 2, summary.ProviderSuccessCount)
	require.Equal(t, 0, summary.ProviderFailureCount)
	require.Equal(t, float64(0), summary.ProviderErrorRate)
	require.Empty(t, summary.Alerts)
}

func TestWorkspaceProviderMonitoringAlertsOnProviderErrorRate(t *testing.T) {
	summary := BuildWorkspaceProviderMonitoringSummary(WorkspaceProviderMonitoringInput{
		Reports: []WorkspaceProviderReconciliationReport{
			workspaceMonitoringHealthyReport(),
			workspaceMonitoringHealthyReport(),
			workspaceMonitoringHealthyReport(),
			workspaceMonitoringFailureReport("workspace_provider_error"),
		},
		Thresholds: workspaceMonitoringTestThresholds(),
	})

	require.Equal(t, 4, summary.ProviderCalledCount)
	require.Equal(t, 1, summary.ProviderFailureCount)
	require.Equal(t, float64(25), summary.ProviderErrorRate)
	require.True(t, workspaceMonitoringHasAlert(summary, WorkspaceProviderAlertErrorRateExceeded))
}

func TestWorkspaceProviderMonitoringAlertsOnConsecutiveFailures(t *testing.T) {
	summary := BuildWorkspaceProviderMonitoringSummary(WorkspaceProviderMonitoringInput{
		Reports: []WorkspaceProviderReconciliationReport{
			workspaceMonitoringFailureReport("workspace_provider_error"),
			workspaceMonitoringFailureReport("workspace_provider_empty_response"),
			workspaceMonitoringFailureReport("workspace_provider_error"),
		},
		Thresholds: workspaceMonitoringTestThresholds(),
	})

	alert := workspaceMonitoringAlertByCode(summary, WorkspaceProviderAlertConsecutiveFailures)
	require.NotNil(t, alert)
	require.Equal(t, WorkspaceProviderMonitoringAlertSeverityWarning, alert.Severity)
	require.Equal(t, 3, alert.Count)
}

func TestWorkspaceProviderMonitoringAlertsOnTimeoutRate(t *testing.T) {
	summary := BuildWorkspaceProviderMonitoringSummary(WorkspaceProviderMonitoringInput{
		Reports: []WorkspaceProviderReconciliationReport{
			workspaceMonitoringHealthyReport(),
			workspaceMonitoringHealthyReport(),
			workspaceMonitoringHealthyReport(),
			workspaceMonitoringFailureReport("workspace_provider_timeout"),
		},
		Thresholds: workspaceMonitoringTestThresholds(),
	})

	require.Equal(t, 1, summary.ProviderTimeoutCount)
	require.True(t, workspaceMonitoringHasAlert(summary, WorkspaceProviderAlertTimeoutRateExceeded))
}

func TestWorkspaceProviderMonitoringAlertsOnUsageAndAuditMissing(t *testing.T) {
	summary := BuildWorkspaceProviderMonitoringSummary(WorkspaceProviderMonitoringInput{
		Reports: []WorkspaceProviderReconciliationReport{
			workspaceMonitoringFindingReport(WorkspaceProviderFindingProviderCalledWithoutUsage),
			workspaceMonitoringFindingReport(WorkspaceProviderFindingProviderCalledWithoutAudit),
		},
		Thresholds: workspaceMonitoringTestThresholds(),
	})

	require.Equal(t, 1, summary.UsageMissingCount)
	require.Equal(t, 1, summary.AuditMissingCount)
	require.True(t, workspaceMonitoringHasAlert(summary, WorkspaceProviderAlertUsageMissing))
	require.True(t, workspaceMonitoringHasAlert(summary, WorkspaceProviderAlertAuditMissing))
	require.True(t, workspaceMonitoringHasAlert(summary, WorkspaceProviderAlertCalledWithoutUsage))
	require.True(t, workspaceMonitoringHasAlert(summary, WorkspaceProviderAlertCalledWithoutAudit))
}

func TestWorkspaceProviderMonitoringAlertsOnCounterBlockedButProviderCalled(t *testing.T) {
	summary := BuildWorkspaceProviderMonitoringSummary(WorkspaceProviderMonitoringInput{
		Reports: []WorkspaceProviderReconciliationReport{{
			ProviderCalled:     true,
			AuditStatus:        "succeeded",
			UsagePresent:       true,
			BetaCounterAllowed: boolPtr(false),
			Findings:           []WorkspaceProviderReconciliationFinding{WorkspaceProviderFindingCounterBlockedButProviderCalled},
		}},
		Thresholds: workspaceMonitoringTestThresholds(),
	})

	alert := workspaceMonitoringAlertByCode(summary, WorkspaceProviderAlertCounterBlockedButCalled)
	require.NotNil(t, alert)
	require.Equal(t, WorkspaceProviderMonitoringAlertSeverityCritical, alert.Severity)
	require.Equal(t, 1, alert.Count)
}

func TestWorkspaceProviderMonitoringAlertsOnKillSwitchBlockedButProviderCalled(t *testing.T) {
	summary := BuildWorkspaceProviderMonitoringSummary(WorkspaceProviderMonitoringInput{
		Reports: []WorkspaceProviderReconciliationReport{{
			ProviderCalled:    true,
			AuditStatus:       "succeeded",
			UsagePresent:      true,
			KillSwitchBlocked: true,
			Findings:          []WorkspaceProviderReconciliationFinding{WorkspaceProviderFindingBlockedButProviderCalled},
		}},
		Thresholds: workspaceMonitoringTestThresholds(),
	})

	alert := workspaceMonitoringAlertByCode(summary, WorkspaceProviderAlertKillSwitchBlockedButCalled)
	require.NotNil(t, alert)
	require.Equal(t, WorkspaceProviderMonitoringAlertSeverityCritical, alert.Severity)
	require.Equal(t, 1, alert.Count)
}

func TestWorkspaceProviderMonitoringTracksBlockAndFallbackCounters(t *testing.T) {
	summary := BuildWorkspaceProviderMonitoringSummary(WorkspaceProviderMonitoringInput{
		Reports: []WorkspaceProviderReconciliationReport{
			{ProviderCalled: false, BetaAllowlistAllowed: boolPtr(false), ReconciliationStatus: WorkspaceProviderReconciliationStatusBlocked},
			{ProviderCalled: false, BetaCounterAllowed: boolPtr(false), BetaCounterBlockReasons: []string{WorkspaceTextProviderBetaCounterReasonTestRunExceeded}, ReconciliationStatus: WorkspaceProviderReconciliationStatusBlocked},
			{ProviderCalled: false, KillSwitchBlocked: true, ReconciliationStatus: WorkspaceProviderReconciliationStatusBlocked},
			{ProviderCalled: true, AuditStatus: "succeeded", UsagePresent: true, FallbackUsed: true},
		},
		Thresholds: workspaceMonitoringTestThresholds(),
	})

	require.Equal(t, 1, summary.BetaAllowlistBlockCount)
	require.Equal(t, 1, summary.BetaCounterBlockCount)
	require.Equal(t, 1, summary.RequestCapExceededCount)
	require.Equal(t, 1, summary.KillSwitchBlockedCount)
	require.Equal(t, 1, summary.FallbackCount)
}

func TestWorkspaceProviderMonitoringAlertsOnExternalSafetySignals(t *testing.T) {
	summary := BuildWorkspaceProviderMonitoringSummary(WorkspaceProviderMonitoringInput{
		Reports: []WorkspaceProviderReconciliationReport{workspaceMonitoringHealthyReport()},
		Signals: []WorkspaceProviderMonitoringSignal{
			WorkspaceProviderMonitoringSignalBrowserDirectProviderCall,
			WorkspaceProviderMonitoringSignalKeyOrTokenLeakage,
			WorkspaceProviderMonitoringSignalBillingLedgerPayment,
			WorkspaceProviderMonitoringSignalImageAssetTask,
		},
		Thresholds: workspaceMonitoringTestThresholds(),
	})

	require.Equal(t, 1, summary.BrowserDirectProviderCallCount)
	require.Equal(t, 1, summary.KeyLeakageSignalCount)
	require.Equal(t, 1, summary.BillingLedgerPaymentAnomalyCount)
	require.Equal(t, 1, summary.ImageAssetTaskUnexpectedCount)
	require.Equal(t, WorkspaceProviderMonitoringAlertSeverityCritical, workspaceMonitoringAlertByCode(summary, WorkspaceProviderAlertBrowserDirectCall).Severity)
	require.Equal(t, WorkspaceProviderMonitoringAlertSeverityCritical, workspaceMonitoringAlertByCode(summary, WorkspaceProviderAlertKeyOrTokenLeakage).Severity)
	require.Equal(t, WorkspaceProviderMonitoringAlertSeverityCritical, workspaceMonitoringAlertByCode(summary, WorkspaceProviderAlertBillingLedgerPaymentAnomaly).Severity)
	require.Equal(t, WorkspaceProviderMonitoringAlertSeverityWarning, workspaceMonitoringAlertByCode(summary, WorkspaceProviderAlertImageAssetTaskUnexpected).Severity)
}

func TestWorkspaceProviderMonitoringOutputDoesNotContainSensitivePromptOrSecrets(t *testing.T) {
	report := workspaceMonitoringHealthyReport()
	report.RequestID = "req-safe"
	summary := BuildWorkspaceProviderMonitoringSummary(WorkspaceProviderMonitoringInput{
		Reports: []WorkspaceProviderReconciliationReport{report},
		Signals: []WorkspaceProviderMonitoringSignal{WorkspaceProviderMonitoringSignalKeyOrTokenLeakage},
	})

	payload, err := json.Marshal(summary)
	require.NoError(t, err)
	text := string(payload)

	require.NotContains(t, text, "full sensitive prompt")
	require.NotContains(t, text, "Authorization")
	require.NotContains(t, text, "provider-token")
	require.NotContains(t, text, "session-cookie")
	require.NotContains(t, text, "sk-provider-key")
}

func workspaceMonitoringHealthyReport() WorkspaceProviderReconciliationReport {
	return WorkspaceProviderReconciliationReport{
		RequestID:              "req-ok",
		RequestedModel:         "deepseek-v4-flash",
		MappedModel:            "deepseek-v4-flash",
		UpstreamModel:          "deepseek-v4-flash",
		ProviderLabel:          "deepseek-staging",
		ProviderCalled:         true,
		AssistantMessageStatus: WorkspaceMessageStatusCompleted,
		AuditStatus:            "succeeded",
		UsagePresent:           true,
		UsageTotalTokens:       6,
		LatencyMS:              120,
		ReconciliationStatus:   WorkspaceProviderReconciliationStatusOK,
	}
}

func workspaceMonitoringFailureReport(errorCode string) WorkspaceProviderReconciliationReport {
	report := workspaceMonitoringHealthyReport()
	report.AuditStatus = "failed"
	report.ErrorCode = errorCode
	report.ReconciliationStatus = WorkspaceProviderReconciliationStatusFinding
	return report
}

func workspaceMonitoringFindingReport(finding WorkspaceProviderReconciliationFinding) WorkspaceProviderReconciliationReport {
	report := workspaceMonitoringHealthyReport()
	report.ReconciliationStatus = WorkspaceProviderReconciliationStatusFinding
	report.Findings = []WorkspaceProviderReconciliationFinding{finding}
	if finding == WorkspaceProviderFindingProviderCalledWithoutUsage {
		report.UsagePresent = false
		report.UsageTotalTokens = 0
	}
	if finding == WorkspaceProviderFindingProviderCalledWithoutAudit {
		report.AuditStatus = ""
	}
	return report
}

func workspaceMonitoringTestThresholds() WorkspaceProviderMonitoringThresholds {
	return WorkspaceProviderMonitoringThresholds{
		ErrorRateThresholdPercent:   20,
		ConsecutiveFailureThreshold: 3,
		TimeoutRateThresholdPercent: 20,
		UsageMissingThreshold:       1,
		AuditMissingThreshold:       1,
	}
}

func workspaceMonitoringHasAlert(summary WorkspaceProviderMonitoringSummary, code WorkspaceProviderMonitoringAlertCode) bool {
	return workspaceMonitoringAlertByCode(summary, code) != nil
}

func workspaceMonitoringAlertByCode(summary WorkspaceProviderMonitoringSummary, code WorkspaceProviderMonitoringAlertCode) *WorkspaceProviderMonitoringAlert {
	for i := range summary.Alerts {
		if summary.Alerts[i].Code == code {
			return &summary.Alerts[i]
		}
	}
	return nil
}
