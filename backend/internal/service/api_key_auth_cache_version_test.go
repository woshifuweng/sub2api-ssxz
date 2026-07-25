package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAPIKeyService_AuthSnapshotPreservesCurrentModelAndGroupScope(t *testing.T) {
	groupID := int64(9)
	svc := &APIKeyService{}

	apiKey, ok, err := svc.applyAuthCacheEntry("k-current-scope", &APIKeyAuthCacheEntry{
		Snapshot: &APIKeyAuthSnapshot{
			APIKeyID:      1,
			UserID:        2,
			GroupID:       &groupID,
			GroupIDs:      []int64{groupID, 10},
			AllowedModels: []string{"gpt-5.5", "claude-opus-4-8"},
			Status:        StatusActive,
			User: APIKeyAuthUserSnapshot{
				ID:          2,
				Status:      StatusActive,
				Role:        RoleUser,
				Balance:     10,
				Concurrency: 3,
			},
			Group: &APIKeyAuthGroupSnapshot{
				ID:               groupID,
				Name:             "openai",
				Platform:         PlatformOpenAI,
				Status:           StatusActive,
				SubscriptionType: SubscriptionTypeStandard,
				RateMultiplier:   1,
			},
		},
	})

	require.NoError(t, err)
	require.True(t, ok)
	require.NotNil(t, apiKey)
	require.Equal(t, []int64{groupID, 10}, apiKey.GroupIDs)
	require.Equal(t, []string{"gpt-5.5", "claude-opus-4-8"}, apiKey.AllowedModels)
}

func TestAPIKeyService_AuthCacheEntryWithoutSnapshotFallsBackToRepository(t *testing.T) {
	apiKey, ok, err := (&APIKeyService{}).applyAuthCacheEntry("k-missing-snapshot", &APIKeyAuthCacheEntry{})

	require.NoError(t, err)
	require.False(t, ok)
	require.Nil(t, apiKey)
}
