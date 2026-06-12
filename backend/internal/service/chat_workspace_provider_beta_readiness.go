package service

import (
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

type WorkspaceProviderBetaReadinessCheckStatus string

const (
	WorkspaceProviderBetaReadinessStatusPass WorkspaceProviderBetaReadinessCheckStatus = "pass"
	WorkspaceProviderBetaReadinessStatusFail WorkspaceProviderBetaReadinessCheckStatus = "fail"
	WorkspaceProviderBetaReadinessStatusWarn WorkspaceProviderBetaReadinessCheckStatus = "warn"
)

type WorkspaceProviderBetaReadinessSeverity string

const (
	WorkspaceProviderBetaReadinessSeverityBlocker WorkspaceProviderBetaReadinessSeverity = "blocker"
	WorkspaceProviderBetaReadinessSeverityHigh    WorkspaceProviderBetaReadinessSeverity = "high"
	WorkspaceProviderBetaReadinessSeverityMedium  WorkspaceProviderBetaReadinessSeverity = "medium"
	WorkspaceProviderBetaReadinessSeverityLow     WorkspaceProviderBetaReadinessSeverity = "low"
)

type WorkspaceProviderBetaReadinessInput struct {
	TextProvider                      config.WorkspaceTextProviderConfig
	ProviderKeyPresent                bool
	ProviderBaseURLPresent            bool
	ProviderModelPresent              bool
	ProductionDisabled                bool
	StagingServiceActive              bool
	ReconciliationAvailable           bool
	MonitoringAlertingAvailable       bool
	TemporaryNginxPathRemoved         bool
	ImageAssetTaskDisabled            bool
	BrowserDirectProviderCallExpected bool
}

type WorkspaceProviderBetaReadinessReport struct {
	Ready    bool                                  `json:"ready"`
	Checks   []WorkspaceProviderBetaReadinessCheck `json:"checks"`
	Blockers []string                              `json:"blockers,omitempty"`
	Warnings []string                              `json:"warnings,omitempty"`
	Summary  string                                `json:"summary"`
}

type WorkspaceProviderBetaReadinessCheck struct {
	Name        string                                    `json:"name"`
	Status      WorkspaceProviderBetaReadinessCheckStatus `json:"status"`
	Severity    WorkspaceProviderBetaReadinessSeverity    `json:"severity"`
	Message     string                                    `json:"message"`
	Remediation string                                    `json:"remediation,omitempty"`
}

func BuildWorkspaceProviderBetaReadinessReport(input WorkspaceProviderBetaReadinessInput) WorkspaceProviderBetaReadinessReport {
	builder := workspaceProviderBetaReadinessBuilder{}
	textProvider := input.TextProvider

	builder.require("kill_switch_true_before_gate_on", textProvider.KillSwitch, WorkspaceProviderBetaReadinessSeverityBlocker,
		"workspace text provider kill switch is true before gate-on",
		"WORKSPACE_TEXT_PROVIDER_KILL_SWITCH must be true before readiness approval")
	builder.require("production_disabled", input.ProductionDisabled, WorkspaceProviderBetaReadinessSeverityBlocker,
		"production workspace text provider path is disabled",
		"keep production disabled before limited beta gate-on")
	builder.require("staging_environment_confirmed", workspaceProviderBetaReadinessStagingConfirmed(textProvider, input.StagingServiceActive), WorkspaceProviderBetaReadinessSeverityBlocker,
		"staging or beta runtime is active and non-production",
		"confirm the beta service is active and WORKSPACE_TEXT_PROVIDER_ENVIRONMENT is staging/test/local")
	builder.require("beta_allowlist_enabled", textProvider.BetaAllowlist.Enabled, WorkspaceProviderBetaReadinessSeverityBlocker,
		"beta allowlist is enabled",
		"set WORKSPACE_TEXT_PROVIDER_BETA_ALLOWLIST_ENABLED=true")
	builder.require("beta_allowlist_non_empty", workspaceProviderBetaReadinessAllowlistNonEmpty(textProvider.BetaAllowlist), WorkspaceProviderBetaReadinessSeverityBlocker,
		"beta allowlist has subject, provider, and model constraints",
		"configure beta users or groups plus provider labels and models")
	builder.require("beta_allowed_users_or_groups_present", len(textProvider.BetaAllowlist.AllowedUserIDs) > 0 || len(textProvider.BetaAllowlist.AllowedGroupIDs) > 0, WorkspaceProviderBetaReadinessSeverityBlocker,
		"beta allowlist includes user IDs or group IDs",
		"set WORKSPACE_TEXT_PROVIDER_BETA_ALLOWED_USER_IDS or WORKSPACE_TEXT_PROVIDER_BETA_ALLOWED_GROUP_IDS")
	builder.require("beta_allowed_provider_labels_present", len(textProvider.BetaAllowlist.AllowedProviderLabels) > 0, WorkspaceProviderBetaReadinessSeverityBlocker,
		"beta allowlist includes provider labels",
		"set WORKSPACE_TEXT_PROVIDER_BETA_ALLOWED_PROVIDER_LABELS")
	builder.require("beta_allowed_models_present", len(textProvider.BetaAllowlist.AllowedModels) > 0, WorkspaceProviderBetaReadinessSeverityBlocker,
		"beta allowlist includes models",
		"set WORKSPACE_TEXT_PROVIDER_BETA_ALLOWED_MODELS")
	builder.require("beta_request_caps_present", workspaceProviderBetaReadinessCapsPresent(textProvider.BetaRequestCaps), WorkspaceProviderBetaReadinessSeverityBlocker,
		"beta request caps are configured",
		"configure all WORKSPACE_TEXT_PROVIDER_BETA_*_REQUEST_CAP values")
	builder.require("beta_request_caps_positive", workspaceProviderBetaReadinessCapsPositive(textProvider.BetaRequestCaps), WorkspaceProviderBetaReadinessSeverityBlocker,
		"beta request caps are positive",
		"all beta request cap values must be greater than zero")
	builder.require("billing_eligibility_known", textProvider.BillingEligibilityKnown, WorkspaceProviderBetaReadinessSeverityBlocker,
		"billing eligibility is known",
		"set WORKSPACE_TEXT_PROVIDER_BILLING_ELIGIBILITY_KNOWN=true only after backend verification")
	builder.require("billing_eligible", textProvider.BillingEligible, WorkspaceProviderBetaReadinessSeverityBlocker,
		"billing eligibility is positive for the beta path",
		"set WORKSPACE_TEXT_PROVIDER_BILLING_ELIGIBLE=true only for the controlled beta context")
	builder.require("billing_policy_present", workspaceProviderBetaReadinessPolicyPresent(textProvider.BillingPolicy), WorkspaceProviderBetaReadinessSeverityBlocker,
		"billing policy is configured",
		"set WORKSPACE_TEXT_PROVIDER_BILLING_POLICY")
	builder.require("usage_policy_present", workspaceProviderBetaReadinessPolicyPresent(textProvider.UsagePolicy), WorkspaceProviderBetaReadinessSeverityBlocker,
		"usage policy is configured",
		"set WORKSPACE_TEXT_PROVIDER_USAGE_POLICY")
	builder.require("failure_policy_present", workspaceProviderBetaReadinessPolicyPresent(textProvider.FailurePolicy), WorkspaceProviderBetaReadinessSeverityBlocker,
		"failure policy is configured",
		"set WORKSPACE_TEXT_PROVIDER_FAILURE_POLICY")
	builder.require("low_cost_model_allowlist_present", len(textProvider.LowCostModelAllowlist) > 0, WorkspaceProviderBetaReadinessSeverityBlocker,
		"low-cost model allowlist is configured",
		"set WORKSPACE_TEXT_PROVIDER_LOW_COST_MODEL_ALLOWLIST")
	builder.require("test_provider_label_present", strings.TrimSpace(textProvider.TestProviderLabel) != "", WorkspaceProviderBetaReadinessSeverityBlocker,
		"test provider label is configured",
		"set WORKSPACE_TEXT_PROVIDER_TEST_PROVIDER_LABEL")
	builder.require("max_requests_per_test_run_positive", textProvider.MaxRequestsPerTestRun > 0, WorkspaceProviderBetaReadinessSeverityBlocker,
		"max requests per test run is positive",
		"set WORKSPACE_TEXT_PROVIDER_MAX_REQUESTS_PER_TEST_RUN greater than zero")
	builder.require("provider_key_present_server_side", input.ProviderKeyPresent, WorkspaceProviderBetaReadinessSeverityBlocker,
		"provider key is present server-side",
		"configure the provider key only in the backend secret store or server environment")
	builder.require("provider_base_url_present_server_side", input.ProviderBaseURLPresent, WorkspaceProviderBetaReadinessSeverityBlocker,
		"provider base URL is present server-side",
		"configure provider base URL in server-side runtime config")
	builder.require("provider_model_present_server_side", input.ProviderModelPresent, WorkspaceProviderBetaReadinessSeverityBlocker,
		"provider model is present server-side",
		"configure provider model in server-side runtime config")
	builder.require("reconciliation_available", input.ReconciliationAvailable, WorkspaceProviderBetaReadinessSeverityHigh,
		"workspace provider reconciliation report is available",
		"deploy reconciliation helper before beta gate-on")
	builder.require("monitoring_alerting_available", input.MonitoringAlertingAvailable, WorkspaceProviderBetaReadinessSeverityHigh,
		"workspace provider monitoring alerts are available",
		"deploy monitoring alert helper before beta gate-on")
	builder.require("temporary_nginx_path_removed", input.TemporaryNginxPathRemoved, WorkspaceProviderBetaReadinessSeverityBlocker,
		"temporary Nginx staging path is removed",
		"remove temporary staging proxy paths and verify old URLs return 404")
	builder.require("no_image_asset_task_scope", input.ImageAssetTaskDisabled, WorkspaceProviderBetaReadinessSeverityBlocker,
		"image, asset, and task paths are outside beta scope",
		"keep image generation, asset upload, and image tasks disabled for text provider beta")
	builder.require("no_browser_direct_provider_call_expected", !input.BrowserDirectProviderCallExpected, WorkspaceProviderBetaReadinessSeverityBlocker,
		"browser direct provider calls are not expected",
		"route provider calls through the backend only")
	builder.require("no_secret_in_report", true, WorkspaceProviderBetaReadinessSeverityBlocker,
		"readiness report accepts only safe booleans and config summaries",
		"do not pass credential values or full prompts into the report")

	return builder.report()
}

type workspaceProviderBetaReadinessBuilder struct {
	checks []WorkspaceProviderBetaReadinessCheck
}

func (builder *workspaceProviderBetaReadinessBuilder) require(name string, passed bool, severity WorkspaceProviderBetaReadinessSeverity, message, remediation string) {
	check := WorkspaceProviderBetaReadinessCheck{
		Name:     name,
		Severity: severity,
		Message:  message,
	}
	if passed {
		check.Status = WorkspaceProviderBetaReadinessStatusPass
	} else {
		check.Status = WorkspaceProviderBetaReadinessStatusFail
		check.Message = workspaceProviderBetaReadinessFailureMessage(message)
		check.Remediation = remediation
	}
	builder.checks = append(builder.checks, check)
}

func (builder workspaceProviderBetaReadinessBuilder) report() WorkspaceProviderBetaReadinessReport {
	report := WorkspaceProviderBetaReadinessReport{
		Ready:  true,
		Checks: append([]WorkspaceProviderBetaReadinessCheck(nil), builder.checks...),
	}
	for _, check := range report.Checks {
		if check.Status != WorkspaceProviderBetaReadinessStatusPass {
			report.Ready = false
			if check.Severity == WorkspaceProviderBetaReadinessSeverityBlocker {
				report.Blockers = append(report.Blockers, check.Name)
			} else {
				report.Warnings = append(report.Warnings, check.Name)
			}
		}
	}
	if report.Ready {
		report.Summary = fmt.Sprintf("ready: %d checks passed", len(report.Checks))
	} else {
		report.Summary = fmt.Sprintf("not ready: %d blocker(s), %d warning(s)", len(report.Blockers), len(report.Warnings))
	}
	return report
}

func workspaceProviderBetaReadinessFailureMessage(message string) string {
	if strings.TrimSpace(message) == "" {
		return "required readiness check failed"
	}
	return "missing or unsafe: " + message
}

func workspaceProviderBetaReadinessStagingConfirmed(textProvider config.WorkspaceTextProviderConfig, stagingServiceActive bool) bool {
	return stagingServiceActive && isWorkspaceTextProviderNonProductionEnvironment(textProvider.Environment)
}

func workspaceProviderBetaReadinessAllowlistNonEmpty(allowlist config.WorkspaceTextProviderBetaConfig) bool {
	return (len(allowlist.AllowedUserIDs) > 0 || len(allowlist.AllowedGroupIDs) > 0) &&
		len(allowlist.AllowedProviderLabels) > 0 &&
		len(allowlist.AllowedModels) > 0
}

func workspaceProviderBetaReadinessCapsPresent(caps config.WorkspaceTextProviderBetaRequestCapConfig) bool {
	return caps.DailyRequestCap != 0 &&
		caps.TestRunRequestCap != 0 &&
		caps.ProviderRequestCap != 0 &&
		caps.ModelRequestCap != 0
}

func workspaceProviderBetaReadinessCapsPositive(caps config.WorkspaceTextProviderBetaRequestCapConfig) bool {
	return caps.DailyRequestCap > 0 &&
		caps.TestRunRequestCap > 0 &&
		caps.ProviderRequestCap > 0 &&
		caps.ModelRequestCap > 0
}

func workspaceProviderBetaReadinessPolicyPresent(policy string) bool {
	return strings.TrimSpace(policy) != ""
}
