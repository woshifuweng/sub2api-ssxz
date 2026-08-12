package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
)

func (s *AccountTestService) testOpenAICompactConnection(c gatewayctx.GatewayContext, account *Account, testModelID string) error {
	ctx := c.Request().Context()
	credentialAccount := account
	if account.IsShadow() {
		resolved, err := resolveCredentialAccount(ctx, s.accountRepo, account)
		if err != nil {
			return s.sendErrorAndEnd(c, "Failed to resolve account credentials")
		}
		credentialAccount = resolved
	}

	authToken := ""
	apiURL := ""
	isOAuth := false
	switch {
	case credentialAccount.IsOAuth():
		isOAuth = true
		if !credentialAccount.IsOpenAIAgentIdentity() {
			authToken = credentialAccount.GetOpenAIAccessToken()
		}
		if authToken == "" && !credentialAccount.IsOpenAIAgentIdentity() {
			return s.sendErrorAndEnd(c, "No access token available")
		}
		apiURL = chatgptCodexAPIURL + "/compact"
	case account.Type == AccountTypeAPIKey:
		authToken = account.GetOpenAIApiKey()
		if authToken == "" {
			return s.sendErrorAndEnd(c, "No API key available")
		}
		baseURL := account.GetOpenAIBaseURL()
		if baseURL == "" {
			baseURL = "https://api.openai.com"
		}
		normalizedBaseURL, err := s.validateUpstreamBaseURL(baseURL)
		if err != nil {
			return s.sendErrorAndEnd(c, fmt.Sprintf("Invalid base URL: %s", err.Error()))
		}
		apiURL = appendOpenAIResponsesRequestPathSuffix(buildOpenAIResponsesURL(normalizedBaseURL), "/compact")
	default:
		return s.sendErrorAndEnd(c, fmt.Sprintf("Unsupported account type: %s", account.Type))
	}

	c.SetHeader("Content-Type", "text/event-stream")
	c.SetHeader("Cache-Control", "no-cache")
	c.SetHeader("Connection", "keep-alive")
	c.SetHeader("X-Accel-Buffering", "no")
	_ = c.Flush()

	payloadBytes, _ := json.Marshal(createOpenAICompactProbePayload(testModelID))
	if !agentIdentityTaskRecoveryWasTried(ctx) {
		s.sendEvent(c, TestEvent{Type: "test_start", Model: testModelID})
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return s.sendErrorAndEnd(c, "Failed to create request")
	}
	req = req.WithContext(WithHTTPUpstreamProfile(req.Context(), HTTPUpstreamProfileOpenAI))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if credentialAccount.IsOpenAIAgentIdentity() {
		authHeaders, authErr := buildAgentIdentityAuthenticationHeaders(ctx, s.accountRepo, s.agentIdentityWS, &s.agentIdentityTaskMu, credentialAccount)
		if authErr != nil {
			return s.sendErrorAndEnd(c, "Failed to build Agent Identity authentication")
		}
		for key, values := range authHeaders {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}
	} else {
		req.Header.Set("Authorization", "Bearer "+authToken)
	}
	applyOpenAICodexProbeHeaders(req.Header)
	probeSessionID := compactProbeSessionID(account.ID)
	req.Header.Set("Session_ID", probeSessionID)
	req.Header.Set("Conversation_ID", probeSessionID)
	if isOAuth {
		req.Host = "chatgpt.com"
		setOpenAIChatGPTAccountHeaders(req.Header, credentialAccount)
	}
	account.ApplyHeaderOverrides(req.Header)

	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	var tlsProfile *tlsfingerprint.Profile
	if s.tlsFPProfileService != nil {
		tlsProfile = s.tlsFPProfileService.ResolveTLSProfile(account)
	}
	resp, err := s.httpUpstream.DoWithTLS(req, proxyURL, account.ID, account.Concurrency, tlsProfile)
	if err != nil {
		s.persistOpenAICompactProbeResult(ctx, account, nil, nil, err)
		return s.sendErrorAndEnd(c, fmt.Sprintf("Request failed: %s", err.Error()))
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	body = redactAgentIdentitySensitiveBodyForAccount(ctx, s.accountRepo, credentialAccount, body)
	if !agentIdentityTaskRecoveryWasTried(ctx) && credentialAccount.IsOpenAIAgentIdentity() && isAgentIdentityTaskInvalidHTTPResponse(resp.StatusCode, body) {
		expectedTaskID := credentialAccount.GetCredential("task_id")
		if err := ensureAgentIdentityTaskForAccount(ctx, s.accountRepo, s.agentIdentityWS, &s.agentIdentityTaskMu, credentialAccount, expectedTaskID); err != nil {
			return s.sendErrorAndEnd(c, fmt.Sprintf("Agent Identity task recovery failed: %s", err.Error()))
		}
		c.SetRequest(c.Request().WithContext(markAgentIdentityTaskRecoveryTried(ctx)))
		return s.testOpenAICompactConnection(c, account, testModelID)
	}

	s.persistOpenAICompactProbeResult(ctx, account, resp, body, nil)
	if resp.StatusCode == http.StatusTooManyRequests {
		s.reconcileOpenAICompact429State(ctx, account, resp.Header, body)
	}
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusUnauthorized && s.accountRepo != nil {
			_ = s.accountRepo.SetError(ctx, account.ID, fmt.Sprintf("Authentication failed (401): %s", string(body)))
		}
		return s.sendErrorAndEnd(c, fmt.Sprintf("API returned %d: %s", resp.StatusCode, string(body)))
	}

	s.sendEvent(c, TestEvent{Type: "content", Text: "Compact probe succeeded"})
	s.sendEvent(c, TestEvent{Type: "test_complete", Success: true})
	return nil
}

func (s *AccountTestService) persistOpenAICompactProbeResult(ctx context.Context, account *Account, resp *http.Response, body []byte, probeErr error) {
	if s == nil || s.accountRepo == nil || account == nil {
		return
	}
	updates := buildOpenAICompactProbeExtraUpdates(resp, body, probeErr, time.Now())
	if resp != nil {
		if codexUpdates, err := extractOpenAICodexProbeUpdates(resp); err == nil {
			updates = mergeExtraUpdates(updates, codexUpdates)
		}
	}
	if len(updates) > 0 {
		_ = s.accountRepo.UpdateExtra(ctx, account.ID, updates)
		mergeAccountExtra(account, updates)
	}
}

func (s *AccountTestService) reconcileOpenAICompact429State(ctx context.Context, account *Account, headers http.Header, body []byte) {
	if s == nil || s.accountRepo == nil || account == nil {
		return
	}
	persistOpenAI429PlanType(ctx, s.accountRepo, account, body)
	resetAt := calculateOpenAI429ResetTime(headers)
	if resetAt == nil {
		if unixTimestamp := parseOpenAIRateLimitResetTime(body); unixTimestamp != nil {
			value := time.Unix(*unixTimestamp, 0)
			resetAt = &value
		}
	}
	if resetAt == nil || s.accountRepo.SetRateLimited(ctx, account.ID, *resetAt) != nil {
		return
	}
	now := time.Now()
	account.RateLimitedAt = &now
	account.RateLimitResetAt = resetAt
}
