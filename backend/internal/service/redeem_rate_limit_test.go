//go:build unit

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// redeemRateLimitCacheStub 实现 RedeemCache，记录各方法调用次数并允许注入错误。
type redeemRateLimitCacheStub struct {
	count          int
	getErr         error
	incrementErr   error
	getCalls       int
	incrementCalls int
	acquireCalls   int
	releaseCalls   int
}

func (c *redeemRateLimitCacheStub) GetRedeemAttemptCount(context.Context, int64) (int, error) {
	c.getCalls++
	return c.count, c.getErr
}

func (c *redeemRateLimitCacheStub) IncrementRedeemAttemptCount(context.Context, int64) error {
	c.incrementCalls++
	if c.incrementErr != nil {
		return c.incrementErr
	}
	c.count++
	return nil
}

func (c *redeemRateLimitCacheStub) AcquireRedeemLock(context.Context, string, time.Duration) (bool, error) {
	c.acquireCalls++
	return true, nil
}

func (c *redeemRateLimitCacheStub) ReleaseRedeemLock(context.Context, string) error {
	c.releaseCalls++
	return nil
}

func TestPublicRedeemAllowsLastAttemptBeforeLimit(t *testing.T) {
	cache := &redeemRateLimitCacheStub{count: redeemMaxFailedAttempts - 1}
	redeemRepo := &redeemRejectRepo{code: RedeemCode{Code: "OTHER"}}
	svc := &RedeemService{redeemRepo: redeemRepo, cache: cache}

	result, err := svc.Redeem(context.Background(), 42, "MISSING")

	require.Nil(t, result)
	require.ErrorIs(t, err, ErrRedeemCodeNotFound)
	require.Equal(t, redeemMaxFailedAttempts, cache.count)
	require.Equal(t, 1, cache.getCalls)
	require.Equal(t, 1, cache.incrementCalls)
	require.Equal(t, 1, cache.acquireCalls)
	require.Equal(t, 1, cache.releaseCalls)
}

func TestPublicRedeemCountsOnlyConfiguredDomainFailures(t *testing.T) {
	past := time.Now().Add(-time.Minute)
	tests := []struct {
		name    string
		code    RedeemCode
		input   string
		wantErr error
	}{
		{
			name:    "not found",
			code:    RedeemCode{Code: "OTHER"},
			input:   "MISSING",
			wantErr: ErrRedeemCodeNotFound,
		},
		{
			name: "expired",
			code: RedeemCode{
				ID: 1, Code: "EXPIRED", Type: RedeemTypeBalance, Value: 10,
				Status: StatusUnused, ExpiresAt: &past,
			},
			input:   "EXPIRED",
			wantErr: ErrRedeemCodeExpired,
		},
		{
			name: "used",
			code: RedeemCode{
				ID: 2, Code: "USED", Type: RedeemTypeBalance, Value: 10,
				Status: StatusUsed,
			},
			input:   "USED",
			wantErr: ErrRedeemCodeUsed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := &redeemRateLimitCacheStub{}
			svc := &RedeemService{
				redeemRepo: &redeemRejectRepo{code: tt.code},
				cache:      cache,
			}

			result, err := svc.Redeem(context.Background(), 42, tt.input)

			require.Nil(t, result)
			require.ErrorIs(t, err, tt.wantErr)
			require.Equal(t, 1, cache.getCalls)
			require.Equal(t, 1, cache.incrementCalls)
			require.Equal(t, 1, cache.count)
		})
	}
}

func TestPublicRedeemRateLimitFailsOpenWhenCounterReadFails(t *testing.T) {
	cache := &redeemRateLimitCacheStub{
		count:  redeemMaxFailedAttempts,
		getErr: errors.New("redis unavailable"),
	}
	svc := &RedeemService{
		redeemRepo: &redeemRejectRepo{code: RedeemCode{Code: "OTHER"}},
		cache:      cache,
	}

	result, err := svc.Redeem(context.Background(), 42, "MISSING")

	require.Nil(t, result)
	require.ErrorIs(t, err, ErrRedeemCodeNotFound)
	require.Equal(t, 1, cache.getCalls)
	require.Equal(t, 1, cache.incrementCalls)
}

func TestPublicRedeemCounterWriteFailurePreservesDomainError(t *testing.T) {
	cache := &redeemRateLimitCacheStub{incrementErr: errors.New("redis unavailable")}
	svc := &RedeemService{
		redeemRepo: &redeemRejectRepo{code: RedeemCode{Code: "OTHER"}},
		cache:      cache,
	}

	result, err := svc.Redeem(context.Background(), 42, "MISSING")

	require.Nil(t, result)
	require.ErrorIs(t, err, ErrRedeemCodeNotFound)
	require.Equal(t, 1, cache.incrementCalls)
	require.Zero(t, cache.count)
}

func TestSuccessfulPublicRedeemDoesNotMutateFailureCounter(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	userID := int64(42)
	code := &RedeemCode{
		ID: 101, Code: "PUBLIC-SUCCESS", Type: RedeemTypeBalance, Value: 10, Status: StatusUnused,
	}
	redeemRepo := &paymentOrderLifecycleRedeemRepo{
		codesByCode: map[string]*RedeemCode{code.Code: code},
	}
	userRepo := &mockUserRepo{getByIDUser: &User{ID: userID}}
	userRepo.adjustBalanceFn = func(_ context.Context, id int64, delta float64) (BalanceChange, error) {
		require.Equal(t, userID, id)
		require.Equal(t, code.Value, delta)
		change := BalanceChange{Old: userRepo.getByIDUser.Balance}
		userRepo.getByIDUser.Balance += delta
		change.New = userRepo.getByIDUser.Balance
		return change, nil
	}
	cache := &redeemRateLimitCacheStub{count: 7}
	ledgerRepo := &balanceLedgerRepoStub{}
	svc := NewRedeemService(redeemRepo, userRepo, nil, cache, nil, client, nil, nil, ledgerRepo)

	result, err := svc.Redeem(ctx, userID, code.Code)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 1, cache.getCalls)
	require.Zero(t, cache.incrementCalls)
	require.Equal(t, 7, cache.count)
	require.Equal(t, 1, cache.acquireCalls)
	require.Equal(t, 1, cache.releaseCalls)
	require.Len(t, redeemRepo.useCalls, 1)
	require.Len(t, ledgerRepo.inserted, 1)
	entry := ledgerRepo.inserted[0]
	require.Equal(t, userID, entry.UserID)
	require.Equal(t, float64(10), entry.AmountDelta)
	require.Zero(t, entry.BalanceBefore)
	require.Equal(t, float64(10), entry.BalanceAfter)
	require.NotNil(t, entry.SourceType)
	require.Equal(t, BalanceLedgerSourceRedeemCode, *entry.SourceType)
	require.NotNil(t, entry.SourceID)
	require.Equal(t, "101", *entry.SourceID)
}
