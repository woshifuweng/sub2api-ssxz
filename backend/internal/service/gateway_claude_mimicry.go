package service

import (
	"context"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
)

type claudeMimicMetadataBuilder func(fp *Fingerprint, body []byte) string

func (s *GatewayService) applyClaudeCodeOAuthMimicryToParsedRequestBody(
	ctx context.Context,
	c gatewayctx.GatewayContext,
	account *Account,
	parsed *ParsedRequest,
	body []byte,
	model string,
) ([]byte, string) {
	var systemRaw any
	var metadataBuilder claudeMimicMetadataBuilder
	if parsed != nil {
		if value, ok := parsed.SystemValue(); ok {
			systemRaw = value
		}
		metadataBuilder = func(fp *Fingerprint, _ []byte) string {
			return s.buildOAuthMetadataUserID(parsed, account, fp)
		}
	}
	return s.applyClaudeCodeOAuthMimicryToBodyContext(ctx, c, account, body, systemRaw, model, metadataBuilder)
}

// applyClaudeCodeOAuthMimicryToBodyContext applies the full Claude Code mimicry
// pipeline for non-Claude-Code clients using Anthropic OAuth accounts.
func (s *GatewayService) applyClaudeCodeOAuthMimicryToBodyContext(
	ctx context.Context,
	c gatewayctx.GatewayContext,
	account *Account,
	body []byte,
	systemRaw any,
	model string,
	metadataBuilder claudeMimicMetadataBuilder,
) ([]byte, string) {
	if account == nil || !account.IsOAuth() || len(body) == 0 {
		return body, model
	}

	systemRewritten := false
	if !strings.Contains(strings.ToLower(model), "haiku") && !systemIncludesClaudeCodePrompt(systemRaw) {
		body = injectClaudeCodePrompt(body, systemRaw)
		systemRewritten = true
	}

	normalizeOpts := claudeOAuthNormalizeOptions{stripSystemCacheControl: !systemRewritten}
	var fp *Fingerprint
	if s.identityService != nil {
		requestHeaders := http.Header{}
		if c != nil {
			if req := c.Request(); req != nil {
				requestHeaders = req.Header
			}
		}
		if got, err := s.identityService.GetOrCreateFingerprint(ctx, account.ID, requestHeaders); err == nil {
			fp = got
		}
	}
	metadataUserID := ""
	if metadataBuilder != nil {
		metadataUserID = metadataBuilder(fp, body)
	}
	if metadataUserID == "" {
		metadataUserID = s.buildOAuthMetadataUserIDFromBody(ctx, account, fp, body)
	}
	if metadataUserID != "" {
		normalizeOpts.injectMetadata = true
		normalizeOpts.metadataUserID = metadataUserID
	}

	body, model = normalizeClaudeOAuthRequestBody(body, model, normalizeOpts)

	body = stripMessageCacheControl(body)
	body = addMessageCacheBreakpoints(body)

	if rw := buildToolNameRewriteFromBody(body); rw != nil {
		body = applyToolNameRewriteToBody(body, rw)
		if c != nil {
			c.SetValue(toolNameRewriteKey, rw)
		}
	} else {
		body = applyToolsLastCacheBreakpoint(body)
	}

	return body, model
}
