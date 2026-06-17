package handler

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

type imageStudioTestGenRepo struct {
	created []*service.SoraGeneration
}

func (r *imageStudioTestGenRepo) Create(_ context.Context, gen *service.SoraGeneration) error {
	gen.ID = int64(len(r.created) + 1)
	gen.CreatedAt = time.Now()
	r.created = append(r.created, gen)
	return nil
}

func (r *imageStudioTestGenRepo) GetByID(context.Context, int64) (*service.SoraGeneration, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *imageStudioTestGenRepo) Update(context.Context, *service.SoraGeneration) error {
	return fmt.Errorf("not implemented")
}

func (r *imageStudioTestGenRepo) Delete(context.Context, int64) error {
	return fmt.Errorf("not implemented")
}

func (r *imageStudioTestGenRepo) List(context.Context, service.SoraGenerationListParams) ([]*service.SoraGeneration, int64, error) {
	return nil, 0, fmt.Errorf("not implemented")
}

func (r *imageStudioTestGenRepo) CountByUserAndStatus(context.Context, int64, []string) (int64, error) {
	return 0, nil
}

func TestExtractImageStudioResultURLs_OnlyReturnsURLResults(t *testing.T) {
	body := []byte(`{
		"created": 123,
		"data": [
			{"url": "https://cdn.example.com/a.png"},
			{"b64_json": "large-image-body"},
			{"url": " "},
			{"url": "https://cdn.example.com/a.png"},
			{"url": "https://cdn.example.com/b.png"}
		]
	}`)

	urls := extractImageStudioResultURLs(body)

	require.Equal(t, []string{
		"https://cdn.example.com/a.png",
		"https://cdn.example.com/b.png",
	}, urls)
}

func TestExtractImageStudioResultURLs_InvalidJSON(t *testing.T) {
	require.Nil(t, extractImageStudioResultURLs([]byte(`not-json`)))
}

func TestImageStudioCaptureContextSuccessRequiresNonTruncated2xx(t *testing.T) {
	capture := &imageStudioCaptureContext{}

	capture.status = http.StatusOK
	require.True(t, capture.success())

	capture.truncated = true
	require.False(t, capture.success())

	capture.truncated = false
	capture.status = http.StatusBadRequest
	require.False(t, capture.success())
}

func TestPersistImageStudioWork_CreatesCompletedImageRecordForURLResults(t *testing.T) {
	repo := &imageStudioTestGenRepo{}
	handler := &ImageStudioHandler{
		genService: service.NewSoraGenerationService(repo, nil, nil),
	}
	capture := &imageStudioCaptureContext{status: http.StatusOK}
	capture.capture([]byte(`{"data":[{"url":"https://cdn.example.com/work.png"}]}`))

	handler.persistImageStudioWork(context.Background(), capture, 7, 42, "product prompt")

	require.Len(t, repo.created, 1)
	gen := repo.created[0]
	require.Equal(t, int64(7), gen.UserID)
	require.NotNil(t, gen.APIKeyID)
	require.Equal(t, int64(42), *gen.APIKeyID)
	require.Equal(t, imageStudioModel, gen.Model)
	require.Equal(t, "product prompt", gen.Prompt)
	require.Equal(t, "image", gen.MediaType)
	require.Equal(t, service.SoraGenStatusCompleted, gen.Status)
	require.Equal(t, service.SoraStorageTypeUpstream, gen.StorageType)
	require.Equal(t, "https://cdn.example.com/work.png", gen.MediaURL)
	require.NotNil(t, gen.CompletedAt)
}

func TestPersistImageStudioWork_DoesNotPersistBase64OnlyResults(t *testing.T) {
	repo := &imageStudioTestGenRepo{}
	handler := &ImageStudioHandler{
		genService: service.NewSoraGenerationService(repo, nil, nil),
	}
	capture := &imageStudioCaptureContext{status: http.StatusOK}
	capture.capture([]byte(`{"data":[{"b64_json":"large-image-body"}]}`))

	handler.persistImageStudioWork(context.Background(), capture, 7, 42, "product prompt")

	require.Empty(t, repo.created)
}
