package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type sqlTxBeginner interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

type adminAPIKeyLockedState struct {
	userID    int64
	key       string
	name      string
	status    string
	groupID   sql.NullInt64
	expiresAt sql.NullTime
	quota     float64
	quotaUsed float64
}

func (r *apiKeyRepository) AdminMutateAPIKey(ctx context.Context, command service.AdminAPIKeyMutationCommand) (*service.AdminAPIKeyMutationRecord, error) {
	beginner, ok := r.sql.(sqlTxBeginner)
	if !ok {
		return nil, fmt.Errorf("admin API key mutation transaction is unavailable")
	}
	if command.ActorUserID <= 0 {
		return nil, service.ErrAdminActorRequired
	}

	tx, err := beginner.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin admin API key mutation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	state, err := lockAdminAPIKey(ctx, tx, command.APIKeyID)
	if err != nil {
		return nil, err
	}

	before := adminAPIKeyAuditSnapshot(state, false)
	after := cloneAdminAPIKeyAuditSnapshot(before)
	record := &service.AdminAPIKeyMutationRecord{
		APIKeyID: command.APIKeyID, UserID: state.userID,
		AuthenticationKey: state.key, Status: state.status,
	}
	if state.groupID.Valid {
		gid := state.groupID.Int64
		record.GroupID = &gid
	}

	switch command.Action {
	case service.AdminAPIKeyAuditActionEnable, service.AdminAPIKeyAuditActionDisable:
		status := service.StatusAPIKeyDisabled
		if command.Action == service.AdminAPIKeyAuditActionEnable {
			status = service.StatusAPIKeyActive
		}
		if err := updateAdminAPIKeyStatus(ctx, tx, command.APIKeyID, status); err != nil {
			return nil, err
		}
		record.Status = status
		after["status"] = status

	case service.AdminAPIKeyAuditActionChangeGroup:
		if command.GroupID == nil || *command.GroupID < 0 {
			return nil, service.ErrAdminAPIKeyInvalidAction
		}
		groupID, groupName, autoGranted, err := changeAdminAPIKeyGroup(ctx, tx, state.userID, command.APIKeyID, *command.GroupID)
		if err != nil {
			return nil, err
		}
		record.GroupID = groupID
		record.GroupName = groupName
		record.AutoGrantedGroupAccess = autoGranted
		if groupID == nil {
			after["group_id"] = nil
		} else {
			after["group_id"] = *groupID
		}
		if autoGranted {
			after["auto_granted_group_access"] = true
		}

	case service.AdminAPIKeyAuditActionDelete:
		result, err := tx.ExecContext(ctx, `
UPDATE api_keys SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL`, command.APIKeyID)
		if err != nil {
			return nil, fmt.Errorf("soft delete admin API key: %w", err)
		}
		if err := requireOneAffectedRow(result); err != nil {
			return nil, err
		}
		record.Deleted = true
		after["deleted"] = true

	default:
		return nil, service.ErrAdminAPIKeyInvalidAction
	}

	if err := insertAdminAuditLog(ctx, tx, command, state.userID, before, after); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit admin API key mutation: %w", err)
	}
	return record, nil
}

func lockAdminAPIKey(ctx context.Context, tx *sql.Tx, keyID int64) (*adminAPIKeyLockedState, error) {
	row := tx.QueryRowContext(ctx, `
SELECT k.user_id, k.key, k.name, k.status, k.group_id, k.expires_at, k.quota, k.quota_used
FROM api_keys k
JOIN users u ON u.id = k.user_id AND u.deleted_at IS NULL
WHERE k.id = $1 AND k.deleted_at IS NULL
FOR UPDATE OF k`, keyID)
	var state adminAPIKeyLockedState
	if err := row.Scan(
		&state.userID, &state.key, &state.name, &state.status, &state.groupID,
		&state.expiresAt, &state.quota, &state.quotaUsed,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrAPIKeyNotFound
		}
		return nil, fmt.Errorf("lock admin API key: %w", err)
	}
	return &state, nil
}

func updateAdminAPIKeyStatus(ctx context.Context, tx *sql.Tx, keyID int64, status string) error {
	result, err := tx.ExecContext(ctx, `
UPDATE api_keys SET status = $2, updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL`, keyID, status)
	if err != nil {
		return fmt.Errorf("update admin API key status: %w", err)
	}
	return requireOneAffectedRow(result)
}

func changeAdminAPIKeyGroup(ctx context.Context, tx *sql.Tx, userID, keyID, targetGroupID int64) (*int64, string, bool, error) {
	if targetGroupID == 0 {
		var resultingGroupID sql.NullInt64
		err := tx.QueryRowContext(ctx, `
WITH normalized AS (
  SELECT
    CASE
      WHEN cardinality(COALESCE(group_ids, ARRAY[]::bigint[])) > 0
        THEN COALESCE(group_ids, ARRAY[]::bigint[])
      WHEN group_id IS NOT NULL THEN ARRAY[group_id]::bigint[]
      ELSE ARRAY[]::bigint[]
    END AS ids,
    COALESCE(group_id, group_ids[1]) AS remove_id
  FROM api_keys
  WHERE id = $1 AND deleted_at IS NULL
), remaining AS (
  SELECT COALESCE(array_remove(ids, remove_id), ARRAY[]::bigint[]) AS ids
  FROM normalized
)
UPDATE api_keys AS k
SET group_ids = remaining.ids,
    group_id = CASE
      WHEN cardinality(remaining.ids) > 0 THEN remaining.ids[1]
      ELSE NULL
    END,
    updated_at = NOW()
FROM remaining
WHERE k.id = $1 AND k.deleted_at IS NULL
RETURNING k.group_id`, keyID).Scan(&resultingGroupID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, "", false, service.ErrAPIKeyNotFound
			}
			return nil, "", false, fmt.Errorf("unbind admin API key group: %w", err)
		}
		if resultingGroupID.Valid {
			groupID := resultingGroupID.Int64
			return &groupID, "", false, nil
		}
		return nil, "", false, nil
	}

	var groupName, status, subscriptionType string
	var exclusive bool
	err := tx.QueryRowContext(ctx, `
SELECT name, status, is_exclusive, subscription_type
FROM groups
WHERE id = $1 AND deleted_at IS NULL
FOR SHARE`, targetGroupID).Scan(&groupName, &status, &exclusive, &subscriptionType)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "", false, service.ErrGroupNotFound
		}
		return nil, "", false, fmt.Errorf("load target API key group: %w", err)
	}
	if status != service.StatusActive {
		return nil, "", false, service.ErrAdminAPIKeyGroupInactive
	}
	if subscriptionType == service.SubscriptionTypeSubscription {
		var allowed bool
		err = tx.QueryRowContext(ctx, `
SELECT EXISTS (
  SELECT 1 FROM user_subscriptions
  WHERE user_id = $1 AND group_id = $2 AND status = 'active'
    AND deleted_at IS NULL AND starts_at <= NOW() AND expires_at > NOW()
)`, userID, targetGroupID).Scan(&allowed)
		if err != nil {
			return nil, "", false, fmt.Errorf("validate API key group subscription: %w", err)
		}
		if !allowed {
			return nil, "", false, service.ErrAdminAPIKeySubscription
		}
	}

	autoGranted := false
	if exclusive && subscriptionType != service.SubscriptionTypeSubscription {
		result, err := tx.ExecContext(ctx, `
INSERT INTO user_allowed_groups (user_id, group_id, created_at)
VALUES ($1, $2, NOW())
ON CONFLICT (user_id, group_id) DO NOTHING`, userID, targetGroupID)
		if err != nil {
			return nil, "", false, fmt.Errorf("grant API key group access: %w", err)
		}
		if affected, affectedErr := result.RowsAffected(); affectedErr == nil && affected > 0 {
			autoGranted = true
		}
	}

	result, err := tx.ExecContext(ctx, `
UPDATE api_keys SET group_id = $2, group_ids = ARRAY[$2]::bigint[], updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL`, keyID, targetGroupID)
	if err != nil {
		return nil, "", false, fmt.Errorf("change admin API key group: %w", err)
	}
	if err := requireOneAffectedRow(result); err != nil {
		return nil, "", false, err
	}
	groupID := targetGroupID
	return &groupID, groupName, autoGranted, nil
}

func requireOneAffectedRow(result sql.Result) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return service.ErrAPIKeyNotFound
	}
	return nil
}

func adminAPIKeyAuditSnapshot(state *adminAPIKeyLockedState, deleted bool) map[string]any {
	var groupID any
	if state.groupID.Valid {
		groupID = state.groupID.Int64
	}
	return map[string]any{
		"name": state.name, "status": state.status, "group_id": groupID, "deleted": deleted,
	}
}

func cloneAdminAPIKeyAuditSnapshot(source map[string]any) map[string]any {
	clone := make(map[string]any, len(source)+1)
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func insertAdminAuditLog(
	ctx context.Context,
	tx *sql.Tx,
	command service.AdminAPIKeyMutationCommand,
	targetUserID int64,
	before, after map[string]any,
) error {
	beforeJSON, err := json.Marshal(before)
	if err != nil {
		return fmt.Errorf("encode admin audit before values: %w", err)
	}
	afterJSON, err := json.Marshal(after)
	if err != nil {
		return fmt.Errorf("encode admin audit after values: %w", err)
	}
	_, err = tx.ExecContext(ctx, `
INSERT INTO admin_audit_logs (
  actor_user_id, actor_role, action, resource_type, resource_id,
  target_user_id, before_values, after_values
)
VALUES ($1, 'admin', $2, 'api_key', $3, $4, $5::jsonb, $6::jsonb)`,
		command.ActorUserID, command.Action, command.APIKeyID, targetUserID,
		string(beforeJSON), string(afterJSON),
	)
	if err != nil {
		return fmt.Errorf("insert admin audit log: %w", err)
	}
	return nil
}
