//go:build unit

package service

import (
	"context"
	"errors"
	"testing"

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
