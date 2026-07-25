package repository

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigrationChecksumCompatibilityRulesAreExplicitAndBidirectional(t *testing.T) {
	expectedNames := []string{
		"001_init.sql",
		"054_drop_legacy_cache_columns.sql",
		"061_add_usage_log_request_type.sql",
		"109_auth_identity_compat_backfill.sql",
		"110_pending_auth_and_provider_default_grants.sql",
		"112_add_payment_order_provider_key_snapshot.sql",
		"115_auth_identity_legacy_external_backfill.sql",
		"116_auth_identity_legacy_external_safety_reports.sql",
		"118_wechat_dual_mode_and_auth_source_defaults.sql",
		"119_enforce_payment_orders_out_trade_no_unique.sql",
		"120_enforce_payment_orders_out_trade_no_unique_notx.sql",
		"123_fix_legacy_auth_source_grant_on_signup_defaults.sql",
		"131_affiliate_rebate_hardening.sql",
		"132_affiliate_custom_settings.sql",
		"133_affiliate_rebate_freeze.sql",
		"159_batch_image_foundation.sql",
		"161_batch_image_pricing_snapshot.sql",
	}
	actualNames := make([]string, 0, len(migrationChecksumCompatibilityRules))
	for name := range migrationChecksumCompatibilityRules {
		actualNames = append(actualNames, name)
	}
	sort.Strings(actualNames)
	require.Equal(t, expectedNames, actualNames, "checksum compatibility must remain an explicit allowlist")

	for _, name := range expectedNames {
		rule := migrationChecksumCompatibilityRules[name]
		require.NotEmpty(t, rule.fileChecksum, name)
		require.Contains(t, rule.acceptedChecksums, rule.fileChecksum, name)
		require.GreaterOrEqual(t, len(rule.acceptedChecksums), 2, name)

		for dbChecksum := range rule.acceptedChecksums {
			for fileChecksum := range rule.acceptedChecksums {
				require.Truef(t, isMigrationChecksumCompatible(name, dbChecksum, fileChecksum),
					"known checksum pair rejected for %s: db=%s file=%s", name, dbChecksum, fileChecksum)
			}
		}
	}
}

func TestMigrationChecksumCompatibilityRejectsUnknownInputs(t *testing.T) {
	const unknown = "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"

	require.False(t, isMigrationChecksumCompatible(
		"002_account_type_migration.sql",
		"182c193f3359946cf094090cd9e57d5c3fd9abaffbc1e8fc378646b8a6fa12b4",
		"82de761156e03876653e7a6a4eee883cd927847036f779b0b9f34c42a8af7a7d",
	))
	require.False(t, isMigrationChecksumCompatible(
		"054_drop_legacy_cache_columns.sql",
		"182c193f3359946cf094090cd9e57d5c3fd9abaffbc1e8fc378646b8a6fa12b4",
		unknown,
	))
	require.False(t, isMigrationChecksumCompatible(
		"054_drop_legacy_cache_columns.sql",
		unknown,
		"82de761156e03876653e7a6a4eee883cd927847036f779b0b9f34c42a8af7a7d",
	))
}
