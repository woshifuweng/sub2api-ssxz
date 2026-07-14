package service

import (
	"context"
	"errors"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type adminAPIKeyListRepoStub struct {
	APIKeyRepository
	got    AdminAPIKeyListParams
	result *AdminAPIKeyListResult
	err    error
}

func (s *adminAPIKeyListRepoStub) ListAdminAPIKeys(_ context.Context, params AdminAPIKeyListParams) (*AdminAPIKeyListResult, error) {
	s.got = params
	return s.result, s.err
}

func TestAdminServiceListAdminAPIKeysNormalizesQuery(t *testing.T) {
	want := &AdminAPIKeyListResult{Summary: AdminAPIKeyListSummary{Total: 3}}
	repo := &adminAPIKeyListRepoStub{result: want}
	svc := &adminServiceImpl{apiKeyRepo: repo}

	got, err := svc.ListAdminAPIKeys(context.Background(), AdminAPIKeyListParams{
		Pagination: pagination.PaginationParams{
			Page: 2, PageSize: 25, SortBy: " created_at ", SortOrder: "ASC",
		},
		Filters: AdminAPIKeyListFilters{Search: "  customer@example.com  "},
	})

	require.NoError(t, err)
	require.Same(t, want, got)
	require.Equal(t, AdminAPIKeySortCreatedAt, repo.got.Pagination.SortBy)
	require.Equal(t, pagination.SortOrderAsc, repo.got.Pagination.SortOrder)
	require.Equal(t, "customer@example.com", repo.got.Filters.Search)
}

func TestAdminServiceListAdminAPIKeysRejectsMissingReportingRepository(t *testing.T) {
	svc := &adminServiceImpl{apiKeyRepo: &adminAPIKeyListRepoStub{APIKeyRepository: nil}}
	// Use a wrapper that exposes only the hot-path interface at runtime.
	svc.apiKeyRepo = struct{ APIKeyRepository }{APIKeyRepository: svc.apiKeyRepo}

	got, err := svc.ListAdminAPIKeys(context.Background(), AdminAPIKeyListParams{})

	require.Nil(t, got)
	require.ErrorContains(t, err, "repository is unavailable")
}

func TestAdminServiceListAdminAPIKeysPropagatesRepositoryError(t *testing.T) {
	wantErr := errors.New("database unavailable")
	repo := &adminAPIKeyListRepoStub{err: wantErr}
	svc := &adminServiceImpl{apiKeyRepo: repo}

	got, err := svc.ListAdminAPIKeys(context.Background(), AdminAPIKeyListParams{})

	require.Nil(t, got)
	require.ErrorIs(t, err, wantErr)
}
