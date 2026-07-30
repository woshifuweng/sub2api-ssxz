// Package repository — resellerRepository implements service.ResellerRepository
// using raw SQL against PostgreSQL (via the ent client's underlying *sql.DB).
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type resellerRepository struct {
	db *sql.DB
}

// NewResellerRepository constructs the repository. All queries run directly
// against the underlying *sql.DB; the ent client is not needed here.
func NewResellerRepository(_ *dbent.Client, db *sql.DB) service.ResellerRepository {
	return &resellerRepository{db: db}
}

// --- GrantRole ---

func (r *resellerRepository) GrantRole(ctx context.Context, userID int64, role string, grantedBy int64, notes string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO user_reseller_roles (user_id, role, granted_by, granted_at, notes)
		 VALUES ($1, $2, $3, NOW(), $4)
		 ON CONFLICT (user_id) DO UPDATE SET
		     role       = EXCLUDED.role,
		     granted_by = EXCLUDED.granted_by,
		     granted_at = NOW(),
		     revoked_at = NULL,
		     notes      = EXCLUDED.notes`,
		userID, role, grantedBy, notes,
	)
	if err != nil {
		return fmt.Errorf("reseller repo GrantRole: %w", err)
	}
	return nil
}

func (r *resellerRepository) GrantManagedAgent(ctx context.Context, userID, managerID int64, notes string) error {
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO user_reseller_roles (user_id, role, granted_by, granted_at, notes)
		VALUES ($1, 'agent', $2, NOW(), $3)
		ON CONFLICT (user_id) DO UPDATE SET
			role = 'agent',
			granted_by = EXCLUDED.granted_by,
			granted_at = NOW(),
			revoked_at = NULL,
			notes = EXCLUDED.notes
		WHERE user_reseller_roles.revoked_at IS NOT NULL
		   OR (user_reseller_roles.role = 'agent' AND user_reseller_roles.granted_by = $2)`,
		userID, managerID, notes,
	)
	if err != nil {
		return fmt.Errorf("reseller repo GrantManagedAgent: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("reseller repo GrantManagedAgent rows affected: %w", err)
	}
	if affected == 0 {
		return service.ErrResellerAgentNotManaged
	}
	return nil
}

// --- RevokeRole ---

func (r *resellerRepository) RevokeRole(ctx context.Context, userID int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE user_reseller_roles SET revoked_at = NOW()
		  WHERE user_id = $1 AND revoked_at IS NULL`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("reseller repo RevokeRole: %w", err)
	}
	return nil
}

func (r *resellerRepository) RevokeManagedAgent(ctx context.Context, userID, managerID int64) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE user_reseller_roles
		SET revoked_at = NOW()
		WHERE user_id = $1
		  AND role = 'agent'
		  AND granted_by = $2
		  AND revoked_at IS NULL`,
		userID, managerID,
	)
	if err != nil {
		return fmt.Errorf("reseller repo RevokeManagedAgent: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("reseller repo RevokeManagedAgent rows affected: %w", err)
	}
	if affected == 0 {
		return service.ErrResellerAgentNotManaged
	}
	return nil
}

// --- GetManagerDashboard ---

func (r *resellerRepository) GetManagerDashboard(ctx context.Context, managerID int64) (*service.ManagerDashboard, error) {
	var d service.ManagerDashboard
	err := r.db.QueryRowContext(ctx, `
		SELECT
		    (SELECT COUNT(*) FROM user_reseller_roles
		      WHERE role = 'agent' AND granted_by = $1 AND revoked_at IS NULL)::int,
		    (SELECT COUNT(DISTINCT ua.user_id)
		       FROM user_affiliates ua
		       JOIN user_reseller_roles rr
		         ON rr.user_id = ua.inviter_id AND rr.role = 'agent'
		        AND rr.granted_by = $1 AND rr.revoked_at IS NULL)::int,
		    (SELECT COUNT(*)
		       FROM affiliate_withdraw_requests wr
		       JOIN user_reseller_roles rr ON rr.user_id = wr.user_id
		      WHERE wr.status = 'pending' AND rr.role = 'agent'
		        AND rr.granted_by = $1 AND rr.revoked_at IS NULL)::int
	`, managerID).Scan(&d.TotalAgents, &d.TotalRecruits, &d.PendingWithdrawals)
	if err != nil {
		return nil, fmt.Errorf("reseller repo GetManagerDashboard: %w", err)
	}
	return &d, nil
}

// --- GetRole ---

func (r *resellerRepository) GetRole(ctx context.Context, userID int64) (*service.ResellerRoleRecord, error) {
	var rec service.ResellerRoleRecord
	var grantedBy sql.NullInt64
	var revokedAt sql.NullTime
	err := r.db.QueryRowContext(ctx,
		`SELECT user_id, role, granted_by, granted_at, revoked_at, notes,
		        commission_rate::double precision
		   FROM user_reseller_roles
		  WHERE user_id = $1 AND revoked_at IS NULL
		  LIMIT 1`,
		userID,
	).Scan(
		&rec.UserID, &rec.Role, &grantedBy, &rec.GrantedAt, &revokedAt, &rec.Notes,
		&rec.CommissionRate,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrResellerRoleNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("reseller repo GetRole: %w", err)
	}
	if grantedBy.Valid {
		rec.GrantedBy = &grantedBy.Int64
	}
	if revokedAt.Valid {
		rec.RevokedAt = &revokedAt.Time
	}
	return &rec, nil
}

// --- ListAgents ---

func (r *resellerRepository) ListAgents(ctx context.Context, filter service.AgentFilter) ([]service.AgentSummary, int64, error) {
	page, pageSize := filter.Page, filter.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var total int64
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM user_reseller_roles rr
		  JOIN users u ON u.id = rr.user_id
		 WHERE rr.revoked_at IS NULL
		   AND (rr.role = 'agent' OR ($3 AND rr.role = 'agent_manager'))
		   AND ($1 = '' OR u.email ILIKE '%' || $1 || '%')
		   AND ($2 = 0 OR rr.granted_by = $2)`,
		filter.Search, filter.ManagerID, filter.IncludeAllRoles,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("reseller ListAgents count: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT rr.user_id, u.email, COALESCE(u.username, ''),
		       rr.role, rr.commission_rate::double precision,
		       COALESCE(ua.aff_code, ''), COALESCE(ua.aff_count, 0)::int,
		       COALESCE(ua.aff_quota, 0)::double precision,
		       rr.granted_at, rr.granted_by
		  FROM user_reseller_roles rr
		  JOIN users u ON u.id = rr.user_id
		  LEFT JOIN user_affiliates ua ON ua.user_id = rr.user_id
		 WHERE rr.revoked_at IS NULL
		   AND (rr.role = 'agent' OR ($3 AND rr.role = 'agent_manager'))
		   AND ($1 = '' OR u.email ILIKE '%' || $1 || '%')
		   AND ($2 = 0 OR rr.granted_by = $2)
		 ORDER BY rr.granted_at DESC
		 LIMIT $4 OFFSET $5`,
		filter.Search, filter.ManagerID, filter.IncludeAllRoles, pageSize, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("reseller ListAgents query: %w", err)
	}
	defer rows.Close()

	var out []service.AgentSummary
	for rows.Next() {
		var s service.AgentSummary
		var grantedBy sql.NullInt64
		if err := rows.Scan(
			&s.UserID, &s.Email, &s.Username,
			&s.Role, &s.CommissionRate,
			&s.AffCode, &s.RecruitCount, &s.AffQuota,
			&s.GrantedAt, &grantedBy,
		); err != nil {
			return nil, 0, fmt.Errorf("reseller ListAgents scan: %w", err)
		}
		if grantedBy.Valid {
			s.GrantedBy = &grantedBy.Int64
		}
		out = append(out, s)
	}
	return out, total, rows.Err()
}

// --- GetAgentDetail ---

func (r *resellerRepository) GetAgentDetail(ctx context.Context, agentUserID, managerID int64) (*service.AgentDetail, error) {
	var d service.AgentDetail
	var grantedBy sql.NullInt64
	err := r.db.QueryRowContext(ctx, `
		SELECT rr.user_id, u.email, COALESCE(u.username, ''),
		       rr.role, rr.commission_rate::double precision,
		       COALESCE(ua.aff_code, ''), COALESCE(ua.aff_count, 0)::int,
		       COALESCE(ua.aff_quota, 0)::double precision,
		       rr.granted_at, rr.granted_by
		  FROM user_reseller_roles rr
		  JOIN users u ON u.id = rr.user_id
		  LEFT JOIN user_affiliates ua ON ua.user_id = rr.user_id
		 WHERE rr.user_id = $1 AND rr.role = 'agent' AND rr.revoked_at IS NULL
		   AND ($2 = 0 OR rr.granted_by = $2)`,
		agentUserID, managerID,
	).Scan(
		&d.UserID, &d.Email, &d.Username,
		&d.Role, &d.CommissionRate,
		&d.AffCode, &d.RecruitCount, &d.AffQuota,
		&d.GrantedAt, &grantedBy,
	)
	if errors.Is(err, sql.ErrNoRows) {
		if managerID > 0 {
			return nil, service.ErrResellerAgentNotManaged
		}
		return nil, service.ErrResellerRoleNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("reseller GetAgentDetail: %w", err)
	}
	if grantedBy.Valid {
		d.GrantedBy = &grantedBy.Int64
	}
	recruits, _, err := r.ListMyRecruits(ctx, agentUserID, 1, 500, false)
	if err != nil {
		return nil, err
	}
	d.Recruits = recruits
	return &d, nil
}

// --- GetAgentDashboard ---

func (r *resellerRepository) GetAgentDashboard(ctx context.Context, agentUserID int64) (*service.AgentDashboard, error) {
	d := &service.AgentDashboard{UserID: agentUserID}
	err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(aff_code, ''),
		       COALESCE(aff_quota, 0)::double precision,
		       COALESCE(aff_frozen_quota, 0)::double precision,
		       COALESCE(aff_history_quota, 0)::double precision,
		       COALESCE(aff_count, 0)::int,
		       COALESCE(aff_rebate_rate_percent, 0)::double precision
		  FROM user_affiliates
		 WHERE user_id = $1`,
		agentUserID,
	).Scan(&d.AffCode, &d.AffQuota, &d.AffFrozenQuota, &d.AffHistoryQuota, &d.RecruitCount, &d.RebateRate)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("reseller GetAgentDashboard: %w", err)
	}
	var approvedWithdraw float64
	if err := r.db.QueryRowContext(ctx, `
		SELECT
		       COALESCE(SUM(amount) FILTER (WHERE status = 'pending'), 0)::double precision,
		       COALESCE(SUM(amount) FILTER (WHERE status = 'approved'), 0)::double precision
		  FROM affiliate_withdraw_requests
		 WHERE user_id = $1`,
		agentUserID,
	).Scan(&d.PendingWithdraw, &approvedWithdraw); err != nil {
		return nil, fmt.Errorf("reseller GetAgentDashboard withdrawal totals: %w", err)
	}
	d.CommissionEarned = d.AffQuota + approvedWithdraw
	return d, nil
}

// --- ListMyRecruits ---

func (r *resellerRepository) ListMyRecruits(ctx context.Context, agentUserID int64, page, pageSize int, maskEmail bool) ([]service.RecruitRecord, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var total int64
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM user_affiliates WHERE inviter_id = $1`,
		agentUserID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("reseller ListMyRecruits count: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT ua.user_id, u.email, COALESCE(u.username, ''),
		       u.created_at,
		       COALESCE((
		           SELECT SUM(ual.amount)
		           FROM user_affiliate_ledger ual
		           WHERE ual.user_id = $1
		             AND ual.source_user_id = ua.user_id
		             AND ual.action = 'accrue'
		       ), 0)::double precision,
		       EXISTS (
		           SELECT 1 FROM usage_logs ul
		           WHERE ul.user_id = ua.user_id
		             AND ul.created_at >= NOW() - INTERVAL '30 days'
		       )
		  FROM user_affiliates ua
		  JOIN users u ON u.id = ua.user_id
		 WHERE ua.inviter_id = $1
		 ORDER BY ua.created_at DESC
		 LIMIT $2 OFFSET $3`,
		agentUserID, pageSize, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("reseller ListMyRecruits query: %w", err)
	}
	defer rows.Close()

	var out []service.RecruitRecord
	for rows.Next() {
		var rec service.RecruitRecord
		var joinedAt sql.NullTime
		var email string
		if err := rows.Scan(
			&rec.UserID, &email, &rec.Username,
			&joinedAt, &rec.TotalRebate, &rec.IsActive,
		); err != nil {
			return nil, 0, fmt.Errorf("reseller ListMyRecruits scan: %w", err)
		}
		if maskEmail {
			rec.Email = service.MaskEmail(email)
		} else {
			rec.Email = email
		}
		if joinedAt.Valid {
			rec.JoinedAt = &joinedAt.Time
		}
		out = append(out, rec)
	}
	return out, total, rows.Err()
}

// --- CreateWithdrawRequest ---

func (r *resellerRepository) CreateWithdrawRequest(ctx context.Context, userID int64, input service.WithdrawInput) (*service.WithdrawRequest, error) {
	accountJSON, err := json.Marshal(input.AccountInfo)
	if err != nil {
		return nil, fmt.Errorf("reseller CreateWithdrawRequest marshal: %w", err)
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("reseller CreateWithdrawRequest begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var availableQuota float64
	if err := tx.QueryRowContext(ctx, `
		SELECT aff_quota::double precision
		FROM user_affiliates
		WHERE user_id = $1
		FOR UPDATE`, userID).Scan(&availableQuota); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrWithdrawInsufficientBalance
		}
		return nil, fmt.Errorf("reseller CreateWithdrawRequest lock quota: %w", err)
	}

	var pendingAmount float64
	if err := tx.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(amount), 0)::double precision
		FROM affiliate_withdraw_requests
		WHERE user_id = $1 AND status = 'pending'`, userID).Scan(&pendingAmount); err != nil {
		return nil, fmt.Errorf("reseller CreateWithdrawRequest pending amount: %w", err)
	}
	if availableQuota-pendingAmount < input.Amount {
		return nil, service.ErrWithdrawInsufficientBalance
	}

	var req service.WithdrawRequest
	var accountRaw []byte
	err = tx.QueryRowContext(ctx, `
		INSERT INTO affiliate_withdraw_requests (user_id, amount, method, account_info, status)
		VALUES ($1, $2, $3, $4, 'pending')
		RETURNING id, user_id, amount, method, account_info, status, note, requested_at`,
		userID, input.Amount, input.Method, accountJSON,
	).Scan(&req.ID, &req.UserID, &req.Amount, &req.Method, &accountRaw, &req.Status, &req.Note, &req.RequestedAt)
	if err != nil {
		return nil, fmt.Errorf("reseller CreateWithdrawRequest: %w", err)
	}
	if len(accountRaw) > 0 {
		_ = json.Unmarshal(accountRaw, &req.AccountInfo)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("reseller CreateWithdrawRequest commit: %w", err)
	}
	return &req, nil
}

// --- CancelWithdrawRequest ---

func (r *resellerRepository) CancelWithdrawRequest(ctx context.Context, withdrawalID, userID int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("reseller CancelWithdrawRequest begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var ownerID int64
	var currentStatus string
	if err := tx.QueryRowContext(ctx, `
		SELECT user_id, status
		FROM affiliate_withdraw_requests
		WHERE id = $1
		FOR UPDATE`, withdrawalID).Scan(&ownerID, &currentStatus); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return service.ErrWithdrawRequestNotFound
		}
		return fmt.Errorf("reseller CancelWithdrawRequest lock request: %w", err)
	}
	if ownerID != userID {
		return service.ErrWithdrawNotOwner
	}
	if currentStatus != service.WithdrawStatusPending {
		return service.ErrWithdrawNotPending.WithMetadata(map[string]string{
			"current_status": currentStatus,
		})
	}

	result, err := tx.ExecContext(ctx, `
		UPDATE affiliate_withdraw_requests
		SET status = 'cancelled', updated_at = NOW()
		WHERE id = $1 AND user_id = $2 AND status = 'pending'`,
		withdrawalID, userID,
	)
	if err != nil {
		return fmt.Errorf("reseller CancelWithdrawRequest update: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("reseller CancelWithdrawRequest rows affected: %w", err)
	}
	if affected != 1 {
		return service.ErrWithdrawNotPending
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("reseller CancelWithdrawRequest commit: %w", err)
	}
	return nil
}

// --- ListWithdrawRequests ---

func (r *resellerRepository) ListWithdrawRequests(ctx context.Context, filter service.WithdrawFilter) ([]service.WithdrawRequest, int64, error) {
	page, pageSize := filter.Page, filter.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var total int64
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM affiliate_withdraw_requests
		 WHERE ($1 = 0 OR user_id = $1)
		   AND ($2 = '' OR status = $2)
		   AND ($3 = 0 OR EXISTS (
		       SELECT 1 FROM user_reseller_roles rr
		       WHERE rr.user_id = affiliate_withdraw_requests.user_id
		         AND rr.role = 'agent' AND rr.granted_by = $3 AND rr.revoked_at IS NULL
		   ))`,
		filter.UserID, filter.Status, filter.ManagerID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("reseller ListWithdrawRequests count: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT wr.id, wr.user_id, COALESCE(u.email, ''), COALESCE(u.username, ''),
		       wr.amount, wr.method, wr.account_info,
		       wr.status, wr.note, wr.requested_at, wr.reviewed_at, wr.reviewed_by
		  FROM affiliate_withdraw_requests wr
		  LEFT JOIN users u ON u.id = wr.user_id
		 WHERE ($1 = 0 OR wr.user_id = $1) AND ($2 = '' OR wr.status = $2)
		   AND ($3 = 0 OR EXISTS (
		       SELECT 1 FROM user_reseller_roles rr
		       WHERE rr.user_id = wr.user_id
		         AND rr.role = 'agent' AND rr.granted_by = $3 AND rr.revoked_at IS NULL
		   ))
		 ORDER BY wr.requested_at DESC
		 LIMIT $4 OFFSET $5`,
		filter.UserID, filter.Status, filter.ManagerID, pageSize, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("reseller ListWithdrawRequests query: %w", err)
	}
	defer rows.Close()

	var out []service.WithdrawRequest
	for rows.Next() {
		req, err := scanWithdrawRequest(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("reseller ListWithdrawRequests scan: %w", err)
		}
		out = append(out, *req)
	}
	return out, total, rows.Err()
}

// --- ReviewWithdrawRequest ---

func (r *resellerRepository) ReviewWithdrawRequest(ctx context.Context, id, reviewerID int64, status, note string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("reseller ReviewWithdrawRequest begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var userID int64
	var amount float64
	var currentStatus string
	if err := tx.QueryRowContext(ctx, `
		SELECT user_id, amount::double precision, status
		FROM affiliate_withdraw_requests
		WHERE id = $1
		FOR UPDATE`, id).Scan(&userID, &amount, &currentStatus); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return service.ErrWithdrawRequestNotFound
		}
		return fmt.Errorf("reseller ReviewWithdrawRequest lock request: %w", err)
	}
	if currentStatus != service.WithdrawStatusPending {
		return service.ErrWithdrawAlreadyReviewed.WithMetadata(map[string]string{
			"current_status": currentStatus,
		})
	}

	if status == service.WithdrawStatusApproved {
		var quotaAfter, frozenAfter, historyAfter float64
		err = tx.QueryRowContext(ctx, `
			UPDATE user_affiliates
			SET aff_quota = aff_quota - $1, updated_at = NOW()
			WHERE user_id = $2 AND aff_quota >= $1
			RETURNING aff_quota::double precision,
			          aff_frozen_quota::double precision,
			          aff_history_quota::double precision`,
			amount, userID,
		).Scan(&quotaAfter, &frozenAfter, &historyAfter)
		if errors.Is(err, sql.ErrNoRows) {
			return service.ErrWithdrawInsufficientBalance
		}
		if err != nil {
			return fmt.Errorf("reseller ReviewWithdrawRequest debit quota: %w", err)
		}

		var balanceAfter float64
		if err := tx.QueryRowContext(ctx, `
			UPDATE users
			SET balance = balance + $1,
			    total_recharged = total_recharged + $1,
			    updated_at = NOW()
			WHERE id = $2 AND deleted_at IS NULL
			RETURNING balance::double precision`,
			amount, userID,
		).Scan(&balanceAfter); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return service.ErrUserNotFound
			}
			return fmt.Errorf("reseller ReviewWithdrawRequest credit balance: %w", err)
		}

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO user_affiliate_ledger (
				user_id, action, amount, source_user_id,
				balance_after, aff_quota_after, aff_frozen_quota_after,
				aff_history_quota_after, notes, created_at, updated_at
			)
			VALUES ($1, 'transfer', $2, NULL, $3, $4, $5, $6, $7, NOW(), NOW())`,
			userID, amount, balanceAfter, quotaAfter, frozenAfter, historyAfter,
			fmt.Sprintf("reseller withdrawal #%d", id),
		); err != nil {
			return fmt.Errorf("reseller ReviewWithdrawRequest insert ledger: %w", err)
		}
	} else if status != service.WithdrawStatusRejected {
		return service.ErrWithdrawInvalidStatus
	}

	result, err := tx.ExecContext(ctx, `
		UPDATE affiliate_withdraw_requests
		SET status = $1, note = $2, reviewed_at = NOW(), reviewed_by = $3, updated_at = NOW()
		WHERE id = $4 AND status = 'pending'`,
		status, note, reviewerID, id,
	)
	if err != nil {
		return fmt.Errorf("reseller ReviewWithdrawRequest update status: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("reseller ReviewWithdrawRequest rows affected: %w", err)
	}
	if affected != 1 {
		return service.ErrWithdrawAlreadyReviewed
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("reseller ReviewWithdrawRequest commit: %w", err)
	}
	return nil
}

// --- helpers ---

// scanWithdrawRequest reads one row from the affiliate_withdraw_requests JOIN users query.
func scanWithdrawRequest(rows *sql.Rows) (*service.WithdrawRequest, error) {
	var req service.WithdrawRequest
	var accountRaw []byte
	var reviewedAt sql.NullTime
	var reviewedBy sql.NullInt64
	if err := rows.Scan(
		&req.ID, &req.UserID, &req.UserEmail, &req.Username,
		&req.Amount, &req.Method, &accountRaw,
		&req.Status, &req.Note, &req.RequestedAt, &reviewedAt, &reviewedBy,
	); err != nil {
		return nil, err
	}
	if reviewedAt.Valid {
		req.ReviewedAt = &reviewedAt.Time
	}
	if reviewedBy.Valid {
		req.ReviewedBy = &reviewedBy.Int64
	}
	if len(accountRaw) > 0 {
		_ = json.Unmarshal(accountRaw, &req.AccountInfo)
	}
	return &req, nil
}
