package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (r *opsRepository) InsertRetryAttempt(ctx context.Context, input *service.OpsInsertRetryAttemptInput) (int64, error) {
	if r == nil || r.db == nil {
		return 0, fmt.Errorf("nil ops repository")
	}
	if input == nil || input.SourceErrorID <= 0 || strings.TrimSpace(input.Mode) == "" {
		return 0, fmt.Errorf("invalid retry attempt input")
	}
	const query = `
INSERT INTO ops_retry_attempts (
  requested_by_user_id, source_error_id, mode, pinned_account_id, status, started_at
) VALUES ($1,$2,$3,$4,$5,$6)
RETURNING id`
	var id int64
	err := r.db.QueryRowContext(ctx, query,
		opsNullInt64(&input.RequestedByUserID), input.SourceErrorID,
		strings.TrimSpace(input.Mode), opsNullInt64(input.PinnedAccountID),
		strings.TrimSpace(input.Status), input.StartedAt,
	).Scan(&id)
	return id, err
}

func (r *opsRepository) UpdateRetryAttempt(ctx context.Context, input *service.OpsUpdateRetryAttemptInput) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("nil ops repository")
	}
	if input == nil || input.ID <= 0 {
		return fmt.Errorf("invalid retry attempt input")
	}
	const query = `
UPDATE ops_retry_attempts SET
  status=$2, finished_at=$3, duration_ms=$4, success=$5,
  http_status_code=$6, upstream_request_id=$7, used_account_id=$8,
  response_preview=$9, response_truncated=$10, result_request_id=$11,
  result_error_id=$12, error_message=$13
WHERE id=$1`
	_, err := r.db.ExecContext(ctx, query,
		input.ID, strings.TrimSpace(input.Status), retryNullTime(input.FinishedAt),
		input.DurationMs, retryNullBool(input.Success), nullInt(input.HTTPStatusCode),
		opsNullString(input.UpstreamRequestID), nullInt64(input.UsedAccountID),
		opsNullString(input.ResponsePreview), retryNullBool(input.ResponseTruncated),
		opsNullString(input.ResultRequestID), nullInt64(input.ResultErrorID),
		opsNullString(input.ErrorMessage),
	)
	return err
}

func (r *opsRepository) GetLatestRetryAttemptForError(ctx context.Context, sourceErrorID int64) (*service.OpsRetryAttempt, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("nil ops repository")
	}
	if sourceErrorID <= 0 {
		return nil, fmt.Errorf("invalid source_error_id")
	}
	const query = `
SELECT r.id, r.created_at, COALESCE(r.requested_by_user_id,0), r.source_error_id,
  COALESCE(r.mode,''), r.pinned_account_id, '' AS pinned_account_name,
  COALESCE(r.status,''), r.started_at, r.finished_at, r.duration_ms, r.success,
  r.http_status_code, r.upstream_request_id, r.used_account_id,
  '' AS used_account_name, r.response_preview, r.response_truncated,
  r.result_request_id, r.result_error_id, r.error_message
FROM ops_retry_attempts r
WHERE r.source_error_id=$1
ORDER BY r.created_at DESC LIMIT 1`
	return scanOpsRetryAttempt(r.db.QueryRowContext(ctx, query, sourceErrorID))
}

func (r *opsRepository) ListRetryAttemptsByErrorID(ctx context.Context, sourceErrorID int64, limit int) ([]*service.OpsRetryAttempt, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("nil ops repository")
	}
	if sourceErrorID <= 0 {
		return nil, fmt.Errorf("invalid source_error_id")
	}
	if limit <= 0 {
		limit = 50
	} else if limit > 200 {
		limit = 200
	}
	const query = `
SELECT r.id, r.created_at, COALESCE(r.requested_by_user_id,0), r.source_error_id,
  COALESCE(r.mode,''), r.pinned_account_id, COALESCE(pa.name,''),
  COALESCE(r.status,''), r.started_at, r.finished_at, r.duration_ms, r.success,
  r.http_status_code, r.upstream_request_id, r.used_account_id, COALESCE(ua.name,''),
  r.response_preview, r.response_truncated, r.result_request_id, r.result_error_id,
  r.error_message
FROM ops_retry_attempts r
LEFT JOIN accounts pa ON pa.id=r.pinned_account_id
LEFT JOIN accounts ua ON ua.id=r.used_account_id
WHERE r.source_error_id=$1
ORDER BY r.created_at DESC LIMIT $2`
	rows, err := r.db.QueryContext(ctx, query, sourceErrorID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]*service.OpsRetryAttempt, 0)
	for rows.Next() {
		item, scanErr := scanOpsRetryAttempt(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

type opsRetryScanner interface {
	Scan(dest ...any) error
}

func scanOpsRetryAttempt(scanner opsRetryScanner) (*service.OpsRetryAttempt, error) {
	var out service.OpsRetryAttempt
	var requestedBy, pinnedAccountID, durationMs, httpStatusCode, usedAccountID, resultErrorID sql.NullInt64
	var startedAt, finishedAt sql.NullTime
	var success, responseTruncated sql.NullBool
	var upstreamRequestID, responsePreview, resultRequestID, errorMessage sql.NullString
	if err := scanner.Scan(
		&out.ID, &out.CreatedAt, &requestedBy, &out.SourceErrorID, &out.Mode,
		&pinnedAccountID, &out.PinnedAccountName, &out.Status, &startedAt, &finishedAt,
		&durationMs, &success, &httpStatusCode, &upstreamRequestID, &usedAccountID,
		&out.UsedAccountName, &responsePreview, &responseTruncated, &resultRequestID,
		&resultErrorID, &errorMessage,
	); err != nil {
		return nil, err
	}
	out.RequestedByUserID = requestedBy.Int64
	if pinnedAccountID.Valid {
		v := pinnedAccountID.Int64
		out.PinnedAccountID = &v
	}
	if startedAt.Valid {
		v := startedAt.Time
		out.StartedAt = &v
	}
	if finishedAt.Valid {
		v := finishedAt.Time
		out.FinishedAt = &v
	}
	if durationMs.Valid {
		v := durationMs.Int64
		out.DurationMs = &v
	}
	if success.Valid {
		v := success.Bool
		out.Success = &v
	}
	if httpStatusCode.Valid {
		v := int(httpStatusCode.Int64)
		out.HTTPStatusCode = &v
	}
	if upstreamRequestID.Valid {
		v := upstreamRequestID.String
		out.UpstreamRequestID = &v
	}
	if usedAccountID.Valid {
		v := usedAccountID.Int64
		out.UsedAccountID = &v
	}
	if responsePreview.Valid {
		v := responsePreview.String
		out.ResponsePreview = &v
	}
	if responseTruncated.Valid {
		v := responseTruncated.Bool
		out.ResponseTruncated = &v
	}
	if resultRequestID.Valid {
		v := resultRequestID.String
		out.ResultRequestID = &v
	}
	if resultErrorID.Valid {
		v := resultErrorID.Int64
		out.ResultErrorID = &v
	}
	if errorMessage.Valid {
		v := errorMessage.String
		out.ErrorMessage = &v
	}
	return &out, nil
}

func retryNullTime(value time.Time) sql.NullTime {
	if value.IsZero() {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: value, Valid: true}
}

func retryNullBool(value *bool) sql.NullBool {
	if value == nil {
		return sql.NullBool{}
	}
	return sql.NullBool{Bool: *value, Valid: true}
}
