package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func newAdminAPIKeyListRepositoryMock(t *testing.T) (*apiKeyRepository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return &apiKeyRepository{sql: db}, mock
}

func TestAPIKeyRepositoryListAdminAPIKeysUsesActualCostAndAllowlistedSort(t *testing.T) {
	repo, mock := newAdminAPIKeyListRepositoryMock(t)
	userID := int64(9)
	groupID := int64(12)
	now := time.Now().UTC().Truncate(time.Second)

	mock.ExpectQuery(`(?s)WITH filtered_keys AS.*k\.user_id = \$1.*k\.group_id = \$2.*RIGHT\(k\.key, 4\) = \$4.*SUM\(ul\.actual_cost\).*SELECT.*COUNT\(\*\) AS total`).
		WithArgs(userID, groupID, "%7890%", "7890").
		WillReturnRows(sqlmock.NewRows([]string{
			"total", "active", "inactive", "expired", "last_30_days_actual_cost",
		}).AddRow(int64(3), int64(2), int64(1), int64(0), 4.25))

	mock.ExpectQuery(`(?s)WITH filtered_keys AS.*SUM\(ul\.actual_cost\).*ORDER BY COALESCE\(usage\.total_actual_cost, 0\) ASC NULLS LAST, fk\.id DESC.*LIMIT \$5 OFFSET \$6`).
		WithArgs(userID, groupID, "%7890%", "7890", 25, 25).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "key", "name", "group_id", "status", "last_used_at",
			"quota", "quota_used", "expires_at", "created_at", "updated_at",
			"user_email", "user_username", "user_balance",
			"group_name", "group_platform", "group_rate_multiplier",
			"today_actual_cost", "last_30_days_actual_cost", "total_actual_cost",
		}).AddRow(
			int64(41), userID, "sk-admin-inventory-secret-1234567890", "customer-key", groupID,
			service.StatusAPIKeyActive, now, 10.0, 1.25, nil, now, now,
			"customer@example.com", "customer", 8.75,
			"Claude CCMAX", service.PlatformAnthropic, 1.2,
			0.3, 2.4, 6.8,
		))

	result, err := repo.ListAdminAPIKeys(context.Background(), service.AdminAPIKeyListParams{
		Pagination: pagination.PaginationParams{
			Page: 2, PageSize: 25,
			SortBy: service.AdminAPIKeySortTotalActualCost, SortOrder: pagination.SortOrderAsc,
		},
		Filters: service.AdminAPIKeyListFilters{
			Search: "7890", UserID: &userID, GroupID: &groupID,
		},
	})

	require.NoError(t, err)
	require.Equal(t, int64(3), result.Pagination.Total)
	require.Equal(t, 2, result.Pagination.Page)
	require.Equal(t, 4.25, result.Summary.Last30DaysActualCost)
	require.Len(t, result.Items, 1)
	item := result.Items[0]
	require.Equal(t, int64(41), item.APIKey.ID)
	require.Equal(t, "customer@example.com", item.APIKey.User.Email)
	require.Equal(t, "Claude CCMAX", item.APIKey.Group.Name)
	require.Equal(t, 6.8, item.TotalActualCost)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAPIKeyRepositoryListAdminAPIKeysEmptyResultShowsZeroSummary(t *testing.T) {
	repo, mock := newAdminAPIKeyListRepositoryMock(t)
	mock.ExpectQuery(`(?s)WITH filtered_keys AS.*SELECT.*COUNT\(\*\) AS total`).
		WillReturnRows(sqlmock.NewRows([]string{
			"total", "active", "inactive", "expired", "last_30_days_actual_cost",
		}).AddRow(int64(0), int64(0), int64(0), int64(0), 0.0))
	mock.ExpectQuery(`(?s)ORDER BY fk\.last_used_at DESC NULLS LAST, fk\.id DESC.*LIMIT \$1 OFFSET \$2`).
		WithArgs(20, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "key", "name", "group_id", "status", "last_used_at",
			"quota", "quota_used", "expires_at", "created_at", "updated_at",
			"user_email", "user_username", "user_balance", "group_name", "group_platform",
			"group_rate_multiplier", "today_actual_cost", "last_30_days_actual_cost", "total_actual_cost",
		}))

	result, err := repo.ListAdminAPIKeys(context.Background(), service.AdminAPIKeyListParams{
		Pagination: pagination.PaginationParams{Page: 1, PageSize: 20},
	})

	require.NoError(t, err)
	require.Empty(t, result.Items)
	require.Equal(t, int64(0), result.Summary.Total)
	require.Equal(t, 1, result.Pagination.Pages)
	require.NoError(t, mock.ExpectationsWereMet())
}
