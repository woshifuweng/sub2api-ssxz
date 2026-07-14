package repository

import (
	"context"
	"database/sql/driver"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

type redactedAuditJSON struct {
	required string
}

func (m redactedAuditJSON) Match(value driver.Value) bool {
	raw, ok := value.(string)
	if !ok {
		bytes, bytesOK := value.([]byte)
		if !bytesOK {
			return false
		}
		raw = string(bytes)
	}
	return strings.Contains(raw, m.required) && !strings.Contains(raw, "sk-admin-secret")
}

func TestAPIKeyRepositoryAdminSetEnabledPersistsAuditInSameTransaction(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := &apiKeyRepository{sql: db}
	now := time.Now().UTC()

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT.*k\.user_id.*k\.key.*FROM api_keys k.*FOR UPDATE OF k`).
		WithArgs(int64(41)).
		WillReturnRows(sqlmock.NewRows([]string{
			"user_id", "key", "name", "status", "group_id", "expires_at", "quota", "quota_used",
		}).AddRow(int64(9), "sk-admin-secret-never-audit", "CC Switch", service.StatusAPIKeyActive, nil, now.Add(time.Hour), 10.0, 1.0))
	mock.ExpectExec(`UPDATE api_keys SET status = \$2`).
		WithArgs(int64(41), service.StatusAPIKeyDisabled).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO admin_audit_logs`).
		WithArgs(
			int64(42), service.AdminAPIKeyAuditActionDisable, int64(41), int64(9),
			redactedAuditJSON{required: `"status":"active"`},
			redactedAuditJSON{required: `"status":"disabled"`},
		).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	result, err := repo.AdminMutateAPIKey(context.Background(), service.AdminAPIKeyMutationCommand{
		APIKeyID:    41,
		ActorUserID: 42,
		Action:      service.AdminAPIKeyAuditActionDisable,
	})

	require.NoError(t, err)
	require.Equal(t, int64(41), result.APIKeyID)
	require.Equal(t, int64(9), result.UserID)
	require.Equal(t, service.StatusAPIKeyDisabled, result.Status)
	require.Equal(t, "sk-admin-secret-never-audit", result.AuthenticationKey)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAPIKeyRepositoryAdminDeletePersistsAuditBeforeSoftDeleteCommit(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := &apiKeyRepository{sql: db}

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT.*FROM api_keys k.*FOR UPDATE OF k`).
		WithArgs(int64(52)).
		WillReturnRows(sqlmock.NewRows([]string{
			"user_id", "key", "name", "status", "group_id", "expires_at", "quota", "quota_used",
		}).AddRow(int64(10), "sk-admin-secret-delete", "Temporary", service.StatusAPIKeyDisabled, int64(12), nil, 0.0, 0.0))
	mock.ExpectExec(`UPDATE api_keys SET deleted_at = NOW\(\), updated_at = NOW\(\)`).
		WithArgs(int64(52)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO admin_audit_logs`).
		WithArgs(
			int64(42), service.AdminAPIKeyAuditActionDelete, int64(52), int64(10),
			redactedAuditJSON{required: `"deleted":false`},
			redactedAuditJSON{required: `"deleted":true`},
		).
		WillReturnResult(sqlmock.NewResult(2, 1))
	mock.ExpectCommit()

	result, err := repo.AdminMutateAPIKey(context.Background(), service.AdminAPIKeyMutationCommand{
		APIKeyID:    52,
		ActorUserID: 42,
		Action:      service.AdminAPIKeyAuditActionDelete,
	})

	require.NoError(t, err)
	require.True(t, result.Deleted)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAPIKeyRepositoryAdminMutationRollsBackWhenAuditInsertFails(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	repo := &apiKeyRepository{sql: db}

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT.*FROM api_keys k.*FOR UPDATE OF k`).
		WithArgs(int64(41)).
		WillReturnRows(sqlmock.NewRows([]string{
			"user_id", "key", "name", "status", "group_id", "expires_at", "quota", "quota_used",
		}).AddRow(int64(9), "sk-admin-secret-never-audit", "CC Switch", service.StatusAPIKeyActive, nil, nil, 10.0, 1.0))
	mock.ExpectExec(`UPDATE api_keys SET status = \$2`).
		WithArgs(int64(41), service.StatusAPIKeyDisabled).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO admin_audit_logs`).
		WillReturnError(assertionError("audit table unavailable"))
	mock.ExpectRollback()

	_, err = repo.AdminMutateAPIKey(context.Background(), service.AdminAPIKeyMutationCommand{
		APIKeyID:    41,
		ActorUserID: 42,
		Action:      service.AdminAPIKeyAuditActionDisable,
	})

	require.Error(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

type assertionError string

func (e assertionError) Error() string { return string(e) }
