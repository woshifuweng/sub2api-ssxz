//go:build integration

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestUsageBillingRepositoryApply_DeduplicatesBalanceBilling(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := NewUsageBillingRepository(client, integrationDB)

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("usage-billing-user-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      100,
	})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{
		UserID: user.ID,
		Key:    "sk-usage-billing-" + uuid.NewString(),
		Name:   "billing",
		Quota:  1,
	})
	account := mustCreateAccount(t, client, &service.Account{
		Name: "usage-billing-account-" + uuid.NewString(),
		Type: service.AccountTypeAPIKey,
	})

	requestID := uuid.NewString()
	cmd := &service.UsageBillingCommand{
		RequestID:           requestID,
		APIKeyID:            apiKey.ID,
		UserID:              user.ID,
		AccountID:           account.ID,
		AccountType:         service.AccountTypeAPIKey,
		BalanceCost:         1.25,
		APIKeyQuotaCost:     1.25,
		APIKeyRateLimitCost: 1.25,
	}

	result1, err := repo.Apply(ctx, cmd)
	require.NoError(t, err)
	require.NotNil(t, result1)
	require.True(t, result1.Applied)
	require.True(t, result1.APIKeyQuotaExhausted)

	result2, err := repo.Apply(ctx, cmd)
	require.NoError(t, err)
	require.NotNil(t, result2)
	require.False(t, result2.Applied)

	var balance float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT balance FROM users WHERE id = $1", user.ID).Scan(&balance))
	require.InDelta(t, 98.75, balance, 0.000001)

	var quotaUsed float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT quota_used FROM api_keys WHERE id = $1", apiKey.ID).Scan(&quotaUsed))
	require.InDelta(t, 1.25, quotaUsed, 0.000001)

	var usage5h float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT usage_5h FROM api_keys WHERE id = $1", apiKey.ID).Scan(&usage5h))
	require.InDelta(t, 1.25, usage5h, 0.000001)

	var status string
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT status FROM api_keys WHERE id = $1", apiKey.ID).Scan(&status))
	require.Equal(t, service.StatusAPIKeyQuotaExhausted, status)

	var dedupCount int
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM usage_billing_dedup WHERE request_id = $1 AND api_key_id = $2", requestID, apiKey.ID).Scan(&dedupCount))
	require.Equal(t, 1, dedupCount)
}

func TestUsageBillingRepositoryApply_AccruesConsumptionAffiliateRebateExactlyOnce(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := NewUsageBillingRepository(client, integrationDB)
	setUsageAffiliateSettings(t, ctx, map[string]string{
		service.SettingKeyAffiliateEnabled:             "true",
		service.SettingKeyAffiliateRebateRate:          "5",
		service.SettingKeyAffiliateRebateFreezeHours:   "0",
		service.SettingKeyAffiliateRebateDurationDays:  "0",
		service.SettingKeyAffiliateRebatePerInviteeCap: "0",
	})

	inviter := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("usage-affiliate-inviter-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
	})
	invitee := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("usage-affiliate-invitee-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      10,
	})
	insertUsageAffiliateProfile(t, ctx, inviter.ID, nil)
	insertUsageAffiliateProfile(t, ctx, invitee.ID, &inviter.ID)
	apiKey := mustCreateApiKey(t, client, &service.APIKey{
		UserID: invitee.ID,
		Key:    "sk-usage-affiliate-" + uuid.NewString(),
		Name:   "usage-affiliate",
	})

	cmd := &service.UsageBillingCommand{
		RequestID:   "usage-affiliate-" + uuid.NewString(),
		APIKeyID:    apiKey.ID,
		UserID:      invitee.ID,
		BalanceCost: 2,
	}
	result, err := repo.Apply(ctx, cmd)
	require.NoError(t, err)
	require.True(t, result.Applied)
	require.InDelta(t, 2, result.BalanceCharged, 0.00000001)
	require.InDelta(t, 0.1, result.AffiliateRebate, 0.00000001)

	duplicate, err := repo.Apply(ctx, cmd)
	require.NoError(t, err)
	require.False(t, duplicate.Applied)
	require.Zero(t, duplicate.AffiliateRebate)

	var available, history float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		SELECT aff_quota::double precision, aff_history_quota::double precision
		FROM user_affiliates WHERE user_id = $1
	`, inviter.ID).Scan(&available, &history))
	require.InDelta(t, 0.1, available, 0.00000001)
	require.InDelta(t, 0.1, history, 0.00000001)

	var ledgerCount int
	var ledgerAmount float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		SELECT COUNT(*), COALESCE(SUM(amount), 0)::double precision
		FROM user_affiliate_ledger
		WHERE user_id = $1 AND source_user_id = $2 AND action = 'accrue'
	`, inviter.ID, invitee.ID).Scan(&ledgerCount, &ledgerAmount))
	require.Equal(t, 1, ledgerCount)
	require.InDelta(t, 0.1, ledgerAmount, 0.00000001)
}

func TestUsageBillingRepositoryApply_AffiliateRebateUsesCollectedBalanceNotShortfall(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := NewUsageBillingRepository(client, integrationDB)
	setUsageAffiliateSettings(t, ctx, map[string]string{
		service.SettingKeyAffiliateEnabled:             "true",
		service.SettingKeyAffiliateRebateRate:          "5",
		service.SettingKeyAffiliateRebateFreezeHours:   "0",
		service.SettingKeyAffiliateRebateDurationDays:  "0",
		service.SettingKeyAffiliateRebatePerInviteeCap: "0",
	})

	inviter := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("usage-affiliate-shortfall-inviter-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
	})
	invitee := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("usage-affiliate-shortfall-invitee-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      1,
	})
	insertUsageAffiliateProfile(t, ctx, inviter.ID, nil)
	insertUsageAffiliateProfile(t, ctx, invitee.ID, &inviter.ID)
	apiKey := mustCreateApiKey(t, client, &service.APIKey{
		UserID: invitee.ID,
		Key:    "sk-usage-affiliate-shortfall-" + uuid.NewString(),
		Name:   "usage-affiliate-shortfall",
	})

	result, err := repo.Apply(ctx, &service.UsageBillingCommand{
		RequestID:   "usage-affiliate-shortfall-" + uuid.NewString(),
		APIKeyID:    apiKey.ID,
		UserID:      invitee.ID,
		BalanceCost: 2,
	})
	require.NoError(t, err)
	require.True(t, result.Applied)
	require.InDelta(t, 1, result.BalanceCharged, 0.00000001)
	require.InDelta(t, 1, result.BalanceShortfall, 0.00000001)
	require.InDelta(t, 0.05, result.AffiliateRebate, 0.00000001)
}

func setUsageAffiliateSettings(t *testing.T, ctx context.Context, values map[string]string) {
	t.Helper()

	previous := make(map[string]string, len(values))
	present := make(map[string]bool, len(values))
	for key := range values {
		var value string
		err := integrationDB.QueryRowContext(ctx, "SELECT value FROM settings WHERE key = $1", key).Scan(&value)
		if err == nil {
			previous[key] = value
			present[key] = true
		} else {
			require.ErrorIs(t, err, sql.ErrNoRows)
		}
	}

	for key, value := range values {
		_, err := integrationDB.ExecContext(ctx, `
			INSERT INTO settings (key, value, updated_at)
			VALUES ($1, $2, NOW())
			ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()
		`, key, value)
		require.NoError(t, err)
	}

	t.Cleanup(func() {
		for key := range values {
			if !present[key] {
				_, err := integrationDB.ExecContext(context.Background(), "DELETE FROM settings WHERE key = $1", key)
				require.NoError(t, err)
				continue
			}
			_, err := integrationDB.ExecContext(context.Background(), `
				INSERT INTO settings (key, value, updated_at)
				VALUES ($1, $2, NOW())
				ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()
			`, key, previous[key])
			require.NoError(t, err)
		}
	})
}

func insertUsageAffiliateProfile(t *testing.T, ctx context.Context, userID int64, inviterID *int64) {
	t.Helper()
	code := strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	_, err := integrationDB.ExecContext(ctx, `
		INSERT INTO user_affiliates (user_id, aff_code, inviter_id, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
	`, userID, code, inviterID)
	require.NoError(t, err)
}

func TestUsageBillingRepositoryApply_SettlesBalanceShortfallWithoutLeavingReusableBalance(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := NewUsageBillingRepository(client, integrationDB)

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("usage-billing-no-overdraft-user-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      1,
	})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{
		UserID: user.ID,
		Key:    "sk-usage-billing-no-overdraft-" + uuid.NewString(),
		Name:   "billing-no-overdraft",
		Quota:  10,
	})
	requestID := uuid.NewString()

	cmd := &service.UsageBillingCommand{
		RequestID:           requestID,
		APIKeyID:            apiKey.ID,
		UserID:              user.ID,
		BalanceCost:         1.25,
		APIKeyQuotaCost:     1.25,
		APIKeyRateLimitCost: 1.25,
	}
	result, err := repo.Apply(ctx, cmd)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Applied)
	require.True(t, result.BalanceExhausted)
	require.NotNil(t, result.NewBalance)
	require.InDelta(t, 0, *result.NewBalance, 0.000001)
	require.InDelta(t, 1, result.BalanceCharged, 0.000001)
	require.InDelta(t, 0.25, result.BalanceShortfall, 0.000001)

	var balance float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT balance FROM users WHERE id = $1", user.ID).Scan(&balance))
	require.InDelta(t, 0, balance, 0.000001)

	var quotaUsed float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT quota_used FROM api_keys WHERE id = $1", apiKey.ID).Scan(&quotaUsed))
	require.InDelta(t, 1.25, quotaUsed, 0.000001)

	var usage5h float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT usage_5h FROM api_keys WHERE id = $1", apiKey.ID).Scan(&usage5h))
	require.InDelta(t, 1.25, usage5h, 0.000001)

	var dedupCount int
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM usage_billing_dedup WHERE request_id = $1 AND api_key_id = $2", requestID, apiKey.ID).Scan(&dedupCount))
	require.Equal(t, 1, dedupCount)

	duplicate, err := repo.Apply(ctx, cmd)
	require.NoError(t, err)
	require.NotNil(t, duplicate)
	require.False(t, duplicate.Applied)
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT balance FROM users WHERE id = $1", user.ID).Scan(&balance))
	require.InDelta(t, 0, balance, 0.000001)
}

func TestUsageBillingRepositoryApply_ConcurrentShortfallsNeverGoNegativeOrDoubleApply(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := NewUsageBillingRepository(client, integrationDB)

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("usage-billing-concurrent-shortfall-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      1,
	})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{
		UserID: user.ID,
		Key:    "sk-usage-billing-concurrent-shortfall-" + uuid.NewString(),
		Name:   "billing-concurrent-shortfall",
	})

	const requestCount = 5
	type applyOutcome struct {
		result *service.UsageBillingApplyResult
		err    error
	}
	outcomes := make(chan applyOutcome, requestCount)
	var wg sync.WaitGroup
	for i := 0; i < requestCount; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			result, err := repo.Apply(ctx, &service.UsageBillingCommand{
				RequestID:   fmt.Sprintf("concurrent-shortfall-%d-%s", index, uuid.NewString()),
				APIKeyID:    apiKey.ID,
				UserID:      user.ID,
				BalanceCost: 0.4,
			})
			outcomes <- applyOutcome{result: result, err: err}
		}(i)
	}
	wg.Wait()
	close(outcomes)

	var chargedTotal float64
	var shortfallTotal float64
	for outcome := range outcomes {
		require.NoError(t, outcome.err)
		require.NotNil(t, outcome.result)
		require.True(t, outcome.result.Applied)
		chargedTotal += outcome.result.BalanceCharged
		shortfallTotal += outcome.result.BalanceShortfall
	}
	require.InDelta(t, 1, chargedTotal, 0.000001)
	require.InDelta(t, 1, shortfallTotal, 0.000001)

	var balance float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT balance FROM users WHERE id = $1", user.ID).Scan(&balance))
	require.InDelta(t, 0, balance, 0.000001)

	var dedupCount int
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM usage_billing_dedup WHERE api_key_id = $1 AND request_id LIKE 'concurrent-shortfall-%'", apiKey.ID).Scan(&dedupCount))
	require.Equal(t, requestCount, dedupCount)
}

func TestUsageBillingRepositoryApply_DeduplicatesSubscriptionBilling(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := NewUsageBillingRepository(client, integrationDB)

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("usage-billing-sub-user-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
	})
	group := mustCreateGroup(t, client, &service.Group{
		Name:             "usage-billing-group-" + uuid.NewString(),
		Platform:         service.PlatformAnthropic,
		SubscriptionType: service.SubscriptionTypeSubscription,
	})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{
		UserID:  user.ID,
		GroupID: &group.ID,
		Key:     "sk-usage-billing-sub-" + uuid.NewString(),
		Name:    "billing-sub",
	})
	subscription := mustCreateSubscription(t, client, &service.UserSubscription{
		UserID:  user.ID,
		GroupID: group.ID,
	})

	requestID := uuid.NewString()
	cmd := &service.UsageBillingCommand{
		RequestID:        requestID,
		APIKeyID:         apiKey.ID,
		UserID:           user.ID,
		AccountID:        0,
		SubscriptionID:   &subscription.ID,
		SubscriptionCost: 2.5,
	}

	result1, err := repo.Apply(ctx, cmd)
	require.NoError(t, err)
	require.True(t, result1.Applied)

	result2, err := repo.Apply(ctx, cmd)
	require.NoError(t, err)
	require.False(t, result2.Applied)

	var dailyUsage float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT daily_usage_usd FROM user_subscriptions WHERE id = $1", subscription.ID).Scan(&dailyUsage))
	require.InDelta(t, 2.5, dailyUsage, 0.000001)
}

func TestUsageBillingRepositoryApply_RequestFingerprintConflict(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := NewUsageBillingRepository(client, integrationDB)

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("usage-billing-conflict-user-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      100,
	})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{
		UserID: user.ID,
		Key:    "sk-usage-billing-conflict-" + uuid.NewString(),
		Name:   "billing-conflict",
	})

	requestID := uuid.NewString()
	_, err := repo.Apply(ctx, &service.UsageBillingCommand{
		RequestID:   requestID,
		APIKeyID:    apiKey.ID,
		UserID:      user.ID,
		BalanceCost: 1.25,
	})
	require.NoError(t, err)

	_, err = repo.Apply(ctx, &service.UsageBillingCommand{
		RequestID:   requestID,
		APIKeyID:    apiKey.ID,
		UserID:      user.ID,
		BalanceCost: 2.50,
	})
	require.ErrorIs(t, err, service.ErrUsageBillingRequestConflict)
}

func TestUsageBillingRepositoryApply_UpdatesAccountQuota(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := NewUsageBillingRepository(client, integrationDB)

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("usage-billing-account-user-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
	})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{
		UserID: user.ID,
		Key:    "sk-usage-billing-account-" + uuid.NewString(),
		Name:   "billing-account",
	})
	account := mustCreateAccount(t, client, &service.Account{
		Name: "usage-billing-account-quota-" + uuid.NewString(),
		Type: service.AccountTypeAPIKey,
		Extra: map[string]any{
			"quota_limit": 100.0,
		},
	})

	_, err := repo.Apply(ctx, &service.UsageBillingCommand{
		RequestID:        uuid.NewString(),
		APIKeyID:         apiKey.ID,
		UserID:           user.ID,
		AccountID:        account.ID,
		AccountType:      service.AccountTypeAPIKey,
		AccountQuotaCost: 3.5,
	})
	require.NoError(t, err)

	var quotaUsed float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT COALESCE((extra->>'quota_used')::numeric, 0) FROM accounts WHERE id = $1", account.ID).Scan(&quotaUsed))
	require.InDelta(t, 3.5, quotaUsed, 0.000001)
}

func TestDashboardAggregationRepositoryCleanupUsageBillingDedup_BatchDeletesOldRows(t *testing.T) {
	ctx := context.Background()
	repo := newDashboardAggregationRepositoryWithSQL(integrationDB)

	oldRequestID := "dedup-old-" + uuid.NewString()
	newRequestID := "dedup-new-" + uuid.NewString()
	oldCreatedAt := time.Now().UTC().AddDate(0, 0, -400)
	newCreatedAt := time.Now().UTC().Add(-time.Hour)

	_, err := integrationDB.ExecContext(ctx, `
		INSERT INTO usage_billing_dedup (request_id, api_key_id, request_fingerprint, created_at)
		VALUES ($1, 1, $2, $3), ($4, 1, $5, $6)
	`,
		oldRequestID, strings.Repeat("a", 64), oldCreatedAt,
		newRequestID, strings.Repeat("b", 64), newCreatedAt,
	)
	require.NoError(t, err)

	require.NoError(t, repo.CleanupUsageBillingDedup(ctx, time.Now().UTC().AddDate(0, 0, -365)))

	var oldCount int
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM usage_billing_dedup WHERE request_id = $1", oldRequestID).Scan(&oldCount))
	require.Equal(t, 0, oldCount)

	var newCount int
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM usage_billing_dedup WHERE request_id = $1", newRequestID).Scan(&newCount))
	require.Equal(t, 1, newCount)

	var archivedCount int
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM usage_billing_dedup_archive WHERE request_id = $1", oldRequestID).Scan(&archivedCount))
	require.Equal(t, 1, archivedCount)
}

func TestUsageBillingRepositoryApply_DeduplicatesAgainstArchivedKey(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := NewUsageBillingRepository(client, integrationDB)
	aggRepo := newDashboardAggregationRepositoryWithSQL(integrationDB)

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("usage-billing-archive-user-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      100,
	})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{
		UserID: user.ID,
		Key:    "sk-usage-billing-archive-" + uuid.NewString(),
		Name:   "billing-archive",
	})

	requestID := uuid.NewString()
	cmd := &service.UsageBillingCommand{
		RequestID:   requestID,
		APIKeyID:    apiKey.ID,
		UserID:      user.ID,
		BalanceCost: 1.25,
	}

	result1, err := repo.Apply(ctx, cmd)
	require.NoError(t, err)
	require.True(t, result1.Applied)

	_, err = integrationDB.ExecContext(ctx, `
		UPDATE usage_billing_dedup
		SET created_at = $1
		WHERE request_id = $2 AND api_key_id = $3
	`, time.Now().UTC().AddDate(0, 0, -400), requestID, apiKey.ID)
	require.NoError(t, err)
	require.NoError(t, aggRepo.CleanupUsageBillingDedup(ctx, time.Now().UTC().AddDate(0, 0, -365)))

	result2, err := repo.Apply(ctx, cmd)
	require.NoError(t, err)
	require.False(t, result2.Applied)

	var balance float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT balance FROM users WHERE id = $1", user.ID).Scan(&balance))
	require.InDelta(t, 98.75, balance, 0.000001)
}
