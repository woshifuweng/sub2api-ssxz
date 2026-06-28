//go:build unit

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type apiKeyUpdateRepoStub struct {
	quotaBaseAPIKeyRepoStub
	key     *APIKey
	updated *APIKey
}

func (s *apiKeyUpdateRepoStub) GetByID(_ context.Context, id int64) (*APIKey, error) {
	if s.key == nil || s.key.ID != id {
		return nil, ErrAPIKeyNotFound
	}
	out := *s.key
	out.IPWhitelist = append([]string(nil), s.key.IPWhitelist...)
	out.IPBlacklist = append([]string(nil), s.key.IPBlacklist...)
	out.AllowedModels = append([]string(nil), s.key.AllowedModels...)
	out.GroupIDs = append([]int64(nil), s.key.GroupIDs...)
	return &out, nil
}

func (s *apiKeyUpdateRepoStub) Update(_ context.Context, key *APIKey) error {
	out := *key
	out.IPWhitelist = append([]string(nil), key.IPWhitelist...)
	out.IPBlacklist = append([]string(nil), key.IPBlacklist...)
	out.AllowedModels = append([]string(nil), key.AllowedModels...)
	out.GroupIDs = append([]int64(nil), key.GroupIDs...)
	s.updated = &out
	return nil
}

type apiKeyRateLimitInvalidatorStub struct {
	invalidatedIDs []int64
}

func ptrTimeForAPIKeyUpdateTest() *time.Time {
	t := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	return &t
}

func (s *apiKeyRateLimitInvalidatorStub) InvalidateAPIKeyRateLimit(_ context.Context, keyID int64) error {
	s.invalidatedIDs = append(s.invalidatedIDs, keyID)
	return nil
}

func TestAPIKeyService_Update_RejectsOwnerMismatch(t *testing.T) {
	repo := &apiKeyUpdateRepoStub{
		key: &APIKey{
			ID:     100,
			UserID: 7,
			Key:    "sk-owner-key",
			Name:   "restricted",
			Status: StatusAPIKeyActive,
		},
	}
	cache := &apiKeyCacheStub{}
	svc := &APIKeyService{apiKeyRepo: repo, cache: cache}
	status := StatusAPIKeyDisabled

	_, err := svc.Update(context.Background(), 100, 8, UpdateAPIKeyRequest{Status: &status})

	require.ErrorIs(t, err, ErrInsufficientPerms)
	require.Nil(t, repo.updated)
	require.Empty(t, cache.invalidated)
	require.Empty(t, cache.deleteAuthKeys)
}

func TestAPIKeyService_Update_InvalidatesAuthCacheByKey(t *testing.T) {
	repo := &apiKeyUpdateRepoStub{
		key: &APIKey{
			ID:     100,
			UserID: 7,
			Key:    "sk-owner-key",
			Name:   "restricted",
			Status: StatusAPIKeyActive,
		},
	}
	cache := &apiKeyCacheStub{}
	svc := &APIKeyService{apiKeyRepo: repo, cache: cache}
	name := "renamed"

	_, err := svc.Update(context.Background(), 100, 7, UpdateAPIKeyRequest{Name: &name})

	require.NoError(t, err)
	require.NotNil(t, repo.updated)
	require.Equal(t, []string{svc.authCacheKey("sk-owner-key")}, cache.deleteAuthKeys)
}

func TestAPIKeyService_Update_ResetRateLimitUsageInvalidatesRateLimitCache(t *testing.T) {
	repo := &apiKeyUpdateRepoStub{
		key: &APIKey{
			ID:             100,
			UserID:         7,
			Key:            "sk-owner-key",
			Name:           "restricted",
			Status:         StatusAPIKeyActive,
			Usage5h:        1.25,
			Usage1d:        2.5,
			Usage7d:        3.75,
			Window5hStart:  ptrTimeForAPIKeyUpdateTest(),
			Window1dStart:  ptrTimeForAPIKeyUpdateTest(),
			Window7dStart:  ptrTimeForAPIKeyUpdateTest(),
		},
	}
	cache := &apiKeyCacheStub{}
	invalidator := &apiKeyRateLimitInvalidatorStub{}
	svc := &APIKeyService{
		apiKeyRepo:            repo,
		cache:                 cache,
		rateLimitCacheInvalid: invalidator,
	}
	reset := true

	got, err := svc.Update(context.Background(), 100, 7, UpdateAPIKeyRequest{ResetRateLimitUsage: &reset})

	require.NoError(t, err)
	require.Zero(t, got.Usage5h)
	require.Zero(t, got.Usage1d)
	require.Zero(t, got.Usage7d)
	require.Nil(t, got.Window5hStart)
	require.Nil(t, got.Window1dStart)
	require.Nil(t, got.Window7dStart)
	require.Equal(t, []string{svc.authCacheKey("sk-owner-key")}, cache.deleteAuthKeys)
	require.Equal(t, []int64{100}, invalidator.invalidatedIDs)
}

func TestAPIKeyService_Update_PreservesIPRestrictionsWhenOmitted(t *testing.T) {
	repo := &apiKeyUpdateRepoStub{
		key: &APIKey{
			ID:          101,
			UserID:      7,
			Key:         "sk-test",
			Name:        "restricted",
			Status:      StatusAPIKeyActive,
			IPWhitelist: []string{"203.0.113.10"},
			IPBlacklist: []string{"198.51.100.20"},
		},
	}
	svc := &APIKeyService{apiKeyRepo: repo}
	status := StatusAPIKeyDisabled

	got, err := svc.Update(context.Background(), 101, 7, UpdateAPIKeyRequest{Status: &status})

	require.NoError(t, err)
	require.NotNil(t, repo.updated)
	require.Equal(t, []string{"203.0.113.10"}, got.IPWhitelist)
	require.Equal(t, []string{"198.51.100.20"}, got.IPBlacklist)
	require.Equal(t, got.IPWhitelist, repo.updated.IPWhitelist)
	require.Equal(t, got.IPBlacklist, repo.updated.IPBlacklist)
}

func TestAPIKeyService_Update_ClearsIPRestrictionsWhenExplicitEmpty(t *testing.T) {
	repo := &apiKeyUpdateRepoStub{
		key: &APIKey{
			ID:          102,
			UserID:      7,
			Key:         "sk-test",
			Name:        "restricted",
			Status:      StatusAPIKeyActive,
			IPWhitelist: []string{"203.0.113.10"},
			IPBlacklist: []string{"198.51.100.20"},
		},
	}
	svc := &APIKeyService{apiKeyRepo: repo}
	emptyWhitelist := []string{}
	emptyBlacklist := []string{}

	got, err := svc.Update(context.Background(), 102, 7, UpdateAPIKeyRequest{
		IPWhitelist: &emptyWhitelist,
		IPBlacklist: &emptyBlacklist,
	})

	require.NoError(t, err)
	require.NotNil(t, repo.updated)
	require.Empty(t, got.IPWhitelist)
	require.Empty(t, got.IPBlacklist)
	require.Empty(t, repo.updated.IPWhitelist)
	require.Empty(t, repo.updated.IPBlacklist)
}

func TestAPIKeyService_Update_RejectsInvalidExplicitIPRestriction(t *testing.T) {
	repo := &apiKeyUpdateRepoStub{
		key: &APIKey{
			ID:     103,
			UserID: 7,
			Key:    "sk-test",
			Name:   "restricted",
			Status: StatusAPIKeyActive,
		},
	}
	svc := &APIKeyService{apiKeyRepo: repo}
	invalidWhitelist := []string{"not-an-ip"}

	_, err := svc.Update(context.Background(), 103, 7, UpdateAPIKeyRequest{
		IPWhitelist: &invalidWhitelist,
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, ErrInvalidIPPattern))
	require.Nil(t, repo.updated)
}
