package service

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/gin-gonic/gin"
)

// GetRequestCredentialContext bridges GatewayContext handlers to the
// request-path credential failover contract, which still accepts Gin context
// for request-scoped deadline and response metadata handling.
func (s *OpenAIGatewayService) GetRequestCredentialContext(
	ctx context.Context,
	c gatewayctx.GatewayContext,
	account *Account,
) (string, string, error) {
	var ginContext *gin.Context
	if c != nil {
		ginContext, _ = c.Native().(*gin.Context)
	}
	return s.GetRequestCredential(ctx, ginContext, account)
}

// ResolveChannelMappingAndRestrict keeps channel mapping at the gateway-service
// boundary while restriction enforcement remains in account scheduling.
func (s *OpenAIGatewayService) ResolveChannelMappingAndRestrict(ctx context.Context, groupID *int64, model string) (ChannelMappingResult, bool) {
	if s == nil || s.channelService == nil {
		return ChannelMappingResult{MappedModel: model}, false
	}
	return s.channelService.ResolveChannelMappingAndRestrict(ctx, groupID, model)
}

func (s *OpenAIGatewayService) SelectAccountWithSchedulerWithCompact(
	ctx context.Context,
	groupID *int64,
	previousResponseID string,
	sessionHash string,
	requestedModel string,
	excludedIDs map[int64]struct{},
	requiredTransport OpenAIUpstreamTransport,
	requireCompact bool,
) (*AccountSelectionResult, OpenAIAccountScheduleDecision, error) {
	return s.SelectAccountWithScheduler(ctx, groupID, previousResponseID, sessionHash, requestedModel, excludedIDs, requiredTransport, requireCompact)
}

// SelectAccountWithSchedulerForPlatform preserves the legacy scheduler
// boundary while allowing compatibility handlers to select the account pool
// for the API key's actual platform.
func (s *OpenAIGatewayService) SelectAccountWithSchedulerForPlatform(
	ctx context.Context,
	groupID *int64,
	previousResponseID string,
	sessionHash string,
	requestedModel string,
	excludedIDs map[int64]struct{},
	requiredTransport OpenAIUpstreamTransport,
	platform string,
	requireCompact bool,
) (*AccountSelectionResult, OpenAIAccountScheduleDecision, error) {
	if platform == "" {
		platform = PlatformOpenAI
	}
	if platform == PlatformGrok && requiredTransport == OpenAIUpstreamTransportResponsesWebsocketV2 {
		// Grok ingress WebSockets are bridged to the Grok HTTP API rather than
		// dialed as OpenAI upstream WebSockets.
		requiredTransport = OpenAIUpstreamTransportHTTPSSE
	}
	return s.SelectAccountWithSchedulerForCapability(
		ctx,
		groupID,
		previousResponseID,
		sessionHash,
		requestedModel,
		excludedIDs,
		requiredTransport,
		"",
		requireCompact,
		false,
		true,
		platform,
	)
}

// ReportOpenAIAccountModelScheduleResult retains the model-scoped transient
// recovery added by the production scheduler without changing legacy callers.
func (s *OpenAIGatewayService) ReportOpenAIAccountModelScheduleResult(accountID int64, model string, success bool, firstTokenMs *int) {
	if success {
		s.clearOpenAIAccountModelTransientState(accountID, normalizeOpenAIAccountModelTransientModel(model))
	}
	s.ReportOpenAIAccountScheduleResult(accountID, model, success, firstTokenMs)
}
