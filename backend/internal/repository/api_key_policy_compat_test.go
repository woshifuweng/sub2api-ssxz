package repository

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestAPIKeyRepositoryCreateUpdateReadPolicyFields(t *testing.T) {
	repo, client := newAPIKeyRepoSQLite(t)
	ctx := context.Background()
	user := mustCreateAPIKeyRepoUser(t, ctx, client, "api-key-policy@test.local")
	groupOne, err := client.Group.Create().SetName("policy-one").SetPlatform(service.PlatformOpenAI).Save(ctx)
	require.NoError(t, err)
	groupTwo, err := client.Group.Create().SetName("policy-two").SetPlatform(service.PlatformOpenAI).Save(ctx)
	require.NoError(t, err)

	primaryGroupID := groupOne.ID
	key := &service.APIKey{
		UserID:        user.ID,
		Key:           "sk-policy-compat",
		Name:          "policy compatibility",
		Status:        service.StatusActive,
		GroupID:       &primaryGroupID,
		GroupIDs:      []int64{groupOne.ID, groupTwo.ID, groupOne.ID},
		AllowedModels: []string{" gpt-5.6 ", "gpt-5.6", "claude-sonnet-4"},
	}
	require.NoError(t, repo.Create(ctx, key))

	created, err := repo.GetByID(ctx, key.ID)
	require.NoError(t, err)
	require.Equal(t, []int64{groupOne.ID, groupTwo.ID}, created.GroupIDs)
	require.Equal(t, []string{"gpt-5.6", "claude-sonnet-4"}, created.AllowedModels)
	require.Len(t, created.Groups, 2)
	require.Equal(t, groupOne.ID, created.Group.ID)

	created.AllowedModels = []string{" claude-opus-4 ", "claude-opus-4"}
	require.NoError(t, repo.Update(ctx, created, service.APIKeyUpdateFields{AllowedModels: true}))

	updated, err := repo.GetByKeyForAuth(ctx, key.Key)
	require.NoError(t, err)
	require.Equal(t, []string{"claude-opus-4"}, updated.AllowedModels)
	require.Equal(t, []int64{groupOne.ID, groupTwo.ID}, updated.GroupIDs)
	require.Len(t, updated.Groups, 2)
}
