package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWorkspaceTextProviderAdapterFeatureGateOffDoesNotCallExecutor(t *testing.T) {
	repo := newMemoryChatWorkspaceRepo()
	executor := &recordingWorkspaceTextProviderExecutor{
		result: successfulWorkspaceTextProviderResult(),
	}
	svc := NewChatWorkspaceServiceWithProviderAdapter(repo, WorkspaceTextProviderAdapter{
		Executor:                executor,
		BillingEligibilityKnown: true,
		BillingEligible:         true,
		BillingPolicy:           WorkspaceProviderBillingPolicyRecordUsageOnProviderUsage,
		UsagePolicy:             WorkspaceProviderUsagePolicyRecordProviderReported,
	})
	conversation, err := svc.CreateConversation(context.Background(), 10, WorkspaceCreateConversationInput{})
	require.NoError(t, err)

	_, assistantMessage, err := svc.AppendMessageWithAssistantResponse(context.Background(), 10, validWorkspaceTextAppendInput(conversation.ID))
	require.NoError(t, err)

	require.Zero(t, executor.calls)
	require.Equal(t, WorkspaceAssistantUnavailableContent, assistantMessage.Content)
	require.Equal(t, "unavailable", assistantMessage.Metadata["status"])
	require.Equal(t, false, assistantMessage.Metadata["provider_called"])
	require.Contains(t, assistantMessage.Metadata["execution_block_reasons"], "feature_gate_disabled")
	require.Equal(t, false, assistantMessage.Metadata["billing_touched"])
	require.Equal(t, false, assistantMessage.Metadata["asset_touched"])
}

func TestWorkspaceTextProviderAdapterRequiresBillingAndUsagePolicy(t *testing.T) {
	repo := newMemoryChatWorkspaceRepo()
	executor := &recordingWorkspaceTextProviderExecutor{
		result: successfulWorkspaceTextProviderResult(),
	}
	svc := NewChatWorkspaceServiceWithProviderAdapter(repo, WorkspaceTextProviderAdapter{
		FeatureGateEnabled:      true,
		Executor:                executor,
		BillingEligibilityKnown: true,
		BillingEligible:         true,
	})
	conversation, err := svc.CreateConversation(context.Background(), 10, WorkspaceCreateConversationInput{})
	require.NoError(t, err)

	_, assistantMessage, err := svc.AppendMessageWithAssistantResponse(context.Background(), 10, validWorkspaceTextAppendInput(conversation.ID))
	require.NoError(t, err)

	require.Zero(t, executor.calls)
	require.Equal(t, WorkspaceAssistantUnavailableContent, assistantMessage.Content)
	require.Equal(t, false, assistantMessage.Metadata["provider_called"])
	require.Contains(t, assistantMessage.Metadata["execution_block_reasons"], "billing_policy_missing")
	require.Contains(t, assistantMessage.Metadata["execution_block_reasons"], "usage_policy_missing")
}

func TestWorkspaceTextProviderAdapterCallsFakeExecutorWhenContractAllows(t *testing.T) {
	repo := newMemoryChatWorkspaceRepo()
	executor := &recordingWorkspaceTextProviderExecutor{
		result: successfulWorkspaceTextProviderResult(),
	}
	svc := NewChatWorkspaceServiceWithProviderAdapter(repo, safeWorkspaceTextProviderAdapter(executor))
	conversation, err := svc.CreateConversation(context.Background(), 10, WorkspaceCreateConversationInput{})
	require.NoError(t, err)

	userMessage, assistantMessage, err := svc.AppendMessageWithAssistantResponse(context.Background(), 10, validWorkspaceTextAppendInput(conversation.ID))
	require.NoError(t, err)

	require.NotNil(t, userMessage)
	require.Equal(t, 1, executor.calls)
	require.Equal(t, int64(10), executor.lastInput.UserID)
	require.Equal(t, conversation.ID, executor.lastInput.ConversationID)
	require.Equal(t, userMessage.ID, executor.lastInput.UserMessageID)
	require.Equal(t, "gpt-5.5", executor.lastInput.Model)
	require.Equal(t, "gpt-5.5-mapped", executor.lastInput.MappedModel)
	require.Equal(t, "gpt-5.5-upstream", executor.lastInput.UpstreamModel)
	require.True(t, executor.lastInput.ExecutionContract.CanCallProvider)

	require.Equal(t, "safe fake provider response", assistantMessage.Content)
	require.Equal(t, WorkspaceRoleAssistant, assistantMessage.Role)
	require.Equal(t, WorkspaceMessageStatusCompleted, assistantMessage.Status)
	require.Equal(t, "completed", assistantMessage.Metadata["status"])
	require.Equal(t, true, assistantMessage.Metadata["provider_called"])
	require.Equal(t, true, assistantMessage.Metadata["provider_connected"])
	require.Equal(t, false, assistantMessage.Metadata["billing_touched"])
	require.Equal(t, false, assistantMessage.Metadata["asset_touched"])
	require.Equal(t, "workspace_fake_text_provider", assistantMessage.Metadata["provider_name"])
	require.Equal(t, "gpt-5.5-mapped", assistantMessage.Metadata["mapped_model"])
	require.Equal(t, "gpt-5.5-upstream", assistantMessage.Metadata["upstream_model"])
	require.Equal(t, "workspace-fake-text-endpoint", assistantMessage.Metadata["endpoint_label"])
	require.Equal(t, 42, assistantMessage.Metadata["usage_total_tokens"])
	require.NotEmpty(t, assistantMessage.Metadata["audit_prompt_hash"])
	require.Equal(t, "succeeded", assistantMessage.Metadata["audit_status"])

	messages, err := svc.ListMessages(context.Background(), 10, conversation.ID)
	require.NoError(t, err)
	require.Len(t, messages, 2)
	require.Equal(t, WorkspaceRoleUser, messages[0].Role)
	require.Equal(t, WorkspaceRoleAssistant, messages[1].Role)
}

func TestWorkspaceTextProviderAdapterProviderErrorIsSafe(t *testing.T) {
	repo := newMemoryChatWorkspaceRepo()
	executor := &recordingWorkspaceTextProviderExecutor{
		err: errors.New("provider exploded at https://internal.example with Authorization: Bearer sk-secret"),
	}
	svc := NewChatWorkspaceServiceWithProviderAdapter(repo, safeWorkspaceTextProviderAdapter(executor))
	conversation, err := svc.CreateConversation(context.Background(), 10, WorkspaceCreateConversationInput{})
	require.NoError(t, err)

	_, assistantMessage, err := svc.AppendMessageWithAssistantResponse(context.Background(), 10, validWorkspaceTextAppendInput(conversation.ID))
	require.NoError(t, err)

	require.Equal(t, 1, executor.calls)
	require.Equal(t, WorkspaceAssistantUnavailableContent, assistantMessage.Content)
	require.Equal(t, "failed", assistantMessage.Metadata["status"])
	require.Equal(t, true, assistantMessage.Metadata["provider_called"])
	require.Equal(t, false, assistantMessage.Metadata["billing_touched"])
	require.Equal(t, "workspace_provider_error", assistantMessage.Metadata["audit_error_code"])

	payload, err := json.Marshal(assistantMessage)
	require.NoError(t, err)
	body := strings.ToLower(string(payload))
	require.NotContains(t, body, "sk-secret")
	require.NotContains(t, body, "authorization: bearer")
	require.NotContains(t, body, "internal.example")
}

func TestWorkspaceTextProviderAdapterBlocksUnsafePlansBeforeExecutor(t *testing.T) {
	cases := []struct {
		name    string
		adapter WorkspaceTextProviderAdapter
		input   WorkspaceAppendMessageInput
		reason  string
	}{
		{
			name:    "billing unknown",
			adapter: workspaceTextProviderAdapterWithOverrides(nil, func(adapter *WorkspaceTextProviderAdapter) { adapter.BillingEligibilityKnown = false }),
			input:   validWorkspaceTextAppendInput(1),
			reason:  "billing_unknown",
		},
		{
			name: "arbitrary base url",
			adapter: workspaceTextProviderAdapterWithOverrides(nil, func(adapter *WorkspaceTextProviderAdapter) {
				adapter.EndpointBaseURL = "https://provider.example.com/v1"
			}),
			input:  validWorkspaceTextAppendInput(1),
			reason: "arbitrary_base_url",
		},
		{
			name:    "disabled capability",
			adapter: safeWorkspaceTextProviderAdapter(nil),
			input: func() WorkspaceAppendMessageInput {
				input := validWorkspaceTextAppendInput(1)
				input.Intent = "image_generation"
				return input
			}(),
			reason: "capability_disabled",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := newMemoryChatWorkspaceRepo()
			executor := &recordingWorkspaceTextProviderExecutor{
				result: successfulWorkspaceTextProviderResult(),
			}
			tc.adapter.Executor = executor
			svc := NewChatWorkspaceServiceWithProviderAdapter(repo, tc.adapter)
			conversation, err := svc.CreateConversation(context.Background(), 10, WorkspaceCreateConversationInput{})
			require.NoError(t, err)
			tc.input.ConversationID = conversation.ID

			_, assistantMessage, err := svc.AppendMessageWithAssistantResponse(context.Background(), 10, tc.input)
			if tc.input.Intent == "image_generation" {
				require.ErrorIs(t, err, ErrWorkspaceCapabilityDisabled)
				require.Zero(t, executor.calls)
				return
			}

			require.NoError(t, err)
			require.Zero(t, executor.calls)
			require.Equal(t, false, assistantMessage.Metadata["provider_called"])
			require.Contains(t, assistantMessage.Metadata["execution_block_reasons"], tc.reason)
		})
	}
}

func TestWorkspaceTextProviderAdapterDoesNotLeakPromptOrSecrets(t *testing.T) {
	repo := newMemoryChatWorkspaceRepo()
	executor := &recordingWorkspaceTextProviderExecutor{
		result: successfulWorkspaceTextProviderResult(),
	}
	svc := NewChatWorkspaceServiceWithProviderAdapter(repo, safeWorkspaceTextProviderAdapter(executor))
	conversation, err := svc.CreateConversation(context.Background(), 10, WorkspaceCreateConversationInput{})
	require.NoError(t, err)
	input := validWorkspaceTextAppendInput(conversation.ID)
	input.Content = "Customer campaign Authorization: Bearer sk-secret access_token=abc cookie=session private launch brief"

	_, assistantMessage, err := svc.AppendMessageWithAssistantResponse(context.Background(), 10, input)
	require.NoError(t, err)

	payload, err := json.Marshal(assistantMessage)
	require.NoError(t, err)
	body := strings.ToLower(string(payload))
	require.NotContains(t, body, "sk-secret")
	require.NotContains(t, body, "access_token=abc")
	require.NotContains(t, body, "cookie=session")
	require.NotContains(t, body, "authorization: bearer")
	require.NotContains(t, body, input.Content)
	require.NotContains(t, body, "provider_key")
	require.NotContains(t, body, "base_url")
}

type recordingWorkspaceTextProviderExecutor struct {
	calls     int
	lastInput WorkspaceTextProviderExecutionInput
	result    WorkspaceTextProviderExecutionResult
	err       error
}

func (e *recordingWorkspaceTextProviderExecutor) ExecuteWorkspaceTextProvider(_ context.Context, input WorkspaceTextProviderExecutionInput) (WorkspaceTextProviderExecutionResult, error) {
	e.calls++
	e.lastInput = input
	if e.err != nil {
		return WorkspaceTextProviderExecutionResult{}, e.err
	}
	return e.result, nil
}

func validWorkspaceTextAppendInput(conversationID int64) WorkspaceAppendMessageInput {
	return WorkspaceAppendMessageInput{
		ConversationID: conversationID,
		MessageType:    WorkspaceMessageTypeText,
		Role:           WorkspaceRoleUser,
		Content:        "hello workspace text provider",
		Model:          "gpt-5.5",
		Intent:         WorkspaceIntentChat,
	}
}

func safeWorkspaceTextProviderAdapter(executor WorkspaceTextProviderExecutor) WorkspaceTextProviderAdapter {
	return workspaceTextProviderAdapterWithOverrides(executor, nil)
}

func workspaceTextProviderAdapterWithOverrides(executor WorkspaceTextProviderExecutor, mutate func(*WorkspaceTextProviderAdapter)) WorkspaceTextProviderAdapter {
	adapter := WorkspaceTextProviderAdapter{
		FeatureGateEnabled:      true,
		Executor:                executor,
		RequestID:               "req_workspace_text_provider_test",
		ProviderName:            "workspace_fake_text_provider",
		EndpointLabel:           "workspace-fake-text-endpoint",
		ServiceTier:             "test",
		MappedModel:             "gpt-5.5-mapped",
		UpstreamModel:           "gpt-5.5-upstream",
		BillingEligibilityKnown: true,
		BillingEligible:         true,
		BillingPolicy:           WorkspaceProviderBillingPolicyRecordUsageOnProviderUsage,
		UsagePolicy:             WorkspaceProviderUsagePolicyRecordProviderReported,
		FailurePolicy:           WorkspaceProviderFailurePolicyFailClosed,
		BetaAllowlist: WorkspaceTextProviderBetaAllowlist{
			Enabled:               true,
			AllowedUserIDs:        []int64{10},
			AllowedProviderLabels: []string{"workspace_fake_text_provider"},
			AllowedModels:         []string{"gpt-5.5"},
		},
		StagingQA: NewWorkspaceTextProviderStagingQA(workspaceTextProviderAdapterGateDecisionForTest(100, "gpt-5.5")),
	}
	if mutate != nil {
		mutate(&adapter)
	}
	return adapter
}

func workspaceTextProviderAdapterGateDecisionForTest(limit int, models ...string) WorkspaceTextProviderGateDecision {
	return WorkspaceTextProviderGateDecision{
		Enabled:               true,
		StagingAllowed:        true,
		Environment:           "test",
		TestProviderLabel:     "workspace_fake_text_provider",
		LowCostModelAllowlist: models,
		MaxRequestsPerTestRun: limit,
	}
}

func successfulWorkspaceTextProviderResult() WorkspaceTextProviderExecutionResult {
	return WorkspaceTextProviderExecutionResult{
		Content:       "safe fake provider response",
		Model:         "gpt-5.5",
		MappedModel:   "gpt-5.5-mapped",
		UpstreamModel: "gpt-5.5-upstream",
		ProviderName:  "workspace_fake_text_provider",
		EndpointLabel: "workspace-fake-text-endpoint",
		ServiceTier:   "test",
		LatencyMs:     123,
		TokenUsage: UpstreamQualityUsage{
			InputTokens:  12,
			OutputTokens: 30,
			TotalTokens:  42,
		},
	}
}
