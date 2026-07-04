package repository

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
)

func newAffiliateRepositoryMock(t *testing.T) (*affiliateRepository, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	drv := entsql.OpenDB(dialect.Postgres, db)
	client := dbent.NewClient(dbent.Driver(drv))
	t.Cleanup(func() { _ = client.Close() })

	return &affiliateRepository{client: client}, mock
}

func TestListUsersWithCustomSettingsIncludesAffiliateStats(t *testing.T) {
	repo, mock := newAffiliateRepositoryMock(t)

	mock.ExpectQuery(`(?s)SELECT COUNT\(\*\).*FROM user_affiliates ua.*JOIN users u ON u.id = ua.user_id.*`).
		WithArgs("%promo%", "promo").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectQuery(`(?s)SELECT ua\.user_id,.*user_affiliate_ledger.*payment_orders.*ORDER BY ua\.updated_at DESC.*`).
		WithArgs("%promo%", "promo", 20, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"user_id",
			"email",
			"username",
			"aff_code",
			"aff_code_custom",
			"aff_rebate_rate_percent",
			"aff_count",
			"aff_quota",
			"aff_frozen_quota",
			"aff_history_quota",
			"accrued_rebate_total",
			"transferred_rebate_total",
			"invitee_recharge_total",
		}).AddRow(
			int64(7),
			"promoter@example.com",
			"promoter",
			"SSXZ7",
			true,
			12.5,
			3,
			8.25,
			2.5,
			13.0,
			13.0,
			2.25,
			100.0,
		))

	entries, total, err := repo.ListUsersWithCustomSettings(context.Background(), service.AffiliateAdminFilter{
		Search:   "promo",
		Page:     1,
		PageSize: 20,
	})

	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, entries, 1)

	entry := entries[0]
	require.Equal(t, int64(7), entry.UserID)
	require.Equal(t, "SSXZ7", entry.AffCode)
	require.Equal(t, 3, entry.AffCount)
	require.Equal(t, 8.25, entry.AffQuota)
	require.Equal(t, 2.5, entry.AffFrozenQuota)
	require.Equal(t, 13.0, entry.AffHistoryQuota)
	require.Equal(t, 13.0, entry.AccruedRebateTotal)
	require.Equal(t, 2.25, entry.TransferredRebateTotal)
	require.Equal(t, 100.0, entry.InviteeRechargeTotal)
	require.NotNil(t, entry.AffRebateRatePercent)
	require.Equal(t, 12.5, *entry.AffRebateRatePercent)
	require.NoError(t, mock.ExpectationsWereMet())
}
