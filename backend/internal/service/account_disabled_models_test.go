package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExplicitModelQuarantineCatalog(t *testing.T) {
	account := &Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
		Extra: map[string]any{"openai_passthrough": true, openAIDisabledModelsKey: []any{"gpt-5.5"}}}
	for _, codex := range []bool{false, true} {
		body := []byte(`{"object":"list","data":[{"id":"gpt-5.5"},{"id":"gpt-5.6-sol","custom":true}]}`)
		if codex {
			body = []byte(`{"models":[{"slug":"gpt-5.5"},{"slug":"gpt-5.6-sol","custom":true}]}`)
		}
		result, err := projectAccountModelsBody(body, account, &Group{}, codex)
		require.NoError(t, err)
		require.NotContains(t, string(result), `gpt-5.5`)
		require.Contains(t, string(result), `gpt-5.6-sol`)
		require.Contains(t, string(result), `"custom":true`)
		require.Contains(t, string(body), `gpt-5.5`, "shared upstream cache is immutable")
	}
}

func TestExplicitModelQuarantineLegacyCompact(t *testing.T) {
	account := &Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true,
		Extra:       map[string]any{"openai_passthrough": true, openAIDisabledModelsKey: []any{"gpt-5.5"}},
		Credentials: map[string]any{"compact_model_mapping": map[string]any{"gpt-6-astra": "gpt-5.5"}}}
	require.Equal(t, "model_not_supported", openAICompatibleAccountEligibilityFailureReasonBeforeProfit(context.Background(), account, PlatformOpenAI, "gpt-5.5", false, OpenAIEndpointCapabilityResponses))
	require.Equal(t, "model_not_supported", openAICompatibleAccountEligibilityFailureReasonBeforeProfit(context.Background(), account, PlatformOpenAI, "gpt-6-astra", true, OpenAIEndpointCapabilityResponses))
}

func TestExplicitModelQuarantineOverridesPassthrough(t *testing.T) {
	account := &Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
		Extra:       map[string]any{"openai_passthrough": true, openAIDisabledModelsKey: []any{"gpt-5.5"}},
		Credentials: map[string]any{"model_mapping": map[string]any{"gpt-5.5": "gpt-5.5"}}}
	require.False(t, account.IsModelSupported("gpt-5.5"))
	require.False(t, (&GatewayService{}).isModelSupportedByAccount(account, "gpt-5.5"))
	for _, model := range []string{"gpt-5.6-sol", "gpt-5.6-terra", "gpt-6-astra", "gpt-5.50"} {
		require.True(t, account.IsModelSupported(model), model)
	}
	// Persistence roundtrip must retain the quarantine independently of TTLs.
	data, err := json.Marshal(account.Extra)
	require.NoError(t, err)
	account.Extra = nil
	require.NoError(t, json.Unmarshal(data, &account.Extra))
	require.False(t, account.IsModelSupported("gpt-5.5"))
	delete(account.Extra, openAIDisabledModelsKey)
	require.True(t, account.IsModelSupported("gpt-5.5"))
}

func TestExplicitModelQuarantineMappedTarget(t *testing.T) {
	account := &Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
		Extra:       map[string]any{openAIDisabledModelsKey: []string{"gpt-5.5"}},
		Credentials: map[string]any{"model_mapping": map[string]any{"friendly": "gpt-5.5", "working": "gpt-5.6-sol"}}}
	require.False(t, account.IsModelSupported("friendly"))
	require.True(t, account.IsModelSupported("working"))
	account.Extra["openai_passthrough"] = true
	// Passthrough does NOT forward to the stale mapping target.
	require.True(t, account.IsModelSupported("friendly"))
}

func TestValidateExplicitModelQuarantine(t *testing.T) {
	for _, raw := range []any{[]string{"gpt-5.5"}, []any{"gpt-5.5"}, []any{}} {
		require.NoError(t, ValidateOpenAIDisabledModelsExtra(map[string]any{openAIDisabledModelsKey: raw}))
	}
	for _, raw := range []any{"gpt-5.5", nil, []any{1}, []string{"gpt-*"}, []string{""}, []string{" gpt-5.5"}} {
		require.Error(t, ValidateOpenAIDisabledModelsExtra(map[string]any{openAIDisabledModelsKey: raw}))
	}
}
