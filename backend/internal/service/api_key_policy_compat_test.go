package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAPIKeyAllowedModels_NormalizesAndMatchesExactly(t *testing.T) {
	models := NormalizeAPIKeyAllowedModels([]string{" gpt-5.4 ", "gpt-5.4", "", "gpt-5.4-mini"})
	require.Equal(t, []string{"gpt-5.4", "gpt-5.4-mini"}, models)

	key := &APIKey{AllowedModels: models}
	require.True(t, key.AllowsModel("gpt-5.4"))
	require.False(t, key.AllowsModel("gpt-5.4-latest"))
	require.True(t, (&APIKey{}).AllowsModel("anything"), "empty allowlist remains unrestricted")
}

func TestAPIKeyAuthSnapshotRoundTripsPolicyFields(t *testing.T) {
	group10 := &Group{ID: 10, Name: "one", Status: StatusActive, Platform: PlatformOpenAI, Hydrated: true}
	group20 := &Group{ID: 20, Name: "two", Status: StatusActive, Platform: PlatformOpenAI, Hydrated: true}
	primary := group10.ID
	key := &APIKey{
		ID:            1,
		UserID:        2,
		Key:           "sk-policy-cache",
		Status:        StatusActive,
		GroupID:       &primary,
		GroupIDs:      []int64{10, 20},
		Group:         group10,
		Groups:        []*Group{group10, group20},
		AllowedModels: []string{"gpt-5.6", "claude-sonnet-4"},
		User:          &User{ID: 2, Status: StatusActive},
	}
	svc := &APIKeyService{}

	snapshot := svc.snapshotFromAPIKey(context.Background(), key)
	restored := svc.snapshotToAPIKey(key.Key, snapshot)

	require.Equal(t, []int64{10, 20}, restored.GroupIDs)
	require.Equal(t, []string{"gpt-5.6", "claude-sonnet-4"}, restored.AllowedModels)
	require.Len(t, restored.Groups, 2)
	require.Equal(t, int64(10), restored.Group.ID)
}

func TestAPIKeyGroupIDs_NormalizeAndPreserveSelectedUsageGroup(t *testing.T) {
	primary := int64(20)
	require.Equal(t, []int64{20, 10, 30}, NormalizeAPIKeyGroupIDs(&primary, []int64{10, 20, 10, 0, -1, 30}))
	require.Equal(t, []int64{20}, NormalizeAPIKeyGroupIDs(&primary, nil))

	key := &APIKey{GroupIDs: []int64{10, 20}}
	require.Nil(t, key.GroupIDForUsage(), "an unresolved multi-group key must fail closed for attribution")
	key.GroupID = &primary
	require.Equal(t, &primary, key.GroupIDForUsage(), "the routing-selected group must win")
}
