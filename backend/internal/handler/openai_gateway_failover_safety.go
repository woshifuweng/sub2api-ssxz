package handler

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

func openAIForwardMayFailover(c *gin.Context, writerSizeBeforeForward int, failoverErr *service.UpstreamFailoverError) bool {
	return openAIForwardMayFailoverContext(gatewayctx.FromGin(c), writerSizeBeforeForward, failoverErr)
}

func openAIForwardErrorAlreadyCommunicated(c *gin.Context, writerSizeBeforeForward int, err error) bool {
	if err == nil || c == nil || c.Writer == nil {
		return false
	}
	if service.OpenAICompactKeepaliveAdjustedWrittenSize(c) == writerSizeBeforeForward ||
		service.OpenAIImagesJSONKeepaliveAdjustedWrittenSize(c) == writerSizeBeforeForward {
		return false
	}
	if service.GetOpsCyberPolicy(c) != nil {
		return true
	}
	message := strings.TrimSpace(err.Error())
	for _, prefix := range []string{
		"upstream response failed:",
		"non-streaming openai protocol error:",
	} {
		if strings.HasPrefix(message, prefix) {
			return true
		}
	}
	return false
}

func openAIForwardMayFailoverContext(c gatewayctx.GatewayContext, writerSizeBeforeForward int, failoverErr *service.UpstreamFailoverError) bool {
	if c == nil || service.RequestPayloadStarted(c) {
		return false
	}
	if service.OpenAICompactKeepaliveAdjustedWrittenSizeContext(c) == writerSizeBeforeForward {
		return true
	}
	return failoverErr != nil && failoverErr.SafeToFailoverAfterWrite
}

func openAIRequestAllowsFailoverReplay(c *gin.Context) bool {
	return openAIRequestAllowsFailoverReplayContext(gatewayctx.FromGin(c))
}

func openAIRequestAllowsFailoverReplayContext(c gatewayctx.GatewayContext) bool {
	return c != nil && c.Request() != nil && c.Context() != nil && c.Context().Err() == nil
}

func openAIFirstOutputFailoverExhausted(failoverErr *service.UpstreamFailoverError, switchCount *int) bool {
	if failoverErr == nil || !failoverErr.SafeToFailoverAfterWrite || switchCount == nil {
		return false
	}
	if *switchCount >= maxOpenAIFirstOutputTimeoutSwitches {
		return true
	}
	*switchCount = *switchCount + 1
	return false
}

func openAIResponsesRequiredCapability(imageIntent bool, platform string) service.OpenAIEndpointCapability {
	if imageIntent && platform == service.PlatformOpenAI {
		return service.OpenAIEndpointCapabilityResponses
	}
	return service.OpenAIEndpointCapabilityChatCompletions
}

func resolveOpenAIMessagesMetadataSession(sessionHash, promptCacheKey, reqModel string, body []byte) (string, string) {
	if sessionHash != "" {
		return sessionHash, promptCacheKey
	}
	if userID := strings.TrimSpace(gjson.GetBytes(body, "metadata.user_id").String()); userID != "" {
		sessionHash = service.DeriveSessionHashFromSeed(reqModel + "-" + userID)
	}
	return sessionHash, promptCacheKey
}
