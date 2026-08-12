package service

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const (
	BalanceLedgerEventAdminCredit = "admin_credit"
	BalanceLedgerEventAdminDebit  = "admin_debit"
	BalanceLedgerEventAdminSet    = "admin_set"
	BalanceLedgerEventRedeemCode  = "redeem_code"
	BalanceLedgerEventAdminRedeem = "admin_redeem"

	BalanceLedgerActorAdmin = "admin"
	BalanceLedgerActorUser  = "user"

	BalanceLedgerSourceRedeemCode = "redeem_code"
)

// BalanceLedgerRepository stores and reads immutable balance changes.
// Insert joins a transaction already attached to ctx and never starts its own.
type BalanceLedgerRepository interface {
	Insert(ctx context.Context, entry BalanceLedgerEntry) error
	ListByUser(ctx context.Context, userID int64, offset, limit int) ([]BalanceLedgerEntry, int64, error)
}

type RedeemCodeLedgerHistoryRepository interface {
	ListByUserPaginatedExcludingLedger(
		ctx context.Context,
		userID int64,
		params pagination.PaginationParams,
		codeType string,
	) ([]RedeemCode, *pagination.PaginationResult, error)
}

type BalanceLedgerEntry struct {
	ID            int64
	UserID        int64
	EventType     string
	AmountDelta   float64
	BalanceBefore float64
	BalanceAfter  float64
	ActorType     string
	ActorID       *int64
	SourceType    *string
	SourceID      *string
	Note          string
	CreatedAt     time.Time
}

type redeemActorContextKey struct{}

type redeemActor struct {
	eventType string
	actorType string
	actorID   *int64
}

// ContextWithAdminRedeemActor marks an admin-created redemption for audit.
func ContextWithAdminRedeemActor(ctx context.Context, adminUserID int64) context.Context {
	id := adminUserID
	return context.WithValue(ctx, redeemActorContextKey{}, redeemActor{
		eventType: BalanceLedgerEventAdminRedeem,
		actorType: BalanceLedgerActorAdmin,
		actorID:   &id,
	})
}

func redeemActorFromContext(ctx context.Context, userID int64) redeemActor {
	if actor, ok := ctx.Value(redeemActorContextKey{}).(redeemActor); ok {
		return actor
	}
	id := userID
	return redeemActor{
		eventType: BalanceLedgerEventRedeemCode,
		actorType: BalanceLedgerActorUser,
		actorID:   &id,
	}
}
