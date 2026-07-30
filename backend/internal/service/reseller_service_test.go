package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type resellerRepositoryStub struct {
	role             *ResellerRoleRecord
	createdInput     WithdrawInput
	agentFilter      AgentFilter
	withdrawFilter   WithdrawFilter
	managedGrant     [2]int64
	managedRevoke    [2]int64
	cancelled        [2]int64
	reviewStatus     string
	managerDashboard int64
	withdrawals      []WithdrawRequest
}

func (s *resellerRepositoryStub) GetRole(context.Context, int64) (*ResellerRoleRecord, error) {
	if s.role == nil {
		return nil, ErrResellerRoleNotFound
	}
	return s.role, nil
}
func (s *resellerRepositoryStub) GrantRole(context.Context, int64, string, int64, string) error {
	return nil
}
func (s *resellerRepositoryStub) GrantManagedAgent(_ context.Context, userID, managerID int64, _ string) error {
	s.managedGrant = [2]int64{userID, managerID}
	return nil
}
func (s *resellerRepositoryStub) RevokeRole(context.Context, int64) error { return nil }
func (s *resellerRepositoryStub) RevokeManagedAgent(_ context.Context, userID, managerID int64) error {
	s.managedRevoke = [2]int64{userID, managerID}
	return nil
}
func (s *resellerRepositoryStub) ListAgents(_ context.Context, filter AgentFilter) ([]AgentSummary, int64, error) {
	s.agentFilter = filter
	return nil, 0, nil
}
func (s *resellerRepositoryStub) GetAgentDetail(context.Context, int64, int64) (*AgentDetail, error) {
	return &AgentDetail{}, nil
}
func (s *resellerRepositoryStub) GetAgentDashboard(context.Context, int64) (*AgentDashboard, error) {
	return &AgentDashboard{}, nil
}
func (s *resellerRepositoryStub) ListMyRecruits(context.Context, int64, int, int, bool) ([]RecruitRecord, int64, error) {
	return nil, 0, nil
}
func (s *resellerRepositoryStub) CreateWithdrawRequest(_ context.Context, _ int64, input WithdrawInput) (*WithdrawRequest, error) {
	s.createdInput = input
	return &WithdrawRequest{Amount: input.Amount, Method: input.Method, AccountInfo: input.AccountInfo}, nil
}
func (s *resellerRepositoryStub) ListWithdrawRequests(_ context.Context, filter WithdrawFilter) ([]WithdrawRequest, int64, error) {
	s.withdrawFilter = filter
	return s.withdrawals, int64(len(s.withdrawals)), nil
}
func (s *resellerRepositoryStub) ReviewWithdrawRequest(_ context.Context, _, _ int64, status, _ string) error {
	s.reviewStatus = status
	return nil
}
func (s *resellerRepositoryStub) CancelWithdrawRequest(_ context.Context, withdrawalID, userID int64) error {
	s.cancelled = [2]int64{withdrawalID, userID}
	return nil
}
func (s *resellerRepositoryStub) GetManagerDashboard(_ context.Context, managerID int64) (*ManagerDashboard, error) {
	s.managerDashboard = managerID
	return &ManagerDashboard{}, nil
}

func TestResellerServiceRequestWithdrawNormalizesValidatedAccount(t *testing.T) {
	repo := &resellerRepositoryStub{role: &ResellerRoleRecord{Role: ResellerRoleAgent}}
	svc := NewResellerService(repo)

	req, err := svc.RequestWithdraw(context.Background(), 7, WithdrawInput{
		Amount:      12.5,
		Method:      " AliPay ",
		AccountInfo: map[string]any{"account": " agent@example.com "},
	})

	require.NoError(t, err)
	require.Equal(t, "alipay", req.Method)
	require.Equal(t, "agent@example.com", req.AccountInfo["account"])
}

func TestResellerServiceRequestWithdrawDefaultsToBalanceTransfer(t *testing.T) {
	repo := &resellerRepositoryStub{role: &ResellerRoleRecord{Role: ResellerRoleAgent}}
	svc := NewResellerService(repo)

	req, err := svc.RequestWithdraw(context.Background(), 7, WithdrawInput{Amount: 12.5})

	require.NoError(t, err)
	require.Equal(t, "balance_transfer", req.Method)
	require.Empty(t, req.AccountInfo)
	require.Equal(t, "balance_transfer", repo.createdInput.Method)
	require.Empty(t, repo.createdInput.AccountInfo)
}

func TestResellerServiceRequestWithdrawRejectsInvalidAccount(t *testing.T) {
	repo := &resellerRepositoryStub{role: &ResellerRoleRecord{Role: ResellerRoleAgent}}
	svc := NewResellerService(repo)

	_, err := svc.RequestWithdraw(context.Background(), 7, WithdrawInput{
		Amount:      12.5,
		Method:      "alipay",
		AccountInfo: map[string]any{},
	})

	require.ErrorIs(t, err, ErrWithdrawInvalidAccount)
}

func TestResellerServiceManagerQueriesAreScoped(t *testing.T) {
	repo := &resellerRepositoryStub{}
	svc := NewResellerService(repo)

	_, _, err := svc.ListManagedAgents(context.Background(), 33, AgentFilter{Search: "agent"})
	require.NoError(t, err)
	require.Equal(t, int64(33), repo.agentFilter.ManagerID)

	_, _, err = svc.ListManagedWithdrawRequests(context.Background(), 33, WithdrawFilter{Status: "pending"})
	require.NoError(t, err)
	require.Equal(t, int64(33), repo.withdrawFilter.ManagerID)

	_, err = svc.ManagerDashboard(context.Background(), 33)
	require.NoError(t, err)
	require.Equal(t, int64(33), repo.managerDashboard)
}

func TestResellerServiceManagedWithdrawalsHidePaymentAccount(t *testing.T) {
	repo := &resellerRepositoryStub{
		withdrawals: []WithdrawRequest{{
			ID:          12,
			UserID:      7,
			AccountInfo: map[string]any{"account": "private@example.com"},
			Status:      "pending",
		}},
	}
	svc := NewResellerService(repo)

	items, total, err := svc.ListManagedWithdrawRequests(context.Background(), 33, WithdrawFilter{
		Page:     1,
		PageSize: 20,
	})

	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	require.Nil(t, items[0].AccountInfo)
	require.Equal(t, int64(33), repo.withdrawFilter.ManagerID)
}

func TestResellerServiceManagerCannotManageSelf(t *testing.T) {
	repo := &resellerRepositoryStub{}
	svc := NewResellerService(repo)

	require.ErrorIs(t, svc.GrantManagedAgent(context.Background(), 33, 33, ""), ErrResellerCannotManageSelf)
	require.ErrorIs(t, svc.RevokeManagedAgent(context.Background(), 33, 33), ErrResellerCannotManageSelf)
	require.Equal(t, [2]int64{}, repo.managedGrant)
	require.Equal(t, [2]int64{}, repo.managedRevoke)
}

func TestResellerServiceReviewMapsActionToStableStatus(t *testing.T) {
	repo := &resellerRepositoryStub{}
	svc := NewResellerService(repo)

	require.NoError(t, svc.ReviewWithdrawRequest(context.Background(), 1, 2, "approve", ""))
	require.Equal(t, "approved", repo.reviewStatus)
	require.NoError(t, svc.ReviewWithdrawRequest(context.Background(), 1, 2, "reject", "declined"))
	require.Equal(t, "rejected", repo.reviewStatus)
}

func TestResellerServiceReviewRejectRequiresReason(t *testing.T) {
	repo := &resellerRepositoryStub{}
	svc := NewResellerService(repo)

	err := svc.ReviewWithdrawRequest(context.Background(), 1, 2, "reject", "  ")

	require.ErrorIs(t, err, ErrWithdrawReasonRequired)
	require.Empty(t, repo.reviewStatus)
}

func TestResellerServiceReviewRejectsUnknownAction(t *testing.T) {
	repo := &resellerRepositoryStub{}
	svc := NewResellerService(repo)

	err := svc.ReviewWithdrawRequest(context.Background(), 1, 2, "approve-and-pay", "invalid")

	require.ErrorIs(t, err, ErrWithdrawInvalidAction)
	require.Empty(t, repo.reviewStatus)
}

func TestResellerServiceCancelWithdrawalDelegatesOwnershipCheck(t *testing.T) {
	repo := &resellerRepositoryStub{}
	svc := NewResellerService(repo)

	require.NoError(t, svc.CancelWithdrawal(context.Background(), 7, 12))
	require.Equal(t, [2]int64{12, 7}, repo.cancelled)
}
