package service

type WorkspaceProviderMonitoringAlertSeverity string

const (
	WorkspaceProviderMonitoringAlertSeverityWarning  WorkspaceProviderMonitoringAlertSeverity = "warning"
	WorkspaceProviderMonitoringAlertSeverityCritical WorkspaceProviderMonitoringAlertSeverity = "critical"
)

type WorkspaceProviderMonitoringAlertCode string

const (
	WorkspaceProviderAlertErrorRateExceeded           WorkspaceProviderMonitoringAlertCode = "provider_error_rate_exceeded"
	WorkspaceProviderAlertConsecutiveFailures         WorkspaceProviderMonitoringAlertCode = "provider_consecutive_failures"
	WorkspaceProviderAlertTimeoutRateExceeded         WorkspaceProviderMonitoringAlertCode = "provider_timeout_rate_exceeded"
	WorkspaceProviderAlertUsageMissing                WorkspaceProviderMonitoringAlertCode = "usage_missing_detected"
	WorkspaceProviderAlertAuditMissing                WorkspaceProviderMonitoringAlertCode = "audit_missing_detected"
	WorkspaceProviderAlertCalledWithoutUsage          WorkspaceProviderMonitoringAlertCode = "provider_called_without_usage"
	WorkspaceProviderAlertCalledWithoutAudit          WorkspaceProviderMonitoringAlertCode = "provider_called_without_audit"
	WorkspaceProviderAlertCounterBlockedButCalled     WorkspaceProviderMonitoringAlertCode = "counter_blocked_but_provider_called"
	WorkspaceProviderAlertKillSwitchBlockedButCalled  WorkspaceProviderMonitoringAlertCode = "kill_switch_blocked_but_provider_called"
	WorkspaceProviderAlertBrowserDirectCall           WorkspaceProviderMonitoringAlertCode = "browser_direct_provider_call_detected"
	WorkspaceProviderAlertKeyOrTokenLeakage           WorkspaceProviderMonitoringAlertCode = "key_or_token_leakage_signal"
	WorkspaceProviderAlertBillingLedgerPaymentAnomaly WorkspaceProviderMonitoringAlertCode = "billing_ledger_payment_anomaly_signal"
	WorkspaceProviderAlertImageAssetTaskUnexpected    WorkspaceProviderMonitoringAlertCode = "image_asset_task_unexpected_signal"
)

type WorkspaceProviderMonitoringSignal string

const (
	WorkspaceProviderMonitoringSignalBrowserDirectProviderCall WorkspaceProviderMonitoringSignal = "browser_direct_provider_call"
	WorkspaceProviderMonitoringSignalKeyOrTokenLeakage         WorkspaceProviderMonitoringSignal = "key_or_token_leakage"
	WorkspaceProviderMonitoringSignalBillingLedgerPayment      WorkspaceProviderMonitoringSignal = "billing_ledger_payment_anomaly"
	WorkspaceProviderMonitoringSignalImageAssetTask            WorkspaceProviderMonitoringSignal = "image_asset_task_unexpected"
)

type WorkspaceProviderMonitoringThresholds struct {
	ErrorRateThresholdPercent   int
	ConsecutiveFailureThreshold int
	TimeoutRateThresholdPercent int
	UsageMissingThreshold       int
	AuditMissingThreshold       int
}

type WorkspaceProviderMonitoringInput struct {
	Reports    []WorkspaceProviderReconciliationReport
	Signals    []WorkspaceProviderMonitoringSignal
	Thresholds WorkspaceProviderMonitoringThresholds
}

type WorkspaceProviderMonitoringSummary struct {
	ProviderCalledCount              int                                `json:"provider_called_count"`
	ProviderSuccessCount             int                                `json:"provider_success_count"`
	ProviderFailureCount             int                                `json:"provider_failure_count"`
	ProviderTimeoutCount             int                                `json:"provider_timeout_count"`
	ProviderErrorRate                float64                            `json:"provider_error_rate"`
	FallbackCount                    int                                `json:"fallback_count"`
	BetaAllowlistBlockCount          int                                `json:"beta_allowlist_block_count"`
	BetaCounterBlockCount            int                                `json:"beta_counter_block_count"`
	RequestCapExceededCount          int                                `json:"request_cap_exceeded_count"`
	KillSwitchBlockedCount           int                                `json:"kill_switch_blocked_count"`
	UsageMissingCount                int                                `json:"usage_missing_count"`
	AuditMissingCount                int                                `json:"audit_missing_count"`
	ReconciliationErrorCount         int                                `json:"reconciliation_error_count"`
	BrowserDirectProviderCallCount   int                                `json:"browser_direct_provider_call_count"`
	KeyLeakageSignalCount            int                                `json:"key_leakage_signal_count"`
	BillingLedgerPaymentAnomalyCount int                                `json:"billing_ledger_payment_anomaly_count"`
	ImageAssetTaskUnexpectedCount    int                                `json:"image_asset_task_unexpected_count"`
	Alerts                           []WorkspaceProviderMonitoringAlert `json:"alerts,omitempty"`
}

type WorkspaceProviderMonitoringAlert struct {
	Code     WorkspaceProviderMonitoringAlertCode     `json:"code"`
	Severity WorkspaceProviderMonitoringAlertSeverity `json:"severity"`
	Count    int                                      `json:"count"`
}

func BuildWorkspaceProviderMonitoringSummary(input WorkspaceProviderMonitoringInput) WorkspaceProviderMonitoringSummary {
	summary := WorkspaceProviderMonitoringSummary{}
	consecutiveFailures := 0
	maxConsecutiveFailures := 0

	for _, report := range input.Reports {
		if report.ProviderCalled {
			summary.ProviderCalledCount++
			if workspaceProviderMonitoringReportFailed(report) {
				summary.ProviderFailureCount++
				consecutiveFailures++
				if consecutiveFailures > maxConsecutiveFailures {
					maxConsecutiveFailures = consecutiveFailures
				}
			} else {
				summary.ProviderSuccessCount++
				consecutiveFailures = 0
			}
		} else {
			consecutiveFailures = 0
		}
		if workspaceProviderMonitoringReportTimedOut(report) {
			summary.ProviderTimeoutCount++
		}
		if report.FallbackUsed {
			summary.FallbackCount++
		}
		if report.BetaAllowlistAllowed != nil && !*report.BetaAllowlistAllowed {
			summary.BetaAllowlistBlockCount++
		}
		if report.BetaCounterAllowed != nil && !*report.BetaCounterAllowed {
			summary.BetaCounterBlockCount++
		}
		if workspaceProviderMonitoringRequestCapExceeded(report) {
			summary.RequestCapExceededCount++
		}
		if report.KillSwitchBlocked {
			summary.KillSwitchBlockedCount++
		}
		if workspaceProviderMonitoringHasFinding(report, WorkspaceProviderFindingProviderCalledWithoutUsage, WorkspaceProviderFindingCompletedMessageWithoutUsage) {
			summary.UsageMissingCount++
		}
		if workspaceProviderMonitoringHasFinding(report, WorkspaceProviderFindingProviderCalledWithoutAudit, WorkspaceProviderFindingCompletedMessageWithoutAudit) {
			summary.AuditMissingCount++
		}
		if report.ReconciliationStatus == WorkspaceProviderReconciliationStatusFinding {
			summary.ReconciliationErrorCount++
		}
	}

	if summary.ProviderCalledCount > 0 {
		summary.ProviderErrorRate = float64(summary.ProviderFailureCount) * 100 / float64(summary.ProviderCalledCount)
	}
	workspaceProviderMonitoringApplySignals(&summary, input.Signals)
	workspaceProviderMonitoringAppendAlerts(&summary, input.Reports, input.Thresholds, maxConsecutiveFailures)
	return summary
}

func workspaceProviderMonitoringReportFailed(report WorkspaceProviderReconciliationReport) bool {
	if report.ErrorCode != "" {
		return true
	}
	return report.AuditStatus == "failed" || report.AssistantMessageStatus == "failed"
}

func workspaceProviderMonitoringReportTimedOut(report WorkspaceProviderReconciliationReport) bool {
	return report.ErrorCode == "timeout" || report.ErrorCode == "provider_timeout" || report.ErrorCode == "workspace_provider_timeout"
}

func workspaceProviderMonitoringRequestCapExceeded(report WorkspaceProviderReconciliationReport) bool {
	for _, reason := range report.BetaCounterBlockReasons {
		switch reason {
		case WorkspaceTextProviderBetaCounterReasonUserLimitExceeded,
			WorkspaceTextProviderBetaCounterReasonProviderExceeded,
			WorkspaceTextProviderBetaCounterReasonModelExceeded,
			WorkspaceTextProviderBetaCounterReasonTestRunExceeded,
			WorkspaceTextProviderStagingQAReasonRequestCapExceeded:
			return true
		}
	}
	return false
}

func workspaceProviderMonitoringHasFinding(report WorkspaceProviderReconciliationReport, findings ...WorkspaceProviderReconciliationFinding) bool {
	for _, existing := range report.Findings {
		for _, expected := range findings {
			if existing == expected {
				return true
			}
		}
	}
	return false
}

func workspaceProviderMonitoringApplySignals(summary *WorkspaceProviderMonitoringSummary, signals []WorkspaceProviderMonitoringSignal) {
	for _, signal := range signals {
		switch signal {
		case WorkspaceProviderMonitoringSignalBrowserDirectProviderCall:
			summary.BrowserDirectProviderCallCount++
		case WorkspaceProviderMonitoringSignalKeyOrTokenLeakage:
			summary.KeyLeakageSignalCount++
		case WorkspaceProviderMonitoringSignalBillingLedgerPayment:
			summary.BillingLedgerPaymentAnomalyCount++
		case WorkspaceProviderMonitoringSignalImageAssetTask:
			summary.ImageAssetTaskUnexpectedCount++
		}
	}
}

func workspaceProviderMonitoringAppendAlerts(summary *WorkspaceProviderMonitoringSummary, reports []WorkspaceProviderReconciliationReport, thresholds WorkspaceProviderMonitoringThresholds, maxConsecutiveFailures int) {
	if thresholds.ErrorRateThresholdPercent > 0 && summary.ProviderCalledCount > 0 && summary.ProviderErrorRate >= float64(thresholds.ErrorRateThresholdPercent) {
		summary.Alerts = append(summary.Alerts, WorkspaceProviderMonitoringAlert{Code: WorkspaceProviderAlertErrorRateExceeded, Severity: WorkspaceProviderMonitoringAlertSeverityWarning, Count: summary.ProviderFailureCount})
	}
	if thresholds.ConsecutiveFailureThreshold > 0 && maxConsecutiveFailures >= thresholds.ConsecutiveFailureThreshold {
		summary.Alerts = append(summary.Alerts, WorkspaceProviderMonitoringAlert{Code: WorkspaceProviderAlertConsecutiveFailures, Severity: WorkspaceProviderMonitoringAlertSeverityWarning, Count: maxConsecutiveFailures})
	}
	if thresholds.TimeoutRateThresholdPercent > 0 && summary.ProviderCalledCount > 0 {
		timeoutRate := float64(summary.ProviderTimeoutCount) * 100 / float64(summary.ProviderCalledCount)
		if timeoutRate >= float64(thresholds.TimeoutRateThresholdPercent) {
			summary.Alerts = append(summary.Alerts, WorkspaceProviderMonitoringAlert{Code: WorkspaceProviderAlertTimeoutRateExceeded, Severity: WorkspaceProviderMonitoringAlertSeverityWarning, Count: summary.ProviderTimeoutCount})
		}
	}
	if thresholds.UsageMissingThreshold > 0 && summary.UsageMissingCount >= thresholds.UsageMissingThreshold {
		summary.Alerts = append(summary.Alerts, WorkspaceProviderMonitoringAlert{Code: WorkspaceProviderAlertUsageMissing, Severity: WorkspaceProviderMonitoringAlertSeverityWarning, Count: summary.UsageMissingCount})
	}
	if thresholds.AuditMissingThreshold > 0 && summary.AuditMissingCount >= thresholds.AuditMissingThreshold {
		summary.Alerts = append(summary.Alerts, WorkspaceProviderMonitoringAlert{Code: WorkspaceProviderAlertAuditMissing, Severity: WorkspaceProviderMonitoringAlertSeverityWarning, Count: summary.AuditMissingCount})
	}

	workspaceProviderMonitoringAppendFindingAlert(summary, reports, WorkspaceProviderFindingProviderCalledWithoutUsage, WorkspaceProviderAlertCalledWithoutUsage, WorkspaceProviderMonitoringAlertSeverityWarning)
	workspaceProviderMonitoringAppendFindingAlert(summary, reports, WorkspaceProviderFindingProviderCalledWithoutAudit, WorkspaceProviderAlertCalledWithoutAudit, WorkspaceProviderMonitoringAlertSeverityWarning)
	workspaceProviderMonitoringAppendFindingAlert(summary, reports, WorkspaceProviderFindingCounterBlockedButProviderCalled, WorkspaceProviderAlertCounterBlockedButCalled, WorkspaceProviderMonitoringAlertSeverityCritical)
	workspaceProviderMonitoringAppendKillSwitchBlockedButCalledAlert(summary, reports)

	if summary.BrowserDirectProviderCallCount > 0 {
		summary.Alerts = append(summary.Alerts, WorkspaceProviderMonitoringAlert{Code: WorkspaceProviderAlertBrowserDirectCall, Severity: WorkspaceProviderMonitoringAlertSeverityCritical, Count: summary.BrowserDirectProviderCallCount})
	}
	if summary.KeyLeakageSignalCount > 0 {
		summary.Alerts = append(summary.Alerts, WorkspaceProviderMonitoringAlert{Code: WorkspaceProviderAlertKeyOrTokenLeakage, Severity: WorkspaceProviderMonitoringAlertSeverityCritical, Count: summary.KeyLeakageSignalCount})
	}
	if summary.BillingLedgerPaymentAnomalyCount > 0 {
		summary.Alerts = append(summary.Alerts, WorkspaceProviderMonitoringAlert{Code: WorkspaceProviderAlertBillingLedgerPaymentAnomaly, Severity: WorkspaceProviderMonitoringAlertSeverityCritical, Count: summary.BillingLedgerPaymentAnomalyCount})
	}
	if summary.ImageAssetTaskUnexpectedCount > 0 {
		summary.Alerts = append(summary.Alerts, WorkspaceProviderMonitoringAlert{Code: WorkspaceProviderAlertImageAssetTaskUnexpected, Severity: WorkspaceProviderMonitoringAlertSeverityWarning, Count: summary.ImageAssetTaskUnexpectedCount})
	}
}

func workspaceProviderMonitoringAppendFindingAlert(summary *WorkspaceProviderMonitoringSummary, reports []WorkspaceProviderReconciliationReport, finding WorkspaceProviderReconciliationFinding, code WorkspaceProviderMonitoringAlertCode, severity WorkspaceProviderMonitoringAlertSeverity) {
	count := 0
	for _, report := range reports {
		if workspaceProviderMonitoringHasFinding(report, finding) {
			count++
		}
	}
	if count == 0 {
		return
	}
	summary.Alerts = append(summary.Alerts, WorkspaceProviderMonitoringAlert{Code: code, Severity: severity, Count: count})
}

func workspaceProviderMonitoringAppendKillSwitchBlockedButCalledAlert(summary *WorkspaceProviderMonitoringSummary, reports []WorkspaceProviderReconciliationReport) {
	count := 0
	for _, report := range reports {
		if report.KillSwitchBlocked && report.ProviderCalled {
			count++
		}
	}
	if count == 0 {
		return
	}
	summary.Alerts = append(summary.Alerts, WorkspaceProviderMonitoringAlert{Code: WorkspaceProviderAlertKillSwitchBlockedButCalled, Severity: WorkspaceProviderMonitoringAlertSeverityCritical, Count: count})
}
