package repository

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAgentLifecycleMigrationIsReplaySafe(t *testing.T) {
	sqlBytes, err := os.ReadFile("../../migrations/202_agent_lifecycle.sql")
	require.NoError(t, err)

	sqlText := string(sqlBytes)
	for _, column := range []string{
		"status",
		"manager_id",
		"updated_at",
		"updated_by",
		"disabled_at",
		"disabled_by",
		"disabled_reason",
	} {
		require.Contains(t, sqlText, "ADD COLUMN IF NOT EXISTS "+column)
	}
	require.NotContains(t, sqlText, "paid_out")
	require.Equal(t, 2, strings.Count(sqlText, "CREATE INDEX IF NOT EXISTS"))
}

func TestAgentLifecycleMigrationBackfillsOnlyConfirmedManagers(t *testing.T) {
	sqlBytes, err := os.ReadFile("../../migrations/202_agent_lifecycle.sql")
	require.NoError(t, err)

	sqlText := string(sqlBytes)
	require.Contains(t, sqlText, "manager_row.role = 'agent_manager'")
	require.Contains(t, sqlText, "child.role = 'agent'")
	require.Contains(t, sqlText, "child.manager_id IS NULL")
	require.Contains(t, sqlText, "child.granted_by = manager_row.user_id")
	require.Contains(t, sqlText, "manager_row.revoked_at IS NULL")
}
