//go:build integration

package repository

import (
	"context"
	"testing"

	dbmigrations "github.com/Wei-Shaw/sub2api/migrations"
	"github.com/stretchr/testify/require"
)

func TestMigration231BackfillsOnlyActiveExclusiveStandardAPIKeys(t *testing.T) {
	tx := testTx(t)
	ctx := context.Background()
	migrationSQL, err := dbmigrations.FS.ReadFile("231_ssxz_backfill_exclusive_api_key_grants.sql")
	require.NoError(t, err)

	var userID, exclusiveGroupID, publicGroupID, subscriptionGroupID, disabledKeyGroupID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO users (email, password_hash, role, status)
VALUES ('migration-231@example.com', 'test-password-hash', 'user', 'active')
RETURNING id
`).Scan(&userID))

	for _, fixture := range []struct {
		name             string
		exclusive        bool
		subscriptionType string
		target           *int64
	}{
		{name: "migration-231-exclusive", exclusive: true, subscriptionType: "standard", target: &exclusiveGroupID},
		{name: "migration-231-public", exclusive: false, subscriptionType: "standard", target: &publicGroupID},
		{name: "migration-231-subscription", exclusive: true, subscriptionType: "subscription", target: &subscriptionGroupID},
		{name: "migration-231-disabled-key", exclusive: true, subscriptionType: "standard", target: &disabledKeyGroupID},
	} {
		require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO groups (name, platform, status, is_exclusive, subscription_type)
VALUES ($1, 'openai', 'active', $2, $3)
RETURNING id
`, fixture.name, fixture.exclusive, fixture.subscriptionType).Scan(fixture.target))
	}

	for _, fixture := range []struct {
		key     string
		groupID int64
		status  string
	}{
		{key: "sk-migration-231-exclusive", groupID: exclusiveGroupID, status: "active"},
		{key: "sk-migration-231-public", groupID: publicGroupID, status: "active"},
		{key: "sk-migration-231-subscription", groupID: subscriptionGroupID, status: "active"},
		{key: "sk-migration-231-disabled", groupID: disabledKeyGroupID, status: "disabled"},
	} {
		_, err := tx.ExecContext(ctx, `
INSERT INTO api_keys (user_id, key, name, group_id, status)
VALUES ($1, $2, 'migration-231', $3, $4)
`, userID, fixture.key, fixture.groupID, fixture.status)
		require.NoError(t, err)
	}

	_, err = tx.ExecContext(ctx, string(migrationSQL))
	require.NoError(t, err)
	_, err = tx.ExecContext(ctx, string(migrationSQL))
	require.NoError(t, err, "migration must be idempotent")

	var granted []int64
	rows, err := tx.QueryContext(ctx, `
SELECT group_id FROM user_allowed_groups WHERE user_id = $1 ORDER BY group_id
`, userID)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var groupID int64
		require.NoError(t, rows.Scan(&groupID))
		granted = append(granted, groupID)
	}
	require.NoError(t, rows.Err())
	require.Equal(t, []int64{exclusiveGroupID}, granted)
}
