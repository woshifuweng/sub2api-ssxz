package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWorkspaceTextProviderStagingQAMissingHarnessBlocksProvider(t *testing.T) {
	repo := newMemoryChatWorkspaceRepo()
	executor := &recordingWorkspaceTextProviderExecutor{
		result: successfulWorkspaceTextProviderResult(),
	}
	adapter := workspaceTextProviderAdapterWithOverrides(executor, func(adapter *WorkspaceTextProviderAdapter) {
		adapter.StagingQA = nil
	})
	svc := NewChatWorkspaceServiceWithProviderAdapter(repo, adapter)
	conversation, err := svc.CreateConversation(context.Background(), 10, WorkspaceCreateConversationInput{})
	require.NoError(t, err)

	_, assistantMessage, err := svc.AppendMessageWithAssistantResponse(context.Background(), 10, validWorkspaceTextAppendInput(conversation.ID))
	require.NoError(t, err)

	require.Zero(t, executor.calls)
	require.Equal(t, WorkspaceAssistantUnavailableContent, assistantMessage.Content)
	require.Equal(t, false, assistantMessage.Metadata["provider_called"])
	require.Equal(t, false, assistantMessage.Metadata["staging_qa_allowed"])
	require.Contains(t, assistantMessage.Metadata["staging_qa_block_reasons"], WorkspaceTextProviderStagingQAReasonMissingHarness)
}

func TestWorkspaceTextProviderStagingQALowCostAllowlistBlocksProvider(t *testing.T) {
	repo := newMemoryChatWorkspaceRepo()
	executor := &recordingWorkspaceTextProviderExecutor{
		result: successfulWorkspaceTextProviderResult(),
	}
	adapter := workspaceTextProviderAdapterWithOverrides(executor, func(adapter *WorkspaceTextProviderAdapter) {
		adapter.StagingQA = NewWorkspaceTextProviderStagingQA(workspaceTextProviderAdapterGateDecisionForTest(2, "gpt-5.5-mini"))
	})
	svc := NewChatWorkspaceServiceWithProviderAdapter(repo, adapter)
	conversation, err := svc.CreateConversation(context.Background(), 10, WorkspaceCreateConversationInput{})
	require.NoError(t, err)

	_, assistantMessage, err := svc.AppendMessageWithAssistantResponse(context.Background(), 10, validWorkspaceTextAppendInput(conversation.ID))
	require.NoError(t, err)

	require.Zero(t, executor.calls)
	require.Equal(t, WorkspaceAssistantUnavailableContent, assistantMessage.Content)
	require.Equal(t, false, assistantMessage.Metadata["provider_called"])
	require.Equal(t, false, assistantMessage.Metadata["staging_qa_allowed"])
	require.Contains(t, assistantMessage.Metadata["staging_qa_block_reasons"], WorkspaceTextProviderStagingQAReasonModelNotAllowlisted)
}

func TestWorkspaceTextProviderStagingQAEnforcesRequestCap(t *testing.T) {
	repo := newMemoryChatWorkspaceRepo()
	executor := &recordingWorkspaceTextProviderExecutor{
		result: successfulWorkspaceTextProviderResult(),
	}
	adapter := workspaceTextProviderAdapterWithOverrides(executor, func(adapter *WorkspaceTextProviderAdapter) {
		adapter.StagingQA = NewWorkspaceTextProviderStagingQA(workspaceTextProviderAdapterGateDecisionForTest(2, "gpt-5.5"))
	})
	svc := NewChatWorkspaceServiceWithProviderAdapter(repo, adapter)
	conversation, err := svc.CreateConversation(context.Background(), 10, WorkspaceCreateConversationInput{})
	require.NoError(t, err)

	for i := 0; i < 2; i++ {
		_, assistantMessage, err := svc.AppendMessageWithAssistantResponse(context.Background(), 10, validWorkspaceTextAppendInput(conversation.ID))
		require.NoError(t, err)
		require.Equal(t, "completed", assistantMessage.Metadata["status"])
		require.Equal(t, true, assistantMessage.Metadata["provider_called"])
		require.Equal(t, true, assistantMessage.Metadata["staging_qa_allowed"])
	}

	_, assistantMessage, err := svc.AppendMessageWithAssistantResponse(context.Background(), 10, validWorkspaceTextAppendInput(conversation.ID))
	require.NoError(t, err)

	require.Equal(t, 2, executor.calls)
	require.Equal(t, WorkspaceAssistantUnavailableContent, assistantMessage.Content)
	require.Equal(t, false, assistantMessage.Metadata["provider_called"])
	require.Equal(t, false, assistantMessage.Metadata["staging_qa_allowed"])
	require.Equal(t, int64(2), assistantMessage.Metadata["staging_qa_used"])
	require.Equal(t, int64(2), assistantMessage.Metadata["staging_qa_limit"])
	require.Contains(t, assistantMessage.Metadata["staging_qa_block_reasons"], WorkspaceTextProviderStagingQAReasonRequestCapExceeded)
}

func TestWorkspaceTextProviderStagingQAConfigRequiresRuntimeGuardrailsBeforeProvider(t *testing.T) {
	cfg := workspaceTextProviderGateConfig()
	cfg.Workspace.TextProvider.LowCostModelAllowlist = []string{"gpt-5.5"}
	repo := newMemoryChatWorkspaceRepo()
	executor := &recordingWorkspaceTextProviderExecutor{
		result: successfulWorkspaceTextProviderResult(),
	}
	adapter := NewWorkspaceTextProviderAdapterFromConfig(cfg, executor)
	adapter.BillingEligibilityKnown = true
	adapter.BillingEligible = true
	adapter.BillingPolicy = WorkspaceProviderBillingPolicyRecordUsageOnProviderUsage
	adapter.UsagePolicy = WorkspaceProviderUsagePolicyRecordProviderReported
	svc := NewChatWorkspaceServiceWithProviderAdapter(repo, adapter)
	conversation, err := svc.CreateConversation(context.Background(), 10, WorkspaceCreateConversationInput{})
	require.NoError(t, err)

	_, assistantMessage, err := svc.AppendMessageWithAssistantResponse(context.Background(), 10, validWorkspaceTextAppendInput(conversation.ID))
	require.NoError(t, err)

	require.Equal(t, 1, executor.calls)
	require.Equal(t, "completed", assistantMessage.Metadata["status"])
	require.Equal(t, true, assistantMessage.Metadata["staging_qa_allowed"])
	require.Equal(t, "staging-low-cost-provider", assistantMessage.Metadata["staging_qa_provider"])

	payload, err := json.Marshal(assistantMessage)
	require.NoError(t, err)
	body := strings.ToLower(string(payload))
	require.NotContains(t, body, "provider_key")
	require.NotContains(t, body, "authorization")
	require.NotContains(t, body, "cookie")
	require.NotContains(t, body, "secret")
	require.NotContains(t, body, "base_url")
}
