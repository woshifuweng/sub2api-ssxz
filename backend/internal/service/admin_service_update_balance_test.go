//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type balanceUserRepoStub struct {
	*userRepoStub
	updateErr error
	updated   []*User
}

func (s *balanceUserRepoStub) Update(ctx context.Context, user *User) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	if user == nil {
		return nil
	}
	clone := *user
	s.updated = append(s.updated, &clone)
	if s.userRepoStub != nil {
		s.userRepoStub.user = &clone
	}
	return nil
}

type balanceRedeemRepoStub struct {
	*redeemRepoStub
	created []*RedeemCode
}

func (s *balanceRedeemRepoStub) Create(ctx context.Context, code *RedeemCode) error {
	if code == nil {
		return nil
	}
	clone := *code
	s.created = append(s.created, &clone)
	return nil
}

type authCacheInvalidatorStub struct {
	userIDs  []int64
	groupIDs []int64
	keys     []string
}

func (s *authCacheInvalidatorStub) InvalidateAuthCacheByKey(ctx context.Context, key string) {
	s.keys = append(s.keys, key)
}

func (s *authCacheInvalidatorStub) InvalidateAuthCacheByUserID(ctx context.Context, userID int64) {
	s.userIDs = append(s.userIDs, userID)
}

func (s *authCacheInvalidatorStub) InvalidateAuthCacheByGroupID(ctx context.Context, groupID int64) {
	s.groupIDs = append(s.groupIDs, groupID)
}

func TestAdminService_UpdateUserBalance_InvalidatesAuthCache(t *testing.T) {
	baseRepo := &userRepoStub{user: &User{ID: 7, Balance: 10}}
	repo := &balanceUserRepoStub{userRepoStub: baseRepo}
	redeemRepo := &balanceRedeemRepoStub{redeemRepoStub: &redeemRepoStub{}}
	invalidator := &authCacheInvalidatorStub{}
	svc := &adminServiceImpl{
		userRepo:             repo,
		redeemCodeRepo:       redeemRepo,
		authCacheInvalidator: invalidator,
	}

	_, err := svc.UpdateUserBalance(context.Background(), 7, 5, "add", "")
	require.NoError(t, err)
	require.Equal(t, []int64{7}, invalidator.userIDs)
	require.Len(t, redeemRepo.created, 1)
}

func TestAdminService_UpdateUserBalance_RecordsAdminBalanceAdjustment(t *testing.T) {
	tests := []struct {
		name            string
		startBalance    float64
		operation       string
		amount          float64
		expectedBalance float64
		expectedDiff    float64
	}{
		{
			name:            "add",
			startBalance:    10,
			operation:       "add",
			amount:          5,
			expectedBalance: 15,
			expectedDiff:    5,
		},
		{
			name:            "subtract",
			startBalance:    10,
			operation:       "subtract",
			amount:          4,
			expectedBalance: 6,
			expectedDiff:    -4,
		},
		{
			name:            "set",
			startBalance:    10,
			operation:       "set",
			amount:          7,
			expectedBalance: 7,
			expectedDiff:    -3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseRepo := &userRepoStub{user: &User{ID: 7, Balance: tt.startBalance}}
			repo := &balanceUserRepoStub{userRepoStub: baseRepo}
			redeemRepo := &balanceRedeemRepoStub{redeemRepoStub: &redeemRepoStub{}}
			invalidator := &authCacheInvalidatorStub{}
			svc := &adminServiceImpl{
				userRepo:             repo,
				redeemCodeRepo:       redeemRepo,
				authCacheInvalidator: invalidator,
			}

			user, err := svc.UpdateUserBalance(context.Background(), 7, tt.amount, tt.operation, "operator=admin:99")

			require.NoError(t, err)
			require.Equal(t, tt.expectedBalance, user.Balance)
			require.Equal(t, []int64{7}, invalidator.userIDs)
			require.Len(t, repo.updated, 1)
			require.Equal(t, tt.expectedBalance, repo.updated[0].Balance)
			require.Len(t, redeemRepo.created, 1)
			adjustment := redeemRepo.created[0]
			require.NotEmpty(t, adjustment.Code)
			require.Equal(t, AdjustmentTypeAdminBalance, adjustment.Type)
			require.Equal(t, tt.expectedDiff, adjustment.Value)
			require.Equal(t, StatusUsed, adjustment.Status)
			require.NotNil(t, adjustment.UsedBy)
			require.Equal(t, int64(7), *adjustment.UsedBy)
			require.Equal(t, "operator=admin:99", adjustment.Notes)
			require.NotNil(t, adjustment.UsedAt)
		})
	}
}

func TestAdminService_UpdateUserBalance_RejectsNegativeResultWithoutPersistence(t *testing.T) {
	baseRepo := &userRepoStub{user: &User{ID: 7, Balance: 10}}
	repo := &balanceUserRepoStub{userRepoStub: baseRepo}
	redeemRepo := &balanceRedeemRepoStub{redeemRepoStub: &redeemRepoStub{}}
	invalidator := &authCacheInvalidatorStub{}
	svc := &adminServiceImpl{
		userRepo:             repo,
		redeemCodeRepo:       redeemRepo,
		authCacheInvalidator: invalidator,
	}

	_, err := svc.UpdateUserBalance(context.Background(), 7, 15, "subtract", "operator=admin:99")

	require.Error(t, err)
	require.ErrorContains(t, err, "balance cannot be negative")
	require.Empty(t, repo.updated)
	require.Empty(t, redeemRepo.created)
	require.Empty(t, invalidator.userIDs)
}

func TestAdminService_UpdateUserBalance_NoChangeNoInvalidate(t *testing.T) {
	baseRepo := &userRepoStub{user: &User{ID: 7, Balance: 10}}
	repo := &balanceUserRepoStub{userRepoStub: baseRepo}
	redeemRepo := &balanceRedeemRepoStub{redeemRepoStub: &redeemRepoStub{}}
	invalidator := &authCacheInvalidatorStub{}
	svc := &adminServiceImpl{
		userRepo:             repo,
		redeemCodeRepo:       redeemRepo,
		authCacheInvalidator: invalidator,
	}

	_, err := svc.UpdateUserBalance(context.Background(), 7, 10, "set", "")
	require.NoError(t, err)
	require.Empty(t, invalidator.userIDs)
	require.Empty(t, redeemRepo.created)
}
