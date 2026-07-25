package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const (
	AdminAPIKeySortCreatedAt       = "created_at"
	AdminAPIKeySortLastUsedAt      = "last_used_at"
	AdminAPIKeySortTodayCost       = "today_actual_cost"
	AdminAPIKeySortLast30DaysCost  = "last_30_days_actual_cost"
	AdminAPIKeySortTotalActualCost = "total_actual_cost"
)

type AdminAPIKeyListFilters struct {
	Search  string
	UserID  *int64
	GroupID *int64
	Status  string
}

type AdminAPIKeyListParams struct {
	Pagination pagination.PaginationParams
	Filters    AdminAPIKeyListFilters
}

type AdminAPIKeyListItem struct {
	APIKey               APIKey
	TodayActualCost      float64
	Last30DaysActualCost float64
	TotalActualCost      float64
}

type AdminAPIKeyListSummary struct {
	Total                int64
	Active               int64
	Inactive             int64
	Expired              int64
	Last30DaysActualCost float64
}

type AdminAPIKeyListResult struct {
	Items      []AdminAPIKeyListItem
	Pagination pagination.PaginationResult
	Summary    AdminAPIKeyListSummary
}

// AdminAPIKeyListRepository is intentionally separate from the authentication
// repository contract. The full-site inventory is an admin reporting query and
// must not widen the API key hot-path interface.
type AdminAPIKeyListRepository interface {
	ListAdminAPIKeys(ctx context.Context, params AdminAPIKeyListParams) (*AdminAPIKeyListResult, error)
}

func NormalizeAdminAPIKeySort(sortBy, sortOrder string) (string, string) {
	sortBy = strings.TrimSpace(sortBy)
	switch sortBy {
	case AdminAPIKeySortCreatedAt,
		AdminAPIKeySortLastUsedAt,
		AdminAPIKeySortTodayCost,
		AdminAPIKeySortLast30DaysCost,
		AdminAPIKeySortTotalActualCost:
	default:
		sortBy = AdminAPIKeySortLastUsedAt
	}
	return sortBy, pagination.NormalizeSortOrder(sortOrder, pagination.SortOrderDesc)
}

func (s *adminServiceImpl) ListAdminAPIKeys(ctx context.Context, params AdminAPIKeyListParams) (*AdminAPIKeyListResult, error) {
	repo, ok := s.apiKeyRepo.(AdminAPIKeyListRepository)
	if !ok {
		return nil, fmt.Errorf("admin API key list repository is unavailable")
	}

	params.Pagination.SortBy, params.Pagination.SortOrder = NormalizeAdminAPIKeySort(
		params.Pagination.SortBy,
		params.Pagination.SortOrder,
	)
	params.Filters.Search = strings.TrimSpace(params.Filters.Search)
	return repo.ListAdminAPIKeys(ctx, params)
}
