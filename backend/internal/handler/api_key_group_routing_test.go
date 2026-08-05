package handler

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestSelectOpenAIAPIKeyGroupWithCompact_PropagatesRequireCompact(t *testing.T) {
	groupID := int64(42)
	apiKey := &service.APIKey{ID: 7, GroupID: &groupID}
	var gotRequireCompact bool

	selection, err := selectOpenAIAPIKeyGroupWithCompact(
		context.Background(),
		apiKey,
		"resp_123",
		"session_123",
		"gpt-5.4",
		nil,
		service.OpenAIUpstreamTransportAny,
		true,
		func(ctx context.Context, groupID *int64, previousResponseID, sessionHash, requestedModel string, excludedIDs map[int64]struct{}, requiredTransport service.OpenAIUpstreamTransport, requireCompact bool) (*service.AccountSelectionResult, service.OpenAIAccountScheduleDecision, error) {
			gotRequireCompact = requireCompact
			return &service.AccountSelectionResult{
				Account: &service.Account{ID: 99, Platform: service.PlatformOpenAI},
			}, service.OpenAIAccountScheduleDecision{}, nil
		},
	)

	require.NoError(t, err)
	require.NotNil(t, selection)
	require.True(t, gotRequireCompact)
}

type usageChannelMappingResolverStub struct {
	byGroup map[int64]service.ChannelMappingResult
}

func (s usageChannelMappingResolverStub) ResolveChannelMappingAndRestrict(_ context.Context, groupID *int64, model string) (service.ChannelMappingResult, bool) {
	if groupID != nil {
		if mapping, ok := s.byGroup[*groupID]; ok {
			return mapping, false
		}
	}
	return service.ChannelMappingResult{MappedModel: model}, false
}

func TestResponsesUsageChannelMapping_UsesSelectedMultiGroup(t *testing.T) {
	defaultGroupID := int64(10)
	apiKey := &service.APIKey{
		GroupID:  &defaultGroupID,
		GroupIDs: []int64{10, 20},
		Groups:   []*service.Group{{ID: 10}, {ID: 20}},
	}
	selectedAPIKey := cloneAPIKeyWithGroup(apiKey, &service.Group{ID: 20})

	mapping := usageChannelMappingForAPIKey(
		context.Background(),
		usageChannelMappingResolverStub{byGroup: map[int64]service.ChannelMappingResult{
			20: {MappedModel: "gpt-5.5", ChannelID: 2002},
		}},
		selectedAPIKey,
		"gpt-5.5",
	)

	require.Equal(t, int64(2002), mapping.ChannelID)
}

func TestResponsesUsageChannelMapping_RefreshesAfterAccountSwitch(t *testing.T) {
	groupOne := int64(10)
	groupTwo := int64(20)
	initialAPIKey := &service.APIKey{GroupID: &groupOne}
	selectedAPIKey := &service.APIKey{GroupID: &groupTwo}
	resolver := usageChannelMappingResolverStub{byGroup: map[int64]service.ChannelMappingResult{
		10: {MappedModel: "gpt-5.5", ChannelID: 1001},
		20: {MappedModel: "gpt-5.5", ChannelID: 2002},
	}}

	initialMapping := usageChannelMappingForAPIKey(context.Background(), resolver, initialAPIKey, "gpt-5.5")
	refreshedMapping := usageChannelMappingForAPIKey(context.Background(), resolver, selectedAPIKey, "gpt-5.5")

	require.Equal(t, int64(1001), initialMapping.ChannelID)
	require.Equal(t, int64(2002), refreshedMapping.ChannelID)
}

func TestResponsesUsageChannelMapping_LeavesTrueNoChannelAsZero(t *testing.T) {
	groupID := int64(99)
	mapping := usageChannelMappingForAPIKey(
		context.Background(),
		usageChannelMappingResolverStub{byGroup: map[int64]service.ChannelMappingResult{}},
		&service.APIKey{GroupID: &groupID},
		"gpt-5.5",
	)

	require.Equal(t, int64(0), mapping.ChannelID)
}
