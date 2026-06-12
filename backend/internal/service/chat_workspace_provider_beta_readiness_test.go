package service

import (
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestWorkspaceProviderBetaReadinessAllRequiredChecksPass(t *testing.T) {
	report := BuildWorkspaceProviderBetaReadinessReport(workspaceProviderBetaReadinessValidInput())

	require.True(t, report.Ready)
	require.Empty(t, report.Blockers)
	require.Empty(t, report.Warnings)
	require.Equal(t, "ready: 27 checks passed", report.Summary)
	require.True(t, workspaceProviderBetaReadinessHasCheck(report, "no_secret_in_report", WorkspaceProviderBetaReadinessStatusPass))
}

func TestWorkspaceProviderBetaReadinessBlocksUnsafeGateOnInputs(t *testing.T) {
	tests := []struct {
		name      string
		mutate    func(*WorkspaceProviderBetaReadinessInput)
		checkName string
	}{
		{
			name: "kill switch false",
			mutate: func(input *WorkspaceProviderBetaReadinessInput) {
				input.TextProvider.KillSwitch = false
			},
			checkName: "kill_switch_true_before_gate_on",
		},
		{
			name: "production enabled",
			mutate: func(input *WorkspaceProviderBetaReadinessInput) {
				input.ProductionDisabled = false
			},
			checkName: "production_disabled",
		},
		{
			name: "beta allowlist disabled",
			mutate: func(input *WorkspaceProviderBetaReadinessInput) {
				input.TextProvider.BetaAllowlist.Enabled = false
			},
			checkName: "beta_allowlist_enabled",
		},
		{
			name: "missing beta subjects",
			mutate: func(input *WorkspaceProviderBetaReadinessInput) {
				input.TextProvider.BetaAllowlist.AllowedUserIDs = nil
				input.TextProvider.BetaAllowlist.AllowedGroupIDs = nil
			},
			checkName: "beta_allowed_users_or_groups_present",
		},
		{
			name: "missing provider labels",
			mutate: func(input *WorkspaceProviderBetaReadinessInput) {
				input.TextProvider.BetaAllowlist.AllowedProviderLabels = nil
			},
			checkName: "beta_allowed_provider_labels_present",
		},
		{
			name: "missing models",
			mutate: func(input *WorkspaceProviderBetaReadinessInput) {
				input.TextProvider.BetaAllowlist.AllowedModels = nil
			},
			checkName: "beta_allowed_models_present",
		},
		{
			name: "missing caps",
			mutate: func(input *WorkspaceProviderBetaReadinessInput) {
				input.TextProvider.BetaRequestCaps.DailyRequestCap = 0
			},
			checkName: "beta_request_caps_present",
		},
		{
			name: "negative caps",
			mutate: func(input *WorkspaceProviderBetaReadinessInput) {
				input.TextProvider.BetaRequestCaps.ModelRequestCap = -1
			},
			checkName: "beta_request_caps_positive",
		},
		{
			name: "missing billing policy",
			mutate: func(input *WorkspaceProviderBetaReadinessInput) {
				input.TextProvider.BillingPolicy = ""
			},
			checkName: "billing_policy_present",
		},
		{
			name: "missing usage policy",
			mutate: func(input *WorkspaceProviderBetaReadinessInput) {
				input.TextProvider.UsagePolicy = ""
			},
			checkName: "usage_policy_present",
		},
		{
			name: "missing failure policy",
			mutate: func(input *WorkspaceProviderBetaReadinessInput) {
				input.TextProvider.FailurePolicy = ""
			},
			checkName: "failure_policy_present",
		},
		{
			name: "missing provider key",
			mutate: func(input *WorkspaceProviderBetaReadinessInput) {
				input.ProviderKeyPresent = false
			},
			checkName: "provider_key_present_server_side",
		},
		{
			name: "temporary nginx path remains",
			mutate: func(input *WorkspaceProviderBetaReadinessInput) {
				input.TemporaryNginxPathRemoved = false
			},
			checkName: "temporary_nginx_path_removed",
		},
		{
			name: "image asset task scope enabled",
			mutate: func(input *WorkspaceProviderBetaReadinessInput) {
				input.ImageAssetTaskDisabled = false
			},
			checkName: "no_image_asset_task_scope",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := workspaceProviderBetaReadinessValidInput()
			tt.mutate(&input)

			report := BuildWorkspaceProviderBetaReadinessReport(input)

			require.False(t, report.Ready)
			require.Contains(t, report.Blockers, tt.checkName)
			require.True(t, workspaceProviderBetaReadinessHasCheck(report, tt.checkName, WorkspaceProviderBetaReadinessStatusFail))
		})
	}
}

func TestWorkspaceProviderBetaReadinessRequiresProviderServerSideConfig(t *testing.T) {
	tests := []struct {
		name      string
		mutate    func(*WorkspaceProviderBetaReadinessInput)
		checkName string
	}{
		{
			name: "missing base URL",
			mutate: func(input *WorkspaceProviderBetaReadinessInput) {
				input.ProviderBaseURLPresent = false
			},
			checkName: "provider_base_url_present_server_side",
		},
		{
			name: "missing model",
			mutate: func(input *WorkspaceProviderBetaReadinessInput) {
				input.ProviderModelPresent = false
			},
			checkName: "provider_model_present_server_side",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := workspaceProviderBetaReadinessValidInput()
			tt.mutate(&input)

			report := BuildWorkspaceProviderBetaReadinessReport(input)

			require.False(t, report.Ready)
			require.Contains(t, report.Blockers, tt.checkName)
		})
	}
}

func TestWorkspaceProviderBetaReadinessRequiresObservabilityHelpers(t *testing.T) {
	input := workspaceProviderBetaReadinessValidInput()
	input.ReconciliationAvailable = false
	input.MonitoringAlertingAvailable = false

	report := BuildWorkspaceProviderBetaReadinessReport(input)

	require.False(t, report.Ready)
	require.Empty(t, report.Blockers)
	require.Contains(t, report.Warnings, "reconciliation_available")
	require.Contains(t, report.Warnings, "monitoring_alerting_available")
	require.True(t, workspaceProviderBetaReadinessHasCheck(report, "reconciliation_available", WorkspaceProviderBetaReadinessStatusFail))
	require.True(t, workspaceProviderBetaReadinessHasCheck(report, "monitoring_alerting_available", WorkspaceProviderBetaReadinessStatusFail))
}

func TestWorkspaceProviderBetaReadinessRequiresNoBrowserDirectProviderCall(t *testing.T) {
	input := workspaceProviderBetaReadinessValidInput()
	input.BrowserDirectProviderCallExpected = true

	report := BuildWorkspaceProviderBetaReadinessReport(input)

	require.False(t, report.Ready)
	require.Contains(t, report.Blockers, "no_browser_direct_provider_call_expected")
}

func TestWorkspaceProviderBetaReadinessRequiresStagingRuntime(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*WorkspaceProviderBetaReadinessInput)
	}{
		{
			name: "staging service inactive",
			mutate: func(input *WorkspaceProviderBetaReadinessInput) {
				input.StagingServiceActive = false
			},
		},
		{
			name: "production environment",
			mutate: func(input *WorkspaceProviderBetaReadinessInput) {
				input.TextProvider.Environment = "production"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := workspaceProviderBetaReadinessValidInput()
			tt.mutate(&input)

			report := BuildWorkspaceProviderBetaReadinessReport(input)

			require.False(t, report.Ready)
			require.Contains(t, report.Blockers, "staging_environment_confirmed")
		})
	}
}

func TestWorkspaceProviderBetaReadinessReportExcludesPromptAndSecrets(t *testing.T) {
	report := BuildWorkspaceProviderBetaReadinessReport(workspaceProviderBetaReadinessValidInput())

	payload, err := json.Marshal(report)
	require.NoError(t, err)
	text := string(payload)

	require.NotContains(t, text, "full sensitive prompt")
	require.NotContains(t, text, "sk-provider-key")
	require.NotContains(t, text, "Authorization")
	require.NotContains(t, text, "Bearer provider-token")
	require.NotContains(t, text, "session-cookie")
}

func workspaceProviderBetaReadinessValidInput() WorkspaceProviderBetaReadinessInput {
	return WorkspaceProviderBetaReadinessInput{
		TextProvider: config.WorkspaceTextProviderConfig{
			Enabled:                 true,
			KillSwitch:              true,
			StagingOnly:             true,
			Environment:             "staging",
			TestProviderLabel:       "deepseek-staging",
			LowCostModelAllowlist:   []string{"deepseek-v4-flash"},
			MaxRequestsPerTestRun:   3,
			BillingEligibilityKnown: true,
			BillingEligible:         true,
			BillingPolicy:           string(WorkspaceProviderBillingPolicyRecordUsageOnProviderUsage),
			UsagePolicy:             string(WorkspaceProviderUsagePolicyRecordProviderReported),
			FailurePolicy:           string(WorkspaceProviderFailurePolicyNoChargeOnFailure),
			BetaAllowlist: config.WorkspaceTextProviderBetaConfig{
				Enabled:               true,
				AllowedUserIDs:        []int64{1001},
				AllowedGroupIDs:       []int64{2002},
				AllowedProviderLabels: []string{"deepseek-staging"},
				AllowedModels:         []string{"deepseek-v4-flash"},
			},
			BetaRequestCaps: config.WorkspaceTextProviderBetaRequestCapConfig{
				DailyRequestCap:    10,
				TestRunRequestCap:  3,
				ProviderRequestCap: 10,
				ModelRequestCap:    10,
			},
		},
		ProviderKeyPresent:                true,
		ProviderBaseURLPresent:            true,
		ProviderModelPresent:              true,
		ProductionDisabled:                true,
		StagingServiceActive:              true,
		ReconciliationAvailable:           true,
		MonitoringAlertingAvailable:       true,
		TemporaryNginxPathRemoved:         true,
		ImageAssetTaskDisabled:            true,
		BrowserDirectProviderCallExpected: false,
	}
}

func workspaceProviderBetaReadinessHasCheck(report WorkspaceProviderBetaReadinessReport, name string, status WorkspaceProviderBetaReadinessCheckStatus) bool {
	for _, check := range report.Checks {
		if check.Name == name && check.Status == status {
			return true
		}
	}
	return false
}
