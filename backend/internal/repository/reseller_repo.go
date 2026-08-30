// Package repository — resellerRepository implements service.ResellerRepository
// using raw SQL against PostgreSQL (via the ent client's underlying *sql.DB).
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type resellerRepository struct {
	db *sql.DB
}

const resellerMutationAdvisoryLock int64 = 2026073001

// NewResellerRepository constructs the repository. All queries run directly
// against the underlying *sql.DB; the ent client is not needed here.
func NewResellerRepository(_ *dbent.Client, db *sql.DB) service.ResellerRepository {
	return &resellerRepository{db: db}
}

// --- GrantRole ---

func (r *resellerRepository) GrantRole(ctx context.Context, userID int64, role string, grantedBy int64, notes string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("reseller repo GrantRole begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := lockResellerMutations(ctx, tx); err != nil {
		return err
	}

	var currentRole string
	var revokedAt sql.NullTime
	err = tx.QueryRowContext(ctx, `
		SELECT role, revoked_at
		FROM user_reseller_roles
		WHERE user_id = $1
		FOR UPDATE`, userID).Scan(&currentRole, &revokedAt)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		_, err = tx.ExecContext(ctx, `
			INSERT INTO user_reseller_roles (
				user_id, role, granted_by, granted_at, notes, status, updated_at, updated_by
			)
			VALUES ($1, $2, $3, NOW(), $4, 'active', NOW(), $3)`,
			userID, role, grantedBy, notes,
		)
	case err != nil:
		return fmt.Errorf("reseller repo GrantRole lock: %w", err)
	default:
		if !revokedAt.Valid && currentRole == service.ResellerRoleManager && role == service.ResellerRoleAgent {
			hasChildren, checkErr := hasDirectAgents(ctx, tx, userID)
			if checkErr != nil {
				return checkErr
			}
			if hasChildren {
				return service.ErrResellerHasDirectAgents
			}
		}
		_, err = tx.ExecContext(ctx, `
			UPDATE user_reseller_roles
			SET role = $2,
			    granted_by = $3,
			    granted_at = NOW(),
			    revoked_at = NULL,
			    notes = $4,
			    status = 'active',
			    manager_id = CASE
			        WHEN role = 'agent' AND $2 = 'agent' THEN manager_id
			        ELSE NULL
			    END,
			    updated_at = NOW(),
			    updated_by = $3,
			    disabled_at = NULL,
			    disabled_by = NULL,
			    disabled_reason = ''
			WHERE user_id = $1`,
			userID, role, grantedBy, notes,
		)
	}
	if err != nil {
		return fmt.Errorf("reseller repo GrantRole write: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("reseller repo GrantRole commit: %w", err)
	}
	return nil
}

func (r *resellerRepository) GrantManagedAgent(ctx context.Context, userID, managerID int64, notes string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("reseller repo GrantManagedAgent begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := lockResellerMutations(ctx, tx); err != nil {
		return err
	}
	if err := validateManagerAssignment(ctx, tx, userID, managerID); err != nil {
		return err
	}

	result, err := tx.ExecContext(ctx, `
		INSERT INTO user_reseller_roles (
			user_id, role, granted_by, granted_at, notes, status, manager_id, updated_at, updated_by
		)
		VALUES ($1, 'agent', $2, NOW(), $3, 'active', $2, NOW(), $2)
		ON CONFLICT (user_id) DO UPDATE SET
			role = 'agent',
			granted_by = EXCLUDED.granted_by,
			granted_at = NOW(),
			revoked_at = NULL,
			notes = EXCLUDED.notes,
			status = 'active',
			manager_id = $2,
			updated_at = NOW(),
			updated_by = $2,
			disabled_at = NULL,
			disabled_by = NULL,
			disabled_reason = ''
		WHERE user_reseller_roles.revoked_at IS NOT NULL
		   OR (user_reseller_roles.role = 'agent' AND user_reseller_roles.manager_id = $2)`,
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
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("reseller repo GrantManagedAgent commit: %w", err)
	}
	return nil
}

// --- RevokeRole ---

func (r *resellerRepository) RevokeRole(ctx context.Context, userID, updatedBy int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("reseller repo RevokeRole begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := lockResellerMutations(ctx, tx); err != nil {
		return err
	}
	if _, _, err := lockResellerRole(ctx, tx, userID); err != nil {
		return err
	}
	if err := ensureAgentCanBeRevoked(ctx, tx, userID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE user_reseller_roles
		SET revoked_at = NOW(), updated_at = NOW(), updated_by = $2
		WHERE user_id = $1 AND revoked_at IS NULL`, userID, updatedBy); err != nil {
		return fmt.Errorf("reseller repo RevokeRole: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("reseller repo RevokeRole commit: %w", err)
	}
	return nil
}

func (r *resellerRepository) RevokeManagedAgent(ctx context.Context, userID, managerID int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("reseller repo RevokeManagedAgent begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := lockResellerMutations(ctx, tx); err != nil {
		return err
	}
	var ownerID sql.NullInt64
	var role string
	var revokedAt sql.NullTime
	if err := tx.QueryRowContext(ctx, `
		SELECT role, manager_id, revoked_at
		FROM user_reseller_roles
		WHERE user_id = $1
		FOR UPDATE`, userID).Scan(&role, &ownerID, &revokedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return service.ErrResellerAgentNotManaged
		}
		return fmt.Errorf("reseller repo RevokeManagedAgent lock: %w", err)
	}
	if role != service.ResellerRoleAgent || !ownerID.Valid || ownerID.Int64 != managerID || revokedAt.Valid {
		return service.ErrResellerAgentNotManaged
	}
	if err := ensureAgentCanBeRevoked(ctx, tx, userID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE user_reseller_roles
		SET revoked_at = NOW(), updated_at = NOW(), updated_by = $2
		WHERE user_id = $1
		  AND role = 'agent'
		  AND manager_id = $2
		  AND revoked_at IS NULL`,
		userID, managerID,
	); err != nil {
		return fmt.Errorf("reseller repo RevokeManagedAgent: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("reseller repo RevokeManagedAgent commit: %w", err)
	}
	return nil
}

// --- GetManagerDashboard ---

func (r *resellerRepository) GetManagerDashboard(ctx context.Context, managerID int64) (*service.ManagerDashboard, error) {
	var d service.ManagerDashboard
	err := r.db.QueryRowContext(ctx, `
		SELECT
		    (SELECT COUNT(*) FROM user_reseller_roles
		      WHERE role = 'agent' AND manager_id = $1 AND revoked_at IS NULL)::int,
		    (SELECT COUNT(DISTINCT ua.user_id)
		       FROM user_affiliates ua
		       JOIN user_reseller_roles rr
		         ON rr.user_id = ua.inviter_id AND rr.role = 'agent'
		        AND rr.manager_id = $1 AND rr.revoked_at IS NULL)::int,
		    (SELECT COUNT(*)
		       FROM affiliate_withdraw_requests wr
		       JOIN user_reseller_roles rr ON rr.user_id = wr.user_id
		      WHERE wr.status = 'pending' AND rr.role = 'agent'
		        AND rr.manager_id = $1 AND rr.revoked_at IS NULL)::int
	`, managerID).Scan(&d.TotalAgents, &d.TotalRecruits, &d.PendingWithdrawals)
	if err != nil {
		return nil, fmt.Errorf("reseller repo GetManagerDashboard: %w", err)
	}
	return &d, nil
}

// --- GetRole ---

func (r *resellerRepository) GetRole(ctx context.Context, userID int64) (*service.ResellerRoleRecord, error) {
	var rec service.ResellerRoleRecord
	var managerID sql.NullInt64
	var grantedBy sql.NullInt64
	var revokedAt sql.NullTime
	err := r.db.QueryRowContext(ctx,
		`SELECT user_id, role, status, manager_id, granted_by, granted_at, revoked_at, notes,
		        commission_rate::double precision
		   FROM user_reseller_roles
		  WHERE user_id = $1 AND status = 'active' AND revoked_at IS NULL
		  LIMIT 1`,
		userID,
	).Scan(
		&rec.UserID, &rec.Role, &rec.Status, &managerID, &grantedBy, &rec.GrantedAt, &revokedAt, &rec.Notes,
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
	if managerID.Valid {
		rec.ManagerID = &managerID.Int64
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
	status := filter.Status
	if status == "" {
		status = "current"
	}

	var total int64
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM user_reseller_roles rr
		  JOIN users u ON u.id = rr.user_id
		 WHERE (
		       $4 = 'all'
		       OR ($4 = 'current' AND rr.revoked_at IS NULL)
		       OR ($4 = 'revoked' AND rr.revoked_at IS NOT NULL)
		       OR ($4 IN ('active', 'disabled') AND rr.revoked_at IS NULL AND rr.status = $4)
		   )
		   AND (rr.role = 'agent' OR ($3 AND rr.role = 'agent_manager'))
		   AND ($5 = '' OR rr.role = $5)
		   AND ($1 = '' OR u.email ILIKE '%' || $1 || '%')
		   AND ($2 = 0 OR rr.manager_id = $2)`,
		filter.Search, filter.ManagerID, filter.IncludeAllRoles, status, filter.Role,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("reseller ListAgents count: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT rr.user_id, u.email, COALESCE(u.username, ''),
		       rr.role,
		       CASE WHEN rr.revoked_at IS NOT NULL THEN 'revoked' ELSE rr.status END,
		       rr.manager_id, manager.email,
		       ua.aff_rebate_rate_percent::double precision,
		       CASE
		           WHEN ua.aff_rebate_rate_percent IS NULL THEN 'global'
		           WHEN ua.aff_rebate_rate_percent = 0 THEN 'disabled'
		           ELSE 'custom'
		       END,
		       COALESCE(ua.aff_code, ''), COALESCE(ua.aff_count, 0)::int,
		       TO_CHAR(COALESCE(ua.aff_quota, 0), 'FM999999999999990.00'),
		       TO_CHAR((
		           COALESCE(ua.aff_quota, 0)
		           + COALESCE((
		               SELECT SUM(wr.amount)
		               FROM affiliate_withdraw_requests wr
		               WHERE wr.user_id = rr.user_id AND wr.status = 'approved'
		           ), 0)
		       ), 'FM999999999999990.00'),
		       rr.notes, rr.granted_at, rr.updated_at, rr.disabled_at,
		       disabled_user.email, rr.disabled_reason, rr.revoked_at, rr.granted_by
		  FROM user_reseller_roles rr
		  JOIN users u ON u.id = rr.user_id
		  LEFT JOIN user_affiliates ua ON ua.user_id = rr.user_id
		  LEFT JOIN users manager ON manager.id = rr.manager_id
		  LEFT JOIN users disabled_user ON disabled_user.id = rr.disabled_by
		 WHERE (
		       $4 = 'all'
		       OR ($4 = 'current' AND rr.revoked_at IS NULL)
		       OR ($4 = 'revoked' AND rr.revoked_at IS NOT NULL)
		       OR ($4 IN ('active', 'disabled') AND rr.revoked_at IS NULL AND rr.status = $4)
		   )
		   AND (rr.role = 'agent' OR ($3 AND rr.role = 'agent_manager'))
		   AND ($5 = '' OR rr.role = $5)
		   AND ($1 = '' OR u.email ILIKE '%' || $1 || '%')
		   AND ($2 = 0 OR rr.manager_id = $2)
		 ORDER BY rr.granted_at DESC
		 LIMIT $6 OFFSET $7`,
		filter.Search, filter.ManagerID, filter.IncludeAllRoles, status, filter.Role, pageSize, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("reseller ListAgents query: %w", err)
	}
	defer rows.Close()

	var out []service.AgentSummary
	for rows.Next() {
		s, err := scanAgentSummary(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("reseller ListAgents scan: %w", err)
		}
		out = append(out, *s)
	}
	return out, total, rows.Err()
}

// --- GetAgentDetail ---

func (r *resellerRepository) GetAgentDetail(ctx context.Context, agentUserID, managerID int64) (*service.AgentDetail, error) {
	d, err := queryAgentDetail(ctx, r.db, agentUserID, managerID, false)
	if errors.Is(err, sql.ErrNoRows) {
		if managerID > 0 {
			return nil, service.ErrResellerAgentNotManaged
		}
		return nil, service.ErrResellerRoleNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("reseller GetAgentDetail: %w", err)
	}
	recruits, _, err := r.ListMyRecruits(ctx, agentUserID, 1, 500, false)
	if err != nil {
		return nil, err
	}
	d.Recruits = recruits
	return d, nil
}

func (r *resellerRepository) GetAdminAgentDetail(ctx context.Context, agentUserID int64) (*service.AgentDetail, error) {
	d, err := queryAgentDetail(ctx, r.db, agentUserID, 0, true)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrResellerRoleNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("reseller GetAdminAgentDetail: %w", err)
	}
	return d, nil
}

func (r *resellerRepository) UpdateAgent(
	ctx context.Context,
	agentUserID, updatedBy int64,
	input service.UpdateAgentInput,
) (*service.AgentDetail, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("reseller UpdateAgent begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := lockResellerMutations(ctx, tx); err != nil {
		return nil, err
	}
	currentRole, _, err := lockResellerRole(ctx, tx, agentUserID)
	if err != nil {
		return nil, err
	}
	if input.Role != nil && currentRole == service.ResellerRoleManager && *input.Role == service.ResellerRoleAgent {
		hasChildren, err := hasDirectAgents(ctx, tx, agentUserID)
		if err != nil {
			return nil, err
		}
		if hasChildren {
			return nil, service.ErrResellerHasDirectAgents
		}
	}
	if input.ManagerID.Set && input.ManagerID.Value != nil {
		if err := validateManagerAssignment(ctx, tx, agentUserID, *input.ManagerID.Value); err != nil {
			return nil, err
		}
	}

	roleSet := input.Role != nil
	role := ""
	if roleSet {
		role = *input.Role
	}
	notesSet := input.Notes != nil
	notes := ""
	if notesSet {
		notes = *input.Notes
	}
	var managerValue any
	if input.ManagerID.Value != nil {
		managerValue = *input.ManagerID.Value
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE user_reseller_roles
		SET role = CASE WHEN $3 THEN $4 ELSE role END,
		    manager_id = CASE WHEN $5 THEN $6 ELSE manager_id END,
		    notes = CASE WHEN $7 THEN $8 ELSE notes END,
		    updated_at = NOW(),
		    updated_by = $2
		WHERE user_id = $1 AND revoked_at IS NULL`,
		agentUserID, updatedBy,
		roleSet, role,
		input.ManagerID.Set, managerValue,
		notesSet, notes,
	); err != nil {
		return nil, fmt.Errorf("reseller UpdateAgent update role: %w", err)
	}

	if input.RebatePolicy != nil {
		var rate any
		switch input.RebatePolicy.Mode {
		case service.RebateModeGlobal:
			rate = nil
		case service.RebateModeDisabled:
			rate = float64(0)
		case service.RebateModeCustom:
			rate = *input.RebatePolicy.RatePercent
		}
		result, err := tx.ExecContext(ctx, `
			UPDATE user_affiliates
			SET aff_rebate_rate_percent = $2, updated_at = NOW()
			WHERE user_id = $1`, agentUserID, rate)
		if err != nil {
			return nil, fmt.Errorf("reseller UpdateAgent update rebate: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return nil, fmt.Errorf("reseller UpdateAgent rebate rows affected: %w", err)
		}
		if affected != 1 {
			return nil, service.ErrAffiliateProfileNotFound
		}
	}

	detail, err := queryAgentDetail(ctx, tx, agentUserID, 0, true)
	if err != nil {
		return nil, fmt.Errorf("reseller UpdateAgent detail: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("reseller UpdateAgent commit: %w", err)
	}
	return detail, nil
}

func (r *resellerRepository) DisableAgent(ctx context.Context, agentUserID, updatedBy int64, reason string) (*service.AgentDetail, error) {
	return r.changeAgentStatus(ctx, agentUserID, updatedBy, service.ResellerStatusActive, service.ResellerStatusDisabled, reason)
}

func (r *resellerRepository) EnableAgent(ctx context.Context, agentUserID, updatedBy int64) (*service.AgentDetail, error) {
	return r.changeAgentStatus(ctx, agentUserID, updatedBy, service.ResellerStatusDisabled, service.ResellerStatusActive, "")
}

func (r *resellerRepository) changeAgentStatus(
	ctx context.Context,
	agentUserID, updatedBy int64,
	expected, next, reason string,
) (*service.AgentDetail, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("reseller changeAgentStatus begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := lockResellerMutations(ctx, tx); err != nil {
		return nil, err
	}
	_, status, err := lockResellerRole(ctx, tx, agentUserID)
	if err != nil {
		return nil, err
	}
	if status != expected {
		return nil, service.ErrResellerStateConflict.WithMetadata(map[string]string{
			"current_status": status,
		})
	}
	if next == service.ResellerStatusDisabled {
		_, err = tx.ExecContext(ctx, `
			UPDATE user_reseller_roles
			SET status = 'disabled', disabled_at = NOW(), disabled_by = $2,
			    disabled_reason = $3, updated_at = NOW(), updated_by = $2
			WHERE user_id = $1 AND revoked_at IS NULL`, agentUserID, updatedBy, reason)
	} else {
		_, err = tx.ExecContext(ctx, `
			UPDATE user_reseller_roles
			SET status = 'active', disabled_at = NULL, disabled_by = NULL,
			    disabled_reason = '', updated_at = NOW(), updated_by = $2
			WHERE user_id = $1 AND revoked_at IS NULL`, agentUserID, updatedBy)
	}
	if err != nil {
		return nil, fmt.Errorf("reseller changeAgentStatus update: %w", err)
	}
	detail, err := queryAgentDetail(ctx, tx, agentUserID, 0, true)
	if err != nil {
		return nil, fmt.Errorf("reseller changeAgentStatus detail: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("reseller changeAgentStatus commit: %w", err)
	}
	return detail, nil
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
		       COALESCE(u.status, ''),
		       COALESCE((
		           SELECT rr.role
		           FROM user_reseller_roles rr
		           WHERE rr.user_id = ua.user_id AND rr.revoked_at IS NULL
		           LIMIT 1
		       ), ''),
		       COALESCE(
		           ua.aff_rebate_rate_percent::double precision,
		           NULLIF((SELECT value FROM settings WHERE key = 'affiliate_rebate_rate' LIMIT 1), '')::double precision,
		           0
		       ),
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
			&joinedAt, &rec.Status, &rec.ResellerRole, &rec.CommissionRate,
			&rec.TotalRebate, &rec.IsActive,
		); err != nil {
			return nil, 0, fmt.Errorf("reseller ListMyRecruits scan: %w", err)
		}
		if maskEmail {
			rec.Email = service.MaskEmail(email)
		} else {
			rec.Email = email
		}
		if joinedAt.Valid {
			rec.CreatedAt = &joinedAt.Time
			rec.JoinedAt = &joinedAt.Time
		}
		out = append(out, rec)
	}
	return out, total, rows.Err()
}

// ListAdminAgentRecruits returns the direct recruits of an agent with the
// financial aggregates needed by the admin detail view.
func (r *resellerRepository) ListAdminAgentRecruits(ctx context.Context, agentUserID int64, page, pageSize int) ([]service.AdminRecruitRecord, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var total int64
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*)
           FROM user_affiliates
          WHERE inviter_id = $1
            AND user_id <> $1`,
		agentUserID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("reseller ListAdminAgentRecruits count: %w", err)
	}

	rateSQL := `COALESCE(
        agent_aff.aff_rebate_rate_percent::double precision,
        NULLIF((SELECT value FROM settings WHERE key = 'affiliate_rebate_rate' LIMIT 1), '')::double precision,
        0
    ) / 100.0`

	rows, err := r.db.QueryContext(ctx, `
SELECT ua.user_id,
       COALESCE(u.email, ''),
       COALESCE(u.username, ''),
       COALESCE(u.status, ''),
       COALESCE((
           SELECT rr.role
             FROM user_reseller_roles rr
            WHERE rr.user_id = ua.user_id
              AND rr.revoked_at IS NULL
            LIMIT 1
       ), ''),
       u.created_at,
       ua.created_at,
       EXISTS (
           SELECT 1
             FROM usage_logs ul
            WHERE ul.user_id = ua.user_id
              AND ul.created_at >= NOW() - INTERVAL '30 days'
       ),
       COALESCE((
           SELECT SUM(po.amount)
             FROM payment_orders po
            WHERE po.user_id = ua.user_id
              AND po.order_type = 'balance'
              AND UPPER(po.status) IN ('SUCCEEDED', 'PAID', 'COMPLETED', 'RECHARGING')
       ), 0)::double precision,
       COALESCE((
           SELECT SUM(ul.actual_cost)
             FROM usage_logs ul
            WHERE ul.user_id = ua.user_id
       ), 0)::double precision,
       COALESCE(u.balance, 0)::double precision,
       (COALESCE((
           SELECT SUM(ul.actual_cost)
             FROM usage_logs ul
            WHERE ul.user_id = ua.user_id
       ), 0)::double precision * (`+rateSQL+`))::double precision
  FROM user_affiliates ua
  JOIN users u ON u.id = ua.user_id
  LEFT JOIN user_affiliates agent_aff ON agent_aff.user_id = $1
 WHERE ua.inviter_id = $1
   AND ua.user_id <> $1
 ORDER BY u.created_at DESC, ua.user_id DESC
 LIMIT $2 OFFSET $3`,
		agentUserID, pageSize, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("reseller ListAdminAgentRecruits query: %w", err)
	}
	defer rows.Close()

	out := make([]service.AdminRecruitRecord, 0, pageSize)
	for rows.Next() {
		var rec service.AdminRecruitRecord
		var createdAt sql.NullTime
		var joinedAt sql.NullTime
		if err := rows.Scan(
			&rec.UserID,
			&rec.Email,
			&rec.Username,
			&rec.Status,
			&rec.ResellerRole,
			&createdAt,
			&joinedAt,
			&rec.IsActive,
			&rec.TotalRechargeDollars,
			&rec.TotalCostDollars,
			&rec.CurrentBalance,
			&rec.CommissionContrib,
		); err != nil {
			return nil, 0, fmt.Errorf("reseller ListAdminAgentRecruits scan: %w", err)
		}
		if createdAt.Valid {
			rec.CreatedAt = &createdAt.Time
		}
		if joinedAt.Valid {
			rec.JoinedAt = &joinedAt.Time
		}
		rec.Email = service.MaskEmailForResellerAdmin(rec.Email)
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("reseller ListAdminAgentRecruits rows: %w", err)
	}
	return out, total, nil
}

const recruitDownlineScopeSQL = `WITH RECURSIVE downline AS (
    SELECT ua.user_id
    FROM user_affiliates ua
    WHERE ua.inviter_id = $1
    UNION ALL
    SELECT child.user_id
    FROM user_affiliates child
    JOIN downline parent ON parent.user_id = child.inviter_id
)`

func (r *resellerRepository) GetRecruitDetail(ctx context.Context, agentUserID, recruitUserID int64, maskEmail bool) (*service.RecruitRecord, error) {
	var rec service.RecruitRecord
	var joinedAt sql.NullTime
	var email string
	err := r.db.QueryRowContext(ctx, recruitDownlineScopeSQL+`
SELECT ua.user_id, u.email, COALESCE(u.username, ''),
       u.created_at,
       COALESCE(u.status, ''),
       COALESCE((
           SELECT rr.role
           FROM user_reseller_roles rr
           WHERE rr.user_id = ua.user_id AND rr.revoked_at IS NULL
           LIMIT 1
       ), ''),
       COALESCE(
           ua.aff_rebate_rate_percent::double precision,
           NULLIF((SELECT value FROM settings WHERE key = 'affiliate_rebate_rate' LIMIT 1), '')::double precision,
           0
       ),
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
WHERE ua.user_id = $2
  AND EXISTS (SELECT 1 FROM downline WHERE user_id = $2)`,
		agentUserID, recruitUserID,
	).Scan(
		&rec.UserID, &email, &rec.Username, &joinedAt, &rec.Status,
		&rec.ResellerRole, &rec.CommissionRate, &rec.TotalRebate, &rec.IsActive,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrResellerAgentNotManaged
	}
	if err != nil {
		return nil, fmt.Errorf("reseller GetRecruitDetail: %w", err)
	}
	if maskEmail {
		rec.Email = service.MaskEmail(email)
	} else {
		rec.Email = email
	}
	if joinedAt.Valid {
		rec.CreatedAt = &joinedAt.Time
		rec.JoinedAt = &joinedAt.Time
	}
	return &rec, nil
}

func (r *resellerRepository) ListRecruitUsageLogs(ctx context.Context, agentUserID, recruitUserID int64, page, pageSize int) ([]service.RecruitUsageLog, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var total int64
	if err := r.db.QueryRowContext(ctx, recruitDownlineScopeSQL+`
SELECT COUNT(*)
FROM usage_logs ul
JOIN downline d ON d.user_id = ul.user_id
WHERE d.user_id = $2`, agentUserID, recruitUserID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("reseller ListRecruitUsageLogs count: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, recruitDownlineScopeSQL+`
SELECT ul.id,
       ul.created_at,
       COALESCE(NULLIF(ul.requested_model, ''), ul.model),
       COALESCE(ul.request_type, 0)::smallint,
       (COALESCE(ul.input_tokens, 0) + COALESCE(ul.output_tokens, 0)
        + COALESCE(ul.cache_creation_tokens, 0) + COALESCE(ul.cache_read_tokens, 0)
        + COALESCE(ul.cache_creation_5m_tokens, 0) + COALESCE(ul.cache_creation_1h_tokens, 0))::bigint,
       ul.actual_cost::double precision
FROM usage_logs ul
JOIN downline d ON d.user_id = ul.user_id
WHERE d.user_id = $2
ORDER BY ul.created_at DESC, ul.id DESC
LIMIT $3 OFFSET $4`, agentUserID, recruitUserID, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("reseller ListRecruitUsageLogs query: %w", err)
	}
	defer rows.Close()

	items := make([]service.RecruitUsageLog, 0)
	for rows.Next() {
		var item service.RecruitUsageLog
		if err := rows.Scan(&item.ID, &item.CreatedAt, &item.Model, &item.RequestType, &item.TotalTokens, &item.ActualCost); err != nil {
			return nil, 0, fmt.Errorf("reseller ListRecruitUsageLogs scan: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("reseller ListRecruitUsageLogs rows: %w", err)
	}
	return items, total, nil
}

func (r *resellerRepository) ListRecruitRecharges(ctx context.Context, agentUserID, recruitUserID int64, page, pageSize int) ([]service.RecruitRecharge, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var total int64
	if err := r.db.QueryRowContext(ctx, recruitDownlineScopeSQL+`
SELECT COUNT(*)
FROM account_balance_ledger bl
JOIN downline d ON d.user_id = bl.user_id
WHERE d.user_id = $2
  AND bl.amount_delta > 0`, agentUserID, recruitUserID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("reseller ListRecruitRecharges count: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, recruitDownlineScopeSQL+`
SELECT bl.id,
       bl.event_type,
       bl.amount_delta::double precision,
       bl.note,
       bl.created_at
FROM account_balance_ledger bl
JOIN downline d ON d.user_id = bl.user_id
WHERE d.user_id = $2
  AND bl.amount_delta > 0
ORDER BY bl.created_at DESC, bl.id DESC
LIMIT $3 OFFSET $4`, agentUserID, recruitUserID, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("reseller ListRecruitRecharges query: %w", err)
	}
	defer rows.Close()

	items := make([]service.RecruitRecharge, 0)
	for rows.Next() {
		var item service.RecruitRecharge
		if err := rows.Scan(&item.ID, &item.EventType, &item.Amount, &item.Note, &item.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("reseller ListRecruitRecharges scan: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("reseller ListRecruitRecharges rows: %w", err)
	}
	return items, total, nil
}

// ListCommission lists usage rows generated by the agent's direct recruits.
// The rebate rate is resolved from the agent's custom setting first, then the global setting.
func (r *resellerRepository) ListCommission(ctx context.Context, filter service.CommissionFilter) ([]service.CommissionRecord, int64, float64, error) {
	page, pageSize := filter.Page, filter.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 50
	}
	offset := (page - 1) * pageSize

	where := []string{
		"invitee.inviter_id = $1",
		"ul.actual_cost > 0",
	}
	args := []any{filter.AgentUserID}
	if filter.StartAt != nil {
		args = append(args, *filter.StartAt)
		where = append(where, fmt.Sprintf("ul.created_at >= $%d", len(args)))
	}
	if filter.EndAt != nil {
		args = append(args, *filter.EndAt)
		where = append(where, fmt.Sprintf("ul.created_at < $%d", len(args)))
	}

	fromSQL := `
FROM usage_logs ul
JOIN user_affiliates invitee ON invitee.user_id = ul.user_id
JOIN users source_user ON source_user.id = ul.user_id
LEFT JOIN user_affiliates inviter_aff ON inviter_aff.user_id = invitee.inviter_id
LEFT JOIN settings affiliate_rate ON affiliate_rate.key = 'affiliate_rebate_rate'`
	rateSQL := `COALESCE(inviter_aff.aff_rebate_rate_percent::double precision,
                         NULLIF(affiliate_rate.value, '')::double precision,
                         0) / 100.0`
	whereSQL := strings.Join(where, " AND ")

	var total int64
	var totalCommission float64
	if err := r.db.QueryRowContext(ctx, `
SELECT COUNT(*)::bigint,
       COALESCE(SUM((ul.actual_cost::double precision) * (`+rateSQL+`)), 0)::double precision
`+fromSQL+`
WHERE `+whereSQL, args...).Scan(&total, &totalCommission); err != nil {
		return nil, 0, 0, fmt.Errorf("reseller ListCommission summary: %w", err)
	}

	args = append(args, pageSize, offset)
	rows, err := r.db.QueryContext(ctx, `
SELECT ul.id,
       ul.created_at,
       COALESCE(source_user.email, ''),
       ul.actual_cost::double precision,
       ((ul.actual_cost::double precision) * (`+rateSQL+`))::double precision,
       (`+rateSQL+`)::double precision
`+fromSQL+`
WHERE `+whereSQL+`
ORDER BY ul.created_at DESC, ul.id DESC
LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("reseller ListCommission query: %w", err)
	}
	defer rows.Close()

	items := make([]service.CommissionRecord, 0)
	for rows.Next() {
		var item service.CommissionRecord
		var email string
		if err := rows.Scan(
			&item.ID,
			&item.Time,
			&email,
			&item.SourceConsumptionUSD,
			&item.CommissionUSD,
			&item.CommissionRate,
		); err != nil {
			return nil, 0, 0, fmt.Errorf("reseller ListCommission scan: %w", err)
		}
		item.SourceUserMaskedEmail = service.MaskEmail(email)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, 0, fmt.Errorf("reseller ListCommission rows: %w", err)
	}
	return items, total, totalCommission, nil
}

// GetInviteSummary reads the existing affiliate code and recruit counters without creating data.
func (r *resellerRepository) GetInviteSummary(ctx context.Context, agentUserID int64) (*service.InviteSummary, error) {
	summary := &service.InviteSummary{}
	err := r.db.QueryRowContext(ctx, `
SELECT COALESCE(aff_code, ''),
       (SELECT COUNT(*) FROM user_affiliates WHERE inviter_id = $1)::int,
       (SELECT COUNT(*)
          FROM user_affiliates
         WHERE inviter_id = $1
           AND created_at >= date_trunc('month', CURRENT_TIMESTAMP))::int
  FROM user_affiliates
 WHERE user_id = $1`,
		agentUserID,
	).Scan(&summary.InviteCode, &summary.TotalRecruited, &summary.RecruitedThisMonth)
	if errors.Is(err, sql.ErrNoRows) {
		return summary, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reseller GetInviteSummary: %w", err)
	}
	return summary, nil
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

	if err := lockResellerMutations(ctx, tx); err != nil {
		return nil, err
	}
	if _, _, err := lockResellerRole(ctx, tx, userID); err != nil {
		return nil, err
	}

	var req service.WithdrawRequest
	var accountRaw []byte
	err = tx.QueryRowContext(ctx, `
		SELECT id, user_id, amount::double precision, method, account_info, status, note, requested_at
		FROM affiliate_withdraw_requests
		WHERE user_id = $1 AND idempotency_key = $2
		FOR UPDATE`, userID, input.IdempotencyKey,
	).Scan(&req.ID, &req.UserID, &req.Amount, &req.Method, &accountRaw, &req.Status, &req.Note, &req.RequestedAt)
	if err == nil {
		if math.Abs(req.Amount-input.Amount) > 1e-9 || req.Method != input.Method {
			return nil, service.ErrWithdrawIdempotencyConflict
		}
		if len(accountRaw) > 0 {
			_ = json.Unmarshal(accountRaw, &req.AccountInfo)
		}
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("reseller CreateWithdrawRequest idempotent commit: %w", err)
		}
		return &req, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("reseller CreateWithdrawRequest find idempotency key: %w", err)
	}

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

	err = tx.QueryRowContext(ctx, `
		INSERT INTO affiliate_withdraw_requests (user_id, amount, method, account_info, status, idempotency_key)
		VALUES ($1, $2, $3, $4, 'pending', $5)
		RETURNING id, user_id, amount, method, account_info, status, note, requested_at`,
		userID, input.Amount, input.Method, accountJSON, input.IdempotencyKey,
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

	if err := lockResellerMutations(ctx, tx); err != nil {
		return err
	}
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
		         AND rr.role = 'agent' AND rr.manager_id = $3 AND rr.revoked_at IS NULL
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
		         AND rr.role = 'agent' AND rr.manager_id = $3 AND rr.revoked_at IS NULL
		   ))
		 ORDER BY wr.requested_at DESC
		 LIMIT $4 OFFSET $5`,
		filter.UserID, filter.Status, filter.ManagerID, pageSize, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("reseller ListWithdrawRequests query: %w", err)
	}
	defer rows.Close()

	out := make([]service.WithdrawRequest, 0)
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

	if err := lockResellerMutations(ctx, tx); err != nil {
		return err
	}
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

type resellerQueryRower interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type resellerRowScanner interface {
	Scan(...any) error
}

func scanAgentSummary(row resellerRowScanner) (*service.AgentSummary, error) {
	var out service.AgentSummary
	var managerID, grantedBy sql.NullInt64
	var managerEmail, disabledByEmail sql.NullString
	var rebateRate sql.NullFloat64
	var disabledAt, revokedAt sql.NullTime
	if err := row.Scan(
		&out.UserID, &out.Email, &out.Username,
		&out.Role, &out.Status,
		&managerID, &managerEmail,
		&rebateRate, &out.RebateMode,
		&out.AffCode, &out.RecruitCount,
		&out.CommissionBalance, &out.CommissionTotal,
		&out.Notes, &out.GrantedAt, &out.UpdatedAt,
		&disabledAt, &disabledByEmail, &out.DisabledReason,
		&revokedAt, &grantedBy,
	); err != nil {
		return nil, err
	}
	if managerID.Valid {
		out.ManagerID = &managerID.Int64
	}
	if managerEmail.Valid {
		out.ManagerEmail = &managerEmail.String
	}
	if rebateRate.Valid {
		out.EffectiveRebateRatePercent = &rebateRate.Float64
	}
	if disabledAt.Valid {
		out.DisabledAt = &disabledAt.Time
	}
	if disabledByEmail.Valid {
		out.DisabledByEmail = &disabledByEmail.String
	}
	if revokedAt.Valid {
		out.RevokedAt = &revokedAt.Time
	}
	if grantedBy.Valid {
		out.GrantedBy = &grantedBy.Int64
	}
	return &out, nil
}

func queryAgentDetail(
	ctx context.Context,
	queryer resellerQueryRower,
	agentUserID, managerID int64,
	includeRevoked bool,
) (*service.AgentDetail, error) {
	row := queryer.QueryRowContext(ctx, `
		SELECT rr.user_id, u.email, COALESCE(u.username, ''),
		       rr.role,
		       CASE WHEN rr.revoked_at IS NOT NULL THEN 'revoked' ELSE rr.status END,
		       rr.manager_id, manager.email,
		       ua.aff_rebate_rate_percent::double precision,
		       CASE
		           WHEN ua.aff_rebate_rate_percent IS NULL THEN 'global'
		           WHEN ua.aff_rebate_rate_percent = 0 THEN 'disabled'
		           ELSE 'custom'
		       END,
		       COALESCE(ua.aff_code, ''), COALESCE(ua.aff_count, 0)::int,
		       TO_CHAR(COALESCE(ua.aff_quota, 0), 'FM999999999999990.00'),
		       TO_CHAR((
		           COALESCE(ua.aff_quota, 0)
		           + COALESCE((
		               SELECT SUM(wr.amount)
		               FROM affiliate_withdraw_requests wr
		               WHERE wr.user_id = rr.user_id AND wr.status = 'approved'
		           ), 0)
		       ), 'FM999999999999990.00'),
		       rr.notes, rr.granted_at, rr.updated_at, rr.disabled_at,
		       disabled_user.email, rr.disabled_reason, rr.revoked_at, rr.granted_by,
		       COALESCE(ua.aff_history_quota, 0)::double precision,
		       (
		           SELECT COUNT(*)
		           FROM affiliate_withdraw_requests pending
		           WHERE pending.user_id = rr.user_id AND pending.status = 'pending'
		       )::int
		  FROM user_reseller_roles rr
		  JOIN users u ON u.id = rr.user_id
		  LEFT JOIN user_affiliates ua ON ua.user_id = rr.user_id
		  LEFT JOIN users manager ON manager.id = rr.manager_id
		  LEFT JOIN users disabled_user ON disabled_user.id = rr.disabled_by
		 WHERE rr.user_id = $1
		   AND ($2 = 0 OR (rr.role = 'agent' AND rr.manager_id = $2))
		   AND ($3 OR rr.revoked_at IS NULL)`,
		agentUserID, managerID, includeRevoked,
	)
	var detail service.AgentDetail
	var managerIDValue, grantedBy sql.NullInt64
	var managerEmail, disabledByEmail sql.NullString
	var rebateRate sql.NullFloat64
	var disabledAt, revokedAt sql.NullTime
	if err := row.Scan(
		&detail.UserID, &detail.Email, &detail.Username,
		&detail.Role, &detail.Status,
		&managerIDValue, &managerEmail,
		&rebateRate, &detail.RebateMode,
		&detail.AffCode, &detail.RecruitCount,
		&detail.CommissionBalance, &detail.CommissionTotal,
		&detail.Notes, &detail.GrantedAt, &detail.UpdatedAt,
		&disabledAt, &disabledByEmail, &detail.DisabledReason,
		&revokedAt, &grantedBy,
		&detail.AffHistoryQuota, &detail.PendingRedemptionCount,
	); err != nil {
		return nil, err
	}
	if managerIDValue.Valid {
		detail.ManagerID = &managerIDValue.Int64
	}
	if managerEmail.Valid {
		detail.ManagerEmail = &managerEmail.String
	}
	if rebateRate.Valid {
		detail.EffectiveRebateRatePercent = &rebateRate.Float64
	}
	if disabledAt.Valid {
		detail.DisabledAt = &disabledAt.Time
	}
	if disabledByEmail.Valid {
		detail.DisabledByEmail = &disabledByEmail.String
	}
	if revokedAt.Valid {
		detail.RevokedAt = &revokedAt.Time
	}
	if grantedBy.Valid {
		detail.GrantedBy = &grantedBy.Int64
	}
	return &detail, nil
}

func lockResellerMutations(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1)`, resellerMutationAdvisoryLock); err != nil {
		return fmt.Errorf("reseller mutation lock: %w", err)
	}
	return nil
}

func lockResellerRole(ctx context.Context, tx *sql.Tx, userID int64) (string, string, error) {
	var role, status string
	var revokedAt sql.NullTime
	err := tx.QueryRowContext(ctx, `
		SELECT role, status, revoked_at
		FROM user_reseller_roles
		WHERE user_id = $1
		FOR UPDATE`, userID).Scan(&role, &status, &revokedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", service.ErrResellerRoleNotFound
	}
	if err != nil {
		return "", "", fmt.Errorf("reseller lock role: %w", err)
	}
	if revokedAt.Valid {
		return "", "", service.ErrResellerStateConflict.WithMetadata(map[string]string{
			"current_status": service.ResellerStatusRevoked,
		})
	}
	return role, status, nil
}

func hasDirectAgents(ctx context.Context, tx *sql.Tx, managerID int64) (bool, error) {
	var exists bool
	if err := tx.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM user_reseller_roles
			WHERE manager_id = $1 AND role = 'agent' AND revoked_at IS NULL
		)`, managerID).Scan(&exists); err != nil {
		return false, fmt.Errorf("reseller check direct agents: %w", err)
	}
	return exists, nil
}

func ensureAgentCanBeRevoked(ctx context.Context, tx *sql.Tx, userID int64) error {
	hasChildren, err := hasDirectAgents(ctx, tx, userID)
	if err != nil {
		return err
	}
	if hasChildren {
		return service.ErrResellerHasDirectAgents
	}
	var pending bool
	if err := tx.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM affiliate_withdraw_requests
			WHERE user_id = $1 AND status = 'pending'
		)`, userID).Scan(&pending); err != nil {
		return fmt.Errorf("reseller check pending withdrawals: %w", err)
	}
	if pending {
		return service.ErrResellerHasPendingWithdraw
	}
	return nil
}

func validateManagerAssignment(ctx context.Context, tx *sql.Tx, targetUserID, managerID int64) error {
	var valid bool
	if err := tx.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM user_reseller_roles
			WHERE user_id = $1
			  AND role = 'agent_manager'
			  AND status = 'active'
			  AND revoked_at IS NULL
		)`, managerID).Scan(&valid); err != nil {
		return fmt.Errorf("reseller validate manager: %w", err)
	}
	if !valid {
		return service.ErrResellerManagerInvalid
	}

	current := managerID
	for depth := 0; depth < 10; depth++ {
		if current == targetUserID {
			return service.ErrResellerManagerCycle
		}
		var parent sql.NullInt64
		err := tx.QueryRowContext(ctx, `
			SELECT manager_id
			FROM user_reseller_roles
			WHERE user_id = $1 AND revoked_at IS NULL`, current).Scan(&parent)
		if errors.Is(err, sql.ErrNoRows) || !parent.Valid {
			return nil
		}
		if err != nil {
			return fmt.Errorf("reseller validate manager hierarchy: %w", err)
		}
		current = parent.Int64
	}
	return service.ErrResellerManagerCycle
}
