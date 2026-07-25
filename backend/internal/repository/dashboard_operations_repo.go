package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type dashboardOperationsRepository struct {
	db *sql.DB
}

func NewDashboardOperationsRepository(db *sql.DB) service.DashboardOperationsRepository {
	return &dashboardOperationsRepository{db: db}
}

const dashboardOperationsSummarySQL = `
WITH new_customers AS (
    SELECT COUNT(*)::bigint AS count
    FROM users
    WHERE role = 'user'
      AND deleted_at IS NULL
      AND created_at >= $1
      AND created_at < $2
), usage_summary AS (
    SELECT COALESCE(SUM(ul.actual_cost), 0)::double precision AS actual_cost,
           COUNT(DISTINCT ul.user_id)::bigint AS active_customers,
           COUNT(DISTINCT ul.api_key_id)::bigint AS active_api_keys
    FROM usage_logs ul
    JOIN users u ON u.id = ul.user_id
    WHERE u.role = 'user'
      AND u.deleted_at IS NULL
      AND ul.created_at >= $1
      AND ul.created_at < $2
), invitee_recharges AS (
    SELECT COALESCE(SUM(po.amount), 0)::double precision AS amount
    FROM payment_orders po
    JOIN user_affiliates invitee ON invitee.user_id = po.user_id
    WHERE invitee.inviter_id IS NOT NULL
      AND po.order_type = 'balance'
      AND po.status = 'COMPLETED'
      AND COALESCE(po.completed_at, po.paid_at, po.updated_at) >= $1
      AND COALESCE(po.completed_at, po.paid_at, po.updated_at) < $2
), rebate_events AS (
    SELECT COALESCE(SUM(CASE
               WHEN action = 'accrue'
                AND frozen_until IS NOT NULL
                AND frozen_until > NOW()
                AND created_at >= $1 AND created_at < $2
               THEN amount ELSE 0 END), 0)::double precision AS pending,
           COALESCE(SUM(CASE
               WHEN action = 'accrue'
                AND frozen_until IS NULL
                AND updated_at >= $1 AND updated_at < $2
               THEN amount ELSE 0 END), 0)::double precision AS available,
           COALESCE(SUM(CASE
               WHEN action = 'transfer'
                AND created_at >= $1 AND created_at < $2
               THEN amount ELSE 0 END), 0)::double precision AS transferred
    FROM user_affiliate_ledger
)
SELECT nc.count,
       us.actual_cost,
       ir.amount,
       re.pending,
       re.available,
       re.transferred,
       us.active_customers,
       us.active_api_keys
FROM new_customers nc
CROSS JOIN usage_summary us
CROSS JOIN invitee_recharges ir
CROSS JOIN rebate_events re`

const dashboardOperationsTopCustomersSQL = `
SELECT ul.user_id,
       COALESCE(u.email, ''),
       COALESCE(u.username, ''),
       COALESCE(SUM(ul.actual_cost), 0)::double precision AS actual_cost,
       COUNT(*)::bigint AS requests,
       COUNT(DISTINCT ul.api_key_id)::bigint AS active_keys
FROM usage_logs ul
JOIN users u ON u.id = ul.user_id
WHERE u.role = 'user'
  AND u.deleted_at IS NULL
  AND ul.created_at >= $1
  AND ul.created_at < $2
GROUP BY ul.user_id, u.email, u.username
ORDER BY actual_cost DESC, ul.user_id ASC
LIMIT $3`

func (r *dashboardOperationsRepository) GetOperationsSummary(ctx context.Context, startTime, endTime time.Time, topLimit int) (*service.DashboardOperationsSummary, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("nil dashboard operations repository")
	}

	summary := &service.DashboardOperationsSummary{
		StartDate:    startTime,
		EndDate:      endTime,
		TopCustomers: make([]service.DashboardOperationsTopCustomer, 0),
	}
	if err := r.db.QueryRowContext(ctx, dashboardOperationsSummarySQL, startTime, endTime).Scan(
		&summary.NewCustomers,
		&summary.CustomerActualCost,
		&summary.InviteeRechargeAmount,
		&summary.RebatePending,
		&summary.RebateAvailable,
		&summary.RebateTransferred,
		&summary.ActiveCustomers,
		&summary.ActiveAPIKeys,
	); err != nil {
		return nil, fmt.Errorf("query dashboard operations summary: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, dashboardOperationsTopCustomersSQL, startTime, endTime, topLimit)
	if err != nil {
		return nil, fmt.Errorf("query dashboard operations top customers: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var item service.DashboardOperationsTopCustomer
		if err := rows.Scan(
			&item.UserID,
			&item.Email,
			&item.Username,
			&item.ActualCost,
			&item.Requests,
			&item.ActiveKeys,
		); err != nil {
			return nil, fmt.Errorf("scan dashboard operations top customer: %w", err)
		}
		summary.TopCustomers = append(summary.TopCustomers, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate dashboard operations top customers: %w", err)
	}
	return summary, nil
}
