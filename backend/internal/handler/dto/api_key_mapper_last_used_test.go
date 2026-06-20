package dto

import (
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestAPIKeyFromService_MapsLastUsedAt(t *testing.T) {
	lastUsed := time.Now().UTC().Truncate(time.Second)
	src := &service.APIKey{
		ID:         1,
		UserID:     2,
		Key:        "sk-map-last-used",
		Name:       "Mapper",
		Status:     service.StatusActive,
		LastUsedAt: &lastUsed,
	}

	out := APIKeyFromService(src)
	require.NotNil(t, out)
	require.NotNil(t, out.LastUsedAt)
	require.WithinDuration(t, lastUsed, *out.LastUsedAt, time.Second)
}

func TestAPIKeyFromService_MapsNilLastUsedAt(t *testing.T) {
	src := &service.APIKey{
		ID:     1,
		UserID: 2,
		Key:    "sk-map-last-used-nil",
		Name:   "MapperNil",
		Status: service.StatusActive,
	}

	out := APIKeyFromService(src)
	require.NotNil(t, out)
	require.Nil(t, out.LastUsedAt)
}

func TestAPIKeyFromService_MasksKeyByDefault(t *testing.T) {
	src := &service.APIKey{
		ID:     1,
		UserID: 2,
		Key:    "sk-test-secret-value-123456",
		Name:   "Masked",
		Status: service.StatusActive,
	}

	out := APIKeyFromService(src)
	require.NotNil(t, out)
	require.Equal(t, "sk-test-...3456", out.Key)
	require.NotContains(t, out.Key, "secret-value")
	require.NotEqual(t, src.Key, out.Key)
}

func TestAPIKeyFromService_MasksShortKeyByDefault(t *testing.T) {
	src := &service.APIKey{
		ID:     1,
		UserID: 2,
		Key:    "short-key",
		Name:   "ShortMasked",
		Status: service.StatusActive,
	}

	out := APIKeyFromService(src)
	require.NotNil(t, out)
	require.Equal(t, "[redacted]", out.Key)
}

func TestAPIKeyFromServiceWithPlaintextKey_ReturnsFullKeyForCreateOnly(t *testing.T) {
	src := &service.APIKey{
		ID:     1,
		UserID: 2,
		Key:    "sk-test-secret-value-123456",
		Name:   "Create",
		Status: service.StatusActive,
	}

	out := APIKeyFromServiceWithPlaintextKey(src)
	require.NotNil(t, out)
	require.Equal(t, src.Key, out.Key)
}

func TestAPIKeyForSafeReplay_MasksPlaintextCreateKey(t *testing.T) {
	src := &APIKey{
		ID:            1,
		UserID:        2,
		Key:           "sk-test-secret-value-123456",
		Name:          "Create",
		Status:        service.StatusActive,
		GroupIDs:      []int64{1, 2},
		AllowedModels: []string{"gpt-image-2"},
		IPWhitelist:   []string{"203.0.113.10"},
		IPBlacklist:   []string{"198.51.100.20"},
	}

	out := APIKeyForSafeReplay(src)
	require.NotNil(t, out)
	require.Equal(t, "sk-test-...3456", out.Key)
	require.NotContains(t, out.Key, "secret-value")
	require.Equal(t, src.ID, out.ID)
	require.Equal(t, src.Name, out.Name)

	out.GroupIDs[0] = 99
	out.AllowedModels[0] = "mutated"
	out.IPWhitelist[0] = "mutated"
	out.IPBlacklist[0] = "mutated"

	require.Equal(t, []int64{1, 2}, src.GroupIDs)
	require.Equal(t, []string{"gpt-image-2"}, src.AllowedModels)
	require.Equal(t, []string{"203.0.113.10"}, src.IPWhitelist)
	require.Equal(t, []string{"198.51.100.20"}, src.IPBlacklist)
}
