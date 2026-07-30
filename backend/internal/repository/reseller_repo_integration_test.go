//go:build integration

package repository

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func newResellerIntegrationUser(t *testing.T, quota float64) int64 {
	t.Helper()

	ctx := context.Background()
	user := mustCreateUser(t, integrationEntClient, &service.User{
		Email:        fmt.Sprintf("reseller-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
		Concurrency:  5,
	})
	affCode := fmt.Sprintf("RS%010d", time.Now().UnixNano()%10_000_000_000)
	_, err := integrationDB.ExecContext(ctx, `
		INSERT INTO user_affiliates (
			user_id, aff_code, aff_quota, aff_history_quota, created_at, updated_at
		)
		VALUES ($1, $2, $3, $3, NOW(), NOW())`,
		user.ID, affCode, quota,
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = integrationDB.ExecContext(context.Background(),
			"DELETE FROM user_affiliate_ledger WHERE user_id = $1 OR source_user_id = $1", user.ID)
		_, _ = integrationDB.ExecContext(context.Background(),
			"DELETE FROM affiliate_withdraw_requests WHERE user_id = $1", user.ID)
		_, _ = integrationDB.ExecContext(context.Background(),
			"DELETE FROM user_reseller_roles WHERE user_id = $1", user.ID)
		_, _ = integrationDB.ExecContext(context.Background(),
			"DELETE FROM user_affiliates WHERE user_id = $1", user.ID)
		_, _ = integrationDB.ExecContext(context.Background(),
			"DELETE FROM users WHERE id = $1", user.ID)
	})

	return user.ID
}

func resellerWithdrawInput(amount float64) service.WithdrawInput {
	return service.WithdrawInput{
		Amount:      amount,
		Method:      "manual",
		AccountInfo: map[string]any{"account": "internal-balance"},
	}
}

func TestResellerRepositoryConcurrentWithdrawalsReserveAvailableQuota(t *testing.T) {
	ctx := context.Background()
	userID := newResellerIntegrationUser(t, 100)
	repo := NewResellerRepository(integrationEntClient, integrationDB)

	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := repo.CreateWithdrawRequest(ctx, userID, resellerWithdrawInput(60))
			errs <- err
		}()
	}
	close(start)
	wg.Wait()
	close(errs)

	var succeeded, insufficient int
	for err := range errs {
		switch {
		case err == nil:
			succeeded++
		case errors.Is(err, service.ErrWithdrawInsufficientBalance):
			insufficient++
		default:
			t.Fatalf("unexpected concurrent withdrawal result: %v", err)
		}
	}
	require.Equal(t, 1, succeeded)
	require.Equal(t, 1, insufficient)

	var quota float64
	require.NoError(t, integrationDB.QueryRowContext(ctx,
		"SELECT aff_quota::double precision FROM user_affiliates WHERE user_id = $1",
		userID,
	).Scan(&quota))
	require.InDelta(t, 100, quota, 1e-9)
}

func TestResellerRepositoryApprovedWithdrawalExhaustsQuota(t *testing.T) {
	ctx := context.Background()
	userID := newResellerIntegrationUser(t, 100)
	repo := NewResellerRepository(integrationEntClient, integrationDB)

	request, err := repo.CreateWithdrawRequest(ctx, userID, resellerWithdrawInput(100))
	require.NoError(t, err)
	require.NoError(t, repo.ReviewWithdrawRequest(ctx, request.ID, userID, "approved", "approved"))

	var quota float64
	require.NoError(t, integrationDB.QueryRowContext(ctx,
		"SELECT aff_quota::double precision FROM user_affiliates WHERE user_id = $1",
		userID,
	).Scan(&quota))
	require.InDelta(t, 0, quota, 1e-9)

	_, err = repo.CreateWithdrawRequest(ctx, userID, resellerWithdrawInput(1))
	require.ErrorIs(t, err, service.ErrWithdrawInsufficientBalance)
}

func TestResellerRepositoryCannotCancelAnotherUsersWithdrawal(t *testing.T) {
	ctx := context.Background()
	ownerID := newResellerIntegrationUser(t, 20)
	otherUserID := newResellerIntegrationUser(t, 20)
	repo := NewResellerRepository(integrationEntClient, integrationDB)

	request, err := repo.CreateWithdrawRequest(ctx, ownerID, resellerWithdrawInput(10))
	require.NoError(t, err)

	err = repo.CancelWithdrawRequest(ctx, request.ID, otherUserID)
	require.ErrorIs(t, err, service.ErrWithdrawNotOwner)

	var status string
	require.NoError(t, integrationDB.QueryRowContext(ctx,
		"SELECT status FROM affiliate_withdraw_requests WHERE id = $1",
		request.ID,
	).Scan(&status))
	require.Equal(t, "pending", status)
}
