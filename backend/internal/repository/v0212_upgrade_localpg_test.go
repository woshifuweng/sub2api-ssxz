//go:build localpg && !integration

package repository

import (
	"context"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/Wei-Shaw/sub2api/migrations"
	"github.com/stretchr/testify/require"
)

func TestV0212Upgrade_LocalPostgres(t *testing.T) {
	db := isolatedLocalPostgres(t)
	ctx := context.Background()
	_, err := db.ExecContext(ctx, schemaMigrationsTableDDL+`
		INSERT INTO schema_migrations(filename, checksum)
			VALUES ('234_usage_billing_shortfall_ledger.sql', 'retained-ssxz-checksum');
		CREATE TABLE payment_orders (
			id BIGINT PRIMARY KEY, amount NUMERIC(20,2), pay_amount NUMERIC(20,2),
			idempotency_key TEXT, idempotency_request_hash TEXT, idempotency_response JSONB);
		INSERT INTO payment_orders VALUES (1, 5, 9, 'fixture', 'request-hash', '{"status":"completed"}');
		CREATE TABLE user_platform_quotas (
			id BIGINT PRIMARY KEY, platform TEXT,
			CONSTRAINT user_platform_quotas_platform_check CHECK(platform IN ('openai')));
		INSERT INTO user_platform_quotas VALUES (1, 'openai');
		CREATE TABLE composite_model_routes (
			id BIGINT PRIMARY KEY, target_platform TEXT,
			CONSTRAINT composite_model_routes_target_platform_check CHECK(target_platform IN ('openai')));
		INSERT INTO composite_model_routes VALUES (1, 'openai');
	`)
	require.NoError(t, err)
	fsys := fstest.MapFS{}
	for _, name := range []string{"241_add_payment_order_bonus_amount.sql", "241_add_typesafe_platform.sql"} {
		data, err := fs.ReadFile(migrations.FS, name)
		require.NoError(t, err)
		fsys[name] = &fstest.MapFile{Data: data}
	}
	require.NoError(t, applyMigrationsFS(ctx, db, fsys))
	var amount, paid, bonus float64
	var key, requestHash, response string
	require.NoError(t, db.QueryRowContext(ctx, `
		SELECT amount,pay_amount,bonus_amount,idempotency_key,idempotency_request_hash,idempotency_response::text
		FROM payment_orders WHERE id=1`).Scan(&amount, &paid, &bonus, &key, &requestHash, &response))
	require.Equal(t, 5.0, amount)
	require.Equal(t, 9.0, paid)
	require.Zero(t, bonus)
	require.Equal(t, "fixture", key)
	require.Equal(t, "request-hash", requestHash)
	require.JSONEq(t, `{"status":"completed"}`, response)
	_, err = db.ExecContext(ctx, `
		UPDATE payment_orders SET bonus_amount=1 WHERE id=1;
		INSERT INTO user_platform_quotas VALUES (2,'typesafe');
		INSERT INTO composite_model_routes VALUES (2,'typesafe');`)
	require.NoError(t, err)
	require.NoError(t, applyMigrationsFS(ctx, db, fsys), "restart must not erase credited bonus or rewrite checksums")
	require.NoError(t, db.QueryRowContext(ctx, "SELECT bonus_amount FROM payment_orders WHERE id=1").Scan(&bonus))
	require.Equal(t, 1.0, bonus)
	var checksum string
	require.NoError(t, db.QueryRowContext(ctx, "SELECT checksum FROM schema_migrations WHERE filename='234_usage_billing_shortfall_ledger.sql'").Scan(&checksum))
	require.Equal(t, "retained-ssxz-checksum", checksum)
	var count int
	require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations").Scan(&count))
	require.Equal(t, 3, count, "same 241 prefix must still record two distinct filenames")
	_, err = db.ExecContext(ctx, "UPDATE schema_migrations SET checksum='tampered' WHERE filename='241_add_payment_order_bonus_amount.sql'")
	require.NoError(t, err)
	require.ErrorContains(t, applyMigrationsFS(ctx, db, fsys), "checksum mismatch")
}
