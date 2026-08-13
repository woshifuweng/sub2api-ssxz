package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	"github.com/Wei-Shaw/sub2api/internal/pkg/geminicli"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
)

// AccountModelsRefreshResult is returned after refreshing an account's fetched model cache.
type AccountModelsRefreshResult struct {
	AccountID    int64                      `json:"account_id"`
	Models       []string                   `json:"models"`
	FetchedAt    *time.Time                 `json:"fetched_at,omitempty"`
	Source       string                     `json:"source,omitempty"`
	RefreshError string                     `json:"refresh_error,omitempty"`
	Audit        AccountModelDiscoveryAudit `json:"audit"`
}

const accountModelsRefreshTimeout = 20 * time.Second

func buildVersionedModelsURL(baseURL, version string) string {
	normalized := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if normalized == "" {
		return ""
	}
	if strings.HasSuffix(normalized, "/models") {
		return normalized
	}
	if strings.HasSuffix(normalized, version) {
		return normalized + "/models"
	}
	return normalized + version + "/models"
}

func (s *AccountTestService) executeAccountRequestForModels(req *http.Request, account *Account) (*http.Response, error) {
	if s.httpUpstream == nil {
		return nil, errors.New("http upstream is not configured")
	}
	proxyURL := ""
	if account != nil && account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	var profile *tlsfingerprint.Profile
	if s.tlsFPProfileService != nil && account != nil {
		profile = s.tlsFPProfileService.ResolveTLSProfile(account)
	}
	var accountID int64
	var concurrency int
	if account != nil {
		accountID = account.ID
		concurrency = account.Concurrency
	}
	return s.httpUpstream.DoWithTLS(req, proxyURL, accountID, concurrency, profile)
}

func (s *AccountTestService) fetchOpenAIAvailableModels(ctx context.Context, account *Account) ([]string, string, error) {
	baseURL := account.GetOpenAIBaseURL()
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://api.openai.com"
	}
	normalizedBaseURL, err := s.validateUpstreamBaseURL(baseURL)
	if err != nil {
		return nil, "", fmt.Errorf("invalid openai base_url: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, buildVersionedModelsURL(normalizedBaseURL, "/v1"), nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Accept", "application/json")
	switch {
	case account.IsOpenAIOAuth():
		token := strings.TrimSpace(account.GetOpenAIAccessToken())
		if token == "" {
			return nil, "", errors.New("openai access token is not configured")
		}
		req.Header.Set("Authorization", "Bearer "+token)
		if id := strings.TrimSpace(account.GetChatGPTAccountID()); id != "" {
			req.Header.Set("chatgpt-account-id", id)
		}
		if id := strings.TrimSpace(account.GetOpenAIOrganizationID()); id != "" {
			req.Header.Set("OpenAI-Organization", id)
		}
	case account.IsOpenAIApiKey():
		key := strings.TrimSpace(account.GetOpenAIApiKey())
		if key == "" {
			return nil, "", errors.New("openai api_key is not configured")
		}
		req.Header.Set("Authorization", "Bearer "+key)
	default:
		return nil, "", fmt.Errorf("openai account type %q does not support model refresh", account.Type)
	}
	resp, err := s.executeAccountRequestForModels(req, account)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("openai models request returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var payload struct {
		Data []struct{ ID string `json:"id"` } `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, "", fmt.Errorf("parse openai models response: %w", err)
	}
	models := make([]string, 0, len(payload.Data))
	for _, item := range payload.Data {
		models = append(models, item.ID)
	}
	return NormalizeFetchedModelIDs(models), "openai_v1_models", nil
}

func (s *AccountTestService) fetchAnthropicAvailableModels(ctx context.Context, account *Account) ([]string, string, error) {
	if account == nil {
		return nil, "", errors.New("account is nil")
	}
	if account.IsBedrock() {
		return nil, "", errors.New("bedrock accounts do not support model refresh")
	}
	if account.Type != AccountTypeAPIKey && !account.IsOAuth() {
		return nil, "", fmt.Errorf("anthropic account type %q does not support model refresh", account.Type)
	}
	baseURL := "https://api.anthropic.com"
	if account.Type == AccountTypeAPIKey && strings.TrimSpace(account.GetBaseURL()) != "" {
		baseURL = account.GetBaseURL()
	}
	normalizedBaseURL, err := s.validateUpstreamBaseURL(baseURL)
	if err != nil {
		return nil, "", fmt.Errorf("invalid anthropic base_url: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, buildVersionedModelsURL(normalizedBaseURL, "/v1"), nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("anthropic-version", "2023-06-01")
	if account.IsOAuth() {
		token := strings.TrimSpace(account.GetCredential("access_token"))
		if token == "" {
			return nil, "", errors.New("anthropic access_token is not configured")
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("anthropic-beta", claude.DefaultBetaHeader)
	} else {
		key := strings.TrimSpace(account.GetCredential("api_key"))
		if key == "" {
			return nil, "", errors.New("anthropic api_key is not configured")
		}
		req.Header.Set("x-api-key", key)
		req.Header.Set("anthropic-beta", claude.APIKeyBetaHeader)
	}
	resp, err := s.executeAccountRequestForModels(req, account)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("anthropic models request returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var payload struct {
		Data []struct{ ID string `json:"id"` } `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, "", fmt.Errorf("parse anthropic models response: %w", err)
	}
	models := make([]string, 0, len(payload.Data))
	for _, item := range payload.Data {
		models = append(models, item.ID)
	}
	return NormalizeFetchedModelIDs(models), "anthropic_v1_models", nil
}

func (s *AccountTestService) fetchGeminiAvailableModels(ctx context.Context, account *Account) ([]string, string, error) {
	if account == nil {
		return nil, "", errors.New("account is nil")
	}
	normalizedBaseURL, err := s.validateUpstreamBaseURL(account.GetGeminiBaseURL(geminicli.AIStudioBaseURL))
	if err != nil {
		return nil, "", fmt.Errorf("invalid gemini base_url: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, buildVersionedModelsURL(normalizedBaseURL, "/v1beta"), nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Accept", "application/json")
	switch account.Type {
	case AccountTypeAPIKey:
		key := strings.TrimSpace(account.GetCredential("api_key"))
		if key == "" {
			return nil, "", errors.New("gemini api_key is not configured")
		}
		req.Header.Set("x-goog-api-key", key)
	case AccountTypeOAuth:
		if s.geminiTokenProvider == nil {
			return nil, "", errors.New("gemini token provider is not configured")
		}
		token, err := s.geminiTokenProvider.GetAccessToken(ctx, account)
		if err != nil {
			return nil, "", fmt.Errorf("get gemini access token: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
	default:
		return nil, "", fmt.Errorf("gemini account type %q does not support model refresh", account.Type)
	}
	resp, err := s.executeAccountRequestForModels(req, account)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("gemini models request returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var payload struct {
		Models []struct{ Name string `json:"name"` } `json:"models"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, "", fmt.Errorf("parse gemini models response: %w", err)
	}
	models := make([]string, 0, len(payload.Models))
	for _, item := range payload.Models {
		models = append(models, item.Name)
	}
	return NormalizeFetchedModelIDs(models), "gemini_v1beta_models", nil
}

func (s *AccountTestService) fetchAvailableModelsForAccount(ctx context.Context, account *Account) ([]string, string, error) {
	switch {
	case account == nil:
		return nil, "", errors.New("account is nil")
	case account.IsOpenAI():
		return s.fetchOpenAIAvailableModels(ctx, account)
	case account.IsGemini():
		return s.fetchGeminiAvailableModels(ctx, account)
	case account.IsAnthropic() && !account.IsBedrock():
		return s.fetchAnthropicAvailableModels(ctx, account)
	default:
		return nil, "", fmt.Errorf("platform %q does not support model refresh", account.Platform)
	}
}

func (s *AccountTestService) FetchAndCacheAvailableModels(ctx context.Context, accountID int64) (*AccountModelsRefreshResult, error) {
	if s.accountRepo == nil {
		return nil, errors.New("account repository is not configured")
	}
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	refreshCtx, cancel := context.WithTimeout(ctx, accountModelsRefreshTimeout)
	defer cancel()
	modelIDs, source, err := s.fetchAvailableModelsForAccount(refreshCtx, account)
	if err != nil {
		refreshErr := truncateModelsRefreshError(err)
		updates := map[string]any{AccountExtraModelsRefreshErrorKey: refreshErr}
		for key, value := range BuildAccountModelDiscoveryExtraUpdates(account, account.GetFetchedModelIDs(), time.Now().UTC(), account.GetExtraString(AccountExtraModelsSourceKey), refreshErr) {
			updates[key] = value
		}
		_ = s.accountRepo.UpdateExtra(refreshCtx, account.ID, updates)
		mergeAccountExtra(account, updates)
		return &AccountModelsRefreshResult{AccountID: account.ID, Models: account.GetFetchedModelIDs(), FetchedAt: account.GetModelsFetchedAt(), Source: strings.TrimSpace(account.GetExtraString(AccountExtraModelsSourceKey)), RefreshError: refreshErr, Audit: BuildAccountModelDiscoveryAudit(account, account.GetFetchedModelIDs(), account.GetExtraString(AccountExtraModelsSourceKey), time.Now().UTC(), refreshErr)}, err
	}
	fetchedAt := time.Now().UTC()
	updates := BuildFetchedModelsExtraUpdates(modelIDs, fetchedAt, source)
	for key, value := range BuildAccountModelDiscoveryExtraUpdates(account, modelIDs, fetchedAt, source, "") {
		updates[key] = value
	}
	if err := s.accountRepo.UpdateExtra(refreshCtx, account.ID, updates); err != nil {
		return nil, err
	}
	mergeAccountExtra(account, updates)
	return &AccountModelsRefreshResult{AccountID: account.ID, Models: account.GetFetchedModelIDs(), FetchedAt: account.GetModelsFetchedAt(), Source: strings.TrimSpace(account.GetExtraString(AccountExtraModelsSourceKey)), Audit: BuildAccountModelDiscoveryAudit(account, account.GetFetchedModelIDs(), source, fetchedAt, "")}, nil
}
