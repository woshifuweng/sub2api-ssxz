package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUsageBillingShortfallMigrationAllowsAuditEvent(t *testing.T) {
	content, err := FS.ReadFile("234_usage_billing_shortfall_ledger.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "DROP CONSTRAINT IF EXISTS chk_abl_event_type")
	require.Contains(t, sql, "ADD CONSTRAINT chk_abl_event_type CHECK")
	require.Contains(t, sql, "'usage_shortfall'")
}
