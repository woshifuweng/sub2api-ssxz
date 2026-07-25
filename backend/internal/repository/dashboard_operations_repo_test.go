package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestDashboardOperationsRepository_GetOperationsSummary(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`(?s)WITH new_customers AS .*SUM\(ul\.actual_cost\).*payment_orders.*user_affiliate_ledger.*`).
		WithArgs(start, end).
		WillReturnRows(sqlmock.NewRows([]string{
			"new_customers",
			"actual_cost",
			"recharge_amount",
			"pending",
			"available",
			"transferred",
			"active_customers",
			"active_api_keys",
		}).AddRow(7, 12.34, 50.00, 1.25, 2.50, 0.75, 4, 6))

	mock.ExpectQuery(`(?s)SELECT ul\.user_id,.*SUM\(ul\.actual_cost\).*ORDER BY actual_cost DESC.*`).
		WithArgs(start, end, 10).
		WillReturnRows(sqlmock.NewRows([]string{
			"user_id", "email", "username", "actual_cost", "requests", "active_keys",
		}).
			AddRow(11, "one@example.com", "One", 8.25, 12, 2).
			AddRow(15, "two@example.com", "", 4.09, 7, 1))

	repo := NewDashboardOperationsRepository(db)
	summary, err := repo.GetOperationsSummary(context.Background(), start, end, 10)
	require.NoError(t, err)
	require.Equal(t, int64(7), summary.NewCustomers)
	require.InDelta(t, 12.34, summary.CustomerActualCost, 0.000001)
	require.InDelta(t, 50.00, summary.InviteeRechargeAmount, 0.000001)
	require.InDelta(t, 1.25, summary.RebatePending, 0.000001)
	require.InDelta(t, 2.50, summary.RebateAvailable, 0.000001)
	require.InDelta(t, 0.75, summary.RebateTransferred, 0.000001)
	require.Equal(t, int64(4), summary.ActiveCustomers)
	require.Equal(t, int64(6), summary.ActiveAPIKeys)
	require.Len(t, summary.TopCustomers, 2)
	require.Equal(t, int64(11), summary.TopCustomers[0].UserID)
	require.InDelta(t, 8.25, summary.TopCustomers[0].ActualCost, 0.000001)
	require.NoError(t, mock.ExpectationsWereMet())
}
