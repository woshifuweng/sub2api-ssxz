package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWorkspaceOpenAICompatibleTextExecutorSuccessReturnsContentUsageAndAudit(t *testing.T) {
	upstream := &fakeWorkspaceOpenAICompatibleTextUpstream{
		response: WorkspaceOpenAICompatibleExecutionResponse{
			Content:       "STAGING_OK",
			Status:        "succeeded",
			MappedModel:   "deepseek-v4-flash",
			UpstreamModel: "deepseek-v4-flash",
			ProviderName:  "deepseek-staging",
			EndpointLabel: "deepseek-staging",
			ServiceTier:   "staging",
			LatencyMs:     88,
			Usage: UpstreamQualityUsage{
				InputTokens:  3,
				OutputTokens: 2,
				TotalTokens:  5,
			},
		},
	}
	executor := workspaceOpenAICompatibleExecutorForTest(upstream)
	input := validWorkspaceOpenAICompatibleExecutionInput()

	result, err := executor.ExecuteWorkspaceTextProvider(context.Background(), input)
	require.NoError(t, err)

	require.Equal(t, 1, upstream.calls)
	require.Equal(t, "STAGING_OK", result.Content)
	require.Equal(t, "deepseek-v4-flash", result.Model)
	require.Equal(t, "deepseek-v4-flash", result.MappedModel)
	require.Equal(t, "deepseek-v4-flash", result.UpstreamModel)
	require.Equal(t, "deepseek-staging", result.ProviderName)
	require.Equal(t, "deepseek-staging", result.EndpointLabel)
	require.Equal(t, 88, int(result.LatencyMs))
	require.Equal(t, 5, result.TokenUsage.TotalTokens)
	require.Equal(t, true, result.Metadata["provider_called"])
	require.Equal(t, false, result.Metadata["billing_touched"])
	require.Equal(t, false, result.Metadata["asset_touched"])
	require.Equal(t, "openai_compatible", result.Metadata["provider_adapter"])
	require.Equal(t, "record_usage_on_provider_reported_usage", result.Metadata["billing_policy"])
	require.Equal(t, "record_provider_reported", result.Metadata["usage_policy"])
	require.Equal(t, "provider_failure_no_charge", result.Metadata["failure_policy"])
	require.NotEmpty(t, result.Metadata["audit_prompt_hash"])
	require.Equal(t, "succeeded", result.Metadata["audit_status"])

	require.Equal(t, "openai_compatible", upstream.lastRequest.Platform)
	require.Equal(t, "deepseek-staging", upstream.lastRequest.ProviderLabel)
	require.Equal(t, "provider_failure_no_charge", string(upstream.lastRequest.FailurePolicy))
	require.Len(t, upstream.lastRequest.Messages, 1)
	require.Equal(t, input.Content, upstream.lastRequest.Messages[0].Content)
	require.NotEmpty(t, upstream.lastRequest.AuditContext.PromptHash)

	assertWorkspaceOpenAICompatiblePayloadSafe(t, result)
}

func TestWorkspaceOpenAICompatibleTextExecutorFailureReturnsSanitizedError(t *testing.T) {
	upstream := &fakeWorkspaceOpenAICompatibleTextUpstream{
		err: errors.New("provider failed at https://internal.example with Authorization: Bearer sk-secret-token and sql stack"),
	}
	executor := workspaceOpenAICompatibleExecutorForTest(upstream)

	result, err := executor.ExecuteWorkspaceTextProvider(context.Background(), validWorkspaceOpenAICompatibleExecutionInput())
	require.Error(t, err)
	require.ErrorIs(t, err, errWorkspaceOpenAICompatibleProviderFailed)
	require.Equal(t, 1, upstream.calls)
	require.Equal(t, workspaceOpenAICompatibleTextExecutorErrorProvider, result.ErrorCode)
	require.Equal(t, true, result.Metadata["provider_called"])
	require.Equal(t, false, result.Metadata["billing_touched"])
	require.Equal(t, false, result.Metadata["asset_touched"])
	require.Equal(t, "failed", result.Metadata["status"])
	require.Equal(t, "workspace openai-compatible provider failed safely", err.Error())

	assertWorkspaceOpenAICompatiblePayloadSafe(t, result)
	assertWorkspaceOpenAICompatiblePayloadSafe(t, err.Error())
}

func TestWorkspaceOpenAICompatibleTextExecutorRejectsUnsafePlanBeforeUpstream(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*WorkspaceTextProviderExecutionInput)
	}{
		{
			name: "contract denied",
			mutate: func(input *WorkspaceTextProviderExecutionInput) {
				input.ExecutionContract = WorkspaceProviderExecutionContract{
					Decision:        WorkspaceProviderExecutionDecisionDeny,
					CanCallProvider: false,
				}
			},
		},
		{
			name: "missing billing policy",
			mutate: func(input *WorkspaceTextProviderExecutionInput) {
				input.ExecutionContract.BillingPolicy = ""
			},
		},
		{
			name: "missing usage policy",
			mutate: func(input *WorkspaceTextProviderExecutionInput) {
				input.ExecutionContract.UsagePolicy = ""
			},
		},
		{
			name: "invalid model",
			mutate: func(input *WorkspaceTextProviderExecutionInput) {
				input.Model = "not-allowed"
			},
		},
		{
			name: "invalid intent",
			mutate: func(input *WorkspaceTextProviderExecutionInput) {
				input.Intent = "image_generation"
			},
		},
		{
			name: "unsafe inline payload",
			mutate: func(input *WorkspaceTextProviderExecutionInput) {
				input.Content = "data:image/png;base64,AAAA"
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			upstream := &fakeWorkspaceOpenAICompatibleTextUpstream{}
			executor := workspaceOpenAICompatibleExecutorForTest(upstream)
			input := validWorkspaceOpenAICompatibleExecutionInput()
			tc.mutate(&input)

			result, err := executor.ExecuteWorkspaceTextProvider(context.Background(), input)
			require.Error(t, err)
			require.ErrorIs(t, err, errWorkspaceOpenAICompatibleExecutionBlocked)
			require.Zero(t, upstream.calls)
			require.Equal(t, workspaceOpenAICompatibleTextExecutorErrorBlocked, result.ErrorCode)
			require.Equal(t, false, result.Metadata["provider_called"])
			require.Equal(t, false, result.Metadata["billing_touched"])
			assertWorkspaceOpenAICompatiblePayloadSafe(t, result)
		})
	}
}

func TestWorkspaceOpenAICompatibleTextExecutorNilUpstreamFailsClosed(t *testing.T) {
	executor := workspaceOpenAICompatibleExecutorForTest(nil)

	result, err := executor.ExecuteWorkspaceTextProvider(context.Background(), validWorkspaceOpenAICompatibleExecutionInput())
	require.Error(t, err)
	require.ErrorIs(t, err, errWorkspaceOpenAICompatibleUnavailable)
	require.Equal(t, workspaceOpenAICompatibleTextExecutorErrorUnavailable, result.ErrorCode)
	require.Equal(t, false, result.Metadata["provider_called"])
	require.Equal(t, false, result.Metadata["billing_touched"])
	assertWorkspaceOpenAICompatiblePayloadSafe(t, result)
}

func TestWorkspaceOpenAICompatibleTextExecutorEmptyResponseFailsSafely(t *testing.T) {
	upstream := &fakeWorkspaceOpenAICompatibleTextUpstream{
		response: WorkspaceOpenAICompatibleExecutionResponse{
			Content:      "   ",
			ProviderName: "deepseek-staging",
		},
	}
	executor := workspaceOpenAICompatibleExecutorForTest(upstream)

	result, err := executor.ExecuteWorkspaceTextProvider(context.Background(), validWorkspaceOpenAICompatibleExecutionInput())
	require.Error(t, err)
	require.ErrorIs(t, err, errWorkspaceOpenAICompatibleEmptyResponse)
	require.Equal(t, 1, upstream.calls)
	require.Equal(t, workspaceOpenAICompatibleTextExecutorErrorEmpty, result.ErrorCode)
	require.Equal(t, true, result.Metadata["provider_called"])
	require.Equal(t, false, result.Metadata["billing_touched"])
	assertWorkspaceOpenAICompatiblePayloadSafe(t, result)
}

type fakeWorkspaceOpenAICompatibleTextUpstream struct {
	calls       int
	lastRequest WorkspaceOpenAICompatibleExecutionRequest
	response    WorkspaceOpenAICompatibleExecutionResponse
	err         error
}

func (u *fakeWorkspaceOpenAICompatibleTextUpstream) ExecuteWorkspaceOpenAICompatibleText(_ context.Context, req WorkspaceOpenAICompatibleExecutionRequest) (WorkspaceOpenAICompatibleExecutionResponse, error) {
	u.calls++
	u.lastRequest = req
	if u.err != nil {
		return WorkspaceOpenAICompatibleExecutionResponse{}, u.err
	}
	return u.response, nil
}

func workspaceOpenAICompatibleExecutorForTest(upstream WorkspaceOpenAICompatibleTextUpstream) WorkspaceOpenAICompatibleTextExecutor {
	return WorkspaceOpenAICompatibleTextExecutor{
		Upstream:      upstream,
		ProviderLabel: "deepseek-staging",
		Platform:      "openai_compatible",
		EndpointLabel: "deepseek-staging",
		ServiceTier:   "staging",
		Now: func() time.Time {
			return time.Date(2026, 6, 12, 10, 0, 0, 0, time.UTC)
		},
	}
}

func validWorkspaceOpenAICompatibleExecutionInput() WorkspaceTextProviderExecutionInput {
	contract := ValidateWorkspaceProviderExecutionPlan(WorkspaceProviderExecutionRequest{
		RequestID:               "req_openai_compatible_boundary_test",
		FeatureGateEnabled:      true,
		UserID:                  1001,
		ConversationID:          2002,
		UserMessageID:           3003,
		Content:                 "please answer with staging ok for the executor boundary",
		Model:                   "deepseek-v4-flash",
		Intent:                  WorkspaceIntentChat,
		Capability:              WorkspaceProviderCapabilityText,
		ProviderAvailable:       true,
		BillingEligibilityKnown: true,
		BillingEligible:         true,
		BillingPolicy:           WorkspaceProviderBillingPolicyRecordUsageOnProviderUsage,
		UsagePolicy:             WorkspaceProviderUsagePolicyRecordProviderReported,
		FailurePolicy:           WorkspaceProviderFailurePolicyNoChargeOnFailure,
		Diagnostics: WorkspaceProviderDiagnostics{
			RequestedModel:        "deepseek-v4-flash",
			MappedModel:           "deepseek-v4-flash",
			UpstreamModel:         "deepseek-v4-flash",
			ProviderName:          "deepseek-staging",
			ServiceTier:           "staging",
			SupportedCapabilities: []WorkspaceProviderCapability{WorkspaceProviderCapabilityText},
		},
		EndpointLabel: "deepseek-staging",
		CreatedAt:     time.Date(2026, 6, 12, 10, 0, 0, 0, time.UTC),
	})
	return WorkspaceTextProviderExecutionInput{
		RequestID:         "req_openai_compatible_boundary_test",
		UserID:            1001,
		ConversationID:    2002,
		UserMessageID:     3003,
		Content:           "please answer with staging ok for the executor boundary",
		Model:             "deepseek-v4-flash",
		Intent:            WorkspaceIntentChat,
		MappedModel:       "deepseek-v4-flash",
		UpstreamModel:     "deepseek-v4-flash",
		ProviderName:      "deepseek-staging",
		EndpointLabel:     "deepseek-staging",
		ServiceTier:       "staging",
		ExecutionContract: contract,
	}
}

func assertWorkspaceOpenAICompatiblePayloadSafe(t *testing.T, value any) {
	t.Helper()
	payload, err := json.Marshal(value)
	require.NoError(t, err)
	body := strings.ToLower(string(payload))
	for _, forbidden := range []string{
		"sk-secret",
		"authorization: bearer",
		"access_token",
		"refresh_token",
		"cookie=session",
		"internal.example",
		"sql stack",
		"please answer with staging ok for the executor boundary",
	} {
		require.NotContains(t, body, forbidden)
	}
}
