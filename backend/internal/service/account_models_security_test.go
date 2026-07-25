//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMergeCredentialUpdatesPreservingSecretsRetainsOmittedCredentials(t *testing.T) {
	existing := map[string]any{
		"api_key":       "sk-existing",
		"base_url":      "https://upstream.example.com",
		"model_mapping": map[string]any{"gpt-old": "gpt-old"},
	}
	incoming := map[string]any{
		"model_mapping": map[string]any{"gpt-new": "gpt-new"},
	}

	got := MergeCredentialUpdatesPreservingSecrets(existing, incoming)

	require.Equal(t, "sk-existing", got["api_key"])
	require.Equal(t, "https://upstream.example.com", got["base_url"])
	require.Equal(t, incoming["model_mapping"], got["model_mapping"])
}
