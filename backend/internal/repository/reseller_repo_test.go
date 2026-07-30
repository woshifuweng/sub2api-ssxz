package repository

import (
	"context"
	"database/sql/driver"
	"strings"
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

func TestResellerRepositoryCreateWithdrawRequestRejectsReservedBalance(t *testing.T) {
	repo, mock := newResellerRepoMock(t)

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT aff_quota::double precision.*FOR UPDATE`).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"aff_quota"}).AddRow(100.0))
	mock.ExpectQuery(`SELECT COALESCE\(SUM\(amount\), 0\)::double precision`).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"pending"}).AddRow(80.0))
	mock.ExpectRollback()

	_, err := repo.CreateWithdrawRequest(context.Background(), 7, service.WithdrawInput{
		Amount:      30,
		Method:      "alipay",
		AccountInfo: map[string]any{"account": "agent@example.com"},
	})

	require.ErrorIs(t, err, service.ErrWithdrawInsufficientBalance)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestResellerRepositoryCreateWithdrawRequestCommitsWithinAvailableBalance(t *testing.T) {
	repo, mock := newResellerRepoMock(t)
	now := time.Now().UTC()

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT aff_quota::double precision.*FOR UPDATE`).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"aff_quota"}).AddRow(100.0))
	mock.ExpectQuery(`SELECT COALESCE\(SUM\(amount\), 0\)::double precision`).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"pending"}).AddRow(80.0))
	mock.ExpectQuery(`INSERT INTO affiliate_withdraw_requests`).
		WithArgs(int64(7), 20.0, "alipay", jsonObjectMatcher{required: `"account":"agent@example.com"`}).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "amount", "method", "account_info", "status", "note", "requested_at",
		}).AddRow(int64(11), int64(7), 20.0, "alipay", []byte(`{"account":"agent@example.com"}`), "pending", "", now))
	mock.ExpectCommit()

	req, err := repo.CreateWithdrawRequest(context.Background(), 7, service.WithdrawInput{
		Amount:      20,
		Method:      "alipay",
		AccountInfo: map[string]any{"account": "agent@example.com"},
	})

	require.NoError(t, err)
	require.Equal(t, int64(11), req.ID)
	require.Equal(t, "agent@example.com", req.AccountInfo["account"])
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

func TestResellerRepositoryListAgentsIncludesManagersForAdmin(t *testing.T) {
	repo, mock := newResellerRepoMock(t)
	now := time.Now().UTC()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM user_reseller_roles`).
		WithArgs("", int64(0), true).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	mock.ExpectQuery(`SELECT rr.user_id.*rr.role, rr.commission_rate`).
		WithArgs("", int64(0), true, 20, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"user_id", "email", "username", "role", "commission_rate",
			"aff_code", "aff_count", "aff_quota", "granted_at", "granted_by",
		}).
			AddRow(int64(7), "agent@example.com", "Agent", "agent", 0.05, "A7", 3, 12.0, now, int64(1)).
			AddRow(int64(8), "manager@example.com", "Manager", "agent_manager", 0.10, "M8", 5, 20.0, now, int64(1)))

	items, total, err := repo.ListAgents(context.Background(), service.AgentFilter{IncludeAllRoles: true})

	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Len(t, items, 2)
	require.Equal(t, "agent_manager", items[1].Role)
	require.Equal(t, 0.10, items[1].CommissionRate)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestResellerRepositoryListWithdrawRequestsIncludesUsername(t *testing.T) {
	repo, mock := newResellerRepoMock(t)
	now := time.Now().UTC()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM affiliate_withdraw_requests`).
		WithArgs(int64(7), "", int64(0)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`SELECT wr.id, wr.user_id, COALESCE\(u.email, ''\), COALESCE\(u.username, ''\)`).
		WithArgs(int64(7), "", int64(0), 20, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "email", "username", "amount", "method", "account_info",
			"status", "note", "requested_at", "reviewed_at", "reviewed_by",
		}).AddRow(
			int64(11), int64(7), "agent@example.com", "Agent Seven", 20.0,
			"balance_transfer", []byte(`{}`), "pending", "", now, nil, nil,
		))

	items, total, err := repo.ListWithdrawRequests(context.Background(), service.WithdrawFilter{
		UserID: 7,
	})

	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	require.Equal(t, "Agent Seven", items[0].Username)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestResellerRepositoryReviewWithdrawTransfersOnlyRequestedAmount(t *testing.T) {
	repo, mock := newResellerRepoMock(t)

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT user_id, amount::double precision, status.*FOR UPDATE`).
		WithArgs(int64(12)).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "amount", "status"}).AddRow(int64(7), 25.0, "pending"))
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
		WithArgs("approved", "paid", int64(1), int64(12)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := repo.ReviewWithdrawRequest(context.Background(), 12, 1, "approved", "paid")

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestResellerRepositoryReviewWithdrawRejectsRepeatedReview(t *testing.T) {
	repo, mock := newResellerRepoMock(t)

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT user_id, amount::double precision, status.*FOR UPDATE`).
		WithArgs(int64(12)).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "amount", "status"}).AddRow(int64(7), 25.0, "approved"))
	mock.ExpectRollback()

	err := repo.ReviewWithdrawRequest(context.Background(), 12, 1, "approved", "paid")

	require.ErrorIs(t, err, service.ErrWithdrawAlreadyReviewed)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestResellerRepositoryCancelWithdrawRejectsForeignOwner(t *testing.T) {
	repo, mock := newResellerRepoMock(t)

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT user_id, status.*FOR UPDATE`).
		WithArgs(int64(12)).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "status"}).AddRow(int64(8), "pending"))
	mock.ExpectRollback()

	err := repo.CancelWithdrawRequest(context.Background(), 12, 7)

	require.ErrorIs(t, err, service.ErrWithdrawNotOwner)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestResellerRepositoryCancelWithdrawRejectsNonPendingState(t *testing.T) {
	repo, mock := newResellerRepoMock(t)

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT user_id, status.*FOR UPDATE`).
		WithArgs(int64(12)).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "status"}).AddRow(int64(7), "approved"))
	mock.ExpectRollback()

	err := repo.CancelWithdrawRequest(context.Background(), 12, 7)

	require.ErrorIs(t, err, service.ErrWithdrawNotPending)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestResellerRepositoryCancelWithdrawCommitsPendingRequest(t *testing.T) {
	repo, mock := newResellerRepoMock(t)

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT user_id, status.*FOR UPDATE`).
		WithArgs(int64(12)).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "status"}).AddRow(int64(7), "pending"))
	mock.ExpectExec(`UPDATE affiliate_withdraw_requests.*status = 'cancelled'`).
		WithArgs(int64(12), int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := repo.CancelWithdrawRequest(context.Background(), 12, 7)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestResellerRepositoryRevokeManagedAgentRejectsForeignAgent(t *testing.T) {
	repo, mock := newResellerRepoMock(t)

	mock.ExpectExec(`UPDATE user_reseller_roles`).
		WithArgs(int64(9), int64(3)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.RevokeManagedAgent(context.Background(), 9, 3)

	require.ErrorIs(t, err, service.ErrResellerAgentNotManaged)
	require.NoError(t, mock.ExpectationsWereMet())
}

type jsonObjectMatcher struct {
	required string
}

func (m jsonObjectMatcher) Match(value driver.Value) bool {
	var raw string
	switch typed := value.(type) {
	case string:
		raw = typed
	case []byte:
		raw = string(typed)
	default:
		return false
	}
	return strings.Contains(raw, m.required)
}
