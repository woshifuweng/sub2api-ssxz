package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestWorkspaceOpenAICompatibleHTTPUpstreamSuccessReturnsContentUsageAndAudit(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		require.Equal(t, "/v1/chat/completions", r.URL.Path)
		require.Equal(t, "Bearer test-secret-value", r.Header.Get("Authorization"))
		var req struct {
			Model    string `json:"model"`
			Stream   bool   `json:"stream"`
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		require.Equal(t, "deepseek-v4-flash", req.Model)
		require.False(t, req.Stream)
		require.Len(t, req.Messages, 1)
		require.Equal(t, WorkspaceRoleUser, req.Messages[0].Role)
		require.NotEmpty(t, req.Messages[0].Content)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"model":"deepseek-v4-flash",
			"choices":[{"message":{"role":"assistant","content":"STAGING_OK"}}],
			"usage":{"prompt_tokens":4,"completion_tokens":2,"total_tokens":6}
		}`))
	}))
	defer server.Close()

	upstream := WorkspaceOpenAICompatibleHTTPUpstream{
		BaseURL:        server.URL,
		Model:          "deepseek-v4-flash",
		APIKey:         "test-secret-value",
		ProviderLabel:  "deepseek-staging",
		EndpointLabel:  "deepseek-staging",
		Timeout:        3 * time.Second,
		allowTestLocal: true,
	}
	executor := workspaceOpenAICompatibleExecutorForTest(upstream)

	result, err := executor.ExecuteWorkspaceTextProvider(context.Background(), validWorkspaceOpenAICompatibleExecutionInput())
	require.NoError(t, err)

	require.Equal(t, 1, requests)
	require.Equal(t, "STAGING_OK", result.Content)
	require.Equal(t, "deepseek-v4-flash", result.UpstreamModel)
	require.Equal(t, "deepseek-staging", result.ProviderName)
	require.Equal(t, 6, result.TokenUsage.TotalTokens)
	require.Equal(t, true, result.Metadata["provider_called"])
	require.Equal(t, false, result.Metadata["billing_touched"])
	require.NotEmpty(t, result.Metadata["endpoint_label"])
	require.NotEmpty(t, result.Metadata["audit_prompt_hash"])
	assertWorkspaceOpenAICompatiblePayloadSafe(t, result)
}

func TestWorkspaceOpenAICompatibleHTTPUpstreamProviderErrorIsSanitized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"Authorization: Bearer redacted-provider-token internal.example sql stack"}`))
	}))
	defer server.Close()

	upstream := WorkspaceOpenAICompatibleHTTPUpstream{
		BaseURL:        server.URL,
		Model:          "deepseek-v4-flash",
		APIKey:         "test-secret-value",
		ProviderLabel:  "deepseek-staging",
		EndpointLabel:  "deepseek-staging",
		Timeout:        3 * time.Second,
		allowTestLocal: true,
	}
	executor := workspaceOpenAICompatibleExecutorForTest(upstream)

	result, err := executor.ExecuteWorkspaceTextProvider(context.Background(), validWorkspaceOpenAICompatibleExecutionInput())
	require.Error(t, err)
	require.ErrorIs(t, err, errWorkspaceOpenAICompatibleProviderFailed)
	require.Equal(t, true, result.Metadata["provider_called"])
	require.Equal(t, "failed", result.Metadata["status"])
	assertWorkspaceOpenAICompatiblePayloadSafe(t, result)
	assertWorkspaceOpenAICompatiblePayloadSafe(t, err.Error())
}

func TestWorkspaceOpenAICompatibleHTTPUpstreamTimeoutIsSanitized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		time.Sleep(200 * time.Millisecond)
	}))
	defer server.Close()

	upstream := WorkspaceOpenAICompatibleHTTPUpstream{
		BaseURL:        server.URL,
		Model:          "deepseek-v4-flash",
		APIKey:         "test-secret-value",
		ProviderLabel:  "deepseek-staging",
		EndpointLabel:  "deepseek-staging",
		Timeout:        10 * time.Millisecond,
		allowTestLocal: true,
	}
	executor := workspaceOpenAICompatibleExecutorForTest(upstream)

	result, err := executor.ExecuteWorkspaceTextProvider(context.Background(), validWorkspaceOpenAICompatibleExecutionInput())
	require.Error(t, err)
	require.ErrorIs(t, err, errWorkspaceOpenAICompatibleProviderFailed)
	require.Equal(t, true, result.Metadata["provider_called"])
	assertWorkspaceOpenAICompatiblePayloadSafe(t, result)
}

func TestWorkspaceOpenAICompatibleHTTPUpstreamFromConfigFailsClosedWithoutSecretsOrURL(t *testing.T) {
	cfg := workspaceTextProviderExecutorWiringConfig()
	decision := BuildWorkspaceTextProviderGateDecision(cfg)

	require.Nil(t, NewWorkspaceOpenAICompatibleHTTPUpstreamFromConfig(cfg, decision))

	cfg.Workspace.TextProvider.OpenAICompatible.BaseURL = "https://api.provider.example"
	require.Nil(t, NewWorkspaceOpenAICompatibleHTTPUpstreamFromConfig(cfg, decision))

	cfg.Workspace.TextProvider.OpenAICompatible.APIKey = "test-secret-value"
	require.Nil(t, NewWorkspaceOpenAICompatibleHTTPUpstreamFromConfig(cfg, decision))
}

func TestWorkspaceOpenAICompatibleHTTPUpstreamRejectsUnsafeRuntimeBaseURLs(t *testing.T) {
	cfg := workspaceTextProviderExecutorWiringConfig()
	cfg.Workspace.TextProvider.OpenAICompatible.APIKey = "test-secret-value"
	cfg.Workspace.TextProvider.OpenAICompatible.Model = "deepseek-v4-flash"
	decision := BuildWorkspaceTextProviderGateDecision(cfg)

	for _, rawURL := range []string{
		"http://api.provider.example",
		"https://localhost",
		"https://127.0.0.1",
		"https://10.0.0.10",
		"https://169.254.169.254",
		"https://api.provider.example/v1",
		"https://user:pass@api.provider.example",
		"https://api.provider.example?x=1",
	} {
		t.Run(rawURL, func(t *testing.T) {
			cfg.Workspace.TextProvider.OpenAICompatible.BaseURL = rawURL
			require.Nil(t, NewWorkspaceOpenAICompatibleHTTPUpstreamFromConfig(cfg, decision))
		})
	}
}

func TestWorkspaceOpenAICompatibleHTTPUpstreamRuntimeWiringRespectsGateAndUserMetadataCannotOverrideBaseURL(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		require.Equal(t, "/v1/chat/completions", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"model":"deepseek-v4-flash",
			"choices":[{"message":{"role":"assistant","content":"STAGING_OK"}}],
			"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}
		}`))
	}))
	defer server.Close()

	cfg := workspaceTextProviderExecutorWiringConfig()
	cfg.Workspace.TextProvider.OpenAICompatible = config.WorkspaceOpenAICompatibleConfig{
		BaseURL:        server.URL,
		Model:          "deepseek-v4-flash",
		APIKey:         "test-secret-value",
		TimeoutSeconds: 3,
	}
	adapter := NewWorkspaceTextProviderAdapterFromConfigWithExecutorProvider(cfg, func(cfg *config.Config, decision WorkspaceTextProviderGateDecision) WorkspaceTextProviderExecutor {
		upstream := newWorkspaceOpenAICompatibleHTTPUpstreamFromConfig(cfg, decision, true)
		return NewWorkspaceOpenAICompatibleTextExecutorFromConfig(cfg, decision, upstream)
	})
	repo := newMemoryChatWorkspaceRepo()
	svc := NewChatWorkspaceServiceWithProviderAdapter(repo, adapter)
	conversation, err := svc.CreateConversation(context.Background(), 10, WorkspaceCreateConversationInput{})
	require.NoError(t, err)
	input := validWorkspaceTextAppendInput(conversation.ID)
	input.Model = "deepseek-v4-flash"
	input.Metadata = map[string]any{"base_url": "https://evil.example"}

	_, assistantMessage, err := svc.AppendMessageWithAssistantResponse(context.Background(), 10, input)
	require.NoError(t, err)

	require.Equal(t, 1, requests)
	require.Equal(t, "STAGING_OK", assistantMessage.Content)
	require.Equal(t, WorkspaceMessageStatusCompleted, assistantMessage.Status)
	require.Equal(t, true, assistantMessage.Metadata["provider_called"])
	require.Equal(t, "deepseek-staging", assistantMessage.Metadata["provider_name"])
	assertWorkspaceOpenAICompatiblePayloadSafe(t, assistantMessage)
	payload, err := json.Marshal(assistantMessage)
	require.NoError(t, err)
	require.NotContains(t, strings.ToLower(string(payload)), "evil.example")
}
