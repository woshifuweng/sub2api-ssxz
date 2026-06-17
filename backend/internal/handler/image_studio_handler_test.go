package handler

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
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

func TestExtractImageStudioResultBase64(t *testing.T) {
	body := []byte(`{
		"data": [
			{"b64_json": "Zmlyc3Q="},
			{"url": "https://cdn.example.com/a.png"},
			{"b64_json": " "},
			{"b64_json": "c2Vjb25k"}
		]
	}`)

	require.Equal(t, []string{"Zmlyc3Q=", "c2Vjb25k"}, extractImageStudioResultBase64(body))
	require.Nil(t, extractImageStudioResultBase64([]byte(`not-json`)))
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

func TestPersistImageStudioWork_PersistsBase64OnlyResultsToLocalStorage(t *testing.T) {
	tmpDir := t.TempDir()
	repo := &imageStudioTestGenRepo{}
	mediaStorage := service.NewSoraMediaStorage(&config.Config{
		Sora: config.SoraConfig{
			Storage: config.SoraStorageConfig{
				Type:                   "local",
				LocalPath:              tmpDir,
				MaxConcurrentDownloads: 1,
				MaxDownloadBytes:       1024,
			},
		},
	})
	handler := &ImageStudioHandler{
		genService:   service.NewSoraGenerationService(repo, nil, nil),
		mediaStorage: mediaStorage,
	}
	capture := &imageStudioCaptureContext{status: http.StatusOK}
	capture.capture([]byte(fmt.Sprintf(`{"data":[{"b64_json":%q}]}`, base64.StdEncoding.EncodeToString([]byte("png-data")))))

	handler.persistImageStudioWork(context.Background(), capture, 7, 42, "product prompt")

	require.Len(t, repo.created, 1)
	gen := repo.created[0]
	require.Equal(t, service.SoraStorageTypeLocal, gen.StorageType)
	require.True(t, strings.HasPrefix(gen.MediaURL, "/image/"))
	require.True(t, strings.HasSuffix(gen.MediaURL, ".png"))
	require.Equal(t, int64(len("png-data")), gen.FileSizeBytes)
	localPath := filepath.Join(tmpDir, filepath.FromSlash(strings.TrimPrefix(gen.MediaURL, "/")))
	require.FileExists(t, localPath)
	content, err := os.ReadFile(localPath)
	require.NoError(t, err)
	require.Equal(t, []byte("png-data"), content)
}

func TestPersistImageStudioWork_DoesNotStoreBase64InRecordWhenLocalStorageDisabled(t *testing.T) {
	repo := &imageStudioTestGenRepo{}
	handler := &ImageStudioHandler{
		genService: service.NewSoraGenerationService(repo, nil, nil),
	}
	capture := &imageStudioCaptureContext{status: http.StatusOK}
	capture.capture([]byte(fmt.Sprintf(`{"data":[{"b64_json":%q}]}`, base64.StdEncoding.EncodeToString([]byte("png-data")))))

	handler.persistImageStudioWork(context.Background(), capture, 7, 42, "product prompt")

	require.Empty(t, repo.created)
}
