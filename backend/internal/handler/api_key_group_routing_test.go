package handler

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestSelectGatewayAPIKeyGroupRoutesAcrossBindingsAndPreservesUsageGroup(t *testing.T) {
	group10 := &service.Group{ID: 10, Status: service.StatusActive, Platform: service.PlatformOpenAI}
	group20 := &service.Group{ID: 20, Status: service.StatusActive, Platform: service.PlatformOpenAI}
	primary := group10.ID
	apiKey := &service.APIKey{GroupID: &primary, GroupIDs: []int64{10, 20}, Group: group10, Groups: []*service.Group{group10, group20}}

	var attempts []int64
	selected, err := selectGatewayAPIKeyGroup(apiKey, "stable-session", func(groupID *int64) (*service.AccountSelectionResult, error) {
		require.NotNil(t, groupID)
		attempts = append(attempts, *groupID)
		if len(attempts) == 1 {
			return nil, service.ErrNoAvailableAccounts
		}
		return &service.AccountSelectionResult{Account: &service.Account{ID: 99}, Acquired: true}, nil
	})

	require.NoError(t, err)
	require.Len(t, attempts, 2)
	require.Equal(t, attempts[1], *selected.APIKey.GroupID)
	require.Equal(t, attempts[1], *selected.APIKey.GroupIDForUsage())
	require.Equal(t, int64(99), selected.Selection.Account.ID)
}

func TestSelectGatewayAPIKeyGroupRetainsSingleGroupCompatibility(t *testing.T) {
	group := &service.Group{ID: 10, Status: service.StatusActive}
	groupID := group.ID
	apiKey := &service.APIKey{GroupID: &groupID, Group: group}
	callCount := 0

	selected, err := selectGatewayAPIKeyGroup(apiKey, "", func(got *int64) (*service.AccountSelectionResult, error) {
		callCount++
		require.Equal(t, groupID, *got)
		return &service.AccountSelectionResult{Account: &service.Account{ID: 7}, Acquired: true}, nil
	})

	require.NoError(t, err)
	require.Equal(t, 1, callCount)
	require.Same(t, apiKey, selected.APIKey)
}

func TestSelectGatewayAPIKeyGroupRetainsGroupIDOnlyCompatibility(t *testing.T) {
	groupID := int64(10)
	apiKey := &service.APIKey{GroupID: &groupID}
	callCount := 0

	selected, err := selectGatewayAPIKeyGroup(apiKey, "", func(got *int64) (*service.AccountSelectionResult, error) {
		callCount++
		require.NotNil(t, got)
		require.Equal(t, groupID, *got)
		return &service.AccountSelectionResult{Account: &service.Account{ID: 7}, Acquired: true}, nil
	})

	require.NoError(t, err)
	require.Equal(t, 1, callCount)
	require.Same(t, apiKey, selected.APIKey)
}

func TestSelectGatewayAPIKeyGroupFailsClosedWhenBindingIsIncomplete(t *testing.T) {
	group := &service.Group{ID: 10, Status: service.StatusActive}
	groupID := group.ID
	apiKey := &service.APIKey{GroupID: &groupID, GroupIDs: []int64{10, 20}, Group: group, Groups: []*service.Group{group}}

	_, err := selectGatewayAPIKeyGroup(apiKey, "", func(*int64) (*service.AccountSelectionResult, error) {
		t.Fatal("selector must not run with incomplete bindings")
		return nil, nil
	})
	require.ErrorIs(t, err, errAPIKeyGroupBindingIncomplete)
}

type recordingChannelMappingResolver struct {
	groupID *int64
}

func (r *recordingChannelMappingResolver) ResolveChannelMappingAndRestrict(_ context.Context, groupID *int64, model string) (service.ChannelMappingResult, bool) {
	r.groupID = groupID
	return service.ChannelMappingResult{MappedModel: model}, false
}

func TestUsageChannelMappingUsesSelectedRequestGroup(t *testing.T) {
	selectedGroupID := int64(20)
	apiKey := &service.APIKey{GroupID: &selectedGroupID, GroupIDs: []int64{10, 20}}
	resolver := &recordingChannelMappingResolver{}

	usageChannelMappingForAPIKey(context.Background(), resolver, apiKey, "gpt-5.6")
	require.NotNil(t, resolver.groupID)
	require.Equal(t, selectedGroupID, *resolver.groupID)
}
