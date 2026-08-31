package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func newResellerRepoMock(t *testing.T) (*resellerRepository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return &resellerRepository{db: db}, mock
}

func expectResellerMutationLock(mock sqlmock.Sqlmock) {
	mock.ExpectExec(`SELECT pg_advisory_xact_lock\(\$1\)`).
		WithArgs(resellerMutationAdvisoryLock).
		WillReturnResult(sqlmock.NewResult(0, 1))
}

func expectActiveResellerRoleLock(mock sqlmock.Sqlmock, userID int64, role string) {
	mock.ExpectQuery(`SELECT role, status, revoked_at.*FOR UPDATE`).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"role", "status", "revoked_at"}).
			AddRow(role, service.ResellerStatusActive, nil))
}

func TestResellerRepositoryCreateWithdrawRequestReturnsExistingIdempotentRequest(t *testing.T) {
	repo, mock := newResellerRepoMock(t)
	requestedAt := time.Now()

	mock.ExpectBegin()
	expectResellerMutationLock(mock)
	expectActiveResellerRoleLock(mock, 7, service.ResellerRoleAgent)
	mock.ExpectQuery(`SELECT id, user_id, amount::double precision, method, account_info, status, note, requested_at.*idempotency_key = \$2.*FOR UPDATE`).
		WithArgs(int64(7), "withdraw-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "amount", "method", "account_info", "status", "note", "requested_at",
		}).AddRow(int64(12), int64(7), 5.0, "balance_transfer", []byte(`{}`), service.WithdrawStatusPending, "", requestedAt))
	mock.ExpectCommit()

	req, err := repo.CreateWithdrawRequest(context.Background(), 7, service.WithdrawInput{
		Amount:         5,
		Method:         "balance_transfer",
		AccountInfo:    map[string]any{},
		IdempotencyKey: "withdraw-1",
	})

	require.NoError(t, err)
	require.Equal(t, int64(12), req.ID)
	require.Equal(t, requestedAt, req.RequestedAt)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestResellerRepositoryCreateWithdrawRequestRejectsReusedKeyWithDifferentAmount(t *testing.T) {
	repo, mock := newResellerRepoMock(t)

	mock.ExpectBegin()
	expectResellerMutationLock(mock)
	expectActiveResellerRoleLock(mock, 7, service.ResellerRoleAgent)
	mock.ExpectQuery(`SELECT id, user_id, amount::double precision, method, account_info, status, note, requested_at.*idempotency_key = \$2.*FOR UPDATE`).
		WithArgs(int64(7), "withdraw-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "amount", "method", "account_info", "status", "note", "requested_at",
		}).AddRow(int64(12), int64(7), 6.0, "balance_transfer", []byte(`{}`), service.WithdrawStatusPending, "", time.Now()))
	mock.ExpectRollback()

	_, err := repo.CreateWithdrawRequest(context.Background(), 7, service.WithdrawInput{
		Amount:         5,
		Method:         "balance_transfer",
		AccountInfo:    map[string]any{},
		IdempotencyKey: "withdraw-1",
	})

	require.ErrorIs(t, err, service.ErrWithdrawIdempotencyConflict)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestResellerRepositoryCreateWithdrawRequestUsesLocksAndReservedBalance(t *testing.T) {
	repo, mock := newResellerRepoMock(t)

	mock.ExpectBegin()
	expectResellerMutationLock(mock)
	expectActiveResellerRoleLock(mock, 7, service.ResellerRoleAgent)
	mock.ExpectQuery(`SELECT id, user_id, amount::double precision, method, account_info, status, note, requested_at.*idempotency_key = \$2.*FOR UPDATE`).
		WithArgs(int64(7), "withdraw-2").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(`SELECT aff_quota::double precision.*FOR UPDATE`).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"aff_quota"}).AddRow(100.0))
	mock.ExpectQuery(`SELECT COALESCE\(SUM\(amount\), 0\)::double precision`).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"pending"}).AddRow(80.0))
	mock.ExpectRollback()

	_, err := repo.CreateWithdrawRequest(context.Background(), 7, service.WithdrawInput{
		Amount:         30,
		Method:         "balance_transfer",
		IdempotencyKey: "withdraw-2",
	})

	require.ErrorIs(t, err, service.ErrWithdrawInsufficientBalance)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestResellerRepositoryReviewWithdrawLocksAndTransfersRequestedAmount(t *testing.T) {
	repo, mock := newResellerRepoMock(t)

	mock.ExpectBegin()
	expectResellerMutationLock(mock)
	mock.ExpectQuery(`SELECT user_id, amount::double precision, status.*FOR UPDATE`).
		WithArgs(int64(12)).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "amount", "status"}).AddRow(int64(7), 25.0, service.WithdrawStatusPending))
	mock.ExpectQuery(`UPDATE user_affiliates.*aff_quota = aff_quota - \$1`).
		WithArgs(25.0, int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"aff_quota", "aff_frozen_quota", "aff_history_quota"}).AddRow(75.0, 5.0, 150.0))
	mock.ExpectQuery(`UPDATE users.*balance = balance \+ \$1`).
		WithArgs(25.0, int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"balance"}).AddRow(40.0))
	mock.ExpectExec(`INSERT INTO user_affiliate_ledger`).
		WithArgs(int64(7), 25.0, 40.0, 75.0, 5.0, 150.0, "reseller withdrawal #12").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`UPDATE affiliate_withdraw_requests`).
		WithArgs(service.WithdrawStatusApproved, "paid", int64(1), int64(12)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := repo.ReviewWithdrawRequest(context.Background(), 12, 1, service.WithdrawStatusApproved, "paid")

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestResellerRepositoryCancelWithdrawRejectsForeignOwner(t *testing.T) {
	repo, mock := newResellerRepoMock(t)

	mock.ExpectBegin()
	expectResellerMutationLock(mock)
	mock.ExpectQuery(`SELECT user_id, status.*FOR UPDATE`).
		WithArgs(int64(12)).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "status"}).AddRow(int64(8), service.WithdrawStatusPending))
	mock.ExpectRollback()

	err := repo.CancelWithdrawRequest(context.Background(), 12, 7)

	require.ErrorIs(t, err, service.ErrWithdrawNotOwner)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestResellerRepositoryRevokeManagedAgentRejectsForeignAgent(t *testing.T) {
	repo, mock := newResellerRepoMock(t)

	mock.ExpectBegin()
	expectResellerMutationLock(mock)
	mock.ExpectQuery(`SELECT role, manager_id, revoked_at.*FOR UPDATE`).
		WithArgs(int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"role", "manager_id", "revoked_at"}).
			AddRow(service.ResellerRoleAgent, int64(4), nil))
	mock.ExpectRollback()

	err := repo.RevokeManagedAgent(context.Background(), 9, 3)

	require.ErrorIs(t, err, service.ErrResellerAgentNotManaged)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestResellerRepositoryGetAgentDashboardCalculatesCommissionEarned(t *testing.T) {
	repo, mock := newResellerRepoMock(t)

	mock.ExpectQuery(`SELECT COALESCE\(aff_code, ''\)`).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{
			"aff_code", "aff_quota", "aff_frozen_quota", "aff_history_quota", "aff_count", "rebate_rate",
		}).AddRow("AGENT7", 40.0, 5.0, 80.0, 3, 0.1))
	mock.ExpectQuery(`SUM\(amount\) FILTER \(WHERE status = 'pending'\).*SUM\(amount\) FILTER \(WHERE status = 'approved'\)`).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"pending", "approved"}).AddRow(10.0, 30.0))

	dashboard, err := repo.GetAgentDashboard(context.Background(), 7)

	require.NoError(t, err)
	require.Equal(t, 10.0, dashboard.PendingWithdraw)
	require.Equal(t, 70.0, dashboard.CommissionEarned)
	require.NoError(t, mock.ExpectationsWereMet())
}

var _ = time.Time{}
