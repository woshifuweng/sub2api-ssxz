//go:build unit

package service

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeProviderBaseURLRejectsUnsafeValues(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{name: "http", raw: "http://api.example.com"},
		{name: "file scheme", raw: "file:///etc/passwd"},
		{name: "localhost", raw: "https://localhost"},
		{name: "localhost subdomain", raw: "https://api.localhost"},
		{name: "loopback ip", raw: "https://127.0.0.1"},
		{name: "private ip", raw: "https://10.0.0.1"},
		{name: "metadata ip", raw: "https://169.254.169.254"},
		{name: "metadata host", raw: "https://metadata.google.internal"},
		{name: "userinfo", raw: "https://user:pass@example.com"},
		{name: "query", raw: "https://api.example.com/v1?token=secret"},
		{name: "fragment", raw: "https://api.example.com/v1#frag"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NormalizeProviderBaseURL(tt.raw)
			require.Error(t, err)
		})
	}
}

func TestNormalizeProviderBaseURLAcceptsPublicHTTPS(t *testing.T) {
	normalized, err := NormalizeProviderBaseURL(" https://api.example.com/v1/ ")
	require.NoError(t, err)
	require.Equal(t, "https://api.example.com/v1", normalized)
}

func TestAdminServiceCreateAccountRejectsUnsafeProviderBaseURL(t *testing.T) {
	repo := &adminSoraAccountRepoStub{}
	svc := &adminServiceImpl{
		accountRepo: repo,
		groupRepo:   &adminSoraGroupRepoStub{},
	}

	_, err := svc.CreateAccount(context.Background(), &CreateAccountInput{
		Name:        "openai",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Credentials: map[string]any{"api_key": "sk-test", "base_url": "http://127.0.0.1"},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "base_url")
	require.Nil(t, repo.created)
}

func TestAdminServiceUpdateAccountRejectsUnsafeProviderBaseURL(t *testing.T) {
	repo := &adminSoraAccountRepoStub{
		accountToReturn: &Account{
			ID:          1,
			Name:        "openai",
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Credentials: map[string]any{"api_key": "sk-test", "base_url": "https://api.openai.com"},
		},
	}
	svc := &adminServiceImpl{
		accountRepo: repo,
		groupRepo:   &adminSoraGroupRepoStub{},
	}

	_, err := svc.UpdateAccount(context.Background(), 1, &UpdateAccountInput{
		Credentials: map[string]any{"base_url": "https://metadata.google.internal"},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "base_url")
	require.Nil(t, repo.updated)
}

func TestNormalizeAccountCredentialsBaseURLNormalizesStoredValue(t *testing.T) {
	credentials := map[string]any{"base_url": " https://api.example.com/v1/ "}
	err := NormalizeAccountCredentialsBaseURL(PlatformOpenAI, AccountTypeAPIKey, credentials)
	require.NoError(t, err)
	require.Equal(t, "https://api.example.com/v1", credentials["base_url"])
}

func TestNormalizeAccountCredentialsBaseURLRejectsNonStringBaseURL(t *testing.T) {
	credentials := map[string]any{"base_url": 123}
	err := NormalizeAccountCredentialsBaseURL(PlatformOpenAI, AccountTypeAPIKey, credentials)
	require.Error(t, err)
	require.Contains(t, err.Error(), "base_url")
}

func TestNormalizeSoraAPIKeyBaseURLRejectsHTTP(t *testing.T) {
	_, err := NormalizeSoraAPIKeyBaseURL("http://sora.example.com")
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "scheme") || strings.Contains(err.Error(), "https"))
}
