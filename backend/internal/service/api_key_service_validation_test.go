//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type apiKeyCreateValidationRepoStub struct {
	quotaBaseAPIKeyRepoStub
	created *APIKey
}

func (s *apiKeyCreateValidationRepoStub) Create(_ context.Context, key *APIKey) error {
	out := *key
	out.IPWhitelist = append([]string(nil), key.IPWhitelist...)
	out.IPBlacklist = append([]string(nil), key.IPBlacklist...)
	out.AllowedModels = append([]string(nil), key.AllowedModels...)
	out.GroupIDs = append([]int64(nil), key.GroupIDs...)
	s.created = &out
	return nil
}

func TestAPIKeyService_Create_RejectsNegativeQuota(t *testing.T) {
	repo := &apiKeyCreateValidationRepoStub{}
	svc := &APIKeyService{
		apiKeyRepo: repo,
		userRepo:   &mockUserRepo{},
		cfg:        &config.Config{},
	}

	_, err := svc.Create(context.Background(), 7, CreateAPIKeyRequest{
		Name:  "negative quota",
		Quota: -0.01,
	})

	require.ErrorIs(t, err, ErrAPIKeyQuotaInvalid)
	require.Nil(t, repo.created)
}

func TestAPIKeyService_Create_RejectsNegativeRateLimit(t *testing.T) {
	tests := []struct {
		name string
		req  CreateAPIKeyRequest
	}{
		{
			name: "5h",
			req:  CreateAPIKeyRequest{Name: "negative 5h", RateLimit5h: -0.01},
		},
		{
			name: "1d",
			req:  CreateAPIKeyRequest{Name: "negative 1d", RateLimit1d: -0.01},
		},
		{
			name: "7d",
			req:  CreateAPIKeyRequest{Name: "negative 7d", RateLimit7d: -0.01},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &apiKeyCreateValidationRepoStub{}
			svc := &APIKeyService{
				apiKeyRepo: repo,
				userRepo:   &mockUserRepo{},
			}

			_, err := svc.Create(context.Background(), 7, tt.req)

			require.ErrorIs(t, err, ErrAPIKeyRateLimitInvalid)
			require.Nil(t, repo.created)
		})
	}
}

func TestAPIKeyService_Create_AllowsZeroLimitsAsUnlimited(t *testing.T) {
	repo := &apiKeyCreateValidationRepoStub{}
	svc := &APIKeyService{
		apiKeyRepo: repo,
		userRepo:   &mockUserRepo{},
		cfg:        &config.Config{},
	}

	got, err := svc.Create(context.Background(), 7, CreateAPIKeyRequest{
		Name:        "zero limits",
		Quota:       0,
		RateLimit5h: 0,
		RateLimit1d: 0,
		RateLimit7d: 0,
	})

	require.NoError(t, err)
	require.NotNil(t, got)
	require.NotNil(t, repo.created)
	require.Zero(t, repo.created.Quota)
	require.Zero(t, repo.created.RateLimit5h)
	require.Zero(t, repo.created.RateLimit1d)
	require.Zero(t, repo.created.RateLimit7d)
}

func TestAPIKeyService_Update_RejectsNegativeQuota(t *testing.T) {
	repo := &apiKeyUpdateRepoStub{
		key: &APIKey{
			ID:     100,
			UserID: 7,
			Key:    "sk-owner-key",
			Name:   "limited",
			Status: StatusAPIKeyActive,
		},
	}
	svc := &APIKeyService{apiKeyRepo: repo}
	quota := -0.01

	_, err := svc.Update(context.Background(), 100, 7, UpdateAPIKeyRequest{Quota: &quota})

	require.ErrorIs(t, err, ErrAPIKeyQuotaInvalid)
	require.Nil(t, repo.updated)
}

func TestAPIKeyService_Update_RejectsNegativeRateLimit(t *testing.T) {
	tests := []struct {
		name string
		req  func(float64) UpdateAPIKeyRequest
	}{
		{
			name: "5h",
			req: func(value float64) UpdateAPIKeyRequest {
				return UpdateAPIKeyRequest{RateLimit5h: &value}
			},
		},
		{
			name: "1d",
			req: func(value float64) UpdateAPIKeyRequest {
				return UpdateAPIKeyRequest{RateLimit1d: &value}
			},
		},
		{
			name: "7d",
			req: func(value float64) UpdateAPIKeyRequest {
				return UpdateAPIKeyRequest{RateLimit7d: &value}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &apiKeyUpdateRepoStub{
				key: &APIKey{
					ID:     100,
					UserID: 7,
					Key:    "sk-owner-key",
					Name:   "limited",
					Status: StatusAPIKeyActive,
				},
			}
			svc := &APIKeyService{apiKeyRepo: repo}
			negative := -0.01

			_, err := svc.Update(context.Background(), 100, 7, tt.req(negative))

			require.ErrorIs(t, err, ErrAPIKeyRateLimitInvalid)
			require.Nil(t, repo.updated)
		})
	}
}

func TestAPIKeyService_Update_AllowsZeroLimitsAsUnlimited(t *testing.T) {
	repo := &apiKeyUpdateRepoStub{
		key: &APIKey{
			ID:          100,
			UserID:      7,
			Key:         "sk-owner-key",
			Name:        "limited",
			Status:      StatusAPIKeyActive,
			Quota:       12,
			RateLimit5h: 3,
			RateLimit1d: 4,
			RateLimit7d: 5,
		},
	}
	svc := &APIKeyService{apiKeyRepo: repo}
	quota := 0.0
	rateLimit5h := 0.0
	rateLimit1d := 0.0
	rateLimit7d := 0.0

	got, err := svc.Update(context.Background(), 100, 7, UpdateAPIKeyRequest{
		Quota:       &quota,
		RateLimit5h: &rateLimit5h,
		RateLimit1d: &rateLimit1d,
		RateLimit7d: &rateLimit7d,
	})

	require.NoError(t, err)
	require.NotNil(t, got)
	require.NotNil(t, repo.updated)
	require.Zero(t, repo.updated.Quota)
	require.Zero(t, repo.updated.RateLimit5h)
	require.Zero(t, repo.updated.RateLimit1d)
	require.Zero(t, repo.updated.RateLimit7d)
}
