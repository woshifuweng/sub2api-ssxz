package handler

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

func (h *OpenAIGatewayHandler) normalizeOpenAIResponsesCompactRequest(c *gin.Context, log *zap.Logger, body []byte) ([]byte, bool) {
	return h.normalizeOpenAIResponsesCompactRequestContext(gatewayctx.FromGin(c), log, body)
}

func (h *OpenAIGatewayHandler) normalizeOpenAIResponsesCompactRequestContext(c gatewayctx.GatewayContext, log *zap.Logger, body []byte) ([]byte, bool) {
	if c == nil || c.Request() == nil || c.Request().URL == nil {
		return body, true
	}

	path := strings.TrimRight(strings.TrimSpace(c.Request().URL.Path), "/")
	pathBasedCompact := isOpenAIRemoteCompactPathContext(c)
	bodySignalCompact := service.HasCompactionTriggerInInput(body) && isOpenAIResponsesRootPath(path)

	if bodySignalCompact && isOpenAIRemoteCompactionV2Request(body) {
		return body, true
	}

	if bodySignalCompact {
		clientWantsStream := gjson.GetBytes(body, "stream").Bool()
		c.Request().URL.Path = path + "/compact"
		pathBasedCompact = true
		if clientWantsStream {
			c.SetValue(service.OpenAICompactClientStreamKeyForTest(), true)
		}
	}

	if !pathBasedCompact {
		return body, true
	}

	normalized, _, err := service.NormalizeOpenAICompactRequestBodyForTest(body)
	if err != nil {
		if log != nil {
			log.Warn("openai_compact.normalize_request_failed", zap.Error(err))
		}
		return body, false
	}
	return normalized, true
}

func isOpenAIResponsesRootPath(path string) bool {
	switch path {
	case "/responses", "/v1/responses", "/openai/v1/responses", "/backend-api/codex/responses":
		return true
	default:
		return false
	}
}
