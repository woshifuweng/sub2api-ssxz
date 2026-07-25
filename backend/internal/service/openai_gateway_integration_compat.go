package service

import "context"

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

// ReportOpenAIAccountModelScheduleResult retains the model-scoped transient
// recovery added by the production scheduler without changing legacy callers.
func (s *OpenAIGatewayService) ReportOpenAIAccountModelScheduleResult(accountID int64, model string, success bool, firstTokenMs *int) {
	if success {
		s.clearOpenAIAccountModelTransientState(accountID, normalizeOpenAIAccountModelTransientModel(model))
	}
	s.ReportOpenAIAccountScheduleResult(accountID, model, success, firstTokenMs)
}
