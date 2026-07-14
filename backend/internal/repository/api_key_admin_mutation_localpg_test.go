//go:build localpg

package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func TestAdminAPIKeyMutationPersistsAuditInLocalPostgres(t *testing.T) {
	dsn := os.Getenv("SUB2API_LOCALPG_TEST_DSN")
	if dsn == "" {
		t.Skip("SUB2API_LOCALPG_TEST_DSN is not set")
	}

	ctx := context.Background()
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, db.PingContext(ctx))
	require.NoError(t, ApplyMigrations(ctx, db))

	var actorID, customerID int64
	require.NoError(t, db.QueryRowContext(ctx, `
INSERT INTO users (email, password_hash, role, status)
VALUES ('p1d2-admin@example.invalid', 'test-only', 'admin', 'active')
RETURNING id`).Scan(&actorID))
	require.NoError(t, db.QueryRowContext(ctx, `
INSERT INTO users (email, password_hash, role, status)
VALUES ('p1d2-customer@example.invalid', 'test-only', 'user', 'active')
RETURNING id`).Scan(&customerID))

	const rawKey = "sk-p1d2-audit-secret-must-not-persist"
	var keyID int64
	require.NoError(t, db.QueryRowContext(ctx, `
INSERT INTO api_keys (user_id, key, name, status)
VALUES ($1, $2, 'P1-D2 local audit proof', 'active')
RETURNING id`, customerID, rawKey).Scan(&keyID))

	repo := newAPIKeyRepositoryWithSQL(nil, db)
	record, err := repo.AdminMutateAPIKey(ctx, service.AdminAPIKeyMutationCommand{
		APIKeyID: keyID, ActorUserID: actorID, Action: service.AdminAPIKeyAuditActionDisable,
	})
	require.NoError(t, err)
	require.Equal(t, service.StatusAPIKeyDisabled, record.Status)

	var status string
	require.NoError(t, db.QueryRowContext(ctx, "SELECT status FROM api_keys WHERE id = $1", keyID).Scan(&status))
	require.Equal(t, service.StatusAPIKeyDisabled, status)

	var (
		auditedActorID, auditedTargetUserID int64
		action, resourceType                string
		beforeJSON, afterJSON               []byte
	)
	require.NoError(t, db.QueryRowContext(ctx, `
SELECT actor_user_id, target_user_id, action, resource_type, before_values, after_values
FROM admin_audit_logs
WHERE resource_id = $1
ORDER BY id DESC
LIMIT 1`, keyID).Scan(
		&auditedActorID, &auditedTargetUserID, &action, &resourceType, &beforeJSON, &afterJSON,
	))
	require.Equal(t, actorID, auditedActorID)
	require.Equal(t, customerID, auditedTargetUserID)
	require.Equal(t, service.AdminAPIKeyAuditActionDisable, action)
	require.Equal(t, "api_key", resourceType)

	var before, after map[string]any
	require.NoError(t, json.Unmarshal(beforeJSON, &before))
	require.NoError(t, json.Unmarshal(afterJSON, &after))
	require.Equal(t, service.StatusAPIKeyActive, before["status"])
	require.Equal(t, service.StatusAPIKeyDisabled, after["status"])
	require.NotContains(t, strings.Join([]string{string(beforeJSON), string(afterJSON)}, " "), rawKey)
}
