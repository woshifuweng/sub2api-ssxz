//go:build unit

package service

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestSoraMediaStorage_StoreFromURLs(t *testing.T) {
	tmpDir := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data"))
	}))
	defer server.Close()

	cfg := &config.Config{
		Sora: config.SoraConfig{
			Storage: config.SoraStorageConfig{
				Type:                   "local",
				LocalPath:              tmpDir,
				MaxConcurrentDownloads: 1,
			},
		},
		Security: config.SecurityConfig{
			URLAllowlist: config.URLAllowlistConfig{
				Enabled:           true,
				AllowInsecureHTTP: true,
				AllowPrivateHosts: true,
			},
		},
	}

	storage := NewSoraMediaStorage(cfg)
	urls, err := storage.StoreFromURLs(context.Background(), "image", []string{server.URL + "/img.png"})
	require.NoError(t, err)
	require.Len(t, urls, 1)
	require.True(t, strings.HasPrefix(urls[0], "/image/"))
	require.True(t, strings.HasSuffix(urls[0], ".png"))

	localPath := filepath.Join(tmpDir, filepath.FromSlash(strings.TrimPrefix(urls[0], "/")))
	require.FileExists(t, localPath)
}

func TestSoraMediaStorage_StoreFromURLsBlocksPrivateHostsByDefault(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewSoraMediaStorage(&config.Config{
		Sora: config.SoraConfig{
			Storage: config.SoraStorageConfig{
				Type:      "local",
				LocalPath: tmpDir,
			},
		},
	})

	urls, err := storage.StoreFromURLs(context.Background(), "image", []string{"https://127.0.0.1/img.png"})

	require.Error(t, err)
	require.Nil(t, urls)
	require.Contains(t, err.Error(), "host is not allowed")
}

func TestSoraMediaStorage_StoreFromURLsDoesNotFallbackBlockedPrivateHost(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewSoraMediaStorage(&config.Config{
		Sora: config.SoraConfig{
			Storage: config.SoraStorageConfig{
				Type:               "local",
				LocalPath:          tmpDir,
				FallbackToUpstream: true,
			},
		},
	})

	urls, err := storage.StoreFromURLs(context.Background(), "image", []string{"https://127.0.0.1/img.png"})

	require.Error(t, err)
	require.Nil(t, urls)
	require.Contains(t, err.Error(), "host is not allowed")
}

func TestSoraMediaStorage_StoreFromURLsBlocksMetadataHost(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewSoraMediaStorage(&config.Config{
		Sora: config.SoraConfig{
			Storage: config.SoraStorageConfig{
				Type:      "local",
				LocalPath: tmpDir,
			},
		},
	})

	urls, err := storage.StoreFromURLs(context.Background(), "image", []string{"https://metadata.google.internal/computeMetadata/v1/instance"})

	require.Error(t, err)
	require.Nil(t, urls)
	require.Contains(t, err.Error(), "host is not allowed")
}

func TestSoraMediaStorage_StoreBase64Images(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Sora: config.SoraConfig{
			Storage: config.SoraStorageConfig{
				Type:                   "local",
				LocalPath:              tmpDir,
				MaxConcurrentDownloads: 1,
				MaxDownloadBytes:       1024,
			},
		},
	}

	storage := NewSoraMediaStorage(cfg)
	paths, totalSize, err := storage.StoreBase64Images(context.Background(), []string{
		base64.StdEncoding.EncodeToString([]byte("image-one")),
		base64.StdEncoding.EncodeToString([]byte("image-two")),
	})

	require.NoError(t, err)
	require.Len(t, paths, 2)
	require.Equal(t, int64(len("image-one")+len("image-two")), totalSize)
	for _, p := range paths {
		require.True(t, strings.HasPrefix(p, "/image/"))
		require.True(t, strings.HasSuffix(p, ".png"))
		require.NotContains(t, p, "data:image")
		require.FileExists(t, filepath.Join(tmpDir, filepath.FromSlash(strings.TrimPrefix(p, "/"))))
	}
}

func TestSoraMediaStorage_StoreBase64ImagesRequiresLocalStorage(t *testing.T) {
	storage := NewSoraMediaStorage(&config.Config{})

	paths, totalSize, err := storage.StoreBase64Images(context.Background(), []string{
		base64.StdEncoding.EncodeToString([]byte("image")),
	})

	require.Error(t, err)
	require.Nil(t, paths)
	require.Zero(t, totalSize)
}

func TestSoraMediaStorage_StoreBase64ImagesMaxBytes(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Sora: config.SoraConfig{
			Storage: config.SoraStorageConfig{
				Type:             "local",
				LocalPath:        tmpDir,
				MaxDownloadBytes: 1,
			},
		},
	}

	storage := NewSoraMediaStorage(cfg)
	_, _, err := storage.StoreBase64Images(context.Background(), []string{
		base64.StdEncoding.EncodeToString([]byte("too-large")),
	})

	require.Error(t, err)
}

func TestSoraMediaStorage_FallbackToUpstream(t *testing.T) {
	tmpDir := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := &config.Config{
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
	}

	storage := NewSoraMediaStorage(cfg)
	url := server.URL + "/broken.png"
	urls, err := storage.StoreFromURLs(context.Background(), "image", []string{url})
	require.NoError(t, err)
	require.Equal(t, []string{url}, urls)
}

func TestSoraMediaStorage_MaxDownloadBytes(t *testing.T) {
	tmpDir := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("too-large"))
	}))
	defer server.Close()

	cfg := &config.Config{
		Sora: config.SoraConfig{
			Storage: config.SoraStorageConfig{
				Type:             "local",
				LocalPath:        tmpDir,
				MaxDownloadBytes: 1,
			},
		},
		Security: config.SecurityConfig{
			URLAllowlist: config.URLAllowlistConfig{
				Enabled:           true,
				AllowInsecureHTTP: true,
				AllowPrivateHosts: true,
			},
		},
	}

	storage := NewSoraMediaStorage(cfg)
	_, err := storage.StoreFromURLs(context.Background(), "image", []string{server.URL + "/img.png"})
	require.Error(t, err)
}

func TestNormalizeSoraFileExt(t *testing.T) {
	require.Equal(t, ".png", normalizeSoraFileExt(".PNG"))
	require.Equal(t, ".mp4", normalizeSoraFileExt(".mp4"))
	require.Equal(t, "", normalizeSoraFileExt("../../etc/passwd"))
	require.Equal(t, "", normalizeSoraFileExt(".php"))
}

func TestRemovePartialDownload(t *testing.T) {
	tmpDir := t.TempDir()
	root, err := os.OpenRoot(tmpDir)
	require.NoError(t, err)
	defer func() { _ = root.Close() }()

	filePath := "partial.bin"
	f, err := root.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	require.NoError(t, err)
	_, _ = f.WriteString("partial")
	_ = f.Close()

	removePartialDownload(root, filePath)
	_, err = root.Stat(filePath)
	require.Error(t, err)
	require.True(t, os.IsNotExist(err))
}
