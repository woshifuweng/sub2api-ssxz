//go:build localpg && !integration

package repository

import (
	"context"
	"database/sql"
	"io/fs"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/Wei-Shaw/sub2api/migrations"
	"github.com/stretchr/testify/require"
)

const preserveListingMigration = "234_ssxz_preserve_legacy_model_listing.sql"
const archiveQuotaMigration = "237_ssxz_archive_unlimited_quotas.sql"

// A real fresh PostgreSQL cluster, not a production clone or Docker-skipped test.
// Exercise the actual runner and immutable upstream SQL, including checksum replay.
func TestV027Upgrade_LocalPostgres(t *testing.T) {
	db := isolatedLocalPostgres(t)
	ctx := context.Background()
	execSQL := func(query string) {
		t.Helper()
		_, err := db.ExecContext(ctx, query)
		require.NoError(t, err)
	}
	queryText := func(query string) string {
		t.Helper()
		var result string
		require.NoError(t, db.QueryRowContext(ctx, query).Scan(&result))
		return result
	}
	fixture := `CREATE TABLE groups (id BIGINT PRIMARY KEY, models_list_config JSONB NOT NULL DEFAULT '{}');
	CREATE TABLE user_platform_quotas (id BIGINT PRIMARY KEY, user_id BIGINT, platform TEXT,
	daily_limit_usd NUMERIC, weekly_limit_usd NUMERIC, monthly_limit_usd NUMERIC,
	daily_usage_usd NUMERIC, weekly_usage_usd NUMERIC, monthly_usage_usd NUMERIC,
	daily_window_start TIMESTAMPTZ, deleted_at TIMESTAMPTZ);
	INSERT INTO groups VALUES (1,'{"enabled":true,"models":["gpt-5"]}'),(2,'{}');
	INSERT INTO user_platform_quotas VALUES
	(1,1,'openai',NULL,NULL,NULL,1.25,2.50,3.75,'2026-09-21',NULL),
	(2,2,'openai',0,NULL,NULL,0,0,0,NULL,NULL),
	(3,3,'openai',NULL,10,NULL,0,0,0,NULL,NULL),
	(4,4,'openai',NULL,NULL,NULL,0,0,0,NULL,NOW());`
	fsys := fstest.MapFS{}
	for _, name := range []string{preserveListingMigration, "235_group_model_allowlist.sql", "236_group_model_allowlist_repair.sql", archiveQuotaMigration, "238_purge_unlimited_user_platform_quotas.sql"} {
		data, err := fs.ReadFile(migrations.FS, name)
		require.NoError(t, err)
		fsys[name] = &fstest.MapFile{Data: data}
	}
	reset := func() {
		t.Helper()
		execSQL(`DROP SCHEMA public CASCADE; CREATE SCHEMA public;` + fixture)
	}
	t.Run("legacy_migrate_restart_old_queries_and_reupgrade", func(t *testing.T) {
		reset()
		require.NoError(t, applyMigrationsFS(ctx, db, fsys))
		require.JSONEq(t, `{"enabled":true,"models":["gpt-5"]}`, queryText(`SELECT models_list_config FROM groups WHERE id=1`))
		require.JSONEq(t, `{"enabled":false,"models":[]}`, queryText(`SELECT model_allowlist FROM groups WHERE id=1`))
		require.Equal(t, "2", queryText(`SELECT COUNT(*) FROM ssxz_v027_legacy_model_lists`))
		require.Equal(t, "2", queryText(`SELECT COUNT(*) FROM ssxz_v027_unlimited_quota_archive`))
		require.Equal(t, "1.25", queryText(`SELECT row_data->>'daily_usage_usd' FROM ssxz_v027_unlimited_quota_archive WHERE quota_id=1`))
		require.Equal(t, "2,3", queryText(`SELECT string_agg(id::text,',' ORDER BY id) FROM user_platform_quotas`))
		require.NoError(t, applyMigrationsFS(ctx, db, fsys), "repeat startup")
		// Old binary column access still works; its writes must not overwrite a
		// native restriction on re-upgrade, and checksums must remain intact.
		execSQL(`UPDATE groups SET models_list_config='{"enabled":true,"models":["legacy-edit"]}' WHERE id=1`)
		execSQL(`UPDATE groups SET model_allowlist='{"enabled":true,"models":["gpt-6"]}' WHERE id=1`)
		require.NoError(t, applyMigrationsFS(ctx, db, fsys))
		require.JSONEq(t, `{"enabled":true,"models":["gpt-6"]}`, queryText(`SELECT model_allowlist FROM groups WHERE id=1`))
		require.JSONEq(t, `{"enabled":true,"models":["gpt-5"]}`, queryText(`SELECT models_list_config FROM ssxz_v027_legacy_model_lists WHERE group_id=1`))
		require.Equal(t, "5", queryText(`SELECT COUNT(*) FROM schema_migrations`))
	})
	t.Run("already_native_policy_never_disabled", func(t *testing.T) {
		reset()
		execSQL(`ALTER TABLE groups RENAME COLUMN models_list_config TO model_allowlist`)
		require.NoError(t, applyMigrationsFS(ctx, db, fsys))
		require.JSONEq(t, `{"enabled":true,"models":["gpt-5"]}`, queryText(`SELECT model_allowlist FROM groups WHERE id=1`))
		require.Equal(t, "0", queryText(`SELECT COUNT(*) FROM ssxz_v027_legacy_model_lists`))
	})
	t.Run("ambiguous_both_columns_fail_closed", func(t *testing.T) {
		reset()
		execSQL(`ALTER TABLE groups ADD COLUMN model_allowlist JSONB NOT NULL DEFAULT '{}'`)
		err := applyMigrationsFS(ctx, db, fsys)
		require.ErrorContains(t, err, "ambiguous")
		require.Equal(t, "{}", queryText(`SELECT model_allowlist FROM groups WHERE id=1`))
	})
	t.Run("recorded_native_migration_but_old_schema_fail_closed", func(t *testing.T) {
		reset()
		execSQL(schemaMigrationsTableDDL + `INSERT INTO schema_migrations(filename,checksum) VALUES ('235_group_model_allowlist.sql','fixture');`)
		err := applyMigrationsFS(ctx, db, fsys)
		require.ErrorContains(t, err, "previously applied")
	})
	t.Run("invalid_checksum_is_not_bypassed", func(t *testing.T) {
		reset()
		require.NoError(t, applyMigrationsFS(ctx, db, fsys))
		execSQL(`UPDATE schema_migrations SET checksum='tampered' WHERE filename='235_group_model_allowlist.sql'`)
		require.ErrorContains(t, applyMigrationsFS(ctx, db, fsys), "checksum mismatch")
	})
}

// Full migration-chain smoke test on a fresh database. This catches SQL which
// passes string assertions but cannot run against preceding native migrations.
func TestV027FreshSchema_LocalPostgres(t *testing.T) {
	db := isolatedLocalPostgres(t)
	require.NoError(t, ApplyMigrations(context.Background(), db))
	require.NoError(t, ApplyMigrations(context.Background(), db))
	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&count))
	files, err := fs.Glob(migrations.FS, "*.sql")
	require.NoError(t, err)
	nonempty := 0
	for _, file := range files {
		content, err := fs.ReadFile(migrations.FS, file)
		require.NoError(t, err)
		if strings.TrimSpace(string(content)) != "" {
			nonempty++
		}
	}
	require.Equal(t, nonempty, count)
	var old, native sql.NullString
	require.NoError(t, db.QueryRow(`SELECT models_list_config,model_allowlist FROM groups LIMIT 1`).Scan(&old, &native))
	require.True(t, old.Valid && native.Valid)

	guard, err := os.ReadFile("../../../ops/production/v027-rollback-check.sql")
	require.NoError(t, err)
	// Run the exact read-only operational script on one pinned connection so a
	// rejected DO block can be rolled back without leaking an aborted transaction.
	conn, err := db.Conn(context.Background())
	require.NoError(t, err)
	defer conn.Close()
	checkGuard := func(want string) {
		t.Helper()
		_, err := conn.ExecContext(context.Background(), string(guard))
		if want == "" {
			require.NoError(t, err)
		} else {
			require.ErrorContains(t, err, want)
			_, rollbackErr := conn.ExecContext(context.Background(), "ROLLBACK")
			require.NoError(t, rollbackErr)
		}
	}
	checkGuard("")
	for _, tc := range []struct{ name, setup, cleanup, message string }{
		{"native_model_restriction", `UPDATE groups SET model_allowlist='{"enabled":true,"models":["gpt-6"]}'`, `UPDATE groups SET model_allowlist='{"enabled":false,"models":[]}'`, "request policy"},
		{"force_fast", `UPDATE groups SET force_openai_fast=true`, `UPDATE groups SET force_openai_fast=false`, "request policy"},
		{"free_fast", `UPDATE groups SET free_openai_fast=true`, `UPDATE groups SET free_openai_fast=false`, "request policy"},
		{"effort_rejection", `UPDATE groups SET max_reasoning_effort_over_limit='reject'`, `UPDATE groups SET max_reasoning_effort_over_limit='downgrade'`, "request policy"},
		{"pinned_manifest", `UPDATE groups SET codex_models_manifest_config='{"enabled":true}'`, `UPDATE groups SET codex_models_manifest_config='{}'`, "request policy"},
		{"public_group_restriction", `INSERT INTO users(email,password_hash,restrict_public_groups) VALUES ('fixture@example.invalid','not-a-login-hash',true)`, `DELETE FROM users WHERE email='fixture@example.invalid'`, "request policy"},
		{"native_billing", `INSERT INTO channels(id,name) VALUES (9876,'fixture'); INSERT INTO channel_model_pricing(channel_id,max_reasoning_effort_multiplier) VALUES(9876,2)`, `DELETE FROM channels WHERE id=9876`, "billing rules"},
		{"native_provider", `INSERT INTO accounts(name,platform,type,status,schedulable) VALUES ('fixture','minimax','apikey','disabled',false)`, `DELETE FROM accounts WHERE name='fixture'`, "provider data"},
		{"missing_legacy_column", `ALTER TABLE groups RENAME COLUMN models_list_config TO hidden_legacy`, `ALTER TABLE groups RENAME COLUMN hidden_legacy TO models_list_config`, "both legacy and native"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := db.Exec(tc.setup)
			require.NoError(t, err)
			checkGuard(tc.message)
			_, err = db.Exec(tc.cleanup)
			require.NoError(t, err)
			checkGuard("")
		})
	}
}
