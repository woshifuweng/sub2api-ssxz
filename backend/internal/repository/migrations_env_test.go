package repository

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestApplyMigrationsSkipsWhenConfigured(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	t.Setenv("SKIP_MIGRATIONS", "true")

	require.NoError(t, ApplyMigrations(context.Background(), db))
}

func TestApplyMigrationsKeepsNilDatabaseGuardWhenConfigured(t *testing.T) {
	t.Setenv("SKIP_MIGRATIONS", "true")

	require.ErrorContains(t, ApplyMigrations(context.Background(), nil), "nil sql db")
}

func TestSkipMigrationsByEnvRecognizesExplicitTruthyValues(t *testing.T) {
	for _, value := range []string{"1", "true", "TRUE", "yes", "on"} {
		t.Setenv("SKIP_MIGRATIONS", value)
		require.True(t, skipMigrationsByEnv(), value)
	}

	for _, value := range []string{"", "false", "0", "no", "off", "random"} {
		t.Setenv("SKIP_MIGRATIONS", value)
		require.False(t, skipMigrationsByEnv(), value)
	}
}
