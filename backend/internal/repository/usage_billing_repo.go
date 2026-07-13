package repository

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"strconv"
	"strings"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type usageBillingRepository struct {
	db *sql.DB
}

func NewUsageBillingRepository(_ *dbent.Client, sqlDB *sql.DB) service.UsageBillingRepository {
	return &usageBillingRepository{db: sqlDB}
}

func (r *usageBillingRepository) Apply(ctx context.Context, cmd *service.UsageBillingCommand) (_ *service.UsageBillingApplyResult, err error) {
	if cmd == nil {
		return &service.UsageBillingApplyResult{}, nil
	}
	if r == nil || r.db == nil {
		return nil, errors.New("usage billing repository db is nil")
	}

	cmd.Normalize()
	if cmd.RequestID == "" {
		return nil, service.ErrUsageBillingRequestIDRequired
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	applied, err := r.claimUsageBillingKey(ctx, tx, cmd)
	if err != nil {
		return nil, err
	}
	if !applied {
		return &service.UsageBillingApplyResult{Applied: false}, nil
	}

	result := &service.UsageBillingApplyResult{Applied: true}
	if err := r.applyUsageBillingEffects(ctx, tx, cmd, result); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	tx = nil
	return result, nil
}

func (r *usageBillingRepository) claimUsageBillingKey(ctx context.Context, tx *sql.Tx, cmd *service.UsageBillingCommand) (bool, error) {
	var id int64
	err := tx.QueryRowContext(ctx, `
		INSERT INTO usage_billing_dedup (request_id, api_key_id, request_fingerprint)
		VALUES ($1, $2, $3)
		ON CONFLICT (request_id, api_key_id) DO NOTHING
		RETURNING id
	`, cmd.RequestID, cmd.APIKeyID, cmd.RequestFingerprint).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		var existingFingerprint string
		if err := tx.QueryRowContext(ctx, `
			SELECT request_fingerprint
			FROM usage_billing_dedup
			WHERE request_id = $1 AND api_key_id = $2
		`, cmd.RequestID, cmd.APIKeyID).Scan(&existingFingerprint); err != nil {
			return false, err
		}
		if strings.TrimSpace(existingFingerprint) != strings.TrimSpace(cmd.RequestFingerprint) {
			return false, service.ErrUsageBillingRequestConflict
		}
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var archivedFingerprint string
	err = tx.QueryRowContext(ctx, `
		SELECT request_fingerprint
		FROM usage_billing_dedup_archive
		WHERE request_id = $1 AND api_key_id = $2
	`, cmd.RequestID, cmd.APIKeyID).Scan(&archivedFingerprint)
	if err == nil {
		if strings.TrimSpace(archivedFingerprint) != strings.TrimSpace(cmd.RequestFingerprint) {
			return false, service.ErrUsageBillingRequestConflict
		}
		return false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}
	return true, nil
}

func (r *usageBillingRepository) applyUsageBillingEffects(ctx context.Context, tx *sql.Tx, cmd *service.UsageBillingCommand, result *service.UsageBillingApplyResult) error {
	if cmd.SubscriptionCost > 0 && cmd.SubscriptionID != nil {
		if err := incrementUsageBillingSubscription(ctx, tx, *cmd.SubscriptionID, cmd.SubscriptionCost); err != nil {
			return err
		}
	}

	if cmd.BalanceCost > 0 {
		newBalance, charged, shortfall, err := settleUsageBillingBalance(ctx, tx, cmd.UserID, cmd.BalanceCost)
		if err != nil {
			return err
		}
		result.NewBalance = &newBalance
		result.BalanceCharged = charged
		result.BalanceShortfall = shortfall
		result.BalanceExhausted = newBalance <= 0

		rebate, err := accrueUsageAffiliateRebate(ctx, tx, cmd.UserID, charged)
		if err != nil {
			return err
		}
		result.AffiliateRebate = rebate
	}

	if cmd.APIKeyQuotaCost > 0 {
		exhausted, err := incrementUsageBillingAPIKeyQuota(ctx, tx, cmd.APIKeyID, cmd.APIKeyQuotaCost)
		if err != nil {
			return err
		}
		result.APIKeyQuotaExhausted = exhausted
	}

	if cmd.APIKeyRateLimitCost > 0 {
		if err := incrementUsageBillingAPIKeyRateLimit(ctx, tx, cmd.APIKeyID, cmd.APIKeyRateLimitCost); err != nil {
			return err
		}
	}

	if cmd.AccountQuotaCost > 0 && (strings.EqualFold(cmd.AccountType, service.AccountTypeAPIKey) || strings.EqualFold(cmd.AccountType, service.AccountTypeBedrock)) {
		if err := incrementUsageBillingAccountQuota(ctx, tx, cmd.AccountID, cmd.AccountQuotaCost); err != nil {
			return err
		}
	}

	return nil
}

type usageAffiliateSettings struct {
	enabled      bool
	ratePercent  float64
	freezeHours  int
	durationDays int
	perUserCap   float64
}

func accrueUsageAffiliateRebate(ctx context.Context, tx *sql.Tx, inviteeUserID int64, chargedAmount float64) (float64, error) {
	if tx == nil || inviteeUserID <= 0 || chargedAmount <= 0 || math.IsNaN(chargedAmount) || math.IsInf(chargedAmount, 0) {
		return 0, nil
	}

	settings, err := loadUsageAffiliateSettings(ctx, tx)
	if err != nil {
		return 0, err
	}
	if !settings.enabled || settings.ratePercent <= 0 {
		return 0, nil
	}

	var inviterID int64
	var customRate sql.NullFloat64
	var eligible bool
	err = tx.QueryRowContext(ctx, `
		SELECT invitee.inviter_id,
		       inviter.aff_rebate_rate_percent::double precision,
		       ($2 <= 0 OR invitee.created_at + ($2 * INTERVAL '1 day') >= NOW()) AS eligible
		FROM user_affiliates invitee
		JOIN user_affiliates inviter ON inviter.user_id = invitee.inviter_id
		WHERE invitee.user_id = $1
		FOR UPDATE OF inviter
	`, inviteeUserID, settings.durationDays).Scan(&inviterID, &customRate, &eligible)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if !eligible || inviterID <= 0 {
		return 0, nil
	}

	ratePercent := settings.ratePercent
	if customRate.Valid {
		ratePercent = clampUsageAffiliateRate(customRate.Float64)
	}
	rebate := roundUsageAffiliateAmount(chargedAmount * ratePercent / 100)
	if rebate <= 0 {
		return 0, nil
	}

	if settings.perUserCap > 0 {
		var accrued float64
		if err := tx.QueryRowContext(ctx, `
			SELECT COALESCE(SUM(amount), 0)::double precision
			FROM user_affiliate_ledger
			WHERE user_id = $1 AND source_user_id = $2 AND action = 'accrue'
		`, inviterID, inviteeUserID).Scan(&accrued); err != nil {
			return 0, err
		}
		remaining := roundUsageAffiliateAmount(settings.perUserCap - accrued)
		if remaining <= 0 {
			return 0, nil
		}
		if rebate > remaining {
			rebate = remaining
		}
	}

	quotaColumn := "aff_quota"
	if settings.freezeHours > 0 {
		quotaColumn = "aff_frozen_quota"
	}
	res, err := tx.ExecContext(ctx, `
		UPDATE user_affiliates
		SET `+quotaColumn+` = `+quotaColumn+` + $1,
		    aff_history_quota = aff_history_quota + $1,
		    updated_at = NOW()
		WHERE user_id = $2
	`, rebate, inviterID)
	if err != nil {
		return 0, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	if affected == 0 {
		return 0, nil
	}

	if settings.freezeHours > 0 {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO user_affiliate_ledger
				(user_id, action, amount, source_user_id, frozen_until, created_at, updated_at)
			VALUES ($1, 'accrue', $2, $3, NOW() + make_interval(hours => $4), NOW(), NOW())
		`, inviterID, rebate, inviteeUserID, settings.freezeHours)
	} else {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO user_affiliate_ledger
				(user_id, action, amount, source_user_id, created_at, updated_at)
			VALUES ($1, 'accrue', $2, $3, NOW(), NOW())
		`, inviterID, rebate, inviteeUserID)
	}
	if err != nil {
		return 0, err
	}
	return rebate, nil
}

func loadUsageAffiliateSettings(ctx context.Context, tx *sql.Tx) (usageAffiliateSettings, error) {
	settings := usageAffiliateSettings{
		ratePercent: service.AffiliateRebateRateDefault,
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT key, value
		FROM settings
		WHERE key IN ($1, $2, $3, $4, $5)
	`,
		service.SettingKeyAffiliateEnabled,
		service.SettingKeyAffiliateRebateRate,
		service.SettingKeyAffiliateRebateFreezeHours,
		service.SettingKeyAffiliateRebateDurationDays,
		service.SettingKeyAffiliateRebatePerInviteeCap,
	)
	if err != nil {
		return settings, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return settings, err
		}
		switch key {
		case service.SettingKeyAffiliateEnabled:
			settings.enabled = strings.EqualFold(strings.TrimSpace(value), "true")
		case service.SettingKeyAffiliateRebateRate:
			if parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64); err == nil {
				settings.ratePercent = clampUsageAffiliateRate(parsed)
			}
		case service.SettingKeyAffiliateRebateFreezeHours:
			if parsed, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && parsed >= 0 {
				if parsed > service.AffiliateRebateFreezeHoursMax {
					parsed = service.AffiliateRebateFreezeHoursMax
				}
				settings.freezeHours = parsed
			}
		case service.SettingKeyAffiliateRebateDurationDays:
			if parsed, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && parsed >= 0 {
				if parsed > service.AffiliateRebateDurationDaysMax {
					parsed = service.AffiliateRebateDurationDaysMax
				}
				settings.durationDays = parsed
			}
		case service.SettingKeyAffiliateRebatePerInviteeCap:
			if parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64); err == nil && parsed >= 0 {
				settings.perUserCap = parsed
			}
		}
	}
	return settings, rows.Err()
}

func clampUsageAffiliateRate(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return service.AffiliateRebateRateDefault
	}
	if value < service.AffiliateRebateRateMin {
		return service.AffiliateRebateRateMin
	}
	if value > service.AffiliateRebateRateMax {
		return service.AffiliateRebateRateMax
	}
	return value
}

func roundUsageAffiliateAmount(value float64) float64 {
	return math.Round(value*1e8) / 1e8
}

func incrementUsageBillingSubscription(ctx context.Context, tx *sql.Tx, subscriptionID int64, costUSD float64) error {
	const updateSQL = `
		UPDATE user_subscriptions us
		SET
			daily_usage_usd = us.daily_usage_usd + $1,
			weekly_usage_usd = us.weekly_usage_usd + $1,
			monthly_usage_usd = us.monthly_usage_usd + $1,
			updated_at = NOW()
		FROM groups g
		WHERE us.id = $2
			AND us.deleted_at IS NULL
			AND us.group_id = g.id
			AND g.deleted_at IS NULL
	`
	res, err := tx.ExecContext(ctx, updateSQL, costUSD, subscriptionID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected > 0 {
		return nil
	}
	return service.ErrSubscriptionNotFound
}

func settleUsageBillingBalance(ctx context.Context, tx *sql.Tx, userID int64, amount float64) (newBalance, charged, shortfall float64, err error) {
	var oldBalance float64
	err = tx.QueryRowContext(ctx, `
		SELECT balance
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
		FOR UPDATE
	`, userID).Scan(&oldBalance)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, 0, 0, service.ErrUserNotFound
	}
	if err != nil {
		return 0, 0, 0, err
	}

	available := oldBalance
	if available < 0 {
		available = 0
	}
	charged = amount
	if charged > available {
		charged = available
	}
	if charged < 0 {
		charged = 0
	}
	newBalance = available - charged
	if newBalance < 1e-9 {
		newBalance = 0
	}
	shortfall = amount - charged
	if shortfall < 1e-9 {
		shortfall = 0
	}

	if _, err = tx.ExecContext(ctx, `
		UPDATE users
		SET balance = $1,
			updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
	`, newBalance, userID); err != nil {
		return 0, 0, 0, err
	}
	return newBalance, charged, shortfall, nil
}

func incrementUsageBillingAPIKeyQuota(ctx context.Context, tx *sql.Tx, apiKeyID int64, amount float64) (bool, error) {
	var exhausted bool
	err := tx.QueryRowContext(ctx, `
		UPDATE api_keys
		SET quota_used = quota_used + $1,
			status = CASE
				WHEN quota > 0
					AND status = $3
					AND quota_used < quota
					AND quota_used + $1 >= quota
				THEN $4
				ELSE status
			END,
			updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
		RETURNING quota > 0 AND quota_used >= quota AND quota_used - $1 < quota
	`, amount, apiKeyID, service.StatusAPIKeyActive, service.StatusAPIKeyQuotaExhausted).Scan(&exhausted)
	if errors.Is(err, sql.ErrNoRows) {
		return false, service.ErrAPIKeyNotFound
	}
	if err != nil {
		return false, err
	}
	return exhausted, nil
}

func incrementUsageBillingAPIKeyRateLimit(ctx context.Context, tx *sql.Tx, apiKeyID int64, cost float64) error {
	res, err := tx.ExecContext(ctx, `
		UPDATE api_keys SET
			usage_5h = CASE WHEN window_5h_start IS NOT NULL AND window_5h_start + INTERVAL '5 hours' <= NOW() THEN $1 ELSE usage_5h + $1 END,
			usage_1d = CASE WHEN window_1d_start IS NOT NULL AND window_1d_start + INTERVAL '24 hours' <= NOW() THEN $1 ELSE usage_1d + $1 END,
			usage_7d = CASE WHEN window_7d_start IS NOT NULL AND window_7d_start + INTERVAL '7 days' <= NOW() THEN $1 ELSE usage_7d + $1 END,
			window_5h_start = CASE WHEN window_5h_start IS NULL OR window_5h_start + INTERVAL '5 hours' <= NOW() THEN NOW() ELSE window_5h_start END,
			window_1d_start = CASE WHEN window_1d_start IS NULL OR window_1d_start + INTERVAL '24 hours' <= NOW() THEN date_trunc('day', NOW()) ELSE window_1d_start END,
			window_7d_start = CASE WHEN window_7d_start IS NULL OR window_7d_start + INTERVAL '7 days' <= NOW() THEN date_trunc('day', NOW()) ELSE window_7d_start END,
			updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
	`, cost, apiKeyID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrAPIKeyNotFound
	}
	return nil
}

func incrementUsageBillingAccountQuota(ctx context.Context, tx *sql.Tx, accountID int64, amount float64) error {
	rows, err := tx.QueryContext(ctx,
		`UPDATE accounts SET extra = (
			COALESCE(extra, '{}'::jsonb)
			|| jsonb_build_object('quota_used', COALESCE((extra->>'quota_used')::numeric, 0) + $1)
			|| CASE WHEN COALESCE((extra->>'quota_daily_limit')::numeric, 0) > 0 THEN
				jsonb_build_object(
					'quota_daily_used',
					CASE WHEN COALESCE((extra->>'quota_daily_start')::timestamptz, '1970-01-01'::timestamptz)
						+ '24 hours'::interval <= NOW()
					THEN $1
					ELSE COALESCE((extra->>'quota_daily_used')::numeric, 0) + $1 END,
					'quota_daily_start',
					CASE WHEN COALESCE((extra->>'quota_daily_start')::timestamptz, '1970-01-01'::timestamptz)
						+ '24 hours'::interval <= NOW()
					THEN `+nowUTC+`
					ELSE COALESCE(extra->>'quota_daily_start', `+nowUTC+`) END
				)
			ELSE '{}'::jsonb END
			|| CASE WHEN COALESCE((extra->>'quota_weekly_limit')::numeric, 0) > 0 THEN
				jsonb_build_object(
					'quota_weekly_used',
					CASE WHEN COALESCE((extra->>'quota_weekly_start')::timestamptz, '1970-01-01'::timestamptz)
						+ '168 hours'::interval <= NOW()
					THEN $1
					ELSE COALESCE((extra->>'quota_weekly_used')::numeric, 0) + $1 END,
					'quota_weekly_start',
					CASE WHEN COALESCE((extra->>'quota_weekly_start')::timestamptz, '1970-01-01'::timestamptz)
						+ '168 hours'::interval <= NOW()
					THEN `+nowUTC+`
					ELSE COALESCE(extra->>'quota_weekly_start', `+nowUTC+`) END
				)
			ELSE '{}'::jsonb END
		), updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
		RETURNING
			COALESCE((extra->>'quota_used')::numeric, 0),
			COALESCE((extra->>'quota_limit')::numeric, 0)`,
		amount, accountID)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	var newUsed, limit float64
	if rows.Next() {
		if err := rows.Scan(&newUsed, &limit); err != nil {
			return err
		}
	} else {
		if err := rows.Err(); err != nil {
			return err
		}
		return service.ErrAccountNotFound
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if limit > 0 && newUsed >= limit && (newUsed-amount) < limit {
		if err := enqueueSchedulerOutbox(ctx, tx, service.SchedulerOutboxEventAccountChanged, &accountID, nil, nil); err != nil {
			logger.LegacyPrintf("repository.usage_billing", "[SchedulerOutbox] enqueue quota exceeded failed: account=%d err=%v", accountID, err)
			return err
		}
	}
	return nil
}
