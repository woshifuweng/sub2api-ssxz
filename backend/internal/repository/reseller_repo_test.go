package repository

import (
	"context"
	"database/sql/driver"
	"errors"
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

func TestResellerRepositoryCreateWithdrawRequestRejectsReservedBalance(t *testing.T) {
	repo, mock := newResellerRepoMock(t)

	mock.ExpectBegin()
	expectResellerMutationLock(mock)
	expectActiveResellerRoleLock(mock, 7, service.ResellerRoleAgent)
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
	expectResellerMutationLock(mock)
	expectActiveResellerRoleLock(mock, 7, service.ResellerRoleAgent)
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
		WithArgs("", int64(0), true, "current", "").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	mock.ExpectQuery(`SELECT rr.user_id.*rr.role`).
		WithArgs("", int64(0), true, "current", "", 20, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"user_id", "email", "username", "role", "status", "manager_id", "manager_email",
			"rebate_rate", "rebate_mode", "aff_code", "aff_count", "commission_balance",
			"commission_total", "notes", "granted_at", "updated_at", "disabled_at",
			"disabled_by_email", "disabled_reason", "revoked_at", "granted_by",
		}).
			AddRow(int64(7), "agent@example.com", "Agent", "agent", "active", nil, nil,
				5.0, "custom", "A7", 3, "12.00", "20.00", "agent", now, now, nil, nil, "", nil, int64(1)).
			AddRow(int64(8), "manager@example.com", "Manager", "agent_manager", "active", nil, nil,
				10.0, "custom", "M8", 5, "20.00", "30.00", "manager", now, now, nil, nil, "", nil, int64(1)))

	items, total, err := repo.ListAgents(context.Background(), service.AgentFilter{IncludeAllRoles: true})

	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Len(t, items, 2)
	require.Equal(t, "agent_manager", items[1].Role)
	require.Equal(t, 10.0, *items[1].EffectiveRebateRatePercent)
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

func TestResellerRepositoryListWithdrawRequestsScopesManagerByLifecycleManagerID(t *testing.T) {
	repo, mock := newResellerRepoMock(t)

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM affiliate_withdraw_requests.*rr\.manager_id = \$3`).
		WithArgs(int64(0), "pending", int64(33)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`SELECT wr\.id.*rr\.manager_id = \$3`).
		WithArgs(int64(0), "pending", int64(33), 20, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "email", "username", "amount", "method", "account_info",
			"status", "note", "requested_at", "reviewed_at", "reviewed_by",
		}))

	items, total, err := repo.ListWithdrawRequests(context.Background(), service.WithdrawFilter{
		ManagerID: 33,
		Status:    "pending",
	})

	require.NoError(t, err)
	require.Zero(t, total)
	require.Empty(t, items)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestResellerRepositoryReviewWithdrawTransfersOnlyRequestedAmount(t *testing.T) {
	repo, mock := newResellerRepoMock(t)

	mock.ExpectBegin()
	expectResellerMutationLock(mock)
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
	expectResellerMutationLock(mock)
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
	expectResellerMutationLock(mock)
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
	expectResellerMutationLock(mock)
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
	expectResellerMutationLock(mock)
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

	mock.ExpectBegin()
	expectResellerMutationLock(mock)
	mock.ExpectQuery(`SELECT role, manager_id, revoked_at.*FOR UPDATE`).
		WithArgs(int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"role", "manager_id", "revoked_at"}).
			AddRow("agent", int64(4), nil))
	mock.ExpectRollback()

	err := repo.RevokeManagedAgent(context.Background(), 9, 3)

	require.ErrorIs(t, err, service.ErrResellerAgentNotManaged)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestResellerRepositoryGrantRoleRejectsManagerDowngradeWithDirectAgents(t *testing.T) {
	repo, mock := newResellerRepoMock(t)

	mock.ExpectBegin()
	expectResellerMutationLock(mock)
	mock.ExpectQuery(`SELECT role, revoked_at.*FOR UPDATE`).
		WithArgs(int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"role", "revoked_at"}).
			AddRow(service.ResellerRoleManager, nil))
	mock.ExpectQuery(`SELECT EXISTS.*manager_id = \$1`).
		WithArgs(int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectRollback()

	err := repo.GrantRole(context.Background(), 9, service.ResellerRoleAgent, 1, "downgrade")

	require.ErrorIs(t, err, service.ErrResellerHasDirectAgents)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestResellerRepositoryUpdateAgentRejectsManagerDowngradeWithDirectAgents(t *testing.T) {
	repo, mock := newResellerRepoMock(t)
	nextRole := service.ResellerRoleAgent

	mock.ExpectBegin()
	expectResellerMutationLock(mock)
	expectActiveResellerRoleLock(mock, 9, service.ResellerRoleManager)
	mock.ExpectQuery(`SELECT EXISTS.*manager_id = \$1`).
		WithArgs(int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectRollback()

	_, err := repo.UpdateAgent(context.Background(), 9, 1, service.UpdateAgentInput{
		Role: &nextRole,
	})

	require.ErrorIs(t, err, service.ErrResellerHasDirectAgents)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestResellerRepositoryUpdateAgentRollsBackWhenRebateWriteFails(t *testing.T) {
	repo, mock := newResellerRepoMock(t)
	nextRole := service.ResellerRoleManager
	rate := 5.0

	mock.ExpectBegin()
	expectResellerMutationLock(mock)
	expectActiveResellerRoleLock(mock, 9, service.ResellerRoleAgent)
	mock.ExpectExec(`UPDATE user_reseller_roles`).
		WithArgs(int64(9), int64(1), true, nextRole, false, nil, false, "").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE user_affiliates`).
		WithArgs(int64(9), rate).
		WillReturnError(errors.New("write failed"))
	mock.ExpectRollback()

	_, err := repo.UpdateAgent(context.Background(), 9, 1, service.UpdateAgentInput{
		Role: &nextRole,
		RebatePolicy: &service.RebatePolicyInput{
			Mode:        service.RebateModeCustom,
			RatePercent: &rate,
		},
	})

	require.ErrorContains(t, err, "update rebate")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestResellerRepositoryDisableAgentRejectsAlreadyDisabledState(t *testing.T) {
	repo, mock := newResellerRepoMock(t)

	mock.ExpectBegin()
	expectResellerMutationLock(mock)
	mock.ExpectQuery(`SELECT role, status, revoked_at.*FOR UPDATE`).
		WithArgs(int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"role", "status", "revoked_at"}).
			AddRow(service.ResellerRoleAgent, service.ResellerStatusDisabled, nil))
	mock.ExpectRollback()

	_, err := repo.DisableAgent(context.Background(), 9, 1, "manual review")

	require.ErrorIs(t, err, service.ErrResellerStateConflict)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestResellerRepositoryRevokeRoleRejectsDirectAgents(t *testing.T) {
	repo, mock := newResellerRepoMock(t)

	mock.ExpectBegin()
	expectResellerMutationLock(mock)
	expectActiveResellerRoleLock(mock, 9, service.ResellerRoleManager)
	mock.ExpectQuery(`SELECT EXISTS.*manager_id = \$1`).
		WithArgs(int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectRollback()

	err := repo.RevokeRole(context.Background(), 9, 1)

	require.ErrorIs(t, err, service.ErrResellerHasDirectAgents)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestResellerRepositoryRevokeRoleRejectsPendingWithdrawals(t *testing.T) {
	repo, mock := newResellerRepoMock(t)

	mock.ExpectBegin()
	expectResellerMutationLock(mock)
	expectActiveResellerRoleLock(mock, 9, service.ResellerRoleAgent)
	mock.ExpectQuery(`SELECT EXISTS.*manager_id = \$1`).
		WithArgs(int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectQuery(`SELECT EXISTS.*affiliate_withdraw_requests`).
		WithArgs(int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectRollback()

	err := repo.RevokeRole(context.Background(), 9, 1)

	require.ErrorIs(t, err, service.ErrResellerHasPendingWithdraw)
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
