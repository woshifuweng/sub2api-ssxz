package handler

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type imageStudioTestGenRepo struct {
	created []*service.SoraGeneration
}

func newImageStudioMultipartContext(t *testing.T, fields map[string]string) gatewayctx.GatewayContext {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for name, value := range fields {
		require.NoError(t, writer.WriteField(name, value))
	}
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/image-studio/generate", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = req
	return gatewayctx.FromGin(ctx)
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

	handler.persistImageStudioWork(context.Background(), capture, 7, 42, imageStudioModel, "product prompt")

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

func TestPersistImageStudioWork_StoresURLResultsToLocalStorage(t *testing.T) {
	tmpDir := t.TempDir()
	imageBody := []byte("url-image-data")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(imageBody)
	}))
	defer server.Close()

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
		Security: config.SecurityConfig{
			URLAllowlist: config.URLAllowlistConfig{
				Enabled:           true,
				AllowInsecureHTTP: true,
				AllowPrivateHosts: true,
			},
		},
	})
	handler := &ImageStudioHandler{
		genService:   service.NewSoraGenerationService(repo, nil, nil),
		mediaStorage: mediaStorage,
	}
	capture := &imageStudioCaptureContext{status: http.StatusOK}
	capture.capture([]byte(fmt.Sprintf(`{"data":[{"url":%q}]}`, server.URL+"/work.png")))

	handler.persistImageStudioWork(context.Background(), capture, 7, 42, imageStudioModel, "product prompt")

	require.Len(t, repo.created, 1)
	gen := repo.created[0]
	require.Equal(t, service.SoraStorageTypeLocal, gen.StorageType)
	require.True(t, strings.HasPrefix(gen.MediaURL, "/image/"))
	require.True(t, strings.HasSuffix(gen.MediaURL, ".png"))
	require.Equal(t, int64(len(imageBody)), gen.FileSizeBytes)
	localPath := filepath.Join(tmpDir, filepath.FromSlash(strings.TrimPrefix(gen.MediaURL, "/")))
	require.FileExists(t, localPath)
	content, err := os.ReadFile(localPath)
	require.NoError(t, err)
	require.Equal(t, imageBody, content)
}

func TestPersistImageStudioWork_KeepsUpstreamStorageWhenURLStorageFallsBack(t *testing.T) {
	tmpDir := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	repo := &imageStudioTestGenRepo{}
	mediaStorage := service.NewSoraMediaStorage(&config.Config{
		Sora: config.SoraConfig{
			Storage: config.SoraStorageConfig{
				Type:               "local",
				LocalPath:          tmpDir,
				FallbackToUpstream: true,
			},
		},
		Security: config.SecurityConfig{
			URLAllowlist: config.URLAllowlistConfig{
				Enabled:           true,
				AllowInsecureHTTP: true,
				AllowPrivateHosts: true,
			},
		},
	})
	handler := &ImageStudioHandler{
		genService:   service.NewSoraGenerationService(repo, nil, nil),
		mediaStorage: mediaStorage,
	}
	upstreamURL := server.URL + "/work.png"
	capture := &imageStudioCaptureContext{status: http.StatusOK}
	capture.capture([]byte(fmt.Sprintf(`{"data":[{"url":%q}]}`, upstreamURL)))

	handler.persistImageStudioWork(context.Background(), capture, 7, 42, imageStudioModel, "product prompt")

	require.Len(t, repo.created, 1)
	gen := repo.created[0]
	require.Equal(t, service.SoraStorageTypeUpstream, gen.StorageType)
	require.Equal(t, upstreamURL, gen.MediaURL)
	require.Equal(t, []string{upstreamURL}, gen.MediaURLs)
	require.Zero(t, gen.FileSizeBytes)
}

func TestPersistImageStudioWork_RecordsRequestedImageModel(t *testing.T) {
	repo := &imageStudioTestGenRepo{}
	handler := &ImageStudioHandler{
		genService: service.NewSoraGenerationService(repo, nil, nil),
	}
	capture := &imageStudioCaptureContext{status: http.StatusOK}
	capture.capture([]byte(`{"data":[{"url":"https://cdn.example.com/work.png"}]}`))

	handler.persistImageStudioWork(context.Background(), capture, 7, 42, "gemini-2.5-flash-image", "product prompt")

	require.Len(t, repo.created, 1)
	require.Equal(t, "gemini-2.5-flash-image", repo.created[0].Model)
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

	handler.persistImageStudioWork(context.Background(), capture, 7, 42, imageStudioModel, "product prompt")

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

	handler.persistImageStudioWork(context.Background(), capture, 7, 42, imageStudioModel, "product prompt")

	require.Empty(t, repo.created)
}

func TestParseImageStudioRequest_DefaultsModelWhenMissing(t *testing.T) {
	ctx := newImageStudioMultipartContext(t, map[string]string{
		"template_id": "background",
	})

	req, err := parseImageStudioRequest(ctx)

	require.NoError(t, err)
	require.Equal(t, imageStudioModel, req.Model)
}

func TestParseImageStudioRequest_UsesSubmittedModel(t *testing.T) {
	ctx := newImageStudioMultipartContext(t, map[string]string{
		"model": " gemini-2.5-flash-image ",
	})

	req, err := parseImageStudioRequest(ctx)

	require.NoError(t, err)
	require.Equal(t, "gemini-2.5-flash-image", req.Model)
}

func TestParseImageStudioRequest_AcceptsThreeImageCount(t *testing.T) {
	ctx := newImageStudioMultipartContext(t, map[string]string{
		"count": " 3 ",
	})

	req, err := parseImageStudioRequest(ctx)

	require.NoError(t, err)
	require.Equal(t, 3, req.Count)
}

func TestBuildImageStudioGatewayBody_UsesRequestedModel(t *testing.T) {
	req := &imageStudioRequest{
		Model: "gemini-2.5-flash-image",
		Size:  "1024x1024",
		Count: 3,
	}

	body, contentType, endpoint, err := buildImageStudioGatewayBody(req, "product prompt")

	require.NoError(t, err)
	require.Equal(t, "application/json", contentType)
	require.Equal(t, "/v1/images/generations", endpoint)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	require.Equal(t, "gemini-2.5-flash-image", payload["model"])
	require.Equal(t, float64(3), payload["n"])
}

func TestBuildImageStudioGatewayBody_EditUsesRequestedModel(t *testing.T) {
	req := &imageStudioRequest{
		Model:    "gemini-2.5-flash-image",
		Size:     "1024x1024",
		Count:    1,
		FileName: "reference.png",
		FileType: "image/png",
		FileData: []byte("png-data"),
	}

	body, contentType, endpoint, err := buildImageStudioGatewayBody(req, "product prompt")

	require.NoError(t, err)
	require.Contains(t, contentType, "multipart/form-data")
	require.Equal(t, "/v1/images/edits", endpoint)
	require.Contains(t, string(body), `name="model"`)
	require.Contains(t, string(body), "gemini-2.5-flash-image")
}
