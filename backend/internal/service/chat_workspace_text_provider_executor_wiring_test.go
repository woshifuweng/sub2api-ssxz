package service

import (
	"context"
	"errors"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestWorkspaceTextProviderExecutorWiringDefaultRuntimeKeepsExecutorUnavailable(t *testing.T) {
	cfg := workspaceTextProviderExecutorWiringConfig()
	cfg.Workspace.TextProvider.Enabled = false
	providerCalls := 0

	adapter := NewWorkspaceTextProviderAdapterFromConfigWithExecutorProvider(cfg, func(_ *config.Config, _ WorkspaceTextProviderGateDecision) WorkspaceTextProviderExecutor {
		providerCalls++
		return workspaceOpenAICompatibleExecutorForTest(&fakeWorkspaceOpenAICompatibleTextUpstream{})
	})

	require.False(t, adapter.FeatureGateEnabled)
	require.Nil(t, adapter.Executor)
	require.Zero(t, providerCalls)
}

func TestWorkspaceTextProviderExecutorWiringKillSwitchBlocksExecutorFactory(t *testing.T) {
	cfg := workspaceTextProviderExecutorWiringConfig()
	cfg.Workspace.TextProvider.KillSwitch = true
	providerCalls := 0

	adapter := NewWorkspaceTextProviderAdapterFromConfigWithExecutorProvider(cfg, func(_ *config.Config, _ WorkspaceTextProviderGateDecision) WorkspaceTextProviderExecutor {
		providerCalls++
		return workspaceOpenAICompatibleExecutorForTest(&fakeWorkspaceOpenAICompatibleTextUpstream{})
	})
	repo := newMemoryChatWorkspaceRepo()
	svc := NewChatWorkspaceServiceWithProviderAdapter(repo, adapter)
	conversation, err := svc.CreateConversation(context.Background(), 10, WorkspaceCreateConversationInput{})
	require.NoError(t, err)
	input := validWorkspaceTextAppendInput(conversation.ID)
	input.Model = "deepseek-v4-flash"

	_, assistantMessage, err := svc.AppendMessageWithAssistantResponse(context.Background(), 10, input)
	require.NoError(t, err)

	require.Zero(t, providerCalls)
	require.Nil(t, adapter.Executor)
	require.Equal(t, WorkspaceAssistantUnavailableContent, assistantMessage.Content)
	require.Equal(t, false, assistantMessage.Metadata["provider_called"])
	require.Contains(t, assistantMessage.Metadata["execution_block_reasons"], "feature_gate_disabled")
}

func TestWorkspaceTextProviderExecutorWiringFakeOpenAICompatibleSuccess(t *testing.T) {
	cfg := workspaceTextProviderExecutorWiringConfig()
	upstream := &fakeWorkspaceOpenAICompatibleTextUpstream{
		response: WorkspaceOpenAICompatibleExecutionResponse{
			Content:       "STAGING_OK",
			Status:        "succeeded",
			MappedModel:   "deepseek-v4-flash",
			UpstreamModel: "deepseek-v4-flash",
			ProviderName:  "deepseek-staging",
			EndpointLabel: "deepseek-staging",
			ServiceTier:   "staging",
			LatencyMs:     77,
			Usage: UpstreamQualityUsage{
				InputTokens:  4,
				OutputTokens: 2,
				TotalTokens:  6,
			},
		},
	}
	adapter := NewWorkspaceTextProviderAdapterFromConfigWithExecutorProvider(cfg, func(cfg *config.Config, decision WorkspaceTextProviderGateDecision) WorkspaceTextProviderExecutor {
		return NewWorkspaceOpenAICompatibleTextExecutorFromConfig(cfg, decision, upstream)
	})
	repo := newMemoryChatWorkspaceRepo()
	svc := NewChatWorkspaceServiceWithProviderAdapter(repo, adapter)
	conversation, err := svc.CreateConversation(context.Background(), 10, WorkspaceCreateConversationInput{})
	require.NoError(t, err)
	input := validWorkspaceTextAppendInput(conversation.ID)
	input.Model = "deepseek-v4-flash"

	_, assistantMessage, err := svc.AppendMessageWithAssistantResponse(context.Background(), 10, input)
	require.NoError(t, err)

	require.Equal(t, 1, upstream.calls)
	require.NotNil(t, adapter.Executor)
	require.Equal(t, "STAGING_OK", assistantMessage.Content)
	require.Equal(t, WorkspaceMessageStatusCompleted, assistantMessage.Status)
	require.Equal(t, true, assistantMessage.Metadata["provider_called"])
	require.Equal(t, "text", assistantMessage.Metadata["provider_adapter"])
	require.Equal(t, "deepseek-v4-flash", assistantMessage.Metadata["requested_model"])
	require.Equal(t, "deepseek-v4-flash", assistantMessage.Metadata["mapped_model"])
	require.Equal(t, "deepseek-v4-flash", assistantMessage.Metadata["upstream_model"])
	require.Equal(t, "deepseek-staging", assistantMessage.Metadata["provider_name"])
	require.Equal(t, "deepseek-staging", assistantMessage.Metadata["endpoint_label"])
	require.Equal(t, 6, assistantMessage.Metadata["usage_total_tokens"])
	require.Equal(t, "succeeded", assistantMessage.Metadata["audit_status"])
	require.NotEmpty(t, assistantMessage.Metadata["audit_prompt_hash"])
	require.Equal(t, "deepseek-staging", upstream.lastRequest.ProviderLabel)
	require.Equal(t, "openai_compatible", upstream.lastRequest.Platform)
}

func TestWorkspaceTextProviderExecutorWiringMissingPoliciesDoesNotCallFakeUpstream(t *testing.T) {
	cfg := workspaceTextProviderExecutorWiringConfig()
	cfg.Workspace.TextProvider.BillingPolicy = ""
	cfg.Workspace.TextProvider.UsagePolicy = ""
	upstream := &fakeWorkspaceOpenAICompatibleTextUpstream{
		response: WorkspaceOpenAICompatibleExecutionResponse{Content: "should not run"},
	}
	adapter := NewWorkspaceTextProviderAdapterFromConfigWithExecutorProvider(cfg, func(cfg *config.Config, decision WorkspaceTextProviderGateDecision) WorkspaceTextProviderExecutor {
		return NewWorkspaceOpenAICompatibleTextExecutorFromConfig(cfg, decision, upstream)
	})
	repo := newMemoryChatWorkspaceRepo()
	svc := NewChatWorkspaceServiceWithProviderAdapter(repo, adapter)
	conversation, err := svc.CreateConversation(context.Background(), 10, WorkspaceCreateConversationInput{})
	require.NoError(t, err)
	input := validWorkspaceTextAppendInput(conversation.ID)
	input.Model = "deepseek-v4-flash"

	_, assistantMessage, err := svc.AppendMessageWithAssistantResponse(context.Background(), 10, input)
	require.NoError(t, err)

	require.Zero(t, upstream.calls)
	require.Equal(t, WorkspaceAssistantUnavailableContent, assistantMessage.Content)
	require.Equal(t, false, assistantMessage.Metadata["provider_called"])
	require.Contains(t, assistantMessage.Metadata["execution_block_reasons"], "billing_policy_missing")
	require.Contains(t, assistantMessage.Metadata["execution_block_reasons"], "usage_policy_missing")
}

func TestWorkspaceTextProviderExecutorWiringFakeOpenAICompatibleFailureIsSafe(t *testing.T) {
	cfg := workspaceTextProviderExecutorWiringConfig()
	upstream := &fakeWorkspaceOpenAICompatibleTextUpstream{
		err: errors.New("provider failed with Authorization: Bearer sk-secret and https://internal.example"),
	}
	adapter := NewWorkspaceTextProviderAdapterFromConfigWithExecutorProvider(cfg, func(cfg *config.Config, decision WorkspaceTextProviderGateDecision) WorkspaceTextProviderExecutor {
		return NewWorkspaceOpenAICompatibleTextExecutorFromConfig(cfg, decision, upstream)
	})
	repo := newMemoryChatWorkspaceRepo()
	svc := NewChatWorkspaceServiceWithProviderAdapter(repo, adapter)
	conversation, err := svc.CreateConversation(context.Background(), 10, WorkspaceCreateConversationInput{})
	require.NoError(t, err)
	input := validWorkspaceTextAppendInput(conversation.ID)
	input.Model = "deepseek-v4-flash"

	_, assistantMessage, err := svc.AppendMessageWithAssistantResponse(context.Background(), 10, input)
	require.NoError(t, err)

	require.Equal(t, 1, upstream.calls)
	require.Equal(t, WorkspaceAssistantUnavailableContent, assistantMessage.Content)
	require.Equal(t, "failed", assistantMessage.Metadata["status"])
	require.Equal(t, true, assistantMessage.Metadata["provider_called"])
	require.Equal(t, "workspace_provider_error", assistantMessage.Metadata["audit_error_code"])
	assertWorkspaceOpenAICompatiblePayloadSafe(t, assistantMessage)
}

func workspaceTextProviderExecutorWiringConfig() *config.Config {
	return &config.Config{
		Log: config.LogConfig{Environment: "staging"},
		Workspace: config.WorkspaceConfig{
			TextProvider: config.WorkspaceTextProviderConfig{
				Enabled:                 true,
				KillSwitch:              false,
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
					AllowedUserIDs:        []int64{10},
					AllowedGroupIDs:       []int64{20},
					AllowedProviderLabels: []string{"deepseek-staging"},
					AllowedModels:         []string{"deepseek-v4-flash"},
				},
			},
		},
	}
}
