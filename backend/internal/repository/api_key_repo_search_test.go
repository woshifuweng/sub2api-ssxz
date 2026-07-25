package repository

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/ent/apikey"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestAPIKeyUserListSearchPredicateDoesNotMatchRawKey(t *testing.T) {
	repo, client := newAPIKeyRepoSQLite(t)
	ctx := context.Background()
	user := mustCreateAPIKeyRepoUser(t, ctx, client, "list-search-name-only@test.com")

	rawKeyOnly := &service.APIKey{
		UserID: user.ID,
		Key:    "sk-secretneedle-raw-value",
		Name:   "Production",
		Status: service.StatusActive,
	}
	require.NoError(t, repo.Create(ctx, rawKeyOnly))

	nameMatch := &service.APIKey{
		UserID: user.ID,
		Key:    "sk-display-name-value",
		Name:   "secretneedle display",
		Status: service.StatusActive,
	}
	require.NoError(t, repo.Create(ctx, nameMatch))

	keys, err := repo.activeQuery().
		Where(apikey.UserIDEQ(user.ID), apiKeyUserListSearchPredicate("secretneedle")).
		All(ctx)

	require.NoError(t, err)
	require.Len(t, keys, 1)
	require.Equal(t, nameMatch.ID, keys[0].ID)
	require.NotEqual(t, rawKeyOnly.ID, keys[0].ID)
}
