package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResellerWithdrawalIdempotencyMigrationIsRetrySafe(t *testing.T) {
	content, err := FS.ReadFile("232_reseller_withdrawal_idempotency.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS idempotency_key VARCHAR(128)")
	require.Contains(t, sql, "CREATE UNIQUE INDEX IF NOT EXISTS idx_affiliate_withdraw_requests_user_idempotency_key")
	require.Contains(t, sql, "ON affiliate_withdraw_requests(user_id, idempotency_key)")
	require.Contains(t, sql, "WHERE idempotency_key IS NOT NULL AND idempotency_key <> ''")
}
