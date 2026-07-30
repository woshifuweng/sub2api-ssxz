package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

func TestMergeBalanceHistoryCodesIncludesAffiliateTransfersByDefault(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	older := now.Add(-2 * time.Hour)
	newer := now.Add(time.Hour)

	usedBy := int64(10)
	redeemCodes := []RedeemCode{
		{
			ID:        1,
			Type:      RedeemTypeBalance,
			Value:     8,
			Status:    StatusUsed,
			UsedBy:    &usedBy,
			UsedAt:    &now,
			CreatedAt: now,
		},
		{
			ID:        2,
			Type:      RedeemTypeConcurrency,
			Value:     1,
			Status:    StatusUsed,
			UsedBy:    &usedBy,
			UsedAt:    &older,
			CreatedAt: older,
		},
	}
	affiliateCodes := []RedeemCode{
		{
			ID:        -20,
			Type:      RedeemTypeAffiliateBalance,
			Value:     3.5,
			Status:    StatusUsed,
			UsedBy:    &usedBy,
			UsedAt:    &newer,
			CreatedAt: newer,
		},
	}

	got := mergeBalanceHistoryCodes(redeemCodes, affiliateCodes, pagination.PaginationParams{
		Page:     1,
		PageSize: 2,
	})

	require.Len(t, got, 2)
	require.Equal(t, RedeemTypeAffiliateBalance, got[0].Type)
	require.Equal(t, RedeemTypeBalance, got[1].Type)
}

func TestMergeBalanceHistoryCodesPaginatesAfterCombiningSources(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	usedBy := int64(10)
	at := func(hours int) *time.Time {
		v := base.Add(time.Duration(hours) * time.Hour)
		return &v
	}

	got := mergeBalanceHistoryCodes(
		[]RedeemCode{
			{ID: 1, Type: RedeemTypeBalance, UsedBy: &usedBy, UsedAt: at(4), CreatedAt: *at(4)},
			{ID: 2, Type: RedeemTypeConcurrency, UsedBy: &usedBy, UsedAt: at(2), CreatedAt: *at(2)},
		},
		[]RedeemCode{
			{ID: -3, Type: RedeemTypeAffiliateBalance, UsedBy: &usedBy, UsedAt: at(3), CreatedAt: *at(3)},
			{ID: -4, Type: RedeemTypeAffiliateBalance, UsedBy: &usedBy, UsedAt: at(1), CreatedAt: *at(1)},
		},
		pagination.PaginationParams{Page: 2, PageSize: 2},
	)

	require.Len(t, got, 2)
	require.Equal(t, RedeemTypeConcurrency, got[0].Type)
	require.Equal(t, int64(-4), got[1].ID)
}

func TestMergeBalanceHistoryCodesIncludesLedgerEntries(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	usedBy := int64(10)
	redeemAt := base.Add(-time.Hour)
	affiliateAt := base
	ledgerAt := base.Add(time.Hour)

	got := mergeBalanceHistoryCodes(
		[]RedeemCode{{ID: 1, Type: RedeemTypeBalance, UsedBy: &usedBy, UsedAt: &redeemAt, CreatedAt: redeemAt}},
		[]RedeemCode{{ID: -2, Type: RedeemTypeAffiliateBalance, UsedBy: &usedBy, UsedAt: &affiliateAt, CreatedAt: affiliateAt}},
		pagination.PaginationParams{Page: 1, PageSize: 3},
		[]RedeemCode{{ID: -3, Type: AdjustmentTypeAdminBalance, UsedBy: &usedBy, UsedAt: &ledgerAt, CreatedAt: ledgerAt}},
	)

	require.Len(t, got, 3)
	require.Equal(t, AdjustmentTypeAdminBalance, got[0].Type)
	require.Equal(t, RedeemTypeAffiliateBalance, got[1].Type)
	require.Equal(t, RedeemTypeBalance, got[2].Type)
}

func TestBalanceLedgerEntryToHistoryMapsSupportedEvents(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		eventType string
		wantType  string
	}{
		{eventType: BalanceLedgerEventAdminCredit, wantType: AdjustmentTypeAdminBalance},
		{eventType: BalanceLedgerEventAdminDebit, wantType: AdjustmentTypeAdminBalance},
		{eventType: BalanceLedgerEventAdminSet, wantType: AdjustmentTypeAdminBalance},
		{eventType: BalanceLedgerEventRedeemCode, wantType: RedeemTypeBalance},
		{eventType: BalanceLedgerEventAdminRedeem, wantType: RedeemTypeBalance},
	}

	for _, tt := range tests {
		t.Run(tt.eventType, func(t *testing.T) {
			got, ok := balanceLedgerEntryToHistory(BalanceLedgerEntry{
				ID:          9,
				UserID:      10,
				EventType:   tt.eventType,
				AmountDelta: 2.5,
				Note:        "test",
				CreatedAt:   createdAt,
			})

			require.True(t, ok)
			require.Equal(t, tt.wantType, got.Type)
			require.Equal(t, 2.5, got.Value)
			require.Equal(t, "test", got.Notes)
			require.Equal(t, createdAt, *got.UsedAt)
		})
	}
}

func TestBalanceLedgerEntryToHistoryRejectsUnsupportedEvents(t *testing.T) {
	t.Parallel()

	_, ok := balanceLedgerEntryToHistory(BalanceLedgerEntry{EventType: "payment"})
	require.False(t, ok)
}

func TestRedeemActorFromContext(t *testing.T) {
	t.Parallel()

	t.Run("user redemption", func(t *testing.T) {
		actor := redeemActorFromContext(context.Background(), 42)
		require.Equal(t, BalanceLedgerEventRedeemCode, actor.eventType)
		require.Equal(t, BalanceLedgerActorUser, actor.actorType)
		require.NotNil(t, actor.actorID)
		require.Equal(t, int64(42), *actor.actorID)
	})

	t.Run("admin redemption", func(t *testing.T) {
		ctx := ContextWithAdminRedeemActor(context.Background(), 7)
		actor := redeemActorFromContext(ctx, 42)
		require.Equal(t, BalanceLedgerEventAdminRedeem, actor.eventType)
		require.Equal(t, BalanceLedgerActorAdmin, actor.actorType)
		require.NotNil(t, actor.actorID)
		require.Equal(t, int64(7), *actor.actorID)
	})
}
