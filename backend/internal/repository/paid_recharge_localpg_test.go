//go:build localpg && !integration

package repository

import (
	"context"
	"fmt"
	"math"
	"sync"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	_ "github.com/Wei-Shaw/sub2api/ent/runtime"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestPaidRecharge_LocalPostgres(t *testing.T) {
	db := isolatedLocalPostgres(t)
	ctx := context.Background()
	client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	require.NoError(t, client.Schema.Create(ctx))
	// The balance ledger is migration-owned, not an Ent schema. This fixture is
	// local to the fresh cluster and retains the redemption source uniqueness.
	_, err := db.ExecContext(ctx, `CREATE TABLE account_balance_ledger (
		id BIGSERIAL PRIMARY KEY, user_id BIGINT NOT NULL, event_type TEXT NOT NULL,
		amount_delta NUMERIC(20,8) NOT NULL, balance_before NUMERIC(20,8) NOT NULL,
		balance_after NUMERIC(20,8) NOT NULL, actor_type TEXT NOT NULL, actor_id BIGINT,
		source_type TEXT, source_id TEXT, note TEXT, created_at TIMESTAMPTZ DEFAULT NOW(),
		UNIQUE(source_type, source_id))`)
	require.NoError(t, err)
	repo := newUserRepositoryWithSQL(client, db)
	u, err := client.User.Create().SetEmail("paid-recharge@example.com").SetPasswordHash("fixture").
		SetBalance(-10).SetTotalRecharged(30).Save(ctx)
	require.NoError(t, err)
	reset := func(t *testing.T) {
		t.Helper()
		_, err := client.User.UpdateOneID(u.ID).SetBalance(-10).SetTotalRecharged(30).Save(ctx)
		require.NoError(t, err)
	}
	check := func(t *testing.T, balance, total float64) {
		t.Helper()
		got, err := client.User.Get(ctx, u.ID)
		require.NoError(t, err)
		require.Equal(t, balance, got.Balance)
		require.Equal(t, total, got.TotalRecharged)
	}
	codes := NewRedeemCodeRepository(client)
	ledger := NewBalanceLedgerRepository(client)
	redeemer := service.NewRedeemService(codes, repo, nil, nil, nil, client, nil, nil, ledger)
	payments := service.NewPaymentService(client, nil, nil, redeemer, nil, nil, repo, nil, nil)
	orderFor := func(t *testing.T, name string) *dbent.PaymentOrder {
		t.Helper()
		o, err := client.PaymentOrder.Create().SetUserID(u.ID).SetUserEmail(u.Email).SetUserName("fixture").
			SetAmount(5).SetPayAmount(9).SetRechargeCode(name).SetOutTradeNo(name).
			SetPaymentType("fixture").SetPaymentTradeNo(name).SetStatus(service.OrderStatusPaid).
			SetPaidAt(time.Now()).SetExpiresAt(time.Now().Add(time.Hour)).SetClientIP("127.0.0.1").SetSrcHost("local.test").Save(ctx)
		require.NoError(t, err)
		return o
	}

	t.Run("generic_credit_is_not_recharge", func(t *testing.T) {
		reset(t)
		require.NoError(t, repo.UpdateBalance(ctx, u.ID, 5))
		got, err := client.User.Get(ctx, u.ID)
		require.NoError(t, err)
		require.Equal(t, -5.0, got.Balance)
		require.Equal(t, 30.0, got.TotalRecharged)
	})
	t.Run("paid_credit_reduces_debt_and_counts_gross_balance_units", func(t *testing.T) {
		reset(t)
		o := orderFor(t, "paid-once")
		require.NoError(t, payments.ExecuteBalanceFulfillment(ctx, o.ID))
		check(t, -5, 35) // Amount=5; PayAmount=9 must not become balance-unit total.
		require.NoError(t, payments.ExecuteBalanceFulfillment(ctx, o.ID))
		check(t, -5, 35)
		entries, total, err := ledger.ListByUser(ctx, u.ID, 0, 20)
		require.NoError(t, err)
		require.EqualValues(t, 1, total)
		require.Equal(t, 5.0, entries[0].AmountDelta)
		require.Equal(t, -10.0, entries[0].BalanceBefore)
		require.Equal(t, -5.0, entries[0].BalanceAfter)
		_, err = client.PaymentOrder.UpdateOneID(o.ID).SetStatus(service.OrderStatusRecharging).
			SetUpdatedAt(time.Now().Add(-24 * time.Hour)).Save(ctx)
		require.NoError(t, err)
		require.NoError(t, payments.ExecuteBalanceFulfillment(ctx, o.ID))
		check(t, -5, 35) // Recovery after committed redemption must not add again.
	})
	t.Run("paid_bonus_credits_balance_but_not_total_recharged", func(t *testing.T) {
		reset(t)
		o := orderFor(t, "paid-bonus")
		o, err = client.PaymentOrder.UpdateOneID(o.ID).SetBonusAmount(1).Save(ctx)
		require.NoError(t, err)
		require.NoError(t, payments.ExecuteBalanceFulfillment(ctx, o.ID))
		check(t, -5, 34) // Amount=5 includes bonus=1; PayAmount=9 is a different currency.
		require.NoError(t, payments.ExecuteBalanceFulfillment(ctx, o.ID))
		check(t, -5, 34)
		_, err = client.PaymentOrder.UpdateOneID(o.ID).SetStatus(service.OrderStatusRecharging).
			SetUpdatedAt(time.Now().Add(-24 * time.Hour)).Save(ctx)
		require.NoError(t, err)
		require.NoError(t, payments.ExecuteBalanceFulfillment(ctx, o.ID))
		check(t, -5, 34)
		entries, _, err := ledger.ListByUser(ctx, u.ID, 0, 20)
		require.NoError(t, err)
		code, err := codes.GetByCode(ctx, o.RechargeCode)
		require.NoError(t, err)
		var bonusEntries int
		for _, entry := range entries {
			if entry.SourceID != nil && *entry.SourceID == fmt.Sprint(code.ID) {
				bonusEntries++
				require.Equal(t, 5.0, entry.AmountDelta)
			}
		}
		require.Equal(t, 1, bonusEntries)
	})
	t.Run("ledger_failure_rolls_back_code_balance_and_total", func(t *testing.T) {
		reset(t)
		o := orderFor(t, "paid-rollback")
		o, err = client.PaymentOrder.UpdateOneID(o.ID).SetBonusAmount(1).Save(ctx)
		require.NoError(t, err)
		_, err := db.ExecContext(ctx, `ALTER TABLE account_balance_ledger ADD CONSTRAINT reject_fixture_credit CHECK (amount_delta <> 5) NOT VALID`)
		require.NoError(t, err)
		err = payments.ExecuteBalanceFulfillment(ctx, o.ID)
		require.ErrorContains(t, err, "insert balance ledger")
		check(t, -10, 30)
		code, err := codes.GetByCode(ctx, o.RechargeCode)
		require.NoError(t, err)
		require.Equal(t, service.StatusUnused, code.Status)
		_, err = db.ExecContext(ctx, `ALTER TABLE account_balance_ledger DROP CONSTRAINT reject_fixture_credit`)
		require.NoError(t, err)
		require.NoError(t, payments.ExecuteBalanceFulfillment(ctx, o.ID))
		check(t, -5, 34)
	})
	t.Run("concurrent_fulfillment_counts_once", func(t *testing.T) {
		reset(t)
		o := orderFor(t, "paid-concurrent")
		o, err = client.PaymentOrder.UpdateOneID(o.ID).SetBonusAmount(1).Save(ctx)
		require.NoError(t, err)
		var wg sync.WaitGroup
		results := make(chan error, 2)
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func() { defer wg.Done(); results <- payments.ExecuteBalanceFulfillment(ctx, o.ID) }()
		}
		wg.Wait()
		close(results)
		succeeded := false
		for err := range results {
			if err == nil {
				succeeded = true
			}
		}
		require.True(t, succeeded)
		check(t, -5, 34)
	})
	for _, admin := range []bool{false, true} {
		t.Run(fmt.Sprintf("unverified_redeem_admin_%t", admin), func(t *testing.T) {
			reset(t)
			code := &service.RedeemCode{Code: fmt.Sprintf("free-%t", admin), Type: service.RedeemTypeBalance, Value: 5, Status: service.StatusUnused}
			require.NoError(t, codes.Create(ctx, code))
			if admin {
				_, err = redeemer.RedeemForAdminFulfillment(ctx, u.ID, code.Code)
			} else {
				_, err = redeemer.Redeem(ctx, u.ID, code.Code)
			}
			require.NoError(t, err)
			check(t, -5, 30)
		})
	}
	t.Run("generic_adjustment_and_refund_restoration_do_not_count", func(t *testing.T) {
		reset(t)
		_, err := repo.AdjustBalance(ctx, u.ID, 5)
		require.NoError(t, err)
		check(t, -5, 30)
		plan := &service.RefundPlan{Order: &dbent.PaymentOrder{UserID: u.ID}, DeductionType: payment.DeductionTypeBalance, BalanceToDeduct: 5}
		require.True(t, payments.RollbackRefund(ctx, plan, nil))
		check(t, 0, 30)
	})
	for _, value := range []float64{0, -1, math.NaN(), math.Inf(1), math.Inf(-1)} {
		t.Run(fmt.Sprintf("invalid_paid_amount_%v", value), func(t *testing.T) {
			reset(t)
			_, err := repo.CreditPaidRecharge(ctx, u.ID, value, value)
			require.ErrorIs(t, err, service.ErrInvalidPaidRechargeAmount)
			check(t, -10, 30)
		})
	}
	t.Run("missing_user", func(t *testing.T) {
		_, err := repo.CreditPaidRecharge(ctx, u.ID+99999, 5, 5)
		require.ErrorIs(t, err, service.ErrUserNotFound)
	})
	t.Run("promo_gift_does_not_count", func(t *testing.T) {
		reset(t)
		_, err := client.PromoCode.Create().SetCode("gift-only").SetBonusAmount(5).Save(ctx)
		require.NoError(t, err)
		promo := service.NewPromoService(NewPromoCodeRepository(client), repo, nil, client, nil)
		require.NoError(t, promo.ApplyPromoCode(ctx, u.ID, "gift-only"))
		check(t, -5, 30)
		require.ErrorIs(t, promo.ApplyPromoCode(ctx, u.ID, "gift-only"), service.ErrPromoCodeAlreadyUsed)
		check(t, -5, 30)
	})
	t.Run("internal_conversions_preserve_commissions_without_counting_recharge", func(t *testing.T) {
		// These SSXZ tables are migration-owned. Only columns used by the actual
		// transfer methods are needed; no production migration/backfill is run.
		_, err := db.ExecContext(ctx, `
		CREATE TABLE user_affiliates (
		 user_id BIGINT PRIMARY KEY, aff_code TEXT, aff_code_custom BOOLEAN DEFAULT FALSE,
		 aff_rebate_rate_percent NUMERIC, inviter_id BIGINT, aff_count INT DEFAULT 0,
		 aff_quota NUMERIC NOT NULL, aff_frozen_quota NUMERIC NOT NULL, aff_history_quota NUMERIC NOT NULL,
		 created_at TIMESTAMPTZ DEFAULT NOW(), updated_at TIMESTAMPTZ DEFAULT NOW());
		CREATE TABLE user_reseller_roles (user_id BIGINT PRIMARY KEY, status TEXT, revoked_at TIMESTAMPTZ);
		CREATE TABLE user_affiliate_ledger (
		 id BIGSERIAL PRIMARY KEY, user_id BIGINT, action TEXT, amount NUMERIC, source_user_id BIGINT,
		 balance_after NUMERIC, aff_quota_after NUMERIC, aff_frozen_quota_after NUMERIC, aff_history_quota_after NUMERIC,
		 notes TEXT, frozen_until TIMESTAMPTZ, created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ);
		CREATE TABLE affiliate_withdraw_requests (
		 id BIGINT PRIMARY KEY, user_id BIGINT, amount NUMERIC, status TEXT, note TEXT,
		 reviewed_at TIMESTAMPTZ, reviewed_by BIGINT, updated_at TIMESTAMPTZ)`)
		require.NoError(t, err)
		for _, reseller := range []bool{false, true} {
			t.Run(fmt.Sprintf("reseller_%t", reseller), func(t *testing.T) {
				reset(t)
				_, err := db.ExecContext(ctx, `INSERT INTO user_affiliates (user_id,aff_code,aff_quota,aff_frozen_quota,aff_history_quota)
				 VALUES ($1,'fixture-aff',5,9,100) ON CONFLICT (user_id) DO UPDATE SET aff_quota=5`, u.ID)
				require.NoError(t, err)
				if reseller {
					_, err = db.ExecContext(ctx, `INSERT INTO affiliate_withdraw_requests (id,user_id,amount,status) VALUES (1,$1,5,'pending')`, u.ID)
					require.NoError(t, err)
					r := NewResellerRepository(client, db)
					require.NoError(t, r.ReviewWithdrawRequest(ctx, 1, u.ID, service.WithdrawStatusApproved, "fixture"))
					require.ErrorIs(t, r.ReviewWithdrawRequest(ctx, 1, u.ID, service.WithdrawStatusApproved, "fixture"), service.ErrWithdrawAlreadyReviewed)
				} else {
					r := NewAffiliateRepository(client, db)
					transferred, balance, err := r.TransferQuotaToBalance(ctx, u.ID)
					require.NoError(t, err)
					require.Equal(t, 5.0, transferred)
					require.Equal(t, -5.0, balance)
					_, _, err = r.TransferQuotaToBalance(ctx, u.ID)
					require.ErrorIs(t, err, service.ErrAffiliateQuotaEmpty)
				}
				check(t, -5, 30)
				var quota, frozen, history float64
				require.NoError(t, db.QueryRowContext(ctx, `SELECT aff_quota,aff_frozen_quota,aff_history_quota FROM user_affiliates WHERE user_id=$1`, u.ID).Scan(&quota, &frozen, &history))
				require.Equal(t, 0.0, quota)
				require.Equal(t, 9.0, frozen)
				require.Equal(t, 100.0, history)
			})
		}
		var count int
		require.NoError(t, db.QueryRowContext(ctx, `SELECT count(*) FROM user_affiliate_ledger WHERE user_id=$1 AND action='transfer' AND amount=5 AND balance_after=-5 AND aff_quota_after=0 AND aff_frozen_quota_after=9 AND aff_history_quota_after=100`, u.ID).Scan(&count))
		require.Equal(t, 2, count)
	})
	t.Run("deleted_user", func(t *testing.T) {
		_, err := client.User.UpdateOneID(u.ID).SetDeletedAt(time.Now()).Save(ctx)
		require.NoError(t, err)
		_, err = repo.CreditPaidRecharge(ctx, u.ID, 5, 5)
		require.ErrorIs(t, err, service.ErrUserNotFound)
	})
}
