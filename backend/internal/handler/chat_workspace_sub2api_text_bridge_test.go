package handler

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseWorkspaceSub2APITextBridgeResultMarksUsageWhenPayloadPresent(t *testing.T) {
	body := []byte(`{
		"id": "chatcmpl-deepseek",
		"object": "chat.completion",
		"model": "deepseek-v4-flash",
		"choices": [{"message": {"role": "assistant", "content": "STAGING_TEXT_OK"}}],
		"usage": {
			"prompt_tokens": 7,
			"completion_tokens": 3,
			"total_tokens": 10
		}
	}`)

	result, err := parseWorkspaceSub2APITextBridgeResult(body, "deepseek-v4-flash")

	require.NoError(t, err)
	require.Equal(t, "STAGING_TEXT_OK", result.Content)
	require.Equal(t, "deepseek-v4-flash", result.Model)
	require.True(t, result.ProviderCalled)
	require.True(t, result.UsageRecorded)
	require.True(t, result.BillingManaged)
	require.Equal(t, "usage_payload_present", result.AdditionalFields["usage_status"])
	require.Equal(t, "sub2api_gateway_usage_path", result.AdditionalFields["billing_status"])
}

func TestParseWorkspaceSub2APITextBridgeResultDoesNotClaimBillingWhenUsageMissing(t *testing.T) {
	body := []byte(`{
		"id": "chatcmpl-no-usage",
		"object": "chat.completion",
		"model": "deepseek-v4-flash",
		"choices": [{"message": {"role": "assistant", "content": "STAGING_TEXT_OK"}}]
	}`)

	result, err := parseWorkspaceSub2APITextBridgeResult(body, "deepseek-v4-flash")

	require.NoError(t, err)
	require.Equal(t, "STAGING_TEXT_OK", result.Content)
	require.True(t, result.ProviderCalled)
	require.False(t, result.UsageRecorded)
	require.False(t, result.BillingManaged)
	require.Equal(t, "usage_missing", result.AdditionalFields["usage_status"])
	require.Equal(t, "billing_not_recorded", result.AdditionalFields["billing_status"])
}

func TestParseWorkspaceSub2APITextBridgeResultDoesNotClaimBillingWhenUsageIsZero(t *testing.T) {
	body := []byte(`{
		"id": "chatcmpl-zero-usage",
		"object": "chat.completion",
		"model": "deepseek-v4-flash",
		"choices": [{"message": {"role": "assistant", "content": "STAGING_TEXT_OK"}}],
		"usage": {
			"prompt_tokens": 0,
			"completion_tokens": 0,
			"total_tokens": 0
		}
	}`)

	result, err := parseWorkspaceSub2APITextBridgeResult(body, "deepseek-v4-flash")

	require.NoError(t, err)
	require.True(t, result.ProviderCalled)
	require.False(t, result.UsageRecorded)
	require.False(t, result.BillingManaged)
	require.Equal(t, "usage_missing", result.AdditionalFields["usage_status"])
	require.Equal(t, "billing_not_recorded", result.AdditionalFields["billing_status"])
}
