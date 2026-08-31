package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPaymentOrderIdempotencyMigrationIsRetrySafe(t *testing.T) {
	content, err := FS.ReadFile("233_payment_order_idempotency.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS idempotency_key VARCHAR(128)")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS idempotency_request_hash VARCHAR(64)")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS idempotency_response JSONB")
	require.Contains(t, sql, "CREATE UNIQUE INDEX IF NOT EXISTS idx_payment_orders_user_idempotency_key")
	require.Contains(t, sql, "ON payment_orders(user_id, idempotency_key)")
	require.Contains(t, sql, "WHERE idempotency_key IS NOT NULL AND idempotency_key <> ''")
}
