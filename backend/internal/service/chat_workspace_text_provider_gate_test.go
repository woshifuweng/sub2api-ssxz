package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestWorkspaceTextProviderGateDefaultFailsClosed(t *testing.T) {
	decision := BuildWorkspaceTextProviderGateDecision(nil)

	require.False(t, decision.Enabled)
	require.True(t, decision.KillSwitchActive)
	require.Contains(t, decision.Reasons, WorkspaceTextProviderGateReasonMissingConfig)
}

func TestWorkspaceTextProviderGateBlocksProductionByDefault(t *testing.T) {
	cfg := workspaceTextProviderGateConfig()
	cfg.Log.Environment = "production"

	decision := BuildWorkspaceTextProviderGateDecision(cfg)

	require.False(t, decision.Enabled)
	require.False(t, decision.StagingAllowed)
	require.Contains(t, decision.Reasons, WorkspaceTextProviderGateReasonProductionEnvironment)
}

func TestWorkspaceTextProviderGateBlocksKillSwitch(t *testing.T) {
	cfg := workspaceTextProviderGateConfig()
	cfg.Workspace.TextProvider.KillSwitch = true

	decision := BuildWorkspaceTextProviderGateDecision(cfg)

	require.False(t, decision.Enabled)
	require.True(t, decision.KillSwitchActive)
	require.Contains(t, decision.Reasons, WorkspaceTextProviderGateReasonKillSwitchActive)
}

func TestWorkspaceTextProviderGateKillSwitchBlocksDeepSeekStagingModel(t *testing.T) {
	cfg := workspaceTextProviderGateConfig()
	cfg.Workspace.TextProvider.KillSwitch = true
	cfg.Workspace.TextProvider.TestProviderLabel = "deepseek-staging"
	cfg.Workspace.TextProvider.LowCostModelAllowlist = []string{"deepseek-v4-flash"}
	repo := newMemoryChatWorkspaceRepo()
	executor := &recordingWorkspaceTextProviderExecutor{
		result: successfulWorkspaceTextProviderResult(),
	}
	adapter := NewWorkspaceTextProviderAdapterFromConfig(cfg, executor)
	svc := NewChatWorkspaceServiceWithProviderAdapter(repo, adapter)
	conversation, err := svc.CreateConversation(context.Background(), 10, WorkspaceCreateConversationInput{})
	require.NoError(t, err)

	input := validWorkspaceTextAppendInput(conversation.ID)
	input.Model = "deepseek-v4-flash"
	_, assistantMessage, err := svc.AppendMessageWithAssistantResponse(context.Background(), 10, input)
	require.NoError(t, err)

	require.Zero(t, executor.calls)
	require.Equal(t, WorkspaceAssistantUnavailableContent, assistantMessage.Content)
	require.Equal(t, false, assistantMessage.Metadata["provider_called"])
	require.Contains(t, assistantMessage.Metadata["execution_block_reasons"], "feature_gate_disabled")
}

func TestWorkspaceTextProviderGateRequiresTestGuardrails(t *testing.T) {
	cfg := workspaceTextProviderGateConfig()
	cfg.Workspace.TextProvider.TestProviderLabel = ""
	cfg.Workspace.TextProvider.LowCostModelAllowlist = nil
	cfg.Workspace.TextProvider.MaxRequestsPerTestRun = 0

	decision := BuildWorkspaceTextProviderGateDecision(cfg)

	require.False(t, decision.Enabled)
	require.Contains(t, decision.Reasons, WorkspaceTextProviderGateReasonMissingProviderLabel)
	require.Contains(t, decision.Reasons, WorkspaceTextProviderGateReasonMissingModelAllowlist)
	require.Contains(t, decision.Reasons, WorkspaceTextProviderGateReasonInvalidRequestLimit)
}

func TestWorkspaceTextProviderGateAllowsOnlyExplicitStagingGuardrails(t *testing.T) {
	cfg := workspaceTextProviderGateConfig()

	decision := BuildWorkspaceTextProviderGateDecision(cfg)

	require.True(t, decision.Enabled)
	require.False(t, decision.KillSwitchActive)
	require.True(t, decision.StagingAllowed)
	require.Empty(t, decision.Reasons)
	require.Equal(t, "staging", decision.Environment)
	require.Equal(t, "staging-low-cost-provider", decision.TestProviderLabel)
	require.Equal(t, []string{"gpt-5.5-mini"}, decision.LowCostModelAllowlist)
	require.Equal(t, 2, decision.MaxRequestsPerTestRun)
}

func TestWorkspaceTextProviderGateWiringStillRequiresBillingSafeContract(t *testing.T) {
	cfg := workspaceTextProviderGateConfig()
	repo := newMemoryChatWorkspaceRepo()
	executor := &recordingWorkspaceTextProviderExecutor{
		result: successfulWorkspaceTextProviderResult(),
	}
	adapter := NewWorkspaceTextProviderAdapterFromConfig(cfg, executor)
	svc := NewChatWorkspaceServiceWithProviderAdapter(repo, adapter)
	conversation, err := svc.CreateConversation(context.Background(), 10, WorkspaceCreateConversationInput{})
	require.NoError(t, err)

	_, assistantMessage, err := svc.AppendMessageWithAssistantResponse(context.Background(), 10, validWorkspaceTextAppendInput(conversation.ID))
	require.NoError(t, err)

	require.Zero(t, executor.calls)
	require.Equal(t, WorkspaceAssistantUnavailableContent, assistantMessage.Content)
	require.Equal(t, false, assistantMessage.Metadata["provider_called"])
	require.Contains(t, assistantMessage.Metadata["execution_block_reasons"], "billing_unknown")
	require.Contains(t, assistantMessage.Metadata["execution_block_reasons"], "billing_policy_missing")
	require.Contains(t, assistantMessage.Metadata["execution_block_reasons"], "usage_policy_missing")
}

func TestWorkspaceTextProviderGateWiringAllowsFakeExecutorWithExplicitSafePolicies(t *testing.T) {
	cfg := workspaceTextProviderGateConfig()
	cfg.Workspace.TextProvider.BillingEligibilityKnown = true
	cfg.Workspace.TextProvider.BillingEligible = true
	cfg.Workspace.TextProvider.BillingPolicy = string(WorkspaceProviderBillingPolicyRecordUsageOnProviderUsage)
	cfg.Workspace.TextProvider.UsagePolicy = string(WorkspaceProviderUsagePolicyRecordProviderReported)
	cfg.Workspace.TextProvider.FailurePolicy = string(WorkspaceProviderFailurePolicyNoChargeOnFailure)
	cfg.Workspace.TextProvider.LowCostModelAllowlist = []string{"gpt-5.5"}
	repo := newMemoryChatWorkspaceRepo()
	executor := &recordingWorkspaceTextProviderExecutor{
		result: successfulWorkspaceTextProviderResult(),
	}
	adapter := NewWorkspaceTextProviderAdapterFromConfig(cfg, executor)
	svc := NewChatWorkspaceServiceWithProviderAdapter(repo, adapter)
	conversation, err := svc.CreateConversation(context.Background(), 10, WorkspaceCreateConversationInput{})
	require.NoError(t, err)

	_, assistantMessage, err := svc.AppendMessageWithAssistantResponse(context.Background(), 10, validWorkspaceTextAppendInput(conversation.ID))
	require.NoError(t, err)

	require.Equal(t, 1, executor.calls)
	require.NotEqual(t, WorkspaceAssistantUnavailableContent, assistantMessage.Content)
	require.Equal(t, true, assistantMessage.Metadata["provider_called"])
	require.Equal(t, "completed", assistantMessage.Metadata["status"])
	require.Equal(t, "record_usage_on_provider_reported_usage", assistantMessage.Metadata["billing_policy"])
	require.Equal(t, "record_provider_reported", assistantMessage.Metadata["usage_policy"])
	require.Equal(t, "provider_failure_no_charge", assistantMessage.Metadata["failure_policy"])
	require.Equal(t, "succeeded", assistantMessage.Metadata["audit_status"])
}

func TestWorkspaceTextProviderGateKillSwitchBlocksEvenWithExplicitSafePolicies(t *testing.T) {
	cfg := workspaceTextProviderGateConfig()
	cfg.Workspace.TextProvider.KillSwitch = true
	cfg.Workspace.TextProvider.BillingEligibilityKnown = true
	cfg.Workspace.TextProvider.BillingEligible = true
	cfg.Workspace.TextProvider.BillingPolicy = string(WorkspaceProviderBillingPolicyRecordUsageOnProviderUsage)
	cfg.Workspace.TextProvider.UsagePolicy = string(WorkspaceProviderUsagePolicyRecordProviderReported)
	cfg.Workspace.TextProvider.FailurePolicy = string(WorkspaceProviderFailurePolicyNoChargeOnFailure)
	cfg.Workspace.TextProvider.LowCostModelAllowlist = []string{"gpt-5.5"}
	repo := newMemoryChatWorkspaceRepo()
	executor := &recordingWorkspaceTextProviderExecutor{
		result: successfulWorkspaceTextProviderResult(),
	}
	adapter := NewWorkspaceTextProviderAdapterFromConfig(cfg, executor)
	svc := NewChatWorkspaceServiceWithProviderAdapter(repo, adapter)
	conversation, err := svc.CreateConversation(context.Background(), 10, WorkspaceCreateConversationInput{})
	require.NoError(t, err)

	_, assistantMessage, err := svc.AppendMessageWithAssistantResponse(context.Background(), 10, validWorkspaceTextAppendInput(conversation.ID))
	require.NoError(t, err)

	require.Zero(t, executor.calls)
	require.Equal(t, WorkspaceAssistantUnavailableContent, assistantMessage.Content)
	require.Equal(t, false, assistantMessage.Metadata["provider_called"])
	require.Contains(t, assistantMessage.Metadata["execution_block_reasons"], "feature_gate_disabled")
}

func workspaceTextProviderGateConfig() *config.Config {
	return &config.Config{
		Log: config.LogConfig{Environment: "staging"},
		Workspace: config.WorkspaceConfig{
			TextProvider: config.WorkspaceTextProviderConfig{
				Enabled:               true,
				KillSwitch:            false,
				StagingOnly:           true,
				TestProviderLabel:     "staging-low-cost-provider",
				LowCostModelAllowlist: []string{"gpt-5.5-mini"},
				MaxRequestsPerTestRun: 2,
			},
		},
	}
}
