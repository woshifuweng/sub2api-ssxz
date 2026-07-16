package service

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var (
	ErrAvatarInvalid  = infraerrors.BadRequest("AVATAR_INVALID", "avatar must be a valid image data URL or http(s) URL")
	ErrAvatarTooLarge = infraerrors.BadRequest("AVATAR_TOO_LARGE", "avatar image must be 100KB or smaller")
	ErrAvatarNotImage = infraerrors.BadRequest("AVATAR_NOT_IMAGE", "avatar content must be an image")
)

const maxInlineAvatarBytes = 100 * 1024

var allowedAvatarUploadContentTypes = map[string]struct{}{
	"image/jpeg": {},
	"image/png":  {},
	"image/webp": {},
}

type UserAvatar struct {
	StorageProvider string `json:"storage_provider"`
	StorageKey      string `json:"storage_key"`
	URL             string `json:"url"`
	ContentType     string `json:"content_type"`
	ByteSize        int    `json:"byte_size"`
	SHA256          string `json:"sha256"`
}

type UpsertUserAvatarInput struct {
	StorageProvider string
	StorageKey      string
	URL             string
	ContentType     string
	ByteSize        int
	SHA256          string
}

type userAvatarRepo interface {
	UpsertUserAvatar(ctx context.Context, userID int64, input UpsertUserAvatarInput) (*UserAvatar, error)
	DeleteUserAvatar(ctx context.Context, userID int64) error
}

func ValidateUserAvatar(raw string) error {
	_, err := normalizeUserAvatarInput(raw)
	return err
}

// ValidateUserAvatarUpload is the narrower contract used by the authenticated
// self-service avatar endpoint. Existing OAuth adoption may still store a
// remote URL, but user uploads must be cropped/re-encoded data URLs.
func ValidateUserAvatarUpload(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" || !strings.HasPrefix(raw, "data:") {
		return ErrAvatarInvalid
	}

	input, err := normalizeInlineUserAvatarInput(raw)
	if err != nil {
		return err
	}
	if _, ok := allowedAvatarUploadContentTypes[strings.ToLower(input.ContentType)]; !ok {
		return ErrAvatarNotImage
	}

	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(strings.SplitN(raw, ",", 2)[1]))
	if err != nil || len(decoded) == 0 {
		return ErrAvatarInvalid
	}
	detected := http.DetectContentType(decoded)
	if !strings.EqualFold(detected, input.ContentType) {
		return ErrAvatarNotImage
	}
	return nil
}

func normalizeUserAvatarInput(raw string) (UpsertUserAvatarInput, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return UpsertUserAvatarInput{}, ErrAvatarInvalid
	}
	if strings.HasPrefix(raw, "data:") {
		return normalizeInlineUserAvatarInput(raw)
	}

	parsed, err := url.Parse(raw)
	if err != nil || parsed == nil {
		return UpsertUserAvatarInput{}, ErrAvatarInvalid
	}
	if !strings.EqualFold(parsed.Scheme, "http") && !strings.EqualFold(parsed.Scheme, "https") {
		return UpsertUserAvatarInput{}, ErrAvatarInvalid
	}
	if strings.TrimSpace(parsed.Host) == "" {
		return UpsertUserAvatarInput{}, ErrAvatarInvalid
	}

	return UpsertUserAvatarInput{
		StorageProvider: "remote_url",
		URL:             raw,
	}, nil
}

func normalizeInlineUserAvatarInput(raw string) (UpsertUserAvatarInput, error) {
	body := strings.TrimPrefix(raw, "data:")
	meta, encoded, ok := strings.Cut(body, ",")
	if !ok {
		return UpsertUserAvatarInput{}, ErrAvatarInvalid
	}
	meta = strings.TrimSpace(meta)
	encoded = strings.TrimSpace(encoded)
	if !strings.HasSuffix(strings.ToLower(meta), ";base64") {
		return UpsertUserAvatarInput{}, ErrAvatarInvalid
	}

	contentType := strings.TrimSpace(meta[:len(meta)-len(";base64")])
	if contentType == "" || !strings.HasPrefix(strings.ToLower(contentType), "image/") {
		return UpsertUserAvatarInput{}, ErrAvatarNotImage
	}

	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return UpsertUserAvatarInput{}, ErrAvatarInvalid
	}
	if len(decoded) > maxInlineAvatarBytes {
		return UpsertUserAvatarInput{}, ErrAvatarTooLarge
	}

	sum := sha256.Sum256(decoded)
	return UpsertUserAvatarInput{
		StorageProvider: "inline",
		URL:             raw,
		ContentType:     contentType,
		ByteSize:        len(decoded),
		SHA256:          hex.EncodeToString(sum[:]),
	}, nil
}

func (s *UserService) SetAvatar(ctx context.Context, userID int64, raw string) (*UserAvatar, error) {
	repo, ok := s.userRepo.(userAvatarRepo)
	if !ok {
		return nil, infraerrors.ServiceUnavailable("USER_AVATAR_NOT_SUPPORTED", "user avatar storage is not configured")
	}

	avatarValue := strings.TrimSpace(raw)
	if avatarValue == "" {
		if err := repo.DeleteUserAvatar(ctx, userID); err != nil {
			return nil, fmt.Errorf("delete avatar: %w", err)
		}
		return nil, nil
	}

	avatarInput, err := normalizeUserAvatarInput(avatarValue)
	if err != nil {
		return nil, err
	}

	avatar, err := repo.UpsertUserAvatar(ctx, userID, avatarInput)
	if err != nil {
		return nil, fmt.Errorf("upsert avatar: %w", err)
	}
	return avatar, nil
}

type userAvatarReader interface {
	GetUserAvatar(ctx context.Context, userID int64) (*UserAvatar, error)
}

// GetAvatar reads only the avatar belonging to the supplied authenticated user.
// The repository returns (nil, nil) when the user has no custom avatar.
func (s *UserService) GetAvatar(ctx context.Context, userID int64) (*UserAvatar, error) {
	repo, ok := s.userRepo.(userAvatarReader)
	if !ok {
		return nil, infraerrors.ServiceUnavailable("USER_AVATAR_NOT_SUPPORTED", "user avatar storage is not configured")
	}
	return repo.GetUserAvatar(ctx, userID)
}
