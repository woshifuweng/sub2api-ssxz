package service

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type billingCacheWorkerStub struct {
	balanceUpdates      int64
	subscriptionUpdates int64
}

func (b *billingCacheWorkerStub) GetUserBalance(ctx context.Context, userID int64) (float64, error) {
	return 0, errors.New("not implemented")
}

func (b *billingCacheWorkerStub) SetUserBalance(ctx context.Context, userID int64, balance float64) error {
	atomic.AddInt64(&b.balanceUpdates, 1)
	return nil
}

func (b *billingCacheWorkerStub) DeductUserBalance(ctx context.Context, userID int64, amount float64) error {
	atomic.AddInt64(&b.balanceUpdates, 1)
	return nil
}

func (b *billingCacheWorkerStub) InvalidateUserBalance(ctx context.Context, userID int64) error {
	return nil
}

func (b *billingCacheWorkerStub) GetSubscriptionCache(ctx context.Context, userID, groupID int64) (*SubscriptionCacheData, error) {
	return nil, errors.New("not implemented")
}

func (b *billingCacheWorkerStub) SetSubscriptionCache(ctx context.Context, userID, groupID int64, data *SubscriptionCacheData) error {
	atomic.AddInt64(&b.subscriptionUpdates, 1)
	return nil
}

func (b *billingCacheWorkerStub) UpdateSubscriptionUsage(ctx context.Context, userID, groupID int64, cost float64) error {
	atomic.AddInt64(&b.subscriptionUpdates, 1)
	return nil
}

func (b *billingCacheWorkerStub) InvalidateSubscriptionCache(ctx context.Context, userID, groupID int64) error {
	return nil
}

func (b *billingCacheWorkerStub) GetAPIKeyRateLimit(ctx context.Context, keyID int64) (*APIKeyRateLimitCacheData, error) {
	return nil, errors.New("not implemented")
}

func (b *billingCacheWorkerStub) SetAPIKeyRateLimit(ctx context.Context, keyID int64, data *APIKeyRateLimitCacheData) error {
	return nil
}

func (b *billingCacheWorkerStub) UpdateAPIKeyRateLimitUsage(ctx context.Context, keyID int64, cost float64) error {
	return nil
}

func (b *billingCacheWorkerStub) InvalidateAPIKeyRateLimit(ctx context.Context, keyID int64) error {
	return nil
}

func TestBillingCacheServiceQueueHighLoad(t *testing.T) {
	cache := &billingCacheWorkerStub{}
	svc := NewBillingCacheService(cache, nil, nil, nil, &config.Config{})
	t.Cleanup(svc.Stop)

	start := time.Now()
	for i := 0; i < cacheWriteBufferSize*2; i++ {
		svc.QueueDeductBalance(1, 1)
	}
	require.Less(t, time.Since(start), 2*time.Second)

	svc.QueueUpdateSubscriptionUsage(1, 2, 1.5)

	require.Eventually(t, func() bool {
		return atomic.LoadInt64(&cache.balanceUpdates) > 0
	}, 2*time.Second, 10*time.Millisecond)

	require.Eventually(t, func() bool {
		return atomic.LoadInt64(&cache.subscriptionUpdates) > 0
	}, 2*time.Second, 10*time.Millisecond)
}

func TestBillingCacheServiceEnqueueAfterStopReturnsFalse(t *testing.T) {
	cache := &billingCacheWorkerStub{}
	svc := NewBillingCacheService(cache, nil, nil, nil, &config.Config{})
	svc.Stop()

	enqueued := svc.enqueueCacheWrite(cacheWriteTask{
		kind:   cacheWriteDeductBalance,
		userID: 1,
		amount: 1,
	})
	require.False(t, enqueued)
}

type billingEligibilityUserRepoStub struct {
	UserRepository
	user *User
}

func (s *billingEligibilityUserRepoStub) GetByID(context.Context, int64) (*User, error) {
	return s.user, nil
}

type billingEligibilitySubRepoStub struct {
	UserSubscriptionRepository
	sub *UserSubscription
}

func (s *billingEligibilitySubRepoStub) GetActiveByUserIDAndGroupID(context.Context, int64, int64) (*UserSubscription, error) {
	return s.sub, nil
}

type billingEligibilityRateLimitLoaderStub struct {
	data *APIKeyRateLimitData
}

func (s *billingEligibilityRateLimitLoaderStub) GetRateLimitData(context.Context, int64) (*APIKeyRateLimitData, error) {
	return s.data, nil
}

func TestBillingCacheServiceCheckBillingEligibilityForCostRejectsBalanceOverdraft(t *testing.T) {
	svc := &BillingCacheService{
		userRepo: &billingEligibilityUserRepoStub{user: &User{ID: 1, Balance: 1}},
		cfg:      &config.Config{},
	}

	err := svc.CheckBillingEligibilityForCost(context.Background(), &User{ID: 1}, nil, nil, nil, 2)

	require.ErrorIs(t, err, ErrInsufficientBalance)
}

func TestBillingCacheServiceCheckBillingEligibilityForCostRejectsAPIKeyQuotaOverage(t *testing.T) {
	svc := &BillingCacheService{
		userRepo: &billingEligibilityUserRepoStub{user: &User{ID: 1, Balance: 10}},
		cfg:      &config.Config{},
	}
	apiKey := &APIKey{
		ID:        2,
		Quota:     5,
		QuotaUsed: 4.5,
	}

	err := svc.CheckBillingEligibilityForCost(context.Background(), &User{ID: 1}, apiKey, nil, nil, 1)

	require.ErrorIs(t, err, ErrAPIKeyQuotaExhausted)
}

func TestBillingCacheServiceCheckBillingEligibilityForCostRejectsAPIKeyRateLimitOverage(t *testing.T) {
	windowStart := time.Now()
	svc := &BillingCacheService{
		userRepo: &billingEligibilityUserRepoStub{user: &User{ID: 1, Balance: 10}},
		apiKeyRateLimitLoader: &billingEligibilityRateLimitLoaderStub{
			data: &APIKeyRateLimitData{
				Usage5h:       4.5,
				Window5hStart: &windowStart,
			},
		},
		cfg: &config.Config{},
	}
	apiKey := &APIKey{
		ID:          2,
		RateLimit5h: 5,
	}

	err := svc.CheckBillingEligibilityForCost(context.Background(), &User{ID: 1}, apiKey, nil, nil, 1)

	require.ErrorIs(t, err, ErrAPIKeyRateLimit5hExceeded)
}

func TestBillingCacheServiceCheckBillingEligibilityForCostRejectsSubscriptionLimitOverage(t *testing.T) {
	dailyLimit := 5.0
	group := &Group{
		ID:               3,
		SubscriptionType: SubscriptionTypeSubscription,
		DailyLimitUSD:    &dailyLimit,
	}
	sub := &UserSubscription{
		ID:            4,
		UserID:        1,
		GroupID:       group.ID,
		Status:        SubscriptionStatusActive,
		ExpiresAt:     time.Now().Add(time.Hour),
		DailyUsageUSD: 4.5,
	}
	svc := &BillingCacheService{
		subRepo: &billingEligibilitySubRepoStub{sub: sub},
		cfg:     &config.Config{},
	}

	err := svc.CheckBillingEligibilityForCost(context.Background(), &User{ID: 1}, nil, group, sub, 1)

	require.ErrorIs(t, err, ErrDailyLimitExceeded)
}
