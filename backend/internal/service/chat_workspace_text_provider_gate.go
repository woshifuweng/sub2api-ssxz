package service

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

const (
	WorkspaceTextProviderGateReasonMissingConfig         = "missing_config"
	WorkspaceTextProviderGateReasonEnabledFalse          = "enabled_false"
	WorkspaceTextProviderGateReasonKillSwitchActive      = "kill_switch_active"
	WorkspaceTextProviderGateReasonProductionEnvironment = "production_environment"
	WorkspaceTextProviderGateReasonMissingProviderLabel  = "missing_test_provider_label"
	WorkspaceTextProviderGateReasonMissingModelAllowlist = "missing_low_cost_model_allowlist"
	WorkspaceTextProviderGateReasonInvalidRequestLimit   = "invalid_max_requests_per_test_run"
	WorkspaceTextProviderGateReasonBetaAllowlistDisabled = "beta_allowlist_disabled"
	WorkspaceTextProviderGateReasonMissingBetaSubjects   = "missing_beta_allowlist_subjects"
	WorkspaceTextProviderGateReasonMissingBetaProviders  = "missing_beta_provider_labels"
	WorkspaceTextProviderGateReasonMissingBetaModels     = "missing_beta_models"
	WorkspaceTextProviderBetaReasonGateDisabled          = "beta_gate_disabled"
	WorkspaceTextProviderBetaReasonSubjectNotAllowed     = "beta_subject_not_allowed"
	WorkspaceTextProviderBetaReasonProviderNotAllowed    = "beta_provider_label_not_allowed"
	WorkspaceTextProviderBetaReasonModelNotAllowed       = "beta_model_not_allowed"
)

type WorkspaceTextProviderGateDecision struct {
	Enabled               bool
	KillSwitchActive      bool
	StagingAllowed        bool
	Environment           string
	TestProviderLabel     string
	LowCostModelAllowlist []string
	MaxRequestsPerTestRun int
	BetaAllowlist         WorkspaceTextProviderBetaAllowlist
	BetaRequestCaps       WorkspaceTextProviderBetaRequestCaps
	Reasons               []string
}

type WorkspaceTextProviderBetaAllowlist struct {
	Enabled               bool
	AllowedUserIDs        []int64
	AllowedGroupIDs       []int64
	AllowedProviderLabels []string
	AllowedModels         []string
}

type WorkspaceTextProviderBetaAllowlistDecision struct {
	Allowed       bool
	Reasons       []string
	UserID        int64
	GroupIDs      []int64
	ProviderLabel string
	Model         string
}

type WorkspaceTextProviderBetaRequestCaps struct {
	DailyRequestCap    int
	TestRunRequestCap  int
	ProviderRequestCap int
	ModelRequestCap    int
}

type WorkspaceTextProviderExecutorProvider func(cfg *config.Config, decision WorkspaceTextProviderGateDecision) WorkspaceTextProviderExecutor

func BuildWorkspaceTextProviderGateDecision(cfg *config.Config) WorkspaceTextProviderGateDecision {
	decision := WorkspaceTextProviderGateDecision{
		KillSwitchActive: true,
		Environment:      "production",
		Reasons:          []string{WorkspaceTextProviderGateReasonMissingConfig},
	}
	if cfg == nil {
		return decision
	}

	gate := cfg.Workspace.TextProvider
	environment := strings.ToLower(strings.TrimSpace(gate.Environment))
	if environment == "" {
		environment = strings.ToLower(strings.TrimSpace(cfg.Log.Environment))
	}
	if environment == "" {
		environment = "production"
	}

	decision = WorkspaceTextProviderGateDecision{
		KillSwitchActive:      gate.KillSwitch,
		StagingAllowed:        !gate.StagingOnly || isWorkspaceTextProviderNonProductionEnvironment(environment),
		Environment:           environment,
		TestProviderLabel:     strings.TrimSpace(gate.TestProviderLabel),
		LowCostModelAllowlist: cloneWorkspaceStringSlice(gate.LowCostModelAllowlist),
		MaxRequestsPerTestRun: gate.MaxRequestsPerTestRun,
		BetaAllowlist: WorkspaceTextProviderBetaAllowlist{
			Enabled:               gate.BetaAllowlist.Enabled,
			AllowedUserIDs:        cloneWorkspaceInt64Slice(gate.BetaAllowlist.AllowedUserIDs),
			AllowedGroupIDs:       cloneWorkspaceInt64Slice(gate.BetaAllowlist.AllowedGroupIDs),
			AllowedProviderLabels: cloneWorkspaceStringSlice(gate.BetaAllowlist.AllowedProviderLabels),
			AllowedModels:         cloneWorkspaceStringSlice(gate.BetaAllowlist.AllowedModels),
		},
		BetaRequestCaps: WorkspaceTextProviderBetaRequestCaps{
			DailyRequestCap:    gate.BetaRequestCaps.DailyRequestCap,
			TestRunRequestCap:  gate.BetaRequestCaps.TestRunRequestCap,
			ProviderRequestCap: gate.BetaRequestCaps.ProviderRequestCap,
			ModelRequestCap:    gate.BetaRequestCaps.ModelRequestCap,
		},
	}
	if !gate.Enabled {
		decision.Reasons = append(decision.Reasons, WorkspaceTextProviderGateReasonEnabledFalse)
	}
	if gate.KillSwitch {
		decision.Reasons = append(decision.Reasons, WorkspaceTextProviderGateReasonKillSwitchActive)
	}
	if gate.StagingOnly && !decision.StagingAllowed {
		decision.Reasons = append(decision.Reasons, WorkspaceTextProviderGateReasonProductionEnvironment)
	}
	if decision.TestProviderLabel == "" {
		decision.Reasons = append(decision.Reasons, WorkspaceTextProviderGateReasonMissingProviderLabel)
	}
	if len(decision.LowCostModelAllowlist) == 0 {
		decision.Reasons = append(decision.Reasons, WorkspaceTextProviderGateReasonMissingModelAllowlist)
	}
	if decision.MaxRequestsPerTestRun <= 0 {
		decision.Reasons = append(decision.Reasons, WorkspaceTextProviderGateReasonInvalidRequestLimit)
	}
	if !decision.BetaAllowlist.Enabled {
		decision.Reasons = append(decision.Reasons, WorkspaceTextProviderGateReasonBetaAllowlistDisabled)
	}
	if len(decision.BetaAllowlist.AllowedUserIDs) == 0 && len(decision.BetaAllowlist.AllowedGroupIDs) == 0 {
		decision.Reasons = append(decision.Reasons, WorkspaceTextProviderGateReasonMissingBetaSubjects)
	}
	if len(decision.BetaAllowlist.AllowedProviderLabels) == 0 {
		decision.Reasons = append(decision.Reasons, WorkspaceTextProviderGateReasonMissingBetaProviders)
	}
	if len(decision.BetaAllowlist.AllowedModels) == 0 {
		decision.Reasons = append(decision.Reasons, WorkspaceTextProviderGateReasonMissingBetaModels)
	}
	decision.Enabled = len(decision.Reasons) == 0
	return decision
}

func NewWorkspaceTextProviderAdapterFromConfig(cfg *config.Config, executor WorkspaceTextProviderExecutor) WorkspaceTextProviderAdapter {
	decision := BuildWorkspaceTextProviderGateDecision(cfg)
	return newWorkspaceTextProviderAdapterFromDecision(cfg, decision, executor)
}

func NewWorkspaceTextProviderAdapterFromConfigWithExecutorProvider(cfg *config.Config, provider WorkspaceTextProviderExecutorProvider) WorkspaceTextProviderAdapter {
	decision := BuildWorkspaceTextProviderGateDecision(cfg)
	var executor WorkspaceTextProviderExecutor
	if decision.Enabled && provider != nil {
		executor = provider(cfg, decision)
	}
	return newWorkspaceTextProviderAdapterFromDecision(cfg, decision, executor)
}

func newWorkspaceTextProviderAdapterFromDecision(cfg *config.Config, decision WorkspaceTextProviderGateDecision, executor WorkspaceTextProviderExecutor) WorkspaceTextProviderAdapter {
	billingPolicy, usagePolicy, failurePolicy := workspaceTextProviderExecutionPoliciesFromConfig(cfg)
	return WorkspaceTextProviderAdapter{
		FeatureGateEnabled:      decision.Enabled,
		Executor:                executor,
		ProviderName:            firstNonEmptyWorkspaceValue(decision.TestProviderLabel, WorkspaceProviderNameTextAdapter),
		EndpointLabel:           workspaceTextProviderEndpoint,
		ServiceTier:             decision.Environment,
		BillingEligibilityKnown: workspaceTextProviderBillingEligibilityKnown(cfg),
		BillingEligible:         workspaceTextProviderBillingEligible(cfg),
		BillingPolicy:           billingPolicy,
		UsagePolicy:             usagePolicy,
		FailurePolicy:           failurePolicy,
		StagingQA:               NewWorkspaceTextProviderStagingQA(decision),
		BetaAllowlist:           decision.BetaAllowlist,
		BetaCounter:             NewWorkspaceTextProviderBetaRequestCounter(decision),
	}
}

func ProvideChatWorkspaceService(repo ChatWorkspaceRepository, cfg *config.Config) *ChatWorkspaceService {
	adapter := NewWorkspaceTextProviderAdapterFromConfigWithExecutorProvider(cfg, workspaceOpenAICompatibleTextExecutorProviderFromConfig)
	return NewChatWorkspaceServiceWithProviderAdapter(repo, adapter)
}

func workspaceOpenAICompatibleTextExecutorProviderFromConfig(cfg *config.Config, decision WorkspaceTextProviderGateDecision) WorkspaceTextProviderExecutor {
	upstream := NewWorkspaceOpenAICompatibleHTTPUpstreamFromConfig(cfg, decision)
	return NewWorkspaceOpenAICompatibleTextExecutorFromConfig(cfg, decision, upstream)
}

func isWorkspaceTextProviderNonProductionEnvironment(environment string) bool {
	switch strings.ToLower(strings.TrimSpace(environment)) {
	case "dev", "development", "local", "test", "testing", "stage", "staging":
		return true
	default:
		return false
	}
}

func cloneWorkspaceStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func cloneWorkspaceInt64Slice(values []int64) []int64 {
	if len(values) == 0 {
		return nil
	}
	out := make([]int64, 0, len(values))
	for _, value := range values {
		if value > 0 {
			out = append(out, value)
		}
	}
	return out
}

func evaluateWorkspaceTextProviderBetaAllowlist(beta WorkspaceTextProviderBetaAllowlist, userID int64, groupIDs []int64, providerLabel, model string) WorkspaceTextProviderBetaAllowlistDecision {
	decision := WorkspaceTextProviderBetaAllowlistDecision{
		UserID:        userID,
		GroupIDs:      cloneWorkspaceInt64Slice(groupIDs),
		ProviderLabel: strings.TrimSpace(providerLabel),
		Model:         strings.TrimSpace(model),
	}
	if !beta.Enabled {
		decision.Reasons = append(decision.Reasons, WorkspaceTextProviderBetaReasonGateDisabled)
	}
	if !workspaceTextProviderSubjectAllowlisted(userID, groupIDs, beta) {
		decision.Reasons = append(decision.Reasons, WorkspaceTextProviderBetaReasonSubjectNotAllowed)
	}
	if !workspaceTextProviderStringAllowlisted(providerLabel, beta.AllowedProviderLabels) {
		decision.Reasons = append(decision.Reasons, WorkspaceTextProviderBetaReasonProviderNotAllowed)
	}
	if !workspaceTextProviderStringAllowlisted(model, beta.AllowedModels) {
		decision.Reasons = append(decision.Reasons, WorkspaceTextProviderBetaReasonModelNotAllowed)
	}
	decision.Allowed = len(decision.Reasons) == 0
	return decision
}

func workspaceTextProviderSubjectAllowlisted(userID int64, groupIDs []int64, beta WorkspaceTextProviderBetaAllowlist) bool {
	if workspaceTextProviderInt64Allowlisted(userID, beta.AllowedUserIDs) {
		return true
	}
	for _, groupID := range groupIDs {
		if workspaceTextProviderInt64Allowlisted(groupID, beta.AllowedGroupIDs) {
			return true
		}
	}
	return false
}

func workspaceTextProviderInt64Allowlisted(value int64, allowlist []int64) bool {
	if value <= 0 || len(allowlist) == 0 {
		return false
	}
	for _, allowed := range allowlist {
		if value == allowed {
			return true
		}
	}
	return false
}

func workspaceTextProviderStringAllowlisted(value string, allowlist []string) bool {
	value = strings.TrimSpace(value)
	if value == "" || len(allowlist) == 0 {
		return false
	}
	for _, allowed := range allowlist {
		if strings.EqualFold(value, strings.TrimSpace(allowed)) {
			return true
		}
	}
	return false
}

func workspaceTextProviderBetaAllowlistMetadata(decision WorkspaceTextProviderBetaAllowlistDecision) map[string]any {
	return map[string]any{
		"beta_allowlist_allowed":       decision.Allowed,
		"beta_allowlist_block_reasons": decision.Reasons,
		"beta_allowlist_provider":      decision.ProviderLabel,
		"beta_allowlist_model":         decision.Model,
	}
}

func workspaceTextProviderBillingEligibilityKnown(cfg *config.Config) bool {
	return cfg != nil && cfg.Workspace.TextProvider.BillingEligibilityKnown
}

func workspaceTextProviderBillingEligible(cfg *config.Config) bool {
	return cfg != nil && cfg.Workspace.TextProvider.BillingEligible
}

func workspaceTextProviderExecutionPoliciesFromConfig(cfg *config.Config) (WorkspaceProviderBillingPolicy, WorkspaceProviderUsagePolicy, WorkspaceProviderFailurePolicy) {
	if cfg == nil {
		return "", "", WorkspaceProviderFailurePolicyFailClosed
	}
	gate := cfg.Workspace.TextProvider
	return workspaceProviderBillingPolicyFromConfig(gate.BillingPolicy),
		workspaceProviderUsagePolicyFromConfig(gate.UsagePolicy),
		workspaceProviderFailurePolicyFromConfig(gate.FailurePolicy)
}

func workspaceProviderBillingPolicyFromConfig(value string) WorkspaceProviderBillingPolicy {
	switch policy := WorkspaceProviderBillingPolicy(strings.ToLower(strings.TrimSpace(value))); policy {
	case WorkspaceProviderBillingPolicyPrecheckOnly,
		WorkspaceProviderBillingPolicyRecordUsageAfterSuccess,
		WorkspaceProviderBillingPolicyRecordUsageOnProviderUsage,
		WorkspaceProviderBillingPolicyNoBilling:
		return policy
	default:
		return ""
	}
}

func workspaceProviderUsagePolicyFromConfig(value string) WorkspaceProviderUsagePolicy {
	switch policy := WorkspaceProviderUsagePolicy(strings.ToLower(strings.TrimSpace(value))); policy {
	case WorkspaceProviderUsagePolicyAuditOnly,
		WorkspaceProviderUsagePolicyRecordAfterSuccess,
		WorkspaceProviderUsagePolicyRecordProviderReported:
		return policy
	default:
		return ""
	}
}

func workspaceProviderFailurePolicyFromConfig(value string) WorkspaceProviderFailurePolicy {
	switch policy := WorkspaceProviderFailurePolicy(strings.ToLower(strings.TrimSpace(value))); policy {
	case WorkspaceProviderFailurePolicyNoChargeOnFailure,
		WorkspaceProviderFailurePolicyReconcileRequired,
		WorkspaceProviderFailurePolicyFailClosed:
		return policy
	default:
		return WorkspaceProviderFailurePolicyFailClosed
	}
}
