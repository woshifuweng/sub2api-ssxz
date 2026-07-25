package service

import (
	"context"
	"encoding/base64"
	"net/http"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var allowedAvatarUploadContentTypes = map[string]struct{}{
	"image/jpeg": {},
	"image/png":  {},
	"image/webp": {},
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
