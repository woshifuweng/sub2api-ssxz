package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWorkspaceTextProviderBetaCounterMissingCapsFailClosed(t *testing.T) {
	counter := NewWorkspaceTextProviderBetaRequestCounter(WorkspaceTextProviderGateDecision{
		BetaRequestCaps: WorkspaceTextProviderBetaRequestCaps{},
	})

	decision := counter.Reserve(10, "deepseek-staging", "deepseek-v4-flash")

	require.False(t, decision.Allowed)
	require.Contains(t, decision.Reasons, WorkspaceTextProviderBetaCounterReasonInvalidUserLimit)
	require.Contains(t, decision.Reasons, WorkspaceTextProviderBetaCounterReasonInvalidProviderLimit)
	require.Contains(t, decision.Reasons, WorkspaceTextProviderBetaCounterReasonInvalidModelLimit)
	require.Contains(t, decision.Reasons, WorkspaceTextProviderBetaCounterReasonInvalidTestRunLimit)
	require.Zero(t, counter.TestRunUsed())
}

func TestWorkspaceTextProviderBetaCounterZeroOrNegativeCapsFailClosed(t *testing.T) {
	counter := NewWorkspaceTextProviderBetaRequestCounter(WorkspaceTextProviderGateDecision{
		BetaRequestCaps: WorkspaceTextProviderBetaRequestCaps{
			DailyRequestCap:    0,
			TestRunRequestCap:  -1,
			ProviderRequestCap: 0,
			ModelRequestCap:    -1,
		},
	})

	decision := counter.Reserve(10, "deepseek-staging", "deepseek-v4-flash")

	require.False(t, decision.Allowed)
	require.Contains(t, decision.Reasons, WorkspaceTextProviderBetaCounterReasonInvalidUserLimit)
	require.Contains(t, decision.Reasons, WorkspaceTextProviderBetaCounterReasonInvalidProviderLimit)
	require.Contains(t, decision.Reasons, WorkspaceTextProviderBetaCounterReasonInvalidModelLimit)
	require.Contains(t, decision.Reasons, WorkspaceTextProviderBetaCounterReasonInvalidTestRunLimit)
	require.Zero(t, counter.TestRunUsed())
}

func TestWorkspaceTextProviderBetaCounterBlocksByDimension(t *testing.T) {
	cases := []struct {
		name   string
		caps   WorkspaceTextProviderBetaRequestCaps
		reason string
	}{
		{
			name: "user daily cap",
			caps: WorkspaceTextProviderBetaRequestCaps{
				DailyRequestCap:    1,
				TestRunRequestCap:  10,
				ProviderRequestCap: 10,
				ModelRequestCap:    10,
			},
			reason: WorkspaceTextProviderBetaCounterReasonUserLimitExceeded,
		},
		{
			name: "provider daily cap",
			caps: WorkspaceTextProviderBetaRequestCaps{
				DailyRequestCap:    10,
				TestRunRequestCap:  10,
				ProviderRequestCap: 1,
				ModelRequestCap:    10,
			},
			reason: WorkspaceTextProviderBetaCounterReasonProviderExceeded,
		},
		{
			name: "model daily cap",
			caps: WorkspaceTextProviderBetaRequestCaps{
				DailyRequestCap:    10,
				TestRunRequestCap:  10,
				ProviderRequestCap: 10,
				ModelRequestCap:    1,
			},
			reason: WorkspaceTextProviderBetaCounterReasonModelExceeded,
		},
		{
			name: "test run cap",
			caps: WorkspaceTextProviderBetaRequestCaps{
				DailyRequestCap:    10,
				TestRunRequestCap:  1,
				ProviderRequestCap: 10,
				ModelRequestCap:    10,
			},
			reason: WorkspaceTextProviderBetaCounterReasonTestRunExceeded,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			counter := NewWorkspaceTextProviderBetaRequestCounter(WorkspaceTextProviderGateDecision{
				BetaRequestCaps: tc.caps,
			})

			first := counter.Reserve(10, "deepseek-staging", "deepseek-v4-flash")
			second := counter.Reserve(10, "deepseek-staging", "deepseek-v4-flash")

			require.True(t, first.Allowed)
			require.False(t, second.Allowed)
			require.Contains(t, second.Reasons, tc.reason)
		})
	}
}

func TestWorkspaceTextProviderBetaCounterUsesDateUserProviderAndModelKeys(t *testing.T) {
	counter := NewWorkspaceTextProviderBetaRequestCounter(WorkspaceTextProviderGateDecision{
		BetaRequestCaps: WorkspaceTextProviderBetaRequestCaps{
			DailyRequestCap:    1,
			TestRunRequestCap:  10,
			ProviderRequestCap: 2,
			ModelRequestCap:    2,
		},
	})
	counter.now = func() time.Time { return time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC) }

	first := counter.Reserve(10, "deepseek-staging", "deepseek-v4-flash")
	second := counter.Reserve(11, "deepseek-staging", "deepseek-v4-flash")
	third := counter.Reserve(10, "deepseek-staging", "deepseek-v4-flash")

	require.True(t, first.Allowed)
	require.True(t, second.Allowed)
	require.False(t, third.Allowed)
	require.Contains(t, third.Reasons, WorkspaceTextProviderBetaCounterReasonUserLimitExceeded)
	require.Equal(t, "2026-06-12", first.Date)
	require.Equal(t, int64(2), counter.TestRunUsed())
}

func TestWorkspaceTextProviderBetaCounterBlocksProviderInAdapter(t *testing.T) {
	repo := newMemoryChatWorkspaceRepo()
	executor := &recordingWorkspaceTextProviderExecutor{
		result: successfulWorkspaceTextProviderResult(),
	}
	adapter := workspaceTextProviderAdapterWithOverrides(executor, func(adapter *WorkspaceTextProviderAdapter) {
		adapter.BetaCounter = NewWorkspaceTextProviderBetaRequestCounter(WorkspaceTextProviderGateDecision{
			BetaRequestCaps: WorkspaceTextProviderBetaRequestCaps{
				DailyRequestCap:    1,
				TestRunRequestCap:  10,
				ProviderRequestCap: 10,
				ModelRequestCap:    10,
			},
		})
	})
	svc := NewChatWorkspaceServiceWithProviderAdapter(repo, adapter)
	conversation, err := svc.CreateConversation(context.Background(), 10, WorkspaceCreateConversationInput{})
	require.NoError(t, err)

	_, firstAssistant, err := svc.AppendMessageWithAssistantResponse(context.Background(), 10, validWorkspaceTextAppendInput(conversation.ID))
	require.NoError(t, err)
	_, secondAssistant, err := svc.AppendMessageWithAssistantResponse(context.Background(), 10, validWorkspaceTextAppendInput(conversation.ID))
	require.NoError(t, err)

	require.Equal(t, 1, executor.calls)
	require.Equal(t, true, firstAssistant.Metadata["beta_counter_allowed"])
	require.Equal(t, WorkspaceAssistantUnavailableContent, secondAssistant.Content)
	require.Equal(t, false, secondAssistant.Metadata["provider_called"])
	require.Equal(t, false, secondAssistant.Metadata["beta_counter_allowed"])
	require.Equal(t, int64(1), secondAssistant.Metadata["beta_user_used"])
	require.Equal(t, int64(1), secondAssistant.Metadata["beta_user_limit"])
	require.Contains(t, secondAssistant.Metadata["beta_counter_block_reasons"], WorkspaceTextProviderBetaCounterReasonUserLimitExceeded)
}

func TestWorkspaceTextProviderBetaAllowlistMissBlocksBeforeCounter(t *testing.T) {
	repo := newMemoryChatWorkspaceRepo()
	executor := &recordingWorkspaceTextProviderExecutor{
		result: successfulWorkspaceTextProviderResult(),
	}
	counter := NewWorkspaceTextProviderBetaRequestCounter(workspaceTextProviderAdapterGateDecisionForTest(10, "gpt-5.5"))
	adapter := workspaceTextProviderAdapterWithOverrides(executor, func(adapter *WorkspaceTextProviderAdapter) {
		adapter.BetaCounter = counter
		adapter.BetaAllowlist.AllowedUserIDs = []int64{99}
	})
	svc := NewChatWorkspaceServiceWithProviderAdapter(repo, adapter)
	conversation, err := svc.CreateConversation(context.Background(), 10, WorkspaceCreateConversationInput{})
	require.NoError(t, err)

	_, assistantMessage, err := svc.AppendMessageWithAssistantResponse(context.Background(), 10, validWorkspaceTextAppendInput(conversation.ID))
	require.NoError(t, err)

	require.Zero(t, executor.calls)
	require.Zero(t, counter.TestRunUsed())
	require.Equal(t, false, assistantMessage.Metadata["provider_called"])
	require.Equal(t, false, assistantMessage.Metadata["beta_allowlist_allowed"])
}

func TestWorkspaceTextProviderBetaCounterSuccessAndFailureKeepReservations(t *testing.T) {
	repo := newMemoryChatWorkspaceRepo()
	executor := &recordingWorkspaceTextProviderExecutor{
		result: successfulWorkspaceTextProviderResult(),
	}
	counter := NewWorkspaceTextProviderBetaRequestCounter(workspaceTextProviderAdapterGateDecisionForTest(10, "gpt-5.5"))
	adapter := workspaceTextProviderAdapterWithOverrides(executor, func(adapter *WorkspaceTextProviderAdapter) {
		adapter.BetaCounter = counter
	})
	svc := NewChatWorkspaceServiceWithProviderAdapter(repo, adapter)
	conversation, err := svc.CreateConversation(context.Background(), 10, WorkspaceCreateConversationInput{})
	require.NoError(t, err)

	_, successAssistant, err := svc.AppendMessageWithAssistantResponse(context.Background(), 10, validWorkspaceTextAppendInput(conversation.ID))
	require.NoError(t, err)
	executor.err = errProviderForBetaCounterTest{}
	_, failedAssistant, err := svc.AppendMessageWithAssistantResponse(context.Background(), 10, validWorkspaceTextAppendInput(conversation.ID))
	require.NoError(t, err)

	require.Equal(t, int64(2), counter.TestRunUsed())
	require.Equal(t, "completed", successAssistant.Metadata["status"])
	require.Equal(t, "failed", failedAssistant.Metadata["status"])
	require.Equal(t, true, failedAssistant.Metadata["provider_called"])
	require.Equal(t, false, failedAssistant.Metadata["billing_touched"])

	payload, err := json.Marshal(failedAssistant)
	require.NoError(t, err)
	body := strings.ToLower(string(payload))
	require.NotContains(t, body, "authorization")
	require.NotContains(t, body, "provider_key")
	require.NotContains(t, body, "cookie")
	require.NotContains(t, body, "secret")
}

func TestWorkspaceTextProviderBetaCounterEarlierGuardsDoNotReserve(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*WorkspaceTextProviderAdapter, *WorkspaceAppendMessageInput)
	}{
		{
			name: "kill switch equivalent feature gate",
			mutate: func(adapter *WorkspaceTextProviderAdapter, _ *WorkspaceAppendMessageInput) {
				adapter.FeatureGateEnabled = false
			},
		},
		{
			name: "missing billing policy",
			mutate: func(adapter *WorkspaceTextProviderAdapter, _ *WorkspaceAppendMessageInput) {
				adapter.BillingPolicy = ""
			},
		},
		{
			name: "invalid intent disabled capability",
			mutate: func(_ *WorkspaceTextProviderAdapter, input *WorkspaceAppendMessageInput) {
				input.Intent = "image_generation"
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := newMemoryChatWorkspaceRepo()
			executor := &recordingWorkspaceTextProviderExecutor{
				result: successfulWorkspaceTextProviderResult(),
			}
			counter := NewWorkspaceTextProviderBetaRequestCounter(workspaceTextProviderAdapterGateDecisionForTest(10, "gpt-5.5"))
			adapter := workspaceTextProviderAdapterWithOverrides(executor, func(adapter *WorkspaceTextProviderAdapter) {
				adapter.BetaCounter = counter
			})
			svc := NewChatWorkspaceServiceWithProviderAdapter(repo, adapter)
			conversation, err := svc.CreateConversation(context.Background(), 10, WorkspaceCreateConversationInput{})
			require.NoError(t, err)
			input := validWorkspaceTextAppendInput(conversation.ID)
			tc.mutate(&adapter, &input)
			svc = NewChatWorkspaceServiceWithProviderAdapter(repo, adapter)

			_, _, err = svc.AppendMessageWithAssistantResponse(context.Background(), 10, input)
			if input.Intent == "image_generation" {
				require.ErrorIs(t, err, ErrWorkspaceCapabilityDisabled)
			} else {
				require.NoError(t, err)
			}

			require.Zero(t, executor.calls)
			require.Zero(t, counter.TestRunUsed())
		})
	}
}

type errProviderForBetaCounterTest struct{}

func (errProviderForBetaCounterTest) Error() string {
	return "provider failure with Authorization: Bearer credential-value"
}
