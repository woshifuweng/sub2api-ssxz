package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
)

func (s *GatewayService) ForwardCountTokensContext(ctx context.Context, c gatewayctx.GatewayContext, account *Account, parsed *ParsedRequest) error {
	if parsed == nil {
		s.countTokensErrorContext(c, http.StatusBadRequest, "invalid_request_error", "Request body is empty")
		return fmt.Errorf("parse request: empty request")
	}

	if account != nil && account.Platform == PlatformKiro {
		if s.kiroGatewayService == nil {
			return fmt.Errorf("kiro gateway service is not configured")
		}
		return s.kiroGatewayService.ForwardCountTokensContext(ctx, c, account, parsed)
	}

	if account != nil && account.IsAnthropicAPIKeyPassthroughEnabled() {
		passthroughBody := parsed.Body.Bytes()
		if reqModel := parsed.Model; reqModel != "" {
			if mappedModel := account.GetMappedModel(reqModel); mappedModel != reqModel {
				passthroughBody = s.replaceModelInBody(passthroughBody, mappedModel)
				logger.LegacyPrintf("service.gateway", "CountTokens passthrough model mapping: %s -> %s (account: %s)", reqModel, mappedModel, account.Name)
			}
		}
		return s.forwardCountTokensAnthropicAPIKeyPassthroughContext(ctx, c, account, passthroughBody)
	}

	// Bedrock 不支持 count_tokens 端点
	if account != nil && account.IsBedrock() {
		s.countTokensErrorContext(c, http.StatusNotFound, "not_found_error", "count_tokens endpoint is not supported for Bedrock")
		return nil
	}

	body := parsed.Body.Bytes()
	reqModel := parsed.Model

	isClaudeCode := isClaudeCodeRequestContext(ctx, c, parsed)
	shouldMimicClaudeCode := account.IsOAuth() && !isClaudeCode

	if shouldMimicClaudeCode {
		body, reqModel = s.applyClaudeCodeOAuthMimicryToParsedRequestBody(ctx, c, account, parsed, body, reqModel)
	}

	// Antigravity 账户不支持 count_tokens，返回 404 让客户端 fallback 到本地估算。
	// 返回 nil 避免 handler 层记录为错误，也不设置 ops 上游错误上下文。
	if account.Platform == PlatformAntigravity {
		s.countTokensErrorContext(c, http.StatusNotFound, "not_found_error", "count_tokens endpoint is not supported for this platform")
		return nil
	}

	// 应用模型映射：
	// - APIKey 账号：使用账号级别的显式映射（如果配置），否则透传原始模型名
	// - OAuth/SetupToken 账号：使用 Anthropic 标准映射（短ID → 长ID）
	if reqModel != "" {
		mappedModel := reqModel
		mappingSource := ""
		if account.Type == AccountTypeAPIKey {
			mappedModel = account.GetMappedModel(reqModel)
			if mappedModel != reqModel {
				mappingSource = "account"
			}
		}
		if mappingSource == "" && account.Platform == PlatformAnthropic && account.Type != AccountTypeAPIKey {
			normalized := claude.NormalizeModelID(reqModel)
			if normalized != reqModel {
				mappedModel = normalized
				mappingSource = "prefix"
			}
		}
		if mappedModel != reqModel {
			body = s.replaceModelInBody(body, mappedModel)
			reqModel = mappedModel
			logger.LegacyPrintf("service.gateway", "CountTokens model mapping applied: %s -> %s (account: %s, source=%s)", parsed.Model, mappedModel, account.Name, mappingSource)
		}
	}

	// 获取凭证
	token, tokenType, err := s.GetAccessToken(ctx, account)
	if err != nil {
		s.countTokensErrorContext(c, http.StatusBadGateway, "upstream_error", "Failed to get access token")
		return err
	}

	// 构建上游请求
	upstreamReq, err := s.buildCountTokensRequestContext(ctx, c, account, body, token, tokenType, reqModel, shouldMimicClaudeCode)
	if err != nil {
		s.countTokensErrorContext(c, http.StatusInternalServerError, "api_error", "Failed to build request")
		return err
	}

	// 获取代理URL
	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}

	// 发送请求
	upstreamReq = s.applySampledTLSFingerprintProfile(upstreamReq, account)
	resp, err := s.httpUpstream.DoWithTLS(upstreamReq, proxyURL, account.ID, account.Concurrency, s.resolveTLSFingerprintProfile(account))
	if err != nil {
		setOpsUpstreamErrorContext(c, 0, sanitizeUpstreamErrorMessage(err.Error()), "")
		s.countTokensErrorContext(c, http.StatusBadGateway, "upstream_error", "Request failed")
		return fmt.Errorf("upstream request failed: %w", err)
	}

	// 读取响应体
	maxReadBytes := resolveUpstreamResponseReadLimit(s.cfg)
	respBody, err := readUpstreamResponseBodyLimited(resp.Body, maxReadBytes)
	_ = resp.Body.Close()
	if err != nil {
		if errors.Is(err, ErrUpstreamResponseBodyTooLarge) {
			setOpsUpstreamErrorContext(c, http.StatusBadGateway, "upstream response too large", "")
			s.countTokensErrorContext(c, http.StatusBadGateway, "upstream_error", "Upstream response too large")
			return err
		}
		s.countTokensErrorContext(c, http.StatusBadGateway, "upstream_error", "Failed to read response")
		return err
	}

	// 检测 thinking block 签名错误（400）并重试一次（过滤 thinking blocks）
	if resp.StatusCode == 400 && s.isThinkingBlockSignatureError(respBody) && s.settingService.IsSignatureRectifierEnabled(ctx) {
		logger.LegacyPrintf("service.gateway", "Account %d: detected thinking block signature error on count_tokens, retrying with filtered thinking blocks", account.ID)

		filteredBody := FilterThinkingBlocksForRetry(body, reqModel)
		retryReq, buildErr := s.buildCountTokensRequestContext(ctx, c, account, filteredBody, token, tokenType, reqModel, shouldMimicClaudeCode)
		if buildErr == nil {
			retryReq = s.applySampledTLSFingerprintProfile(retryReq, account)
			retryResp, retryErr := s.httpUpstream.DoWithTLS(retryReq, proxyURL, account.ID, account.Concurrency, s.resolveTLSFingerprintProfile(account))
			if retryErr == nil {
				resp = retryResp
				respBody, err = readUpstreamResponseBodyLimited(resp.Body, maxReadBytes)
				_ = resp.Body.Close()
				if err != nil {
					if errors.Is(err, ErrUpstreamResponseBodyTooLarge) {
						setOpsUpstreamErrorContext(c, http.StatusBadGateway, "upstream response too large", "")
						s.countTokensErrorContext(c, http.StatusBadGateway, "upstream_error", "Upstream response too large")
						return err
					}
					s.countTokensErrorContext(c, http.StatusBadGateway, "upstream_error", "Failed to read response")
					return err
				}
			}
		}
	}

	// 处理错误响应
	if resp.StatusCode >= 400 {
		// 标记账号状态（429/529等）
		s.rateLimitService.HandleUpstreamError(ctx, account, resp.StatusCode, resp.Header, respBody)

		upstreamMsg := strings.TrimSpace(extractUpstreamErrorMessage(respBody))
		upstreamMsg = sanitizeUpstreamErrorMessage(upstreamMsg)
		upstreamDetail := ""
		if s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
			maxBytes := s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes
			if maxBytes <= 0 {
				maxBytes = 2048
			}
			upstreamDetail = truncateString(string(respBody), maxBytes)
		}
		setOpsUpstreamErrorContext(c, resp.StatusCode, upstreamMsg, upstreamDetail)

		// 记录上游错误摘要便于排障（不回显请求内容）
		if s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
			logger.LegacyPrintf("service.gateway",
				"count_tokens upstream error %d (account=%d platform=%s type=%s): %s",
				resp.StatusCode,
				account.ID,
				account.Platform,
				account.Type,
				truncateForLog(respBody, s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes),
			)
		}

		// 返回简化的错误响应
		errMsg := "Upstream request failed"
		switch resp.StatusCode {
		case 429:
			errMsg = "Rate limit exceeded"
		case 529:
			errMsg = "Service overloaded"
		}
		s.countTokensErrorContext(c, resp.StatusCode, "upstream_error", errMsg)
		if upstreamMsg == "" {
			return fmt.Errorf("upstream error: %d", resp.StatusCode)
		}
		return fmt.Errorf("upstream error: %d message=%s", resp.StatusCode, upstreamMsg)
	}

	// 透传成功响应
	if c != nil {
		c.SetHeader("Content-Type", "application/json")
		_, err = c.WriteBytes(resp.StatusCode, respBody)
		return err
	}
	return nil
}

func (s *GatewayService) forwardCountTokensAnthropicAPIKeyPassthroughContext(ctx context.Context, c gatewayctx.GatewayContext, account *Account, body []byte) error {
	token, tokenType, err := s.GetAccessToken(ctx, account)
	if err != nil {
		s.countTokensErrorContext(c, http.StatusBadGateway, "upstream_error", "Failed to get access token")
		return err
	}
	if tokenType != "apikey" {
		s.countTokensErrorContext(c, http.StatusBadGateway, "upstream_error", "Invalid account token type")
		return fmt.Errorf("anthropic api key passthrough requires apikey token, got: %s", tokenType)
	}

	upstreamReq, err := s.buildCountTokensRequestAnthropicAPIKeyPassthroughContext(ctx, c, account, body, token)
	if err != nil {
		s.countTokensErrorContext(c, http.StatusInternalServerError, "api_error", "Failed to build request")
		return err
	}

	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}

	upstreamReq = s.applySampledTLSFingerprintProfile(upstreamReq, account)
	resp, err := s.httpUpstream.DoWithTLS(upstreamReq, proxyURL, account.ID, account.Concurrency, s.resolveTLSFingerprintProfile(account))
	if err != nil {
		setOpsUpstreamErrorContext(c, 0, sanitizeUpstreamErrorMessage(err.Error()), "")
		appendOpsUpstreamErrorContext(c, OpsUpstreamErrorEvent{
			Platform:           account.Platform,
			AccountID:          account.ID,
			AccountName:        account.Name,
			UpstreamStatusCode: 0,
			Passthrough:        true,
			Kind:               "request_error",
			Message:            sanitizeUpstreamErrorMessage(err.Error()),
		})
		s.countTokensErrorContext(c, http.StatusBadGateway, "upstream_error", "Request failed")
		return fmt.Errorf("upstream request failed: %w", err)
	}

	maxReadBytes := resolveUpstreamResponseReadLimit(s.cfg)
	respBody, err := readUpstreamResponseBodyLimited(resp.Body, maxReadBytes)
	_ = resp.Body.Close()
	if err != nil {
		if errors.Is(err, ErrUpstreamResponseBodyTooLarge) {
			setOpsUpstreamErrorContext(c, http.StatusBadGateway, "upstream response too large", "")
			s.countTokensErrorContext(c, http.StatusBadGateway, "upstream_error", "Upstream response too large")
			return err
		}
		s.countTokensErrorContext(c, http.StatusBadGateway, "upstream_error", "Failed to read response")
		return err
	}

	if resp.StatusCode >= 400 {
		if s.rateLimitService != nil {
			s.rateLimitService.HandleUpstreamError(ctx, account, resp.StatusCode, resp.Header, respBody)
		}

		upstreamMsg := strings.TrimSpace(extractUpstreamErrorMessage(respBody))
		upstreamMsg = sanitizeUpstreamErrorMessage(upstreamMsg)

		// 中转站不支持 count_tokens 端点时（404），返回 404 让客户端 fallback 到本地估算。
		// 仅在错误消息明确指向 count_tokens endpoint 不存在时生效，避免误吞其他 404（如错误 base_url）。
		// 返回 nil 避免 handler 层记录为错误，也不设置 ops 上游错误上下文。
		if isCountTokensUnsupported404(resp.StatusCode, respBody) {
			logger.LegacyPrintf("service.gateway",
				"[count_tokens] Upstream does not support count_tokens (404), returning 404: account=%d name=%s msg=%s",
				account.ID, account.Name, truncateString(upstreamMsg, 512))
			s.countTokensErrorContext(c, http.StatusNotFound, "not_found_error", "count_tokens endpoint is not supported by upstream")
			return nil
		}

		upstreamDetail := ""
		if s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
			maxBytes := s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes
			if maxBytes <= 0 {
				maxBytes = 2048
			}
			upstreamDetail = truncateString(string(respBody), maxBytes)
		}
		setOpsUpstreamErrorContext(c, resp.StatusCode, upstreamMsg, upstreamDetail)
		appendOpsUpstreamErrorContext(c, OpsUpstreamErrorEvent{
			Platform:           account.Platform,
			AccountID:          account.ID,
			AccountName:        account.Name,
			UpstreamStatusCode: resp.StatusCode,
			UpstreamRequestID:  resp.Header.Get("x-request-id"),
			Passthrough:        true,
			Kind:               "http_error",
			Message:            upstreamMsg,
			Detail:             upstreamDetail,
		})

		errMsg := "Upstream request failed"
		switch resp.StatusCode {
		case 429:
			errMsg = "Rate limit exceeded"
		case 529:
			errMsg = "Service overloaded"
		}
		s.countTokensErrorContext(c, resp.StatusCode, "upstream_error", errMsg)
		if upstreamMsg == "" {
			return fmt.Errorf("upstream error: %d", resp.StatusCode)
		}
		return fmt.Errorf("upstream error: %d message=%s", resp.StatusCode, upstreamMsg)
	}

	if c != nil {
		writeAnthropicPassthroughResponseHeaders(c.Header(), resp.Header, s.responseHeaderFilter)
	}
	contentType := strings.TrimSpace(resp.Header.Get("Content-Type"))
	if contentType == "" {
		contentType = "application/json"
	}
	if c != nil {
		c.SetHeader("Content-Type", contentType)
		_, err = c.WriteBytes(resp.StatusCode, respBody)
		return err
	}
	return nil
}

func (s *GatewayService) buildCountTokensRequestAnthropicAPIKeyPassthroughContext(
	ctx context.Context,
	c gatewayctx.GatewayContext,
	account *Account,
	body []byte,
	token string,
) (*http.Request, error) {
	targetURL := claudeAPICountTokensURL
	baseURL := account.GetBaseURL()
	if baseURL != "" {
		validatedURL, err := s.validateUpstreamBaseURL(baseURL)
		if err != nil {
			return nil, err
		}
		targetURL = validatedURL + "/v1/messages/count_tokens?beta=true"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	if c != nil {
		if reqCtx := c.Request(); reqCtx != nil {
			for key, values := range reqCtx.Header {
				lowerKey := strings.ToLower(strings.TrimSpace(key))
				if !allowedHeaders[lowerKey] {
					continue
				}
				for _, v := range values {
					req.Header.Add(key, v)
				}
			}
		}
	}

	req.Header.Del("authorization")
	req.Header.Del("x-api-key")
	req.Header.Del("x-goog-api-key")
	req.Header.Del("cookie")
	req.Header.Set("x-api-key", token)

	if req.Header.Get("content-type") == "" {
		req.Header.Set("content-type", "application/json")
	}
	if req.Header.Get("anthropic-version") == "" {
		req.Header.Set("anthropic-version", "2023-06-01")
	}

	return req, nil
}

func (s *GatewayService) buildCountTokensRequestContext(ctx context.Context, c gatewayctx.GatewayContext, account *Account, body []byte, token, tokenType, modelID string, mimicClaudeCode bool) (*http.Request, error) {
	// 确定目标 URL
	targetURL := claudeAPICountTokensURL
	if account.Type == AccountTypeAPIKey {
		baseURL := account.GetBaseURL()
		if baseURL != "" {
			validatedURL, err := s.validateUpstreamBaseURL(baseURL)
			if err != nil {
				return nil, err
			}
			targetURL = validatedURL + "/v1/messages/count_tokens?beta=true"
		}
	}

	clientHeaders := http.Header{}
	if c != nil {
		if req := c.Request(); req != nil {
			clientHeaders = req.Header
		}
	}

	// OAuth 账号：应用统一指纹和重写 userID
	// 如果启用了会话ID伪装，会在重写后替换 session 部分为固定值
	if account.IsOAuth() && s.identityService != nil {
		if IsClaudeCodeClient(ctx) {
			_, _ = s.identityService.ObserveOfficialFingerprintSample(ctx, account.ID, clientHeaders)
		}
		fp, err := s.identityService.GetOrCreateFingerprint(ctx, account.ID, clientHeaders)
		if err == nil {
			accountUUID := account.GetExtraString("account_uuid")
			if accountUUID != "" && fp.ClientID != "" {
				if newBody, err := s.identityService.RewriteUserIDWithMasking(ctx, body, account, accountUUID, fp.ClientID, fp.UserAgent); err == nil && len(newBody) > 0 {
					body = newBody
				}
			}
		}
	}

	req, err := http.NewRequestWithContext(ctx, "POST", targetURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	// 设置认证头
	if tokenType == "oauth" {
		req.Header.Set("authorization", "Bearer "+token)
	} else {
		req.Header.Set("x-api-key", token)
	}

	// 白名单透传 headers
	for key, values := range clientHeaders {
		lowerKey := strings.ToLower(key)
		if allowedHeaders[lowerKey] {
			for _, v := range values {
				req.Header.Add(key, v)
			}
		}
	}

	// OAuth 账号：应用指纹到请求头
	if account.IsOAuth() && s.identityService != nil {
		fp, _ := s.identityService.GetOrCreateFingerprint(ctx, account.ID, clientHeaders)
		if fp != nil {
			s.identityService.ApplyAnthropicFingerprint(req, body, account, fp, IsClaudeCodeClient(ctx))
		}
	}

	// 确保必要的 headers 存在
	if req.Header.Get("content-type") == "" {
		req.Header.Set("content-type", "application/json")
	}
	if req.Header.Get("anthropic-version") == "" {
		req.Header.Set("anthropic-version", "2023-06-01")
	}
	if tokenType == "oauth" {
		applyClaudeOAuthHeaderDefaults(req)
	}
	ensureAnthropicClientRequestID(req)

	// Build effective drop set for count_tokens: merge static defaults with dynamic beta policy filter rules
	ctEffectiveDropSet := mergeDropSets(s.getBetaPolicyFilterSetContext(ctx, c, account, modelID))

	// OAuth 账号：处理 anthropic-beta header
	if tokenType == "oauth" {
		if mimicClaudeCode {
			applyClaudeCodeMimicHeaders(req, false)

			incomingBeta := req.Header.Get("anthropic-beta")
			requiredBetas := []string{claude.BetaClaudeCode, claude.BetaOAuth, claude.BetaInterleavedThinking, claude.BetaTokenCounting}
			req.Header.Set("anthropic-beta", mergeAnthropicBetaDropping(requiredBetas, incomingBeta, ctEffectiveDropSet))
		} else {
			clientBetaHeader := req.Header.Get("anthropic-beta")
			if clientBetaHeader == "" {
				req.Header.Set("anthropic-beta", claude.CountTokensBetaHeader)
			} else {
				beta := s.getBetaHeader(modelID, clientBetaHeader)
				if !strings.Contains(beta, claude.BetaTokenCounting) {
					beta = beta + "," + claude.BetaTokenCounting
				}
				req.Header.Set("anthropic-beta", stripBetaTokensWithSet(beta, ctEffectiveDropSet))
			}
		}
	} else {
		// API-key accounts: apply beta policy filter to strip controlled tokens
		if existingBeta := req.Header.Get("anthropic-beta"); existingBeta != "" {
			req.Header.Set("anthropic-beta", stripBetaTokensWithSet(existingBeta, ctEffectiveDropSet))
		} else if s.cfg != nil && s.cfg.Gateway.InjectBetaForAPIKey {
			// API-key：与 messages 同步的按需 beta 注入（默认关闭）
			if requestNeedsBetaFeatures(body) {
				if beta := defaultAPIKeyBetaHeader(body); beta != "" {
					req.Header.Set("anthropic-beta", beta)
				}
			}
		}
	}

	if c != nil && tokenType == "oauth" {
		c.SetValue(claudeMimicDebugInfoKey, buildClaudeMimicDebugLine(req, body, account, tokenType, mimicClaudeCode))
	}
	if s.debugClaudeMimicEnabled() {
		logClaudeMimicDebug(req, body, account, tokenType, mimicClaudeCode)
	}

	return req, nil
}

func (s *GatewayService) countTokensErrorContext(c gatewayctx.GatewayContext, status int, errType, message string) {
	if c == nil {
		return
	}
	c.WriteJSON(status, map[string]any{
		"type": "error",
		"error": map[string]any{
			"type":    errType,
			"message": message,
		},
	})
}
