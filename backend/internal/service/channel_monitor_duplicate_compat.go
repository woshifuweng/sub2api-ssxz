package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
)

const maxChannelMonitorNameRunes = 100
const ChannelMonitorDuplicateOperationIDMetadataKey = "sub2api:duplicate_operation_id"

// Duplicate creates an independent, disabled copy of an existing monitor.
// The API key stays server-side: it is decrypted only long enough to encrypt a
// fresh ciphertext for the new row. Runtime state and history are not copied.
func (s *ChannelMonitorService) Duplicate(
	ctx context.Context,
	id, createdBy int64,
	actorScope, operationKey string,
) (*ChannelMonitor, error) {
	operationID := duplicateChannelMonitorOperationID(id, actorScope, operationKey)
	existing, err := s.RecoverDuplicate(ctx, id, actorScope, operationKey)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	source, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	plainAPIKey, err := s.decryptAPIKeyForDuplicate(source)
	if err != nil {
		return nil, err
	}
	encryptedAPIKey, err := s.encryptor.Encrypt(plainAPIKey)
	if err != nil {
		return nil, fmt.Errorf("encrypt duplicate channel monitor api key: %w", err)
	}
	bodyOverride, err := cloneChannelMonitorJSONMap(source.BodyOverride)
	if err != nil {
		return nil, fmt.Errorf("clone duplicate channel monitor body override: %w", err)
	}

	duplicate := &ChannelMonitor{
		Name:                 duplicateChannelMonitorName(source.Name),
		Provider:             source.Provider,
		APIMode:              source.APIMode,
		Endpoint:             source.Endpoint,
		APIKey:               encryptedAPIKey,
		PrimaryModel:         source.PrimaryModel,
		ExtraModels:          append([]string{}, source.ExtraModels...),
		GroupName:            source.GroupName,
		Enabled:              false,
		IntervalSeconds:      source.IntervalSeconds,
		JitterSeconds:        source.JitterSeconds,
		CreatedBy:            createdBy,
		TemplateID:           cloneInt64Pointer(source.TemplateID),
		ExtraHeaders:         cloneChannelMonitorHeaders(source.ExtraHeaders),
		BodyOverrideMode:     source.BodyOverrideMode,
		BodyOverride:         bodyOverride,
		DuplicateOperationID: operationID,
	}
	if err := s.repo.Create(ctx, duplicate); err != nil {
		return nil, fmt.Errorf("duplicate channel monitor: %w", err)
	}

	// Match Create/Update response semantics: repository receives ciphertext,
	// while handlers receive plaintext only so they can return the masked form.
	duplicate.APIKey = plainAPIKey
	return duplicate, nil
}

// RecoverDuplicate performs a read-only lookup for a duplicate that was
// already committed for the same actor, source monitor, and idempotency key.
// It deliberately never repeats the create side effect.
func (s *ChannelMonitorService) RecoverDuplicate(
	ctx context.Context,
	id int64,
	actorScope, operationKey string,
) (*ChannelMonitor, error) {
	operationID := duplicateChannelMonitorOperationID(id, actorScope, operationKey)
	if operationID == "" {
		return nil, nil
	}
	monitor, err := s.repo.FindByDuplicateOperationID(ctx, operationID)
	if err != nil {
		return nil, fmt.Errorf("find duplicate channel monitor operation: %w", err)
	}
	if monitor == nil {
		return nil, nil
	}
	s.decryptInPlace(monitor)
	return monitor, nil
}

func duplicateChannelMonitorOperationID(sourceID int64, actorScope, operationKey string) string {
	operationKey = strings.TrimSpace(operationKey)
	if operationKey == "" {
		return ""
	}
	actorScope = strings.TrimSpace(actorScope)
	if actorScope == "" {
		actorScope = "admin:0"
	}
	payload := "admin.channel_monitors.duplicate\x00" + actorScope + "\x00" + strconv.FormatInt(sourceID, 10) + "\x00" + operationKey
	digest := sha256.Sum256([]byte(payload))
	return fmt.Sprintf("%x", digest)
}

func (s *ChannelMonitorService) decryptAPIKeyForDuplicate(source *ChannelMonitor) (string, error) {
	if source == nil || strings.TrimSpace(source.APIKey) == "" {
		return "", ErrChannelMonitorAPIKeyDecryptFailed
	}
	plain, err := s.encryptor.Decrypt(source.APIKey)
	if err != nil || strings.TrimSpace(plain) == "" {
		slog.Warn("channel_monitor: decrypt api key for duplicate failed",
			"monitor_id", source.ID, "error", err)
		return "", ErrChannelMonitorAPIKeyDecryptFailed
	}
	return plain, nil
}

func duplicateChannelMonitorName(sourceName string) string {
	const suffix = " (Copy)"
	nameRunes := []rune(strings.TrimSpace(sourceName))
	maxBaseRunes := maxChannelMonitorNameRunes - len([]rune(suffix))
	if len(nameRunes) > maxBaseRunes {
		nameRunes = nameRunes[:maxBaseRunes]
	}
	return string(nameRunes) + suffix
}

func cloneInt64Pointer(value *int64) *int64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneChannelMonitorHeaders(source map[string]string) map[string]string {
	if source == nil {
		return map[string]string{}
	}
	cloned := make(map[string]string, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func cloneChannelMonitorJSONMap(source map[string]any) (map[string]any, error) {
	if source == nil {
		return nil, nil
	}
	payload, err := json.Marshal(source)
	if err != nil {
		return nil, err
	}
	cloned := make(map[string]any, len(source))
	if err := json.Unmarshal(payload, &cloned); err != nil {
		return nil, err
	}
	return cloned, nil
}
