//go:build unit

package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type modelsRefreshAccountRepoStub struct {
	accountRepoStub
	account       *Account
	updateExtra   map[string]any
	updateExtraID int64
}

func (s *modelsRefreshAccountRepoStub) GetByID(_ context.Context, id int64) (*Account, error) {
	if s.account == nil || s.account.ID != id {
		return nil, fmt.Errorf("account %d not found", id)
	}
	cloned := *s.account
	return &cloned, nil
}

func (s *modelsRefreshAccountRepoStub) UpdateExtra(_ context.Context, id int64, updates map[string]any) error {
	s.updateExtraID = id
	s.updateExtra = updates
	if s.account != nil && s.account.ID == id {
		if s.account.Extra == nil {
			s.account.Extra = make(map[string]any, len(updates))
		}
		for key, value := range updates {
			s.account.Extra[key] = value
		}
	}
	return nil
}

type modelFetchHTTPUpstreamStub struct {
	resp *http.Response
	err  error
	req  *http.Request
}

func (s *modelFetchHTTPUpstreamStub) Do(_ *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	return nil, fmt.Errorf("unexpected Do call")
}

func (s *modelFetchHTTPUpstreamStub) DoWithTLS(req *http.Request, _ string, _ int64, _ int, _ bool) (*http.Response, error) {
	s.req = req
	if s.err != nil {
		return nil, s.err
	}
	return s.resp, nil
}

func TestAccountTestService_FetchAndCacheAvailableModels_OpenAISuccess(t *testing.T) {
	repo := &modelsRefreshAccountRepoStub{
		account: &Account{
			ID:       7,
			Platform: PlatformOpenAI,
			Type:     AccountTypeAPIKey,
			Credentials: map[string]any{
				"api_key": "sk-test",
			},
			Extra: map[string]any{
				"note": "kept",
			},
		},
	}
	upstream := &modelFetchHTTPUpstreamStub{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"data":[{"id":"gpt-5"},{"id":"gpt-5-mini"}]}`)),
		},
	}
	svc := &AccountTestService{
		accountRepo:  repo,
		httpUpstream: upstream,
	}

	result, err := svc.FetchAndCacheAvailableModels(context.Background(), 7)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, []string{"gpt-5", "gpt-5-mini"}, result.Models)
	require.Equal(t, int64(7), repo.updateExtraID)
	require.Equal(t, []string{"gpt-5", "gpt-5-mini"}, repo.updateExtra[AccountExtraFetchedModelsKey])
	require.NotEmpty(t, repo.updateExtra[AccountExtraModelsFetchedAtKey])
	require.Equal(t, "openai_v1_models", repo.updateExtra[AccountExtraModelsSourceKey])
	require.Equal(t, "", repo.updateExtra[AccountExtraModelsRefreshErrorKey])
	require.Equal(t, "openai", repo.updateExtra[AccountExtraModelsDiscoveryProviderTypeKey])
	require.Equal(t, "openai_v1_models", repo.updateExtra[AccountExtraModelsDiscoveryProtocolKey])
	require.Equal(t, "api.openai.com", repo.updateExtra[AccountExtraModelsDiscoveryBaseURLHostKey])
	require.Equal(t, 2, repo.updateExtra[AccountExtraModelsDiscoveryModelCountKey])
	require.NotEmpty(t, repo.updateExtra[AccountExtraModelsDiscoveryAuditedAtKey])
	require.Equal(t, "openai", result.Audit.ProviderType)
	require.Equal(t, "openai_v1_models", result.Audit.Protocol)
	require.Equal(t, "api.openai.com", result.Audit.BaseURLHost)
	require.Equal(t, 2, result.Audit.ModelsReturnedCount)
	require.True(t, result.Audit.ServerSideKeyPresent)
	require.NotContains(t, fmt.Sprintf("%+v", result.Audit), "sk-test")
	require.NotContains(t, fmt.Sprintf("%+v", result.Audit), "Authorization")
	require.NotNil(t, upstream.req)
	require.Equal(t, "https://api.openai.com/v1/models", upstream.req.URL.String())
	require.Equal(t, "Bearer sk-test", upstream.req.Header.Get("Authorization"))
}

func TestAccountTestService_FetchAndCacheAvailableModels_PreservesOldModelsOnFailure(t *testing.T) {
	repo := &modelsRefreshAccountRepoStub{
		account: &Account{
			ID:       8,
			Platform: PlatformOpenAI,
			Type:     AccountTypeAPIKey,
			Credentials: map[string]any{
				"api_key": "sk-test",
			},
			Extra: map[string]any{
				AccountExtraFetchedModelsKey: []any{"gpt-5"},
			},
		},
	}
	upstream := &modelFetchHTTPUpstreamStub{
		err: fmt.Errorf("dial tcp 127.0.0.1:8080: connectex: connection refused"),
	}
	svc := &AccountTestService{
		accountRepo:  repo,
		httpUpstream: upstream,
	}

	result, err := svc.FetchAndCacheAvailableModels(context.Background(), 8)
	require.Error(t, err)
	require.NotNil(t, result)
	require.Equal(t, []string{"gpt-5"}, result.Models)
	require.Equal(t, int64(8), repo.updateExtraID)
	require.Equal(t, truncateModelsRefreshError(err), repo.updateExtra[AccountExtraModelsRefreshErrorKey])
	require.Equal(t, "openai", repo.updateExtra[AccountExtraModelsDiscoveryProviderTypeKey])
	require.Equal(t, "openai_v1_models", repo.updateExtra[AccountExtraModelsDiscoveryProtocolKey])
	require.Equal(t, "api.openai.com", repo.updateExtra[AccountExtraModelsDiscoveryBaseURLHostKey])
	require.Equal(t, 1, repo.updateExtra[AccountExtraModelsDiscoveryModelCountKey])
	require.Equal(t, "openai", result.Audit.ProviderType)
	require.Equal(t, 1, result.Audit.ModelsReturnedCount)
	require.NotEmpty(t, result.Audit.RefreshError)
	require.Equal(t, []any{"gpt-5"}, repo.account.Extra[AccountExtraFetchedModelsKey])
}

func TestAccountTestService_FetchAndCacheAvailableModels_OpenAIChatWebUsesTokenProvider(t *testing.T) {
	repo := &modelsRefreshAccountRepoStub{
		account: &Account{
			ID:       9,
			Platform: PlatformOpenAI,
			Type:     AccountTypeOAuth,
			Credentials: map[string]any{
				"session_token":      "st-live",
				"chatgpt_account_id": "acc-live",
			},
			Extra: map[string]any{
				"openai_auth_mode": OpenAIAuthModeChatWeb,
			},
		},
	}
	upstream := &modelFetchHTTPUpstreamStub{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"data":[{"id":"gpt-5"},{"id":"gpt-5-mini"}]}`)),
		},
	}
	tokenProvider := &openAIAccountTokenProviderStub{token: "fresh-chatweb-token"}
	svc := &AccountTestService{
		accountRepo:         repo,
		httpUpstream:        upstream,
		openAITokenProvider: tokenProvider,
	}

	result, err := svc.FetchAndCacheAvailableModels(context.Background(), 9)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, []string{"gpt-5", "gpt-5-mini"}, result.Models)
	require.Equal(t, 1, tokenProvider.calls)
	require.NotNil(t, upstream.req)
	require.Equal(t, "Bearer fresh-chatweb-token", upstream.req.Header.Get("Authorization"))
	require.Equal(t, "acc-live", upstream.req.Header.Get("chatgpt-account-id"))
	require.Equal(t, "fresh-chatweb-token", repo.account.GetOpenAIAccessToken())
}

func TestAccountTestService_FetchAndCacheAvailableModels_DeepSeekCompatibleAudit(t *testing.T) {
	repo := &modelsRefreshAccountRepoStub{
		account: &Account{
			ID:       10,
			Platform: PlatformOpenAI,
			Type:     AccountTypeAPIKey,
			Credentials: map[string]any{
				"api_key":  "sk-deepseek",
				"base_url": "https://api.deepseek.com",
			},
		},
	}
	upstream := &modelFetchHTTPUpstreamStub{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"data":[{"id":"deepseek-v4-pro"},{"id":"deepseek-v4-flash"},{"id":"deepseek-v4-pro"}]}`)),
		},
	}
	svc := &AccountTestService{
		accountRepo:  repo,
		httpUpstream: upstream,
	}

	result, err := svc.FetchAndCacheAvailableModels(context.Background(), 10)
	require.NoError(t, err)
	require.Equal(t, []string{"deepseek-v4-flash", "deepseek-v4-pro"}, result.Models)
	require.Equal(t, "deepseek", result.Audit.ProviderType)
	require.Equal(t, "openai_v1_models", result.Audit.Protocol)
	require.Equal(t, "api.deepseek.com", result.Audit.BaseURLHost)
	require.Equal(t, 2, result.Audit.ModelsReturnedCount)
	require.True(t, result.Audit.ServerSideKeyPresent)
	require.Equal(t, "deepseek", repo.updateExtra[AccountExtraModelsDiscoveryProviderTypeKey])
	require.Equal(t, "api.deepseek.com", repo.updateExtra[AccountExtraModelsDiscoveryBaseURLHostKey])
	require.NotContains(t, fmt.Sprintf("%+v", result.Audit), "sk-deepseek")
}

func TestBuildAccountModelDiscoveryAudit_SanitizesSensitiveError(t *testing.T) {
	audit := BuildAccountModelDiscoveryAudit(&Account{
		ID:       11,
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key":  "sk-sensitive",
			"base_url": "https://api.openai.com",
		},
	}, []string{"gpt-5"}, "openai_v1_models", time.Now(), "Authorization: Bearer sk-sensitive api_key=abc Cookie=x")

	rendered := fmt.Sprintf("%+v", audit)
	require.NotContains(t, rendered, "Authorization")
	require.NotContains(t, rendered, "Bearer")
	require.NotContains(t, rendered, "api_key")
	require.NotContains(t, rendered, "Cookie")
	require.NotContains(t, rendered, "sk-sensitive")
	require.Contains(t, audit.RefreshError, "[redacted-header]")
	require.Contains(t, audit.RefreshError, "[redacted-bearer]")
}
