package handler

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestAPIKeyModelAllowlistRejectsNonExactMatches(t *testing.T) {
	key := &service.APIKey{AllowedModels: []string{"gpt-5.4", "gemini-2.5-pro", "gpt-5.6-sol"}}
	require.True(t, apiKeyAllowsRequestedModel(key, "gpt-5.4"))
	require.False(t, apiKeyAllowsRequestedModel(key, "gpt-5.4-latest"))
	require.True(t, apiKeyAllowsRequestedModel(key, "models/gemini-2.5-pro"))
	require.True(t, apiKeyAllowsRequestedModel(key, "gpt-5.6"))
	require.True(t, apiKeyAllowsRequestedModel(&service.APIKey{}, "unrestricted"))
}
