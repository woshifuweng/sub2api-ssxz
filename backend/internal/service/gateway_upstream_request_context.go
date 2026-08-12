package service

import (
	"bytes"
	"context"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
)

func (s *GatewayService) buildUpstreamRequestContext(ctx context.Context, c gatewayctx.GatewayContext, account *Account, body []byte, token, tokenType, modelID string, reqStream bool, mimicClaudeCode bool) (*http.Request, error) {
	// 确定目标URL
	targetURL := claudeAPIURL
	if account.Type == AccountTypeAPIKey {
		baseURL := account.GetBaseURL()
		if baseURL != "" {
			validatedURL, err := s.validateUpstreamBaseURL(baseURL)
			if err != nil {
				return nil, err
			}
			targetURL = validatedURL + "/v1/messages?beta=true"
		}
	}

	clientHeaders := http.Header{}
	if c != nil {
		if req := c.Request(); req != nil {
			clientHeaders = req.Header
		}
	}

	// OAuth账号：应用统一指纹
	var fingerprint *Fingerprint
	if account.IsOAuth() && s.identityService != nil {
		if IsClaudeCodeClient(ctx) {
			_, _ = s.identityService.ObserveOfficialFingerprintSample(ctx, account.ID, clientHeaders)
		}
		// 1. 获取或创建指纹（包含随机生成的ClientID）
		fp, err := s.identityService.GetOrCreateFingerprint(ctx, account.ID, clientHeaders)
		if err != nil {
			logger.LegacyPrintf("service.gateway", "Warning: failed to get fingerprint for account %d: %v", account.ID, err)
			// 失败时降级为透传原始headers
		} else {
			fingerprint = fp

			// 2. 重写metadata.user_id（需要指纹中的ClientID和账号的account_uuid）
			// 如果启用了会话ID伪装，会在重写后替换 session 部分为固定值
			accountUUID := account.GetExtraString("account_uuid")
			if accountUUID != "" && fp.ClientID != "" {
				if newBody, err := s.identityService.RewriteUserIDWithMasking(ctx, body, account, accountUUID, fp.ClientID, fp.UserAgent); err == nil && len(newBody) > 0 {
					body = newBody
				}
			}
		}
	}

	// === DEBUG: 打印转发给上游的 body（metadata 已重写） ===
	debugLogRequestBody("UPSTREAM_FORWARD", body)

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

	// 白名单透传headers
	for key, values := range clientHeaders {
		lowerKey := strings.ToLower(key)
		if allowedHeaders[lowerKey] {
			for _, v := range values {
				req.Header.Add(key, v)
			}
		}
	}

	// OAuth账号：应用缓存的指纹到请求头（覆盖白名单透传的头）
	if fingerprint != nil {
		s.identityService.ApplyAnthropicFingerprint(req, body, account, fingerprint, IsClaudeCodeClient(ctx))
	}

	// 确保必要的headers存在
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

	// Build effective drop set: merge static defaults with dynamic beta policy filter rules
	policyFilterSet := s.getBetaPolicyFilterSetContext(ctx, c, account, modelID)
	effectiveDropSet := mergeDropSets(policyFilterSet)
	effectiveDropWithClaudeCodeSet := mergeDropSets(policyFilterSet, claude.BetaClaudeCode)

	// 处理 anthropic-beta header（OAuth 账号需要包含 oauth beta）
	if tokenType == "oauth" {
		if mimicClaudeCode {
			// 非 Claude Code 客户端：按 opencode 的策略处理：
			// - 强制 Claude Code 指纹相关请求头（尤其是 user-agent/x-stainless/x-app）
			// - 保留 incoming beta 的同时，确保 OAuth 所需 beta 存在
			applyClaudeCodeMimicHeaders(req, reqStream)

			incomingBeta := req.Header.Get("anthropic-beta")
			// Match real Claude CLI traffic (per mitmproxy reports):
			// messages requests typically use only oauth + interleaved-thinking.
			// Also drop claude-code beta if a downstream client added it.
			requiredBetas := []string{claude.BetaOAuth, claude.BetaInterleavedThinking}
			req.Header.Set("anthropic-beta", mergeAnthropicBetaDropping(requiredBetas, incomingBeta, effectiveDropWithClaudeCodeSet))
		} else {
			// Claude Code 客户端：尽量透传原始 header，仅补齐 oauth beta
			clientBetaHeader := req.Header.Get("anthropic-beta")
			req.Header.Set("anthropic-beta", stripBetaTokensWithSet(s.getBetaHeader(modelID, clientBetaHeader), effectiveDropSet))
		}
	} else {
		// API-key accounts: apply beta policy filter to strip controlled tokens
		if existingBeta := req.Header.Get("anthropic-beta"); existingBeta != "" {
			req.Header.Set("anthropic-beta", stripBetaTokensWithSet(existingBeta, effectiveDropSet))
		} else if s.cfg != nil && s.cfg.Gateway.InjectBetaForAPIKey {
			// API-key：仅在请求显式使用 beta 特性且客户端未提供时，按需补齐（默认关闭）
			if requestNeedsBetaFeatures(body) {
				if beta := defaultAPIKeyBetaHeader(body); beta != "" {
					req.Header.Set("anthropic-beta", beta)
				}
			}
		}
	}

	// Always capture a compact fingerprint line for later error diagnostics.
	// We only print it when needed (or when the explicit debug flag is enabled).
	if c != nil && tokenType == "oauth" {
		c.SetValue(claudeMimicDebugInfoKey, buildClaudeMimicDebugLine(req, body, account, tokenType, mimicClaudeCode))
	}
	if s.debugClaudeMimicEnabled() {
		logClaudeMimicDebug(req, body, account, tokenType, mimicClaudeCode)
	}

	return req, nil
}

func (s *GatewayService) getBetaPolicyFilterSetContext(ctx context.Context, c gatewayctx.GatewayContext, account *Account, model string) map[string]struct{} {
	if c != nil {
		if v, ok := c.Value(betaPolicyFilterSetKey); ok {
			if filterSet, ok := v.(map[string]struct{}); ok {
				return filterSet
			}
		}
	}
	return s.evaluateBetaPolicy(ctx, "", account, model).filterSet
}

// getBetaHeader 处理anthropic-beta header
// 对于OAuth账号，需要确保包含oauth-2025-04-20
