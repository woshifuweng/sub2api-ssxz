package service

import (
	"context"
	"errors"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/gin-gonic/gin"
)

// ForwardAsChatCompletionsContext adapts the gateway-context handler path to
// the legacy OpenAI service implementation used by this branch.
func (s *OpenAIGatewayService) ForwardAsChatCompletionsContext(
	ctx context.Context,
	c gatewayctx.GatewayContext,
	account *Account,
	body []byte,
	promptCacheKey string,
	defaultMappedModel string,
) (*OpenAIForwardResult, error) {
	native, ok := c.Native().(*gin.Context)
	if !ok || native == nil {
		return nil, errors.New("openai chat forwarding requires a gin context")
	}
	if account != nil && account.IsOpenAIChatWebMode() {
		return s.ForwardAsChatCompletions(ctx, native, account, body, promptCacheKey, defaultMappedModel)
	}
	return s.ForwardContext(ctx, c, account, body, defaultMappedModel)
}

// ForwardAsAnthropicContext adapts the gateway-context handler path to the
// legacy Anthropic bridge implementation.
func (s *OpenAIGatewayService) ForwardAsAnthropicContext(
	ctx context.Context,
	c gatewayctx.GatewayContext,
	account *Account,
	body []byte,
	promptCacheKey string,
	defaultMappedModel string,
) (*OpenAIForwardResult, error) {
	native, ok := c.Native().(*gin.Context)
	if !ok || native == nil {
		return nil, errors.New("anthropic forwarding requires a gin context")
	}
	return s.ForwardAsAnthropic(ctx, native, account, body, promptCacheKey, defaultMappedModel)
}

// shouldNormalizeChatCompatModel reports whether a chat-compatible request
// should pass through the shared Codex model normalizer. Explicit mappings and
// image models are already resolved, while ordinary GPT-5/Codex names need
// the same normalization as the Responses path.
func shouldNormalizeChatCompatModel(model string, mappingMatched bool, defaultMappedModel string) bool {
	if mappingMatched || strings.TrimSpace(defaultMappedModel) != "" {
		return true
	}
	trimmed := strings.TrimSpace(model)
	if trimmed == "" || isOpenAIImageGenerationModel(trimmed) {
		return true
	}
	segment := strings.ToLower(lastOpenAIModelSegment(trimmed))
	if getNormalizedCodexModel(segment) != "" {
		return true
	}
	return strings.Contains(segment, "gpt-5") ||
		strings.Contains(segment, "gpt 5") ||
		strings.Contains(segment, "codex")
}
