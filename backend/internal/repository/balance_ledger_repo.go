package repository

import (
	"context"
	"database/sql"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type balanceLedgerRepository struct {
	client *dbent.Client
}

func NewBalanceLedgerRepository(client *dbent.Client) service.BalanceLedgerRepository {
	return &balanceLedgerRepository{client: client}
}

func (r *balanceLedgerRepository) Insert(ctx context.Context, entry service.BalanceLedgerEntry) error {
	const query = `
INSERT INTO account_balance_ledger (
    user_id, event_type, amount_delta, balance_before, balance_after,
    actor_type, actor_id, source_type, source_id, note
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := clientFromContext(ctx, r.client).ExecContext(ctx, query,
		entry.UserID,
		entry.EventType,
		entry.AmountDelta,
		entry.BalanceBefore,
		entry.BalanceAfter,
		entry.ActorType,
		entry.ActorID,
		entry.SourceType,
		entry.SourceID,
		entry.Note,
	)
	return err
}

func (r *balanceLedgerRepository) ListByUser(
	ctx context.Context,
	userID int64,
	offset int,
	limit int,
) (entries []service.BalanceLedgerEntry, total int64, err error) {
	client := clientFromContext(ctx, r.client)
	rows, err := client.QueryContext(ctx, `
SELECT id, user_id, event_type,
       amount_delta::double precision,
       balance_before::double precision,
       balance_after::double precision,
       actor_type, actor_id, source_type, source_id, note, created_at
FROM account_balance_ledger
WHERE user_id = $1
  AND event_type IN (
      'admin_credit',
      'admin_debit',
      'admin_set',
      'redeem_code',
      'admin_redeem'
  )
ORDER BY created_at DESC, id DESC
OFFSET $2
LIMIT $3`, userID, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	entries = make([]service.BalanceLedgerEntry, 0, limit)
	for rows.Next() {
		var entry service.BalanceLedgerEntry
		var actorID sql.NullInt64
		var sourceType sql.NullString
		var sourceID sql.NullString
		if err := rows.Scan(
			&entry.ID,
			&entry.UserID,
			&entry.EventType,
			&entry.AmountDelta,
			&entry.BalanceBefore,
			&entry.BalanceAfter,
			&entry.ActorType,
			&actorID,
			&sourceType,
			&sourceID,
			&entry.Note,
			&entry.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		if actorID.Valid {
			entry.ActorID = &actorID.Int64
		}
		if sourceType.Valid {
			entry.SourceType = &sourceType.String
		}
		if sourceID.Valid {
			entry.SourceID = &sourceID.String
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	countRows, err := client.QueryContext(ctx, `
SELECT COUNT(*)
FROM account_balance_ledger
WHERE user_id = $1
  AND event_type IN (
      'admin_credit',
      'admin_debit',
      'admin_set',
      'redeem_code',
      'admin_redeem'
  )`, userID)
	if err != nil {
		return nil, 0, err
	}
	defer func() {
		if closeErr := countRows.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()
	if countRows.Next() {
		if err := countRows.Scan(&total); err != nil {
			return nil, 0, err
		}
	}
	if err := countRows.Err(); err != nil {
		return nil, 0, err
	}
	return entries, total, nil
}
