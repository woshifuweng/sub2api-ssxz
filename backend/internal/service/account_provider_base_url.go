package service

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/util/urlvalidator"
)

func NormalizeProviderBaseURL(raw string) (string, error) {
	normalized, err := urlvalidator.ValidateHTTPURL(raw, false, urlvalidator.ValidationOptions{AllowPrivate: false})
	if err != nil {
		return "", err
	}

	parsed, err := url.Parse(normalized)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid url: %s", strings.TrimSpace(raw))
	}
	if parsed.User != nil {
		return "", errors.New("base_url cannot contain userinfo")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("base_url cannot contain query or fragment")
	}

	return strings.TrimRight(normalized, "/"), nil
}

func NormalizeAccountCredentialsBaseURL(platform, accountType string, credentials map[string]any) error {
	if platform == PlatformSora && accountType == AccountTypeAPIKey {
		raw, _ := credentialBaseURLValue(credentials)
		normalized, err := NormalizeSoraAPIKeyBaseURL(raw)
		if err != nil {
			return fmt.Errorf("base_url invalid: %w", err)
		}
		if credentials == nil {
			return errors.New("base_url is required")
		}
		credentials["base_url"] = normalized
		return nil
	}

	if credentials == nil {
		return nil
	}
	raw, exists := credentialBaseURLValue(credentials)
	if !exists {
		return nil
	}
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	normalized, err := NormalizeProviderBaseURL(raw)
	if err != nil {
		return fmt.Errorf("base_url invalid: %w", err)
	}
	credentials["base_url"] = normalized
	return nil
}

func credentialBaseURLValue(credentials map[string]any) (string, bool) {
	if credentials == nil {
		return "", false
	}
	raw, exists := credentials["base_url"]
	if !exists || raw == nil {
		return "", false
	}
	if value, ok := raw.(string); ok {
		return value, true
	}
	return fmt.Sprintf("%v", raw), true
}
