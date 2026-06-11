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
)

type WorkspaceTextProviderGateDecision struct {
	Enabled               bool
	KillSwitchActive      bool
	StagingAllowed        bool
	Environment           string
	TestProviderLabel     string
	LowCostModelAllowlist []string
	MaxRequestsPerTestRun int
	Reasons               []string
}

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
	decision.Enabled = len(decision.Reasons) == 0
	return decision
}

func NewWorkspaceTextProviderAdapterFromConfig(cfg *config.Config, executor WorkspaceTextProviderExecutor) WorkspaceTextProviderAdapter {
	decision := BuildWorkspaceTextProviderGateDecision(cfg)
	return WorkspaceTextProviderAdapter{
		FeatureGateEnabled: decision.Enabled,
		Executor:           executor,
		ProviderName:       firstNonEmptyWorkspaceValue(decision.TestProviderLabel, WorkspaceProviderNameTextAdapter),
		EndpointLabel:      workspaceTextProviderEndpoint,
		ServiceTier:        decision.Environment,
		FailurePolicy:      WorkspaceProviderFailurePolicyFailClosed,
	}
}

func ProvideChatWorkspaceService(repo ChatWorkspaceRepository, cfg *config.Config) *ChatWorkspaceService {
	adapter := NewWorkspaceTextProviderAdapterFromConfig(cfg, nil)
	return NewChatWorkspaceServiceWithProviderAdapter(repo, adapter)
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
