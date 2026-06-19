package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

func TestPersistImageStudioWork_SQLRepoPersistsOnlySuccessfulCaptures(t *testing.T) {
	tests := []struct {
		name        string
		capture     *imageStudioCaptureContext
		body        string
		wantRecords int
	}{
		{
			name:        "upstream_4xx",
			capture:     &imageStudioCaptureContext{status: http.StatusBadRequest},
			body:        `{"data":[{"url":"https://cdn.example.com/failed-4xx.png"}]}`,
			wantRecords: 0,
		},
		{
			name:        "upstream_5xx",
			capture:     &imageStudioCaptureContext{status: http.StatusInternalServerError},
			body:        `{"data":[{"url":"https://cdn.example.com/failed-5xx.png"}]}`,
			wantRecords: 0,
		},
		{
			name:        "truncated_success_body",
			capture:     &imageStudioCaptureContext{status: http.StatusOK, truncated: true},
			body:        `{"data":[{"url":"https://cdn.example.com/truncated.png"}]}`,
			wantRecords: 0,
		},
		{
			name:        "successful_image_response",
			capture:     &imageStudioCaptureContext{status: http.StatusOK},
			body:        `{"data":[{"url":"https://cdn.example.com/work.png"}]}`,
			wantRecords: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newImageStudioHistorySQLite(t)
			handler := &ImageStudioHandler{
				genService: service.NewSoraGenerationService(&imageStudioSQLGenRepo{db: db}, nil, nil),
			}
			tt.capture.capture([]byte(tt.body))

			handler.persistImageStudioWork(context.Background(), tt.capture, 7, 42, "product prompt")

			var count int
			require.NoError(t, db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM sora_generations").Scan(&count))
			require.Equal(t, tt.wantRecords, count)

			if tt.wantRecords == 0 {
				return
			}
			var model, status, mediaURL, storageType string
			var userID, apiKeyID int64
			require.NoError(t, db.QueryRowContext(context.Background(), `
				SELECT user_id, api_key_id, model, status, media_url, storage_type
				FROM sora_generations
				LIMIT 1
			`).Scan(&userID, &apiKeyID, &model, &status, &mediaURL, &storageType))
			require.Equal(t, int64(7), userID)
			require.Equal(t, int64(42), apiKeyID)
			require.Equal(t, imageStudioModel, model)
			require.Equal(t, service.SoraGenStatusCompleted, status)
			require.Equal(t, "https://cdn.example.com/work.png", mediaURL)
			require.Equal(t, service.SoraStorageTypeUpstream, storageType)
		})
	}
}

func newImageStudioHistorySQLite(t *testing.T) *sql.DB {
	t.Helper()

	dsn := "file:" + strings.NewReplacer("/", "_", "\\", "_", " ", "_").Replace(t.Name()) + "?mode=memory&cache=shared"
	db, err := sql.Open("sqlite", dsn)
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(`
		CREATE TABLE sora_generations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			api_key_id INTEGER NULL,
			model TEXT NOT NULL DEFAULT '',
			prompt TEXT NOT NULL DEFAULT '',
			media_type TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT '',
			media_url TEXT NOT NULL DEFAULT '',
			media_urls BLOB,
			file_size_bytes INTEGER NOT NULL DEFAULT 0,
			storage_type TEXT NOT NULL DEFAULT '',
			s3_object_keys BLOB,
			upstream_task_id TEXT NOT NULL DEFAULT '',
			error_message TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			completed_at TIMESTAMP NULL
		)
	`)
	require.NoError(t, err)

	return db
}

type imageStudioSQLGenRepo struct {
	db *sql.DB
}

func (r *imageStudioSQLGenRepo) Create(ctx context.Context, gen *service.SoraGeneration) error {
	mediaURLsJSON, _ := json.Marshal(gen.MediaURLs)
	s3KeysJSON, _ := json.Marshal(gen.S3ObjectKeys)
	return r.db.QueryRowContext(ctx, `
		INSERT INTO sora_generations (
			user_id, api_key_id, model, prompt, media_type,
			status, media_url, media_urls, file_size_bytes,
			storage_type, s3_object_keys, upstream_task_id, error_message,
			completed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id, created_at
	`,
		gen.UserID, gen.APIKeyID, gen.Model, gen.Prompt, gen.MediaType,
		gen.Status, gen.MediaURL, mediaURLsJSON, gen.FileSizeBytes,
		gen.StorageType, s3KeysJSON, gen.UpstreamTaskID, gen.ErrorMessage, gen.CompletedAt,
	).Scan(&gen.ID, &gen.CreatedAt)
}

func (r *imageStudioSQLGenRepo) GetByID(context.Context, int64) (*service.SoraGeneration, error) {
	return nil, errors.New("not implemented")
}

func (r *imageStudioSQLGenRepo) Update(context.Context, *service.SoraGeneration) error {
	return errors.New("not implemented")
}

func (r *imageStudioSQLGenRepo) Delete(context.Context, int64) error {
	return errors.New("not implemented")
}

func (r *imageStudioSQLGenRepo) List(context.Context, service.SoraGenerationListParams) ([]*service.SoraGeneration, int64, error) {
	return nil, 0, errors.New("not implemented")
}

func (r *imageStudioSQLGenRepo) CountByUserAndStatus(context.Context, int64, []string) (int64, error) {
	return 0, errors.New("not implemented")
}

var _ service.SoraGenerationRepository = (*imageStudioSQLGenRepo)(nil)
