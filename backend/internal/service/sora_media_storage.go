package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/httpclient"
	"github.com/Wei-Shaw/sub2api/internal/util/urlvalidator"
	"github.com/google/uuid"
)

const (
	soraStorageDefaultRoot = "/app/data/sora"
)

var errSoraMediaURLNotAllowed = errors.New("media url host is not allowed")

// SoraMediaStorage 负责下载并落地 Sora 媒体
type SoraMediaStorage struct {
	cfg                *config.Config
	root               string
	imageRoot          string
	videoRoot          string
	downloadTimeout    time.Duration
	maxDownloadBytes   int64
	fallbackToUpstream bool
	debug              bool
	sem                chan struct{}
	ready              bool
}

func NewSoraMediaStorage(cfg *config.Config) *SoraMediaStorage {
	storage := &SoraMediaStorage{cfg: cfg}
	storage.refreshConfig()
	if storage.Enabled() {
		if err := storage.EnsureLocalDirs(); err != nil {
			log.Printf("[SoraStorage] 初始化失败: %v", err)
		}
	}
	return storage
}

func (s *SoraMediaStorage) Enabled() bool {
	if s == nil || s.cfg == nil {
		return false
	}
	return strings.ToLower(strings.TrimSpace(s.cfg.Sora.Storage.Type)) == "local"
}

func (s *SoraMediaStorage) Root() string {
	if s == nil {
		return ""
	}
	return s.root
}

func (s *SoraMediaStorage) ImageRoot() string {
	if s == nil {
		return ""
	}
	return s.imageRoot
}

func (s *SoraMediaStorage) VideoRoot() string {
	if s == nil {
		return ""
	}
	return s.videoRoot
}

func (s *SoraMediaStorage) refreshConfig() {
	if s == nil || s.cfg == nil {
		return
	}
	root := strings.TrimSpace(s.cfg.Sora.Storage.LocalPath)
	if root == "" {
		root = soraStorageDefaultRoot
	}
	root = filepath.Clean(root)
	if !filepath.IsAbs(root) {
		if absRoot, err := filepath.Abs(root); err == nil {
			root = absRoot
		}
	}
	s.root = root
	s.imageRoot = filepath.Join(root, "image")
	s.videoRoot = filepath.Join(root, "video")

	maxConcurrent := s.cfg.Sora.Storage.MaxConcurrentDownloads
	if maxConcurrent <= 0 {
		maxConcurrent = 4
	}
	timeoutSeconds := s.cfg.Sora.Storage.DownloadTimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = 120
	}
	s.downloadTimeout = time.Duration(timeoutSeconds) * time.Second

	maxBytes := s.cfg.Sora.Storage.MaxDownloadBytes
	if maxBytes <= 0 {
		maxBytes = 200 << 20
	}
	s.maxDownloadBytes = maxBytes
	s.fallbackToUpstream = s.cfg.Sora.Storage.FallbackToUpstream
	s.debug = s.cfg.Sora.Storage.Debug
	s.sem = make(chan struct{}, maxConcurrent)
}

// EnsureLocalDirs 创建并校验本地目录
func (s *SoraMediaStorage) EnsureLocalDirs() error {
	if s == nil || !s.Enabled() {
		return nil
	}
	if err := os.MkdirAll(s.imageRoot, 0o755); err != nil {
		return fmt.Errorf("create image dir: %w", err)
	}
	if err := os.MkdirAll(s.videoRoot, 0o755); err != nil {
		return fmt.Errorf("create video dir: %w", err)
	}
	s.ready = true
	return nil
}

// StoreFromURLs 下载并存储媒体，返回相对路径或回退 URL
func (s *SoraMediaStorage) StoreFromURLs(ctx context.Context, mediaType string, urls []string) ([]string, error) {
	if len(urls) == 0 {
		return nil, nil
	}
	if s == nil || !s.Enabled() {
		return urls, nil
	}
	if !s.ready {
		if err := s.EnsureLocalDirs(); err != nil {
			return nil, err
		}
	}
	results := make([]string, 0, len(urls))
	for _, raw := range urls {
		relative, err := s.downloadAndStore(ctx, mediaType, raw)
		if err != nil {
			if errors.Is(err, errSoraMediaURLNotAllowed) {
				return nil, err
			}
			if s.fallbackToUpstream {
				results = append(results, raw)
				continue
			}
			return nil, err
		}
		results = append(results, relative)
	}
	return results, nil
}

// StoreBase64Images decodes generated image payloads into local media files.
// It intentionally returns no data URLs, so large image bodies are not stored in DB metadata.
func (s *SoraMediaStorage) StoreBase64Images(ctx context.Context, images []string) ([]string, int64, error) {
	if len(images) == 0 {
		return nil, 0, nil
	}
	if s == nil || !s.Enabled() {
		return nil, 0, errors.New("local image storage is not enabled")
	}
	if !s.ready {
		if err := s.EnsureLocalDirs(); err != nil {
			return nil, 0, err
		}
	}
	results := make([]string, 0, len(images))
	var total int64
	for _, raw := range images {
		relative, size, err := s.storeBase64Image(ctx, raw)
		if err != nil {
			return nil, 0, err
		}
		results = append(results, relative)
		total += size
	}
	return results, total, nil
}

// TotalSizeByRelativePaths 统计本地存储路径总大小（仅统计 /image 和 /video 路径）。
func (s *SoraMediaStorage) TotalSizeByRelativePaths(paths []string) (int64, error) {
	if s == nil || len(paths) == 0 {
		return 0, nil
	}
	var total int64
	for _, p := range paths {
		localPath, err := s.resolveLocalPath(p)
		if err != nil {
			continue
		}
		info, err := os.Stat(localPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return 0, err
		}
		if info.Mode().IsRegular() {
			total += info.Size()
		}
	}
	return total, nil
}

// DeleteByRelativePaths 删除本地媒体路径（仅删除 /image 和 /video 路径）。
func (s *SoraMediaStorage) DeleteByRelativePaths(paths []string) error {
	if s == nil || len(paths) == 0 {
		return nil
	}
	var lastErr error
	for _, p := range paths {
		localPath, err := s.resolveLocalPath(p)
		if err != nil {
			continue
		}
		if err := os.Remove(localPath); err != nil && !os.IsNotExist(err) {
			lastErr = err
		}
	}
	return lastErr
}

func (s *SoraMediaStorage) resolveLocalPath(relativePath string) (string, error) {
	if s == nil || strings.TrimSpace(relativePath) == "" {
		return "", errors.New("empty path")
	}
	cleaned := path.Clean(relativePath)
	if !strings.HasPrefix(cleaned, "/image/") && !strings.HasPrefix(cleaned, "/video/") {
		return "", errors.New("not a local media path")
	}
	if strings.TrimSpace(s.root) == "" {
		return "", errors.New("storage root not configured")
	}
	relative := strings.TrimPrefix(cleaned, "/")
	return filepath.Join(s.root, filepath.FromSlash(relative)), nil
}

func (s *SoraMediaStorage) downloadAndStore(ctx context.Context, mediaType, rawURL string) (string, error) {
	if strings.TrimSpace(rawURL) == "" {
		return "", errors.New("empty url")
	}
	root := s.imageRoot
	if mediaType == "video" {
		root = s.videoRoot
	}
	if root == "" {
		return "", errors.New("storage root not configured")
	}

	retries := 3
	for attempt := 1; attempt <= retries; attempt++ {
		release, err := s.acquire(ctx)
		if err != nil {
			return "", err
		}
		relative, err := s.downloadOnce(ctx, root, mediaType, rawURL)
		release()
		if err == nil {
			return relative, nil
		}
		if errors.Is(err, errSoraMediaURLNotAllowed) {
			return "", err
		}
		if s.debug {
			log.Printf("[SoraStorage] 下载失败(%d/%d): %s err=%v", attempt, retries, sanitizeMediaLogURL(rawURL), err)
		}
		if attempt < retries {
			time.Sleep(time.Duration(attempt*attempt) * time.Second)
			continue
		}
		return "", err
	}
	return "", errors.New("download retries exhausted")
}

func (s *SoraMediaStorage) storeBase64Image(ctx context.Context, raw string) (string, int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", 0, errors.New("empty image data")
	}
	if s.imageRoot == "" {
		return "", 0, errors.New("storage root not configured")
	}
	release, err := s.acquire(ctx)
	if err != nil {
		return "", 0, err
	}
	defer release()

	storageRoot, err := os.OpenRoot(s.imageRoot)
	if err != nil {
		return "", 0, err
	}
	defer func() { _ = storageRoot.Close() }()

	datePath := time.Now().Format("2006/01/02")
	datePathFS := filepath.FromSlash(datePath)
	if err := storageRoot.MkdirAll(datePathFS, 0o755); err != nil {
		return "", 0, err
	}
	filename := uuid.NewString() + ".png"
	filePath := filepath.Join(datePathFS, filename)
	out, err := storageRoot.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return "", 0, err
	}
	defer func() { _ = out.Close() }()

	decoder := base64.NewDecoder(base64.StdEncoding, strings.NewReader(raw))
	limited := io.LimitReader(decoder, s.maxDownloadBytes+1)
	written, err := io.Copy(out, limited)
	if err != nil {
		removePartialDownload(storageRoot, filePath)
		return "", 0, err
	}
	if s.maxDownloadBytes > 0 && written > s.maxDownloadBytes {
		removePartialDownload(storageRoot, filePath)
		return "", 0, fmt.Errorf("image size exceeds limit: %d", written)
	}

	relative := path.Join("/", "image", datePath, filename)
	return relative, written, nil
}

func (s *SoraMediaStorage) downloadOnce(ctx context.Context, root, mediaType, rawURL string) (string, error) {
	downloadURL, err := s.validateDownloadURL(rawURL)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return "", err
	}
	client, err := s.downloadClient()
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		return "", fmt.Errorf("download failed: %d %s", resp.StatusCode, string(body))
	}

	ext := normalizeSoraFileExt(fileExtFromURL(downloadURL))
	if ext == "" {
		ext = normalizeSoraFileExt(fileExtFromContentType(resp.Header.Get("Content-Type")))
	}
	if ext == "" {
		ext = ".bin"
	}
	if s.maxDownloadBytes > 0 && resp.ContentLength > s.maxDownloadBytes {
		return "", fmt.Errorf("download size exceeds limit: %d", resp.ContentLength)
	}

	storageRoot, err := os.OpenRoot(root)
	if err != nil {
		return "", err
	}
	defer func() { _ = storageRoot.Close() }()

	datePath := time.Now().Format("2006/01/02")
	datePathFS := filepath.FromSlash(datePath)
	if err := storageRoot.MkdirAll(datePathFS, 0o755); err != nil {
		return "", err
	}
	filename := uuid.NewString() + ext
	filePath := filepath.Join(datePathFS, filename)
	out, err := storageRoot.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return "", err
	}
	defer func() { _ = out.Close() }()

	limited := io.LimitReader(resp.Body, s.maxDownloadBytes+1)
	written, err := io.Copy(out, limited)
	if err != nil {
		removePartialDownload(storageRoot, filePath)
		return "", err
	}
	if s.maxDownloadBytes > 0 && written > s.maxDownloadBytes {
		removePartialDownload(storageRoot, filePath)
		return "", fmt.Errorf("download size exceeds limit: %d", written)
	}

	relative := path.Join("/", mediaType, datePath, filename)
	if s.debug {
		log.Printf("[SoraStorage] 已落地 %s -> %s", sanitizeMediaLogURL(rawURL), relative)
	}
	return relative, nil
}

func (s *SoraMediaStorage) validateDownloadURL(rawURL string) (string, error) {
	allowInsecureHTTP, allowPrivateHosts := s.mediaDownloadURLPolicy()
	normalized, err := urlvalidator.ValidateHTTPURL(rawURL, allowInsecureHTTP, urlvalidator.ValidationOptions{
		AllowPrivate: allowPrivateHosts,
	})
	if err != nil {
		return "", fmt.Errorf("%w: %v", errSoraMediaURLNotAllowed, err)
	}
	if allowPrivateHosts {
		return normalized, nil
	}

	parsed, err := url.Parse(normalized)
	if err != nil {
		return "", err
	}
	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	if host == "" {
		return "", errors.New("invalid host")
	}
	if soraMediaDownloadHostBlocked(host) {
		return "", fmt.Errorf("%w: %s", errSoraMediaURLNotAllowed, host)
	}
	return normalized, nil
}

func (s *SoraMediaStorage) downloadClient() (*http.Client, error) {
	_, allowPrivateHosts := s.mediaDownloadURLPolicy()
	return httpclient.GetClient(httpclient.Options{
		Timeout:            s.downloadTimeout,
		ValidateResolvedIP: true,
		AllowPrivateHosts:  allowPrivateHosts,
	})
}

func (s *SoraMediaStorage) mediaDownloadURLPolicy() (allowInsecureHTTP bool, allowPrivateHosts bool) {
	if s == nil || s.cfg == nil || !s.cfg.Security.URLAllowlist.Enabled {
		return false, false
	}
	return s.cfg.Security.URLAllowlist.AllowInsecureHTTP, s.cfg.Security.URLAllowlist.AllowPrivateHosts
}

func soraMediaDownloadHostBlocked(host string) bool {
	normalized := strings.ToLower(strings.TrimSpace(host))
	if normalized == "" {
		return true
	}
	if strings.HasSuffix(normalized, ".localhost") || strings.HasSuffix(normalized, ".metadata.google.internal") {
		return true
	}
	if _, blocked := soraBlockedHostnames[normalized]; blocked {
		return true
	}
	return false
}

func (s *SoraMediaStorage) acquire(ctx context.Context) (func(), error) {
	if s.sem == nil {
		return func() {}, nil
	}
	select {
	case s.sem <- struct{}{}:
		return func() { <-s.sem }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func fileExtFromURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	ext := path.Ext(parsed.Path)
	return strings.ToLower(ext)
}

func fileExtFromContentType(ct string) string {
	if ct == "" {
		return ""
	}
	if exts, err := mime.ExtensionsByType(ct); err == nil && len(exts) > 0 {
		return strings.ToLower(exts[0])
	}
	return ""
}

func normalizeSoraFileExt(ext string) string {
	ext = strings.ToLower(strings.TrimSpace(ext))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp", ".svg", ".tif", ".tiff", ".heic",
		".mp4", ".mov", ".webm", ".m4v", ".avi", ".mkv", ".3gp", ".flv":
		return ext
	default:
		return ""
	}
}

func removePartialDownload(root *os.Root, filePath string) {
	if root == nil || strings.TrimSpace(filePath) == "" {
		return
	}
	_ = root.Remove(filePath)
}

// sanitizeMediaLogURL 脱敏 URL 用于日志记录（去除 query 参数中可能的 token 信息）
func sanitizeMediaLogURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		if len(rawURL) > 80 {
			return rawURL[:80] + "..."
		}
		return rawURL
	}
	safe := parsed.Scheme + "://" + parsed.Host + parsed.Path
	if len(safe) > 120 {
		return safe[:120] + "..."
	}
	return safe
}
