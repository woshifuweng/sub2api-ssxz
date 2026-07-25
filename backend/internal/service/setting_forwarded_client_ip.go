package service

import (
	"context"
	"encoding/json"
	"fmt"
)

// LoadForwardedClientIPSettings publishes the persisted proxy trust policy.
// Reads fail closed so a database outage cannot accidentally trust spoofed
// forwarding headers.
func (s *SettingService) LoadForwardedClientIPSettings(ctx context.Context) error {
	if s == nil || s.cfg == nil || s.settingRepo == nil {
		return nil
	}

	keys := []string{
		SettingKeyAPIKeyACLTrustForwardedIP,
		SettingKeyForwardedClientIPHeaders,
		settingKeyForwardedClientIPModeV2,
	}
	values, err := s.settingRepo.GetMultiple(ctx, keys)
	if err != nil {
		s.cfg.SetForwardedClientIPSettings(false, nil)
		return fmt.Errorf("get forwarded client ip settings: %w", err)
	}

	runtimeSettings := s.cfg.ForwardedClientIPSettings()
	enabled := runtimeSettings.TrustForwardedIP
	headers := runtimeSettings.Headers
	updates := make(map[string]string)

	trustValue, hasTrustValue := values[SettingKeyAPIKeyACLTrustForwardedIP]
	migrated := values[settingKeyForwardedClientIPModeV2] == "true"
	if hasTrustValue {
		enabled = trustValue == "true"
	}

	if !migrated {
		// Older installs stored false before forwarded-header trust had an
		// explicit migration marker. Preserve compatibility only when the
		// operator has not configured a trusted-proxy policy of their own.
		if hasTrustValue && !enabled && !s.cfg.Server.TrustedProxiesConfigured {
			enabled = true
			updates[SettingKeyAPIKeyACLTrustForwardedIP] = "true"
		}
		updates[settingKeyForwardedClientIPModeV2] = "true"
	}

	if rawHeaders, ok := values[SettingKeyForwardedClientIPHeaders]; ok {
		headers, err = parseForwardedClientIPHeadersSetting(rawHeaders)
		if err != nil {
			s.cfg.SetForwardedClientIPSettings(false, nil)
			if len(updates) > 0 {
				if writeErr := s.settingRepo.SetMultiple(ctx, updates); writeErr != nil {
					return fmt.Errorf("load forwarded client ip headers: %v; migrate forwarded client ip setting: %w", err, writeErr)
				}
			}
			return fmt.Errorf("load forwarded client ip headers: %w", err)
		}
	} else {
		encoded, marshalErr := json.Marshal(headers)
		if marshalErr != nil {
			s.cfg.SetForwardedClientIPSettings(false, nil)
			return fmt.Errorf("marshal forwarded client ip headers: %w", marshalErr)
		}
		updates[SettingKeyForwardedClientIPHeaders] = string(encoded)
	}

	s.cfg.SetForwardedClientIPSettings(enabled, headers)
	if len(updates) == 0 {
		return nil
	}
	if err := s.settingRepo.SetMultiple(ctx, updates); err != nil {
		return fmt.Errorf("migrate forwarded client ip setting: %w", err)
	}
	return nil
}
